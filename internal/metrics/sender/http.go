package sender

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	models "github.com/chestorix/monmetrics/internal/metrics"
	"net/http"
	"time"
)

type HTTPSender struct {
	baseURL string
	client  *http.Client
}

func NewHTTPSender(baseURL string) *HTTPSender {
	return &HTTPSender{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *HTTPSender) Send(metric models.Metric) error {
	url := fmt.Sprintf("%s/update/%s/%s/%v", s.baseURL, metric.Type, metric.Name, metric.Value)

	resp, err := s.client.Post(url, "text/plain", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

func (s *HTTPSender) SendJSON(metric models.Metric) error {
	var m models.Metrics
	m.ID = metric.Name
	m.MType = metric.Type

	switch metric.Type {
	case models.Gauge:
		if value, ok := metric.Value.(float64); ok {
			m.Value = &value
		} else {
			return fmt.Errorf("invalid gauge value type")
		}
	case models.Counter:
		if value, ok := metric.Value.(int64); ok {
			m.Delta = &value
		} else {
			return fmt.Errorf("invalid counter value type")
		}
	default:
		return models.ErrInvalidMetricType
	}

	jsonData, err := json.Marshal(m)
	if err != nil {
		return err
	}

	resp, err := s.client.Post(s.baseURL+"/update/", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

func (s *HTTPSender) SendBatch(metrics []models.Metrics) error {
	var batch []models.Metrics
	for _, metric := range metrics {
		m := models.Metrics{
			ID:    metric.ID,
			MType: metric.MType,
		}
		switch metric.MType {
		case models.Gauge:
			m.Value = metric.Value

		case models.Counter:

			m.Delta = metric.Delta

		}
		batch = append(batch, m)
	}
	jsonData, err := json.Marshal(batch)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(jsonData); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	req, err := http.NewRequest("POST", s.baseURL+"/updates/", &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	return nil
}
