package middleware

import (
	"bytes"
	"compress/gzip"
	"crypto/rsa"
	"github.com/chestorix/monmetrics/internal/utils"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
)

// DecryptMiddleware создает middleware для дешифрования входящих запросов
func DecryptMiddleware(privateKey *rsa.PrivateKey, logger *logrus.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if privateKey == nil || r.Header.Get("X-Encrypted") != "true" {
				next.ServeHTTP(w, r)
				return
			}

			logger.Debug("Processing encrypted request")

			body, err := io.ReadAll(r.Body)
			if err != nil {
				logger.WithError(err).Error("Failed to read request body")
				http.Error(w, "Failed to read request", http.StatusBadRequest)
				return
			}
			defer r.Body.Close()

			decryptedData, err := utils.DecryptData(body, privateKey)
			if err != nil {
				logger.WithError(err).Error("Decryption failed")
				http.Error(w, "Decryption failed", http.StatusBadRequest)
				return
			}

			logger.Debugf("Successfully decrypted %d bytes to %d bytes", len(body), len(decryptedData))

			r.Body = io.NopCloser(bytes.NewReader(decryptedData))
			r.ContentLength = int64(len(decryptedData))

			r.Header.Del("X-Encrypted")

			next.ServeHTTP(w, r)
		})
	}
}

// GzipDecryptMiddleware комбинирует распаковку gzip и дешифрование
func GzipDecryptMiddleware(privateKey *rsa.PrivateKey, logger *logrus.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if r.Header.Get("Content-Encoding") == "gzip" {
				gz, err := gzip.NewReader(r.Body)
				if err != nil {
					logger.WithError(err).Error("Gzip decompression failed")
					http.Error(w, "Decompression failed", http.StatusBadRequest)
					return
				}
				defer gz.Close()

				body, err := io.ReadAll(gz)
				if err != nil {
					logger.WithError(err).Error("Failed to read decompressed data")
					http.Error(w, "Decompression read failed", http.StatusBadRequest)
					return
				}

				r.Body = io.NopCloser(bytes.NewReader(body))
				r.Header.Del("Content-Encoding")
				r.ContentLength = int64(len(body))
			}

			if privateKey != nil && r.Header.Get("X-Encrypted") == "true" {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					logger.WithError(err).Error("Failed to read request body for decryption")
					http.Error(w, "Failed to read request", http.StatusBadRequest)
					return
				}
				defer r.Body.Close()

				decryptedData, err := utils.DecryptData(body, privateKey)
				if err != nil {
					logger.WithError(err).Error("Decryption failed")
					http.Error(w, "Decryption failed", http.StatusBadRequest)
					return
				}

				r.Body = io.NopCloser(bytes.NewReader(decryptedData))
				r.ContentLength = int64(len(decryptedData))
				r.Header.Del("X-Encrypted")
			}

			next.ServeHTTP(w, r)
		})
	}
}
