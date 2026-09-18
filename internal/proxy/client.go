package proxy

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/utils"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

// Туннель для единого соединения с сервером
type ClientTunnel struct {
	transport  core.Transport  // Подключение и передача информации
	encryptor  core.Encryptor  // Шифрование и расшифровка
	framer     core.Framer     // Оборачивание и парсинг самописных протоколов
	dispatcher core.Dispatcher // Сохранение связи запрос - владелец запроса
	detector   core.Detector   // Определение протокола
	auth       core.Auth

	remoteAddr string // Адрес удаленного узла для подключения
	localAddr  string // Адрес текущего узла

	logger logger.Logger
	mutex  sync.Mutex

	serverTonnelConn net.Conn     // Туннельное подключение к серверу
	listener         net.Listener // Слушатель внешних поключений
	running          bool         // Состояние работы
}

// Запуск соединения между клиентом и удаленным узлом
func (c *ClientTunnel) Start() error {
	if c.running {
		c.logger.Warning("[Start] Trying to start running service, return")
		return nil
	}
	c.running = true
	for c.running {
		conn, err := c.waitConnectionToTunnel()
		if err != nil {
			c.running = false
			c.logger.Error("[ClientTunnel] :: connect remote addr -> " + err.Error())
			return err
		}
		c.logger.Info("Try authenticate")
		_, err = c.auth.ClientHandshake(conn)
		if err != nil {
			c.running = false
			c.logger.Error(err.Error())
			return err
		}
		c.serverTonnelConn = conn
		go c.tunnelReader()

		c.logger.Info("Start listen on " + c.localAddr)
		listener, err := c.transport.Listen(c.localAddr)
		if err != nil {
			c.running = false
			return err
		}
		c.connectionsListener(listener)
		c.logger.Info("Stop tunnel and connections listening")
	}
	return nil
}

func (c *ClientTunnel) Stop() error {
	if !c.running {
		c.logger.Warning("[Stop] Trying to stop stopped service, cencel")
		return nil
	}
	c.running = false
	if c.serverTonnelConn != nil {
		c.serverTonnelConn.Close()
		c.serverTonnelConn = nil
	}
	if c.listener != nil {
		c.listener.Close()
		c.listener = nil
	}
	c.logger.Info("[Stop] Stop tunnel")
	return nil
}

func (c *ClientTunnel) IsRunning() bool {
	return c.running
}

func (c *ClientTunnel) waitConnectionToTunnel() (net.Conn, error) {
	c.logger.Info("[WatiConnection] :: waiting tunnel connection")
	for {
		conn, err := c.transport.Dial(c.remoteAddr)
		if err == nil {
			c.logger.Info("[WatiConnection] :: tunnel found")
			return conn, nil
		}
		time.Sleep(time.Second * 5) // TODO вынести в конфиг
	}
}

func (c *ClientTunnel) connectionsListener(listener net.Listener) error {
	c.listener = listener
	for c.running {
		requestConn, err := c.listener.Accept()
		if err != nil {
			if c.running {
				continue
			} else {
				c.logger.Info("[ClientTunnel] Close listener (" + err.Error() + ")")
				return err
			}
		}
		go c.newConnectionHandler(requestConn)
	}
	return nil
}

// Обработка новых входящих запросов
// Чтение - получение протокола - шифрование - оборачивание в пакет - отправление
func (c *ClientTunnel) newConnectionHandler(requestConn net.Conn) {
	if !c.running {
		return
	}
	c.logger.Debug("New connection: " + requestConn.RemoteAddr().String())

	reader := bufio.NewReader(requestConn)
	data, err := reader.ReadString('\n')
	if err != nil {
		if err != io.EOF {
			c.logger.Error("Error in read data: " + err.Error())
		}
		requestConn.Close()
		return
	}
	c.logger.Debug("Get data: " + utils.BytesToString([]byte(data), 20))

	reader = bufio.NewReader(io.MultiReader(bytes.NewReader([]byte(data)), reader))

	prefixSize := 8
	if len(data) < 4 {
		requestConn.Close()
		return
	}

	prefixSize = min(len(data), 8)

	proto, err := c.detector.Detect([]byte(data[:prefixSize]))
	if err != nil {
		c.logger.Error("Can not detect protocol: " + err.Error())
		requestConn.Close()
		return
	}
	c.switchProtocol(proto, requestConn, reader)
}

