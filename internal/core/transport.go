package core

import (
	"net"
)

// Интерфейс для реализации транспорта (TCP, UDP, ...)
type Transport interface {
	Listen(addres string) (net.Listener, error)
	Dial(address string) (net.Conn, error)
}
