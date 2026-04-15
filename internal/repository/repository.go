package repository

import models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"

type Storage interface {
	GetMetricsByID(id string, mType string) (*models.Metrics, bool)
	AddMetrics(metrics models.Metrics)
	GetAllMetrics() []models.Metrics
	NewMetrics(metricName string, metricType string, delta int64, value float64) models.Metrics
}

type Repository struct {
	Storage
}

func Init() *Repository {
	storage := &models.MemStorage{
		MetricsList: []models.Metrics{},
	}
	return &Repository{Storage: NewStorageRepository(storage)}
}
