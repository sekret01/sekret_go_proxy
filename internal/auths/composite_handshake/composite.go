package composite_handshake

import (
	"net"

	"github.com/sekret01/sekret_go_proxy/internal/auths"
	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/handshake"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type CompositeHandshakeAuth struct {
	cfg          *config.Config
	steps        []core.HandshakeStep
	logger       logger.Logger
	ctxGenerator core.HandshakeContextGenerator
	enc          core.Encryptor
}

func (a *CompositeHandshakeAuth) ServerHandshake(conn net.Conn) (*core.HandshakeContext, error) {
	ctx := a.ctxGenerator.NewHandshakeContext()
	secureConn := core.NewSecureConn(conn, a.enc)
	for _, step := range a.steps {
		if err := step.ServerSide(ctx, secureConn); err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

func (a *CompositeHandshakeAuth) ClientHandshake(conn net.Conn) (*core.HandshakeContext, error) {
	ctx := a.ctxGenerator.NewHandshakeContext()
	secureConn := core.NewSecureConn(conn, a.enc)
	for _, step := range a.steps {
		if err := step.ClientSide(ctx, secureConn); err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

func (a *CompositeHandshakeAuth) SetEncryptor(enc core.Encryptor) {
	a.enc = enc
}

func (a *CompositeHandshakeAuth) SetSteps(steps []string) error {
	a.steps = []core.HandshakeStep{}
	for _, name := range steps {
		step, err := handshake.NewHandshake(a.cfg, name)
		if err != nil {
			return core.ErrInvalidHandshakeStepKey
		}
		a.steps = append(a.steps, step)
	}
	return nil
}

func NewConpositeHandshakeAuth(
	cfg *config.Config,
	ctxGenerator core.HandshakeContextGenerator,
) (core.Auth, error) {
	return &CompositeHandshakeAuth{
		steps:        []core.HandshakeStep{},
		cfg:          cfg,
		logger:       logger.GetLoggerHub().WithModule("CompositeHandshakeAuth"),
		ctxGenerator: ctxGenerator,
	}, nil
}

func init() {
	auths.Reigstrate("composite_handshake", NewConpositeHandshakeAuth)
}
