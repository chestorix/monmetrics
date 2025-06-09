package api

import (
	"context"
	"errors"
	"github.com/chestorix/monmetrics/internal/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type MockMetricsService struct {
	gaugeValues   map[string]float64
	counterValues map[string]int64
	getAllError   bool
	checkDBError  bool
	ctx           context.Context
}

func NewMockMetricsService() *MockMetricsService {
	return &MockMetricsService{
		gaugeValues:   make(map[string]float64),
		counterValues: make(map[string]int64),
	}
}

func (m *MockMetricsService) UpdateGauge(ctx context.Context, name string, value float64) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	m.gaugeValues[name] = value
	return nil
}

func (m *MockMetricsService) UpdateCounter(ctx context.Context, name string, value int64) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	m.counterValues[name] += value
	return nil
}

func (m *MockMetricsService) GetGauge(ctx context.Context, name string) (float64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}
	val, ok := m.gaugeValues[name]
	if !ok {
		return 0, models.ErrMetricNotFound
	}
	return val, nil
}

func (m *MockMetricsService) GetCounter(ctx context.Context, name string) (int64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}
	val, ok := m.counterValues[name]
	if !ok {
		return 0, models.ErrMetricNotFound
	}
	return val, nil
}

func (m *MockMetricsService) GetAll(ctx context.Context) ([]models.Metric, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if m.getAllError {
		return nil, errors.New("mock error")
	}

	var metric []models.Metric
	for name, value := range m.gaugeValues {
		metric = append(metric, models.Metric{
			Name:  name,
			Type:  models.Gauge,
			Value: value,
		})
	}
	for name, value := range m.counterValues {
		metric = append(metric, models.Metric{
			Name:  name,
			Type:  models.Counter,
			Value: value,
		})
	}
	return metric, nil
}

func (m *MockMetricsService) UpdateMetricJSON(ctx context.Context, metric models.Metrics) (models.Metrics, error) {
	select {
	case <-ctx.Done():
		return models.Metrics{}, ctx.Err()
	default:
	}
	switch metric.MType {
	case models.Gauge:
		if metric.Value == nil {
			return metric, models.ErrInvalidMetricType
		}
		m.gaugeValues[metric.ID] = *metric.Value
		return models.Metrics{
			ID:    metric.ID,
			MType: metric.MType,
			Value: metric.Value,
		}, nil
	case models.Counter:
		if metric.Delta == nil {
			return metric, models.ErrInvalidMetricType
		}
		m.counterValues[metric.ID] += *metric.Delta
		updatedValue := m.counterValues[metric.ID]
		return models.Metrics{
			ID:    metric.ID,
			MType: metric.MType,
			Delta: &updatedValue,
		}, nil
	default:
		return metric, models.ErrInvalidMetricType
	}
}

func (m *MockMetricsService) GetMetricJSON(ctx context.Context, metric models.Metrics) (models.Metrics, error) {
	select {
	case <-ctx.Done():
		return models.Metrics{}, ctx.Err()
	default:
	}

	switch metric.MType {
	case models.Gauge:
		val, ok := m.gaugeValues[metric.ID]
		if !ok {
			return metric, models.ErrMetricNotFound
		}

		valueCopy := val
		return models.Metrics{
			ID:    metric.ID,
			MType: metric.MType,
			Value: &valueCopy,
		}, nil

	case models.Counter:
		val, ok := m.counterValues[metric.ID]
		if !ok {
			return metric, models.ErrMetricNotFound
		}

		deltaCopy := val
		return models.Metrics{
			ID:    metric.ID,
			MType: metric.MType,
			Delta: &deltaCopy,
		}, nil

	default:
		return metric, models.ErrInvalidMetricType
	}
}
func (m *MockMetricsService) CheckDB(ctx context.Context, ps string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if ps == "" {
		return nil // Пустой DSN - считаем что БД не используется
	}
	if m.checkDBError {
		return errors.New("mock database error")
	}
	return nil
}

func (m *MockMetricsService) UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	return nil
}