func (c *ClientTunnel) switchProtocol(proto core.ProtocolType, requestConn net.Conn, reader *bufio.Reader) {
	switch proto {
	case core.ProtoUnknown:
		c.logger.Debug("Get uncknown protocol, close")
		requestConn.Close()
		return
	case core.ProtoSOCKS5:
		c.logger.Debug("Get SOCKS5 protocol, close")
		requestConn.Close()
		return
	case core.ProtoHTTP:
		c.logger.Debug("Get HTTP protocol, handle")
		c.handlerHttpProto(reader, requestConn)
		requestConn.Close()
		return
	}
}

func (c *ClientTunnel) handlerHttpProto(reader *bufio.Reader, requestConn net.Conn) {
	headerFirstString, err := reader.ReadString('\n')
	if err != nil {
		c.logger.Error("[handlerHttpProto] Error in read data: " + err.Error())
		requestConn.Close()
		return
	}
	c.logger.Debug("[handlerHttpProto] get header: \n" + strings.Trim(string(headerFirstString), " \n"))

	parsedHeaderFirstString := strings.Fields(headerFirstString)
	method := parsedHeaderFirstString[0]
	target := parsedHeaderFirstString[1]

	if method == "CONNECT" {
		requestConn.Write([]byte(core.HttpConnectResponse))
		c.handleTunnelConnect(requestConn, target)
		return
	} else {
		requestId := c.generateAndRegistrateId(requestConn, core.ProtoHTTP, false)
		prepTarget := strings.Replace(target, "http://", "", 1)
		prepTarget = prepTarget[:len(prepTarget)-1] + ":80"
		c.sendIntoTunnel(requestId, core.MsgConnect, []byte(prepTarget))

		buf := make([]byte, 32*1024)
		n, _ := reader.Read(buf)
		c.logger.Debug("Get http buf: " + utils.BytesToString(buf, 20))

		c.sendIntoTunnel(requestId, core.MsgData, append([]byte(headerFirstString), buf[:n]...))
		return
	}
}

// Отрпавление запроса на подключение сервером к targetHost при HTTPS запросе
func (c *ClientTunnel) handleTunnelConnect(requestConn net.Conn, targetHost string) {
	c.logger.Debug("[handleTunnelConnect] Start handle for " + targetHost)
	requestId := c.generateAndRegistrateId(requestConn, core.ProtoHTTP, true)
	err := c.sendIntoTunnel(requestId, core.MsgConnect, []byte(targetHost))
	if err != nil {
		requestConn.Close()
		c.dispatcher.Delete(requestId)
		return
	}
	c.listenFromClientForTunnel(requestId, requestConn)
}

// Цикличное чтение данных с клиента (при HTTPS)
func (c *ClientTunnel) listenFromClientForTunnel(requestId core.RequestID, requestConn net.Conn) {
	defer requestConn.Close()
	defer c.dispatcher.Delete(requestId)

	buf := make([]byte, 32*1024)
	for c.running {
		size, err := requestConn.Read(buf)
		if err != nil {
			c.logger.Debug("Client [ " + utils.RequestIdToString(requestId) + " ] disconnect with error: " + err.Error())
			c.sendCloseIntoTunnel(requestId)
			return
		}
		if size == 0 {
			c.logger.Debug("Client [ " + utils.RequestIdToString(requestId) + " ] disconnect")
			c.sendCloseIntoTunnel(requestId)
			return
		}
		c.sendIntoTunnel(requestId, core.MsgData, buf[:size])
	}
}

