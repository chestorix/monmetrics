// Package middleware - логика промежуточной обработки http запросов.
package middleware

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"strconv"
	"strings"
)

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		isEncrypted := r.Header.Get("X-Encrypted") == "true"
		if !isEncrypted && strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			defer gz.Close()
			r.Body = gz
		}

		ww := &gzipResponseWriter{ResponseWriter: w}

		next.ServeHTTP(ww, r)

		contentType := ww.Header().Get("Content-Type")
		acceptsGzip := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")

		shouldCompress := acceptsGzip &&
			!strings.Contains(contentType, "text/html") &&
			!strings.Contains(contentType, "image/") &&
			!strings.Contains(contentType, "application/octet-stream") &&
			ww.statusCode == http.StatusOK

		if shouldCompress && ww.buffer.Len() > 0 {
			var buf bytes.Buffer
			gz := gzip.NewWriter(&buf)
			gz.Write(ww.buffer.Bytes())
			gz.Close()

			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Vary", "Accept-Encoding")
			w.Header().Del("Content-Length")
			w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))

			w.WriteHeader(ww.statusCode)
			w.Write(buf.Bytes())
		} else {
			w.WriteHeader(ww.statusCode)
			w.Write(ww.buffer.Bytes())
		}
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	buffer     bytes.Buffer
	statusCode int
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.buffer.Write(b)
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	w.statusCode = code
}
