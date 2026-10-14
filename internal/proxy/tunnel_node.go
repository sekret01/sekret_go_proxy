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
}

func (t *TunnelNode) Stop() error {
	t.logger.Debug("[Stop] Close tunnel")
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
	for t.isRunning {
		frame, err := ReadFrameFromConnection(t.remoteTunnelNode, t.framer)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				t.logger.Warning("[tunnelReader] Close tunel reader: (" + err.Error() + ")")
			} else {
				t.logger.Error("[tunnelReader] ERROR in read frame: " + err.Error())
			}
			t.Stop()
			return
		}
		t.logger.Debug("[tunnelReader] Put frame in read chan")
		t.tunnelReadChan.Put(frame)
	}
	t.logger.Debug("[tunnelReader] Exit from read cycle")
}

func (t *TunnelNode) tunnelWriter() {
	t.logger.Debug("[tunnelWriter] Run tunnelWriter")
	channel := t.tunnelWriteChan.GetChannel()
	for frame := range channel {
		if !t.isRunning {
			t.logger.Debug("[tunnelWriter] Tunnel is close, exit")
			return
		}
		t.connMut.Lock()
		if t.remoteTunnelNode == nil {
			t.logger.Debug("[tunnelWriter] remoteTunnelNode is nil, exit")
			t.connMut.Unlock()
			return
		}
		_, err := t.remoteTunnelNode.Write(frame)
		t.connMut.Unlock()
		if err != nil {
			t.logger.Error("[tunnelWriter] Error in send data: " + err.Error())
			continue
		}
		t.logger.Debug("[tunnelChanReader] Data has been sent")
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
	}
	go node.tunnelReader()
	go node.tunnelWriter()
	return node
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
