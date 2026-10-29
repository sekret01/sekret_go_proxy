package core

import (
	"bytes"
	"encoding/binary"
	"io"
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

func (s *SecureConn) Write(b []byte) (int, error) {
	buf := new(bytes.Buffer)
	data := make([]byte, len(b))
	data = s.enc.Encrypt(b)
	payloadLen := uint16(len(data))
	binary.Write(buf, binary.BigEndian, payloadLen)
	buf.Write(data)
	if _, err := s.Conn.Write(buf.Bytes()); err != nil {
		return 0, err
	}
	return len(b), nil
}

func (s *SecureConn) Read(b []byte) (int, error) {
	sizeBuf := make([]byte, 2)
	if _, err := io.ReadFull(s.Conn, sizeBuf); err != nil {
		return -1, err
	}
	payloadSize := binary.BigEndian.Uint16(sizeBuf)
	dataBuf := make([]byte, payloadSize)
	if _, err := io.ReadFull(s.Conn, dataBuf); err != nil {
		return -1, err
	}
	decrBytes := s.enc.Decrypt(dataBuf)
	n := len(decrBytes)
	copy(b, decrBytes)
	return n, nil
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
