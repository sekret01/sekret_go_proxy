package app

import (
	"github.com/sekret01/sekret_go_proxy/internal/auths"
	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/detectors"
	"github.com/sekret01/sekret_go_proxy/internal/dispatchers"
	"github.com/sekret01/sekret_go_proxy/internal/encryptors"
	"github.com/sekret01/sekret_go_proxy/internal/framers"
	"github.com/sekret01/sekret_go_proxy/internal/proxy"
	"github.com/sekret01/sekret_go_proxy/internal/transports"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type ServerApp struct {
	tunnel *proxy.ServerTunnel
}

func (s *ServerApp) Start() error {
	return s.tunnel.Start()
}

func (p *ServerApp) Stop() error {
	return p.tunnel.Stop()
}

func (p *ServerApp) Status() bool {
	return p.tunnel.IsRunning()
}

func (p *ServerApp) GetInfo() string {
	return "Server App"
}

func (p *ServerApp) Users() []string {
	return []string{}
}

func NewServerApp(cfg *config.Config) (*ServerApp, error) {
	buildSuccess := true

	encryptor, err := encryptors.NewEncryptor(cfg.EncryptorType, cfg)
	buildSuccess = buildSuccess && isContinue(err)
	transport, err := transports.NewTransport(cfg)
	buildSuccess = buildSuccess && isContinue(err)
	framer, err := framers.NewFramer(cfg)
	buildSuccess = buildSuccess && isContinue(err)
	dispatcher, err := dispatchers.NewDispatcher(cfg)
	buildSuccess = buildSuccess && isContinue(err)
	auth, err := auths.NewEncryptor(cfg)
	buildSuccess = buildSuccess && isContinue(err)
	detector := detectors.NewDetector()

	if !buildSuccess {
		logger.GetLoggerHub().Error("Errors in building moduls, stop program")
		return nil, core.ErrBuildServerApp
	}

	tunnel := proxy.NewServerTunnel(transport,
		encryptor,
		framer,
		dispatcher,
		detector,
		cfg,
		logger.GetLoggerHub().WithModule("ServerApp"),
		auth)
	return &ServerApp{
		tunnel: tunnel,
	}, nil
}
