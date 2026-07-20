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
	gaugeNames := benchmarkMetricNames("gauge")
	counterNames := benchmarkMetricNames("counter")
	b.ReportAllocs()

	for b.Loop() {
		storage := NewMemoryStorage()

		for j := 0; j < benchmarkMetricsCount; j++ {
			value := float64(j)
			_, err := storage.SaveMetrics(ctx, models.Metrics{
				ID:    gaugeNames[j],
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
				ID:    counterNames[j],
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
				ID:    counterNames[j],
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
	gaugeNames := benchmarkMetricNames("gauge")

	for j := 0; j < benchmarkMetricsCount; j++ {
		value := float64(j)
		_, err := storage.SaveMetrics(ctx, models.Metrics{
			ID:    gaugeNames[j],
			MType: models.Gauge,
			Value: &value,
		})
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ReportAllocs()

	i := 0
	for b.Loop() {
		id := gaugeNames[i%benchmarkMetricsCount]
		metric, exists, err := storage.GetMetricsByID(ctx, id, models.Gauge)
		if err != nil {
			b.Fatal(err)
		}
		if !exists || metric.ID != id {
			b.Fatalf("metric %s not found", id)
		}
		i++
	}
}

func benchmarkMetricNames(prefix string) []string {
	names := make([]string, benchmarkMetricsCount)
	for i := range names {
		names[i] = fmt.Sprintf("%s_%d", prefix, i)
	}

	return names
}
