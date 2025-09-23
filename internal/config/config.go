// Package config содержит конфигурационные структуры для сервера и агента.
package config

import (
	"encoding/json"
	"fmt"
	"github.com/caarlos0/env/v11"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var logger = logrus.New()

// ServerConfig содержит конфигурационные параметры сервера.
type ServerConfig struct {
	FileStoragePath string        `json:"store_file"`     // путь к файлу для хранения метрик (самый большой)
	DatabaseDSN     string        `json:"database_dsn"`   // строка подключения к БД (если используется)
	Address         string        `json:"address"`        // адрес и порт сервера (например: ":8080")
	Key             string        `json:"key"`            // секретный ключ для проверки хешей
	CryptoKey       string        `json:"crypto_key"`     // Путь к файлу с приватным ключом
	StoreInterval   time.Duration `json:"store_interval"` // интервал сохранения метрик на диск (0 - синхронная запись)
	Restore         bool          `json:"restore"`        // восстанавливать метрики из файла при старте
	TrustedSubnet   string        `json:"trusted_subnet"` // указывает доверенные сети
	GRPCAddress     string        `json:"grpc_address"`   // адрес для gRPC сервера
}

// AgentConfig содержит конфигурационные параметры агента.
type AgentConfig struct {
	Address        string        `json:"address"`         // Адрес сервера для подключения
	Key            string        `json:"key"`             // Ключ для генерации ХЕШ
	CryptoKey      string        `json:"crypto_key"`      //Путь к фалу с публичным ключом
	PollInterval   time.Duration `json:"poll_interval"`   // Интервал опроса метрик
	ReportInterval time.Duration `json:"report_interval"` // Интервал отправки метрик
	RateLimit      int
	UseGRPC        bool   `json:"use_grpc"`     // использовать gRPC вместо HTTP
	GRPCAddress    string `json:"grpc_address"` // адрес gRPC сервера
}

type CfgAgentENV struct {
	Address        string `env:"ADDRESS"`
	SecretKey      string `env:"KEY"`
	CryptoKey      string `env:"CRYPTO_KEY"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	RateLimit      int    `env:"RATE_LIMIT"`
	ConfigFile     string `env:"CONFIG"`
	UseGRPC        bool   `env:"USE_GRPC"`
	GRPCAddress    string `env:"GRPC_ADDRESS"`
}

type CfgServerENV struct {
	Address         string `env:"ADDRESS"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
	SecretKey       string `env:"KEY"`
	CryptoKey       string `env:"CRYPTO_KEY"`
	StoreInterval   int    `env:"STORE_INTERVAL"`
	Restore         bool   `env:"RESTORE"`
	ConfigFile      string `env:"CONFIG"`
	TrustedSubnet   string `env:"TRUSTED_SUBNET"`
	GRPCAddress     string `env:"GRPC_ADDRESS"`
}

func ensureHTTP(address string) string {
	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		return "http://" + address
	}
	return address
}

func (cfg *CfgAgentENV) ApplyFlags(mapFlags map[string]any) AgentConfig {
	if err := env.Parse(cfg); err != nil {
		logger.Fatal("failed to parse env vars:", err)
	}

	// Загружаем конфигурацию из файла, если указан
	var fileConfig AgentConfig
	if cfg.ConfigFile != "" {
		var err error
		fileConfig, err = LoadAgentConfigFromFile(cfg.ConfigFile)
		if err != nil {
			logger.Warnf("warning: failed to load config from file: %v", err)
		}
	}

	flagKey := getStringFromMap(mapFlags, "flagKey")
	flagRunAddr := getStringFromMap(mapFlags, "flagRunAddr")
	flagReportInterval := getIntFromMap(mapFlags, "flagReportInterval")
	flagPollInterval := getIntFromMap(mapFlags, "flagPollInterval")
	flagRateLimit := getIntFromMap(mapFlags, "flagRateLimit")
	flagCryptoKey := getStringFromMap(mapFlags, "flagCryptoKey")
	flagGRPCAddr := getStringFromMap(mapFlags, "flagGRPCAddr")
	flagUseGRPC := getBoolFromMap(mapFlags, "flagUseGRPC")

	key := firstNonEmpty(flagKey, cfg.SecretKey, fileConfig.Key)

	useGRPC := flagUseGRPC
	if !useGRPC {
		useGRPC = cfg.UseGRPC
	}
	if !useGRPC {
		useGRPC = fileConfig.UseGRPC
	}

	var address string
	if useGRPC {
		address = firstNonEmpty(flagGRPCAddr, cfg.GRPCAddress, fileConfig.GRPCAddress)
		if address == "" {
			address = "localhost:8090"
		}
	} else {
		address = firstNonEmpty(flagRunAddr, cfg.Address, fileConfig.Address)
		address = ensureHTTP(address)
	}

	reportInterval := firstNonZeroDuration(
		fileConfig.ReportInterval,
		time.Duration(cfg.ReportInterval)*time.Second,
		time.Duration(flagReportInterval)*time.Second,
	)

	pollInterval := firstNonZeroDuration(
		fileConfig.PollInterval,
		time.Duration(cfg.PollInterval)*time.Second,
		time.Duration(flagPollInterval)*time.Second,
	)

	rateLimit := firstNonZero(
		fileConfig.RateLimit,
		cfg.RateLimit,
		flagRateLimit,
	)

	if rateLimit <= 0 {
		rateLimit = 1
	}

	cryptoKey := firstNonEmpty(
		flagCryptoKey,
		cfg.CryptoKey,
		fileConfig.CryptoKey,
	)

	grpcAddress := firstNonEmpty(
		flagGRPCAddr,
		cfg.GRPCAddress,
		fileConfig.GRPCAddress,
	)

	agentCfg := AgentConfig{
		Address:        address,
		GRPCAddress:    grpcAddress,
		UseGRPC:        useGRPC,
		PollInterval:   pollInterval,
		ReportInterval: reportInterval,
		Key:            key,
		CryptoKey:      cryptoKey,
		RateLimit:      rateLimit,
	}
	return agentCfg
}

