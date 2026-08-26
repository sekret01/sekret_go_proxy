package core

// Интерфейс для паковки и распаковки данных в
// собственный протокол
type Framer interface {
	Frame()
	Unframe()
}
