package core

import (
	"net"
)

// Интерфейс диспетчера управления сессиями.
// Работает с парами request-ID и connection.
type Dispatcher interface {
	Register(id RequestID, requestConn net.Conn)
	Find(id RequestID) (net.Conn, bool)
	Delete(id RequestID)
}