func TestMetricsHandler_UpdateHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name string
		url  string
		want want
	}{
		{
			name: "positive test send gauge",
			url:  "/update/gauge/somemetrics/127.0",
			want: want{
				code:        http.StatusOK,
				response:    "Gauge metric updated\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "positive test send counter",
			url:  "/update/counter/somemetrics/42",
			want: want{
				code:        http.StatusOK,
				response:    "Counter metric updated\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "invalid metric type",
			url:  "/update/invalid/somemetrics/123",
			want: want{
				code:        http.StatusBadRequest,
				response:    "invalid metric type\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "invalid gauge value",
			url:  "/update/gauge/somemetrics/invalid",
			want: want{
				code:        http.StatusBadRequest,
				response:    "invalid value for gauge\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockService := NewMockMetricsService()
			handler := NewMetricsHandler(mockService, "")

			request := httptest.NewRequest(http.MethodPost, test.url, nil)
			w := httptest.NewRecorder()

			handler.UpdateHandler(w, request)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, test.want.code, resp.StatusCode)

			resBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, test.want.response, string(resBody))

			assert.Equal(t, test.want.contentType, resp.Header.Get("Content-Type"))
		})
	}
}

func TestMetricsHandler_GetValuesHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	type want struct {
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name    string
		url     string
		prepare func(service *MockMetricsService)
		want    want
	}{
		{
			name: "get existing gauge",
			url:  "/value/gauge/test_gauge",
			prepare: func(service *MockMetricsService) {
				service.UpdateGauge(ctx, "test_gauge", 123.45)
			},
			want: want{
				code:        http.StatusOK,
				response:    "123.45",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:    "get non-existing gauge",
			url:     "/value/gauge/non_existing",
			prepare: func(service *MockMetricsService) {},
			want: want{
				code:        http.StatusNotFound,
				response:    "metric not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "get existing counter",
			url:  "/value/counter/test_counter",
			prepare: func(service *MockMetricsService) {
				service.UpdateCounter(ctx, "test_counter", 42)
			},
			want: want{
				code:        http.StatusOK,
				response:    "42",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "invalid metric type",
			url:  "/value/invalid/test",
			want: want{
				code:        http.StatusBadRequest,
				response:    "invalid metric type\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockService := NewMockMetricsService()
			if test.prepare != nil {
				test.prepare(mockService)
			}
			handler := NewMetricsHandler(mockService, "")

			request := httptest.NewRequest(http.MethodGet, test.url, nil)
			w := httptest.NewRecorder()

			handler.GetValuesHandler(w, request)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, test.want.code, resp.StatusCode)

			resBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, test.want.response, string(resBody))

			assert.Equal(t, test.want.contentType, resp.Header.Get("Content-Type"))
		})
	}
}

func TestMetricsHandler_GetAllMetricsHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	tests := []struct {
		name       string
		prepare    func(service *MockMetricsService)
		wantCode   int
		wantBody   string
		wantInBody []string
	}{
		{
			name: "success with metrics",
			prepare: func(service *MockMetricsService) {
				service.UpdateGauge(ctx, "alloc", 123.45)
				service.UpdateCounter(ctx, "requests", 42)
			},
			wantCode: http.StatusOK,
			wantInBody: []string{
				"<td>gauge</td>",
				"<td>alloc</td>",
				"<td>123.45</td>",
				"<td>counter</td>",
				"<td>requests</td>",
				"<td>42</td>",
			},
		},
		{
			name:     "empty metrics",
			prepare:  func(service *MockMetricsService) {},
			wantCode: http.StatusOK,
			wantInBody: []string{
				"<h1>Metrics</h1>",
				"<table>",
				"</table>",
			},
		},
		{
			name: "service error",
			prepare: func(service *MockMetricsService) {
				service.getAllError = true
			},
			wantCode: http.StatusInternalServerError,
			wantBody: "Failed to get metrics\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockService := NewMockMetricsService()
			if test.prepare != nil {
				test.prepare(mockService)
			}
			handler := NewMetricsHandler(mockService, "")

			request := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()

			handler.GetAllMetricsHandler(w, request)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, test.wantCode, resp.StatusCode)

			resBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			bodyStr := string(resBody)

			if test.wantBody != "" {
				assert.Equal(t, test.wantBody, bodyStr)
			}

			for _, substr := range test.wantInBody {
				assert.Contains(t, bodyStr, substr)
			}
		})
	}
}

