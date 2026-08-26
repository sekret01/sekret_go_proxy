package mockencryptorgo

import (
	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/encryptors"
)

// Заглушка, для быстрой проверки логики работы системы
type MockEncryptor struct{}

func (m *MockEncryptor) Encrypt() {}

func (m *MockEncryptor) Decrypt() {}

func NewMockCrypto(config config.Config) (core.Encryptor, error) {
	return &MockEncryptor{}, nil
}

func init() {
	encryptors.Reigstrate("mock", NewMockCrypto)
}
