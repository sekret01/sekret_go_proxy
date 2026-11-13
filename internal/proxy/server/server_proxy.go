package server

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/proxy"
	"github.com/sekret01/sekret_go_proxy/internal/utils"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type chanMap map[core.RequestID]*proxy.SafeMessageChannel // SafeChannel

type ServerProxy struct {
	tunnelEngine *ServerTunnelEngine
	dispatcher   core.Dispatcher
	detector     core.Detector
	logger       logger.Logger

	chanMutex    sync.RWMutex
	connChannels chanMap
}

func (s *ServerProxy) Start() error {
	if s.tunnelEngine.IsRunning() {
		s.logger.Warning("[Start] Trying to start running service, return")
		return nil
	}
	err := s.tunnelEngine.RunTunnel()
	go s.NewTunnelsHandler()
	return err
}

func (s *ServerProxy) Stop() error {
	err := s.tunnelEngine.StopListening()
	return err
}

func (s *ServerProxy) IsRunning() bool {
	return s.tunnelEngine.IsRunning()
}

func (s *ServerProxy) GetStatus() *proxy.TunnelStatus {
	return s.tunnelEngine.GetStatus()
}

func (s *ServerProxy) GetUsers() []string {
	return s.tunnelEngine.GetConnectedIps()
}

func (s *ServerProxy) NewHandshakeContext() *core.HandshakeContext {
	ctx := &core.HandshakeContext{}
	encKey := utils.Generate32ByteKey()
	ctx.EncryptorKey = encKey
	return ctx
}

func (s *ServerProxy) NewTunnelsHandler() {
	tunnelsChannel := s.tunnelEngine.GetTunnelNodesChannel()
	for tunWrapper := range tunnelsChannel {
		s.logger.Debug("[NewTunnelsHandler] Get connection from tunnelEngine: " + tunWrapper.ip)
		go s.TunnelReaderHandler(*tunWrapper)
	}
}

func (s *ServerProxy) TunnelReaderHandler(tunWrapper TunnelNodeWrapper) {
	s.logger.Debug("[TunnelReaderHandler] Run TunnelReaderHandler")
	for {
		select {
		case <-tunWrapper.node.Done():
			s.logger.Debug("[TunnelReaderHandler] Tunnel is closed, exit TunnelReaderHandler")
			return
		case msg := <-tunWrapper.msgChannel:
			switch msg.MsgType {
			case core.MsgPacket:
				s.logger.Info("Packet get")
				msg.Buf.Free()
			case core.MsgConnect:
				s.connectAndRegistrate(msg.RequestId, msg.Data, &tunWrapper)
				msg.Buf.Free()
			case core.MsgData:
				s.putMessageIntoChannel(msg.RequestId, &msg)
			case core.MsgClose:
				s.closeChannel(msg.RequestId)
				conWrapper, ok := s.dispatcher.Find(msg.RequestId)
				s.logger.Debug("[TunnelReaderHandler] Get close message [" + utils.RequestIdToString(msg.RequestId) + "]")
				if !ok {
					s.logger.Debug("[TunnelReaderHandler] ERROR: not found connection [" + utils.RequestIdToString(msg.RequestId) + "]")
					msg.Buf.Free()
					continue
				}
				conWrapper.Conn.Close()
				proxy.CloseConnection(s.dispatcher, msg.RequestId)
				msg.Buf.Free()
			}
		}
	}
}

func (s *ServerProxy) getOrCreateChannel(requestId core.RequestID) *proxy.SafeMessageChannel {
	s.chanMutex.Lock()
	if ch, ex := s.connChannels[requestId]; ex {
		s.chanMutex.Unlock()
		return ch
	}
	s.chanMutex.Unlock()

	ch := proxy.NewSafeMessageChannel()

	s.chanMutex.Lock()
	defer s.chanMutex.Unlock()
	s.connChannels[requestId] = ch
	return ch
}

