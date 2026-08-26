package core

// Интерфейс для реализации транспорта (TCP, UDP, ...)
type Transport interface {
	Dial()
	Listen()
}
