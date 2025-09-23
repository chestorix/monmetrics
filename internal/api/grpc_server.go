package api

import (
	"context"
	"crypto/rsa"
	"fmt"
	"github.com/chestorix/monmetrics/internal/utils"

	"github.com/chestorix/monmetrics/internal/config"
	"github.com/chestorix/monmetrics/internal/domain/interfaces"
	models "github.com/chestorix/monmetrics/internal/metrics"
	"github.com/chestorix/monmetrics/internal/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net"
)

type GRPCServer struct {
	proto.UnimplementedMetricsServiceServer
	server     *grpc.Server
	service    interfaces.Service
	cfg        *config.ServerConfig
	logger     *logrus.Logger
	privateKey *rsa.PrivateKey
}

func NewGRPCServer(cfg *config.ServerConfig, service interfaces.Service, logger *logrus.Logger) *GRPCServer {

	var privateKey *rsa.PrivateKey
	var err error
	if cfg.CryptoKey != "" {
		privateKey, err = utils.LoadPrivateKey(cfg.CryptoKey)
		if err != nil {
			logger.Errorf("Error loading private key: %v", err)
		}
	}
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpcAuthInterceptor(cfg.Key, logger)),
	)

	server := &GRPCServer{
		server:     grpcServer,
		service:    service,
		cfg:        cfg,
		logger:     logger,
		privateKey: privateKey,
	}

	proto.RegisterMetricsServiceServer(grpcServer, server)
	return server
}

func (s *GRPCServer) Start() error {
	addr := s.cfg.Address
	if addr == "" {
		addr = ":8080"
	}

	grpcAddr := replacePort(addr, 8090)

	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.logger.Infof("gRPC server starting on %s", grpcAddr)
	return s.server.Serve(listener)
}

func (s *GRPCServer) Shutdown() {
	s.server.GracefulStop()
}

func (s *GRPCServer) UpdateMetric(ctx context.Context, req *proto.UpdateMetricRequest) (*proto.UpdateMetricResponse, error) {
	if req.Metric == nil {
		return nil, status.Error(codes.InvalidArgument, "metric is required")
	}

	metric := models.Metrics{
		ID:    req.Metric.Id,
		MType: req.Metric.Mtype,
	}

	switch req.Metric.Mtype {
	case models.Gauge:
		if req.Metric.Value == nil {
			return nil, status.Error(codes.InvalidArgument, "gauge value is required")
		}
		value := req.Metric.GetGauge()
		metric.Value = &value
	case models.Counter:
		if req.Metric.Value == nil {
			return nil, status.Error(codes.InvalidArgument, "counter delta is required")
		}
		delta := req.Metric.GetDelta()
		metric.Delta = &delta
	default:
		return nil, status.Error(codes.InvalidArgument, "invalid metric type")
	}

	result, err := s.service.UpdateMetricJSON(ctx, metric)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	responseMetric := &proto.Metric{
		Id:    result.ID,
		Mtype: result.MType,
	}

	if result.Value != nil {
		responseMetric.Value = &proto.Metric_Gauge{Gauge: *result.Value}
	}
	if result.Delta != nil {
		responseMetric.Value = &proto.Metric_Delta{Delta: *result.Delta}
	}

	return &proto.UpdateMetricResponse{Metric: responseMetric}, nil
}

func (s *GRPCServer) UpdateMetricsBatch(ctx context.Context, req *proto.UpdateMetricsBatchRequest) (*proto.UpdateMetricsBatchResponse, error) {
	if len(req.Metrics) == 0 {
		return nil, status.Error(codes.InvalidArgument, "metrics batch is empty")
	}

	var metrics []models.Metrics
	for _, protoMetric := range req.Metrics {
		metric := models.Metrics{
			ID:    protoMetric.Id,
			MType: protoMetric.Mtype,
		}

		switch protoMetric.Mtype {
		case models.Gauge:
			if protoMetric.Value == nil {
				return nil, status.Error(codes.InvalidArgument, "gauge value is required")
			}
			value := protoMetric.GetGauge()
			metric.Value = &value
		case models.Counter:
			if protoMetric.Value == nil {
				return nil, status.Error(codes.InvalidArgument, "counter delta is required")
			}
			delta := protoMetric.GetDelta()
			metric.Delta = &delta
		default:
			return nil, status.Error(codes.InvalidArgument, "invalid metric type")
		}
		metrics = append(metrics, metric)
	}

	if err := s.service.UpdateMetricsBatch(ctx, metrics); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &proto.UpdateMetricsBatchResponse{}, nil
}

func (s *GRPCServer) GetMetric(ctx context.Context, req *proto.GetMetricRequest) (*proto.GetMetricResponse, error) {
	if req.Id == "" || req.Mtype == "" {
		return nil, status.Error(codes.InvalidArgument, "id and mtype are required")
	}

	metric := models.Metrics{
		ID:    req.Id,
		MType: req.Mtype,
	}

	result, err := s.service.GetMetricJSON(ctx, metric)
	if err != nil {
		if err == models.ErrMetricNotFound {
			return nil, status.Error(codes.NotFound, "metric not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	responseMetric := &proto.Metric{
		Id:    result.ID,
		Mtype: result.MType,
	}

	if result.Value != nil {
		responseMetric.Value = &proto.Metric_Gauge{Gauge: *result.Value}
	}
	if result.Delta != nil {
		responseMetric.Value = &proto.Metric_Delta{Delta: *result.Delta}
	}

	return &proto.GetMetricResponse{Metric: responseMetric}, nil
}

func (s *GRPCServer) Ping(ctx context.Context, req *proto.PingRequest) (*proto.PingResponse, error) {
	dsn := req.Dsn
	if dsn == "" {
		dsn = s.cfg.DatabaseDSN
	}

	if err := s.service.CheckDB(ctx, dsn); err != nil {
		return &proto.PingResponse{Success: false, Error: err.Error()}, nil
	}

	return &proto.PingResponse{Success: true}, nil
}

func replacePort(addr string, newPort int) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {

		return fmt.Sprintf(":%d", newPort)
	}
	return fmt.Sprintf("%s:%d", host, newPort)
}

func grpcAuthInterceptor(key string, logger *logrus.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if key != "" {
			logger.Debug("gRPC auth interceptor: key required but not implemented")
		}
		return handler(ctx, req)
	}
}
