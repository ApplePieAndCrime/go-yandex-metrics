package repository

import models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"

type RepositoryResponse struct {
	Storage models.MemStorage
}

func Init() RepositoryResponse {
	storage := models.MemStorage{
		MetricsList: []models.Metrics{},
	}
	return RepositoryResponse{Storage: storage}
}
