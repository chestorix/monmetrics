package main

import (
	"context"
	"github.com/caarlos0/env/v11"
	"github.com/chestorix/monmetrics/internal/agent"
	"github.com/chestorix/monmetrics/internal/config"
	"github.com/chestorix/monmetrics/internal/utils"
	"github.com/sirupsen/logrus"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

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

	var cfg config.CfgAgentENV
	cfg.ConfigFile = flagConfigFile
	if err := env.Parse(&cfg); err != nil {
		log.Fatal("Failed to parse env vars:", err)
	}
	agentCfg := cfg.ApplyFlags(flags)

	go func() {
		log.Println("Starting pprof server on :8081")
		if err := http.ListenAndServe(":8081", nil); err != nil {
			logrus.WithError(err).Error("pprof server failed")
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setupAgentGracefulShutdown(ctx, cancel)

	agent := agent.NewAgent(agentCfg)
	agent.Run(context.Background(), cfg.RateLimit)
}

func setupAgentGracefulShutdown(ctx context.Context, cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v. Shutting down agent gracefully...", sig)

		cancel()

		time.Sleep(1 * time.Second)

		log.Println("Agent shutdown completed")
		os.Exit(0)
	}()
}
