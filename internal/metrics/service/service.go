package service

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/chestorix/monmetrics/internal/domain/interfaces"
	"github.com/chestorix/monmetrics/internal/metrics"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type MetricsService struct {
	repo interfaces.Repository
}

func NewService(repo interfaces.Repository) *MetricsService {
	return &MetricsService{repo: repo}
}

func (s *MetricsService) UpdateGauge(ctx context.Context, name string, value float64) error {
	s.repo.UpdateGauge(ctx, name, value)
	return nil
}

func (s *MetricsService) UpdateCounter(ctx context.Context, name string, value int64) error {
	s.repo.UpdateCounter(ctx, name, value)
	return nil
}

func (s *MetricsService) GetGauge(ctx context.Context, name string) (float64, error) {
	value, exists, _ := s.repo.GetGauge(ctx, name)
	if !exists {
		return 0, models.ErrMetricNotFound
	}
	return value, nil
}

func (s *MetricsService) GetCounter(ctx context.Context, name string) (int64, error) {
	value, exists, _ := s.repo.GetCounter(ctx, name)
	if !exists {
		return 0, models.ErrMetricNotFound
	}
	return value, nil
}

func (s *MetricsService) GetAll(ctx context.Context) ([]models.Metric, error) {
	return s.repo.GetAll(ctx)
}

func (s *MetricsService) UpdateMetricJSON(ctx context.Context, metric models.Metrics) (models.Metrics, error) {
	switch metric.MType {
	case models.Gauge:
		if metric.Value == nil {
			return metric, models.ErrInvalidMetricType
		}
		s.repo.UpdateGauge(ctx, metric.ID, *metric.Value)
		return metric, nil
	case models.Counter:
		if metric.Delta == nil {
			return metric, models.ErrInvalidMetricType
		}
		s.repo.UpdateCounter(ctx, metric.ID, *metric.Delta)
		respValue, _, _ := s.repo.GetCounter(ctx, metric.ID)
		metric.Delta = &respValue
		return metric, nil
	default:
		return metric, models.ErrInvalidMetricType
	}
}

func (s *MetricsService) GetMetricJSON(ctx context.Context, metric models.Metrics) (models.Metrics, error) {
	switch metric.MType {
	case models.Gauge:
		respValue, err := s.GetGauge(ctx, metric.ID)
		if err != nil {
			return metric, err
		}
		metric.Value = &respValue
		return metric, nil
	case models.Counter:
		respValue, err := s.GetCounter(ctx, metric.ID)
		if err != nil {
			return metric, err
		}
		metric.Delta = &respValue
		return metric, nil
	default:
		return metric, models.ErrInvalidMetricType
	}
}
func (s *MetricsService) CheckDB(ctx context.Context, ps string) error {

	if ps == "" {
		return nil
	}

	db, err := sql.Open("pgx", ps)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	return nil
}

func (s *MetricsService) UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error {

	if err := s.repo.UpdateMetricsBatch(ctx, metrics); err != nil {
		return err
	}
	return nil
}
