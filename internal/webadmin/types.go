package webadmin

import (
	"github.com/sekret01/sekret_go_proxy/internal/proxy"
)

type HomeData struct {
	Status *proxy.TunnelStatus `json:"status"`
	Time   string              `json:"time"`
	Info   string              `json:"info"`
}
