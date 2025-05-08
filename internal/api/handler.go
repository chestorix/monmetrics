package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/chestorix/monmetrics/internal/domain/interfaces"
	"github.com/chestorix/monmetrics/internal/metrics"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type MetricsHandler struct {
	service interfaces.Service
}

func NewMetricsHandler(service interfaces.Service) *MetricsHandler {
	return &MetricsHandler{service: service}
}

func (h *MetricsHandler) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	log.Println("UpdateHandler start")

	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) < 4 {
		http.Error(w, "Invalid request", http.StatusNotFound)
		return
	}

	metricType := parts[1]
	metricName, metricValue := parts[2], parts[3]

	switch metricType {
	case models.Gauge:
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(w, "invalid value for gauge", http.StatusBadRequest)
			return
		}
		if err := h.service.UpdateGauge(metricName, value); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Gauge metric updated")

	case models.Counter:
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(w, "invalid value for counter", http.StatusBadRequest)
			return
		}
		if err := h.service.UpdateCounter(metricName, value); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Counter metric updated")

	default:
		http.Error(w, models.ErrInvalidMetricType.Error(), http.StatusBadRequest)
	}
}

func (h *MetricsHandler) GetValuesHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid request", http.StatusNotFound)
		return
	}

	metricType := parts[1]
	metricName := parts[2]

	switch metricType {
	case models.Gauge:
		value, err := h.service.GetGauge(metricName)
		if err != nil {
			if err == models.ErrMetricNotFound {
				http.Error(w, err.Error(), http.StatusNotFound)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, value)

	case models.Counter:
		value, err := h.service.GetCounter(metricName)
		if err != nil {
			if err == models.ErrMetricNotFound {
				http.Error(w, err.Error(), http.StatusNotFound)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, value)

	default:
		http.Error(w, models.ErrInvalidMetricType.Error(), http.StatusBadRequest)
	}
}

func (h *MetricsHandler) GetAllMetricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	metrics, err := h.service.GetAll()
	if err != nil {
		http.Error(w, "Failed to get metrics", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	html := `
    <!DOCTYPE html>
    <html>
    <head>
        <title>Metrics</title>
        <style>
            body { font-family: Arial, sans-serif; margin: 20px; }
            h1 { color: #333; }
            table { border-collapse: collapse; width: 100%; max-width: 800px; }
            th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
            th { background-color: #f2f2f2; }
            tr:nth-child(even) { background-color: #f9f9f9; }
        </style>
    </head>
    <body>
        <h1>Metrics</h1>
        <table>
            <tr>
                <th>Type</th>
                <th>Name</th>
                <th>Value</th>
            </tr>
    `

	for _, metric := range metrics {
		html += fmt.Sprintf(`
            <tr>
                <td>%s</td>
                <td>%s</td>
                <td>%v</td>
            </tr>
        `, metric.Type, metric.Name, metric.Value)
	}

	html += `
        </table>
    </body>
    </html>
    `
	w.Write([]byte(html))
}

func (h *MetricsHandler) UpdateJSONHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	log.Println("UpdateJSONHandler start")
	var metric models.Metrics
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if err = json.Unmarshal(buf.Bytes(), &metric); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}
	updateMetric, err := h.service.UpdateMetricJSON(metric)
	if err != nil {
		switch err {
		case models.ErrInvalidMetricType:
			http.Error(w, `{"error": "Invalid metric type"}`, http.StatusBadRequest)
		default:
			http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		}
		return
	}
	resp, err := json.Marshal(updateMetric)
	if err != nil {
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (h *MetricsHandler) ValueJSONHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var metric models.Metrics
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
	}
	if err = json.Unmarshal(buf.Bytes(), &metric); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}
	foundMetric, err := h.service.GetMetricJSON(metric)
	if err != nil {
		switch err {
		case models.ErrMetricNotFound:
			http.Error(w, `{"error": "Metric not found"}`, http.StatusNotFound)
		case models.ErrInvalidMetricType:
			http.Error(w, `{"error": "Invalid metric type"}`, http.StatusBadRequest)
		default:
			http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		}
		return
	}
	resp, err := json.Marshal(foundMetric)
	if err != nil {
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}