func TestMetricsHandler_UpdateJSONHandler(t *testing.T) {
	tests := []struct {
		name         string
		payload      string
		wantStatus   int
		wantResponse string
	}{
		{
			name:         "update gauge",
			payload:      `{"id":"temperature","type":"gauge","value":23.5}`,
			wantStatus:   http.StatusOK,
			wantResponse: `{"id":"temperature","type":"gauge","value":23.5}`,
		},
		{
			name:         "update counter",
			payload:      `{"id":"requests","type":"counter","delta":1}`,
			wantStatus:   http.StatusOK,
			wantResponse: `{"id":"requests","type":"counter","delta":1}`,
		},
		{
			name:         "invalid type",
			payload:      `{"id":"test","type":"invalid","value":1}`,
			wantStatus:   http.StatusBadRequest,
			wantResponse: `{"error":"Invalid metric type"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := NewMockMetricsService()
			handler := NewMetricsHandler(mockService, "")

			req := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(tt.payload))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.UpdateJSONHandler(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			if tt.wantResponse != "" {
				body, _ := io.ReadAll(resp.Body)
				assert.JSONEq(t, tt.wantResponse, string(body))
			}
		})
	}
}

func TestMetricsHandler_ValueJSONHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	tests := []struct {
		name         string
		prepare      func(*MockMetricsService)
		payload      string
		wantStatus   int
		wantResponse string
	}{
		{
			name: "get existing gauge",
			prepare: func(s *MockMetricsService) {
				s.UpdateGauge(ctx, "temperature", 23.5)
			},
			payload:      `{"id":"temperature","type":"gauge"}`,
			wantStatus:   http.StatusOK,
			wantResponse: `{"id":"temperature","type":"gauge","value":23.5}`,
		},
		{
			name:         "get non-existing gauge",
			prepare:      func(s *MockMetricsService) {},
			payload:      `{"id":"nonexistent","type":"gauge"}`,
			wantStatus:   http.StatusNotFound,
			wantResponse: `{"error":"Metric not found"}`,
		},
		{
			name:         "invalid type",
			payload:      `{"id":"test","type":"invalid"}`,
			wantStatus:   http.StatusBadRequest,
			wantResponse: `{"error":"Invalid metric type"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := NewMockMetricsService()
			if tt.prepare != nil {
				tt.prepare(mockService)
			}
			handler := NewMetricsHandler(mockService, "")

			req := httptest.NewRequest(http.MethodPost, "/value/", strings.NewReader(tt.payload))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ValueJSONHandler(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			if tt.wantResponse != "" {
				body, _ := io.ReadAll(resp.Body)
				assert.JSONEq(t, tt.wantResponse, string(body))
			}
		})
	}
}

func TestMetricsHandler_PingHandler(t *testing.T) {
	tests := []struct {
		name      string
		dbDNS     string
		mockError bool
		wantCode  int
	}{
		{
			name:     "successful connection",
			dbDNS:    "valid-dsn",
			wantCode: http.StatusOK,
		},
		{
			name:      "connection error",
			dbDNS:     "invalid-dsn",
			mockError: true,
			wantCode:  http.StatusInternalServerError,
		},
		{
			name:     "no database configured",
			dbDNS:    "",
			wantCode: http.StatusOK, // Пустой DSN - БД не используется
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := NewMockMetricsService()
			mockService.checkDBError = tt.mockError

			handler := NewMetricsHandler(mockService, tt.dbDNS)

			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			w := httptest.NewRecorder()

			handler.PingHandler(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}
