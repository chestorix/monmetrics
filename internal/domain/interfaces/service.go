package interfaces

import "github.com/chestorix/monmetrics/internal/metrics"

type Service interface {
	UpdateGauge(name string, value float64) error
	UpdateCounter(name string, value int64) error
	GetGauge(name string) (float64, error)
	GetCounter(name string) (int64, error)
	GetAll() ([]models.Metric, error)
	UpdateMetricJSON(metric models.Metrics) (models.Metrics, error)
	GetMetricJSON(metric models.Metrics) (models.Metrics, error)
}
