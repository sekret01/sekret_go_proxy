package compositehandshake

import (
	"net"

	"github.com/sekret01/sekret_go_proxy/internal/auths"
	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/handshake"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type CompositeHandshake struct {
	steps        []core.HandshakeStep
	logger       logger.Logger
	ctxGenerator core.HandshakeContextGenerator
}

func (a *CompositeHandshake) ServerHandshake(conn net.Conn) (*core.HandshakeContext, error) {
	ctx := a.ctxGenerator.NewHandshakeContext()
	for _, step := range a.steps {
		if err := step.ServerSide(ctx, conn); err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

func (a *CompositeHandshake) ClientHandshake(conn net.Conn) (*core.HandshakeContext, error) {
	ctx := a.ctxGenerator.NewHandshakeContext()
	for _, step := range a.steps {
		if err := step.ClientSide(ctx, conn); err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

func NewConpositeHandshakeAuth(cfg *config.Config, ctxGenerator core.HandshakeContextGenerator) (core.Auth, error) {
	steps := make([]core.HandshakeStep, 0)
	for _, name := range cfg.HandshakeSteps {
		step, err := handshake.NewHandshake(cfg, name)
		if err != nil {
			continue
		}
		steps = append(steps, step)
	}
	return &CompositeHandshake{
		steps:        steps,
		logger:       logger.GetLoggerHub().WithModule("CompositeHandshakeAuth"),
		ctxGenerator: ctxGenerator,
	}, nil
}

func init() {
	auths.Reigstrate("composite_handshake", NewConpositeHandshakeAuth)
}
