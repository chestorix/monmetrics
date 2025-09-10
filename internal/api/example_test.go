package api_test

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chestorix/monmetrics/internal/api"
	models "github.com/chestorix/monmetrics/internal/metrics"
)

var testPrivateKey *rsa.PrivateKey

// mockService - полная заглушка для интерфейса interfaces.Service
type mockService struct {
	gauges   map[string]float64
	counters map[string]int64
	metrics  map[string]models.Metrics
	dbError  error
}

func newMockService() *mockService {
	return &mockService{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
		metrics:  make(map[string]models.Metrics),
	}
}

// Методы для работы с gauge
func (m *mockService) UpdateGauge(ctx context.Context, name string, value float64) error {
	m.gauges[name] = value
	return nil
}

func (m *mockService) GetGauge(ctx context.Context, name string) (float64, error) {
	val, ok := m.gauges[name]
	if !ok {
		return 0, models.ErrMetricNotFound
	}
	return val, nil
}

// Методы для работы с counter
func (m *mockService) UpdateCounter(ctx context.Context, name string, value int64) error {
	m.counters[name] += value
	return nil
}

func (m *mockService) GetCounter(ctx context.Context, name string) (int64, error) {
	val, ok := m.counters[name]
	if !ok {
		return 0, models.ErrMetricNotFound
	}
	return val, nil
}

// Методы для работы с JSON
func (m *mockService) UpdateMetricJSON(ctx context.Context, metric models.Metrics) (models.Metrics, error) {
	switch metric.MType {
	case models.Gauge:
		if metric.Value == nil {
			return models.Metrics{}, models.ErrInvalidMetricType
		}
		m.gauges[metric.ID] = *metric.Value
		// Создаем копию значения чтобы взять адрес
		value := m.gauges[metric.ID]
		return models.Metrics{
			ID:    metric.ID,
			MType: metric.MType,
			Value: &value,
		}, nil
	case models.Counter:
		if metric.Delta == nil {
			return models.Metrics{}, models.ErrInvalidMetricType
		}
		m.counters[metric.ID] += *metric.Delta
		// Создаем копию значения чтобы взять адрес
		delta := m.counters[metric.ID]
		return models.Metrics{
			ID:    metric.ID,
			MType: metric.MType,
			Delta: &delta,
		}, nil
	default:
		return models.Metrics{}, models.ErrInvalidMetricType
	}
}

func (m *mockService) GetMetricJSON(ctx context.Context, metric models.Metrics) (models.Metrics, error) {
	switch metric.MType {
	case models.Gauge:
		val, ok := m.gauges[metric.ID]
		if !ok {
			return models.Metrics{}, models.ErrMetricNotFound
		}
		value := val
		return models.Metrics{
			ID:    metric.ID,
			MType: metric.MType,
			Value: &value,
		}, nil
	case models.Counter:
		val, ok := m.counters[metric.ID]
		if !ok {
			return models.Metrics{}, models.ErrMetricNotFound
		}
		delta := val
		return models.Metrics{
			ID:    metric.ID,
			MType: metric.MType,
			Delta: &delta,
		}, nil
	default:
		return models.Metrics{}, models.ErrInvalidMetricType
	}
}

// Методы для работы с пачками метрик
func (m *mockService) UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	for _, metric := range metrics {
		_, err := m.UpdateMetricJSON(ctx, metric)
		if err != nil {
			return err
		}
	}
	return nil
}

// Методы для получения всех метрик
func (m *mockService) GetAll(ctx context.Context) ([]models.Metric, error) {
	var result []models.Metric

	for name, value := range m.gauges {
		result = append(result, models.Metric{
			Name:  name,
			Type:  models.Gauge,
			Value: value,
		})
	}

	for name, value := range m.counters {
		result = append(result, models.Metric{
			Name:  name,
			Type:  models.Counter,
			Value: float64(value),
		})
	}

	return result, nil
}

// Метод для проверки соединения с БД
func (m *mockService) CheckDB(ctx context.Context, dsn string) error {
	return m.dbError
}

// Примеры использования всех хендлеров
func ExampleMetricsHandler_UpdateHandler() {
	mock := newMockService()
	handler := api.NewMetricsHandler(mock, "", "", testPrivateKey, logrus.New())

	ts := httptest.NewServer(http.HandlerFunc(handler.UpdateHandler))
	defer ts.Close()

	res, err := http.Post(ts.URL+"/update/gauge/test_metric/123.45", "text/plain", nil)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return
	}
	defer res.Body.Close()

	fmt.Println(res.Status)
	// Output: 200 OK
}

