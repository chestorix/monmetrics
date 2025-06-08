package api

import (
	middleware2 "github.com/chestorix/monmetrics/internal/api/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
)

type Router struct {
	chi.Router
	logger *logrus.Logger
}

func NewRouter(logger *logrus.Logger, key string) *Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware2.NewLoggerMiddleware(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware2.GzipMiddleware)
	r.Use(middleware2.HashCheckMiddleware(key))

	return &Router{
		Router: r,
		logger: logger,
	}
}

func (r *Router) SetupRoutes(metricsHandler *MetricsHandler) {

	r.Route("/", func(r chi.Router) {
		r.Get("/", metricsHandler.GetAllMetricsHandler)

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
