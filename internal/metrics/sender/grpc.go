package sender

import (
	"context"
	"time"

	models "github.com/chestorix/monmetrics/internal/metrics"
	"github.com/chestorix/monmetrics/internal/proto"
	"github.com/chestorix/monmetrics/internal/utils"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCSender struct {
	client      proto.MetricsServiceClient
	conn        *grpc.ClientConn
	baseURL     string
	retryDelays []time.Duration
	logger      *logrus.Logger
}

func NewGRPCSender(baseURL string, logger *logrus.Logger) *GRPCSender {
	return &GRPCSender{
		baseURL:     baseURL,
		retryDelays: []time.Duration{time.Second, 3 * time.Second, 5 * time.Second},
		logger:      logger,
	}
}

func (g *GRPCSender) Connect(ctx context.Context) error {
	return utils.Retry(3, g.retryDelays, func() error {
		conn, err := grpc.DialContext(ctx, g.baseURL,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
			grpc.WithTimeout(5*time.Second),
		)
		if err != nil {
			return err
		}
		g.conn = conn
		g.client = proto.NewMetricsServiceClient(conn)
		return nil
	})
}

func (g *GRPCSender) Close() error {
	if g.conn != nil {
		return g.conn.Close()
	}
	return nil
}

func (g *GRPCSender) SendBatch(metrics []models.Metrics) error {
	return utils.Retry(3, g.retryDelays, func() error {
		var protoMetrics []*proto.Metric

		for _, m := range metrics {
			protoMetric := &proto.Metric{
				Id:    m.ID,
				Mtype: m.MType,
			}

			switch m.MType {
			case models.Gauge:
				if m.Value != nil {
					protoMetric.Value = &proto.Metric_Gauge{Gauge: *m.Value}
				}
			case models.Counter:
				if m.Delta != nil {
					protoMetric.Value = &proto.Metric_Delta{Delta: *m.Delta}
				}
			}
			protoMetrics = append(protoMetrics, protoMetric)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := g.client.UpdateMetricsBatch(ctx, &proto.UpdateMetricsBatchRequest{
			Metrics: protoMetrics,
		})

		return err
	})
}