func (s *ServerProxy) connectAndRegistrate(requestId core.RequestID, decryptData []byte, tunnelNodeWrapper *TunnelNodeWrapper) {
	channel := s.getOrCreateChannel(requestId)
	go func() {
		targetCon, err := s.tunnelEngine.transport.Dial(string(decryptData))
		if err != nil {
			s.logger.Debugf(func() string {
				return "[connectAndRegistrate] Can not connect to target server [" + utils.BytesToString(decryptData, 20) + "] " + err.Error()
			})
			s.closeChannel(requestId)
			s.tunnelEngine.sendCloseIntoTunnel(requestId, tunnelNodeWrapper)
			return
		}
		s.dispatcher.Register(requestId, targetCon, core.ProtoTest, true)
		go s.targetConnectinoHandler(requestId, targetCon, tunnelNodeWrapper)
		go s.conChannelReader(requestId, channel, targetCon)
	}()
}

func (s *ServerProxy) putMessageIntoChannel(requestId core.RequestID, msg *core.MessageData) {
	s.chanMutex.RLock()
	safeChannel, ex := s.connChannels[requestId]
	s.chanMutex.RUnlock()
	if !ex {
		msg.Buf.Free()
		return
	}
	if ok := safeChannel.Put(msg); !ok {
		msg.Buf.Free()
		s.logger.Debug("[putMessageIntoChannel] WARNING cannot put msg [" + utils.RequestIdToString(requestId) + "] into channel")
	}
}

func (s *ServerProxy) conChannelReader(requestId core.RequestID, channel *proxy.SafeMessageChannel, targetCon net.Conn) {
	s.logger.Debug("[conChannelReader] start read chan for [" + utils.RequestIdToString(requestId) + " ]")
	for {
		select {
		case <-channel.Done():
			s.logger.Debugf(func() string {
				return "[conChannelReader] finish chan chan for [" + utils.RequestIdToString(requestId) + "]"
			})
			return
		case msg := <-channel.GetChannel():
			targetCon.Write(msg.Data)
			msg.Buf.Free()
		}
	}
}

func (s *ServerProxy) targetConnectinoHandler(requestId core.RequestID, targetCon net.Conn, tunnelNodeWrapper *TunnelNodeWrapper) {
	s.logger.Debug("[targetConnectinoHandler] Create new connection: " + targetCon.RemoteAddr().String())
	defer func() {
		targetCon.Close()
		s.tunnelEngine.sendCloseIntoTunnel(requestId, tunnelNodeWrapper)
		s.closeChannel(requestId)
		proxy.CloseConnection(s.dispatcher, requestId)

	}()

	buf := make([]byte, 16*1024)
	for {
		select {
		case <-tunnelNodeWrapper.Node().GetWriteChan().Done():
			s.logger.Debug("[targetConnectinoHandler] Tunnel is closed, exit targetConnectinoHandler [" + utils.RequestIdToString(requestId) + "]")
			return
		default:
			targetCon.SetDeadline(time.Now().Add(10 * time.Second))
			size, err := targetCon.Read(buf)
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					s.logger.Debug("[targetConnectinoHandler] Target [" + utils.RequestIdToString(requestId) + "]  disconnect with timeout")
					return
				}
				if !(errors.Is(err, net.ErrClosed) || err == io.EOF) {
					s.logger.Debug("[targetConnectinoHandler] Target [" + utils.RequestIdToString(requestId) + "]  disconnect with error: " + err.Error())
				}
				return
			}
			if size == 0 {
				s.logger.Debug("[targetConnectinoHandler] Target [" + utils.RequestIdToString(requestId) + "] disconnect with data size 0")
				return
			}
			s.tunnelEngine.sendIntoTunnel(requestId, core.MsgData, buf[:size], tunnelNodeWrapper)
		}
	}
}

func (s *ServerProxy) closeChannel(requestId core.RequestID) {
	s.chanMutex.Lock()
	defer s.chanMutex.Unlock()
	if ch, ex := s.connChannels[requestId]; ex {
		ch.Close()
		delete(s.connChannels, requestId)
	}
	s.logger.Debug("[closeChannel] close target-channel for [" + utils.RequestIdToString(requestId) + " ]")
}

func NewServerProxy(
	tunnelEngine *ServerTunnelEngine,
	dispatcher core.Dispatcher,
	detector core.Detector,
	logger logger.Logger,
) *ServerProxy {
	return &ServerProxy{
		tunnelEngine: tunnelEngine,
		dispatcher:   dispatcher,
		detector:     detector,
		logger:       logger,
		connChannels: make(chanMap, 0),
	}
}
