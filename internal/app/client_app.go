package app

import (
	"fmt"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/encryptors"
	"github.com/sekret01/sekret_go_proxy/internal/framers"
	"github.com/sekret01/sekret_go_proxy/internal/proxy"
	"github.com/sekret01/sekret_go_proxy/internal/transports"
)

type ClientApp struct {
	tunnel proxy.ClientTunnel
}

func (p *ClientApp) Run() {

}

func NewClientApp(cfg *config.Config) (*ClientApp, error) {
	buildSuccess := true

	encryptor, err := encryptors.NewEncryptor(cfg.EncryptorType, cfg)
	buildSuccess = isContinue(err)
	transport, err := transports.NewTransport(cfg)
	buildSuccess = isContinue(err)
	framer, err := framers.NewFramer(cfg)

	if !buildSuccess {
		fmt.Printf("Errors in building moduls, stop program\n")
		return nil, core.ErrBuildClientApp
	}

	tunnel := proxy.NewClientTunnel(transport, encryptor, framer, nil, nil)
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
