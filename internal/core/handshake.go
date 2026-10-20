package core

import "net"

type HandshakeStep interface {
	ClientSide(ctx *HandshakeContext, conn net.Conn)
	ServerSide(ctx *HandshakeContext, conn net.Conn)
}

type HandshakeContext struct {
	ClientIP string
}
