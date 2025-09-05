// Package api -  описание хендлеров и эндпоинтов.
package api

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chestorix/monmetrics/internal/domain/interfaces"
	models "github.com/chestorix/monmetrics/internal/metrics"
	"github.com/chestorix/monmetrics/internal/utils"
)

// MetricsHandler обрабатывает HTTP-запросы для операций с метриками.
// Содержит методы для обновления, получения и проверки метрик.
type MetricsHandler struct {
	service    interfaces.Service
	dbDNS      string
	key        string
	privateKey *rsa.PrivateKey
}
type jsonError struct {
	Error string `json:"error"`
}

// NewMetricsHandler создает новый экземпляр MetricsHandler.
// Принимает:
// - service: сервис для работы с метриками
// - dbDNS: строка подключения к БД (может быть пустой)
// - key: ключ для подписи данных (может быть пустым)
// Возвращает указатель на новый MetricsHandler.
func NewMetricsHandler(service interfaces.Service, dbDNS string, key string, crypotKey string) *MetricsHandler {
	var privateKey *rsa.PrivateKey
	var err error
	if crypotKey != "" {
		privateKey, err = utils.LoadPrivateKey(crypotKey)
		if err != nil {
			log.Printf("Error loading private key: %v", err)
		}
	}

	return &MetricsHandler{service: service,
		dbDNS:      dbDNS,
		key:        key,
		privateKey: privateKey,
	}
}

// checkHash проверяет хеш переданных данных.
// Возвращает:
// - bool: совпадает ли хеш
// - string: вычисленный хеш
func (h *MetricsHandler) checkHash(r *http.Request, data []byte) (bool, string) {

	receivedHash := r.Header.Get("HashSHA256")
	if receivedHash == "" {
		return false, ""
	}

	expectedHash := utils.CalculateHash(data, h.key)
	return hmac.Equal([]byte(expectedHash), []byte(receivedHash)), expectedHash
}

// UpdateHandler обрабатывает POST запрос на обновление метрики через URL.
// Формат пути: /update/<metricType>/<metricName>/<metricValue>
// Поддерживаемые типы: gauge, counter
// Возможные коды ответа:
// - 200: успешное обновление
// - 400: неверный запрос
// - 405: метод не разрешен
// - 500: внутренняя ошибка сервера
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
		fmt.Fprintln(w, "Counter metric updated")
	default:
		http.Error(w, models.ErrInvalidMetricType.Error(), http.StatusBadRequest)
	}
}

// GetValuesHandler обрабатывает GET запрос на получение значений метрик.
// Формат пути: /value/<metricType>/<metricName>
// Поддерживаемые типы: gauge, counter
// Возможные коды ответа:
// - 200: успешное получение значения
// - 400: неверный запрос
// - 404: метрика не найдена
// - 405: метод не разрешен
// - 500: внутренняя ошибка сервера
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

// GetAllMetricsHandler обрабатывает GET запрос на получение всех метрик в формате HTML.
// Возможные коды ответа:
// - 200: успешное получение всех метрик
// - 405: метод не разрешен
// - 500: внутренняя ошибка сервера
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

