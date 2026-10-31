package client

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"strings"
	"time"

	"github.com/sekret01/sekret_go_proxy/internal/core"
	clientDns "github.com/sekret01/sekret_go_proxy/internal/dns"
	"github.com/sekret01/sekret_go_proxy/internal/proxy"
	"github.com/sekret01/sekret_go_proxy/internal/utils"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

// Туннель для единого соединения с сервером
type ClientProxy struct {
	mainTunnelEngine   *TunnelEngine
	filterTunnelEngine *TunnelEngine
	filter             core.Filter
	detector           core.Detector // Определение протокола
	dispatcher         core.Dispatcher
	localAddr          string // Адрес текущего узла
	logger             logger.Logger
	listener           net.Listener // Слушатель внешних поключений
	ipMAp              *clientDns.IPDomainMap
}

func (c *ClientProxy) GetStatus() *proxy.TunnelStatus {
	return c.mainTunnelEngine.GetStatus()
}

// Запуск соединения между клиентом и удаленным узлом
func (c *ClientProxy) Start() error {
	go c.startFilterTunnel()
	return c.startMainTunnel()
}

func (c *ClientProxy) startMainTunnel() error {
	if c.mainTunnelEngine.IsRunning() {
		c.logger.Warning("[Start] Trying to start running service, return")
		return nil
	}
	c.mainTunnelEngine.SetLaunch()
	for c.mainTunnelEngine.IsRunning() {
		err := c.mainTunnelEngine.ConnectToTunnel()
		if err != nil {
			return err
		}

		c.logger.Info("Start listen on " + c.localAddr)
		listener, err := c.mainTunnelEngine.transport.Listen(c.localAddr)
		if err != nil {
			c.mainTunnelEngine.SetStopped()
			c.logger.Error("Tunnel connection error: " + err.Error())
			return err
		}
		go c.listenTunnelMessages(c.mainTunnelEngine)
		c.connectionsListener(listener)
		c.logger.Info("Stop tunnel and connections listening")
	}
	return nil
}

func (c *ClientProxy) startFilterTunnel() error {
	if c.filterTunnelEngine.IsRunning() {
		c.logger.Warning("[Start] Trying to start running service, return")
		return nil
	}
	c.filterTunnelEngine.SetLaunch()
	for c.filterTunnelEngine.IsRunning() {
		err := c.filterTunnelEngine.ConnectToTunnel()
		if err != nil {
			return err
		}
		go c.listenTunnelMessages(c.filterTunnelEngine)
		for c.filterTunnelEngine.IsRunning() {
			time.Sleep(100 * time.Millisecond) // TODO костыль, заменить на Done()
		}
	}
	return nil
}

func (c *ClientProxy) Stop() error {
	var totalErr error = nil
	if res := c.mainTunnelEngine.stopListening(true); res != nil {
		c.logger.Error("[Stop] Stop listening mainTunnelEngine error: " + res.Error())
		totalErr = res
	}
	// if res := c.filterTunnelEngine.stopListening(true); res != nil {
	// 	c.logger.Error("[Stop] Stop listening filterTunnelEngine error: " + res.Error())
	// 	totalErr = res
	// }
	c.CloseListen()
	return totalErr
}

func (c *ClientProxy) CloseListen() {
	if c.listener != nil {
		c.listener.Close()
		c.listener = nil
	}
}

func (s *ClientProxy) NewHandshakeContext() *core.HandshakeContext {
	return &core.HandshakeContext{}
}

