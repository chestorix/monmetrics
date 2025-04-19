package main

import (
	"github.com/chestorix/monmetrics/internal/collector"
	"github.com/chestorix/monmetrics/internal/config"
	"github.com/chestorix/monmetrics/internal/sender"
	"log"
	"time"
)

func main() {
	cfg := config.AgentConfig{
		Address:        "http://localhost:8080",
		PollInterval:   2 * time.Second,
		ReportInterval: 10 * time.Second,
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
