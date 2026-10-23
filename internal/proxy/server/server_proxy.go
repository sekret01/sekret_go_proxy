package server

import (
	"errors"
	"io"
	"net"
	"sync"

	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/proxy"
	"github.com/sekret01/sekret_go_proxy/internal/utils"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

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
	// clientIp, err := s.ipPool.Allocate()
	// if err != nil {
	// 	return nil
	// }

	// ctx.ClientIP = clientIp
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
			case core.MsgConnect:
				s.connectAndRegistrate(msg.RequestId, msg.Data, tunWrapper.node)
			case core.MsgData:
				s.putMessageIntoChannel(msg.RequestId, msg.Data)
			case core.MsgClose:
				s.closeChannel(msg.RequestId)
				conWrapper, ok := s.dispatcher.Find(msg.RequestId)
				s.logger.Debug("[TunnelReaderHandler] Get close message [" + utils.RequestIdToString(msg.RequestId) + "]")
				if !ok {
					s.logger.Debug("[TunnelReaderHandler] ERROR: not found connection [" + utils.RequestIdToString(msg.RequestId) + "]")
					continue
				}
				conWrapper.Conn.Close()
				proxy.CloseConnection(s.dispatcher, msg.RequestId)
			}
		}
	}
}

func (s *ServerProxy) getOrCreateChannel(requestId core.RequestID) chan []byte {
	s.chanMutex.Lock()
	defer s.chanMutex.Unlock()
	if ch, ex := s.connChannels[requestId]; ex {
		return ch
	}
	ch := make(chan []byte, 1000)
	s.connChannels[requestId] = ch
	return ch
}

func (s *ServerProxy) connectAndRegistrate(requestId core.RequestID, decryptData []byte, tunNode *proxy.TunnelNode) {
	channel := s.getOrCreateChannel(requestId)
	go func() {
		targetCon, err := s.tunnelEngine.transport.Dial(string(decryptData))
		if err != nil {
			s.logger.Debug("[connectAndRegistrate] Can not connect to target server [" + utils.BytesToString(decryptData, 20) + "] " + err.Error())
			s.closeChannel(requestId)
			s.tunnelEngine.sendCloseIntoTunnel(requestId, tunNode) // TODO добавить сообщения ошибок
			return
		}
		s.dispatcher.Register(requestId, targetCon, core.ProtoTest, true)
		go s.targetConnectinoHandler(requestId, targetCon, tunNode)
		go s.conChannelReader(requestId, channel, targetCon)
	}()
}

func (s *ServerProxy) putMessageIntoChannel(requestId core.RequestID, decryptData []byte) {
	s.chanMutex.RLock()
	defer s.chanMutex.RUnlock()
	channel, ex := s.connChannels[requestId]
	if !ex {
		return
	}
	channel <- decryptData
	s.logger.Debug("[putMessageIntoChannel] put new message [" + utils.RequestIdToString(requestId) + "] ")
}

func (s *ServerProxy) conChannelReader(requestId core.RequestID, channel chan []byte, targetCon net.Conn) {
	s.logger.Debug("[conChannelReader] start read chan for [" + utils.RequestIdToString(requestId) + " ]")
	for data := range channel {
		s.logger.Debug("[conChannelReader] send data [" + utils.BytesToString(data, 20) + "...]")
		targetCon.Write(data)
	}
	s.logger.Debug("[conChannelReader] finish chan chan for [" + utils.RequestIdToString(requestId) + "] ")
}

func (s *ServerProxy) targetConnectinoHandler(requestId core.RequestID, targetCon net.Conn, tunNode *proxy.TunnelNode) {
	s.logger.Debug("[targetConnectinoHandler] Create new connection: " + targetCon.RemoteAddr().String())
	defer func() {
		s.tunnelEngine.sendCloseIntoTunnel(requestId, tunNode)
		s.closeChannel(requestId)
		proxy.CloseConnection(s.dispatcher, requestId)

	}()

	buf := make([]byte, 32*1024)
	for {
		select {
		case <-tunNode.GetWriteChan().Done(): // TODO tunNode.Done() ?
			s.logger.Debug("[targetConnectinoHandler] Tunnel is closed, exit targetConnectinoHandler")
			return
		default:
			size, err := targetCon.Read(buf)
			if err != nil {
				if !(errors.Is(err, net.ErrClosed) || err == io.EOF) {
					s.logger.Debug("[targetConnectinoHandler] Client disconnect with error: " + err.Error())
				}
				return
			}
			if size == 0 {
				s.logger.Debug("[targetConnectinoHandler] Client disconnect with data size 0")
				return
			}
			s.tunnelEngine.sendIntoTunnel(requestId, core.MsgData, buf[:size], tunNode)
		}
	}
}

func (s *ServerProxy) closeChannel(requestId core.RequestID) {
	s.chanMutex.Lock()
	defer s.chanMutex.Unlock()
	if ch, ex := s.connChannels[requestId]; ex {
		close(ch)
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
