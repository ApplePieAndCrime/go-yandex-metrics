package service

import (
	"context"

	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/repository"
)

// Service содержит прикладные операции над метриками.
type Service struct {
	storage repository.Storage
}

// NewService создаёт сервис с указанным хранилищем.
func NewService(storage repository.Storage) *Service {
	return &Service{storage: storage}
}

// GetAllMetrics возвращает все сохранённые метрики.
func (s *Service) GetAllMetrics() ([]models.Metrics, error) {
	return s.storage.GetAllMetrics(context.Background())
}

// GetMetricsByID возвращает метрику по имени и типу.
func (s *Service) GetMetricsByID(id string, mType string) (*models.Metrics, bool, error) {
	return s.storage.GetMetricsByID(context.Background(), id, mType)
}

// UpdateMetrics обновляет существующую метрику переданным значением.
func (s *Service) UpdateMetrics(existingMetric *models.Metrics, delta int64, value float64) (*models.Metrics, error) {
	updated := models.Metrics{
		ID:    existingMetric.ID,
		MType: existingMetric.MType,
	}

	switch existingMetric.MType {
	case models.Counter:
		updated.Delta = &delta
	case models.Gauge:
		updated.Value = &value
	}

	return s.storage.SaveMetrics(context.Background(), updated)
}

// CreateOrUpdateMetrics сохраняет готовую модель метрики с учётом отмены операции.
func (s *Service) CreateOrUpdateMetrics(ctx context.Context, metrics models.Metrics) (*models.Metrics, error) {
	return s.storage.SaveMetrics(ctx, metrics)
}

// CreateOrUpdateMetricsBatch атомарно сохраняет пакет метрик.
func (s *Service) CreateOrUpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) ([]models.Metrics, error) {
	return s.storage.SaveMetricsBatch(ctx, metrics)
}

// CreateMetrics создаёт метрику указанного имени и типа.
func (s *Service) CreateMetrics(metricName string, metricType string, delta int64, value float64) (*models.Metrics, error) {
	newMetrics := models.Metrics{
		ID:    metricName,
		MType: metricType,
	}

	switch metricType {
	case models.Counter:
		newMetrics.Delta = &delta
	case models.Gauge:
		newMetrics.Value = &value
	}

	savedMetrics, err := s.storage.SaveMetrics(context.Background(), newMetrics)
	if err != nil {
		return nil, err
	}

	return savedMetrics, nil
}
