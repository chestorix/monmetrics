package main

import (
	"context"
	"github.com/chestorix/monmetrics/internal/domain/interfaces"
	"github.com/chestorix/monmetrics/internal/metrics/repository"
	"github.com/chestorix/monmetrics/internal/utils"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/chestorix/monmetrics/internal/api"
	"github.com/chestorix/monmetrics/internal/config"
	"github.com/chestorix/monmetrics/internal/metrics/service"
	"github.com/sirupsen/logrus"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

type cfg struct {
	Address         string `env:"ADDRESS"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
	SecretKey       string `env:"KEY"`
	StoreInterval   int    `env:"STORE_INTERVAL"`
	Restore         bool   `env:"RESTORE"`
}

var logger *logrus.Logger

func setupLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)
	return logger
}
func loadConfig() config.ServerConfig {
	var conf cfg
	if err := env.Parse(&conf); err != nil {
		log.Fatal("Failed to parse env vars:", err)
	}
	parseFlags()

	key := conf.SecretKey
	if conf.SecretKey == "" {
		key = flagKey
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

	cfg := config.ServerConfig{
		Address:         serverAddress,
		StoreInterval:   time.Duration(storeInterval) * time.Second,
		FileStoragePath: fileStoragePath,
		Restore:         restore,
		DatabaseDSN:     dbDSN,
		Key:             key,
	}
	return cfg
}

func main() {
	utils.PrintBuildInfo(buildVersion, buildDate, buildCommit)
	logger = setupLogger()
	cfg := loadConfig()
	var err error
	storage, err := repository.NewInitStorage().CreateStorage(cfg.DatabaseDSN, cfg.FileStoragePath)
	if err != nil {
		logger.Fatalf("Failed to create storage: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if cfg.Restore && cfg.FileStoragePath != "" && cfg.DatabaseDSN == "" {
		if err := storage.Load(ctx); err != nil {
			logger.WithError(err).Error("Failed to load metrics from file")
		}
	}

	metricService := service.NewService(storage)
	server := api.NewServer(&cfg, metricService, logger)
	setupBackgroundSaver(context.Background(), storage, cfg.StoreInterval)
	setupGracefulShutdown(context.Background(), cancel, storage, server)

	if err := server.Start(); err != nil {
		logger.WithError(err).Fatal("Server failed")
	}

}
func setupBackgroundSaver(ctx context.Context, storage interfaces.Repository, interval time.Duration) {
	if interval > 0 {
		go func() {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if err := storage.Save(ctx); err != nil {
						logger.WithError(err).Error("Failed to save metrics")
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}
}

func setupGracefulShutdown(ctx context.Context, cancel context.CancelFunc, storage interfaces.Repository, server *api.Server) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-sigChan
		logger.Info("Shutting down server...")

		if err := storage.Save(ctx); err != nil {
			logger.WithError(err).Error("Failed to save metrics on shutdown")
		}

		cancel()
		os.Exit(0)
	}()
}
