package models

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

type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}
