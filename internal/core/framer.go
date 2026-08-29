package core

// Интерфейс для паковки и распаковки данных в
// собственный протокол
type Framer interface {
	Frame(data []byte, messageType MessageType, requestId RequestID) ([]byte, error)
	Unframe(data []byte) ([]byte, MessageType, RequestID, error)
}
