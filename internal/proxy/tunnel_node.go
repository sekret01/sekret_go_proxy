package proxy

import (
	"errors"
	"net"
	"sync"

	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type TunnelNode struct {
	tunnelReadChan   *SafeBufferChannel
	tunnelWriteChan  *SafeBufferChannel
	framer           core.Framer
	connMut          sync.Mutex
	remoteTunnelNode net.Conn
	isRunning        bool
	logger           logger.Logger
	done             chan struct{}
}

func (t *TunnelNode) Stop() error {
	t.logger.Debug("[Stop] Close tunnel")
	select {
	case <-t.done:
	default:
		close(t.done)
	}
	t.connMut.Lock()
	t.isRunning = false
	if t.remoteTunnelNode != nil {
		t.remoteTunnelNode.Close()
		t.remoteTunnelNode = nil
	}
	t.connMut.Unlock()
	t.tunnelReadChan.Close()
	t.tunnelWriteChan.Close()
	return nil
}

func (t *TunnelNode) Done() chan struct{} {
	return t.done
}

func (t *TunnelNode) GetReadChan() *SafeBufferChannel {
	return t.tunnelReadChan
}

func (t *TunnelNode) GetWriteChan() *SafeBufferChannel {
	return t.tunnelWriteChan
}

func (t *TunnelNode) Send(frame []byte, buf core.Buffer) error {
	select {
	case <-t.done:
		if buf != nil {
			buf.Free()
		}
		t.logger.Debug("[Send] Tunnel is close, exit")
		return nil
	default:
		if !t.tunnelWriteChan.Put(&BufferFrame{Frame: frame, Buf: buf}) {
			if buf != nil {
				buf.Free()
			}
			t.logger.Debug("[Send] tunnel is closed, abort sending")
			return core.ErrTunnelClosed
		}
		return nil
	}

}

func (t *TunnelNode) tunnelReader() {
	t.logger.Debug("[tunnelReader] Run tunnelReader")
	for {
		select {
		case <-t.done:
			t.logger.Info("[tunnelReader] Get done signal, exit")
			return
		default:
		}
		frameBuf, frame, err := ReadFrameFromConnection(t.remoteTunnelNode, t.framer)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				t.logger.Warning("[tunnelReader] Close tunel reader: (" + err.Error() + ")")
				t.Stop()
			} else {
				t.logger.Error("[tunnelReader] ERROR in read frame: " + err.Error())
				t.Stop()
			}
			if frameBuf != nil {
				frameBuf.Free()
			}
			return
		}
		if !t.tunnelReadChan.Put(&BufferFrame{Frame: frame, Buf: frameBuf}) {
			t.logger.Debug("[tunnelReader] tunnel is closed or writeChan is full, exit reader")
			frameBuf.Free()
			return
		}
	}
}

func (t *TunnelNode) tunnelWriter() {
	t.logger.Debug("[tunnelWriter] Run tunnelWriter")
	for {
		select {
		case <-t.done:
			t.logger.Debug("[tunnelWriter] Get done signal, exit")
			return
		case <-t.tunnelReadChan.done:
			t.logger.Debug("[tunnelWriter] Read cheannel is close, exit")
			return
		case bufFrame := <-t.tunnelWriteChan.GetChannel():
			t.connMut.Lock()
			if t.remoteTunnelNode == nil {
				t.logger.Debug("[tunnelWriter] remoteTunnelNode is nil, exit")
				t.connMut.Unlock()
				bufFrame.Buf.Free()
				return
			}
			_, err := t.remoteTunnelNode.Write(bufFrame.Frame) // TODO добавить WriteDeadline
			t.connMut.Unlock()
			bufFrame.Buf.Free()
			if err != nil {
				t.logger.Error("[tunnelWriter] Error in send data: " + err.Error())
				t.Stop()
				return
			}
		}
	}
}

func NewTunnelNode(remoteConn net.Conn, framer core.Framer) *TunnelNode {
	node := &TunnelNode{
		tunnelReadChan:   NewSafeBufferChannel(),
		tunnelWriteChan:  NewSafeBufferChannel(),
		remoteTunnelNode: remoteConn,
		framer:           framer,
		isRunning:        true,
		logger:           logger.GetLoggerHub().WithModule("TunnelNode"),
		done:             make(chan struct{}),
	}
	go node.tunnelReader()
	go node.tunnelWriter()
	return node
}
