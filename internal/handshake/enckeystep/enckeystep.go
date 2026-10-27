package enckeystep

import (
	"errors"
	"net"
	"strings"

	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/handshake"
	"github.com/sekret01/sekret_go_proxy/internal/utils"
)

type EncKeyStep struct{}

const (
	name = "enckey"
)

func (s *EncKeyStep) ClientSide(ctx *core.HandshakeContext, conn net.Conn) error {
	if _, err := conn.Write([]byte("vpn-enckey")); err != nil {
		return err
	}
	buf := make([]byte, 100)
	n, err := conn.Read(buf)
	if err != nil {
		return err
	}
	res := string(buf[:n])
	tokens := strings.SplitN(res, ":", 2)
	if len(tokens) < 2 || tokens[0] != "vpn-enckey-r" {
		return errors.New("Unexpected vpn-enckey recieve: " + res)
	}
	ctx.EncryptorKey = []byte(tokens[1])
	return nil
}

func (s *EncKeyStep) ServerSide(ctx *core.HandshakeContext, conn net.Conn) error {
	buf := make([]byte, 100)
	n, err := conn.Read(buf)
	if err != nil {
		return err
	}
	if string(buf[:n]) != "vpn-enckey" {
		return errors.New("Unexpected vpn-enckey request: " + utils.BytesToString(buf[:n], 50))
	}
	_, err = conn.Write([]byte("vpn-enckey-r:" + string(ctx.EncryptorKey)))
	return err
}

func (s *EncKeyStep) Name() string {
	return name
}

func NewEncKeyStep(cfg *handshake.HandshakeStepConfig) (core.HandshakeStep, error) {
	return &EncKeyStep{}, nil
}

func init() {
	handshake.Registrate(name, NewEncKeyStep)
}
