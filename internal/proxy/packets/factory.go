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

func Registrate(key string, factoryFunc FactoryFunc) {
	mu.RLock()
	defer mu.RUnlock()
	if _, ex := factoryMap[key]; ex {
		panic("PacketDevice [" + key + "] already exists")
	}
	factoryMap[key] = factoryFunc
}

func get(key string) (FactoryFunc, error) {
	mu.RLock()
	defer mu.RUnlock()
	val, exist := factoryMap[key]
	if !exist {
		return nil, core.ErrInvalidTransportKey
	}
	return val, nil
}

func NewPacketDevice(osType string, cfg *config.Config) (core.PacketDevice, error) {
	factory, err := get(osType)
	if err != nil {
		return nil, err
	}
	return factory(cfg)
}

func List() []string {
	keys := []string{}
	for key := range factoryMap {
		keys = append(keys, key)
	}
	return keys
}
