package config

import "time"

type ServerConfig struct {
	Address         string
	StoreInterval   time.Duration
	FileStoragePath string
	Restore         bool
	DatabaseDSN     string
	Key             string
}

type AgentConfig struct {
	Address        string
	PollInterval   time.Duration
	ReportInterval time.Duration
	Key            string
}
