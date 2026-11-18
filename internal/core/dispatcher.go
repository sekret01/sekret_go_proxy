package core

import (
	"net"
)

type Dispatcher interface {
	Register(id RequestID, requestConn net.Conn, protoType ProtocolType, isTunnel bool)
	Find(id RequestID) (ConnWrapper, bool)
	Delete(id RequestID)
}
