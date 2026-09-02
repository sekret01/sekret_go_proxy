package chacha20

import (
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"

	"golang.org/x/crypto/chacha20poly1305"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/encryptors"
)

// ChaCha20Encryptor реализует core.Encryptor
type ChaCha20Encryptor struct {
	aead cipher.AEAD // Authenticated Encryption with Associated Data
}

// New создает новый шифратор с ключом (32 байта)
func NewChaCha20(cfg *config.Config) (core.Encryptor, error) {
	key := cfg.Key32Bytes
	if len(key) != chacha20poly1305.KeySize {
		return nil, errors.New("key must be 32 bytes for ChaCha20-Poly1305")
	}

	aead, err := chacha20poly1305.New([]byte(key))
	if err != nil {
		return nil, err
	}

	return &ChaCha20Encryptor{aead: aead}, nil
}

// Encrypt шифрует данные
// Формат: [Nonce (12 байт)] + [Зашифрованные данные + Tag (16 байт)]
func (c *ChaCha20Encryptor) Encrypt(plaintext []byte) []byte {
	nonce := make([]byte, c.aead.NonceSize()) // 12 байт
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil
	}

	// Seal зашифрует данные и добавит Tag (аутентификацию)
	ciphertext := c.aead.Seal(nonce, nonce, plaintext, nil)
	return ciphertext
}

// Decrypt расшифровывает данные
func (c *ChaCha20Encryptor) Decrypt(ciphertext []byte) []byte {
	nonceSize := c.aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil // , errors.New("ciphertext too short")
	}

	nonce, encrypted := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := c.aead.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil //, err // расшифровка не удалась (неправильный ключ или данные повреждены)
	}
	return plaintext //, nil
}

// Регистрация в фабрике
func init() {
	encryptors.Reigstrate("chacha20", NewChaCha20)
}
