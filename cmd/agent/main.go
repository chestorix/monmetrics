package main

import (
	"github.com/caarlos0/env/v11"
	"github.com/chestorix/monmetrics/internal/config"
	"github.com/chestorix/monmetrics/internal/metrics"
	"github.com/chestorix/monmetrics/internal/metrics/collector"
	"github.com/chestorix/monmetrics/internal/metrics/sender"
	"log"
	"strings"
	"time"
)

type cfg struct {
	Address        string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
}

func ensureHTTP(address string) string {
	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		return "http://" + address
	}
	return address
}

func main() {
	parseFlags()
	var cfg cfg
	if err := env.Parse(&cfg); err != nil {
		log.Fatal("Failed to parse env vars:", err)
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

	agentCfg := config.AgentConfig{
		Address:        address,
		PollInterval:   time.Duration(pollInterval) * time.Second,
		ReportInterval: time.Duration(reportInterval) * time.Second,
	}

	collector := collector.NewRuntimeCollector()
	sender := sender.NewHTTPSender(agentCfg.Address)

	pollTicker := time.NewTicker(agentCfg.PollInterval)
	reportTicker := time.NewTicker(agentCfg.ReportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	var lastMetrics []metrics.Metric

	for {
		select {
		case <-pollTicker.C:
			lastMetrics = collector.Collect()
		case <-reportTicker.C:
			if lastMetrics == nil {
				continue
			}
			for _, metric := range lastMetrics {
				if err := sender.Send(metric); err != nil {
					log.Println(err)
				}
			}
		}
	}
}
