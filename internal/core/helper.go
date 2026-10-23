package core

import (
	"net"
	"time"
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
	con net.Conn
	enc Encryptor
	// key []byte
}

func (s *SecureConn) Write(b []byte) (n int, err error) {
	data := s.enc.Encrypt(b)
	return s.con.Write(data)
}

func (s *SecureConn) Read(b []byte) (n int, err error) {
	buf := []byte{}
	n, err = s.con.Read(buf)
	if err != nil {
		return n, err
	}
	b = s.enc.Decrypt(buf[:n])
	return n, nil
}

func (s *SecureConn) Close() error                       { return s.con.Close() }
func (s *SecureConn) LocalAddr() net.Addr                { return s.con.LocalAddr() }
func (s *SecureConn) RemoteAddr() net.Addr               { return s.con.RemoteAddr() }
func (s *SecureConn) SetDeadline(t time.Time) error      { return s.con.SetDeadline(t) }
func (s *SecureConn) SetReadDeadline(t time.Time) error  { return s.con.SetReadDeadline(t) }
func (s *SecureConn) SetWriteDeadline(t time.Time) error { return s.con.SetWriteDeadline(t) }

func NewSecureConn(
	con net.Conn,
	enc Encryptor,
) *SecureConn {
	return &SecureConn{
		con: con,
		enc: enc,
	}
}
