package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
)

type Router struct {
	chi.Router
	logger *logrus.Logger
}

func NewRouter(logger *logrus.Logger) *Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(NewLoggerMiddleware(logger))
	r.Use(middleware.Recoverer)

	return &Router{
		Router: r,
		logger: logger,
	}
}

func (r *Router) SetupRoutes(metricsHandler *MetricsHandler) {

	r.Route("/", func(r chi.Router) {
		r.Get("/", metricsHandler.GetAllMetricsHandler)

		r.Route("/update", func(r chi.Router) {
			r.Post("/{metricType}/{metricName}/{metricValue}", metricsHandler.UpdateHandler)
		})

		r.Route("/value", func(r chi.Router) {
			r.Get("/{metricType}/{metricName}", metricsHandler.GetValuesHandler)
		})
	})
}
