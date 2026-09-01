package proxy

import (
	"io"
	"net"
	"strconv"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/utils"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type ServerTunnel struct {
	transport  core.Transport  // Подключение и передача информации
	encryptor  core.Encryptor  // Шифрование и расшифровка
	framer     core.Framer     // Оборачивание и парсинг самописных протоколов
	dispatcher core.Dispatcher // Сохранение связи запрос - владелец запроса
	detector   core.Detector   // Определение протокола

	localAddr string // Адрес текущего узла

	logger logger.Logger

	tonnelConn net.Conn // Туннельное подключение к серверу
	running    bool     // Состояние работы
}

// Запуск сервера приема данных
func (s *ServerTunnel) Start() error {
	listener, err := s.transport.Listen(s.localAddr)
	s.logger.Info("Start listen on " + s.localAddr)
	if err != nil {
		return err
	}
	// TODO нужен ли FOR, подумать об аутентификации
	for {
		con, err := listener.Accept()
		if err != nil {
			s.logger.Error("[ServerTunnel] Error in connection accept -> " + err.Error())
			if s.running {
				continue
			} else {
				return nil
			}
		}
		s.tonnelConn = con
		go s.tunnelReader()
	}
}

// Чтение, обработка и перессылка данных с клиентских узлов
func (s *ServerTunnel) tunnelReader() {
	s.logger.Info("[tunnelReader] Start tunnel listening")

	for {
		frame, err := ReadFrameFromConnection(s.tonnelConn, s.framer)
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
			targetCon, err := s.transport.Dial(string(decryptData))
			if err != nil {
				s.logger.Error("[tunnelReader] Can not connect to target server [" + utils.BytesToString(decryptData, 20) + "] " + err.Error())
				continue
			}
			s.dispatcher.Register(requestId, targetCon, core.ProtoTest, true)
			go s.targetConnectinoHandler(requestId, targetCon)
		case core.MsgData:
			conWrapper, ok := s.dispatcher.Find(requestId)
			if !ok {
				s.logger.Debug("[tunnelReader] ERROR: not found connection [" + utils.RequestIdToString(requestId) + "]")
				continue
			}
			targetCon := conWrapper.Conn
			s.logger.Debug("[tunnelReader] send data [" + utils.BytesToString(decryptData, 20) + "...]")
			targetCon.Write(decryptData)
		case core.MsgClose:
			conWrapper, ok := s.dispatcher.Find(requestId)
			if !ok {
				s.logger.Debug("[tunnelReader] ERROR: not found connection [" + utils.RequestIdToString(requestId) + "]")
				continue
			}
			conWrapper.Conn.Close()
			closeConnection(s.dispatcher, requestId)
		}
	}
}

// Чтение ответов из подключения к target-серверу
func (s *ServerTunnel) targetConnectinoHandler(requestId core.RequestID, targetCon net.Conn) {
	s.logger.Debug("Create new connection: " + targetCon.RemoteAddr().String())

	buf := make([]byte, 32*1024)
	for {
		size, err := targetCon.Read(buf)
		if err != nil {
			if err != io.EOF && err == net.ErrClosed {
				s.logger.Warning("Client disconnect with error: " + err.Error())
			}
			closeConnection(s.dispatcher, requestId)
			return
		}
		if size == 0 {
			s.logger.Debug("Client disconnect")
			closeConnection(s.dispatcher, requestId)
			return
		}
		s.sendIntoTunnel(requestId, core.MsgData, buf[:size])
	}
}

func (s *ServerTunnel) sendIntoTunnel(requestId core.RequestID, msgType core.MessageType, payload []byte) error {
	s.logger.Debug("[sendIntoTunnel] Prepeare new msg: requestId: [" + utils.RequestIdToString(requestId) + "], msgType: [" + utils.MessageTypeToHexString(msgType) + "], msg: [" + utils.BytesToString(payload, 20) + "]")
	payloadEncode := s.encryptor.Encrypt([]byte(payload))
	frame, err := s.framer.Frame(payloadEncode, msgType, requestId)
	if err != nil {
		s.logger.Error("[sendIntoTunnel] Error in send data: " + err.Error())
		return err
	}
	_, err = s.tonnelConn.Write(frame)
	s.logger.Debug("[sendIntoTunnel] Data has been sent")
	return nil
}

func (s *ServerTunnel) sendCloseIntoTunnel(requestId core.RequestID) {
	s.logger.Debug("[sendCloseIntoTunnel] close " + utils.RequestIdToString(requestId))
	s.sendIntoTunnel(requestId, core.MsgClose, []byte{})
}

func NewServerTunnel(
	transport core.Transport,
	encryptor core.Encryptor,
	framer core.Framer,
	dispatcher core.Dispatcher,
	detector core.Detector,
	cfg *config.Config,
	logger logger.Logger) *ServerTunnel {
	return &ServerTunnel{
		transport:  transport,
		encryptor:  encryptor,
		framer:     framer,
		dispatcher: dispatcher,
		detector:   detector,
		running:    false,
		tonnelConn: nil,
		localAddr:  cfg.LocalHost + ":" + strconv.Itoa(cfg.LocalPort),
		logger:     logger,
	}
}
