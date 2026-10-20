package core

import "net"

type HandshakeStep interface {
	Name() string
	ClientSide(ctx *HandshakeContext, conn net.Conn) error
	ServerSide(ctx *HandshakeContext, conn net.Conn) error
}

type HandshakeContext struct {
	ClientIP string
}
