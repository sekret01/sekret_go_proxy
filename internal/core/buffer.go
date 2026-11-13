package core

type Buffer interface {
	Bytes() []byte
	Free()
	Len() int
	Cap() int
}
