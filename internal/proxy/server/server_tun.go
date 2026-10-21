package server

import (
	"net"
	"sync"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/proxy"
	"github.com/sekret01/sekret_go_proxy/internal/utils"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type vpnSession struct {
	node      *proxy.TunnelNode
	clientIP  string
	sessionID core.RequestID
}

type ServerTun struct {
	tunnelEngine *ServerTunnelEngine
	packetDevice core.PacketDevice
	logger       logger.Logger

	ipPool        *IpPool
	ipToSession   map[string]*vpnSession
	nodeToSession map[*proxy.TunnelNode]*vpnSession
	mu            sync.Mutex
	once          sync.Once

	done chan struct{}
}

func (s *ServerTun) Start() error {
	if s.tunnelEngine.IsRunning() {
		s.logger.Warning("[Start] Trying to start running service, return")
		return nil
	}
	s.tunnelEngine.SetLaunch()
	if err := s.packetDevice.CreateTun("tun0"); err != nil {
		s.logger.Error("[Start] create tun error: " + err.Error())
		s.packetDevice.ServerClose()
		// s.tunnelEngine.SetStopped()
		return err
	}
	s.packetDevice.ServerClose()
	if err := s.packetDevice.ServerConfigure(); err != nil {
		s.logger.Error("[Start] server configure error: " + err.Error())
		s.packetDevice.ServerClose()
		return err
	}
	if err := s.tunnelEngine.RunTunnel(); err != nil {
		s.logger.Error("[Start] run tunnel error: " + err.Error())
		s.packetDevice.ServerClose()
		s.tunnelEngine.SetStopped()
		return err
	}
	go s.NewTunnelsHandler()
	go s.tunReader()
	return nil
}

func (s *ServerTun) NewHandshakeContext() *core.HandshakeContext {
	ctx := &core.HandshakeContext{}
	clientIp, err := s.ipPool.Allocate()
	if err != nil {
		return nil
	}

	ctx.ClientIP = clientIp
	return ctx
}

func (s *ServerTun) GetStatus() *proxy.TunnelStatus {
	return s.tunnelEngine.GetStatus()
}

func (s *ServerTun) GetUsers() []string {
	return s.tunnelEngine.GetConnectedIps()
}

func (s *ServerTun) NewTunnelsHandler() {
	for {
		select {
		case <-s.done:
			return
		case tunWrapper := <-s.tunnelEngine.GetTunnelNodesChannel():
			s.logger.Debug("[NewTunnelHandler] Get connect from " + tunWrapper.Ip())
			go s.TunnelNodeHandler(tunWrapper)
		}
	}
}

func (s *ServerTun) TunnelNodeHandler(tunWrapper *TunnelNodeWrapper) {
	s.logger.Debug("[TunnelNodeHandler] Handle new node: " + tunWrapper.Ip())
	defer s.cleanupNode(tunWrapper)

	for {
		select {
		case <-tunWrapper.Node().Done():
			s.logger.Debug("[TunnelNodeHandler] Tunnel " + tunWrapper.Ip() + " is closed, exit handle")
			return
		case <-s.done:
			s.logger.Debug("[TunnelNodeHandler] Stop server work, exit handle " + tunWrapper.Ip())
			return
		case msg := <-tunWrapper.msgChannel:
			switch msg.MsgType {
			case core.MsgConnect:
				s.HandleVpnConnect(tunWrapper, msg)
			case core.MsgPacket:
				s.HandleVpnPacket(tunWrapper, msg)
			case core.MsgClose:
				s.releaseSession(tunWrapper.Node())
			}
		}
	}
}

func (s *ServerTun) HandleVpnConnect(tunWrapper *TunnelNodeWrapper, msg core.MessageData) {
	// if string(msg.Data) != "vpn" {
	// 	s.logger.Warning("[HandleVpnConnect] Unexpected connect payload: " + string(msg.Data))
	// 	return
	// }
	// clientIp, err := s.ipPool.Allocate()
	// if err != nil {
	// 	s.logger.Error("[HandleVpnConnect] No free IP: " + err.Error())
	// 	s.tunnelEngine.sendCloseIntoTunnel(msg.RequestId, tunWrapper.Node())
	// 	return
	// }

	session := &vpnSession{
		node:      tunWrapper.Node(),
		clientIP:  tunWrapper.HandshakeCtx().ClientIP,
		sessionID: msg.RequestId,
	}

	s.mu.Lock()
	s.ipToSession[session.clientIP] = session
	s.nodeToSession[tunWrapper.Node()] = session
	s.mu.Unlock()

	reply := []byte("vpn:" + session.clientIP)
	s.tunnelEngine.sendIntoTunnel(msg.RequestId, core.MsgConnect, reply, tunWrapper.Node())
	s.logger.Debug("[HandleVpnConnect] Create session [ " + utils.RequestIdToString(session.sessionID) +
		" ] for client [ " + session.clientIP + " ] (" + tunWrapper.Ip())
}

func (s *ServerTun) HandleVpnPacket(tunWrapper *TunnelNodeWrapper, msg core.MessageData) {
	if _, err := s.packetDevice.WritePacket(msg.Data); err != nil {
		s.logger.Error("[HandleVpnPacket] WritePacket error: " + err.Error())
	}
}

func (s *ServerTun) tunReader() {
	buf := make([]byte, s.packetDevice.MTU())
	for {
		select {
		case <-s.done:
			s.logger.Debug("[tunReader] tun is closed, exit tunReader")
			return
		default:
		}
		n, err := s.packetDevice.ReadPacket(buf)
		if err != nil {
			if s.isDone() {
				s.logger.Debug("[tunReader] packetDevice.ReadPacket error, but tun is closed, exit tunReader")
				return
			}
			s.logger.Error("[tunReader] Error in read tun: " + err.Error())
			continue
		}
		data := buf[:n]
		s.logger.Debug("[tunReader] get [" + utils.IntToString(n) + "] bytes: " + utils.BytesToString(data, 20))
		dstIp, ok := extractDstIPv4(data)
		if !ok {
			s.logger.Debug("[tunReader] Cannot get IP from packet: " + string(dstIp))
			continue
		}
		s.mu.Lock()
		session, ex := s.ipToSession[dstIp]
		s.mu.Unlock()
		if !ex {
			continue
		}
		s.tunnelEngine.sendIntoTunnel(core.GetNoopID(), core.MsgPacket, data, session.node)
	}
}

func (s *ServerTun) cleanupNode(w *TunnelNodeWrapper) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ex := s.nodeToSession[w.Node()]; ex {
		delete(s.ipToSession, sess.clientIP)
		delete(s.nodeToSession, w.Node())
		s.ipPool.Release(sess.clientIP)
		s.logger.Info("[cleanupNode] released ip " + sess.clientIP)
	}
}

