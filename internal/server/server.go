package server

import (
	"fmt"
	"github.com/chestorix/monmetrics/internal/config"
	"github.com/chestorix/monmetrics/internal/handlers"
	"github.com/chestorix/monmetrics/internal/storage/interfaces"
	"github.com/go-chi/chi/v5"
	"net/http"
)

type Server struct {
	cfg    *config.ServerConfig
	repo   interfaces.MetricsRepository
	server *http.Server
	router *chi.Mux
}

func New(cfg *config.ServerConfig, repo interfaces.MetricsRepository) *Server {
	return &Server{
		cfg:    cfg,
		repo:   repo,
		router: chi.NewRouter(),
	}
}

func (s *Server) setupRoutes(handler *handlers.MetricsHandler) {
	s.router.Route("/", func(r chi.Router) {
		r.Get("/", handler.GetAllMetricsHandler)

		r.Route("/update", func(r chi.Router) {
			r.Post("/{metricType}/{metricName}/{metricValue}", handler.UpdateHandler)
		})

		r.Route("/value", func(r chi.Router) {
			r.Get("/{metricType}/{metricName}", handler.GetValuesHandler)
		})
	})
}

func (s *Server) Start() error {
	handler := handlers.NewMetricsHandler(s.repo)
	s.setupRoutes(handler)

	s.server = &http.Server{
		Addr:    s.cfg.Address,
		Handler: s.router,
	}

	fmt.Printf("Server is listening on %s\n", s.cfg.Address)
	return s.server.ListenAndServe()
}
