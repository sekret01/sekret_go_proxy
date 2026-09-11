package core

import (
	"crypto/rand"
)

func GenerateID() RequestID {
	id := RequestID{}
	if _, err := rand.Read(id[:]); err != nil {
		panic("CRITICAL ERROR generating UUID: " + err.Error())
	}
	return id
}
