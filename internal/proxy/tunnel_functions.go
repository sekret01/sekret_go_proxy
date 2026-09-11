package proxy

import (
	"io"
	"net"

	"github.com/sekret01/sekret_go_proxy/internal/core"
)

func ReadFrameFromConnection(serverTonnelConn net.Conn, framer core.Framer) ([]byte, error) {
	bufHeader := make([]byte, framer.HeaderSize())
	if _, err := io.ReadFull(serverTonnelConn, bufHeader); err != nil {
		return nil, err
	}
	payloadSize, err := framer.GetPayloadSize(bufHeader)
	if err != nil {
		return nil, err
	}
	bufPayload := make([]byte, payloadSize)
	if _, err := io.ReadFull(serverTonnelConn, bufPayload); err != nil {
		return nil, err
	}
	frame := append(bufHeader, bufPayload...)
	return frame, nil
}

func closeConnection(dispatcher core.Dispatcher, requestId core.RequestID) {
	conWrapper, ok := dispatcher.Find(requestId)
	con := conWrapper.Conn
	if !ok {
		return
	}
	con.Close()
	dispatcher.Delete(requestId)
}
