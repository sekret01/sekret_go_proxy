package app

import (
	"github.com/sekret01/sekret_go_proxy/internal/auths"
	"github.com/sekret01/sekret_go_proxy/internal/bootstrap"
	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/detectors"
	"github.com/sekret01/sekret_go_proxy/internal/dispatchers"
	"github.com/sekret01/sekret_go_proxy/internal/filters"
	"github.com/sekret01/sekret_go_proxy/internal/framers"
	"github.com/sekret01/sekret_go_proxy/internal/proxy"
	"github.com/sekret01/sekret_go_proxy/internal/proxy/client"
	"github.com/sekret01/sekret_go_proxy/internal/proxy/server"
	"github.com/sekret01/sekret_go_proxy/internal/transports"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type ClientApp struct {
	tunnel       *client.ClientProxy
	filterServer *server.ServerProxy
}

func (p *ClientApp) Start() error {
	if p.filterServer != nil {
		p.filterServer.Start()
	}
	return p.tunnel.Start()
}

func (p *ClientApp) Stop() error {
	if p.filterServer != nil {
		p.filterServer.Stop()
	}
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

	transport, err := transports.NewTransport(cfg.TransportType)
	buildSuccess = buildSuccess && isContinue(err)
	framer, err := framers.NewFramer(cfg.FramerType)
	buildSuccess = buildSuccess && isContinue(err)
	dispatcher, err := dispatchers.NewDispatcher(cfg.DispatcherType)
	buildSuccess = buildSuccess && isContinue(err)
	ctxGenConnector := &core.HandshakeGeneratorConnector{}
	auth, err := auths.NewAuth(cfg.AuthType, ctxGenConnector)
	buildSuccess = buildSuccess && isContinue(err)
	filterAuth, err := auths.NewAuth(cfg.AuthType, ctxGenConnector)
	buildSuccess = buildSuccess && isContinue(err)
	filter, err := filters.NewFilter(cfg.FilterType, cfg.FilterActive, cfg.FilterDomain, cfg.FilterAddresses)
	buildSuccess = buildSuccess && isContinue(err)
	detector := detectors.NewDetector()
	filterBootstrap, err := bootstrap.NewBootstrap("default", &bootstrap.BootstrapConfig{HandshakeSteps: cfg.HandshakeSteps})
	buildSuccess = buildSuccess && isContinue(err)
	mainBootstrap, err := bootstrap.NewBootstrap("default", &bootstrap.BootstrapConfig{HandshakeSteps: cfg.HandshakeSteps})
	buildSuccess = buildSuccess && isContinue(err)

	if !buildSuccess {
		logger.GetLoggerHub().Error("Errors in building moduls, stop program")
		return nil, core.ErrBuildClientApp
	}

	tunnelEngin := client.NewTunnelEngine(
		transport,
		framer,
		logger.GetLoggerHub().WithModule("TunnelEngine"),
		auth,
		mainBootstrap,
		cfg.RemoteHost,
		cfg.EncryptorType,
	)
	filterTunnelEngin := client.NewTunnelEngine(
		transport,
		framer,
		logger.GetLoggerHub().WithModule("TunnelEngine [Filter]"),
		filterAuth,
		filterBootstrap,
		"127.0.0.1:35785", // TODO конфиг
		"default",         // TODO конфиг
	)

	tunnel := client.NewClientProxy(
		tunnelEngin,
		filterTunnelEngin,
		filter,
		detector,
		dispatcher,
		cfg.LocalHost,
		logger.GetLoggerHub().WithModule("ClientProxy"),
	)

	var filterServer *server.ServerProxy = nil
	if cfg.FilterActive {
		filterServer, err = NewFilterServerApp(&cfg.FilterServer)
		if err != nil {
			logger.GetLoggerHub().Error("Errors in building filterServer, continue without it")
			return nil, core.ErrBuildClientApp
		}
	}
	ctxGenConnector.SubscribeGenerator(tunnel)
	return &ClientApp{
		tunnel:       tunnel,
		filterServer: filterServer,
	}, nil
}

func NewFilterServerApp(cfg *config.FilterServerConfig) (*server.ServerProxy, error) {
	buildSuccess := true

	transport, err := transports.NewTransport(cfg.Transport)
	buildSuccess = buildSuccess && isContinue(err)
	framer, err := framers.NewFramer(cfg.Framer)
	buildSuccess = buildSuccess && isContinue(err)
	dispatcher, err := dispatchers.NewDispatcher(cfg.Dispatcher)
	buildSuccess = buildSuccess && isContinue(err)
	ctxGenConnector := &core.HandshakeGeneratorConnector{}
	auth, err := auths.NewAuth(cfg.Auth, ctxGenConnector)
	buildSuccess = buildSuccess && isContinue(err)
	detector := detectors.NewDetector()
	bootstrap, err := bootstrap.NewBootstrap("default", &bootstrap.BootstrapConfig{HandshakeSteps: []string{}})

	if !buildSuccess {
		logger.GetLoggerHub().Error("Errors in building moduls, stop program")
		return nil, core.ErrBuildServerApp
	}

	serverTunnelEngine := server.NewServerTunnelEngine(
		transport,
		framer,
		bootstrap,
		auth,
		cfg.Host,
		cfg.Encryptor,
		logger.GetLoggerHub().WithModule("ServerTunnelEngine [Filter]"),
	)

	proxy := server.NewServerProxy(
		serverTunnelEngine,
		dispatcher,
		detector,
		logger.GetLoggerHub().WithModule("ServerProxy [Filter]"),
	)
	ctxGenConnector.SubscribeGenerator(proxy)
	return proxy, nil
}

func isContinue(err error) bool {
	if err != nil {
		logger.GetLoggerHub().Error("Error in build: " + err.Error())
		return false
	}
	return true
}
