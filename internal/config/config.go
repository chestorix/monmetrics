package config

import "time"

// ServerConfig содержит конфигурационные параметры сервера.
type ServerConfig struct {
	Address         string        // Адрес сервера (поумоланию  ":8080")
	StoreInterval   time.Duration // Интервал сохранения данных на диск (0  синхронно )
	FileStoragePath string        // Путь до файла сохранения метрик
	Restore         bool          // Восстановление данных при запуске из хранилища
	DatabaseDSN     string        // Строка подклбчения к Базе данных
	Key             string        // Ключ для генерации ХЕШ
}

// AgentConfig содержит конфигурационные параметры агента.
type AgentConfig struct {
	Address        string        // Адрес сервера для подключения
	PollInterval   time.Duration // Интервал опроса метрик
	ReportInterval time.Duration // Интервал отправки метрик
	Key            string        // Ключ для генерации ХЕШ
}
