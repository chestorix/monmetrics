// Package api -  описание хендлеров и эндпоинтов.
package api

import (
	"context"
	"crypto/rsa"
	"github.com/chestorix/monmetrics/internal/utils"
	"net/http"
	_ "net/http/pprof"

	"github.com/chestorix/monmetrics/internal/config"
	"github.com/chestorix/monmetrics/internal/domain/interfaces"
	"github.com/sirupsen/logrus"
)

type Server struct {
	cfg        *config.ServerConfig
	router     *Router
	service    interfaces.Service
	server     *http.Server
	logger     *logrus.Logger
	privateKey *rsa.PrivateKey
	key        string
}

func NewServer(cfg *config.ServerConfig, metricService interfaces.Service, logger *logrus.Logger) *Server {
	var privateKey *rsa.PrivateKey
	var err error
	if cfg.CryptoKey != "" {
		privateKey, err = utils.LoadPrivateKey(cfg.CryptoKey)
		if err != nil {
			logger.Errorf("Error loading private key: %v", err)
		}
	}
	router := NewRouter(logger, privateKey)
	return &Server{
		cfg:        cfg,
		service:    metricService,
		router:     router,
		logger:     logger,
		key:        cfg.Key,
		privateKey: privateKey,
	}
}

func (s *Server) Start() error {
	handler := NewMetricsHandler(s.service, s.cfg.DatabaseDSN, s.key, s.privateKey, s.logger)
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
