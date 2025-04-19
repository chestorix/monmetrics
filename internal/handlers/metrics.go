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
