package handshake

import (
	"fmt"
	"sync"

	"github.com/sekret01/sekret_go_proxy/internal/core"
)

type FactoryFunc func() (core.HandshakeStep, error)

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
	fmt.Printf("HandshakeStep [%s] has been registrated\n", key)
}

func get(key string) (FactoryFunc, error) {
	mu.RLock()
	defer mu.RUnlock()
	val, exist := framesMap[key]
	if !exist {
		return nil, core.ErrInvalidHandshakeStepKey
	}
	return val, nil
}

func NewHandshake(name string) (core.HandshakeStep, error) {
	factory, err := get(name)
	if err != nil {
		return nil, err
	}
	return factory()
}

func List() []string {
	keys := []string{}
	for key := range framesMap {
		keys = append(keys, key)
	}
	return keys
}
