package proxy

import (
	"errors"
	"net"
	"sync"

	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type TunnelNode struct {
	tunnelReadChan   *SafeChannel
	tunnelWriteChan  *SafeChannel
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

func (t *TunnelNode) GetReadChan() *SafeChannel {
	t.logger.Debug("[GetReadChan] Read chan taken")
	return t.tunnelReadChan
}

func (t *TunnelNode) GetWriteChan() *SafeChannel {
	t.logger.Debug("[GetWriteChan] Write chan taken")
	return t.tunnelWriteChan
}

func (t *TunnelNode) Send(frame []byte) error {
	t.connMut.Lock()
	defer t.connMut.Unlock()
	if !t.isRunning {
		t.logger.Debug("[Send] Tunnel is close, exit")
		return nil
	}
	t.logger.Debug("[Send] Put frame in write chan")
	t.tunnelWriteChan.Put(frame)
	return nil
}

func (t *TunnelNode) tunnelReader() {
	t.logger.Debug("[tunnelReader] Run tunnelReader")
	for {
		select {
		case <-t.done:
			return
		default:
		}
		frame, err := ReadFrameFromConnection(t.remoteTunnelNode, t.framer)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				t.logger.Warning("[tunnelReader] Close tunel reader: (" + err.Error() + ")")
			} else {
				t.logger.Error("[tunnelReader] ERROR in read frame: " + err.Error())
				t.Stop()
			}
			return
		}
		if !t.tunnelReadChan.Put(frame) {
			t.logger.Debug("[tunnelReader] tunnel is closed, exit reader")
			return
		}
		t.logger.Debug("[tunnelReader] Put frame in read chan")
	}
}

func (t *TunnelNode) tunnelWriter() {
	t.logger.Debug("[tunnelWriter] Run tunnelWriter")
	for {
		select {
		case <-t.done:
			t.logger.Debug("[tunnelWriter] Tunnel is close, exit")
			return
		case frame := <-t.tunnelWriteChan.GetChannel():
			t.connMut.Lock()
			if t.remoteTunnelNode == nil {
				t.logger.Debug("[tunnelWriter] remoteTunnelNode is nil, exit")
				t.connMut.Unlock()
				return
			}
			_, err := t.remoteTunnelNode.Write(frame) // TODO мб через done
			t.connMut.Unlock()
			if err != nil {
				t.logger.Error("[tunnelWriter] Error in send data: " + err.Error())
				continue
			}
			t.logger.Debug("[tunnelChanReader] Data has been sent")
		}
	}
}

func NewTunnelNode(remoteConn net.Conn, framer core.Framer) *TunnelNode {
	node := &TunnelNode{
		tunnelReadChan:   NewSafeChannel(),
		tunnelWriteChan:  NewSafeChannel(),
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

// CHAN-HELP

type SafeChannel struct {
	channel chan []byte
	isOpen  bool
	done    chan struct{}
	once    sync.Once
}

func (s *SafeChannel) Put(data []byte) bool {
	select {
	case <-s.done:
		return false
	case s.channel <- data:
		return true
	}
}

func (s *SafeChannel) IsOpen() bool {
	return s.isOpen
}

func (s *SafeChannel) Done() chan struct{} {
	return s.done
}

func (s *SafeChannel) GetChannel() chan []byte {
	return s.channel
}

func (s *SafeChannel) Close() {
	s.once.Do(func() {
		close(s.done)
		s.isOpen = false
	})
}

func NewSafeChannel() *SafeChannel {
	return &SafeChannel{
		channel: make(chan []byte, 1000),
		isOpen:  true,
		done:    make(chan struct{}),
	}
}
