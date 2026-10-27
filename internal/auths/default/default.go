package defaultauth

import (
	"net"

	"github.com/sekret01/sekret_go_proxy/internal/auths"
	"github.com/sekret01/sekret_go_proxy/internal/core"
)

type DefaultAuth struct{}

func (a *DefaultAuth) ServerHandshake(conn net.Conn) (*core.HandshakeContext, error) {
	return &core.HandshakeContext{}, nil
}

func (a *DefaultAuth) ClientHandshake(conn net.Conn) (*core.HandshakeContext, error) {
	return &core.HandshakeContext{}, nil
}
func (a *DefaultAuth) SetEncryptor(enc core.Encryptor) {}
func (a *DefaultAuth) SetSteps(steps []string) error   { return nil }

func NewDefaultAuthAuth(
	ctxGenerator core.HandshakeContextGenerator,
) (core.Auth, error) {
	return &DefaultAuth{}, nil
}

func init() {
	auths.Reigstrate("default", NewDefaultAuthAuth)
}
