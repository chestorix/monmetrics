package service

import (
	"github.com/chestorix/monmetrics/internal/models"
	"github.com/chestorix/monmetrics/internal/storage/interfaces"
)

type MetricsServiceImpl struct {
	repo interfaces.MetricsRepository
}

func NewMetricsService(repo interfaces.MetricsRepository) *MetricsServiceImpl {
	return &MetricsServiceImpl{repo: repo}
}

func (s *MetricsServiceImpl) UpdateGauge(name string, value float64) error {
	s.repo.UpdateGauge(name, value)
	return nil
}

func (s *MetricsServiceImpl) UpdateCounter(name string, value int64) error {
	s.repo.UpdateCounter(name, value)
	return nil
}

func (s *MetricsServiceImpl) GetGauge(name string) (float64, error) {
	value, exists := s.repo.GetGauge(name)
	if !exists {
		return 0, models.ErrMetricNotFound
	}
	return value, nil
}

func (s *MetricsServiceImpl) GetCounter(name string) (int64, error) {
	value, exists := s.repo.GetCounter(name)
	if !exists {
		return 0, models.ErrMetricNotFound
	}
	return value, nil
}

func (s *MetricsServiceImpl) GetAllMetrics() ([]models.Metric, error) {
	return s.repo.GetAllMetrics()
}
