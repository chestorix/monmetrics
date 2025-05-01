package repository

import (
	"github.com/chestorix/monmetrics/internal/domain/interfaces"
	"github.com/chestorix/monmetrics/internal/metrics"
)

type MemStorage struct {
	Gauges   map[string]float64
	Counters map[string]int64
}

func NewMemStorage() interfaces.Repository {
	return &MemStorage{
		Gauges:   make(map[string]float64),
		Counters: make(map[string]int64),
	}
}
func (m *MemStorage) UpdateGauge(name string, value float64) {
	m.Gauges[name] = value

}
func (m *MemStorage) UpdateCounter(name string, value int64) {
	m.Counters[name] += value

}

func (m *MemStorage) GetGauge(name string) (float64, bool) {
	if value, ok := m.Gauges[name]; ok {
		return value, true
	}
	return 0, false
}

func (m *MemStorage) GetCounter(name string) (int64, bool) {
	if value, ok := m.Counters[name]; ok {
		return value, true
	}
	return 0, false
}

func (m *MemStorage) GetAll() ([]metrics.Metric, error) {
	var metric []metrics.Metric

	for name, value := range m.Gauges {
		metric = append(metric, metrics.Metric{
			Name:  name,
			Type:  metrics.Gauge,
			Value: value,
		})
	}

	for name, value := range m.Counters {
		metric = append(metric, metrics.Metric{
			Name:  name,
			Type:  metrics.Counter,
			Value: value,
		})
	}

	return metric, nil
}
