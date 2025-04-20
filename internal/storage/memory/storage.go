package memory

import (
	"github.com/chestorix/monmetrics/internal/models"
	"github.com/chestorix/monmetrics/internal/storage/interfaces"
	"log"
)

type MemStorage struct {
	Gauges   map[string]float64
	Counters map[string]int64
}

func NewMemStorage() interfaces.MetricsRepository {
	return &MemStorage{
		Gauges:   make(map[string]float64),
		Counters: make(map[string]int64),
	}
}
func (m *MemStorage) UpdateGauge(name string, value float64) {
	m.Gauges[name] = value
	log.Println("Gauge ", name, " updated", m.Gauges[name])

}
func (m *MemStorage) UpdateCounter(name string, value int64) {
	m.Counters[name] += value
	log.Println("Counter ", name, " updated", m.Counters[name])

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

func (m *MemStorage) GetAllMetrics() ([]models.Metric, error) {
	var metrics []models.Metric

	for name, value := range m.Gauges {
		metrics = append(metrics, models.Metric{
			Name:  name,
			Type:  "gauge",
			Value: value,
		})
	}

	for name, value := range m.Counters {
		metrics = append(metrics, models.Metric{
			Name:  name,
			Type:  "counter",
			Value: value,
		})
	}

	return metrics, nil
}
