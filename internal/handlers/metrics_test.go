package handlers

import (
	"errors"
	"github.com/chestorix/monmetrics/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type MockMetricsService struct {
	gaugeValues   map[string]float64
	counterValues map[string]int64
	getAllError   bool
}

func NewMockMetricsService() *MockMetricsService {
	return &MockMetricsService{
		gaugeValues:   make(map[string]float64),
		counterValues: make(map[string]int64),
	}
}

func (m *MockMetricsService) UpdateGauge(name string, value float64) error {
	m.gaugeValues[name] = value
	return nil
}

func (m *MockMetricsService) UpdateCounter(name string, value int64) error {
	m.counterValues[name] += value
	return nil
}

func (m *MockMetricsService) GetGauge(name string) (float64, error) {
	val, ok := m.gaugeValues[name]
	if !ok {
		return 0, models.ErrMetricNotFound
	}
	return val, nil
}

func (m *MockMetricsService) GetCounter(name string) (int64, error) {
	val, ok := m.counterValues[name]
	if !ok {
		return 0, models.ErrMetricNotFound
	}
	return val, nil
}

func (m *MockMetricsService) GetAllMetrics() ([]models.Metric, error) {
	if m.getAllError {
		return nil, errors.New("mock error")
	}

	var metrics []models.Metric
	for name, value := range m.gaugeValues {
		metrics = append(metrics, models.Metric{
			Name:  name,
			Type:  models.Gauge,
			Value: value,
		})
	}
	for name, value := range m.counterValues {
		metrics = append(metrics, models.Metric{
			Name:  name,
			Type:  models.Counter,
			Value: value,
		})
	}
	return metrics, nil
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
			handler := NewMetricsHandler(mockService)

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
				service.UpdateGauge("test_gauge", 123.45)
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
				service.UpdateCounter("test_counter", 42)
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
			handler := NewMetricsHandler(mockService)

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
				service.UpdateGauge("alloc", 123.45)
				service.UpdateCounter("requests", 42)
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
			handler := NewMetricsHandler(mockService)

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
