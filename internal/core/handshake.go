package core

import "net"

type HandshakeStep interface {
	ClientSide(ctx *HandshakeContext, conn net.Conn) error
	ServerSide(ctx *HandshakeContext, conn net.Conn) error
}

type HandshakeContext struct {
	ClientIP string
}
