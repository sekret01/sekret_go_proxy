package core

type Detector interface {
	Detect(data []byte) (ProtocolType, error)
}
