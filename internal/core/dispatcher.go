package core

import (
	"net"
)

// Интерфейс диспетчера управления сессиями.
// Работает с парами request-ID и connection.
type Dispatcher interface {
	Register(id RequestID, requestConn net.Conn, protoType ProtocolType, isTunnel bool)
	Find(id RequestID) (ConnWrapper, bool)
	Delete(id RequestID)
}
