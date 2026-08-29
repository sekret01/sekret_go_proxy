package core

import "errors"

// Кастомные ошибки
var (
	ErrInvalidMagic        = errors.New("Invalid magic bytes")
	ErrInvalidEncryptKey   = errors.New("Encryptor with key not found")
	ErrInvalidTransportKey = errors.New("Transport with key not found")
)
