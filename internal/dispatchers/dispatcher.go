package dispatchers

import (
	"fmt"
	"net"
	"sync"

	"github.com/sekret01/sekret_go_proxy/internal/core"
)

type Dispatcher struct {
	connections map[core.RequestID]core.ConnWrapper
	mu          sync.RWMutex
}

var (
	dispatcher *Dispatcher = nil
)

// Регистрация подключения в дистпетчер
func (d *Dispatcher) Register(id core.RequestID, requestConn net.Conn, protoType core.ProtocolType, isTunnel bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	fmt.Printf("[DISPATCHER]: registrate: %x\n", id)
	d.connections[id] = core.ConnWrapper{
		ID:        id,
		Conn:      requestConn,
		ProtoType: protoType,
		IsTunnel:  isTunnel,
	}
}

// Поиск подключения по ID
func (d *Dispatcher) Find(id core.RequestID) (core.ConnWrapper, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	fmt.Println("[DISPATCHER]: len = ", len(d.connections))
	for key := range d.connections {
		fmt.Printf("[DISPATCHER]: key: %x\n", key)
	}
	conn, exist := d.connections[id]
	return conn, exist
}

// Удаление подключения
func (d *Dispatcher) Delete(id core.RequestID) {
	d.mu.Lock()
	defer d.mu.Unlock()
	fmt.Printf("[DISPATCHER]: delete: %x\n", id)
	delete(d.connections, id)
}

// Создает единственный диспетчер
func NewDispatcher() core.Dispatcher {
	if dispatcher == nil {
		dispatcher = &Dispatcher{
			connections: map[core.RequestID]core.ConnWrapper{},
		}
	}
	return dispatcher
}
