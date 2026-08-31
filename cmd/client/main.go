package main

import (
	"fmt"
	"log"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/framers"

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

	hub := logger.GetLoggerHub()
	consoleLogger := loggers.NewConsoleLogger()
	hub.Registrate(consoleLogger)
	hub.SetLevel(logger.INFO)

	hub.Info("START IMPORTS")
	cfg, err := config.LoadConfig("configs/client.example.yaml")
	if err != nil {
		log.Fatal(err)
	}
	hub.Info("CONFIG NAS BEEN IMPORTED")

	hub.Info("REGISTRATED MODULS")
	printRegistrates(hub)

	hub.Info("START BUILD ClientApp MODULS")
	client, err := app.NewClientApp(cfg)
	if err != nil {
		log.Fatal(err)
	}

	hub.Info("RUN ClientApp")
	client.Run()
}

func printRegistrates(hub logger.LoggerHub) {
	hub.Debug("RUUUN")
	fmt.Printf("Encryptors: %s\n", encryptors.List())
	fmt.Printf("Transports: %s\n", transports.List())
	fmt.Printf("Framers: %s\n", framers.List())
}
