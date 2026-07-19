package repository

import (
	"context"
	"fmt"
	"testing"

	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
)

const benchmarkMetricsCount = 1000

func BenchmarkMemoryStorageSaveMetrics(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		storage := NewMemoryStorage()

		for j := 0; j < benchmarkMetricsCount; j++ {
			value := float64(j)
			_, err := storage.SaveMetrics(ctx, models.Metrics{
				ID:    fmt.Sprintf("gauge_%d", j),
				MType: models.Gauge,
				Value: &value,
			})
			if err != nil {
				b.Fatal(err)
			}
		}

		for j := 0; j < benchmarkMetricsCount; j++ {
			delta := int64(j)
			_, err := storage.SaveMetrics(ctx, models.Metrics{
				ID:    fmt.Sprintf("counter_%d", j),
				MType: models.Counter,
				Delta: &delta,
			})
			if err != nil {
				b.Fatal(err)
			}
		}

		for j := 0; j < benchmarkMetricsCount; j++ {
			delta := int64(1)
			_, err := storage.SaveMetrics(ctx, models.Metrics{
				ID:    fmt.Sprintf("counter_%d", j),
				MType: models.Counter,
				Delta: &delta,
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkMemoryStorageGetMetricsByID(b *testing.B) {
	ctx := context.Background()
	storage := NewMemoryStorage()

	for j := 0; j < benchmarkMetricsCount; j++ {
		value := float64(j)
		_, err := storage.SaveMetrics(ctx, models.Metrics{
			ID:    fmt.Sprintf("gauge_%d", j),
			MType: models.Gauge,
			Value: &value,
		})
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		id := fmt.Sprintf("gauge_%d", i%benchmarkMetricsCount)
		metric, exists, err := storage.GetMetricsByID(ctx, id, models.Gauge)
		if err != nil {
			b.Fatal(err)
		}
		if !exists || metric.ID != id {
			b.Fatalf("metric %s not found", id)
		}
	}
}
