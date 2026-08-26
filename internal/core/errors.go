package core

import "errors"

// Кастомные ошибки
var (
	ErrInvalidMagic = errors.New("Invalid magic bytes")
)
