package filters

import (
	"sync"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
)

type FactoryFunc func(active bool, domain string, addresses []string) (core.Filter, error)

var (
	filterMap = make(map[string]FactoryFunc)
	mu        sync.RWMutex
)

func Registrate(key string, factory FactoryFunc) {
	mu.Lock()
	defer mu.Unlock()
	if _, exist := filterMap[key]; exist {
		panic("Filter [" + key + "] already exists")
	}
	filterMap[key] = factory
}

func get(key string) (FactoryFunc, error) {
	mu.RLock()
	defer mu.RUnlock()
	val, exist := filterMap[key]
	if !exist {
		return nil, core.ErrInvalidFilterKey
	}
	return val, nil
}

func NewFilter(cfg *config.Config) (core.Filter, error) {
	factory, err := get(cfg.FilterType)
	if err != nil {
		return nil, err
	}
	return factory(cfg.FilterActive, cfg.FilterDomain, cfg.FilterAddresses)
}

func List() []string {
	keys := []string{}
	for key := range filterMap {
		keys = append(keys, key)
	}
	return keys
}
