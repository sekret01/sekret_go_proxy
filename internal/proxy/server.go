package proxy

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/utils"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type chanMap map[core.RequestID]chan []byte

type ServerTunnel struct {
	transport  core.Transport  // Подключение и передача информации
	encryptor  core.Encryptor  // Шифрование и расшифровка
	framer     core.Framer     // Оборачивание и парсинг самописных протоколов
	dispatcher core.Dispatcher // Сохранение связи запрос - владелец запроса
	detector   core.Detector   // Определение протокола
	auth       core.Auth

	localAddr       string       // Адрес текущего узла
	mainListener    net.Listener // Слушатель внешних подключений
	connectionsList []net.Conn   // Список подключений

	logger    logger.Logger
	connMutex sync.Mutex
	chanMutex sync.RWMutex
	// writeChannel chan []byte
	connChannels chanMap
	status       *TunnelStatus // Состояние работы
}

func (s *ServerTunnel) Start() error {
	listener, err := s.transport.Listen(s.localAddr)
	s.logger.Info("Start listen on " + s.localAddr)
	if err != nil {
		return err
	}
	s.status.SetLaunch()
	s.mainListener = listener
	for {
		con, err := s.mainListener.Accept()
		if err != nil {
			s.logger.Error("[ServerTunnel] Error in connection accept -> " + err.Error())
			if s.status.isRunning {
				continue
			} else {
				s.status.SetStopped()
				return nil
			}
		}
		writeChannel := make(chan []byte, 100)
		s.connectionsList = append(s.connectionsList, con)
		s.status.SetRunning()
		go s.tunnelWriter(con, writeChannel)
		go s.tunnelReader(con, writeChannel)
	}
}

