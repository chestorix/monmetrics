package main

import (
	"github.com/caarlos0/env/v11"
	"github.com/chestorix/monmetrics/internal/config"
	"github.com/chestorix/monmetrics/internal/metrics"
	"github.com/chestorix/monmetrics/internal/metrics/collector"
	"github.com/chestorix/monmetrics/internal/metrics/sender"
	"github.com/chestorix/monmetrics/internal/utils"
	"github.com/sirupsen/logrus"
	"log"
	"strings"
	"time"
)

type cfg struct {
	Address        string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	Key            string `env:"KEY"`
}

func ensureHTTP(address string) string {
	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		return "http://" + address
	}
	return address
}
func startAgent(agentCfg config.AgentConfig) {
	collector := collector.NewRuntimeCollector()
	sender := sender.NewHTTPSender(agentCfg.Address, agentCfg.Key)

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
			var batch []models.Metrics
			for _, m := range lastMetrics {
				metric := models.Metrics{
					ID:    m.Name,
					MType: m.Type,
				}
				switch m.Type {
				case models.Gauge:
					if val, ok := m.Value.(float64); ok {
						metric.Value = &val
					}
				case models.Counter:
					if val, ok := m.Value.(int64); ok {
						metric.Delta = &val
					}
				}
				batch = append(batch, metric)
			}
			retryDelays := []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}

			err := utils.Retry(3, retryDelays, func() error {
				if err := sender.SendBatch(batch); err != nil {
					for _, metric := range lastMetrics {
						if err := sender.SendJSON(metric); err != nil {
							return err
						}
					}
				}
				return nil
			})
			if err != nil {
				logrus.Info("Failed to send metrics after retries:", err)
			}
		}
	}
}

func main() {
	parseFlags()
	var cfg cfg
	if err := env.Parse(&cfg); err != nil {
		log.Fatal("Failed to parse env vars:", err)
	}
	key := flagKey
	if key == "" {
		key = cfg.Key
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
		Key:            key,
	}
	startAgent(agentCfg)
}
