package core

import (
	"net"
)

// Интерфейс создателя HandshakeContext
type HandshakeContextGenerator interface {
	NewHandshakeContext() *HandshakeContext
}

// Интерфейс слушателя генератора HandshakeContext
type HandshakeGeneratorSubscriber interface {
	SubscribeGenerator(*HandshakeContextGenerator)
}

// Прослойка-реализация генератора и слушателя генератора
type HandshakeGeneratorConnector struct {
	generator HandshakeContextGenerator
}

func (c *HandshakeGeneratorConnector) SubscribeGenerator(generator HandshakeContextGenerator) {
	c.generator = generator
}

func (c *HandshakeGeneratorConnector) NewHandshakeContext() *HandshakeContext {
	return c.generator.NewHandshakeContext()
}

// Обертка net.Conn, шифрующая трафик
type SecureConn struct {
	net.Conn
	enc Encryptor
}

func (s *SecureConn) Write(b []byte) (n int, err error) {
	data := make([]byte, len(b))
	data = s.enc.Encrypt(b)
	if _, err := s.Conn.Write(data); err != nil {
		return 0, err
	}
	return len(b), nil
}

func (s *SecureConn) Read(b []byte) (n int, err error) {
	n, err = s.Conn.Read(b)
	if n > 0 {
		copy(b, s.enc.Decrypt(b))
	}
	return n, err
}

func NewSecureConn(
	con net.Conn,
	enc Encryptor,
) net.Conn {
	if enc == nil {
		return con
	}
	return &SecureConn{
		Conn: con,
		enc:  enc,
	}
}
