package ipstep

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/handshake"
)

type IpStep struct{}

func (s *IpStep) ClientSide(ctx *core.HandshakeContext, conn net.Conn) error {
	fmt.Println("write vpn-ip")
	if _, err := conn.Write([]byte("vpn-ip")); err != nil {
		return err
	}
	fmt.Println("read getted ip")
	buf := make([]byte, 25)
	n, err := conn.Read(buf)
	if err != nil {
		return err
	}
	res := string(buf[:n])
	fmt.Printf("read %d bytes\n", n)
	fmt.Println("res ip: " + res)
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

func NewIpStep(cfg *config.Config) (core.HandshakeStep, error) {
	return &IpStep{},
		nil
}

func init() {
	handshake.Registrate("ip", NewIpStep)
}
