package proxy

import (
	"io"
	"net"

	// "strconv"

	"github.com/sekret01/sekret_go_proxy/internal/core"
	// "github.com/sekret01/sekret_go_proxy/pkg/logger"
)

func ReadFrameFromConnection(serverTonnelConn net.Conn, framer core.Framer) ([]byte, error) { // logger logger.Logger,
	bufHeader := make([]byte, framer.HeaderSize())

	if _, err := io.ReadFull(serverTonnelConn, bufHeader); err != nil {
		// logger.Critical("[tunnelReader] CRITIACL ERROR: error in read tunnel (header): " + err.Error())
		return nil, err
	}
	// logger.Debug("[tunnelReader] get header: " + string(bufHeader))

	payloadSize, err := framer.GetPayloadSize(bufHeader)
	if err != nil {
		// logger.Debug("[tunnelReader] ERROR: cannot get payload size (header): " + err.Error())
		return nil, err
	}
	// logger.Debug("[tunnelReader] wait payload size: " + strconv.Itoa(payloadSize))

	bufPayload := make([]byte, payloadSize) // TODO изменять в конфигах bufSize
	if _, err := io.ReadFull(serverTonnelConn, bufPayload); err != nil {
		// logger.Debug("[tunnelReader] ERROR: cannot read payload (payload): " + err.Error())
		return nil, err
	}

	frame := append(bufHeader, bufPayload...)
	return frame, nil
}
