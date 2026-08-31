package proxy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
)

// Туннель для единого соединения с сервером
type ClientTunnel struct {
	transport  core.Transport  // Подключение и передача информации
	encryptor  core.Encryptor  // Шифрование и расшифровка
	framer     core.Framer     // Оборачивание и парсинг самописных протоколов
	dispatcher core.Dispatcher // Сохранение связи запрос - владелец запроса
	detector   core.Detector   // Определение протокола

	remoteAddr string // Адрес удаленного узла для подключения
	localAddr  string // Адрес текущего узла

	serverTonnelConn net.Conn // Туннельное подключение к серверу
	running          bool     // Состояние работы
}

// Запуск соединения между клиентом и удаленным узлом
func (c *ClientTunnel) Start() error {
	conn, err := c.transport.Dial(c.remoteAddr)
	if err != nil {
		fmt.Printf("[ClientTunnel] ERR: connect remote addr -> %s\n", err.Error())
		return err
	}
	c.serverTonnelConn = conn

	// запуск получения данных из туннеля
	go c.tunnelReader()

	listener, err := c.transport.Listen(c.localAddr)
	if err != nil {
		return err
	}

	// Мониторинг подключений
	for {
		requestConn, err := listener.Accept()
		if err != nil {
			fmt.Printf("[ClientTunnel] ERR: connectino accept -> %s\n", err.Error())
			if c.running {
				continue
			} else {
				return nil
			}
		}
		go c.newRequestConnectionHandler(requestConn)
	}
}

// Обработка новых входящих запросов
// Чтение - получение протокола - шифрование - оборачивание в пакет - отправление
func (c *ClientTunnel) newRequestConnectionHandler(requestConn net.Conn) {
	fmt.Printf("New connection: %s\n", requestConn.RemoteAddr().String())

	// Чтение
	reader := bufio.NewReader(requestConn)
	data, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error in read data: %s\n", err.Error())
		// c.dispatcher.Delete(requestId)
		requestConn.Close()
		return
	}
	fmt.Printf("Get data: %s\n", data)

	reader = bufio.NewReader(io.MultiReader(bytes.NewReader([]byte(data)), reader))

	prefixSize := 8
	if len(data) < 4 {
		fmt.Printf("Data len is %d, close\n", prefixSize)
		requestConn.Close()
		return

	} else if len(data) < 8 {
		prefixSize = len(data)
	}

	proto, err := c.detector.Detect([]byte(data[:prefixSize]))
	if err != nil {
		fmt.Printf("Error in detect protocol: %s\n, close", err.Error())
		requestConn.Close()
		return
	}

	switch proto {
	case core.ProtoUnknown:
		fmt.Printf("Get uncknown protocol, close")
		requestConn.Close()
		return
	case core.ProtoSOCKS5:
		fmt.Printf("Get SOCKS5 protocol, close")
		requestConn.Close()
		return
	case core.ProtoHTTP:
		fmt.Printf("Get HTTP protocol, handle")
		c.handlerHttpProto(reader, requestConn)
		requestConn.Close()
		return
	}

}

func (c *ClientTunnel) handlerHttpProto(reader *bufio.Reader, requestConn net.Conn) {
	header, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("[handlerHttpProto] Error in read data: %s\n", err.Error())
		requestConn.Close()
		return
	}
	fmt.Printf("[handlerHttpProto] header: [%#v]\n", header)

	tokens := strings.Fields(header)
	method := tokens[0]
	target := tokens[1]

	// if HTTPS
	if method == "CONNECT" {
		requestConn.Write([]byte(core.HttpConnectResponse))
		c.handleTunnelConnect(requestConn, target)
		return
	}

	// if HTTPS
	requestId := c.generateAndRegistrateId(requestConn, core.ProtoHTTP, false)
	c.sendIntoTunnel(requestId, core.MsgConnect, []byte(target))

	buf := make([]byte, 32*1024)
	n, err := reader.Read(buf)
	fmt.Printf("Get http buf[%d]: %#v\n", n, string(buf[:n]))

	c.sendIntoTunnel(requestId, core.MsgData, append([]byte(header), buf[:n]...))

}

// Отрпавление запроса на подключение сервером к targetHost при HTTPS запросе
func (c *ClientTunnel) handleTunnelConnect(requestConn net.Conn, targetHost string) {
	fmt.Printf("[handleTunnelConnect] Start handle for : %s\n", targetHost)
	requestId := c.generateAndRegistrateId(requestConn, core.ProtoHTTP, true)
	err := c.sendIntoTunnel(requestId, core.MsgConnect, []byte(targetHost))
	if err != nil {
		requestConn.Close()
		c.dispatcher.Delete(requestId)
		return
	}

	c.listenFromClient(requestId, requestConn)

	// TEST
	requestConn.Close()
	c.dispatcher.Delete(requestId)
}

