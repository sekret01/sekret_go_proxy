package detectors

import (
	"bytes"

	"github.com/sekret01/sekret_go_proxy/internal/core"
)

type Detector struct{}

func (d *Detector) Detect(data []byte) (core.ProtocolType, error) {
	if data[0] == 0x05 {
		return core.ProtoSOCKS5, nil
	} else if bytes.HasPrefix(data, []byte("CONNECT")) ||
		bytes.HasPrefix(data, []byte("GET")) ||
		bytes.HasPrefix(data, []byte("POST")) ||
		bytes.HasPrefix(data, []byte("PUT")) ||
		bytes.HasPrefix(data, []byte("DELETE")) ||
		bytes.HasPrefix(data, []byte("PATCH")) ||
		bytes.HasPrefix(data, []byte("HEAD")) ||
		bytes.HasPrefix(data, []byte("OPTIONS")) ||
		bytes.HasPrefix(data, []byte("TRACE")) {
		return core.ProtoHTTP, nil
	}

	return core.ProtoUnknown, nil
}

func NewDetector() core.Detector {
	return &Detector{}
}
