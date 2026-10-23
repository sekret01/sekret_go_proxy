//go:build windows

package packets

import (
	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
)

func newPacketDevice(osType string, cfg *config.Config) (core.PacketDevice, error) {
	return nil, core.ErrOsTypeNotSupported
}
