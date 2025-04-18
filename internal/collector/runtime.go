package collector

import (
	"github.com/chestorix/monmetrics/internal/models"
	"math/rand"
	"runtime"
)

type RuntimeCollector struct {
	pollCount int64
}

func NewRuntimeCollector() *RuntimeCollector {
	return &RuntimeCollector{}
}

func (c *RuntimeCollector) Collect() []models.Metric {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	metrics := []models.Metric{
		{Name: "Alloc", Type: "gauge", Value: float64(stats.Alloc)},
		{Name: "BuckHashSys", Type: "gauge", Value: float64(stats.BuckHashSys)},
		{Name: "Frees", Type: "gauge", Value: float64(stats.Frees)},
		{Name: "GCCPUFraction", Type: "gauge", Value: stats.GCCPUFraction},
		{Name: "GCSys", Type: "gauge", Value: float64(stats.GCSys)},
		{Name: "HeapAlloc", Type: "gauge", Value: float64(stats.HeapAlloc)},
		{Name: "HeapIdle", Type: "gauge", Value: float64(stats.HeapIdle)},
		{Name: "HeapInuse", Type: "gauge", Value: float64(stats.HeapInuse)},
		{Name: "HeapObjects", Type: "gauge", Value: float64(stats.HeapObjects)},
		{Name: "HeapReleased", Type: "gauge", Value: float64(stats.HeapReleased)},
		{Name: "HeapSys", Type: "gauge", Value: float64(stats.HeapSys)},
		{Name: "LastGC", Type: "gauge", Value: float64(stats.LastGC)},
		{Name: "Lookups", Type: "gauge", Value: float64(stats.Lookups)},
		{Name: "MCacheInuse", Type: "gauge", Value: float64(stats.MCacheInuse)},
		{Name: "MCacheSys", Type: "gauge", Value: float64(stats.MCacheSys)},
		{Name: "MSpanInuse", Type: "gauge", Value: float64(stats.MSpanInuse)},
		{Name: "MSpanSys", Type: "gauge", Value: float64(stats.MSpanSys)},
		{Name: "Mallocs", Type: "gauge", Value: float64(stats.Mallocs)},
		{Name: "NextGC", Type: "gauge", Value: float64(stats.NextGC)},
		{Name: "NumForcedGC", Type: "gauge", Value: float64(stats.NumForcedGC)},
		{Name: "NumGC", Type: "gauge", Value: float64(stats.NumGC)},
		{Name: "OtherSys", Type: "gauge", Value: float64(stats.OtherSys)},
		{Name: "PauseTotalNs", Type: "gauge", Value: float64(stats.PauseTotalNs)},
		{Name: "StackInuse", Type: "gauge", Value: float64(stats.StackInuse)},
		{Name: "StackSys", Type: "gauge", Value: float64(stats.StackSys)},
		{Name: "Sys", Type: "gauge", Value: float64(stats.Sys)},
		{Name: "TotalAlloc", Type: "gauge", Value: float64(stats.TotalAlloc)},
		{Name: "RandomValue", Type: "gauge", Value: rand.Float64()},
	}

	c.pollCount++
	metrics = append(metrics, models.Metric{
		Name:  "PollCount",
		Type:  "counter",
		Value: c.pollCount,
	})

	return metrics
}