// Чтение, обработка и перессылка данных с сервера
func (c *ClientTunnel) tunnelReader() {
	c.logger.Info("[tunnelReader] Start tunnel listening")

	for c.running {
		frame, err := ReadFrameFromConnection(c.serverTonnelConn, c.framer)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				c.logger.Warning("[tunnelReader] Close tunel reader: (" + err.Error() + ")")
			} else {
				c.logger.Error("[tunnelReader] ERROR in read frame: " + err.Error())
				fmt.Printf("%#v\n", err)
			}
			c.Stop()
			return
		}
		data, msgType, requestId, err := c.framer.Unframe(frame)
		if err != nil {
			c.logger.Error("[tunnelReader] ERROR: unframe: " + err.Error())
			continue
		}
		decryptData := c.encryptor.Decrypt(data)

		conWrapper, ok := c.dispatcher.Find(requestId)
		requestCon := conWrapper.Conn
		if !ok {
			c.logger.Debug("[tunnelReader] ERROR: not found connection [" + utils.RequestIdToString(requestId) + "]")
			continue
		}
		c.logger.Debug("[tunnelReader] Get data [ " + utils.RequestIdToString(requestId) + " ], msg_type: " + utils.MessageTypeToHexString(msgType))
		switch msgType {
		case core.MsgData:
			n := 10
			if len(decryptData) < n {
				n = len(decryptData)
			}
			c.logger.Debug("[tunnelReader] send data [" + utils.BytesToString(decryptData, 20) + "...]")
			requestCon.Write(decryptData)
			if !conWrapper.IsTunnel {
				dataStr := string(decryptData)
				if strings.HasSuffix(dataStr, "0\r\n\r\n") ||
					strings.Contains(dataStr, "Connection: close") {
					c.logger.Debug("[tunnelReader] close not tunnel and not keep-alive connection [" + utils.RequestIdToString(requestId) + "]")
					c.sendCloseIntoTunnel(requestId)
					closeConnection(c.dispatcher, requestId)
				}

			}
		case core.MsgClose:
			closeConnection(c.dispatcher, requestId)
		case core.MsgError:
			requestCon.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
			closeConnection(c.dispatcher, requestId)
		}
	}
}

func (c *ClientTunnel) sendIntoTunnel(requestId core.RequestID, msgType core.MessageType, payload []byte) error {
	if !c.running {
		return nil
	}
	c.logger.Debug("[sendIntoTunnel] Prepeare new msg: requestId: [" + utils.RequestIdToString(requestId) + "], msgType: [" + utils.MessageTypeToHexString(msgType) + "], msg: [" + utils.BytesToString(payload, 20) + "]")
	payloadEncode := c.encryptor.Encrypt([]byte(payload))
	frame, err := c.framer.Frame(payloadEncode, msgType, requestId)
	if err != nil {
		c.logger.Error("[sendIntoTunnel] Error in send data: " + err.Error())
		return err
	}
	// c.mutex.Lock()
	// defer c.mutex.Unlock()
	_, err = c.serverTonnelConn.Write(frame)
	c.logger.Debug("[sendIntoTunnel] Data has been sent")
	return nil
}

func (c *ClientTunnel) sendCloseIntoTunnel(requestId core.RequestID) {
	c.logger.Debug("[sendCloseIntoTunnel] close " + utils.RequestIdToString(requestId))
	c.sendIntoTunnel(requestId, core.MsgClose, []byte{})
}

func (c *ClientTunnel) generateAndRegistrateId(requestConn net.Conn, protoType core.ProtocolType, isTunnel bool) core.RequestID {
	requestId := core.GenerateID()
	c.dispatcher.Register(requestId, requestConn, protoType, isTunnel)
	c.logger.Debug("Save conn with UUID " + utils.RequestIdToString(requestId))
	return requestId
}

func NewClientTunnel(
	transport core.Transport,
	encryptor core.Encryptor,
	framer core.Framer,
	dispatcher core.Dispatcher,
	detector core.Detector,
	cfg *config.Config,
	logger logger.Logger,
	auth core.Auth) *ClientTunnel {
	return &ClientTunnel{
		transport:        transport,
		encryptor:        encryptor,
		framer:           framer,
		dispatcher:       dispatcher,
		detector:         detector,
		running:          false,
		serverTonnelConn: nil,
		remoteAddr:       cfg.RemoteHost,
		localAddr:        cfg.LocalHost,
		logger:           logger,
		auth:             auth,
	}
}
