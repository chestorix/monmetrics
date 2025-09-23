package sender

import models "github.com/chestorix/monmetrics/internal/metrics"

type MetricSender interface {
	SendBatch(metrics []models.Metrics) error
	Close() error
}
