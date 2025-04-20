package handlers

import (
	"fmt"
	"github.com/chestorix/monmetrics/internal/storage/interfaces"
	"net/http"
	"strconv"
	"strings"
)

type MetricsHandler struct {
	repo interfaces.MetricsRepository
}

func NewMetricsHandler(repo interfaces.MetricsRepository) *MetricsHandler {
	return &MetricsHandler{repo: repo}
}

func (h *MetricsHandler) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 {
		http.Error(w, "Invalid request", http.StatusNotFound)
		return
	}

	if len(parts) == 2 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	metricType := parts[1]

	if len(parts) < 4 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	metricName, metricValue := parts[2], parts[3]

	switch metricType {
	case "gauge":
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(w, "Invalid value for gauge", http.StatusBadRequest)
			return
		}
		h.repo.UpdateGauge(metricName, value)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Gauge metric updated")

	case "counter":
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(w, "Invalid value for counter", http.StatusBadRequest)
			return
		}
		h.repo.UpdateCounter(metricName, value)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Counter metric updated")

	default:
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
	}

}

func (h *MetricsHandler) GetValuesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid request", http.StatusNotFound)
		return
	}

	if len(parts) == 2 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	metricType := parts[1]
	metricName := parts[2]
	switch metricType {
	case "gauge":
		value, exists := h.repo.GetGauge(metricName)
		if !exists {
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}
		strValue := fmt.Sprintf("%v", value)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(strValue))

	case "counter":
		value, exists := h.repo.GetCounter(metricName)
		if !exists {
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}
		strValue := fmt.Sprintf("%v", value)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(strValue))

	default:
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
	}
}

func (h *MetricsHandler) GetAllMetricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics, err := h.repo.GetAllMetrics()
	if err != nil {
		http.Error(w, "Failed to get metrics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
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
