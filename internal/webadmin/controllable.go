package webadmin

import (
	"github.com/sekret01/sekret_go_proxy/internal/proxy"
)

type Controllable interface {
	Start() error
	Stop() error
	Status() *proxy.TunnelStatus
	GetInfo() string
	Users() []string
}
