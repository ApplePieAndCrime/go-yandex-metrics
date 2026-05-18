package repository

import (
	"context"
	"sync"

	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
)

type MemoryStorage struct {
	metrics []models.Metrics
	mu      sync.RWMutex
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{metrics: []models.Metrics{}}
}

func (s *MemoryStorage) GetMetricsByID(_ context.Context, id string, mType string) (*models.Metrics, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i, m := range s.metrics {
		if id == m.ID && mType == m.MType {
			metricCopy := s.metrics[i]
			return &metricCopy, true, nil
		}
	}
	return nil, false, nil
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

	for i := range s.metrics {
		if s.metrics[i].ID == metrics.ID && s.metrics[i].MType == metrics.MType {
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
			metricCopy := s.metrics[i]
			return &metricCopy, nil
		}
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
	metricCopy := s.metrics[len(s.metrics)-1]
	return &metricCopy, nil
}
