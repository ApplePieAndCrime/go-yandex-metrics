package repository

import (
	"context"

	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
)

type MemoryStorage struct {
	metrics []models.Metrics
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{metrics: []models.Metrics{}}
}

func (s *MemoryStorage) GetMetricsByID(_ context.Context, id string, mType string) (*models.Metrics, bool, error) {
	for i, m := range s.metrics {
		if id == m.ID && mType == m.MType {
			return &s.metrics[i], true, nil
		}
	}
	return nil, false, nil
}

func (s *MemoryStorage) GetAllMetrics(_ context.Context) ([]models.Metrics, error) {
	return s.metrics, nil
}

func (s *MemoryStorage) SaveMetrics(_ context.Context, metrics models.Metrics) (*models.Metrics, error) {
	for i := range s.metrics {
		if s.metrics[i].ID == metrics.ID && s.metrics[i].MType == metrics.MType {
			if metrics.MType == models.Counter && metrics.Delta != nil {
				if s.metrics[i].Delta == nil {
					s.metrics[i].Delta = metrics.Delta
				} else {
					*s.metrics[i].Delta += *metrics.Delta
				}
			}
			if metrics.MType == models.Gauge && metrics.Value != nil {
				s.metrics[i].Value = metrics.Value
			}
			return &s.metrics[i], nil
		}
	}

	s.metrics = append(s.metrics, metrics)
	return &s.metrics[len(s.metrics)-1], nil
}
