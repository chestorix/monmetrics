// Package repository - реализация хранилища.
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/chestorix/monmetrics/internal/domain/interfaces"
	models "github.com/chestorix/monmetrics/internal/metrics"
)

type MemStorage struct {
	Gauges   map[string]float64
	Counters map[string]int64
	filePath string
	mu       sync.RWMutex
}

func NewMemStorage(filePath string) interfaces.Repository {
	return &MemStorage{
		Gauges:   make(map[string]float64),
		Counters: make(map[string]int64),
		filePath: filePath,
	}
}

func (m *MemStorage) Load(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

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

func (m *MemStorage) Save(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
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

func (m *MemStorage) UpdateGauge(ctx context.Context, name string, value float64) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	m.mu.Lock()
	m.Gauges[name] = value
	m.mu.Unlock()
	return nil
}
func (m *MemStorage) UpdateCounter(ctx context.Context, name string, value int64) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	m.mu.Lock()
	m.Counters[name] += value
	m.mu.Unlock()
	return nil
}

func (m *MemStorage) GetGauge(ctx context.Context, name string) (float64, bool, error) {
	select {
	case <-ctx.Done():
		return 0, false, ctx.Err()
	default:
	}
	if value, ok := m.Gauges[name]; ok {
		return value, true, nil
	}
	return 0, false, nil
}

func (m *MemStorage) GetCounter(ctx context.Context, name string) (int64, bool, error) {
	select {
	case <-ctx.Done():
		return 0, false, ctx.Err()
	default:
	}
	if value, ok := m.Counters[name]; ok {
		return value, true, nil
	}
	return 0, false, nil
}

func (m *MemStorage) GetAll(ctx context.Context) ([]models.Metric, error) {

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
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

func (m *MemStorage) UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				return fmt.Errorf("gauge value is nil")
			}
			m.Gauges[metric.ID] = *metric.Value
		case models.Counter:
			if metric.Delta == nil {
				return fmt.Errorf("counter delta is nil")
			}
			m.Counters[metric.ID] += *metric.Delta
		}
	}
	return nil
}
