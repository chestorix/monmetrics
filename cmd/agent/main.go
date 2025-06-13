package main

import (
	"github.com/caarlos0/env/v11"
	"github.com/chestorix/monmetrics/internal/config"
	"github.com/chestorix/monmetrics/internal/metrics"
	"github.com/chestorix/monmetrics/internal/metrics/collector"
	"github.com/chestorix/monmetrics/internal/metrics/sender"
	"github.com/chestorix/monmetrics/internal/utils"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/sirupsen/logrus"
	"log"
	"strings"
	"sync"
	"time"
)

type cfg struct {
	Address        string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	SecretKey      string `env:"KEY"`
	RateLimit      int    `env:"RATE_LIMIT"`
}

func ensureHTTP(address string) string {
	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		return "http://" + address
	}
	return address
}
func collectRuntimeMetrics(collector *collector.RuntimeCollector, metricsChan chan<- []models.Metric, pollInterval time.Duration) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for range ticker.C {
		metricsChan <- collector.Collect()
	}
}

func collectGopsutilMetrics(metricsChan chan<- []models.Metric, pollInterval time.Duration) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for range ticker.C {
		var gopsutilMetrics []models.Metric

		if memStat, err := mem.VirtualMemory(); err == nil {
			gopsutilMetrics = append(gopsutilMetrics,
				models.Metric{Name: "TotalMemory", Type: models.Gauge, Value: float64(memStat.Total)},
				models.Metric{Name: "FreeMemory", Type: models.Gauge, Value: float64(memStat.Free)},
			)
		}

		if cpuStats, err := cpu.Percent(time.Second, true); err == nil {
			for i, percent := range cpuStats {
				gopsutilMetrics = append(gopsutilMetrics,
					models.Metric{Name: "CPUutilization" + string(rune('1'+i)), Type: models.Gauge, Value: percent},
				)
			}
		}

		if len(gopsutilMetrics) > 0 {
			metricsChan <- gopsutilMetrics
		}
	}
}

func processMetrics(metricsChan <-chan []models.Metric, sender *sender.HTTPSender, rateLimit int) {
	var wg sync.WaitGroup
	limiter := make(chan struct{}, rateLimit)

	for metricsBatch := range metricsChan {
		limiter <- struct{}{}
		wg.Add(1)

		go func(batch []models.Metric) {
			defer func() {
				<-limiter
				wg.Done()
			}()

			var metricsToSend []models.Metrics
			for _, m := range batch {
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
				metricsToSend = append(metricsToSend, metric)
			}

			retryDelays := []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}
			err := utils.Retry(3, retryDelays, func() error {
				if err := sender.SendBatch(metricsToSend); err != nil {
					for _, metric := range batch {
						if err := sender.SendJSON(metric); err != nil {
							return err
						}
					}
				}
				return nil
			})

			if err != nil {
				logrus.WithError(err).Error("Failed to send metrics after retries")
			}
		}(metricsBatch)
	}

	wg.Wait()
}

func startAgent(agentCfg config.AgentConfig, rateLimit int) {
	collector := collector.NewRuntimeCollector()
	sender := sender.NewHTTPSender(agentCfg.Address, agentCfg.Key)

	metricsChan := make(chan []models.Metric, 100)

	go collectRuntimeMetrics(collector, metricsChan, agentCfg.PollInterval)
	go collectGopsutilMetrics(metricsChan, agentCfg.PollInterval)

	processMetrics(metricsChan, sender, rateLimit)
}

func main() {
	parseFlags()
	var cfg cfg
	if err := env.Parse(&cfg); err != nil {
		log.Fatal("Failed to parse env vars:", err)
	}

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
		rateLimit = 1 // Default to at least 1 worker
	}

	agentCfg := config.AgentConfig{
		Address:        address,
		PollInterval:   time.Duration(pollInterval) * time.Second,
		ReportInterval: time.Duration(reportInterval) * time.Second,
		Key:            key,
	}

	startAgent(agentCfg, rateLimit)
}
