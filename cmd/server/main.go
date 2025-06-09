package main

import (
	"context"
	"github.com/caarlos0/env/v11"
	"github.com/chestorix/monmetrics/internal/api"
	"github.com/chestorix/monmetrics/internal/config"
	"github.com/chestorix/monmetrics/internal/domain/interfaces"
	"github.com/chestorix/monmetrics/internal/metrics/repository"
	"github.com/chestorix/monmetrics/internal/metrics/service"
	"github.com/chestorix/monmetrics/internal/utils"
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
	Key             string `env:"KEY"`
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
	key := flagKey
	if key == "" {
		key = conf.Key
	}
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
	var storage interfaces.Repository

	cfg := config.ServerConfig{
		Address:         serverAddress,
		StoreInterval:   time.Duration(storeInterval) * time.Second,
		FileStoragePath: fileStoragePath,
		Restore:         restore,
		DatabaseDSN:     dbDSN,
		Key:             key,
	}
	logger.Println(cfg)
	if dbDSN != "" {
		retryDelays := []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}
		pgStorage, err := repository.NewPostgresStorage(dbDSN)
		err = utils.Retry(3, retryDelays, func() error {

			if err != nil {
				return err
			}

			storage = pgStorage

			return nil
		})
		if err != nil {
			logger.Fatalf("Failed to initialize PostgreSQL storage after retries: %v", err)
		}

		logger.Info("Using PostgreSQL storage")
		defer pgStorage.Close()
	} else if cfg.FileStoragePath != "" {
		storage = repository.NewMemStorage(cfg.FileStoragePath)
		logger.Info("Using file storage")
	} else {
		storage = repository.NewMemStorage("")
		logger.Info("Using in-memory storage")
	}

	if cfg.Restore && cfg.FileStoragePath != "" && dbDSN == "" {
		if err := storage.Load(); err != nil {
			logger.WithError(err).Error("Failed to load metrics from file")
		}
	}

	metricService := service.NewService(storage)
	server := api.NewServer(&cfg, metricService, logger, key)

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
