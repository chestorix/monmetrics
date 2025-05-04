package sender

import (
	"bytes"
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

/*func (s *HTTPSender) Send(metric models.Metric) error {
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
}*/

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
