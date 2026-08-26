package mockencryptorgo

import (
	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/encryptors"
)

// Заглушка, для быстрой проверки логики работы системы
type MockEncryptor struct{}

func (m *MockEncryptor) Encrypt(data []byte) []byte {
	return data
}

func (m *MockEncryptor) Decrypt(data []byte) []byte {
	return data
}

func NewMockCrypto(config config.Config) (core.Encryptor, error) {
	return &MockEncryptor{}, nil
}

func init() {
	encryptors.Reigstrate("mock", NewMockCrypto)
}
