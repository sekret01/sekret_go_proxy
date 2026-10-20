package compositehandshake

import (
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type CompositeHandshake struct {
	teps   []*core.HandshakeStep
	logger logger.Logger
}
