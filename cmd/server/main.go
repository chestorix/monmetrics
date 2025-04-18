package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type Gauge float64
type Counter int64

type MemStorage struct {
	gauges   map[string]Gauge
	counters map[string]Counter
}

func (m *MemStorage) UpdateGauge(name string, value Gauge) {
	m.gauges[name] = value

}
func (m *MemStorage) UpdateCounter(name string, value Counter) {
	m.counters[name] += value

}

func (m *MemStorage) updateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 5 {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	metricType, metricName, metricValue := parts[2], parts[3], parts[4]

	switch metricType {
	case "gauge":
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(w, "Invalid value for gauge", http.StatusBadRequest)
			return
		}
		m.UpdateGauge(metricName, Gauge(value))
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Gauge metric updated")

	case "counter":
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(w, "Invalid value for counter", http.StatusBadRequest)
			return
		}
		m.UpdateCounter(metricName, Counter(value))
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Counter metric updated")

	default:
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
	}
}

func main() {
	storage := &MemStorage{
		gauges:   make(map[string]Gauge),
		counters: make(map[string]Counter),
	}

	http.HandleFunc("/update/", storage.updateHandler)

	fmt.Println("Server is listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
