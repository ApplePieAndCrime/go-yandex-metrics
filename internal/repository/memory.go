package repository

import (
	"context"
	"sync"

	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
)

type MemoryStorage struct {
	metrics []models.Metrics
	index   map[metricKey]int
	mu      sync.RWMutex
}

type metricKey struct {
	id    string
	mType string
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		metrics: make([]models.Metrics, 0, 2048),
		index:   make(map[metricKey]int, 2048),
	}
}

func (s *MemoryStorage) GetMetricsByID(_ context.Context, id string, mType string) (*models.Metrics, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	i, ok := s.index[metricKey{id: id, mType: mType}]
	if !ok {
		return nil, false, nil
	}

	return &s.metrics[i], true, nil
}

func (s *MemoryStorage) GetAllMetrics(_ context.Context) ([]models.Metrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metricsCopy := make([]models.Metrics, len(s.metrics))
	copy(metricsCopy, s.metrics)
	return metricsCopy, nil
}

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
		return &s.metrics[i], nil
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
	return &s.metrics[len(s.metrics)-1], nil
}
