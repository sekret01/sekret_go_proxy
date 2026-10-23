package packets

import (
	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
)

func NewPacketDevice(osType string, cfg *config.Config) (core.PacketDevice, error) {
	return newPacketDevice(cfg)
}
