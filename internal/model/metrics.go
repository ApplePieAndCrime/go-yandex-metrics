package models

const (
	Counter = "counter"
	Gauge   = "gauge"
)

// NOTE: Не усложняем пример, вводя иерархическую вложенность структур.
// Органичиваясь плоской моделью.
// Delta и Value объявлены через указатели,
// что бы отличать значение "0", от не заданного значения
// и соответственно не кодировать в структуру.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

type MemStorage struct {
	MetricsList []Metrics
}

func (s *MemStorage) AddMetrics(metrics Metrics) {
	s.MetricsList = append(s.MetricsList, metrics)
}

func (s *MemStorage) GetMetricsByID(id string) (Metrics, bool) {
	for _, m := range s.MetricsList {
		if m.ID == id {
			return m, true
		}
	}
	return Metrics{}, false
}

func NewMetrics(metricName string, metricType string, delta int64, value float64) Metrics {
	return Metrics{
		ID:    metricName,
		MType: metricType,
		Delta: &delta,
		Value: &value,
	}
}
