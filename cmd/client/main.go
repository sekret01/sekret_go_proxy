package main

import (
	"log"

	"github.com/sekret01/sekret_go_proxy/internal/config"

	_ "github.com/sekret01/sekret_go_proxy/internal/encryptors/mock"
	_ "github.com/sekret01/sekret_go_proxy/internal/transports/tcp"
)

func main() {
	cfg, err := config.LoadConfig("configs/client.example.yaml")
	if err != nil {
		log.Fatal(err)
	}
	// TEST
	log.Println(cfg)
}
