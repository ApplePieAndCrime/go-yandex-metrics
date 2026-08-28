package server

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/repository"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveMetricsToFileWithContextFlushesOnShutdown(t *testing.T) {
	storage := repository.NewMemoryStorage()
	services := service.NewService(storage)

	value := 42.5
	_, err := services.CreateOrUpdateMetrics(context.Background(), models.Metrics{
		ID:    "Alloc",
		MType: models.Gauge,
		Value: &value,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	storagePath := filepath.Join(t.TempDir(), "metrics.json")
	done := SaveMetricsToFileWithContext(ctx, *services, 3600, storagePath, false)

	cancel()
	require.NoError(t, <-done)

	content, err := os.ReadFile(storagePath)
	require.NoError(t, err)

	var saved []models.Metrics
	require.NoError(t, json.Unmarshal(content, &saved))
	require.Len(t, saved, 1)
	assert.Equal(t, "Alloc", saved[0].ID)
	require.NotNil(t, saved[0].Value)
	assert.Equal(t, value, *saved[0].Value)
}

func TestSaveMetricsToFileWithZeroIntervalStillFlushesOnShutdown(t *testing.T) {
	storage := repository.NewMemoryStorage()
	services := service.NewService(storage)
	ctx, cancel := context.WithCancel(context.Background())
	storagePath := filepath.Join(t.TempDir(), "metrics.json")

	done := SaveMetricsToFileWithContext(ctx, *services, 0, storagePath, false)
	cancel()

	require.NoError(t, <-done)
	_, err := os.Stat(storagePath)
	require.NoError(t, err)
}
