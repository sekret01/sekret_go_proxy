package core

import "errors"

// Кастомные ошибки
var (
	ErrInvalidMagic = errors.New("Invalid magic bytes")

	ErrInvalidEncryptKey    = errors.New("Encryptor with key not found")
	ErrInvalidTransportKey  = errors.New("Transport with key not found")
	ErrInvalidFramerKey     = errors.New("Framer with key not found")
	ErrInvalidDispatcherKey = errors.New("Dispatcher with key not found")
	ErrInvalidAuthKey       = errors.New("Auth module with key not found")

	ErrBuildClientApp = errors.New("Error building ClientApp modules")
	ErrBuildServerApp = errors.New("Error building ServerApp modules")

	ErrInsufficientHeaderLensth  = errors.New("Insufficient length to read the header")
	ErrInsufficientPayloadLensth = errors.New("Insufficient length to read payload")

	ErrAythFailed = errors.New("Authentication failed")
)
