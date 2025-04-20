package server

import (
	"fmt"
	"github.com/chestorix/monmetrics/internal/config"
	"github.com/chestorix/monmetrics/internal/handlers"
	"github.com/chestorix/monmetrics/internal/storage/interfaces"
	"net/http"
)

type Server struct {
	cfg    *config.ServerConfig
	repo   interfaces.MetricsRepository
	server *http.Server
}

func New(cfg *config.ServerConfig, repo interfaces.MetricsRepository) *Server {
	return &Server{
		cfg:  cfg,
		repo: repo,
	}
}

func (s *Server) Start() error {
	handler := handlers.NewMetricsHandler(s.repo)

	mux := http.NewServeMux()
	mux.HandleFunc("/update/", handler.UpdateHandler)
	mux.HandleFunc("/value/", handler.GetValuesHandler)
	mux.HandleFunc("/", handler.GetAllMetricsHandler)
	s.server = &http.Server{
		Addr:    s.cfg.Address,
		Handler: mux,
	}
	fmt.Println("Server is listening on http://localhost:8080")
	return s.server.ListenAndServe()
}
