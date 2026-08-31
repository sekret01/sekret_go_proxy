package app

import (
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
	tunnel proxy.ServerTunnel
}

func (s *ServerApp) Run() {
	s.tunnel.Start()
}

func NewServerApp(cfg *config.Config) (*ServerApp, error) {
	buildSuccess := true

	encryptor, err := encryptors.NewEncryptor(cfg.EncryptorType, cfg)
	buildSuccess = isContinue(err)
	transport, err := transports.NewTransport(cfg)
	buildSuccess = isContinue(err)
	framer, err := framers.NewFramer(cfg)
	buildSuccess = isContinue(err)
	dispatcher := dispatchers.NewDispatcher()
	detector := detectors.NewDetector()

	if !buildSuccess {
		logger.GetLoggerHub().Error("Errors in building moduls, stop program")
		return nil, core.ErrBuildClientApp
	}

	tunnel := proxy.NewServerTunnel(transport,
		encryptor,
		framer,
		dispatcher,
		detector,
		cfg,
		logger.GetLoggerHub().WithModule("ServerApp"))
	return &ServerApp{
		tunnel: *tunnel,
	}, nil
}
