package proxy

import (
	"fmt"
	"net"

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
func (c *ClientTunnel) newRequestConnectionHandler(requestConn net.Conn) error {
	return nil
}

// Чтение, обработка и перессылка данных с сервера
func (c *ClientTunnel) tunnelReader() {

}

func NewClientTunnel(
	transport core.Transport,
	encryptor core.Encryptor,
	framer core.Framer,
	dispatcher core.Dispatcher,
	detector core.Detector) *ClientTunnel {
	return &ClientTunnel{
		transport:        transport,
		encryptor:        encryptor,
		framer:           framer,
		dispatcher:       dispatcher,
		detector:         detector,
		running:          false,
		serverTonnelConn: nil,
	}
}
