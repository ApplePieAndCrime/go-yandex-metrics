package repository

import (
	"context"
	"fmt"
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
func (s *MemoryStorage) SaveMetrics(ctx context.Context, metrics models.Metrics) (*models.Metrics, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateMetric(metrics); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	saved := s.saveMetricsLocked(metrics)
	return &saved, nil
}

// SaveMetricsBatch атомарно сохраняет пакет метрик под одной блокировкой.
func (s *MemoryStorage) SaveMetricsBatch(ctx context.Context, metrics []models.Metrics) ([]models.Metrics, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	for _, metric := range metrics {
		if err := validateMetric(metric); err != nil {
			return nil, err
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	saved := make([]models.Metrics, 0, len(metrics))
	for _, metric := range metrics {
		saved = append(saved, s.saveMetricsLocked(metric))
	}
	return saved, nil
}

func (s *MemoryStorage) saveMetricsLocked(metrics models.Metrics) models.Metrics {
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
		return cloneMetric(s.metrics[i])
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
	return cloneMetric(s.metrics[len(s.metrics)-1])
}

func validateMetric(metric models.Metrics) error {
	if metric.ID == "" {
		return fmt.Errorf("metric id is required")
	}
	switch metric.MType {
	case models.Counter:
		if metric.Delta == nil {
			return fmt.Errorf("counter %q has no delta", metric.ID)
		}
	case models.Gauge:
		if metric.Value == nil {
			return fmt.Errorf("gauge %q has no value", metric.ID)
		}
	default:
		return fmt.Errorf("unsupported metric type %q", metric.MType)
	}
	return nil
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
