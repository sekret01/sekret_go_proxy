package proxy

import (
	"io"
	"net"

	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type stringStatus string

const (
	STOPPED stringStatus = "STOPPED"
	LAUNCH  stringStatus = "LAUNCH"
	RUNNING stringStatus = "RUNNING"
)

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
