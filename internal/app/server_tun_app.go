package app

import (
	"github.com/sekret01/sekret_go_proxy/internal/auths"
	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/encryptors"
	"github.com/sekret01/sekret_go_proxy/internal/framers"
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

	encryptor, err := encryptors.NewEncryptor(cfg.EncryptorType, cfg)
	buildSuccess = buildSuccess && isContinue(err)
	transport, err := transports.NewTransport(cfg)
	buildSuccess = buildSuccess && isContinue(err)
	framer, err := framers.NewFramer(cfg)
	buildSuccess = buildSuccess && isContinue(err)
	buildSuccess = buildSuccess && isContinue(err)
	auth, err := auths.NewEncryptor(cfg)
	buildSuccess = buildSuccess && isContinue(err)

	if !buildSuccess {
		logger.GetLoggerHub().Error("Errors in building moduls, stop program")
		return nil, core.ErrBuildServerApp
	}

	tunnelEngin := server.NewServerTunnelEngine(
		transport,
		encryptor,
		framer,
		auth,
		cfg,
		logger.GetLoggerHub().WithModule("ServerTunnelEngine"),
	)

	packetDevice, err := packets.NewPacketDevice("linux", cfg) // TODO конфиг / env
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
	return &ServerTunApp{
		tunnel: tunnel,
	}, nil
}