func (s *ServerTun) releaseSession(node *proxy.TunnelNode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ex := s.nodeToSession[node]; ex {
		delete(s.ipToSession, sess.clientIP)
		delete(s.nodeToSession, node)
		s.ipPool.Release(sess.clientIP)
	}
}

func (s *ServerTun) Stop() error {
	s.once.Do(func() {
		close(s.done)
		s.packetDevice.ServerClose()
		s.tunnelEngine.StopListening()
		s.tunnelEngine.SetStopped()
	})
	return nil
}

func (s *ServerTun) isDone() bool {
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}

func NewServerTun(
	tunnelEngine *ServerTunnelEngine,
	packetDevice core.PacketDevice,
	logger logger.Logger,
	cfg *config.Config,
) *ServerTun {
	return &ServerTun{
		tunnelEngine: tunnelEngine,
		packetDevice: packetDevice,
		logger:       logger,

		ipPool:        NewIPPool(cfg.VPN.Subnet),
		ipToSession:   make(map[string]*vpnSession),
		nodeToSession: make(map[*proxy.TunnelNode]*vpnSession),
		done:          make(chan struct{}),
	}
}

func extractDstIPv4(pkt []byte) (string, bool) {
	if len(pkt) < 20 {
		return "", false
	}
	if pkt[0]>>4 != 4 {
		return "", false
	}
	return net.IP(pkt[16:20]).String(), true
}
