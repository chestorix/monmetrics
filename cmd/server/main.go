package main

import (
	"github.com/caarlos0/env/v11"
	"github.com/chestorix/monmetrics/internal/api"
	"github.com/chestorix/monmetrics/internal/config"
	"github.com/chestorix/monmetrics/internal/metrics/repository"
	"github.com/chestorix/monmetrics/internal/metrics/service"
	"github.com/sirupsen/logrus"
	"log"
	"strings"
)

type cfg struct {
	Address string `env:"ADDRESS"`
}

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)
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
	storage := repository.NewMemStorage()
	metricService := service.NewService(storage)
	server := api.NewServer(&cfg, metricService, logger)
	if err := server.Start(); err != nil {
		panic(err)
	}

}
