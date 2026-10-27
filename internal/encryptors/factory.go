package encryptors

import (
	"fmt"
	"sync"

	"github.com/sekret01/sekret_go_proxy/internal/core"
)

// Функция-фабрика для создания Encryptor
type factoryFunc func(key []byte) (core.Encryptor, error)

var (
	encryptionsMap = make(map[string]factoryFunc)
	mu             sync.RWMutex
)

// Регистрация encryption-factory по ключу в общий список
func Reigstrate(key string, factory factoryFunc) {
	mu.Lock()
	defer mu.Unlock()
	if _, exist := encryptionsMap[key]; exist {
		panic("Encryption [" + key + "] already exists")
	}
	encryptionsMap[key] = factory
	fmt.Printf("Encryptor [%s] has been registrated\n", key)
}

// Попытка получить encryptor-factory по ключу.
func get(key string) (factoryFunc, error) {
	mu.RLock()
	encr, ex := encryptionsMap[key]
	mu.RUnlock()
	if !ex {
		return nil, core.ErrInvalidEncryptKey
	}
	return encr, nil
}

// Создание нового Encryption, выбор в конфигах
func NewEncryptor(name string, key []byte) (core.Encryptor, error) {
	factory, err := get(name)
	if err != nil {
		return nil, err
	}
	return factory(key)
}

func List() []string {
	keys := []string{}
	for key := range encryptionsMap {
		keys = append(keys, key)
	}
	return keys
}
