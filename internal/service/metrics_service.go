package service

import "github.com/chestorix/monmetrics/internal/models"

type MetricsService interface {
	UpdateGauge(name string, value float64) error
	UpdateCounter(name string, value int64) error
	GetGauge(name string) (float64, error)
	GetCounter(name string) (int64, error)
	GetAllMetrics() ([]models.Metric, error)
}
