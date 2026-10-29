package server

import (
	"net"
	"sync"

	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/encryptors"
	"github.com/sekret01/sekret_go_proxy/internal/proxy"
	"github.com/sekret01/sekret_go_proxy/internal/utils"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type TunnelNodeWrapper struct {
	node       *proxy.TunnelNode
	ip         string
	msgChannel chan core.MessageData
	ctx        *core.HandshakeContext
	Moduls     *TunnelModules
}

func (t *TunnelNodeWrapper) Node() *proxy.TunnelNode              { return t.node }
func (t *TunnelNodeWrapper) Ip() string                           { return t.ip }
func (t *TunnelNodeWrapper) MsgChan() chan core.MessageData       { return t.msgChannel }
func (t *TunnelNodeWrapper) HandshakeCtx() *core.HandshakeContext { return t.ctx }
func NewTunnelNodeWrapper(
	node *proxy.TunnelNode,
	ip string,
	msgChannel chan core.MessageData,
	ctx *core.HandshakeContext,
	moduls *TunnelModules,
) *TunnelNodeWrapper {
	return &TunnelNodeWrapper{
		node:       node,
		ip:         ip,
		msgChannel: msgChannel,
		ctx:        ctx,
		Moduls:     moduls,
	}
}

type TunnelModules struct {
	Encryptor core.Encryptor
}

func NewTunnelModules(
	enc core.Encryptor,
) *TunnelModules {
	return &TunnelModules{
		Encryptor: enc,
	}
}

type ServerTunnelEngine struct {
	transport      core.Transport
	framer         core.Framer
	bootstrap      core.Bootstrap
	auth           core.Auth
	tunnelNodesMap map[string]*TunnelNodeWrapper
	tunnelsChannel chan *TunnelNodeWrapper

	localAddr    string
	encType      string
	mainListener net.Listener
	connMutex    sync.Mutex
	chanMutex    sync.RWMutex

	logger logger.Logger
	status *proxy.TunnelStatus
	done   chan struct{}
}

func (s *ServerTunnelEngine) RunTunnel() error {
	s.status.SetLaunch()
	s.tunnelsChannel = make(chan *TunnelNodeWrapper, 100)
	s.done = make(chan struct{})
	listener, err := s.transport.Listen(s.localAddr)
	s.logger.Info("Start listen on " + s.localAddr)
	if err != nil {
		s.status.SetStopped()
		return err
	}
	s.status.SetRunning()
	s.mainListener = listener
	go s.listenConnectoins()
	return nil
}

func (s *ServerTunnelEngine) StopListening() error {
	if !s.status.IsRunning() {
		s.logger.Warning("[Stop] Trying to stop stopped service, return")
		return nil
	}
	s.status.SetStopped()
	close(s.done)
	close(s.tunnelsChannel)
	s.connMutex.Lock()
	for soket, tunNode := range s.tunnelNodesMap {
		tunNode.Node().Stop()
		delete(s.tunnelNodesMap, soket)
	}
	s.connMutex.Unlock()
	s.mainListener.Close()
	return nil
}

func (s *ServerTunnelEngine) GetTunnelNodesChannel() chan *TunnelNodeWrapper {
	return s.tunnelsChannel
}

func (s *ServerTunnelEngine) GetTunnelNodeWrapperForIp(ip string) *TunnelNodeWrapper {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()
	if node, ex := s.tunnelNodesMap[ip]; ex {
		return node
	}
	return nil
}

func (s *ServerTunnelEngine) GetConnectedIps() []string {
	s.connMutex.Lock()
	res := []string{}
	for ip, _ := range s.tunnelNodesMap {
		res = append(res, ip)
	}
	s.connMutex.Unlock()
	return res
}

func (s *ServerTunnelEngine) IsRunning() bool {
	return s.status.IsRunning()
}

func (s *ServerTunnelEngine) GetStatus() *proxy.TunnelStatus {
	return s.status
}

func (s *ServerTunnelEngine) SetRunning() {
	s.status.SetRunning()
}

func (s *ServerTunnelEngine) SetLaunch() {
	s.status.SetLaunch()
}

func (s *ServerTunnelEngine) SetStopped() {
	s.status.SetStopped()
}

func (s *ServerTunnelEngine) authenticate(conn net.Conn) (*core.HandshakeContext, error) {
	steps, key, err := s.bootstrap.ServerBootstrap(conn)
	if err != nil {
		s.logger.Warning("[authenticate] Bootstrap fail: " + err.Error())
		conn.Close()
		return nil, err
	}
	enc, err := encryptors.NewEncryptor(s.encType, key)
	if err != nil {
		s.logger.Warning("[authenticate] Bootstrap encoder create error: " + err.Error())
		conn.Close()
		return nil, err
	}
	s.auth.SetEncryptor(enc)
	s.auth.SetSteps(steps)
	ctx, err := s.auth.ServerHandshake(conn)
	if err != nil {
		s.logger.Warning("[authenticate] Handshake error: " + err.Error())
		conn.Close()
		return ctx, err
	}
	return ctx, nil
}

func (s *ServerTunnelEngine) listenConnectoins() error {
	for {
		con, err := s.mainListener.Accept()
		if err != nil {
			s.logger.Error("[ServerProxy] Error in connection accept -> " + err.Error())
			if s.status.IsRunning() {
				continue
			} else {
				s.status.SetStopped()
				return nil
			}
		}
		s.logger.Info("[listenConnectoins] Start auth [" + con.RemoteAddr().String() + "]")
		ctx, err := s.authenticate(con)
		if err != nil {
			s.logger.Warning("[listenConnectoins] Auth [" + con.RemoteAddr().String() + "] fail: " + err.Error())
			con.Close()
			continue
		}
		s.logger.Info("[listenConnectoins] Auth success [" + con.RemoteAddr().String() + "]")
		s.logger.Info("[listenConnectoins] Start configure tunnel node [" + con.RemoteAddr().String() + "]")
		tunNodeWrapper, err := s.configureTunNodeWrapper(con, ctx)
		if err != nil {
			s.logger.Warning("[listenConnectoins] Configure tunnel node [ " + tunNodeWrapper.ip + " ] fail: " + err.Error())
			con.Close()
			continue
		}
		s.logger.Info("[listenConnectoins] Configure success [" + tunNodeWrapper.ip + "]")
		s.connMutex.Lock()
		s.tunnelNodesMap[tunNodeWrapper.ip] = tunNodeWrapper
		s.connMutex.Unlock()
		if ok := s.putTunnelInChannel(tunNodeWrapper); !ok {
			s.logger.Debug("[listenConnectoins] Cannot put connection in chan, skip: " + tunNodeWrapper.ip)
			continue
		}
		s.logger.Debug("[listenConnectoins] New connection: " + tunNodeWrapper.ip)
		go s.tunnelReader(tunNodeWrapper.ip, tunNodeWrapper, tunNodeWrapper.msgChannel)
	}
}

func (s *ServerTunnelEngine) putTunnelInChannel(tunNodeWrapper *TunnelNodeWrapper) bool {
	select {
	case <-s.done:
		return false
	case s.tunnelsChannel <- tunNodeWrapper:
		return true
	}

}

func (s *ServerTunnelEngine) configureTunNodeWrapper(con net.Conn, ctx *core.HandshakeContext) (*TunnelNodeWrapper, error) {
	tunnelNode := proxy.NewTunnelNode(con, s.framer)
	msgChannel := make(chan core.MessageData, 1000)
	ip := con.RemoteAddr().String()
	enc, err := encryptors.NewEncryptor(s.encType, ctx.EncryptorKey)
	if err != nil {
		return nil, err
	}
	tunModuls := NewTunnelModules(enc)
	tunNodeWrapper := NewTunnelNodeWrapper(tunnelNode, ip, msgChannel, ctx, tunModuls)
	return tunNodeWrapper, nil
}

func (s *ServerTunnelEngine) tunnelReader(ip string, tunnelNodeWrapper *TunnelNodeWrapper, msgChannel chan core.MessageData) {
	s.logger.Info("[tunnelReader] Start tunnel listening with " + ip)
	for {
		select {
		case <-tunnelNodeWrapper.Node().Done():
			s.logger.Info("[tunnelReader] Exit from read cycle")
			s.connMutex.Lock()
			delete(s.tunnelNodesMap, ip)
			s.connMutex.Unlock()
			return
		case frame := <-tunnelNodeWrapper.Node().GetReadChan().GetChannel():
			data, msgType, requestId, err := s.framer.Unframe(frame)
			if err != nil {
				s.logger.Error("[tunnelReader] ERROR: unframe: " + err.Error())
				continue
			}
			decryptData := tunnelNodeWrapper.Moduls.Encryptor.Decrypt(data)
			frameData := core.MessageData{
				MsgType:   msgType,
				RequestId: requestId,
				Data:      decryptData,
			}
			select {
			case <-tunnelNodeWrapper.Node().Done():
			default:
				msgChannel <- frameData
			}
		}
	}
}

func (s *ServerTunnelEngine) sendIntoTunnel(requestId core.RequestID, msgType core.MessageType, payload []byte, tunnelNodeWrapper *TunnelNodeWrapper) error {
	s.logger.Debug("[sendIntoTunnel] Prepeare new msg: requestId: [" + utils.RequestIdToString(requestId) + "], msgType: [" + utils.MessageTypeToHexString(msgType) + "], msg: [" + utils.BytesToString(payload, 20) + "]")
	payloadEncode := tunnelNodeWrapper.Moduls.Encryptor.Encrypt([]byte(payload))
	frame, err := s.framer.Frame(payloadEncode, msgType, requestId)
	if err != nil {
		s.logger.Error("[sendIntoTunnel] Error in send data: " + err.Error())
		return err
	}
	tunnelNodeWrapper.Node().Send(frame)
	return nil
}

// HELP

func (s *ServerTunnelEngine) sendCloseIntoTunnel(requestId core.RequestID, tunnelNodeWrapper *TunnelNodeWrapper) {
	s.logger.Debug("[sendCloseIntoTunnel] close " + utils.RequestIdToString(requestId))
	s.sendIntoTunnel(requestId, core.MsgClose, []byte{}, tunnelNodeWrapper)
}

func NewServerTunnelEngine(
	transport core.Transport,
	framer core.Framer,
	bootstrap core.Bootstrap,
	auth core.Auth,
	localAddr string,
	encType string,
	logger logger.Logger,
) *ServerTunnelEngine {
	engine := &ServerTunnelEngine{
		transport:      transport,
		framer:         framer,
		bootstrap:      bootstrap,
		auth:           auth,
		tunnelNodesMap: make(map[string]*TunnelNodeWrapper, 100),
		tunnelsChannel: make(chan *TunnelNodeWrapper, 100),
		localAddr:      localAddr,
		encType:        encType,
		mainListener:   nil,
		logger:         logger,
		status:         proxy.NewTunnelStatus(),
		done:           nil,
	}
	return engine
}
