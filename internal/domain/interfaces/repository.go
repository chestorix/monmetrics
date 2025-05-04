package interfaces

import "github.com/chestorix/monmetrics/internal/metrics"

type Repository interface {
	UpdateGauge(name string, value float64)
	UpdateCounter(name string, value int64)
	GetGauge(name string) (float64, bool)
	GetCounter(name string) (int64, bool)
	GetAll() ([]models.Metric, error)
}
