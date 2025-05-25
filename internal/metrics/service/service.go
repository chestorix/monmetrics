package service

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/chestorix/monmetrics/internal/domain/interfaces"
	"github.com/chestorix/monmetrics/internal/metrics"
	_ "github.com/jackc/pgx/v5/stdlib"
	"time"
)

type MetricsService struct {
	repo interfaces.Repository
}

func NewService(repo interfaces.Repository) *MetricsService {
	return &MetricsService{repo: repo}
}

func (s *MetricsService) UpdateGauge(name string, value float64) error {
	s.repo.UpdateGauge(name, value)
	return nil
}

func (s *MetricsService) UpdateCounter(name string, value int64) error {
	s.repo.UpdateCounter(name, value)
	return nil
}

func (s *MetricsService) GetGauge(name string) (float64, error) {
	value, exists, _ := s.repo.GetGauge(name)
	if !exists {
		return 0, models.ErrMetricNotFound
	}
	return value, nil
}

func (s *MetricsService) GetCounter(name string) (int64, error) {
	value, exists, _ := s.repo.GetCounter(name)
	if !exists {
		return 0, models.ErrMetricNotFound
	}
	return value, nil
}

func (s *MetricsService) GetAll() ([]models.Metric, error) {
	return s.repo.GetAll()
}

func (s *MetricsService) UpdateMetricJSON(metric models.Metrics) (models.Metrics, error) {
	switch metric.MType {
	case models.Gauge:
		if metric.Value == nil {
			return metric, models.ErrInvalidMetricType
		}
		s.repo.UpdateGauge(metric.ID, *metric.Value)
		return metric, nil
	case models.Counter:
		if metric.Delta == nil {
			return metric, models.ErrInvalidMetricType
		}
		s.repo.UpdateCounter(metric.ID, *metric.Delta)
		respValue, _, _ := s.repo.GetCounter(metric.ID)
		metric.Delta = &respValue
		return metric, nil
	default:
		return metric, models.ErrInvalidMetricType
	}
}

func (s *MetricsService) GetMetricJSON(metric models.Metrics) (models.Metrics, error) {
	switch metric.MType {
	case models.Gauge:
		respValue, err := s.GetGauge(metric.ID)
		if err != nil {
			return metric, err
		}
		metric.Value = &respValue
		return metric, nil
	case models.Counter:
		respValue, err := s.GetCounter(metric.ID)
		if err != nil {
			return metric, err
		}
		metric.Delta = &respValue
		return metric, nil
	default:
		return metric, models.ErrInvalidMetricType
	}
}
func (s *MetricsService) CheckDB(ps string) error {

	if ps == "" {
		return nil
	}

	db, err := sql.Open("pgx", ps)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

/*
	func (s *MetricsService) UpdateMetricsBatch(metrics []models.Metrics) error {
		for _, metric := range metrics {
			switch metric.MType {
			case models.Gauge:
				if metric.Value == nil {
					return models.ErrInvalidMetricType
				}
				s.repo.UpdateGauge(metric.ID, *metric.Value)
			case models.Counter:
				if metric.Delta == nil {
					return models.ErrInvalidMetricType
				}
				s.repo.UpdateCounter(metric.ID, *metric.Delta)
			default:
				return models.ErrInvalidMetricType
			}
		}
		return nil
	}
*/
func (s *MetricsService) UpdateMetricsBatch(metrics []models.Metrics) error {
	return s.repo.UpdateMetricsBatch(metrics)
}
