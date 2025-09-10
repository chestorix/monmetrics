package main

import (
	"context"
	"github.com/caarlos0/env/v11"
	"github.com/chestorix/monmetrics/internal/agent"
	"github.com/chestorix/monmetrics/internal/config"
	"github.com/chestorix/monmetrics/internal/utils"
	"github.com/sirupsen/logrus"
	"net/http"
	_ "net/http/pprof"
	"os/signal"
	"sync"
	"syscall"
	"time"
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
		"flagKey":            flagKey,
		"flagRunAddr":        flagRunAddr,
		"flagReportInterval": flagReportInterval,
		"flagPollInterval":   flagPollInterval,
		"flagRateLimit":      flagRateLimit,
		"flagCryptoKey":      flagCryptoKey,
	}
	logger = setupLogger()

	var cfg config.CfgAgentENV
	cfg.ConfigFile = flagConfigFile
	if err := env.Parse(&cfg); err != nil {
		logger.Fatal("Failed to parse env vars:", err)
	}
	agentCfg := cfg.ApplyFlags(flags)

	go func() {
		logger.Info("Starting pprof server on :8081")
		if err := http.ListenAndServe(":8081", nil); err != nil {
			logger.WithError(err).Error("pprof server failed")
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	shutdownCtx := setupAgentGracefulShutdown(cancel)

	agent := agent.NewAgent(agentCfg, logger)
	agentDone := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		agentDone <- agent.Run(ctx, cfg.RateLimit)
	}()
	select {
	case err := <-agentDone:
		if err != nil {
			logger.WithError(err).Error("Agent stopped with error")
		} else {
			logger.Info("Agent stopped normally")
		}
	case <-shutdownCtx.Done():
		logger.Info("Shutdown signal received. Waiting for agent to finish...")
		// Даем агенту время на завершение
		select {
		case <-time.After(5 * time.Second):
			logger.Warn("Agent did not stop in time, forcing shutdown")
		case err := <-agentDone:
			if err != nil {
				logger.WithError(err).Error("Agent stopped with error during shutdown")
			} else {
				logger.Info("Agent stopped gracefully during shutdown")
			}
		}
	}
}

func setupAgentGracefulShutdown(cancel context.CancelFunc) (shutdownCtx context.Context) {
	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-shutdownCtx.Done()
		logger.Info("Received shutdown signal. Initiating graceful shutdown...")
		stop()
		cancel()
	}()
	return shutdownCtx
}
