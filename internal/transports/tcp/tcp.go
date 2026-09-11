package tcp

import (
	"net"

	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/transports"
)

const (
	tName = "tcp"
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

func init() {
	transports.Registrate(tName, newTcpTransport)
}
