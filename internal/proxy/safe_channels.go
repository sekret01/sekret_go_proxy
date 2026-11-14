package proxy

import (
	"sync"
	"sync/atomic"

	"github.com/sekret01/sekret_go_proxy/internal/core"
)

type BufferFrame struct {
	Frame []byte
	Buf   core.Buffer
}

type SafeChannel[T any] struct {
	channel chan T
	isOpen  atomic.Bool
	done    chan struct{}
	once    sync.Once
}

func (s *SafeChannel[T]) Put(data T) bool {
	select {
	case <-s.done:
		return false
	case s.channel <- data:
		return true
	}
}

func (s *SafeChannel[T]) Close() {
	s.once.Do(func() {
		close(s.done)
		s.isOpen.Store(false)
	})
}

func (s *SafeChannel[T]) IsOpen() bool        { return s.isOpen.Load() }
func (s *SafeChannel[T]) Done() chan struct{} { return s.done }
func (s *SafeChannel[T]) GetChannel() chan T  { return s.channel }

func NewSafeChannel[T any](cap int) *SafeChannel[T] {
	s := &SafeChannel[T]{
		channel: make(chan T, cap),
		done:    make(chan struct{}),
	}
	s.isOpen.Store(true)
	return s
}

type SafeBufferChannel = SafeChannel[*BufferFrame]
type SafeMessageChannel = SafeChannel[*core.MessageData]
type SafeByteChannel = SafeChannel[[]byte]

func NewSafeByteChannel() *SafeByteChannel       { return NewSafeChannel[[]byte](1000) }
func NewSafeMessageChannel() *SafeMessageChannel { return NewSafeChannel[*core.MessageData](1000) }
func NewSafeBufferChannel() *SafeBufferChannel   { return NewSafeChannel[*BufferFrame](1000) }
