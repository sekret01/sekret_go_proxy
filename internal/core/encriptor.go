package core

// Интерфейс для шифровщиков данных
type Encryptor interface {
	Encrypt()
	Decrypt()
}
