package main

import (
	"github.com/chestorix/monmetrics/internal/collector"
	"github.com/chestorix/monmetrics/internal/config"
	"github.com/chestorix/monmetrics/internal/sender"
	"log"
	"time"
)

func main() {
	parseFlags()
	cfg := config.AgentConfig{
		Address:        "http://localhost" + flagRunAddr,
		PollInterval:   time.Duration(flagPollInterval) * time.Second,
		ReportInterval: time.Duration(flagReportInterval) * time.Second,
	}

	collector := collector.NewRuntimeCollector()
	sender := sender.NewHTTPSender(cfg.Address)

	pollTicker := time.NewTicker(cfg.PollInterval)
	reportTicker := time.NewTicker(cfg.ReportInterval)

	for {
		select {
		case <-pollTicker.C:
		case <-reportTicker.C:
			metrics := collector.Collect()
			for _, metric := range metrics {
				if err := sender.Send(metric); err != nil {
					log.Println(err)
				}
			}
		}
	}
}
