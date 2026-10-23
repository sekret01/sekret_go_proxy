package defaultdispatcher

import (
	"net"
	"sync"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/dispatchers"
	"github.com/sekret01/sekret_go_proxy/internal/utils"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type DefaultDispatcher struct {
	connections map[core.RequestID]core.ConnWrapper
	mu          sync.RWMutex
	logger      logger.Logger
}

// Регистрация подключения в дистпетчер
func (d *DefaultDispatcher) Register(id core.RequestID, requestConn net.Conn, protoType core.ProtocolType, isTunnel bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.logger.Debug("Registrate [ " + requestConn.LocalAddr().String() + " ] > [ " + utils.RequestIdToString(id) + " ]")
	d.connections[id] = core.ConnWrapper{
		ID:        id,
		Conn:      requestConn,
		ProtoType: protoType,
		IsTunnel:  isTunnel,
	}
}

// Поиск подключения по ID
func (d *DefaultDispatcher) Find(id core.RequestID) (core.ConnWrapper, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	d.logger.Debug("Try to find [ " + utils.RequestIdToString(id) + " ]")
	conn, exist := d.connections[id]
	return conn, exist
}

// Удаление подключения
func (d *DefaultDispatcher) Delete(id core.RequestID) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.logger.Debug("Delete: [ " + utils.RequestIdToString(id) + " ]")
	delete(d.connections, id)
}

// Создает единственный диспетчер
func NewDefaultDispatcher(cfg *config.Config) (core.Dispatcher, error) {
	return &DefaultDispatcher{
		connections: map[core.RequestID]core.ConnWrapper{},
		logger:      logger.GetLoggerHub().WithModule("Dispatcher"),
	}, nil
}

func init() {
	dispatchers.Register("default", NewDefaultDispatcher)
}
