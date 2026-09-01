package main

import (
	"os"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/framers"
	"github.com/sekret01/sekret_go_proxy/internal/utils"

	"github.com/sekret01/sekret_go_proxy/internal/app"
	"github.com/sekret01/sekret_go_proxy/internal/encryptors"
	"github.com/sekret01/sekret_go_proxy/internal/transports"

	"github.com/sekret01/sekret_go_proxy/pkg/logger"
	"github.com/sekret01/sekret_go_proxy/pkg/logger/loggers"

	_ "github.com/sekret01/sekret_go_proxy/internal/encryptors/mock"
	_ "github.com/sekret01/sekret_go_proxy/internal/framers/simple"
	_ "github.com/sekret01/sekret_go_proxy/internal/transports/tcp"
)

func main() {

	hub := setupLogger()
	hub.Info("START APP")

	hub.Info("START LOAD CONFIG")
	cfg := loadConfig(hub)
	setLoggerConfig(hub, cfg)

	printRegistrates(hub)

	hub.Info("START BUILD CLIENT-APP MODULS")
	client := builClientApp(hub, cfg)

	hub.Info("RUN CLIENT-APP")
	client.Run()
}

func loadConfig(hub logger.LoggerHub) *config.Config {
	cfg, err := config.LoadConfig("configs/client.yaml")
	if err != nil {
		hub.Critical("[loadConfig]: " + err.Error())
		criticalExit()
	}
	return cfg
}

func setupLogger() logger.LoggerHub {
	hub := logger.GetLoggerHub()
	consoleLogger := loggers.NewConsoleLogger()
	hub.Registrate(consoleLogger)
	hub.SetLevel(logger.INFO)
	return hub
}

func setLoggerConfig(hub logger.LoggerHub, cfg *config.Config) {
	switch cfg.LoggerLevel {
	case logger.DebugStr:
		hub.SetLevel(logger.DEBUG)
	case logger.InfoStr:
		hub.SetLevel(logger.INFO)
	case logger.WarningStr:
		hub.SetLevel(logger.WARNING)
	case logger.ErrorStr:
		hub.SetLevel(logger.ERROR)
	case logger.CriticalStr:
		hub.SetLevel(logger.CRITICAL)
	default:
		hub.Critical("Unknown log_level in configs: " + cfg.LoggerLevel)
		criticalExit()
	}
}

func builClientApp(hub logger.LoggerHub, cfg *config.Config) app.ClientApp {
	client, err := app.NewClientApp(cfg)
	if err != nil {
		hub.Critical("[loadConfig]: " + err.Error())
		criticalExit()
	}
	return *client
}

func printRegistrates(hub logger.LoggerHub) {
	hub.Debug("List of registrated moduls")
	hub.Debug("Encryptors: " + utils.ListToString(encryptors.List()))
	hub.Debug("Transports: " + utils.ListToString(transports.List()))
	hub.Debug("Framers: " + utils.ListToString(framers.List()))
}

func criticalExit() {
	os.Exit(1)
}
