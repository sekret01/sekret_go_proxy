package bootstrap

import (
	"fmt"
	"sync"

	"github.com/sekret01/sekret_go_proxy/internal/core"
)

type BootstrapConfig struct {
	HandshakeSteps []string
}

type FactoryFunc func(*BootstrapConfig) (core.Bootstrap, error)

var (
	bootsrapMap = make(map[string]FactoryFunc)
	mu          sync.RWMutex
)

func Registrate(key string, factory FactoryFunc) {
	mu.Lock()
	defer mu.Unlock()
	if _, exist := bootsrapMap[key]; exist {
		panic("Framer [" + key + "] already exists")
	}
	bootsrapMap[key] = factory
	fmt.Printf("HandshakeStep [%s] has been registrated\n", key)
}

func get(key string) (FactoryFunc, error) {
	mu.RLock()
	defer mu.RUnlock()
	val, exist := bootsrapMap[key]
	if !exist {
		return nil, core.ErrInvalidTransportKey
	}
	return val, nil
}

func NewBootstrap(name string, bootCfg *BootstrapConfig) (core.Bootstrap, error) {
	factory, err := get(name)
	if err != nil {
		return nil, err
	}
	return factory(bootCfg)
}

func List() []string {
	keys := []string{}
	for key := range bootsrapMap {
		keys = append(keys, key)
	}
	return keys
}
