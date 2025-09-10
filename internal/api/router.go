// Package api -  описание хендлеров и эндпоинтов.
package api

import (
	"crypto/rsa"
	"net/http"
	"net/http/pprof"

	middleware2 "github.com/chestorix/monmetrics/internal/api/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
)

type Router struct {
	chi.Router
	logger *logrus.Logger
}

func NewRouter(logger *logrus.Logger, privateKey *rsa.PrivateKey) *Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware2.NewLoggerMiddleware(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware2.GzipDecryptMiddleware(privateKey, logger))

	return &Router{
		Router: r,
		logger: logger,
	}
}

func (r *Router) SetupRoutes(metricsHandler *MetricsHandler) {
	r.Route("/", func(r chi.Router) {
		r.Get("/", metricsHandler.GetAllMetricsHandler)

		r.Handle("/debug/pprof/*", http.HandlerFunc(pprof.Index))
		r.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		r.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		r.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
		r.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
		r.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		r.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
		r.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
		r.Handle("/debug/pprof/block", pprof.Handler("block"))
		r.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))

		r.Route("/update", func(r chi.Router) {
			r.Post("/", metricsHandler.UpdateJSONHandler)
			r.Post("/{metricType}/{metricName}/{metricValue}", metricsHandler.UpdateHandler)
		})

		r.Route("/value", func(r chi.Router) {
			r.Post("/", metricsHandler.ValueJSONHandler)
			r.Get("/{metricType}/{metricName}", metricsHandler.GetValuesHandler)
		})
		r.Route("/ping", func(r chi.Router) {
			r.Get("/", metricsHandler.PingHandler)
		})
		r.Route("/updates", func(r chi.Router) {
			r.Post("/", metricsHandler.UpdatesHandler)
		})
	})
}
