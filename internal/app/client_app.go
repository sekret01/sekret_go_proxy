package app

import (
	"github.com/sekret01/sekret_go_proxy/internal/config"
)

type ClientApp struct{}

func (p *ClientApp) Run() {

}

func NewClientApp(cfg *config.Config) *ClientApp {
	return &ClientApp{}
}
