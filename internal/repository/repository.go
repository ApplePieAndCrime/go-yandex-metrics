package repository

import (
	"context"

	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
)

type Storage interface {
	GetMetricsByID(ctx context.Context, id string, mType string) (*models.Metrics, bool, error)
	GetAllMetrics(ctx context.Context) ([]models.Metrics, error)
	SaveMetrics(ctx context.Context, metrics models.Metrics) (*models.Metrics, error)
}

type Repository struct {
	Storage
}
