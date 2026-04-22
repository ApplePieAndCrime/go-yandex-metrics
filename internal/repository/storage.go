package repository

import models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"

type StorageRepository struct {
	Storage *models.MemStorage
}

func NewStorageRepository(storage *models.MemStorage) *StorageRepository {
	return &StorageRepository{Storage: storage}
}

func (s *StorageRepository) AddMetrics(metrics models.Metrics) {
	s.Storage.MetricsList = append(s.Storage.MetricsList, metrics)
}

func (s *StorageRepository) GetMetricsByID(id string, mType string) (*models.Metrics, bool) {
	for i, m := range s.Storage.MetricsList {
		if id == m.ID && mType == m.MType {
			return &s.Storage.MetricsList[i], true
		}
	}
	return nil, false
}

func (s *StorageRepository) GetAllMetrics() []models.Metrics {
	return s.Storage.MetricsList
}

func (s *StorageRepository) NewMetrics(metricName string, metricType string, delta int64, value float64) *models.Metrics {
	m := models.Metrics{
		ID:    metricName,
		MType: metricType,
	}
	switch metricType {
	case models.Counter:
		m.Delta = &delta
	case models.Gauge:
		m.Value = &value
	}

	return &m
}
