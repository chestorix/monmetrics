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
	}

	var cfg config.CfgAgentENV
	if err := env.Parse(cfg); err != nil {
		log.Fatal("Failed to parse env vars:", err)
	}
	agentCfg := cfg.ApplyFlags(flags)

	go func() {
		log.Println("Starting pprof server on :8081")
		if err := http.ListenAndServe(":8081", nil); err != nil {
			logrus.WithError(err).Error("pprof server failed")
		}
	}()
	agent := agent.NewAgent(agentCfg)
	agent.Run(context.Background(), cfg.RateLimit)
}
