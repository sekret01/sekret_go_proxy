package transports

import (
	"fmt"
	"sync"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
)

type FactoryFunc func() (core.Transport, error)

var (
	transportsMap = make(map[string]FactoryFunc)
	mu            sync.RWMutex
)

func Registrate(key string, factory FactoryFunc) {
	mu.Lock()
	defer mu.Unlock()
	if _, exist := transportsMap[key]; exist {
		panic("Transport [" + key + "] already exists")
	}
	transportsMap[key] = factory
	fmt.Printf("Transport [%s] has been registrated", key)
}

func get(key string) (FactoryFunc, error) {
	mu.RLock()
	defer mu.RUnlock()
	val, exist := transportsMap[key]
	if !exist {
		return nil, core.ErrInvalidTransportKey
	}
	return val, nil
}

func NewTransport(cfg *config.Config) (core.Transport, error) {
	factory, err := get(cfg.TransportType)
	if err != nil {
		return nil, err
	}
	return factory()
}
