package memory

import (
	"github.com/chestorix/monmetrics/internal/storage/interfaces"
	"log"
)

type MemStorage struct {
	Gauges   map[string]float64
	Counters map[string]int64
}

func NewMemStorage() interfaces.MetricsRepository {
	return &MemStorage{
		Gauges:   make(map[string]float64),
		Counters: make(map[string]int64),
	}
}
func (m *MemStorage) UpdateGauge(name string, value float64) {
	m.Gauges[name] = value
	log.Println("Gauge ", name, " updated", m.Gauges[name])

}
func (m *MemStorage) UpdateCounter(name string, value int64) {
	m.Counters[name] += value
	log.Println("Counter ", name, " updated", m.Counters[name])

}

/*func (m *MemStorage) UpdateHandler(w http.ResponseWriter, r *http.Request) {
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
	m.UpdateGauge(metricName, value)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Gauge metric updated")

case "counter":
	value, err := strconv.ParseInt(metricValue, 10, 64)
	if err != nil {
		http.Error(w, "Invalid value for counter", http.StatusBadRequest)
		return
	}
	m.UpdateCounter(metricName, value)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Counter metric updated")

default:
	http.Error(w, "Invalid metric type", http.StatusBadRequest)
}*/
