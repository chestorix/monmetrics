package interfaces

import "github.com/chestorix/monmetrics/internal/models"

type MetricsRepository interface {
	UpdateGauge(name string, value float64)
	UpdateCounter(name string, value int64)
	GetGauge(name string) (float64, bool)
	GetCounter(name string) (int64, bool)
	GetAllMetrics() ([]models.Metric, error)
}
