package main

import (
	"context"
	"github.com/chestorix/monmetrics/internal/agent"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/chestorix/monmetrics/internal/config"
	"github.com/sirupsen/logrus"
)

type cfg struct {
	Address        string `env:"ADDRESS"`
	SecretKey      string `env:"KEY"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	RateLimit      int    `env:"RATE_LIMIT"`
}

func ensureHTTP(address string) string {
	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		return "http://" + address
	}
	return address
}

func applyFlags(cfg cfg) config.AgentConfig {
	key := cfg.SecretKey
	if cfg.SecretKey == "" {
		key = flagKey
	}

	address := cfg.Address
	if address == "" {
		address = flagRunAddr
	}
	address = ensureHTTP(address)

	reportInterval := cfg.ReportInterval
	if reportInterval == 0 {
		reportInterval = flagReportInterval
	}

	pollInterval := cfg.PollInterval
	if pollInterval == 0 {
		pollInterval = flagPollInterval
	}

	rateLimit := cfg.RateLimit
	if rateLimit == 0 {
		rateLimit = flagRateLimit
	}
	if rateLimit <= 0 {
		rateLimit = 1
	}

	agentCfg := config.AgentConfig{
		Address:        address,
		PollInterval:   time.Duration(pollInterval) * time.Second,
		ReportInterval: time.Duration(reportInterval) * time.Second,
		Key:            key,
	}
	return agentCfg
}

func main() {
	parseFlags()
	var cfg cfg
	if err := env.Parse(&cfg); err != nil {
		log.Fatal("Failed to parse env vars:", err)
	}
	agentCfg := applyFlags(cfg)

	go func() {
		log.Println("Starting pprof server on :8081")
		if err := http.ListenAndServe(":8081", nil); err != nil {
			logrus.WithError(err).Error("pprof server failed")
		}
	}()
	agent := agent.NewAgent(agentCfg)
	agent.Run(context.Background(), cfg.RateLimit)
}
