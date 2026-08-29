package tcp

import (
	"net"

	"github.com/sekret01/sekret_go_proxy/internal/core"
)

type TcpTransport struct{}

func (t *TcpTransport) Listen(address string) (net.Listener, error) {
	listener, err := net.Listen("tcp", address)
	return listener, err
}

func (t *TcpTransport) Dial(address string) (net.Conn, error) {
	connection, err := net.Dial("tcp", address)
	return connection, err
}

func newTcpTransport() (core.Transport, error) {
	tcpTrancport := TcpTransport{}
	return &tcpTrancport, nil
}
