package main

import (
	"github.com/caarlos0/env/v11"
	"github.com/chestorix/monmetrics/internal/config"
	"github.com/chestorix/monmetrics/internal/server"
	"github.com/chestorix/monmetrics/internal/storage/memory"
	"log"
	"strings"
)

type cfg struct {
	Address string `env:"ADDRESS"`
}

func main() {
	parseFlags()
	var conf cfg
	if err := env.Parse(&conf); err != nil {
		log.Fatal("Failed to parse env vars:", err)
	}
	serverAddress := conf.Address
	if serverAddress == "" {
		serverAddress = flagRunAddr
	}
	if !strings.Contains(serverAddress, ":") {
		serverAddress = ":" + serverAddress
	}
	cfg := config.ServerConfig{
		Address: serverAddress,
	}

	storage := memory.NewMemStorage()
	srv := server.New(&cfg, storage)

	if err := srv.Start(); err != nil {
		panic(err)
	}

}
