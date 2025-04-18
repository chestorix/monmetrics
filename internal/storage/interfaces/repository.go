package interfaces

type MetricsRepository interface {
	UpdateGauge(name string, value float64)
	UpdateCounter(name string, value int64)
	//GetGauge(name string) (float64, error)
	//GetCounter(name string) (int64, error)
	//GetAllMetrics() ([]models.Metrics, error)
}
