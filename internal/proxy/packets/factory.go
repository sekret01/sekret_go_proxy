package packets

import (
	"sync"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
)

type FactoryFunc func(cfg *config.Config) (core.PacketDevice, error)

var (
	factoryMap map[string]FactoryFunc = map[string]FactoryFunc{}
	mu         sync.RWMutex
)

func NewPacketDevice(osType string, cfg *config.Config) (core.PacketDevice, error) {
	return newPacketDevice(cfg)
}

func List() []string {
	keys := []string{}
	for key := range factoryMap {
		keys = append(keys, key)
	}
	return keys
}
