package dispatchers

import (
	"net"
	"sync"

	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type Dispatcher struct {
	connections map[core.RequestID]core.ConnWrapper
	mu          sync.RWMutex
	logger      logger.Logger
}

var (
	dispatcher *Dispatcher = nil
)

// Регистрация подключения в дистпетчер
func (d *Dispatcher) Register(id core.RequestID, requestConn net.Conn, protoType core.ProtocolType, isTunnel bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.logger.Debug("Registrate [ " + string(id[:]) + " ]")
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
	d.logger.Debug("Try to find [ " + string(id[:]) + " ]")
	conn, exist := d.connections[id]
	return conn, exist
}

// Удаление подключения
func (d *Dispatcher) Delete(id core.RequestID) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.logger.Debug("Delete: [ " + string(id[:]) + " ]")
	delete(d.connections, id)
}

// Создает единственный диспетчер
func NewDispatcher() core.Dispatcher {
	if dispatcher == nil {
		dispatcher = &Dispatcher{
			connections: map[core.RequestID]core.ConnWrapper{},
			logger:      logger.GetLoggerHub().WithModule("Dispatcher"),
		}
	}
	return dispatcher
}
