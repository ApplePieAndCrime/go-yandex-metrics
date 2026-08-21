package repository

import (
	"context"
	"testing"

	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorageReturnsIndependentMetric(t *testing.T) {
	storage := NewMemoryStorage()
	delta := int64(1)

	first, err := storage.SaveMetrics(context.Background(), models.Metrics{
		ID:    "requests",
		MType: models.Counter,
		Delta: &delta,
	})
	require.NoError(t, err)

	_, err = storage.SaveMetrics(context.Background(), models.Metrics{
		ID:    "requests",
		MType: models.Counter,
		Delta: &delta,
	})
	require.NoError(t, err)

	require.NotNil(t, first.Delta)
	assert.Equal(t, int64(1), *first.Delta)
}

func TestMemoryStorageGetAllMetricsReturnsDeepCopy(t *testing.T) {
	storage := NewMemoryStorage()
	value := 10.5
	_, err := storage.SaveMetrics(context.Background(), models.Metrics{
		ID:    "temperature",
		MType: models.Gauge,
		Value: &value,
	})
	require.NoError(t, err)

	metrics, err := storage.GetAllMetrics(context.Background())
	require.NoError(t, err)
	require.Len(t, metrics, 1)
	*metrics[0].Value = 99

	stored, exists, err := storage.GetMetricsByID(context.Background(), "temperature", models.Gauge)
	require.NoError(t, err)
	require.True(t, exists)
	assert.Equal(t, 10.5, *stored.Value)
}

func TestMemoryStorageSaveMetricsBatchIsAtomic(t *testing.T) {
	storage := NewMemoryStorage()
	value := 10.5

	_, err := storage.SaveMetricsBatch(context.Background(), []models.Metrics{
		{ID: "temperature", MType: models.Gauge, Value: &value},
		{ID: "requests", MType: models.Counter},
	})
	require.Error(t, err)

	metrics, err := storage.GetAllMetrics(context.Background())
	require.NoError(t, err)
	assert.Empty(t, metrics)
}

func TestMemoryStorageSaveMetricsBatchUpdatesAllMetrics(t *testing.T) {
	storage := NewMemoryStorage()
	value := 10.5
	delta := int64(3)

	saved, err := storage.SaveMetricsBatch(context.Background(), []models.Metrics{
		{ID: "temperature", MType: models.Gauge, Value: &value},
		{ID: "requests", MType: models.Counter, Delta: &delta},
	})
	require.NoError(t, err)
	require.Len(t, saved, 2)

	metrics, err := storage.GetAllMetrics(context.Background())
	require.NoError(t, err)
	assert.Len(t, metrics, 2)
}
