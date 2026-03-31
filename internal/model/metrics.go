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

type Storage interface {
	AddMetrics(metrics Metrics)
	GetMetricsByID(id string, mType string) (*Metrics, bool)
	GetAllMetrics() []Metrics
}

type MemStorage struct {
	MetricsList []Metrics
}

func (s *MemStorage) AddMetrics(metrics Metrics) {
	s.MetricsList = append(s.MetricsList, metrics)
}

func (s *MemStorage) GetMetricsByID(id string, mType string) (*Metrics, bool) {
	for i, m := range s.MetricsList {
		if id == m.ID && mType == m.MType {
			return &s.MetricsList[i], true
		}
	}
	return nil, false
}

func (s *MemStorage) GetAllMetrics() []Metrics {
	return s.MetricsList
}

func NewMetrics(metricName string, metricType string, delta int64, value float64) Metrics {
	m := Metrics{
		ID:    metricName,
		MType: metricType,
	}
	switch metricType {
	case Counter:
		m.Delta = &delta
	case Gauge:
		m.Value = &value
	}

	return m
}
