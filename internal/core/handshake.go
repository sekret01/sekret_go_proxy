package core

import "net"

type HandshakeStep interface {
	Name() string
	ClientSide(ctx *HandshakeContext, conn net.Conn) error
	ServerSide(ctx *HandshakeContext, conn net.Conn) error
}

type Bootstrap interface {
	ClientBootstrap(con net.Conn) (steps []string, sessionKey []byte, err error)
	ServerBootstrap(con net.Conn) (steps []string, sessionKey []byte, err error)
}

type HandshakeContext struct {
	ClientIP     string
	EncryptorKey []byte
}
