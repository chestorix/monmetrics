// Package config содержит конфигурационные структуры для сервера и агента.
package config

import "time"

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
