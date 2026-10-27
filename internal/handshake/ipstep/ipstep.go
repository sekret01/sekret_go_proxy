package ipstep

import (
	"errors"
	"net"
	"strings"

	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/handshake"
)

type IpStep struct{}

func (s *IpStep) ClientSide(ctx *core.HandshakeContext, conn net.Conn) error {
	if _, err := conn.Write([]byte("vpn-ip")); err != nil {
		return err
	}
	buf := make([]byte, 25)
	n, err := conn.Read(buf)
	if err != nil {
		return err
	}
	res := string(buf[:n])
	tokens := strings.Split(res, ":")
	if len(tokens) < 2 || tokens[0] != "vpn-ip-r" {
		return errors.New("Unexpected vpn-ip recieve: " + res)
	}
	ctx.ClientIP = tokens[1]
	return nil
}

func (s *IpStep) ServerSide(ctx *core.HandshakeContext, conn net.Conn) error {
	buf := make([]byte, 25)
	n, err := conn.Read(buf)
	if err != nil {
		return err
	}
	if string(buf[:n]) != "vpn-ip" {
		return errors.New("Unexpected vpn-ip hello: " + string(buf[:n]))
	}
	_, err = conn.Write([]byte("vpn-ip-r:" + ctx.ClientIP))
	return err
}

func (s *IpStep) Name() string {
	return "ip"
}

func NewIpStep(cfg *handshake.HandshakeStepConfig) (core.HandshakeStep, error) {
	return &IpStep{},
		nil
}

func init() {
	handshake.Registrate("ip", NewIpStep)
}
