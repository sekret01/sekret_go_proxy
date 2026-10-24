package app

import (
	"runtime"

	"github.com/sekret01/sekret_go_proxy/internal/auths"
	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/framers"
	"github.com/sekret01/sekret_go_proxy/internal/handshake"
	"github.com/sekret01/sekret_go_proxy/internal/proxy"
	"github.com/sekret01/sekret_go_proxy/internal/proxy/packets"
	"github.com/sekret01/sekret_go_proxy/internal/proxy/server"
	"github.com/sekret01/sekret_go_proxy/internal/transports"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type ServerTunApp struct {
	tunnel *server.ServerTun
}

func (p *ServerTunApp) Start() error {
	return p.tunnel.Start()
}

func (p *ServerTunApp) Stop() error {
	return p.tunnel.Stop()
}

func (p *ServerTunApp) Status() *proxy.TunnelStatus {
	return p.tunnel.GetStatus()
}

func (p *ServerTunApp) GetInfo() string {
	return "Server Tun App"
}

func (p *ServerTunApp) Users() []string {
	return []string{}
}

func (p *ServerTunApp) GetLogs() []logger.LogMessage {
	return logger.GetLoggerHub().GetLogs()
}

func NewServerTunApp(cfg *config.Config) (*ServerTunApp, error) {
	buildSuccess := true

	transport, err := transports.NewTransport(cfg)
	buildSuccess = buildSuccess && isContinue(err)
	framer, err := framers.NewFramer(cfg)
	buildSuccess = buildSuccess && isContinue(err)
	buildSuccess = buildSuccess && isContinue(err)
	ctxGenConnector := &core.HandshakeGeneratorConnector{}
	auth, err := auths.NewAuth(cfg, ctxGenConnector)
	buildSuccess = buildSuccess && isContinue(err)
	bootstrap := handshake.NewBootstrap(cfg)

	if !buildSuccess {
		logger.GetLoggerHub().Error("Errors in building moduls, stop program")
		return nil, core.ErrBuildServerApp
	}

	tunnelEngin := server.NewServerTunnelEngine(
		transport,
		framer,
		bootstrap,
		auth,
		cfg,
		logger.GetLoggerHub().WithModule("ServerTunnelEngine"),
	)

	packetDevice, err := packets.NewPacketDevice(runtime.GOOS, cfg) // TODO конфиг / env
	if err != nil {
		logger.GetLoggerHub().Error("Errors in building moduls, stop program")
		return nil, core.ErrBuildServerApp
	}

	tunnel := server.NewServerTun(
		tunnelEngin,
		packetDevice,
		logger.GetLoggerHub().WithModule("ServerTun"),
		cfg,
	)
	ctxGenConnector.SubscribeGenerator(tunnel)
	return &ServerTunApp{
		tunnel: tunnel,
	}, nil
}