func ExampleMetricsHandler_GetValuesHandler() {
	mock := newMockService()
	mock.gauges["test_metric"] = 123.45
	handler := api.NewMetricsHandler(mock, "", "", testPrivateKey, logrus.New())

	ts := httptest.NewServer(http.HandlerFunc(handler.GetValuesHandler))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/value/gauge/test_metric")
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return
	}
	defer res.Body.Close()

	var body bytes.Buffer
	if _, err := body.ReadFrom(res.Body); err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	fmt.Println(body.String())
	// Output: 123.45
}

func ExampleMetricsHandler_GetAllMetricsHandler() {
	mock := newMockService()
	mock.gauges["metric1"] = 1.23
	mock.counters["metric2"] = 42
	handler := api.NewMetricsHandler(mock, "", "", testPrivateKey, logrus.New())

	ts := httptest.NewServer(http.HandlerFunc(handler.GetAllMetricsHandler))
	defer ts.Close()

	res, err := http.Get(ts.URL)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return
	}
	defer res.Body.Close()

	fmt.Println(res.Status)
	// Output: 200 OK
}

func ExampleMetricsHandler_UpdateJSONHandler() {
	mock := newMockService()
	handler := api.NewMetricsHandler(mock, "", "", testPrivateKey, logrus.New())

	metric := models.Metrics{
		ID:    "test_metric",
		MType: models.Gauge,
		Value: func() *float64 { v := 123.45; return &v }(),
	}

	jsonData, err := json.Marshal(metric)
	if err != nil {
		fmt.Printf("Error marshaling metric: %v\n", err)
		return
	}

	ts := httptest.NewServer(http.HandlerFunc(handler.UpdateJSONHandler))
	defer ts.Close()

	res, err := http.Post(ts.URL+"/update/", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return
	}
	defer res.Body.Close()

	fmt.Println(res.Status)
	// Output: 200 OK
}

func ExampleMetricsHandler_ValueJSONHandler() {
	mock := newMockService()
	mock.gauges["test_metric"] = 123.45
	handler := api.NewMetricsHandler(mock, "", "", testPrivateKey, logrus.New())

	metric := models.Metrics{
		ID:    "test_metric",
		MType: models.Gauge,
	}

	jsonData, err := json.Marshal(metric)
	if err != nil {
		fmt.Printf("Error marshaling metric: %v\n", err)
		return
	}

	ts := httptest.NewServer(http.HandlerFunc(handler.ValueJSONHandler))
	defer ts.Close()

	res, err := http.Post(ts.URL+"/value/", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return
	}
	defer res.Body.Close()

	fmt.Println(res.Status)
	// Output: 200 OK
}

func ExampleMetricsHandler_PingHandler() {
	mock := newMockService()
	mock.dbError = fmt.Errorf("DB connection error")
	handler := api.NewMetricsHandler(mock, "test_dsn", "", testPrivateKey, logrus.New())

	ts := httptest.NewServer(http.HandlerFunc(handler.PingHandler))
	defer ts.Close()

	res, err := http.Get(ts.URL)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return
	}
	defer res.Body.Close()

	fmt.Println(res.Status)
	// Output: 500 Internal Server Error
}

func ExampleMetricsHandler_UpdatesHandler() {
	mock := newMockService()
	handler := api.NewMetricsHandler(mock, "", "", testPrivateKey, logrus.New())

	metrics := []models.Metrics{
		{
			ID:    "metric1",
			MType: models.Gauge,
			Value: func() *float64 { v := 1.23; return &v }(),
		},
		{
			ID:    "metric2",
			MType: models.Counter,
			Delta: func() *int64 { v := int64(42); return &v }(),
		},
	}

	jsonData, err := json.Marshal(metrics)
	if err != nil {
		fmt.Printf("Error marshaling metrics: %v\n", err)
		return
	}

	ts := httptest.NewServer(http.HandlerFunc(handler.UpdatesHandler))
	defer ts.Close()

	res, err := http.Post(ts.URL+"/updates/", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return
	}
	defer res.Body.Close()

	fmt.Println(res.Status)
	// Output: 200 OK
}

func TestExamples(t *testing.T) {
	// Запускаем все примеры для проверки
	ExampleMetricsHandler_UpdateHandler()
	ExampleMetricsHandler_GetValuesHandler()
	ExampleMetricsHandler_GetAllMetricsHandler()
	ExampleMetricsHandler_UpdateJSONHandler()
	ExampleMetricsHandler_ValueJSONHandler()
	ExampleMetricsHandler_PingHandler()
	ExampleMetricsHandler_UpdatesHandler()
}
