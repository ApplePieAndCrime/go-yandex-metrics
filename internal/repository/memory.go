package repository

import (
	"context"
	"sync"

	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
)

// MemoryStorage хранит метрики в памяти и безопасно работает при конкурентном доступе.
type MemoryStorage struct {
	metrics []models.Metrics
	index   map[metricKey]int
	mu      sync.RWMutex
}

type metricKey struct {
	id    string
	mType string
}

// NewMemoryStorage создаёт пустое хранилище метрик в памяти.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		metrics: make([]models.Metrics, 0, 2048),
		index:   make(map[metricKey]int, 2048),
	}
}

// GetMetricsByID возвращает метрику по имени и типу, а также признак её наличия.
func (s *MemoryStorage) GetMetricsByID(_ context.Context, id string, mType string) (*models.Metrics, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	i, ok := s.index[metricKey{id: id, mType: mType}]
	if !ok {
		return nil, false, nil
	}

	metricCopy := cloneMetric(s.metrics[i])
	return &metricCopy, true, nil
}

// GetAllMetrics возвращает копию всех сохранённых метрик.
func (s *MemoryStorage) GetAllMetrics(_ context.Context) ([]models.Metrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metricsCopy := make([]models.Metrics, len(s.metrics))
	for i := range s.metrics {
		metricsCopy[i] = cloneMetric(s.metrics[i])
	}
	return metricsCopy, nil
}

// SaveMetrics сохраняет метрику, суммируя счётчик или заменяя измеритель.
func (s *MemoryStorage) SaveMetrics(_ context.Context, metrics models.Metrics) (*models.Metrics, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := metricKey{id: metrics.ID, mType: metrics.MType}
	if i, ok := s.index[key]; ok {
		if metrics.MType == models.Counter && metrics.Delta != nil {
			if s.metrics[i].Delta == nil {
				deltaCopy := *metrics.Delta
				s.metrics[i].Delta = &deltaCopy
			} else {
				*s.metrics[i].Delta += *metrics.Delta
			}
		}
		if metrics.MType == models.Gauge && metrics.Value != nil {
			valueCopy := *metrics.Value
			s.metrics[i].Value = &valueCopy
		}
		metricCopy := cloneMetric(s.metrics[i])
		return &metricCopy, nil
	}

	if metrics.Delta != nil {
		deltaCopy := *metrics.Delta
		metrics.Delta = &deltaCopy
	}
	if metrics.Value != nil {
		valueCopy := *metrics.Value
		metrics.Value = &valueCopy
	}

	s.metrics = append(s.metrics, metrics)
	s.index[key] = len(s.metrics) - 1
	metricCopy := cloneMetric(s.metrics[len(s.metrics)-1])
	return &metricCopy, nil
}

func cloneMetric(metric models.Metrics) models.Metrics {
	if metric.Delta != nil {
		delta := *metric.Delta
		metric.Delta = &delta
	}
	if metric.Value != nil {
		value := *metric.Value
		metric.Value = &value
	}

	return metric
}