func (conf *CfgServerENV) ApplyFlags(mapFlags map[string]any) ServerConfig {
	if err := env.Parse(conf); err != nil {
		logger.Fatal("failed to parse env vars:", err)
	}

	var fileConfig ServerConfig
	if conf.ConfigFile != "" {
		var err error
		fileConfig, err = LoadServerConfigFromFile(conf.ConfigFile)
		if err != nil {
			logger.Warnf("warning: failed to load config from file: %v", err)
		}
	}

	flagKey := getStringFromMap(mapFlags, "flagKey")
	flagRunAddr := getStringFromMap(mapFlags, "flagRunAddr")
	flagStoreInterval := getIntFromMap(mapFlags, "flagStoreInterval")
	flagFileStoragePath := getStringFromMap(mapFlags, "flagFileStoragePath")
	flagRestore := getBoolFromMap(mapFlags, "flagRestore")
	flagDatabaseDSN := getStringFromMap(mapFlags, "flagDatabaseDSN")
	flagCryptoKey := getStringFromMap(mapFlags, "flagCryptoKey")
	flagTrustedSubnet := getStringFromMap(mapFlags, "flagTrustedSubnet")
	flagGRPCAddr := getStringFromMap(mapFlags, "flagGRPCAddr")

	key := firstNonEmpty(flagKey, conf.SecretKey, fileConfig.Key)
	serverAddress := firstNonEmpty(flagRunAddr, conf.Address, fileConfig.Address)
	grpcAddress := firstNonEmpty(flagGRPCAddr, conf.GRPCAddress, fileConfig.GRPCAddress)

	storeInterval := firstNonZeroDuration(
		fileConfig.StoreInterval,
		time.Duration(conf.StoreInterval)*time.Second,
		time.Duration(flagStoreInterval)*time.Second,
	)

	fileStoragePath := firstNonEmpty(
		flagFileStoragePath,
		conf.FileStoragePath,
		fileConfig.FileStoragePath,
	)

	restore := flagRestore
	if !restore {
		restore = conf.Restore
	}
	if !restore {
		restore = fileConfig.Restore
	}

	dbDSN := firstNonEmpty(
		flagDatabaseDSN,
		conf.DatabaseDSN,
		fileConfig.DatabaseDSN,
	)

	cryptoKey := firstNonEmpty(
		flagCryptoKey,
		conf.CryptoKey,
		fileConfig.CryptoKey,
	)

	trustedSubnet := firstNonEmpty(
		flagTrustedSubnet, conf.TrustedSubnet, fileConfig.TrustedSubnet)

	if serverAddress != "" && !strings.Contains(serverAddress, ":") {
		serverAddress = ":" + serverAddress
	}
	if grpcAddress == "" {
		grpcAddress = ":8090"
	} else if !strings.Contains(grpcAddress, ":") {
		grpcAddress = ":" + grpcAddress
	}

	cfg := ServerConfig{
		Address:         serverAddress,
		StoreInterval:   storeInterval,
		FileStoragePath: fileStoragePath,
		Restore:         restore,
		DatabaseDSN:     dbDSN,
		Key:             key,
		CryptoKey:       cryptoKey,
		TrustedSubnet:   trustedSubnet,
		GRPCAddress:     grpcAddress,
	}
	return cfg
}

