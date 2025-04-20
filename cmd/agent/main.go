package main

import (
	"github.com/caarlos0/env/v11"
	"github.com/chestorix/monmetrics/internal/collector"
	"github.com/chestorix/monmetrics/internal/config"
	"github.com/chestorix/monmetrics/internal/models"
	"github.com/chestorix/monmetrics/internal/sender"
	"log"
	"time"
)

type cfg struct {
	Address        string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
}

func main() {
	parseFlags()
	var cfg cfg
	if err := env.Parse(&cfg); err != nil {
		log.Fatal("Failed to parse env vars:", err)
	}
	address := cfg.Address
	if address == "" {
		address = "http://" + flagRunAddr
	}

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

	var lastMetrics []models.Metric

	for {
		select {
		case <-pollTicker.C:
			lastMetrics = collector.Collect()
		case <-reportTicker.C:
			if lastMetrics == nil {
				continue
			}
			//	metrics := collector.Collect()
			for _, metric := range lastMetrics {
				if err := sender.Send(metric); err != nil {
					log.Println(err)
				}
			}
		}
	}
}
