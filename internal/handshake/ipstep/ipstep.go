package ipstep

import (
	"errors"
	"net"
	"strings"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/handshake"
	"github.com/sekret01/sekret_go_proxy/internal/proxy/server"
)

type IpStep struct {
	ipPool *server.IpPool // TODO вынести IpPool в другой пакет
}

func (s *IpStep) ClientSide(ctx *core.HandshakeContext, conn net.Conn) error {
	if _, err := conn.Write([]byte("vpn-ip")); err != nil {
		return err
	}
	buf := make([]byte, 16)
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
	buf := make([]byte, 16)
	n, err := conn.Read(buf)
	if err != nil {
		return err
	}
	if string(buf[:n]) != "vpn-ip" {
		return errors.New("Unexpected vpn-ip hello: " + string(buf[:n]))
	}
	clientIp, err := s.ipPool.Allocate()
	if err != nil {
		return err
	}

	ctx.ClientIP = clientIp
	_, err = conn.Write([]byte("vpn-ip-r" + clientIp))
	return err
}

func NewIpStep(cfg *config.Config) (core.HandshakeStep, error) {
	return &IpStep{
			ipPool: server.NewIPPool(cfg.VPN.Subnet),
		},
		nil
}

func init() {
	handshake.Registrate("ip", NewIpStep)
}
