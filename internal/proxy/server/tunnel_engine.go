package server

import (
	"net"
	"sync"

	"github.com/sekret01/sekret_go_proxy/internal/config"
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
}

func (t *TunnelNodeWrapper) Node() *proxy.TunnelNode              { return t.node }
func (t *TunnelNodeWrapper) Ip() string                           { return t.ip }
func (t *TunnelNodeWrapper) MsgChan() chan core.MessageData       { return t.msgChannel }
func (t *TunnelNodeWrapper) HandshakeCtx() *core.HandshakeContext { return t.ctx }
func NewTunnelNodeWrapper(node *proxy.TunnelNode, ip string, msgChannel chan core.MessageData, ctx *core.HandshakeContext) *TunnelNodeWrapper {
	return &TunnelNodeWrapper{
		node:       node,
		ip:         ip,
		msgChannel: msgChannel,
		ctx:        ctx,
	}
}

type ServerTunnelEngine struct {
	transport      core.Transport
	encryptor      core.Encryptor
	framer         core.Framer
	bootstrap      core.Bootstrap
	auth           core.Auth
	tunnelNodesMap map[string]*TunnelNodeWrapper
	tunnelsChannel chan *TunnelNodeWrapper

	localAddr    string
	mainListener net.Listener
	connMutex    sync.Mutex
	chanMutex    sync.RWMutex

	logger logger.Logger
	status *proxy.TunnelStatus
	cfg    *config.Config
}

func (s *ServerTunnelEngine) RunTunnel() error {
	s.status.SetLaunch()
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
		_, key, err := s.bootstrap.ServerBootstrap(con)
		enc, err := encryptors.NewEncryptor(s.cfg.EncryptorType, key, s.cfg)
		if err != nil {
			s.logger.Warning("[listenConnectoins] Bootstrap encoder create error: " + err.Error())
			con.Close()
			continue
		}
		s.auth.SetEncryptor(enc)
		ctx, err := s.auth.ServerHandshake(con)
		if err != nil {
			s.logger.Warning("[listenConnectoins] Handshake error: " + err.Error())
			con.Close()
			continue
		}
		s.logger.Debug("[listenConnectoins] auth success")
		tunnelNode := proxy.NewTunnelNode(con, s.framer)
		msgChannel := make(chan core.MessageData, 1000)
		ip := con.RemoteAddr().String()
		tunNodeWrapper := NewTunnelNodeWrapper(tunnelNode, ip, msgChannel, ctx)
		s.connMutex.Lock()
		s.tunnelNodesMap[ip] = tunNodeWrapper
		s.connMutex.Unlock()
		s.tunnelsChannel <- tunNodeWrapper
		s.logger.Debug("[listenConnectoins] New connection: " + ip)
		go s.tunnelReader(ip, tunnelNode, msgChannel)
	}
}

func (s *ServerTunnelEngine) tunnelReader(ip string, tunnelNode *proxy.TunnelNode, msgChannel chan core.MessageData) {
	s.logger.Info("[tunnelReader] Start tunnel listening with " + ip)
	for {
		select {
		case <-tunnelNode.Done():
			s.logger.Info("[tunnelReader] Exit from read cycle")
			s.connMutex.Lock()
			delete(s.tunnelNodesMap, ip)
			s.connMutex.Unlock()
			return
		case frame := <-tunnelNode.GetReadChan().GetChannel():
			data, msgType, requestId, err := s.framer.Unframe(frame)
			if err != nil {
				s.logger.Error("[tunnelReader] ERROR: unframe: " + err.Error())
				continue
			}
			decryptData := s.encryptor.Decrypt(data)
			frameData := core.MessageData{
				MsgType:   msgType,
				RequestId: requestId,
				Data:      decryptData,
			}
			select {
			case <-tunnelNode.Done():
			default:
				msgChannel <- frameData
			}
		}
	}
}

func (s *ServerTunnelEngine) sendIntoTunnel(requestId core.RequestID, msgType core.MessageType, payload []byte, tunNode *proxy.TunnelNode) error {
	s.logger.Debug("[sendIntoTunnel] Prepeare new msg: requestId: [" + utils.RequestIdToString(requestId) + "], msgType: [" + utils.MessageTypeToHexString(msgType) + "], msg: [" + utils.BytesToString(payload, 20) + "]")
	payloadEncode := s.encryptor.Encrypt([]byte(payload))
	frame, err := s.framer.Frame(payloadEncode, msgType, requestId)
	if err != nil {
		s.logger.Error("[sendIntoTunnel] Error in send data: " + err.Error())
		return err
	}
	tunNode.Send(frame)
	return nil
}

// HELP

func (s *ServerTunnelEngine) sendCloseIntoTunnel(requestId core.RequestID, tunNode *proxy.TunnelNode) {
	s.logger.Debug("[sendCloseIntoTunnel] close " + utils.RequestIdToString(requestId))
	s.sendIntoTunnel(requestId, core.MsgClose, []byte{}, tunNode)
}

func NewServerTunnelEngine(
	transport core.Transport,
	encryptor core.Encryptor,
	framer core.Framer,
	bootstrap core.Bootstrap,
	auth core.Auth,
	cfg *config.Config,
	logger logger.Logger,
) *ServerTunnelEngine {
	engine := &ServerTunnelEngine{
		transport:      transport,
		encryptor:      encryptor,
		framer:         framer,
		bootstrap:      bootstrap,
		auth:           auth,
		tunnelNodesMap: make(map[string]*TunnelNodeWrapper, 100),
		tunnelsChannel: make(chan *TunnelNodeWrapper, 100),
		localAddr:      cfg.LocalHost,
		mainListener:   nil,
		logger:         logger,
		status:         proxy.NewTunnelStatus(),
		cfg:            cfg,
	}
	return engine
}
