package mock

import (
	"net"

	"github.com/sekret01/sekret_go_proxy/internal/auths"
	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
)

type MockAuth struct{}

func (a *MockAuth) ServerHandshake(conn net.Conn) ([]byte, error) {
	return make([]byte, 0), nil
}

func (a *MockAuth) ClientHandshake(conn net.Conn) ([]byte, error) {
	return make([]byte, 0), nil
}

func NewMockAuth(cfg *config.Config) (core.Auth, error) {
	return &MockAuth{}, nil
}

func init() {
	auths.Reigstrate("mock", NewMockAuth)
}
