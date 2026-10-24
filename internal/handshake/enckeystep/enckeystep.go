package enckeystep

import (
	"errors"
	"net"
	"strings"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/handshake"
)

type EncKeyStep struct{}

const (
	name = "enckey"
)

func (s *EncKeyStep) ClientSide(ctx *core.HandshakeContext, conn net.Conn) error {
	if _, err := conn.Write([]byte("vpn-enckey")); err != nil {
		return err
	}
	buf := make([]byte, 45)
	n, err := conn.Read(buf)
	if err != nil {
		return err
	}
	res := string(buf[:n])
	tokens := strings.Split(res, ":")
	if len(tokens) < 2 || tokens[0] != "vpn-enckey-r" {
		return errors.New("Unexpected vpn-enckey recieve: " + res)
	}
	ctx.EncryptorKey = []byte(tokens[1])
	return nil
}

func (s *EncKeyStep) ServerSide(ctx *core.HandshakeContext, conn net.Conn) error {
	buf := make([]byte, 25)
	n, err := conn.Read(buf)
	if err != nil {
		return err
	}
	if string(buf[:n]) != "vpn-enckey" {
		return errors.New("Unexpected vpn-enckey request: " + string(buf[:n]))
	}
	_, err = conn.Write([]byte("vpn-enckey-r:" + string(ctx.EncryptorKey)))
	return err
}

func (s *EncKeyStep) Name() string {
	return name
}

func NewEncKeyStep(cfg *config.Config) (core.HandshakeStep, error) {
	return &EncKeyStep{}, nil
}

func init() {
	handshake.Registrate(name, NewEncKeyStep)
}
