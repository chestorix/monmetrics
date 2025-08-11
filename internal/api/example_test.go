package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	models "github.com/chestorix/monmetrics/internal/metrics"
	"net/http"
	"net/http/httptest"
	"testing"
)

func ExampleUpdateHandler() {
	// Создаем тестовый сервер
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Gauge metric updated")
	}))
	defer ts.Close()

	// создаем запрос на обновление метрики Gauage
	res, err := http.Post(ts.URL+"/update/gauge/test_metric/123.45", "text/plain", nil)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return
	}
	defer res.Body.Close()

	fmt.Println(res.Status)
	// Output: 200 OK
}

func ExampleValueJSONHandler() {
	// Работа с JSON
	metric := models.Metrics{
		ID:    "test_metric",
		MType: models.Gauge,
	}

	// Серелизуем в JSON
	jsonData, err := json.Marshal(metric)
	if err != nil {
		fmt.Printf("Error marshaling metric: %v\n", err)
		return
	}

	// Создаем тестовый сервер
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		response := models.Metrics{
			ID:    "test_metric",
			MType: models.Gauge,
			Value: func() *float64 { v := 123.45; return &v }(),
		}
		json.NewEncoder(w).Encode(response)
	}))

	defer ts.Close()

	// Создаем запрос на получение значения метрики
	res, err := http.Post(ts.URL+"/value/", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return
	}
	defer res.Body.Close()

	var result models.Metrics
	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		fmt.Printf("Error decoding response: %v\n", err)
		return
	}

	fmt.Printf("Metric %s value: %.2f\n", result.ID, *result.Value)
	// Output: Metric test_metric value: 123.45
}

func TestExamples(t *testing.T) {
	ExampleUpdateHandler()
	ExampleValueJSONHandler()
}
