package buffers

import (
	"sync"

	"github.com/sekret01/sekret_go_proxy/internal/core"
)

type poolClass struct {
	size int
	pool sync.Pool
}

var poolSizes []int = []int{
	256, 512,
	1 << 10, 2 << 10, 4 << 10, 8 << 10,
	16 << 10, 32 << 10, 64 << 10, 128 << 10,
	256 << 10, 512 << 10,
	1 << 20,
}

var pools []*poolClass = func() []*poolClass {
	pl := make([]*poolClass, len(poolSizes))
	for i, size := range poolSizes {
		pl[i] = &poolClass{size: size}
		pl[i].pool.New = func() any {
			b := make([]byte, size)
			return &b
		}
	}
	return pl
}()

type BaseBuffer struct {
	pool  *poolClass
	data  []byte
	freed bool
	mu    sync.Mutex
}

func (b *BaseBuffer) Bytes() []byte {
	return b.data
}

func (b *BaseBuffer) Cap() int {
	return cap(b.data)
}

func (b *BaseBuffer) Len() int {
	return len(b.data)
}

func (b *BaseBuffer) Free() {
	if b == nil {
		return
	}
	b.mu.Lock()
	if b.freed {
		b.mu.Unlock()
		return
	}
	b.freed = true
	b.mu.Unlock()
	if b.pool == nil {
		b.data = nil
		return
	}
	tmp := b.data[:0]
	clear(tmp)
	b.pool.pool.Put(&tmp)
	b.data = nil
}

func GetBuf(size int) core.Buffer {
	for _, s := range pools {
		if s.size >= size {
			b := s.pool.Get().(*[]byte)
			*b = (*b)[:size]
			return &BaseBuffer{
				data:  *b,
				pool:  s,
				freed: false,
			}
		}
	}
	return &BaseBuffer{
		data:  make([]byte, size),
		pool:  nil,
		freed: false,
	}

}
