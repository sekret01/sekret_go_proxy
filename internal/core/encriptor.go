package core

// Интерфейс для шифровщиков данных
type Encryptor interface {
	Encrypt(data []byte) []byte
	Decrypt(data []byte) []byte
}
