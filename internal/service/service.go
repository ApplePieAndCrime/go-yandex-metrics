package service

import (
	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/repository"
)

type Service struct {
	repository *repository.Repository
}

func NewService(repository *repository.Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) GetAllMetrics() []models.Metrics {
	return s.repository.Storage.GetAllMetrics()
}

func (s *Service) GetMetricsByID(id string, mType string) (*models.Metrics, bool) {
	return s.repository.Storage.GetMetricsByID(id, mType)
}

func (s *Service) UpdateMetrics(existingMetric *models.Metrics, delta int64, value float64) *models.Metrics {
	switch existingMetric.MType {
	case models.Counter:
		if existingMetric.Delta == nil {
			existingMetric.Delta = &delta
		} else {
			*existingMetric.Delta += delta
		}
	case models.Gauge:
		if existingMetric.Value == nil {
			existingMetric.Value = &value
		} else {
			*existingMetric.Value = value
		}
	}
	return existingMetric
}

func (s *Service) CreateOrUpdateMetrics(metrics models.Metrics) *models.Metrics {
	var d int64
	var v float64

	// Безопасно извлекаем значения
	if metrics.Delta != nil {
		d = *metrics.Delta
	}
	if metrics.Value != nil {
		v = *metrics.Value
	}

	existingMetric, exists := s.GetMetricsByID(metrics.ID, metrics.MType)
	if exists {
		return s.UpdateMetrics(existingMetric, d, v)
	} else {
		return s.CreateMetrics(metrics.ID, metrics.MType, d, v)
	}
}

func (s *Service) CreateMetrics(metricName string, metricType string, delta int64, value float64) *models.Metrics {
	newMetrics := s.repository.NewMetrics(metricName, metricType, delta, value)
	s.repository.Storage.AddMetrics(*newMetrics)
	return newMetrics
}
