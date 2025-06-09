package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/chestorix/monmetrics/internal/domain/interfaces"
	"github.com/chestorix/monmetrics/internal/metrics"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type MetricsHandler struct {
	service interfaces.Service
	dbDNS   string
}
type jsonError struct {
	Error string `json:"error"`
}

func NewMetricsHandler(service interfaces.Service, dbDNS string) *MetricsHandler {
	return &MetricsHandler{service: service,
		dbDNS: dbDNS,
	}
}

func (h *MetricsHandler) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*2)
	defer cancel()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) < 4 {
		http.Error(w, "Invalid request", http.StatusBadRequest)
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
		if err := h.service.UpdateGauge(ctx, metricName, value); err != nil {
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
		if err := h.service.UpdateCounter(ctx, metricName, value); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, models.ErrInvalidMetricType.Error(), http.StatusBadRequest)
	}
}

func (h *MetricsHandler) GetValuesHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*2)
	defer cancel()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
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
		value, err := h.service.GetGauge(ctx, metricName)
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
		value, err := h.service.GetCounter(ctx, metricName)
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
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*2)
	defer cancel()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics, err := h.service.GetAll(ctx)
	if err != nil {
		http.Error(w, "Failed to get metrics", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	html := generateMetricsHTML(metrics)
	w.Write([]byte(html))
}

func (h *MetricsHandler) UpdateJSONHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*2)
	defer cancel()
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	var metric models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		renderError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	updateMetric, err := h.service.UpdateMetricJSON(ctx, metric)
	if err != nil {
		switch err {
		case models.ErrInvalidMetricType:
			renderError(w, "Invalid metric type", http.StatusBadRequest)
		default:
			renderError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(updateMetric); err != nil {
		renderError(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *MetricsHandler) ValueJSONHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*2)
	defer cancel()
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		renderError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var metric models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		renderError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	foundMetric, err := h.service.GetMetricJSON(ctx, metric)
	if err != nil {
		switch err {
		case models.ErrMetricNotFound:
			renderError(w, "Metric not found", http.StatusNotFound)
		case models.ErrInvalidMetricType:
			renderError(w, "Invalid metric type", http.StatusBadRequest)
		default:
			renderError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(foundMetric); err != nil {
		renderError(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *MetricsHandler) PingHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*2)
	defer cancel()
	if h.dbDNS == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := h.service.CheckDB(ctx, h.dbDNS); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
func (h *MetricsHandler) UpdatesHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*2)
	defer cancel()
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		renderError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var metrics []models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		renderError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(metrics) == 0 {
		renderError(w, "Empty batch", http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateMetricsBatch(ctx, metrics); err != nil {

		renderError(w, fmt.Sprintf("Failed to update metrics: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
func generateMetricsHTML(metrics []models.Metric) string {
	var htmlBuilder strings.Builder

	htmlBuilder.WriteString(`
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
    `)

	for _, metric := range metrics {
		htmlBuilder.WriteString(fmt.Sprintf(`
            <tr>
                <td>%s</td>
                <td>%s</td>
                <td>%v</td>
            </tr>
        `, metric.Type, metric.Name, metric.Value))
	}

	htmlBuilder.WriteString(`
        </table>
    </body>
    </html>
    `)

	return htmlBuilder.String()
}

func renderError(w http.ResponseWriter, errorMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(jsonError{Error: errorMsg})
	if err != nil {
		http.Error(w, `{"error": "Failed to encode error"}`, http.StatusInternalServerError)
	}
}
