package interfaces

import (
	"context"
	"github.com/chestorix/monmetrics/internal/metrics"
)

type Repository interface {
	UpdateGauge(ctx context.Context, name string, value float64) error
	UpdateCounter(ctx context.Context, name string, value int64) error
	UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error
	GetGauge(ctx context.Context, name string) (float64, bool, error)
	GetCounter(ctx context.Context, name string) (int64, bool, error)
	GetAll(ctx context.Context) ([]models.Metric, error)
	Save(ctx context.Context) error
	Load(ctx context.Context) error
	Close() error
}
