// Package sender содержит логику отправки метрик через http.
package sender

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	models "github.com/chestorix/monmetrics/internal/metrics"
	"github.com/chestorix/monmetrics/internal/utils"
)

type HTTPSender struct {
	retryDelays []time.Duration
	baseURL     string
	key         string
	client      *http.Client
}

func NewHTTPSender(baseURL string, key string) *HTTPSender {
	return &HTTPSender{
		baseURL:     baseURL,
		client:      &http.Client{Timeout: 5 * time.Second},
		retryDelays: []time.Duration{time.Second, 3 * time.Second, 5 * time.Second},
		key:         key,
	}
}

func (s *HTTPSender) Send(metric models.Metric) error {

	return utils.Retry(3, s.retryDelays, func() error {
		url := fmt.Sprintf("%s/update/%s/%s/%v", s.baseURL, metric.Type, metric.Name, metric.Value)

		resp, err := s.client.Post(url, "text/plain", nil)
		if err != nil {
			if utils.IsNetworkError(err) {
				return err
			}
			return utils.ErrMaxRetriesExceeded
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			return fmt.Errorf("server error: %d", resp.StatusCode)
		}
		if resp.StatusCode != http.StatusOK {
			return utils.ErrMaxRetriesExceeded
		}

		return nil
	})
}

func (s *HTTPSender) SendJSON(metric models.Metric) error {
	return utils.Retry(3, s.retryDelays, func() error {
		var m models.Metrics
		m.ID = metric.Name
		m.MType = metric.Type

		switch metric.Type {
		case models.Gauge:
			if value, ok := metric.Value.(float64); ok {
				m.Value = &value
			} else {
				return utils.ErrMaxRetriesExceeded
			}
		case models.Counter:
			if value, ok := metric.Value.(int64); ok {
				m.Delta = &value
			} else {
				return utils.ErrMaxRetriesExceeded
			}
		default:
			return models.ErrInvalidMetricType
		}

		jsonData, err := json.Marshal(m)
		if err != nil {
			return utils.ErrMaxRetriesExceeded
		}

		req, err := http.NewRequest("POST", s.baseURL+"/update/", bytes.NewBuffer(jsonData))
		if err != nil {
			return utils.ErrMaxRetriesExceeded
		}

		req.Header.Set("Content-Type", "application/json")

		if s.key != "" {
			hash := utils.CalculateHash(jsonData, s.key)
			req.Header.Set("HashSHA256", hash)
		}

		resp, err := s.client.Do(req)
		if err != nil {
			if utils.IsNetworkError(err) {
				return err
			}
			return utils.ErrMaxRetriesExceeded
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			return fmt.Errorf("server error: %d", resp.StatusCode)
		}
		if resp.StatusCode != http.StatusOK {
			return utils.ErrMaxRetriesExceeded
		}

		return nil
	})
}

func (s *HTTPSender) SendBatch(metrics []models.Metrics) error {

	return utils.Retry(3, s.retryDelays, func() error {

		jsonData, err := json.Marshal(metrics)
		if err != nil {
			return utils.ErrMaxRetriesExceeded
		}

		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		if err := enc.Encode(metrics); err != nil {
			return err
		}
		gz := gzip.NewWriter(&buf)
		if _, errWrite := gz.Write(jsonData); errWrite != nil {
			return utils.ErrMaxRetriesExceeded
		}
		if errClose := gz.Close(); errClose != nil {
			return utils.ErrMaxRetriesExceeded
		}

		req, err := http.NewRequest("POST", s.baseURL+"/updates/", &buf)
		if err != nil {
			return utils.ErrMaxRetriesExceeded
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		if hash := utils.CalculateHash(jsonData, s.key); hash != "" {
			req.Header.Set("HashSHA256", hash)
		}
		resp, err := s.client.Do(req)
		if err != nil {
			if utils.IsNetworkError(err) {
				return err
			}
			return utils.ErrMaxRetriesExceeded
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			return fmt.Errorf("server error: %d", resp.StatusCode)
		}
		if resp.StatusCode != http.StatusOK {
			return utils.ErrMaxRetriesExceeded
		}
		return nil
	})
}
