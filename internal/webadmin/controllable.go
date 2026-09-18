package webadmin

import (
	"github.com/sekret01/sekret_go_proxy/internal/proxy"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type Controllable interface {
	Start() error
	Stop() error
	Status() *proxy.TunnelStatus
	GetInfo() string
	Users() []string
	GetLogs() []logger.LogMessage
}
