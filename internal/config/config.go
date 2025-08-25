// Package config содержит конфигурационные структуры для сервера и агента.
package config

import (
	"strings"
	"time"
)

// ServerConfig содержит конфигурационные параметры сервера.
type ServerConfig struct {
	FileStoragePath string        // путь к файлу для хранения метрик (самый большой)
	DatabaseDSN     string        // строка подключения к БД (если используется)
	Address         string        // адрес и порт сервера (например: ":8080")
	Key             string        // секретный ключ для проверки хешей
	StoreInterval   time.Duration // интервал сохранения метрик на диск (0 - синхронная запись)
	Restore         bool          // восстанавливать метрики из файла при старте
}

// AgentConfig содержит конфигурационные параметры агента.
type AgentConfig struct {
	Address        string        // Адрес сервера для подключения
	Key            string        // Ключ для генерации ХЕШ
	PollInterval   time.Duration // Интервал опроса метрик
	ReportInterval time.Duration // Интервал отправки метрик
}

type CfgAgentENV struct {
	Address        string `env:"ADDRESS"`
	SecretKey      string `env:"KEY"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	RateLimit      int    `env:"RATE_LIMIT"`
}

func ensureHTTP(address string) string {
	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		return "http://" + address
	}
	return address
}

func (cfg *CfgAgentENV) ApplyFlags(mapFlags map[string]any) AgentConfig {
	key := cfg.SecretKey
	if cfg.SecretKey == "" {
		if value, ok := mapFlags["flagKey"].(string); ok {
			key = value
		}
	}

	address := cfg.Address
	if address == "" {
		if value, ok := mapFlags["flagRunAddr"].(string); ok {
			address = value
		}
		address = ensureHTTP(address)
	}
	reportInterval := cfg.ReportInterval
	if reportInterval == 0 {
		if value, ok := mapFlags["flagReportInterval"].(int); ok {
			reportInterval = value
		}
	}
	pollInterval := cfg.PollInterval
	if pollInterval == 0 {
		if value, ok := mapFlags["flagPollInterval"].(int); ok {
			pollInterval = value
		}
	}
	rateLimit := cfg.RateLimit
	if rateLimit == 0 {
		if value, ok := mapFlags["flagRateLimit"].(int); ok {
			rateLimit = value

		}
	}
	if rateLimit <= 0 {
		rateLimit = 1
	}

	agentCfg := AgentConfig{
		Address:        address,
		PollInterval:   time.Duration(pollInterval) * time.Second,
		ReportInterval: time.Duration(reportInterval) * time.Second,
		Key:            key,
	}
	return agentCfg
}
