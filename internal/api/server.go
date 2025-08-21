// Package api -  описание хендлеров и эндпоинтов.
package api

import (
	"context"
	"net/http"
	_ "net/http/pprof"

	"github.com/chestorix/monmetrics/internal/config"
	"github.com/chestorix/monmetrics/internal/domain/interfaces"
	"github.com/sirupsen/logrus"
)

type Server struct {
	cfg     *config.ServerConfig
	router  *Router
	service interfaces.Service
	server  *http.Server
	logger  *logrus.Logger
	key     string
}

func NewServer(cfg *config.ServerConfig, metricService interfaces.Service, logger *logrus.Logger) *Server {
	router := NewRouter(logger)
	return &Server{
		cfg:     cfg,
		service: metricService,
		router:  router,
		logger:  logger,
		key:     cfg.Key,
	}
}

func (s *Server) Start() error {
	handler := NewMetricsHandler(s.service, s.cfg.DatabaseDSN, s.key)
	s.router.SetupRoutes(handler)

	httpServer := &http.Server{
		Addr:    s.cfg.Address,
		Handler: s.router,
	}
	s.logger.Infoln("Server listened address: ", s.cfg.Address)

	return httpServer.ListenAndServe()

}
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
