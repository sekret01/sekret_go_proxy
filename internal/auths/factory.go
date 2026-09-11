package auths

import (
	"fmt"
	"sync"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
)

type factoryFunc func(config *config.Config) (core.Auth, error)

var (
	authMap = make(map[string]factoryFunc)
	mu      sync.RWMutex
)

func Reigstrate(key string, factory factoryFunc) {
	mu.Lock()
	defer mu.Unlock()
	if _, exist := authMap[key]; exist {
		panic("Auth module [" + key + "] already exists")
	}
	authMap[key] = factory
	fmt.Printf("Auth [%s] has been registrated\n", key)
}

func get(key string) (factoryFunc, error) {
	mu.RLock()
	encr, ex := authMap[key]
	mu.RUnlock()
	if !ex {
		return nil, core.ErrInvalidEncryptKey
	}
	return encr, nil
}

func NewEncryptor(key string, config *config.Config) (core.Auth, error) {
	factory, err := get(key)
	if err != nil {
		return nil, err
	}
	return factory(config)
}

func List() []string {
	keys := []string{}
	for key := range authMap {
		keys = append(keys, key)
	}
	return keys
}
