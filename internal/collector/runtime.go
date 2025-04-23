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
		{Name: "Alloc", Type: models.Gauge, Value: float64(stats.Alloc)},
		{Name: "BuckHashSys", Type: models.Gauge, Value: float64(stats.BuckHashSys)},
		{Name: "Frees", Type: models.Gauge, Value: float64(stats.Frees)},
		{Name: "GCCPUFraction", Type: models.Gauge, Value: stats.GCCPUFraction},
		{Name: "GCSys", Type: models.Gauge, Value: float64(stats.GCSys)},
		{Name: "HeapAlloc", Type: models.Gauge, Value: float64(stats.HeapAlloc)},
		{Name: "HeapIdle", Type: models.Gauge, Value: float64(stats.HeapIdle)},
		{Name: "HeapInuse", Type: models.Gauge, Value: float64(stats.HeapInuse)},
		{Name: "HeapObjects", Type: models.Gauge, Value: float64(stats.HeapObjects)},
		{Name: "HeapReleased", Type: models.Gauge, Value: float64(stats.HeapReleased)},
		{Name: "HeapSys", Type: models.Gauge, Value: float64(stats.HeapSys)},
		{Name: "LastGC", Type: models.Gauge, Value: float64(stats.LastGC)},
		{Name: "Lookups", Type: models.Gauge, Value: float64(stats.Lookups)},
		{Name: "MCacheInuse", Type: models.Gauge, Value: float64(stats.MCacheInuse)},
		{Name: "MCacheSys", Type: models.Gauge, Value: float64(stats.MCacheSys)},
		{Name: "MSpanInuse", Type: models.Gauge, Value: float64(stats.MSpanInuse)},
		{Name: "MSpanSys", Type: models.Gauge, Value: float64(stats.MSpanSys)},
		{Name: "Mallocs", Type: models.Gauge, Value: float64(stats.Mallocs)},
		{Name: "NextGC", Type: models.Gauge, Value: float64(stats.NextGC)},
		{Name: "NumForcedGC", Type: models.Gauge, Value: float64(stats.NumForcedGC)},
		{Name: "NumGC", Type: models.Gauge, Value: float64(stats.NumGC)},
		{Name: "OtherSys", Type: models.Gauge, Value: float64(stats.OtherSys)},
		{Name: "PauseTotalNs", Type: models.Gauge, Value: float64(stats.PauseTotalNs)},
		{Name: "StackInuse", Type: models.Gauge, Value: float64(stats.StackInuse)},
		{Name: "StackSys", Type: models.Gauge, Value: float64(stats.StackSys)},
		{Name: "Sys", Type: models.Gauge, Value: float64(stats.Sys)},
		{Name: "TotalAlloc", Type: models.Gauge, Value: float64(stats.TotalAlloc)},
		{Name: "RandomValue", Type: models.Gauge, Value: rand.Float64()},
	}

	c.pollCount++
	metrics = append(metrics, models.Metric{
		Name:  "PollCount",
		Type:  models.Counter,
		Value: c.pollCount,
	})

	return metrics
}
