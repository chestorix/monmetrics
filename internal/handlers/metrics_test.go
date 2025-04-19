package handlers

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type MockMetricsRepository struct {
	gaugeValues   map[string]float64
	counterValues map[string]int64
}

func NewMockMetricsRepository() *MockMetricsRepository {
	return &MockMetricsRepository{
		gaugeValues:   make(map[string]float64),
		counterValues: make(map[string]int64),
	}
}

func (m *MockMetricsRepository) UpdateGauge(name string, value float64) {
	m.gaugeValues[name] = value
}

func (m *MockMetricsRepository) UpdateCounter(name string, value int64) {
	m.counterValues[name] = value
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
				response:    "Invalid metric type\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "invalid gauge value",
			url:  "/update/gauge/somemetrics/invalid",
			want: want{
				code:        http.StatusBadRequest,
				response:    "Invalid value for gauge\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockRepo := NewMockMetricsRepository()
			handler := NewMetricsHandler(mockRepo)

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
