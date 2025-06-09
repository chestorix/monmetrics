package main

import (
	"github.com/caarlos0/env/v11"
	"github.com/chestorix/monmetrics/internal/config"
	"github.com/chestorix/monmetrics/internal/metrics"
	"github.com/chestorix/monmetrics/internal/metrics/collector"
	"github.com/chestorix/monmetrics/internal/metrics/sender"
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

	//var lastMetrics []models.Metric
	var metricsBatch []models.Metrics

	for {
		select {
		case <-pollTicker.C:

			metrics := collector.Collect()
			metricsBatch = nil
			for _, m := range metrics {
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
				metricsBatch = append(metricsBatch, metric)
			}
		case <-reportTicker.C:
			if len(metricsBatch) == 0 {
				continue
			}
			err := sender.SendBatch(metricsBatch)
			if err != nil {
				logrus.Info("Batch send failed, falling back to individual sends:", err)

				for _, metric := range metricsBatch {
					if err := sender.SendJSON(models.Metric{
						Name: metric.ID,
						Type: metric.MType,
						Value: func() interface{} {
							if metric.MType == models.Gauge {
								return *metric.Value
							}
							return *metric.Delta
						}(),
					}); err != nil {
						logrus.Info("Failed to send metric:", metric.ID, "error:", err)
					}
				}
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
