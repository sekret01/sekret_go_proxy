package app

import (
	"github.com/sekret01/sekret_go_proxy/internal/auths"
	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/detectors"
	"github.com/sekret01/sekret_go_proxy/internal/dispatchers"
	"github.com/sekret01/sekret_go_proxy/internal/framers"
	"github.com/sekret01/sekret_go_proxy/internal/handshake"
	"github.com/sekret01/sekret_go_proxy/internal/proxy"
	"github.com/sekret01/sekret_go_proxy/internal/proxy/client"
	"github.com/sekret01/sekret_go_proxy/internal/transports"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type ClientApp struct {
	tunnel *client.ClientProxy
}

func (p *ClientApp) Start() error {
	return p.tunnel.Start()
}

func (p *ClientApp) Stop() error {
	return p.tunnel.Stop()
}

func (p *ClientApp) Status() *proxy.TunnelStatus {
	return p.tunnel.GetStatus()
}

func (p *ClientApp) GetInfo() string {
	return "Client App"
}

func (p *ClientApp) Users() []string {
	return []string{}
}

func (p *ClientApp) GetLogs() []logger.LogMessage {
	return logger.GetLoggerHub().GetLogs()
}

func NewClientApp(cfg *config.Config) (*ClientApp, error) {
	buildSuccess := true

	transport, err := transports.NewTransport(cfg)
	buildSuccess = buildSuccess && isContinue(err)
	framer, err := framers.NewFramer(cfg)
	buildSuccess = buildSuccess && isContinue(err)
	dispatcher, err := dispatchers.NewDispatcher(cfg)
	buildSuccess = buildSuccess && isContinue(err)
	ctxGenConnector := &core.HandshakeGeneratorConnector{}
	auth, err := auths.NewAuth(cfg, ctxGenConnector)
	buildSuccess = buildSuccess && isContinue(err)
	detector := detectors.NewDetector()
	bootstrap := handshake.NewBootstrap(*cfg)

	if !buildSuccess {
		logger.GetLoggerHub().Error("Errors in building moduls, stop program")
		return nil, core.ErrBuildClientApp
	}

	tunnelEngin := client.NewTunnelEngine(
		transport,
		framer,
		cfg,
		logger.GetLoggerHub().WithModule("TunnelEngine"),
		auth,
		bootstrap,
	)

	tunnel := client.NewClientProxy(
		tunnelEngin,
		detector,
		dispatcher,
		cfg,
		logger.GetLoggerHub().WithModule("ClientProxy"),
	)
	ctxGenConnector.SubscribeGenerator(tunnel)
	return &ClientApp{
		tunnel: tunnel,
	}, nil
}

func isContinue(err error) bool {
	if err != nil {
		logger.GetLoggerHub().Error("Error in build: " + err.Error())
		return false
	}
	return true
}
