//go:build linux

package packets

import (
	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/proxy/packets/linux"
)

func newPacketDevice(cfg *config.Config) (core.PacketDevice, error) {
	return linux.NewLinuxPacketDevice(cfg)
}
