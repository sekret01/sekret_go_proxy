package main

import (
	"fmt"
	"log"

	"github.com/sekret01/sekret_go_proxy/internal/config"

	"github.com/sekret01/sekret_go_proxy/internal/app"
	"github.com/sekret01/sekret_go_proxy/internal/encryptors"
	"github.com/sekret01/sekret_go_proxy/internal/transports"

	_ "github.com/sekret01/sekret_go_proxy/internal/encryptors/mock"
	_ "github.com/sekret01/sekret_go_proxy/internal/transports/tcp"
)

func main() {
	fmt.Printf("# START IMPORTS\n")
	cfg, err := config.LoadConfig("configs/client.example.yaml")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("# CONFIG NAS BEEN IMPORTED\n")
	log.Println(cfg)

	fmt.Printf("# REGISTRATED MODULS\n\n")
	printRegistrates()
	fmt.Println()

	fmt.Printf("START BUILD ClientApp MODULS\n")
	client, err := app.NewClientApp(cfg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("RUN ClientApp\n")
	client.Run()
}

func printRegistrates() {
	fmt.Printf("Encryptors: %s\n", encryptors.List())
	fmt.Printf("Transports: %s\n", transports.List())
}
