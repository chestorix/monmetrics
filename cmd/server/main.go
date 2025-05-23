package main

import (
	"context"
	"fmt"
	"github.com/caarlos0/env/v11"
	"github.com/chestorix/monmetrics/internal/api"
	"github.com/chestorix/monmetrics/internal/config"
	"github.com/chestorix/monmetrics/internal/metrics/repository"
	"github.com/chestorix/monmetrics/internal/metrics/service"
	"github.com/sirupsen/logrus"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// test commit
type cfg struct {
	Address         string `env:"ADDRESS"`
	StoreInterval   int    `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
}

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)
	var conf cfg
	if err := env.Parse(&conf); err != nil {
		log.Fatal("Failed to parse env vars:", err)
	}
	parseFlags()

	serverAddress := conf.Address
	if serverAddress == "" {
		serverAddress = flagRunAddr
	}
	if !strings.Contains(serverAddress, ":") {
		serverAddress = ":" + serverAddress
	}

	storeInterval := conf.StoreInterval
	if storeInterval == 0 {
		storeInterval = flagStoreInterval
	}

	fileStoragePath := conf.FileStoragePath
	if fileStoragePath == "" {
		fileStoragePath = flagFileStoragePath
	}

	restore := conf.Restore
	if !restore {
		restore = flagRestore
	}
	dbDSN := conf.DatabaseDSN
	if dbDSN == "" {
		dbDSN = flagConnDB

	}
	cfg := config.ServerConfig{
		Address:         serverAddress,
		StoreInterval:   time.Duration(storeInterval) * time.Second,
		FileStoragePath: fileStoragePath,
		Restore:         restore,
		DatabaseDSN:     dbDSN,
	}
	storage := repository.NewMemStorage(cfg.FileStoragePath)
	fmt.Println(cfg)
	if cfg.Restore {
		if err := storage.Load(); err != nil {
			logger.WithError(err).Error("Failed to load metrics from file")
		}
	}

	metricService := service.NewService(storage)
	server := api.NewServer(&cfg, metricService, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if cfg.StoreInterval > 0 {
		go func() {
			ticker := time.NewTicker(cfg.StoreInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if err := storage.Save(); err != nil {
						logger.WithError(err).Error("Failed to save metrics")
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-sigChan
		logger.Info("Shutting down server...")

		if err := storage.Save(); err != nil {
			logger.WithError(err).Error("Failed to save metrics on shutdown")
		}

		cancel()
		os.Exit(0)
	}()

	if err := server.Start(); err != nil {
		logger.WithError(err).Fatal("Server failed")
	}

}
