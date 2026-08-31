package proxy

import (
	"io"
	"net"
	"strconv"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
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
	bufHeader := make([]byte, s.framer.HeaderSize())

	for {
		// HEADER
		if _, err := io.ReadFull(s.tonnelConn, bufHeader); err != nil {
			s.logger.Debug("[tunnelReader] CRITIACL ERROR: error in read tunnel (header): " + err.Error())
			return
		}
		s.logger.Debug("[tunnelReader] get header: " + string(bufHeader))

		// PAYLOAD
		payloadSize, err := s.framer.GetPayloadSize(bufHeader)
		if err != nil {
			s.logger.Debug("[tunnelReader] ERROR: cannot get payload size (header): " + err.Error())
			return
		}
		s.logger.Debug("[tunnelReader] wait payload size: " + strconv.Itoa(payloadSize))

		bufPayload := make([]byte, payloadSize) // TODO изменять в конфигах bufSize
		if _, err := io.ReadFull(s.tonnelConn, bufPayload); err != nil {
			s.logger.Debug("[tunnelReader] ERROR: cannot read payload (payload): " + err.Error())
			return
		}

		frame := append(bufHeader, bufPayload...)
		data, msgType, requestId, err := s.framer.Unframe(frame)
		decryptData := s.encryptor.Decrypt(data)
		if err != nil {
			s.logger.Error("[tunnelReader] ERROR: unframe: " + err.Error())
			continue
		}

		switch msgType {

		// Подключение к target серверу
		case core.MsgConnect:
			targetCon, err := s.transport.Dial(string(decryptData))
			if err != nil {
				s.logger.Error("[tunnelReader] Can not connect to target server [" + string(decryptData) + "] " + err.Error())
				continue
			}
			s.dispatcher.Register(requestId, targetCon, core.ProtoTest, true)
			// Обработка получения данных из conTarget
			go s.targetConnectinoHandler(requestId, targetCon)

		case core.MsgData:
			conWrapper, ok := s.dispatcher.Find(requestId)
			if !ok {
				s.logger.Warning("[tunnelReader] ERROR: not found connection [" + string(requestId[:]) + "]")
				continue
			}
			targetCon := conWrapper.Conn

			// Упростить лог
			// ====================
			n := 10
			if len(decryptData) < n {
				n = len(decryptData)
			}
			s.logger.Debug("[tunnelReader] send data [" + string(decryptData[:n]) + "...]")
			// ====================
			targetCon.Write(decryptData)

		case core.MsgClose:
			conWrapper, ok := s.dispatcher.Find(requestId)
			if !ok {
				s.logger.Warning("[tunnelReader] ERROR: not found connection [" + string(requestId[:]) + "]")
				continue
			}
			conWrapper.Conn.Close()
			s.dispatcher.Delete(requestId)

			// case core.MsgError:
			// 	requestCon.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
			// 	requestCon.Close()
			// 	s.dispatcher.Delete(requestId)
		}

	}
}

// Чтение ответов из подключения к target-серверу
func (s *ServerTunnel) targetConnectinoHandler(requestId core.RequestID, targetCon net.Conn) {
	s.logger.Debug("Create new connection: " + targetCon.RemoteAddr().String())
	defer targetCon.Close()
	defer s.dispatcher.Delete(requestId)

	buf := make([]byte, 32*1024)
	for {
		size, err := targetCon.Read(buf)
		if err != nil {
			s.logger.Warning("Client disconnect with error: " + err.Error())
			s.sendCloseIntoTunnel(requestId)
			s.dispatcher.Delete(requestId)
			return
		}
		if size == 0 {
			s.logger.Warning("Client disconnect")
			s.sendCloseIntoTunnel(requestId)
			s.dispatcher.Delete(requestId)
			return
		}
		s.sendIntoTunnel(requestId, core.MsgData, buf[:size])
	}
}

func (s *ServerTunnel) sendIntoTunnel(requestId core.RequestID, msgType core.MessageType, payload []byte) error {
	s.logger.Debug("[sendIntoTunnel] Prepeare new msg: requestId: [" + string(requestId[:]) + "], msgType: [" + string(msgType) + "], msg: [" + string(payload) + "]")
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
	s.logger.Debug("[sendCloseIntoTunnel] close " + string(requestId[:]))
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
		localAddr:  cfg.RemoteHost + ":" + strconv.Itoa(cfg.RemotePort),
		logger:     logger,
	}
}
