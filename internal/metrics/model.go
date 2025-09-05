// Package models  содержит бизнес-сущности приложения.
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
	Value interface{}
	Name  string
	Type  string
}

type Metrics struct {
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	ID    string   `json:"id"`
	MType string   `json:"type"`
}
