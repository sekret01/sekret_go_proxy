package core

import (
	"net"
)

type Auth interface {
	ServerHandshake(conn net.Conn) ([]byte, error)
	ClientHandshake(conn net.Conn) ([]byte, error)
}
