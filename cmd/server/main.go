package main

import (
	"context"
	"github.com/chestorix/monmetrics/internal/config"
	"github.com/chestorix/monmetrics/internal/domain/interfaces"
	"github.com/chestorix/monmetrics/internal/metrics/repository"
	"github.com/chestorix/monmetrics/internal/utils"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chestorix/monmetrics/internal/api"
	"github.com/chestorix/monmetrics/internal/metrics/service"
	"github.com/sirupsen/logrus"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

var logger *logrus.Logger

func setupLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)
	return logger
}

func main() {
	utils.PrintBuildInfo(buildVersion, buildDate, buildCommit)
	parseFlags()
	flags := map[string]any{
		"flagRunAddr":         flagRunAddr,
		"flagStoreInterval":   flagStoreInterval,
		"flagFileStoragePath": flagFileStoragePath,
		"flagRestore":         flagRestore,
		"flagDatabaseDSN":     flagConnDB,
		"flagKey":             flagKey,
		"flagCryptoKey":       flagCryptoKey,
		"flagTrustedSubnet":   flagTrustedSubnet,
		"flagGRPCAddr":        flagGRPCAddr,
	}
	logger = setupLogger()
	cfg := &config.CfgServerENV{
		ConfigFile: flagConfigFile,
	}
	serverCfg := cfg.ApplyFlags(flags)
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

	grpcServer := api.NewGRPCServer(&serverCfg, metricService, logger)
	go func() {
		logger.Infof("Starting gRPC server on %s", serverCfg.GRPCAddress)
		if err := grpcServer.Start(); err != nil {
			logger.WithError(err).Fatal("gRPC server failed")
		}
	}()

	server := api.NewServer(&serverCfg, metricService, logger)
	setupBackgroundSaver(context.Background(), storage, serverCfg.StoreInterval)
	setupGracefulShutdown(context.Background(), cancel, storage, grpcServer, server)

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

func setupGracefulShutdown(ctx context.Context, cancel context.CancelFunc, storage interfaces.Repository, grpcServer *api.GRPCServer, server *api.Server) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-sigChan
		logger.Info("Shutting down servers...")

		if err := storage.Save(ctx); err != nil {
			logger.WithError(err).Error("Failed to save metrics on shutdown")
		}

		grpcServer.Shutdown()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.WithError(err).Error("HTTP server shutdown error")
		}

		cancel()
		logger.Info("Servers stopped gracefully")
		os.Exit(0)
	}()
}
