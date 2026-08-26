package core

import "errors"

// Кастомные ошибки
var (
	ErrInvalidMagic      = errors.New("Invalid magic bytes")
	ErrInvalidEncryptKey = errors.New("Encryptor with key not found")
)
