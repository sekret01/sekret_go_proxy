package proxy

import (
	"io"
	"net"

	"github.com/sekret01/sekret_go_proxy/internal/buffers"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type stringStatus string

var channels int = 0

const (
	STOPPED stringStatus = "STOPPED"
	LAUNCH  stringStatus = "LAUNCH"
	RUNNING stringStatus = "RUNNING"
)

type TunnelChannel struct {
	Id      int
	Channel chan []byte
	isOpen  bool
}

func (t *TunnelChannel) IsOpen() bool {
	return t.isOpen
}

func (t *TunnelChannel) Close() {
	close(t.Channel)
	t.isOpen = false
}

func NewTunnelChannel() *TunnelChannel {
	ch := &TunnelChannel{
		Id:      channels,
		Channel: make(chan []byte, 100),
		isOpen:  true,
	}
	channels++
	return ch
}

type TunnelStatus struct {
	isRunning  bool
	statusName stringStatus
	logger     logger.Logger
}

func (t *TunnelStatus) IsRunning() bool {
	return t.isRunning
}

func (t *TunnelStatus) StatusName() stringStatus {
	return t.statusName
}

func (t *TunnelStatus) SetLaunch() {
	t.logger.Debug("set LAUNCH")
	t.statusName = LAUNCH
	t.isRunning = true
}

func (t *TunnelStatus) SetRunning() {
	t.logger.Debug("set RUNNING")
	t.statusName = RUNNING
	t.isRunning = true
}

func (t *TunnelStatus) SetStopped() {
	t.logger.Debug("set STOPPED")
	t.statusName = STOPPED
	t.isRunning = false
}

func NewTunnelStatus() *TunnelStatus {
	return &TunnelStatus{
		isRunning:  false,
		statusName: STOPPED,
		logger:     logger.GetLoggerHub().WithModule("TunnelStatus"),
	}
}

func ReadFrameFromConnection(serverTonnelConn net.Conn, framer core.Framer) (core.Buffer, []byte, error) {
	headerSize := framer.HeaderSize()
	var header [64]byte
	if _, err := io.ReadFull(serverTonnelConn, header[:headerSize]); err != nil {
		return nil, nil, err
	}
	payloadSize, err := framer.GetPayloadSize(header[:headerSize]) // bufHeader.Bytes()
	if err != nil {
		return nil, nil, err
	}
	if payloadSize < 0 || payloadSize > core.MAX_PAYLOAD_SIZE {
		return nil, nil, core.ErrPayloadSizeTooLarge
	}

	total := headerSize + payloadSize
	bufPayload := buffers.GetBuf(total)
	copy(bufPayload.Bytes()[:headerSize], header[:headerSize])
	if _, err := io.ReadFull(serverTonnelConn, bufPayload.Bytes()[headerSize:total]); err != nil {
		return nil, nil, err
	}
	return bufPayload, bufPayload.Bytes()[:total], nil
}

func CloseConnection(dispatcher core.Dispatcher, requestId core.RequestID) {
	conWrapper, ok := dispatcher.Find(requestId)
	if !ok {
		return
	}
	conWrapper.Conn.Close()
	dispatcher.Delete(requestId)
}
