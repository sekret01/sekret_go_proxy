package utils

import (
	"crypto/rand"
)

func Generate32ByteKey() []byte {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic("CRITICAL ERROR generating 32 byte key: " + err.Error())
	}
	return key
}

func GetByteBuffer(size int) ([]byte, error) {
	if size < 0 {
		return make([]byte, 0), BufferSizeError
	}
	return make([]byte, size), nil
}
