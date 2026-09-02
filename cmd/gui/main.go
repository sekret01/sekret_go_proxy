package main

import (
	"fmt"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	proxyApp "github.com/sekret01/sekret_go_proxy/internal/app"
	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/pkg/logger"
	"github.com/sekret01/sekret_go_proxy/pkg/logger/loggers"

	_ "github.com/sekret01/sekret_go_proxy/internal/encryptors/mock"
	_ "github.com/sekret01/sekret_go_proxy/internal/framers/simple"
	_ "github.com/sekret01/sekret_go_proxy/internal/transports/tcp"
)

func main() {
	buildGui()
}

func setupLogger() logger.LoggerHub {
	hub := logger.GetLoggerHub()
	consoleLogger := loggers.NewConsoleLogger()
	hub.Registrate(consoleLogger)
	hub.SetLevel(logger.INFO)
	return hub
}

func loadConfig() (*config.Config, error) {
	cfg, err := config.LoadConfig("configs/client.yaml")
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func buildGui() {
	myApp := app.New()
	myWindow := myApp.NewWindow("SGoVPN")

	hub := setupLogger()
	cfg, err := loadConfig()
	fmt.Println(cfg)
	if err != nil {
		hub.Error("Error load config " + err.Error())
		return
	}
	clientApp, err := proxyApp.NewClientApp(cfg)
	if err != nil {
		hub.Error("Error load clientApp " + err.Error())
		return
	}

	label := widget.NewLabel("proxy stopped")
	buttonStart := widget.NewButton("start", func() {
		label.SetText("proxy running...")
		go clientApp.Run()
	})
	buttonStop := widget.NewButton("stop", func() {
		label.SetText("типо остановился")
	})
	myWindow.SetContent(container.NewVBox(
		label,
		buttonStart,
		buttonStop,
	))

	myWindow.ShowAndRun()
}
