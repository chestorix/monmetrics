package repository

import (
	"encoding/json"
	"github.com/chestorix/monmetrics/internal/domain/interfaces"
	"github.com/chestorix/monmetrics/internal/metrics"
	"os"
	"sync"
)

type MemStorage struct {
	Gauges   map[string]float64
	Counters map[string]int64
	mu       sync.RWMutex
	filePath string
}

func NewMemStorage(filePath string) interfaces.Repository {
	return &MemStorage{
		Gauges:   make(map[string]float64),
		Counters: make(map[string]int64),
		filePath: filePath,
	}
}

func (m *MemStorage) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	file, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var data struct {
		Gauges   map[string]float64 `json:"gauges"`
		Counters map[string]int64   `json:"counters"`
	}
	if err := json.Unmarshal(file, &data); err != nil {
		return err
	}
	m.Gauges = data.Gauges
	m.Counters = data.Counters
	return nil
}

func (m *MemStorage) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data := struct {
		Gauges   map[string]float64 `json:"gauges"`
		Counters map[string]int64   `json:"counters"`
	}{
		Gauges:   m.Gauges,
		Counters: m.Counters,
	}

	file, err := json.Marshal(data)
	if err != nil {
		return err
	}

	tmpFile := m.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, file, 0644); err != nil {
		return err
	}

	return os.Rename(tmpFile, m.filePath)
}

func (m *MemStorage) UpdateGauge(name string, value float64) error {
	m.mu.Lock()
	m.Gauges[name] = value
	m.mu.Unlock()
	return nil
}
func (m *MemStorage) UpdateCounter(name string, value int64) error {
	m.mu.Lock()
	m.Counters[name] += value
	m.mu.Unlock()
	return nil
}

func (m *MemStorage) GetGauge(name string) (float64, bool, error) {
	if value, ok := m.Gauges[name]; ok {
		return value, true, nil
	}
	return 0, false, nil
}

func (m *MemStorage) GetCounter(name string) (int64, bool, error) {
	if value, ok := m.Counters[name]; ok {
		return value, true, nil
	}
	return 0, false, nil
}

func (m *MemStorage) GetAll() ([]models.Metric, error) {
	var metric []models.Metric

	for name, value := range m.Gauges {
		metric = append(metric, models.Metric{
			Name:  name,
			Type:  models.Gauge,
			Value: value,
		})
	}

	for name, value := range m.Counters {
		metric = append(metric, models.Metric{
			Name:  name,
			Type:  models.Counter,
			Value: value,
		})
	}

	return metric, nil
}
func (m *MemStorage) Close() error {
	return nil
}
