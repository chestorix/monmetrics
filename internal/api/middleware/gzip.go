package middleware

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

var flateWriterPool = sync.Pool{
	New: func() interface{} {
		w, _ := flate.NewWriter(nil, flate.BestSpeed) // Быстрая компрессия
		return w
	},
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Распаковка входящего gzip (если есть)
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			defer gz.Close()
			r.Body = gz
		}

		// Проверяем, поддерживает ли клиент gzip
		acceptsGzip := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
		if !acceptsGzip {
			next.ServeHTTP(w, r)
			return
		}

		// Создаём gzip.Writer
		gz := flateWriterPool.Get().(*flate.Writer)
		defer flateWriterPool.Put(gz)
		gz.Reset(w)
		defer gz.Close()

		// Устанавливаем заголовки
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		w.Header().Del("Content-Length") // Важно! Удаляем старый Content-Length

		// Передаём управление следующему обработчику
		next.ServeHTTP(&gzipResponseWriter{Writer: gz, ResponseWriter: w}, r)
	})
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	w.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(code)
}
