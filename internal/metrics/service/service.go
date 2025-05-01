package service

import (
	"github.com/chestorix/monmetrics/internal/domain/interfaces"
	"github.com/chestorix/monmetrics/internal/metrics"
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
	value, exists := s.repo.GetGauge(name)
	if !exists {
		return 0, metrics.ErrMetricNotFound
	}
	return value, nil
}

func (s *MetricsService) GetCounter(name string) (int64, error) {
	value, exists := s.repo.GetCounter(name)
	if !exists {
		return 0, metrics.ErrMetricNotFound
	}
	return value, nil
}

func (s *MetricsService) GetAll() ([]metrics.Metric, error) {
	return s.repo.GetAll()
}
