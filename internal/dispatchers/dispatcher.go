package dispatchers

import (
	"net"
	"sync"

	"github.com/sekret01/sekret_go_proxy/internal/core"
)

type Dispatcher struct {
	connections map[core.RequestID]net.Conn
	mu          sync.RWMutex
}

var (
	dispatcher *Dispatcher = nil
)

// Регистрация подключения в дистпетчер
func (d *Dispatcher) Register(id core.RequestID, requestConn net.Conn) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.connections[id] = requestConn
}

// Поиск подключения по ID
func (d *Dispatcher) Find(id core.RequestID) (net.Conn, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	conn, exist := d.connections[id]
	return conn, exist
}

// Удаление подключения
func (d *Dispatcher) Delete(id core.RequestID) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.connections, id)
}

// Создает единственный диспетчер
func NewDispatcher() core.Dispatcher {
	if dispatcher == nil {
		dispatcher = &Dispatcher{}
	}
	return dispatcher
}
