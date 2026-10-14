package server

import (
	"net"
	"sync"
	"time"

	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/proxy"
	"github.com/sekret01/sekret_go_proxy/internal/utils"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type chanMap map[core.RequestID]chan []byte

type ServerTunnelEngine struct {
	transport core.Transport
	encryptor core.Encryptor
	framer    core.Framer
	auth      core.Auth

	localAddr       string
	mainListener    net.Listener
	connectionsList []net.Conn // TODO объединить в структуру
	connChannels    chanMap    // TODO объединить в структуру
	connMutex       sync.Mutex
	chanMutex       sync.RWMutex
	tunnelMessages  chan core.MessageData

	logger logger.Logger

	status *proxy.TunnelStatus
}

func (s *ServerTunnelEngine) RunTunnel() error {
	s.status.SetLaunch()
	listener, err := s.transport.Listen(s.localAddr)
	s.logger.Info("Start listen on " + s.localAddr)
	if err != nil {
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
	for id, connChan := range s.connChannels {
		close(connChan)
		delete(s.connChannels, id)
	}
	for _, conn := range s.connectionsList {
		conn.Close()
	}
	s.connectionsList = nil
	s.connectionsList = []net.Conn{}
	s.mainListener.Close()
	return nil
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
		writeChannel := proxy.NewTunnelChannel()
		s.addConnToList(con)
		go s.tunnelWriter(con, writeChannel)
		go s.tunnelReader(con, writeChannel)
	}
}

func (s *ServerTunnelEngine) tunnelReader(tunnelConn net.Conn, writeChannel *proxy.TunnelChannel) { // writeChannel chan []byte
	s.logger.Info("[tunnelReader] Start tunnel listening with " + tunnelConn.RemoteAddr().String())

	_, err := s.auth.ServerHandshake(tunnelConn)
	if err != nil {
		s.logger.Error(err.Error())
		writeChannel.Close()
		s.removeConnFromList(tunnelConn)
		return
	}
	s.logger.Debug("[tunnelReader] auth success")

	for {
		frame, err := proxy.ReadFrameFromConnection(tunnelConn, s.framer)
		if err != nil {
			s.logger.Error("[tunnelReader] ERROR in read frame: " + err.Error())
			writeChannel.Close()
			s.removeConnFromList(tunnelConn)
			return
		}
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
		s.tunnelMessages <- frameData // TODO для каждого подключениня (туннеля) свой канал
	}
}

func (s *ServerTunnelEngine) tunnelWriter(tunnelConn net.Conn, writeChannel *proxy.TunnelChannel) { // writeChannel chan []byte
	for frame := range writeChannel.Channel {
		_, err := tunnelConn.Write(frame)
		if err != nil {
			s.logger.Error("[tunnelWriter]: write msg error in tunnel [ tun_ch_" + utils.IntToString(writeChannel.Id) + " ]: " + err.Error())
			// close(writeChannel)
			return
		}
	}
}

func (s *ServerTunnelEngine) sendIntoTunnel(requestId core.RequestID, msgType core.MessageType, payload []byte, writeChannel *proxy.TunnelChannel) error { // writeChannel chan []byte
	s.logger.Debug("[sendIntoTunnel] Prepeare new msg: requestId: [" + utils.RequestIdToString(requestId) + "], msgType: [" + utils.MessageTypeToHexString(msgType) + "], msg: [" + utils.BytesToString(payload, 20) + "]")
	payloadEncode := s.encryptor.Encrypt([]byte(payload))
	frame, err := s.framer.Frame(payloadEncode, msgType, requestId)
	if err != nil {
		s.logger.Error("[sendIntoTunnel] Error in send data: " + err.Error())
		return err
	}
	if !writeChannel.IsOpen() {
		s.logger.Debug("[sendIntoTunnel] Trying send data in close tunnel-channel [ tun_ch_" + utils.IntToString(writeChannel.Id) + " ], cencel")
		return nil
	}
	select {
	case writeChannel.Channel <- frame:
		s.logger.Debug("[sendIntoTunnel] Data has been put into writeChannel [ tun_ch_" + utils.IntToString(writeChannel.Id) + " ]")
		return nil
	case <-time.After(time.Second * 5):
		s.logger.Error("[sendIntoTunnel] writeChannel [ tun_ch_" + utils.IntToString(writeChannel.Id) + " ] is full")
	}
	return nil
}

// HELP

func (s *ServerTunnelEngine) sendCloseIntoTunnel(requestId core.RequestID, writeChannel *proxy.TunnelChannel) { // writeChannel chan []byte
	s.logger.Debug("[sendCloseIntoTunnel] close " + utils.RequestIdToString(requestId))
	s.sendIntoTunnel(requestId, core.MsgClose, []byte{}, writeChannel)
}

func (s *ServerTunnelEngine) addConnToList(con net.Conn) {
	s.connectionsList = append(s.connectionsList, con)
}

func (s *ServerTunnelEngine) removeConnFromList(con net.Conn) {
	ind := -1
	for i, el := range s.connectionsList {
		if el.RemoteAddr().String() == con.RemoteAddr().String() {
			ind = i
			break
		}
	}
	if ind == -1 {
		s.logger.Warning("Con not found in list")
		return
	}
	s.connectionsList[ind] = s.connectionsList[len(s.connectionsList)-1]
	s.connectionsList[len(s.connectionsList)-1] = nil
	s.connectionsList = s.connectionsList[:len(s.connectionsList)-1]
}
