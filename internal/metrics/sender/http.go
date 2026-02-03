// Package sender содержит логику отправки метрик через http.
package sender

import (
	"bytes"
	"compress/gzip"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
}

func NewHTTPSender(baseURL string, key string, cryptoKey string) *HTTPSender {

	var err error
	var publicKey *rsa.PublicKey
	if cryptoKey != "" {
		publicKey, err = utils.LoadPublicKey(cryptoKey)
		if err != nil {
			log.Printf("Failed to load public key: %v", err)
		}
	}
	return &HTTPSender{
		baseURL:     baseURL,
		client:      &http.Client{Timeout: 5 * time.Second},
		retryDelays: []time.Duration{time.Second, 3 * time.Second, 5 * time.Second},
		publicKey:   publicKey,
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
	log.Printf("Sending JSON metric %s to %s", metric.Name, s.baseURL)
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
	log.Printf("SendBatch called with %d metrics to %s, encryption: %v",
		len(metrics), s.baseURL, s.publicKey != nil)

	return utils.Retry(3, s.retryDelays, func() error {
		log.Printf("Attempting send (retry attempt)...")

		jsonData, err := json.Marshal(metrics)
		if err != nil {
			log.Printf("JSON marshaling error: %v", err)
			return utils.ErrMaxRetriesExceeded
		}
		log.Printf("JSON data size: %d bytes", len(jsonData))

		var dataToCompress []byte
		var contentType string

		if s.publicKey != nil {
			log.Printf("Encrypting %d bytes of JSON data", len(jsonData))
			encryptedData, err := utils.EncryptData(jsonData, s.publicKey)
			if err != nil {
				log.Printf("Encryption failed: %v", err)
				return fmt.Errorf("encryption failed: %w", err)
			}
			dataToCompress = encryptedData
			contentType = "application/octet-stream"
			log.Printf("Encrypted to %d bytes", len(encryptedData))
		} else {
			dataToCompress = jsonData
			contentType = "application/json"
			log.Printf("No encryption, using plain JSON")
		}

		log.Printf("Compressing data (size: %d bytes)", len(dataToCompress))
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		if _, errWrite := gz.Write(dataToCompress); errWrite != nil {
			log.Printf("Gzip write error: %v", errWrite)
			return utils.ErrMaxRetriesExceeded
		}
		if errClose := gz.Close(); errClose != nil {
			log.Printf("Gzip close error: %v", errClose)
			return utils.ErrMaxRetriesExceeded
		}
		compressedData := buf.Bytes()
		log.Printf("Compressed to %d bytes (ratio: %.1f%%)",
			len(compressedData),
			float64(len(compressedData))/float64(len(dataToCompress))*100)

		req, err := http.NewRequest("POST", s.baseURL+"/updates/", &buf)
		if err != nil {
			log.Printf("Request creation error: %v", err)
			return utils.ErrMaxRetriesExceeded
		}

		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Content-Encoding", "gzip")

		if s.publicKey != nil {
			req.Header.Set("X-Encrypted", "true")
			log.Printf("Set X-Encrypted header: true")
		}

		if hash := utils.CalculateHash(jsonData, s.key); hash != "" {
			req.Header.Set("HashSHA256", hash)
			log.Printf("Set HashSHA256 header: %s", hash)
		}

		log.Printf("Sending request to: %s", req.URL.String())
		log.Printf("Request headers: %+v", req.Header)
		log.Printf("Request body size: %d bytes", buf.Len())

		startTime := time.Now()
		resp, err := s.client.Do(req)
		requestDuration := time.Since(startTime)

		if err != nil {
			log.Printf("HTTP request error: %v (duration: %v)", err, requestDuration)
			if utils.IsNetworkError(err) {
				return err
			}
			return utils.ErrMaxRetriesExceeded
		}
		defer resp.Body.Close()

		log.Printf("Response received: status=%d, duration=%v", resp.StatusCode, requestDuration)
		log.Printf("Response headers: %+v", resp.Header)

		body, _ := io.ReadAll(resp.Body)
		if len(body) > 0 {
			log.Printf("Response body: %s", string(body))
		}

		if resp.StatusCode >= 500 {
			log.Printf("Server error: %d", resp.StatusCode)
			return fmt.Errorf("server error: %d", resp.StatusCode)
		}
		if resp.StatusCode != http.StatusOK {
			log.Printf("Unexpected status code: %d", resp.StatusCode)
			return utils.ErrMaxRetriesExceeded
		}

		log.Printf("SendBatch completed successfully")
		return nil
	})
}
