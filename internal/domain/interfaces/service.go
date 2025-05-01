package interfaces

import "github.com/chestorix/monmetrics/internal/metrics"

type Service interface {
	UpdateGauge(name string, value float64) error
	UpdateCounter(name string, value int64) error
	GetGauge(name string) (float64, error)
	GetCounter(name string) (int64, error)
	GetAll() ([]metrics.Metric, error)
}
