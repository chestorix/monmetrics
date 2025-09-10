// Package sender содержит логику отправки метрик через http.
package sender

import (
	"bytes"
	"compress/gzip"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"time"

	models "github.com/chestorix/monmetrics/internal/metrics"
	"github.com/chestorix/monmetrics/internal/utils"
)

type HTTPSender struct {
	retryDelays []time.Duration
	baseURL     string
	key         string
	publicKey   *rsa.PublicKey
	client      *http.Client
	logger      *logrus.Logger
}

func NewHTTPSender(baseURL string, key string, cryptoKey string, logger *logrus.Logger) *HTTPSender {

	var err error
	var publicKey *rsa.PublicKey
	if cryptoKey != "" {
		publicKey, err = utils.LoadPublicKey(cryptoKey)
		if err != nil {
			logger.Infof("Failed to load public key: %v", err)
		}
	}
	return &HTTPSender{
		baseURL:     baseURL,
		client:      &http.Client{Timeout: 5 * time.Second},
		retryDelays: []time.Duration{time.Second, 3 * time.Second, 5 * time.Second},
		publicKey:   publicKey,
		key:         key,
		logger:      logger,
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
	s.logger.Infof("Sending JSON metric %s to %s", metric.Name, s.baseURL)
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

		var requestBody []byte
		var contentType string

		if s.publicKey != nil {
			encryptedData, err := utils.EncryptData(jsonData, s.publicKey)
			if err != nil {
				return fmt.Errorf("encryption failed: %w", err)
			}
			requestBody = encryptedData
			contentType = "application/octet-stream"
		} else {
			requestBody = jsonData
			contentType = "application/json"
		}

		req, err := http.NewRequest("POST", s.baseURL+"/update/", bytes.NewBuffer(requestBody))
		if err != nil {
			return utils.ErrMaxRetriesExceeded
		}

		req.Header.Set("Content-Type", contentType)

		if s.publicKey != nil {
			req.Header.Set("X-Encrypted", "true")
		}

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
	s.logger.Infof("SendBatch called with %d metrics to %s, encryption: %v",
		len(metrics), s.baseURL, s.publicKey != nil)

	return utils.Retry(3, s.retryDelays, func() error {
		s.logger.Info("Attempting send (retry attempt)...")

		jsonData, err := json.Marshal(metrics)
		if err != nil {
			s.logger.Errorf("JSON marshaling error: %v", err)
			return utils.ErrMaxRetriesExceeded
		}
		s.logger.Infof("JSON data size: %d bytes", len(jsonData))

		var dataToCompress []byte
		var contentType string

		if s.publicKey != nil {
			s.logger.Infof("Encrypting %d bytes of JSON data", len(jsonData))
			encryptedData, err := utils.EncryptData(jsonData, s.publicKey)
			if err != nil {
				s.logger.Errorf("Encryption failed: %v", err)
				return fmt.Errorf("encryption failed: %w", err)
			}
			dataToCompress = encryptedData
			contentType = "application/octet-stream"
			s.logger.Infof("Encrypted to %d bytes", len(encryptedData))
		} else {
			dataToCompress = jsonData
			contentType = "application/json"
			s.logger.Infof("No encryption, using plain JSON")
		}

		s.logger.Infof("Compressing data (size: %d bytes)", len(dataToCompress))
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		if _, errWrite := gz.Write(dataToCompress); errWrite != nil {
			s.logger.Infof("Gzip write error: %v", errWrite)
			return utils.ErrMaxRetriesExceeded
		}
		if errClose := gz.Close(); errClose != nil {
			s.logger.Infof("Gzip close error: %v", errClose)
			return utils.ErrMaxRetriesExceeded
		}
		compressedData := buf.Bytes()
		s.logger.Infof("Compressed to %d bytes (ratio: %.1f%%)",
			len(compressedData),
			float64(len(compressedData))/float64(len(dataToCompress))*100)

		req, err := http.NewRequest("POST", s.baseURL+"/updates/", &buf)
		if err != nil {
			s.logger.Errorf("Request creation error: %v", err)
			return utils.ErrMaxRetriesExceeded
		}

		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Content-Encoding", "gzip")

		if s.publicKey != nil {
			req.Header.Set("X-Encrypted", "true")
			s.logger.Info("Set X-Encrypted header: true")
		}

		if hash := utils.CalculateHash(jsonData, s.key); hash != "" {
			req.Header.Set("HashSHA256", hash)
			s.logger.Infof("Set HashSHA256 header: %s", hash)
		}

		s.logger.Infof("Sending request to: %s", req.URL.String())
		s.logger.Infof("Request headers: %+v", req.Header)
		s.logger.Infof("Request body size: %d bytes", buf.Len())

		startTime := time.Now()
		resp, err := s.client.Do(req)
		requestDuration := time.Since(startTime)

		if err != nil {
			s.logger.Infof("HTTP request error: %v (duration: %v)", err, requestDuration)
			if utils.IsNetworkError(err) {
				return err
			}
			return utils.ErrMaxRetriesExceeded
		}
		defer resp.Body.Close()

		s.logger.Infof("Response received: status=%d, duration=%v", resp.StatusCode, requestDuration)
		s.logger.Infof("Response headers: %+v", resp.Header)

		body, _ := io.ReadAll(resp.Body)
		if len(body) > 0 {
			s.logger.Infof("Response body: %s", string(body))
		}

		if resp.StatusCode >= 500 {
			s.logger.Infof("Server error: %d", resp.StatusCode)
			return fmt.Errorf("server error: %d", resp.StatusCode)
		}
		if resp.StatusCode != http.StatusOK {
			s.logger.Infof("Unexpected status code: %d", resp.StatusCode)
			return utils.ErrMaxRetriesExceeded
		}

		s.logger.Infof("SendBatch completed successfully")
		return nil
	})
}
