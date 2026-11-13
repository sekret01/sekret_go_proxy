package core

import (
	"net"
)

type Transport interface {
	Listen(addres string) (net.Listener, error)
	Dial(address string) (net.Conn, error)
}
