// Package config содержит конфигурационные структуры для сервера и агента.
package config

import "time"

// ServerConfig содержит конфигурационные параметры сервера.

type ServerConfig struct {
	Address         string        // адрес и порт сервера (например: ":8080")
	StoreInterval   time.Duration // интервал сохранения метрик на диск (0 - синхронная запись)
	FileStoragePath string        // путь к файлу для хранения метрик
	Restore         bool          // восстанавливать метрики из файла при старте
	DatabaseDSN     string        // строка подключения к БД (если используется)
	Key             string        // секретный ключ для проверки хешей
}

// AgentConfig содержит конфигурационные параметры агента.

type AgentConfig struct {
	Address        string        // Адрес сервера для подключения
	PollInterval   time.Duration // Интервал опроса метрик
	ReportInterval time.Duration // Интервал отправки метрик
	Key            string        // Ключ для генерации ХЕШ
}
