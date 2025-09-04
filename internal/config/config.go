// Package config содержит конфигурационные структуры для сервера и агента.
package config

import (
	"github.com/caarlos0/env/v11"
	"log"
	"strings"
	"time"
)

// ServerConfig содержит конфигурационные параметры сервера.
type ServerConfig struct {
	FileStoragePath string        // путь к файлу для хранения метрик (самый большой)
	DatabaseDSN     string        // строка подключения к БД (если используется)
	Address         string        // адрес и порт сервера (например: ":8080")
	Key             string        // секретный ключ для проверки хешей
	CryptoKey       string        // Путь к файлу с приватным ключом
	StoreInterval   time.Duration // интервал сохранения метрик на диск (0 - синхронная запись)
	Restore         bool          // восстанавливать метрики из файла при старте
}

// AgentConfig содержит конфигурационные параметры агента.
type AgentConfig struct {
	Address        string        // Адрес сервера для подключения
	Key            string        // Ключ для генерации ХЕШ
	CryptoKey      string        //Путь к фалу с публичным ключом
	PollInterval   time.Duration // Интервал опроса метрик
	ReportInterval time.Duration // Интервал отправки метрик
}

type CfgAgentENV struct {
	Address        string `env:"ADDRESS"`
	SecretKey      string `env:"KEY"`
	CryptoKey      string `env:"CRYPTO_KEY"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	RateLimit      int    `env:"RATE_LIMIT"`
}

type CfgServerENV struct {
	Address         string `env:"ADDRESS"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
	SecretKey       string `env:"KEY"`
	CryptoKey       string `env:"CRYPTO_KEY"`
	StoreInterval   int    `env:"STORE_INTERVAL"`
	Restore         bool   `env:"RESTORE"`
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
	cryptoKey := cfg.CryptoKey
	if cryptoKey == "" {
		if value, ok := mapFlags["flagCryptoKey"].(string); ok {
			cryptoKey = value
		}
	}

	agentCfg := AgentConfig{
		Address:        address,
		PollInterval:   time.Duration(pollInterval) * time.Second,
		ReportInterval: time.Duration(reportInterval) * time.Second,
		Key:            key,
		CryptoKey:      cryptoKey,
	}
	return agentCfg
}

func (conf *CfgServerENV) ApplyFlags(mapFlags map[string]any) ServerConfig {
	if err := env.Parse(conf); err != nil {
		log.Fatal("Failed to parse env vars:", err)
	}

	key := conf.SecretKey
	if conf.SecretKey == "" {
		if value, ok := mapFlags["flagKey"].(string); ok {
			key = value
		}
	}
	serverAddress := conf.Address
	if serverAddress == "" {
		if value, ok := mapFlags["flagRunAddr"].(string); ok {
			serverAddress = value
		}
	}
	if !strings.Contains(serverAddress, ":") {
		serverAddress = ":" + serverAddress
	}

	storeInterval := conf.StoreInterval
	if storeInterval == 0 {
		if value, ok := mapFlags["flagStoreInterval"].(int); ok {
			storeInterval = value
		}
	}

	fileStoragePath := conf.FileStoragePath
	if fileStoragePath == "" {
		if value, ok := mapFlags["flagFileStoragePath"].(string); ok {
			fileStoragePath = value
		}
	}

	restore := conf.Restore
	if !restore {
		if value, ok := mapFlags["flagRestoreInterval"].(bool); ok {
			restore = value
		}
	}
	dbDSN := conf.DatabaseDSN
	if dbDSN == "" {
		if value, ok := mapFlags["flagDatabaseDSN"].(string); ok {
			dbDSN = value
		}

	}
	cryptoKey := conf.CryptoKey
	if cryptoKey == "" {
		if value, ok := mapFlags["flagCryptoKey"].(string); ok {
			cryptoKey = value
		}
	}

	cfg := ServerConfig{
		Address:         serverAddress,
		StoreInterval:   time.Duration(storeInterval) * time.Second,
		FileStoragePath: fileStoragePath,
		Restore:         restore,
		DatabaseDSN:     dbDSN,
		Key:             key,
		CryptoKey:       cryptoKey,
	}
	return cfg
}
