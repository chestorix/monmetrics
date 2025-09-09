// Package interfaces -  определение интерфейсов приложения.
package interfaces

import (
	"context"

	models "github.com/chestorix/monmetrics/internal/metrics"
)

type Service interface {
	UpdateGauge(ctx context.Context, name string, value float64) error
	UpdateCounter(ctx context.Context, name string, value int64) error
	UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error
	GetGauge(ctx context.Context, name string) (float64, error)
	GetCounter(ctx context.Context, name string) (int64, error)
	GetAll(ctx context.Context) ([]models.Metric, error)
	UpdateMetricJSON(ctx context.Context, metric models.Metrics) (models.Metrics, error)
	GetMetricJSON(ctx context.Context, metric models.Metrics) (models.Metrics, error)
	CheckDB(ctx context.Context, ps string) error
}
