package collector

import (
	"github.com/chestorix/monmetrics/internal/metrics"
	"math/rand"
	"runtime"
)

type RuntimeCollector struct {
	pollCount int64
}

func NewRuntimeCollector() *RuntimeCollector {
	return &RuntimeCollector{}
}

func (c *RuntimeCollector) Collect() []metrics.Metric {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	metric := []metrics.Metric{
		{Name: "Alloc", Type: metrics.Gauge, Value: float64(stats.Alloc)},
		{Name: "BuckHashSys", Type: metrics.Gauge, Value: float64(stats.BuckHashSys)},
		{Name: "Frees", Type: metrics.Gauge, Value: float64(stats.Frees)},
		{Name: "GCCPUFraction", Type: metrics.Gauge, Value: stats.GCCPUFraction},
		{Name: "GCSys", Type: metrics.Gauge, Value: float64(stats.GCSys)},
		{Name: "HeapAlloc", Type: metrics.Gauge, Value: float64(stats.HeapAlloc)},
		{Name: "HeapIdle", Type: metrics.Gauge, Value: float64(stats.HeapIdle)},
		{Name: "HeapInuse", Type: metrics.Gauge, Value: float64(stats.HeapInuse)},
		{Name: "HeapObjects", Type: metrics.Gauge, Value: float64(stats.HeapObjects)},
		{Name: "HeapReleased", Type: metrics.Gauge, Value: float64(stats.HeapReleased)},
		{Name: "HeapSys", Type: metrics.Gauge, Value: float64(stats.HeapSys)},
		{Name: "LastGC", Type: metrics.Gauge, Value: float64(stats.LastGC)},
		{Name: "Lookups", Type: metrics.Gauge, Value: float64(stats.Lookups)},
		{Name: "MCacheInuse", Type: metrics.Gauge, Value: float64(stats.MCacheInuse)},
		{Name: "MCacheSys", Type: metrics.Gauge, Value: float64(stats.MCacheSys)},
		{Name: "MSpanInuse", Type: metrics.Gauge, Value: float64(stats.MSpanInuse)},
		{Name: "MSpanSys", Type: metrics.Gauge, Value: float64(stats.MSpanSys)},
		{Name: "Mallocs", Type: metrics.Gauge, Value: float64(stats.Mallocs)},
		{Name: "NextGC", Type: metrics.Gauge, Value: float64(stats.NextGC)},
		{Name: "NumForcedGC", Type: metrics.Gauge, Value: float64(stats.NumForcedGC)},
		{Name: "NumGC", Type: metrics.Gauge, Value: float64(stats.NumGC)},
		{Name: "OtherSys", Type: metrics.Gauge, Value: float64(stats.OtherSys)},
		{Name: "PauseTotalNs", Type: metrics.Gauge, Value: float64(stats.PauseTotalNs)},
		{Name: "StackInuse", Type: metrics.Gauge, Value: float64(stats.StackInuse)},
		{Name: "StackSys", Type: metrics.Gauge, Value: float64(stats.StackSys)},
		{Name: "Sys", Type: metrics.Gauge, Value: float64(stats.Sys)},
		{Name: "TotalAlloc", Type: metrics.Gauge, Value: float64(stats.TotalAlloc)},
		{Name: "RandomValue", Type: metrics.Gauge, Value: rand.Float64()},
	}

	c.pollCount++
	metric = append(metric, metrics.Metric{
		Name:  "PollCount",
		Type:  metrics.Counter,
		Value: c.pollCount,
	})

	return metric
}
