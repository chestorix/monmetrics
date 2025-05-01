package metrics

import "errors"

var (
	ErrMetricNotFound    = errors.New("metric not found")
	ErrInvalidMetricType = errors.New("invalid metric type")
)

const (
	Gauge   = "gauge"
	Counter = "counter"
)

type Metric struct {
	Name  string
	Type  string
	Value interface{}
}
