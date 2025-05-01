package api

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

func NewLoggerMiddleware(logger *logrus.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Специальный ResponseWriter для отслеживания статуса и размера
			ww := &responseWriter{w, http.StatusOK, 0}

			defer func() {
				logger.WithFields(logrus.Fields{
					"uri":      r.RequestURI,
					"method":   r.Method,
					"status":   ww.status,
					"size":     ww.size,
					"duration": time.Since(start).String(),
				}).Info("request completed")
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.size += size
	return size, err
}
