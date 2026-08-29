package core

// Интерфейс для детекторов протоколов
type Detector interface {
	Detect(data []byte) (ProtocolType, error)
}