func (c *ClientProxy) connectionsListener(listener net.Listener) error {
	c.listener = listener
	for c.mainTunnelEngine.IsRunning() {
		requestConn, err := c.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				c.logger.Info("[connectionsListener] Close listener")
				return nil
			}
			if c.mainTunnelEngine.IsRunning() {
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
	if !c.mainTunnelEngine.IsRunning() {
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
	if len(parsedHeaderFirstString) < 2 {
		c.logger.Debug("[handlerHttpProto] Error: header has not method or target: " + headerFirstString)
		requestConn.Close()
		return
	}
	method := parsedHeaderFirstString[0]
	target := parsedHeaderFirstString[1]

	if method == "CONNECT" {
		host, _, err := net.SplitHostPort(target)
		if err != nil {
			c.logger.Debug("[handlerHttpProto]: Target str " + host + " not ip")
			host = target
		}
		filter := false

		if ip := net.ParseIP(host); ip != nil {
			if c.ipMAp == nil {
				c.logger.Debug("[handlerHttpProto]: No ipMap to get " + host)
			} else if domain, ok := c.ipMAp.Get(host); ok {
				c.logger.Debug("[handlerHttpProto]: Check target " + host + " like ip (ipMap)")
				c.logger.Debug("[handlerHttpProto]: IP " + host + " found in ipMap")
				if c.filter.Check(domain) {
					filter = true
				}
			} else {
				c.logger.Debug("[handlerHttpProto]: IP " + host + "fnot ound in ipMap")
			}
		} else {
			c.logger.Debug("[handlerHttpProto]: Check target " + host + " like domain")
			filter = c.filter.Check(host)
		}

		requestConn.Write([]byte(core.HttpConnectResponse))
		c.handleTunnelConnect(requestConn, host, filter) // TODO-DEL target <- host
		return
	} else {
		requestId := c.generateAndRegistrateId(requestConn, core.ProtoHTTP, false)
		prepTarget := strings.Replace(target, "http://", "", 1)
		prepTarget = prepTarget[:len(prepTarget)-1] + ":80"
		filter := c.filter.Check(prepTarget)
		c.distributeTunnelAndSend(requestId, core.MsgConnect, []byte(prepTarget), filter)

		buf := make([]byte, 32*1024)
		n, _ := reader.Read(buf)
		c.logger.Debug("Get http buf: " + utils.BytesToString(buf, 20))

		c.distributeTunnelAndSend(requestId, core.MsgData, append([]byte(headerFirstString), buf[:n]...), filter)
		return
	}
}

func (c *ClientProxy) handleTunnelConnect(requestConn net.Conn, targetHost string, filter bool) {
	c.logger.Debug("[handleTunnelConnect] Start handle for " + targetHost)
	requestId := c.generateAndRegistrateId(requestConn, core.ProtoHTTP, true)
	err := c.distributeTunnelAndSend(requestId, core.MsgConnect, []byte(targetHost), filter)
	if err != nil {
		requestConn.Close()
		c.dispatcher.Delete(requestId)
		return
	}
	c.listenFromClientForTunnel(requestId, requestConn, filter)
}

// Цикличное чтение данных с клиента (при HTTPS)
func (c *ClientProxy) listenFromClientForTunnel(requestId core.RequestID, requestConn net.Conn, filter bool) {
	defer requestConn.Close()
	defer c.dispatcher.Delete(requestId)

	buf := make([]byte, 32*1024)
	for c.mainTunnelEngine.IsRunning() {
		size, err := requestConn.Read(buf)
		if err != nil {
			c.logger.Debug("Client [ " + utils.RequestIdToString(requestId) + " ] disconnect with error: " + err.Error())
			c.distributeTunnelAndSendClose(requestId, filter)
			return
		}
		if size == 0 {
			c.logger.Debug("Client [ " + utils.RequestIdToString(requestId) + " ] disconnect")
			c.distributeTunnelAndSendClose(requestId, filter)
			return
		}
		c.distributeTunnelAndSend(requestId, core.MsgData, buf[:size], filter)
	}
}

func (c *ClientProxy) listenTunnelMessages(tunnelEngine *TunnelEngine) {
	for {
		select {
		case <-tunnelEngine.Done():
			c.logger.Info("[listenTunnelMessages] get Done from tunnel, exit listen")
			return
		case msg := <-tunnelEngine.tunnelMessages:
			conWrapper, ok := c.dispatcher.Find(msg.RequestId)
			if !ok {
				c.logger.Debug("[listenTunnelMessages] ERROR: not found connection [" + utils.RequestIdToString(msg.RequestId) + "]")
				continue
			}
			requestCon := conWrapper.Conn
			c.logger.Debug("[listenTunnelMessages] Get data [ " + utils.RequestIdToString(msg.RequestId) + " ], msg_type: " + utils.MessageTypeToHexString(msg.MsgType))
			switch msg.MsgType {
			case core.MsgData:
				n := 10
				if len(msg.Data) < n {
					n = len(msg.Data)
				}
				c.logger.Debug("[listenTunnelMessages] send data [" + utils.BytesToString(msg.Data, 20) + "...]")
				requestCon.Write(msg.Data)
				if !conWrapper.IsTunnel {
					dataStr := string(msg.Data)
					if strings.HasSuffix(dataStr, "0\r\n\r\n") ||
						strings.Contains(dataStr, "Connection: close") {
						c.logger.Debug("[listenTunnelMessages] close not tunnel and not keep-alive connection [" + utils.RequestIdToString(msg.RequestId) + "]")
						tunnelEngine.sendCloseIntoTunnel(msg.RequestId)
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
}

func (c *ClientProxy) generateAndRegistrateId(requestConn net.Conn, protoType core.ProtocolType, isTunnel bool) core.RequestID {
	requestId := core.GenerateID()
	c.dispatcher.Register(requestId, requestConn, protoType, isTunnel)
	c.logger.Debug("Save conn with UUID " + utils.RequestIdToString(requestId))
	return requestId
}

func (c *ClientProxy) distributeTunnelAndSend(requestId core.RequestID, msgType core.MessageType, payload []byte, filter bool) error {
	if !filter {
		return c.mainTunnelEngine.sendIntoTunnel(requestId, msgType, payload)
	}
	return c.filterTunnelEngine.sendIntoTunnel(requestId, msgType, payload)
}

func (c *ClientProxy) distributeTunnelAndSendClose(requestId core.RequestID, filter bool) {
	if filter {
		c.filterTunnelEngine.sendCloseIntoTunnel(requestId)
	} else {
		c.mainTunnelEngine.sendCloseIntoTunnel(requestId)
	}
}

func NewClientProxy(
	mainTunnelEngine *TunnelEngine,
	filterTunnelEngine *TunnelEngine,
	filter core.Filter,
	detector core.Detector,
	dispatcher core.Dispatcher,
	localAddr string,
	logger logger.Logger,
	ipMAp *clientDns.IPDomainMap,
) *ClientProxy {
	client := &ClientProxy{
		mainTunnelEngine:   mainTunnelEngine,
		filterTunnelEngine: filterTunnelEngine,
		filter:             filter,
		detector:           detector,
		dispatcher:         dispatcher,
		localAddr:          localAddr,
		logger:             logger,
		listener:           nil,
		ipMAp:              ipMAp,
	}
	mainTunnelEngine.RegistrateListener(client)
	return client
}
