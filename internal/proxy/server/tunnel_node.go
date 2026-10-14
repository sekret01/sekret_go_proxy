package server

import (
	"errors"
	"net"
	"sync"

	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/proxy"
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
}

func (t *TunnelNode) Stop() error {
	t.isRunning = false
	t.remoteTunnelNode.Close()
	t.remoteTunnelNode = nil
	t.tunnelReadChan.Close()
	t.tunnelWriteChan.Close()
	return nil
}

func (t *TunnelNode) GetReadChan() *SafeChannel {
	return t.tunnelReadChan
}

func (t *TunnelNode) GetWriteChan() *SafeChannel {
	return t.tunnelWriteChan
}

func (t *TunnelNode) Send(frame []byte) error {
	t.tunnelWriteChan.Put(frame)
	return nil
}

func (t *TunnelNode) tunnelReader() {
	for t.isRunning {
		frame, err := proxy.ReadFrameFromConnection(t.remoteTunnelNode, t.framer)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				t.logger.Warning("[tunnelReader] Close tunel reader: (" + err.Error() + ")")
			} else {
				t.logger.Error("[tunnelReader] ERROR in read frame: " + err.Error())
			}
			t.Stop()
			return
		}
		t.tunnelReadChan.Put(frame)
	}
}

func (t *TunnelNode) tunnelWriter() {
	channel := t.tunnelWriteChan.GetChannel()
	for frame := range channel {
		if !t.isRunning {
			return
		}
		t.connMut.Lock()
		if t.remoteTunnelNode == nil {
			return
		}
		_, err := t.remoteTunnelNode.Write(frame)
		if err != nil {
			t.logger.Error("[tunnelWriter] Error in send data: " + err.Error())
			continue
		}
	}
}

func NewTunnelNode(remoteConn net.Conn) *TunnelNode {
	return &TunnelNode{
		tunnelReadChan:   NewSafeChannel(),
		tunnelWriteChan:  NewSafeChannel(),
		remoteTunnelNode: remoteConn,
		isRunning:        true,
	}
}

// CHAN-HELP

type SafeChannel struct {
	channel chan []byte
	isOpen  bool
	mut     sync.Mutex
}

func (s *SafeChannel) Put(data []byte) {
	s.mut.Lock()
	defer s.mut.Unlock()
	if !s.isOpen {
		return
	}
	s.channel <- data
}

func (s *SafeChannel) IsOpen() bool {
	s.mut.Lock()
	defer s.mut.Unlock()
	return s.isOpen
}

func (s *SafeChannel) GetChannel() chan []byte {
	return s.channel
}

func (s *SafeChannel) Close() {
	s.mut.Lock()
	defer s.mut.Unlock()
	if !s.isOpen {
		return
	}
	s.isOpen = false
	close(s.channel)
}

func NewSafeChannel() *SafeChannel {
	return &SafeChannel{
		channel: make(chan []byte, 1000),
		isOpen:  true,
	}
}
