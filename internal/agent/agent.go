// Package agent - содержит логику инициализации агента сбора метрик.
package agent

import (
	"context"
	"github.com/sirupsen/logrus"
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
	logger    *logrus.Logger
}

func NewAgent(cfg config.AgentConfig, logger *logrus.Logger) *Agent {
	return &Agent{
		cfg:       cfg,
		sender:    sender.NewHTTPSender(cfg.Address, cfg.Key, cfg.CryptoKey, logger),
		collector: collector.NewRuntimeCollector(),
		logger:    logger,
	}
}

func (a *Agent) Run(ctx context.Context, rateLimit int) error {
	a.logger.Info("Starting agent...")
	metricsChan := make(chan []models.Metric, 100)

	var wg sync.WaitGroup
	wg.Add(2)
	collectCtx, cancelCollect := context.WithCancel(ctx)
	defer cancelCollect()

	go func() {
		defer wg.Done()
		a.collectRuntimeMetrics(collectCtx, metricsChan)
	}()

	go func() {
		defer wg.Done()
		a.collectGopsutilMetrics(collectCtx, metricsChan)
	}()
	processCtx, cancelProcess := context.WithCancel(ctx)
	defer cancelProcess()

	var processWg sync.WaitGroup
	processWg.Add(1)
	go func() {
		defer processWg.Done()
		a.processMetrics(processCtx, metricsChan, rateLimit)
	}()

	wg.Wait()
	close(metricsChan)
	processWg.Wait()

	a.logger.Info("Agent stopped gracefully")
	return nil
}

func (a *Agent) collectRuntimeMetrics(ctx context.Context, metricsChan chan<- []models.Metric) {
	a.logger.Info("Starting runtime metrics collection")
	ticker := time.NewTicker(a.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			metrics := a.collector.Collect()
			a.logger.Infof("Collected %d runtime metrics", len(metrics))

			select {
			case <-ctx.Done():
				a.logger.Info("Stopping runtime metrics collection")
				return
			case metricsChan <- metrics:

			}

		case <-ctx.Done():
			a.logger.Info("Stopping runtime metrics collection")
			return
		}
	}
}
func (a *Agent) collectGopsutilMetrics(ctx context.Context, metricsChan chan<- []models.Metric) {
	a.logger.Info("Starting gopsutil metrics collection")
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
			a.logger.Infof("Collected %d gopsutil metrics", len(gopsutilMetrics))

			if len(gopsutilMetrics) > 0 {

				select {
				case <-ctx.Done():
					a.logger.Info("Stopping gopsutil metrics collection")
					return
				case metricsChan <- gopsutilMetrics:

				}
			}

		case <-ctx.Done():
			a.logger.Info("Stopping gopsutil metrics collection")
			return
		}
	}
}

func (a *Agent) processMetrics(ctx context.Context, metricsChan <-chan []models.Metric, rateLimit int) {
	a.logger.Info("Starting metrics processing (simple mode)")

	sendTicker := time.NewTicker(a.cfg.ReportInterval)
	defer sendTicker.Stop()

	var metricsBuffer []models.Metrics

	for {
		select {
		case metricsBatch, ok := <-metricsChan:
			if !ok {
				return
			}

			for _, m := range metricsBatch {
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
				metricsBuffer = append(metricsBuffer, metric)
			}

		case <-sendTicker.C:
			if len(metricsBuffer) > 0 {
				a.logger.Infof("Sending %d metrics", len(metricsBuffer))
				if err := a.sender.SendBatch(metricsBuffer); err != nil {
					a.logger.Infof("Send failed: %v", err)
				} else {
					a.logger.Infof("Send successful")
				}
				metricsBuffer = nil
			}

		case <-ctx.Done():

			if len(metricsBuffer) > 0 {
				a.logger.Infof("Sending remaining %d metrics before shutdown", len(metricsBuffer))
				if err := a.sender.SendBatch(metricsBuffer); err != nil {
					a.logger.Infof("Final send failed: %v", err)
				} else {
					a.logger.Info("Final send successful")
				}
			}
			a.logger.Info("Stopping metrics processing")
			return
		}
	}
}
