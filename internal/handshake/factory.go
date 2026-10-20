package handshake

import (
	"sync"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
)

type FactoryFunc func(*config.Config) (core.HandshakeStep, error)

var (
	framesMap = make(map[string]FactoryFunc)
	mu        sync.RWMutex
)

func Registrate(key string, factory FactoryFunc) {
	mu.Lock()
	defer mu.Unlock()
	if _, exist := framesMap[key]; exist {
		panic("Framer [" + key + "] already exists")
	}
	framesMap[key] = factory
}

func get(key string) (FactoryFunc, error) {
	mu.RLock()
	defer mu.RUnlock()
	val, exist := framesMap[key]
	if !exist {
		return nil, core.ErrInvalidTransportKey
	}
	return val, nil
}

func NewHandshake(cfg *config.Config, name string) (core.HandshakeStep, error) {
	factory, err := get(name)
	if err != nil {
		return nil, err
	}
	return factory(cfg)
}

func List() []string {
	keys := []string{}
	for key := range framesMap {
		keys = append(keys, key)
	}
	return keys
}
