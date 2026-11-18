package core

type Encryptor interface {
	Encrypt(data []byte) []byte
	Decrypt(data []byte) []byte

	EncryptTo(dst, data []byte) []byte
	DecryptTo(dst, data []byte) []byte
}
