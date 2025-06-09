package middleware

import (
	"bytes"
	"github.com/chestorix/monmetrics/internal/utils"
	"io"
	"net/http"
)

func HashCheckMiddleware(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			if r.Method == http.MethodGet || r.Body == nil {
				next.ServeHTTP(w, r)
				return
			}

			incomingHash := r.Header.Get("HashSHA256")
			if incomingHash == "" {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewBuffer(body))

			computedHash := utils.ComputeHmacSHA256(string(body), key)
			if incomingHash != computedHash {
				http.Error(w, "Invalid hash", http.StatusBadRequest)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

type hashResponseWriter struct {
	http.ResponseWriter
	key string
	buf bytes.Buffer
}

func (w *hashResponseWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

func (w *hashResponseWriter) WriteHeader(code int) {
	if w.key != "" {
		computedHash := utils.ComputeHmacSHA256(w.buf.String(), w.key)
		w.Header().Set("HashSHA256", computedHash)
	}
	w.ResponseWriter.WriteHeader(code)
	w.ResponseWriter.Write(w.buf.Bytes())
}
