package service

import (
	"context"

	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/repository"
)

type Service struct {
	storage repository.Storage
}

func NewService(storage repository.Storage) *Service {
	return &Service{storage: storage}
}

func (s *Service) GetAllMetrics() ([]models.Metrics, error) {
	return s.storage.GetAllMetrics(context.Background())
}

func (s *Service) GetMetricsByID(id string, mType string) (*models.Metrics, bool, error) {
	return s.storage.GetMetricsByID(context.Background(), id, mType)
}

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

func (s *Service) CreateOrUpdateMetrics(metrics models.Metrics) (*models.Metrics, error) {
	return s.storage.SaveMetrics(context.Background(), metrics)
}

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
