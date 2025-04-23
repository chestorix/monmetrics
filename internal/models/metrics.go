package models

import "errors"

var (
	ErrMetricNotFound    = errors.New("Metric not found")
	ErrInvalidMetricType = errors.New("Invalid metric type")
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
