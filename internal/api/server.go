package api

import (
	"fmt"
	"github.com/chestorix/monmetrics/internal/config"
	"github.com/chestorix/monmetrics/internal/domain/interfaces"
	"github.com/sirupsen/logrus"
	"net/http"
)

type Server struct {
	cfg     *config.ServerConfig
	router  *Router
	service interfaces.Service
}

func NewServer(cfg *config.ServerConfig, metricService interfaces.Service, logger *logrus.Logger) *Server {
	router := NewRouter(logger)
	return &Server{
		cfg:     cfg,
		service: metricService,
		router:  router,
	}
}

func (s *Server) Start() error {
	handler := NewMetricsHandler(s.service)
	s.router.SetupRoutes(handler)

	httpServer := &http.Server{
		Addr:    s.cfg.Address,
		Handler: s.router,
	}
	fmt.Println("Server listened address: ", s.cfg.Address)

	return httpServer.ListenAndServe()
}
