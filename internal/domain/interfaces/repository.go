package interfaces

import (
	"context"

	models "github.com/chestorix/monmetrics/internal/metrics"
)

// Repository определяет интерфейс для операций хранения метрик.
// Он предоставляет методы для хранения, извлечения данных метрик и управления ими.

type Repository interface {
	// UpdateGauge обновляет метрику Gauage с заданным именем и значением.
	UpdateGauge(ctx context.Context, name string, value float64) error
	// UpdateCounter обновляет или создает метрику Counter(счетчик) с указанным именем и значением.
	// Для счетчиков значение добавляется к существующему значению.
	UpdateCounter(ctx context.Context, name string, value int64) error
	// UpdateMetricsBatch обновляет несколько показателей за одну транзакцию.
	UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error
	// GetGauge извлекает метрику Gauage по имени.
	GetGauge(ctx context.Context, name string) (float64, bool, error)
	// getCounter извлекает метрику счетчика по имени.
	GetCounter(ctx context.Context, name string) (int64, bool, error)
	// GetAll извлекает все сохраненные метрики.
	GetAll(ctx context.Context) ([]models.Metric, error)
	// Save сохраняет текущее состояние метрик в хранилище(Файловое хранилище).
	Save(ctx context.Context) error
	// Load извлекает сохраненное состояние метрик из хранилища (Файловое хранилище).
	Load(ctx context.Context) error
	// Close выполняет очистку и закрывает все ресурсы, используемые хранилищем.
	Close() error
}
