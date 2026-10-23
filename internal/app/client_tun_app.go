package app

import (
	"runtime"

	"github.com/sekret01/sekret_go_proxy/internal/auths"
	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/encryptors"
	"github.com/sekret01/sekret_go_proxy/internal/framers"
	"github.com/sekret01/sekret_go_proxy/internal/handshake"
	"github.com/sekret01/sekret_go_proxy/internal/proxy"
	"github.com/sekret01/sekret_go_proxy/internal/proxy/client"
	"github.com/sekret01/sekret_go_proxy/internal/proxy/packets"
	"github.com/sekret01/sekret_go_proxy/internal/transports"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type ClientTunApp struct {
	tunnel *client.ClientTun
}

func (p *ClientTunApp) Start() error {
	return p.tunnel.Start()
}

func (p *ClientTunApp) Stop() error {
	return p.tunnel.Stop()
}

func (p *ClientTunApp) Status() *proxy.TunnelStatus {
	return p.tunnel.GetStatus()
}

func (p *ClientTunApp) GetInfo() string {
	return "Client App"
}

func (p *ClientTunApp) Users() []string {
	return []string{}
}

func (p *ClientTunApp) GetLogs() []logger.LogMessage {
	return logger.GetLoggerHub().GetLogs()
}

func NewClientTunApp(cfg *config.Config) (*ClientTunApp, error) {
	buildSuccess := true

	encryptor, err := encryptors.NewEncryptor(cfg.EncryptorType, []byte(cfg.Key32Bytes), cfg) // TODO убрать []]byte(cfg.Key32Bytes)
	buildSuccess = buildSuccess && isContinue(err)
	transport, err := transports.NewTransport(cfg)
	buildSuccess = buildSuccess && isContinue(err)
	framer, err := framers.NewFramer(cfg)
	buildSuccess = buildSuccess && isContinue(err)
	buildSuccess = buildSuccess && isContinue(err)
	ctxGenConnector := &core.HandshakeGeneratorConnector{}
	auth, err := auths.NewAuth(cfg, ctxGenConnector)
	buildSuccess = buildSuccess && isContinue(err)
	bootstrap := handshake.NewBootstrap(*cfg)

	if !buildSuccess {
		logger.GetLoggerHub().Error("Errors in building moduls, stop program")
		return nil, core.ErrBuildClientApp
	}

	tunnelEngin := client.NewTunnelEngine(
		transport,
		encryptor,
		framer,
		cfg,
		logger.GetLoggerHub().WithModule("TunnelEngine"),
		auth,
		bootstrap,
	)

	packetDevice, err := packets.NewPacketDevice(runtime.GOOS, cfg) // TODO конфиг / env
	if err != nil {
		logger.GetLoggerHub().Error("Errors in building moduls, stop program")
		return nil, core.ErrBuildClientApp
	}

	tunnel := client.NewClientTun(
		tunnelEngin,
		packetDevice,
		logger.GetLoggerHub().WithModule("ClientApp"),
	)
	ctxGenConnector.SubscribeGenerator(tunnel)
	return &ClientTunApp{
		tunnel: tunnel,
	}, nil
}
