package client

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"strings"

	// "sync"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/proxy"
	"github.com/sekret01/sekret_go_proxy/internal/utils"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

// Туннель для единого соединения с сервером
type ClientProxy struct {
	tunnelEngin *TunnelEngine
	detector    core.Detector // Определение протокола
	dispatcher  core.Dispatcher
	localAddr   string // Адрес текущего узла
	logger      logger.Logger
	listener    net.Listener // Слушатель внешних поключений
}

func (c *ClientProxy) GetStatus() *proxy.TunnelStatus {
	return c.tunnelEngin.GetStatus()
}

// Запуск соединения между клиентом и удаленным узлом
func (c *ClientProxy) Start() error {
	if c.tunnelEngin.IsRunning() {
		c.logger.Warning("[Start] Trying to start running service, return")
		return nil
	}
	c.tunnelEngin.SetLaunch() // TODO сделать отедльные функции в TunnelEngine
	for c.tunnelEngin.IsRunning() {
		err := c.tunnelEngin.ConnectToTunnel()
		if err != nil {
			return err
		}

		c.logger.Info("Start listen on " + c.localAddr)
		listener, err := c.tunnelEngin.transport.Listen(c.localAddr)
		if err != nil {
			c.tunnelEngin.SetStopped()
			c.logger.Error("Tunnel connection error: " + err.Error())
			return err
		}
		go c.listenTunnelMessages()
		c.connectionsListener(listener)
		c.logger.Info("Stop tunnel and connections listening")
	}
	return nil
}

func (c *ClientProxy) Stop() error {
	res := c.tunnelEngin.stopListening(true)
	c.CloseListen()
	return res
}

func (c *ClientProxy) CloseListen() {
	if c.listener != nil {
		c.listener.Close()
		c.listener = nil
	}
}

func (c *ClientProxy) connectionsListener(listener net.Listener) error {
	c.listener = listener
	for c.tunnelEngin.IsRunning() {
		requestConn, err := c.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				c.logger.Info("[connectionsListener] Close listener")
				return nil
			}
			if c.tunnelEngin.IsRunning() {
				continue
			} else {
				c.logger.Info("[connectionsListener] Close listener (" + err.Error() + ")")
				return err
			}
		}
		go c.newConnectionHandler(requestConn)
	}
	return nil
}

// Обработка новых входящих запросов
// Чтение - получение протокола - шифрование - оборачивание в пакет - отправление
func (c *ClientProxy) newConnectionHandler(requestConn net.Conn) {
	if !c.tunnelEngin.IsRunning() {
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

func (c *ClientProxy) switchProtocol(proto core.ProtocolType, requestConn net.Conn, reader *bufio.Reader) {
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
	default:
		c.logger.Debug("Protocol not found, close")
		return
	}
}

func (c *ClientProxy) handlerHttpProto(reader *bufio.Reader, requestConn net.Conn) {
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
		c.tunnelEngin.sendIntoTunnel(requestId, core.MsgConnect, []byte(prepTarget))

		buf := make([]byte, 32*1024)
		n, _ := reader.Read(buf)
		c.logger.Debug("Get http buf: " + utils.BytesToString(buf, 20))

		c.tunnelEngin.sendIntoTunnel(requestId, core.MsgData, append([]byte(headerFirstString), buf[:n]...))
		return
	}
}

// Отрпавление запроса на подключение сервером к targetHost при HTTPS запросе
func (c *ClientProxy) handleTunnelConnect(requestConn net.Conn, targetHost string) {
	c.logger.Debug("[handleTunnelConnect] Start handle for " + targetHost)
	requestId := c.generateAndRegistrateId(requestConn, core.ProtoHTTP, true)
	err := c.tunnelEngin.sendIntoTunnel(requestId, core.MsgConnect, []byte(targetHost))
	if err != nil {
		requestConn.Close()
		c.dispatcher.Delete(requestId) // TODO решить что с диспетчером делать
		return
	}
	c.listenFromClientForTunnel(requestId, requestConn)
}

// Цикличное чтение данных с клиента (при HTTPS)
func (c *ClientProxy) listenFromClientForTunnel(requestId core.RequestID, requestConn net.Conn) {
	defer requestConn.Close()
	defer c.dispatcher.Delete(requestId)

	buf := make([]byte, 32*1024)
	for c.tunnelEngin.IsRunning() {
		size, err := requestConn.Read(buf)
		if err != nil {
			c.logger.Debug("Client [ " + utils.RequestIdToString(requestId) + " ] disconnect with error: " + err.Error())
			c.tunnelEngin.sendCloseIntoTunnel(requestId)
			return
		}
		if size == 0 {
			c.logger.Debug("Client [ " + utils.RequestIdToString(requestId) + " ] disconnect")
			c.tunnelEngin.sendCloseIntoTunnel(requestId)
			return
		}
		c.tunnelEngin.sendIntoTunnel(requestId, core.MsgData, buf[:size])
	}
}

func (c *ClientProxy) listenTunnelMessages() {
	for msg := range c.tunnelEngin.tunnelMessages {
		conWrapper, ok := c.dispatcher.Find(msg.RequestId)
		requestCon := conWrapper.Conn
		if !ok {
			c.logger.Debug("[tunnelReader] ERROR: not found connection [" + utils.RequestIdToString(msg.RequestId) + "]")
			continue
		}
		c.logger.Debug("[tunnelReader] Get data [ " + utils.RequestIdToString(msg.RequestId) + " ], msg_type: " + utils.MessageTypeToHexString(msg.MsgType))
		switch msg.MsgType {
		case core.MsgData:
			n := 10
			if len(msg.Data) < n {
				n = len(msg.Data)
			}
			c.logger.Debug("[tunnelReader] send data [" + utils.BytesToString(msg.Data, 20) + "...]")
			requestCon.Write(msg.Data)
			if !conWrapper.IsTunnel {
				dataStr := string(msg.Data)
				if strings.HasSuffix(dataStr, "0\r\n\r\n") ||
					strings.Contains(dataStr, "Connection: close") {
					c.logger.Debug("[tunnelReader] close not tunnel and not keep-alive connection [" + utils.RequestIdToString(msg.RequestId) + "]")
					c.tunnelEngin.sendCloseIntoTunnel(msg.RequestId)
					proxy.CloseConnection(c.dispatcher, msg.RequestId)
				}

			}
		case core.MsgClose:
			proxy.CloseConnection(c.dispatcher, msg.RequestId)
		case core.MsgError:
			requestCon.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
			proxy.CloseConnection(c.dispatcher, msg.RequestId)
		}
	}
}

func (c *ClientProxy) generateAndRegistrateId(requestConn net.Conn, protoType core.ProtocolType, isTunnel bool) core.RequestID {
	requestId := core.GenerateID()
	c.dispatcher.Register(requestId, requestConn, protoType, isTunnel)
	c.logger.Debug("Save conn with UUID " + utils.RequestIdToString(requestId))
	return requestId
}

func NewClientProxy(
	tunnelEngin *TunnelEngine,
	detector core.Detector,
	dispatcher core.Dispatcher,
	cfg *config.Config,
	logger logger.Logger,
) *ClientProxy {
	client := &ClientProxy{
		tunnelEngin: tunnelEngin,
		detector:    detector,
		dispatcher:  dispatcher,
		localAddr:   cfg.LocalHost,
		logger:      logger,
		listener:    nil,
	}
	tunnelEngin.RegistrateListener(client)
	return client
}