// Цикличное чтение данных с клиента (при HTTPS)
func (c *ClientTunnel) listenFromClient(requestId core.RequestID, requestConn net.Conn) {
	defer requestConn.Close()
	defer c.dispatcher.Delete(requestId)

	buf := make([]byte, 32*1024)
	for {
		size, err := requestConn.Read(buf)
		if err != nil {
			fmt.Printf("Client %x disconnect with error: %s\n", requestId, err.Error())
			c.sendCloseIntoTunnel(requestId)
			return
		}
		if size == 0 {
			fmt.Printf("Client %x disconnect\n", requestId)
			c.sendCloseIntoTunnel(requestId)
			return
		}
		c.sendIntoTunnel(requestId, core.MsgData, buf[:size])
	}
}

// Чтение, обработка и перессылка данных с сервера
func (c *ClientTunnel) tunnelReader() {
	bufHeader := make([]byte, c.framer.HeaderSize())

	for {
		fmt.Printf("[tunnelReader] wait header size: %d\n", c.framer.HeaderSize())
		if _, err := io.ReadFull(c.serverTonnelConn, bufHeader); err != nil {
			fmt.Println("[tunnelReader] CRITIACL ERROR: error in read tunnel (header): " + err.Error())
			return
		}
		fmt.Printf("[tunnelReader] get header [%d]: %#v\n", len(bufHeader), string(bufHeader))

		payloadSize, err := c.framer.GetPayloadSize(bufHeader)
		if err != nil {
			fmt.Println("[tunnelReader] ERROR: cannot get payload size (header): " + err.Error())
			return
		}
		fmt.Printf("[tunnelReader] wait payload size: %d\n", payloadSize)

		bufPayload := make([]byte, payloadSize) // TODO изменять в конфигах bufSize
		if _, err := io.ReadFull(c.serverTonnelConn, bufPayload); err != nil {
			fmt.Println("[tunnelReader] ERROR: cannot read payload (payload): " + err.Error())
			return
		}

		fmt.Printf("[tunnelReader] appending: %#v + %#v\n", string(bufHeader), string(bufPayload))
		frame := append(bufHeader, bufPayload...)
		data, msgType, requestId, err := c.framer.Unframe(frame)
		decryptData := c.encryptor.Decrypt(data)
		if err != nil {
			fmt.Println("[tunnelReader] ERROR: unframe: " + err.Error())
			continue
		}

		conWrapper, ok := c.dispatcher.Find(requestId)
		requestCon := conWrapper.Conn
		if !ok {
			fmt.Printf("[tunnelReader] ERROR: not found connection [%x]\n", requestId)
			continue
		}

		switch msgType {
		case core.MsgData:
			fmt.Printf("[tunnelReader] send data [%s]...\n", decryptData[:10])
			requestCon.Write(decryptData)
			if !conWrapper.IsTunnel {
				fmt.Printf("[tunnelReader] not nunnel connection [%x], close\n", requestId)
				c.sendCloseIntoTunnel(requestId)
				requestCon.Close()
				c.dispatcher.Delete(requestId)
			}
		case core.MsgClose:
			requestCon.Close()
			c.dispatcher.Delete(requestId)
		case core.MsgError:
			requestCon.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
			requestCon.Close()
			c.dispatcher.Delete(requestId)
		}

	}

}

func (c *ClientTunnel) sendIntoTunnel(requestId core.RequestID, msgType core.MessageType, payload []byte) error {
	fmt.Printf("[sendIntoTunnel] Prepeare new msg: requestId: [%x], msgType: [%#v], msg: [%#v]\n", requestId, msgType, string(payload[:]))
	payloadEncode := c.encryptor.Encrypt([]byte(payload))
	frame, err := c.framer.Frame(payloadEncode, msgType, requestId)
	if err != nil {
		fmt.Printf("[sendIntoTunnel] Error in send data: [%s]\n", err.Error())
		return err
	}
	_, err = c.serverTonnelConn.Write(frame)
	fmt.Printf("[sendIntoTunnel] Data has been sent\n")
	return nil
}

func (c *ClientTunnel) sendCloseIntoTunnel(requestId core.RequestID) {
	fmt.Printf("[sendCloseIntoTunnel] close %x\n", requestId)
	c.sendIntoTunnel(requestId, core.MsgClose, []byte{})
}

func (c *ClientTunnel) generateAndRegistrateId(requestConn net.Conn, protoType core.ProtocolType, isTunnel bool) core.RequestID {
	requestId := core.GenerateID()
	c.dispatcher.Register(requestId, requestConn, protoType, isTunnel)
	fmt.Printf("Save conn with UUID %x\n", requestId)
	return requestId
}

func NewClientTunnel(
	transport core.Transport,
	encryptor core.Encryptor,
	framer core.Framer,
	dispatcher core.Dispatcher,
	detector core.Detector,
	cfg *config.Config) *ClientTunnel {
	return &ClientTunnel{
		transport:        transport,
		encryptor:        encryptor,
		framer:           framer,
		dispatcher:       dispatcher,
		detector:         detector,
		running:          false,
		serverTonnelConn: nil,
		remoteAddr:       cfg.RemoteHost + ":" + strconv.Itoa(cfg.RemotePort),
		localAddr:        cfg.LocalHost + ":" + strconv.Itoa(cfg.LocalPort),
	}
}
