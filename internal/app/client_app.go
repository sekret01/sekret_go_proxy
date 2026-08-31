package app

import (
	"fmt"

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

type ClientApp struct {
	tunnel proxy.ClientTunnel
}

func (p *ClientApp) Run() {
	p.tunnel.Start()
}

func NewClientApp(cfg *config.Config) (*ClientApp, error) {
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

	tunnel := proxy.NewClientTunnel(transport,
		encryptor,
		framer,
		dispatcher,
		detector,
		cfg,
		logger.GetLoggerHub().WithModule("ClientApp"))
	return &ClientApp{
		tunnel: *tunnel,
	}, nil
}

func isContinue(err error) bool {
	if err != nil {
		fmt.Printf("Error in build: %s\n", err)
		return false
	}
	return true
}
