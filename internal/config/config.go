package config

import "time"

type ServerConfig struct {
	Address string
	//StoreInterval time.Duration
	//StoreFile     string
	//Restore       bool
}

type AgentConfig struct {
	Address        string
	PollInterval   time.Duration
	ReportInterval time.Duration
}