// LoadServerConfigFromFile загружает конфигурацию сервера из JSON файла
func LoadServerConfigFromFile(filename string) (ServerConfig, error) {
	var config ServerConfig

	if filename == "" {
		return config, nil
	}

	absConfigPath, err := filepath.Abs(filename)
	if err != nil {
		return config, fmt.Errorf("failed to get absolute path for config: %w", err)
	}
	configDir := filepath.Dir(absConfigPath)

	data, err := os.ReadFile(filename)
	if err != nil {
		return config, fmt.Errorf("failed to read config file: %w", err)
	}

	type tempServerConfig struct {
		Address         string `json:"address"`
		GRPCAddress     string `json:"grpc_address"`
		Restore         bool   `json:"restore"`
		StoreInterval   string `json:"store_interval"`
		FileStoragePath string `json:"store_file"`
		DatabaseDSN     string `json:"database_dsn"`
		Key             string `json:"key"`
		CryptoKey       string `json:"crypto_key"`
		TrustedSubnet   string `json:"trusted_subnet"`
	}

	var temp tempServerConfig
	if err := json.Unmarshal(data, &temp); err != nil {
		return config, fmt.Errorf("failed to parse config file: %w", err)
	}

	var storeInterval time.Duration
	if temp.StoreInterval != "" {
		storeInterval, err = time.ParseDuration(temp.StoreInterval)
		if err != nil {
			return config, fmt.Errorf("invalid store_interval format: %w", err)
		}
	}

	fileStoragePath := resolveRelativePath(temp.FileStoragePath, configDir)
	cryptoKey := resolveRelativePath(temp.CryptoKey, configDir)

	config.Address = temp.Address
	config.GRPCAddress = temp.GRPCAddress
	config.Restore = temp.Restore
	config.StoreInterval = storeInterval
	config.FileStoragePath = fileStoragePath
	config.DatabaseDSN = temp.DatabaseDSN
	config.Key = temp.Key
	config.CryptoKey = cryptoKey
	config.TrustedSubnet = temp.TrustedSubnet

	return config, nil
}

// LoadAgentConfigFromFile загружает конфигурацию агента из JSON файла
func LoadAgentConfigFromFile(filename string) (AgentConfig, error) {
	var config AgentConfig

	if filename == "" {
		return config, nil
	}

	absConfigPath, err := filepath.Abs(filename)
	if err != nil {
		return config, fmt.Errorf("failed to get absolute path for config: %w", err)
	}
	configDir := filepath.Dir(absConfigPath)

	data, err := os.ReadFile(filename)
	if err != nil {
		return config, fmt.Errorf("failed to read config file: %w", err)
	}

	type tempAgentConfig struct {
		Address        string `json:"address"`
		GRPCAddress    string `json:"grpc_address"`
		UseGRPC        bool   `json:"use_grpc"`
		ReportInterval string `json:"report_interval"`
		PollInterval   string `json:"poll_interval"`
		Key            string `json:"key"`
		CryptoKey      string `json:"crypto_key"`
		RateLimit      int    `json:"rate_limit"`
	}

	var temp tempAgentConfig
	if err := json.Unmarshal(data, &temp); err != nil {
		return config, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Парсим интервалы времени
	var reportInterval, pollInterval time.Duration
	if temp.ReportInterval != "" {
		reportInterval, err = time.ParseDuration(temp.ReportInterval)
		if err != nil {
			return config, fmt.Errorf("invalid report_interval format: %w", err)
		}
	}
	if temp.PollInterval != "" {
		pollInterval, err = time.ParseDuration(temp.PollInterval)
		if err != nil {
			return config, fmt.Errorf("invalid poll_interval format: %w", err)
		}
	}

	cryptoKey := resolveRelativePath(temp.CryptoKey, configDir)

	config.Address = temp.Address
	config.GRPCAddress = temp.GRPCAddress
	config.UseGRPC = temp.UseGRPC
	config.ReportInterval = reportInterval
	config.PollInterval = pollInterval
	config.Key = temp.Key
	config.CryptoKey = cryptoKey
	config.RateLimit = temp.RateLimit

	return config, nil
}

// firstNonEmpty возвращает первую непустую строку из переданных
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// firstNonZero возвращает первое ненулевое число из переданных
func firstNonZero(values ...int) int {
	for _, v := range values {
		if v != 0 {
			return v
		}
	}
	return 0
}

// firstNonZeroDuration возвращает первый ненулевой интервал из переданных
func firstNonZeroDuration(values ...time.Duration) time.Duration {
	for _, v := range values {
		if v != 0 {
			return v
		}
	}
	return 0
}

// getStringFromMap безопасно извлекает строку из мапы
func getStringFromMap(m map[string]any, key string) string {
	if val, exists := m[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// getIntFromMap безопасно извлекает int из мапы
func getIntFromMap(m map[string]any, key string) int {
	if val, exists := m[key]; exists {
		if i, ok := val.(int); ok {
			return i
		}
	}
	return 0
}

// getBoolFromMap безопасно извлекает bool из мапы
func getBoolFromMap(m map[string]any, key string) bool {
	if val, exists := m[key]; exists {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// resolveRelativePath разрешает относительный путь относительно директории конфига
func resolveRelativePath(path, configDir string) string {
	if path == "" {
		return ""
	}

	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(configDir, path)
}
