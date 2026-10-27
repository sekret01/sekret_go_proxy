package defaultencryptorgo

import (
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/encryptors"
)

// Заглушка, для быстрой проверки логики работы системы
type DefaultEncryptor struct{}

func (m *DefaultEncryptor) Encrypt(data []byte) []byte {
	return data
}

func (m *DefaultEncryptor) Decrypt(data []byte) []byte {
	return data
}

func NewDefaultEncryptor(key []byte) (core.Encryptor, error) {
	return &DefaultEncryptor{}, nil
}

func init() {
	encryptors.Reigstrate("default", NewDefaultEncryptor)
}
