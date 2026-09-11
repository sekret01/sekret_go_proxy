package dispatchers

import (
	"fmt"
	"sync"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
)

type factoryFunc func(config *config.Config) (core.Dispatcher, error)

var (
	dispatchersMap = make(map[string]factoryFunc)
	mu             sync.RWMutex
)

func Register(key string, factory factoryFunc) {
	mu.Lock()
	defer mu.Unlock()
	if _, exist := dispatchersMap[key]; exist {
		panic("Encryption [" + key + "] already exists")
	}
	dispatchersMap[key] = factory
	fmt.Printf("Dispatcher [%s] has been registrated\n", key)
}

func get(key string) (factoryFunc, error) {
	mu.RLock()
	encr, ex := dispatchersMap[key]
	mu.RUnlock()
	if !ex {
		return nil, core.ErrInvalidEncryptKey
	}
	return encr, nil
}

func NewDispatcher(config *config.Config) (core.Dispatcher, error) {
	factory, err := get(config.DispatcherType)
	if err != nil {
		return nil, err
	}
	return factory(config)
}

func List() []string {
	keys := []string{}
	for key := range dispatchersMap {
		keys = append(keys, key)
	}
	return keys
}
