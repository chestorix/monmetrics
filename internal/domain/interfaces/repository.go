package interfaces

import "github.com/chestorix/monmetrics/internal/metrics"

type Repository interface {
	UpdateGauge(name string, value float64) error
	UpdateCounter(name string, value int64) error
	GetGauge(name string) (float64, bool, error)
	GetCounter(name string) (int64, bool, error)
	GetAll() ([]models.Metric, error)
	Save() error
	Load() error
}
