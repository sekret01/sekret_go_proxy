package core

type Filter interface {
	Check(string) bool
}
