// Package agent - содержит логику инициализации агента сбора метрик.
package agent

import (
	"context"
	"sync"
	"time"

	"github.com/chestorix/monmetrics/internal/config"
	models "github.com/chestorix/monmetrics/internal/metrics"
	"github.com/chestorix/monmetrics/internal/metrics/collector"
	"github.com/chestorix/monmetrics/internal/metrics/sender"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

type Agent struct {
	collector *collector.RuntimeCollector
	sender    *sender.HTTPSender
	cfg       config.AgentConfig
}

func NewAgent(cfg config.AgentConfig) *Agent {
	return &Agent{
		cfg:       cfg,
		sender:    sender.NewHTTPSender(cfg.Address, cfg.Key),
		collector: collector.NewRuntimeCollector(),
	}
}

func (a *Agent) Run(ctx context.Context, rateLimit int) {
	metricsChan := make(chan []models.Metric, 100)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		a.collectRuntimeMetrics(ctx, metricsChan)
	}()

	go func() {
		defer wg.Done()
		a.collectGopsutilMetrics(ctx, metricsChan)
	}()

	a.processMetrics(ctx, metricsChan, rateLimit)

	wg.Wait()
	close(metricsChan)
}

func (a *Agent) collectRuntimeMetrics(ctx context.Context, metricsChan chan<- []models.Metric) {
	ticker := time.NewTicker(a.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			metricsChan <- a.collector.Collect()
		case <-ctx.Done():
			return
		}
	}
}

func (a *Agent) collectGopsutilMetrics(ctx context.Context, metricsChan chan<- []models.Metric) {
	ticker := time.NewTicker(a.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
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

		case <-ctx.Done():
			return
		}
	}
}

func (a *Agent) processMetrics(ctx context.Context, metricsChan <-chan []models.Metric, rateLimit int) {
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

			if err := a.sender.SendBatch(metricsToSend); err != nil {
				for _, metric := range batch {
					if err := a.sender.SendJSON(metric); err != nil {
						continue
					}
				}
			}
		}(metricsBatch)
	}

	wg.Wait()
}