// UpdateJSONHandler обрабатывает POST запрос для обновления метрик в формате JSON.
// Формат JSON: {"id": "metricName", "type": "gauge|counter", "value|delta": number}
// Также проверяет хеш при наличии ключа.
// Возможные коды ответа:
// - 200: успешное обновление
// - 400: неверный запрос или неверный хеш
// - 405: метод не разрешен
// - 500: внутренняя ошибка сервера
func (h *MetricsHandler) UpdateJSONHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*2)
	defer cancel()
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		renderError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		renderError(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	isEncrypted := r.Header.Get("X-Encrypted") == "true"
	body, err = h.decryptRequestBody(body, isEncrypted)
	if err != nil {
		renderError(w, fmt.Sprintf("Decryption failed: %v", err), http.StatusBadRequest)
		return
	}

	if h.key != "" {
		receivedHash := r.Header.Get("HashSHA256")
		if receivedHash == "" {
		} else {
			expectedHash := utils.CalculateHash(body, h.key)
			if !hmac.Equal([]byte(expectedHash), []byte(receivedHash)) {
				renderError(w, "Invalid hash", http.StatusBadRequest)
				return
			}
		}
	}

	var metric models.Metrics
	if err := json.Unmarshal(body, &metric); err != nil {
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

	responseBody, err := json.Marshal(updateMetric)
	if err != nil {
		renderError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if h.key != "" {
		hash := utils.CalculateHash(responseBody, h.key)
		w.Header().Set("HashSHA256", hash)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(responseBody)
}

// ValueJSONHandler обрабатывает POST запрос на получение значений метрик в формате JSON.
// Формат JSON: {"id": "metricName", "type": "gauge|counter"}
// Возможные коды ответа:
// - 200: успешное получение значения
// - 400: неверный запрос
// - 404: метрика не найдена
// - 405: метод не разрешен
// - 500: внутренняя ошибка сервера
func (h *MetricsHandler) ValueJSONHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*2)
	defer cancel()
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		renderError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		renderError(w, "Failed to read/decrypt request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	isEncrypted := r.Header.Get("X-Encrypted") == "true"

	body, err = h.decryptRequestBody(body, isEncrypted)
	if err != nil {
		renderError(w, fmt.Sprintf("Decryption failed: %v", err), http.StatusBadRequest)
		return
	}

	var metric models.Metrics
	if err := json.Unmarshal(body, &metric); err != nil {
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

// PingHandler обрабатывает GET запрос для проверки соединения с Базой данных.
// Возвращает:
// - 200: при успешном соединении
// - 500: при ошибке соединения
// Если dbDNS пустая, всегда возвращает 200.
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

// UpdatesHandler обрабатывает POST запрос обновления пачки метрик за одну транзакцию в формате JSON.
// Формат JSON: [{"id": "metric1", "type": "gauge", "value": 1.23}, ...]
// Также проверяет хеш при наличии ключа.
// Возможные коды ответа:
// - 200: успешное обновление
// - 400: неверный запрос или пустой пакет
// - 405: метод не разрешен
// - 500: внутренняя ошибка сервера
func (h *MetricsHandler) UpdatesHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*2)
	defer cancel()
	w.Header().Set("Content-Type", "application/json")

	log.Printf("UpdatesHandler called, method: %s", r.Method)
	if r.Method != http.MethodPost {
		renderError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Читаем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		renderError(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("Raw request body size: %d bytes", len(body))

	isEncrypted := r.Header.Get("X-Encrypted") == "true"
	isCompressed := r.Header.Get("Content-Encoding") == "gzip"

	var processedData []byte
	if isCompressed {
		log.Printf("Decompressing data...")
		gz, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			log.Printf("Gzip decompression failed: %v", err)
			renderError(w, "Decompression failed", http.StatusBadRequest)
			return
		}
		defer gz.Close()
		processedData, err = io.ReadAll(gz)
		if err != nil {
			log.Printf("Gzip read failed: %v", err)
			renderError(w, "Decompression read failed", http.StatusBadRequest)
			return
		}
		log.Printf("Decompressed: %d -> %d bytes", len(body), len(processedData))
	} else {
		processedData = body
	}

	decryptedBody, err := h.decryptRequestBody(processedData, isEncrypted)
	if err != nil {
		log.Printf("Request decryption failed: %v", err)
		renderError(w, fmt.Sprintf("Decryption failed: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("Final data for parsing: %d bytes", len(decryptedBody))
	log.Printf("Data sample: %s", string(decryptedBody[:min(200, len(decryptedBody))]))

	var metrics []models.Metrics
	if err := json.Unmarshal(decryptedBody, &metrics); err != nil {
		log.Printf("JSON unmarshal error: %v", err)
		log.Printf("Problematic JSON: %s", string(decryptedBody))
		renderError(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("Successfully parsed %d metrics", len(metrics))

	if len(metrics) == 0 {
		renderError(w, "Empty batch", http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateMetricsBatch(ctx, metrics); err != nil {
		log.Printf("Update metrics batch failed: %v", err)
		renderError(w, fmt.Sprintf("Failed to update metrics: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully updated %d metrics", len(metrics))
	w.WriteHeader(http.StatusOK)
}

// decryptRequestBody метод расшифровки запроса

func (h *MetricsHandler) decryptRequestBody(data []byte, isEncrypted bool) ([]byte, error) {
	log.Printf("Request encrypted: %v, private key available: %v",
		isEncrypted, h.privateKey != nil)

	if isEncrypted && h.privateKey == nil {
		return nil, fmt.Errorf("encrypted data received but no private key available")
	}
	if !isEncrypted && h.privateKey != nil {
		return nil, fmt.Errorf("unencrypted data received but encryption is expected")
	}

	log.Printf("Received data size: %d bytes, encrypted: %v", len(data), isEncrypted)

	if isEncrypted && h.privateKey != nil {
		log.Printf("Decrypting data with private key...")
		decryptedData, err := utils.DecryptData(data, h.privateKey)
		if err != nil {
			log.Printf("Decryption failed: %v", err)
			return nil, fmt.Errorf("decryption failed: %w", err)
		}
		log.Printf("Decryption successful: %d -> %d bytes", len(data), len(decryptedData))
		log.Printf("Decrypted data (first 200 chars): %s", string(decryptedData[:min(200, len(decryptedData))]))
		return decryptedData, nil
	}

	log.Printf("Plain text data: %s", string(data[:min(200, len(data))]))
	return data, nil
}

// generateMetricsHTML генерирует HTML страницу со списком всех метрик.
// Принимает slice метрик.
// Возвращает сгенерированную HTML строку.
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

// renderError отправляет ошибку в формате JSON.
// Принимает:
// - w: ResponseWriter для записи ответа
// - errorMsg: текст ошибки
// - statusCode: HTTP статус код
func renderError(w http.ResponseWriter, errorMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(jsonError{Error: errorMsg})
	if err != nil {
		http.Error(w, `{"error": "Failed to encode error"}`, http.StatusInternalServerError)
	}
}