func (s *ServerTunnel) Stop() error {
	if !s.status.isRunning {
		s.logger.Warning("[Stop] Trying to stop stopped service, return")
		return nil
	}
	s.status.SetStopped()
	for id, reqChan := range s.connChannels {
		close(reqChan)
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

func (s *ServerTunnel) IsRunning() bool {
	return s.status.isRunning
}

func (s *ServerTunnel) GetStatus() *TunnelStatus {
	return s.status
}

func (s *ServerTunnel) tunnelWriter(tunnelConn net.Conn, writeChannel chan []byte) {
	for frame := range writeChannel {
		_, err := tunnelConn.Write(frame)
		if err != nil {
			s.logger.Error("[tunnelWriter]: write msg error: " + err.Error())
			return
		}
	}
}

func (s *ServerTunnel) tunnelReader(tunnelConn net.Conn, writeChannel chan []byte) {
	s.logger.Info("[tunnelReader] Start tunnel listening with " + tunnelConn.RemoteAddr().String())

	_, err := s.auth.ServerHandshake(tunnelConn)
	if err != nil {
		s.logger.Error(err.Error())
	}
	s.logger.Debug("[tunnelReader] auth success")

	for {
		frame, err := ReadFrameFromConnection(tunnelConn, s.framer)
		if err != nil {
			s.logger.Error("[tunnelReader] ERROR in read frame: " + err.Error())
			return
		}
		data, msgType, requestId, err := s.framer.Unframe(frame)
		decryptData := s.encryptor.Decrypt(data)
		if err != nil {
			s.logger.Error("[tunnelReader] ERROR: unframe: " + err.Error())
			continue
		}

		switch msgType {
		case core.MsgConnect:
			s.connectAndRegistrate(requestId, decryptData, writeChannel)
		case core.MsgData:
			s.putMessageIntoChannel(requestId, decryptData)
		case core.MsgClose:
			s.closeChannel(requestId)
			conWrapper, ok := s.dispatcher.Find(requestId)
			s.logger.Debug("[tunnelReader] Get close message [" + utils.RequestIdToString(requestId) + "]")
			if !ok {
				s.logger.Debug("[tunnelReader] ERROR: not found connection [" + utils.RequestIdToString(requestId) + "]")
				continue
			}
			conWrapper.Conn.Close()
			closeConnection(s.dispatcher, requestId)
		}
	}
}

func (s *ServerTunnel) getOrCreateChannel(requestId core.RequestID) chan []byte {
	s.chanMutex.Lock()
	defer s.chanMutex.Unlock()
	if ch, ex := s.connChannels[requestId]; ex {
		return ch
	}
	ch := make(chan []byte, 100)
	s.connChannels[requestId] = ch
	return ch
}

func (s *ServerTunnel) connectAndRegistrate(requestId core.RequestID, decryptData []byte, writeChannel chan []byte) {
	channel := s.getOrCreateChannel(requestId)
	go func() {
		targetCon, err := s.transport.Dial(string(decryptData))
		if err != nil {
			s.logger.Debug("[connectAndRegistrate] Can not connect to target server [" + utils.BytesToString(decryptData, 20) + "] " + err.Error())
			s.closeChannel(requestId)
			s.sendCloseIntoTunnel(requestId, writeChannel)
			return
		}
		s.dispatcher.Register(requestId, targetCon, core.ProtoTest, true)
		go s.targetConnectinoHandler(requestId, targetCon, writeChannel)
		go s.conChannelReader(requestId, channel, targetCon)
	}()
}

func (s *ServerTunnel) putMessageIntoChannel(requestId core.RequestID, decryptData []byte) {
	s.chanMutex.RLock()
	defer s.chanMutex.RUnlock()
	channel, ex := s.connChannels[requestId]
	if !ex {
		return
	}
	channel <- decryptData
	s.logger.Debug("[putMessageIntoChannel] put new message [" + utils.RequestIdToString(requestId) + "] ")
}

func (s *ServerTunnel) conChannelReader(requestId core.RequestID, channel chan []byte, targetCon net.Conn) {
	s.logger.Debug("[conChannelReader] start read chan fir [" + utils.RequestIdToString(requestId) + " ]")
	for data := range channel {
		s.logger.Debug("[conChannelReader] send data [" + utils.BytesToString(data, 20) + "...]")
		targetCon.Write(data)
	}
	s.logger.Debug("[conChannelReader] finish chan chan for [" + utils.RequestIdToString(requestId) + "] ")
}

func (s *ServerTunnel) targetConnectinoHandler(requestId core.RequestID, targetCon net.Conn, writeChannel chan []byte) {
	s.logger.Debug("[targetConnectinoHandler] Create new connection: " + targetCon.RemoteAddr().String())
	defer func() {
		s.sendCloseIntoTunnel(requestId, writeChannel)
		s.closeChannel(requestId)
		closeConnection(s.dispatcher, requestId)
	}()

	buf := make([]byte, 32*1024)
	for {
		size, err := targetCon.Read(buf)
		if err != nil {
			if !(errors.Is(err, net.ErrClosed) || err == io.EOF) {
				s.logger.Warning("[targetConnectinoHandler] Client disconnect with error: " + err.Error())
			}
			return
		}
		if size == 0 {
			s.logger.Debug("[targetConnectinoHandler] Client disconnect with data size 0")
			return
		}
		s.sendIntoTunnel(requestId, core.MsgData, buf[:size], writeChannel)
	}
}

func (s *ServerTunnel) sendIntoTunnel(requestId core.RequestID, msgType core.MessageType, payload []byte, writeChannel chan []byte) error {
	s.logger.Debug("[sendIntoTunnel] Prepeare new msg: requestId: [" + utils.RequestIdToString(requestId) + "], msgType: [" + utils.MessageTypeToHexString(msgType) + "], msg: [" + utils.BytesToString(payload, 20) + "]")
	payloadEncode := s.encryptor.Encrypt([]byte(payload))
	frame, err := s.framer.Frame(payloadEncode, msgType, requestId)
	if err != nil {
		s.logger.Error("[sendIntoTunnel] Error in send data: " + err.Error())
		return err
	}
	select {
	case writeChannel <- frame:
		s.logger.Debug("[sendIntoTunnel] Data has been put into writeChannel")
		return nil
	case <-time.After(time.Second * 5):
		s.logger.Error("[sendIntoTunnel] writeChannel [" + utils.RequestIdToString(requestId) + "] is full")
	}
	return nil
}

func (s *ServerTunnel) sendCloseIntoTunnel(requestId core.RequestID, writeChannel chan []byte) {
	s.logger.Debug("[sendCloseIntoTunnel] close " + utils.RequestIdToString(requestId))
	s.sendIntoTunnel(requestId, core.MsgClose, []byte{}, writeChannel)
}

func (s *ServerTunnel) closeChannel(requestId core.RequestID) {
	s.chanMutex.Lock()
	defer s.chanMutex.Unlock()
	if ch, ex := s.connChannels[requestId]; ex {
		close(ch)
		delete(s.connChannels, requestId)
	}
	s.logger.Debug("[closeChannel] close channel for [" + utils.RequestIdToString(requestId) + " ]")
}

func NewServerTunnel(
	transport core.Transport,
	encryptor core.Encryptor,
	framer core.Framer,
	dispatcher core.Dispatcher,
	detector core.Detector,
	cfg *config.Config,
	logger logger.Logger,
	auth core.Auth) *ServerTunnel {
	return &ServerTunnel{
		transport:    transport,
		encryptor:    encryptor,
		framer:       framer,
		dispatcher:   dispatcher,
		detector:     detector,
		status:       NewTunnelStatus(),
		localAddr:    cfg.LocalHost,
		logger:       logger,
		auth:         auth,
		connChannels: make(chanMap, 0),
		// writeChannel: make(chan []byte, 100),
	}
}
