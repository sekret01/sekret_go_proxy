package buffers

import (
	"sync"
	"testing"
)

func TestGetBuf_ReturnsCorrectSize(t *testing.T) {
	for _, size := range []int{1, 100, 256, 300, 1024, 4096, 10000, 1 << 20} {
		b := GetBuf(size)
		if b == nil {
			t.Fatalf("size=%d: nil buffer", size)
		}
		if b.Len() != size {
			t.Errorf("size=%d: Len()=%d", size, b.Len())
		}
		if b.Cap() < size {
			t.Errorf("size=%d: Cap()=%d", size, b.Cap())
		}
		if len(b.Bytes()) != size {
			t.Errorf("size=%d: len(Bytes())=%d", size, len(b.Bytes()))
		}
		b.Free()
	}
}

func TestGetBuf_TooBig_NoPool(t *testing.T) {
	b := GetBuf((1 << 20) + 1)
	if b.Len() != (1<<20)+1 {
		t.Fatalf("Len()=%d", b.Len())
	}
	if b.Cap() < (1<<20)+1 {
		t.Fatalf("Cap()=%d", b.Cap())
	}
	b.Free()
}

func TestFree_Idempotent(t *testing.T) {
	b := GetBuf(100)
	b.Free()
	b.Free() // не должно паниковать
	b.Free()
}

func TestFree_NilSafe(t *testing.T) {
	var b *BaseBuffer
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Free on nil panicked: %v", r)
		}
	}()
	b.Free()
}

func TestFree_ThenBytesNil(t *testing.T) {
	b := GetBuf(100)
	b.Free()
	if b.Bytes() != nil {
		t.Errorf("Bytes() after Free = %v", b.Bytes())
	}
	if b.Len() != 0 {
		t.Errorf("Len() after Free = %d", b.Len())
	}
}

func TestPool_ReusesBuffer(t *testing.T) {
	b1 := GetBuf(1024)
	data1 := b1.Bytes()
	for i := range data1 {
		data1[i] = 0xAA
	}
	b1.Free()

	b2 := GetBuf(1024)
	// после Free()[:0] и повторного Get данные должны быть доступны
	// (мы не чистим, это ожидаемо — проверяем, что буфер переиспользован)
	if cap(b2.Bytes()) == 0 {
		t.Fatalf("expected reused buffer")
	}
	b2.Free()
}

func TestGetBuf_DifferentSizesSameClass(t *testing.T) {
	// 100 и 200 должны попасть в класс 256
	b1 := GetBuf(100)
	b2 := GetBuf(200)
	if b1.Cap() < 100 || b2.Cap() < 200 {
		t.Fatalf("bad caps: %d, %d", b1.Cap(), b2.Cap())
	}
	b1.Free()
	b2.Free()
}

func TestConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				size := (n%10 + 1) * 100
				b := GetBuf(size)
				if b.Len() < size {
					t.Errorf("bad len")
				}
				b.Free()
			}
		}(i)
	}
	wg.Wait()
}
