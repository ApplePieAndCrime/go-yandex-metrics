package service

import (
	"fmt"

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

func (s *Service) UpdateMetrics(existingMetric *models.Metrics, delta int64, value float64) {
	switch existingMetric.MType {
	case models.Counter:
		if existingMetric.Delta == nil {
			existingMetric.Delta = &delta
		} else {
			*existingMetric.Delta += delta
		}
		fmt.Printf("Counter value: %d\r\n", *existingMetric.Delta)
	case models.Gauge:
		if existingMetric.Value == nil {
			existingMetric.Value = &value
		} else {
			*existingMetric.Value = value
		}
		fmt.Printf("Gauge value: %f\r\n", *existingMetric.Value)
	}
}

func (s *Service) CreateMetrics(metricName string, metricType string, delta int64, value float64) models.Metrics {
	newMetrics := s.repository.NewMetrics(metricName, metricType, delta, value)
	s.repository.Storage.AddMetrics(newMetrics)
	return newMetrics
}
