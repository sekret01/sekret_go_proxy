package core

import (
	"net"
)

type Auth interface {
	ServerHandshake(conn net.Conn) (*HandshakeContext, error)
	ClientHandshake(conn net.Conn) (*HandshakeContext, error)
}
