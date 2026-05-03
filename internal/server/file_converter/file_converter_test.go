package file_converter

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
)

func TestProducerWriteEventReplacesFileAtomically(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filename := filepath.Join(dir, "storage.json")

	producer, err := NewProducer(filename)
	if err != nil {
		t.Fatalf("NewProducer() error = %v", err)
	}

	initialValue := 42.5
	initialMetrics := []models.Metrics{
		{
			ID:    "Alloc",
			MType: models.Gauge,
			Value: &initialValue,
		},
	}

	if err := producer.WriteEvent(initialMetrics); err != nil {
		t.Fatalf("WriteEvent() error = %v", err)
	}

	invalidValue := math.NaN()
	invalidMetrics := []models.Metrics{
		{
			ID:    "Alloc",
			MType: models.Gauge,
			Value: &invalidValue,
		},
	}

	if err := producer.WriteEvent(invalidMetrics); err == nil {
		t.Fatal("WriteEvent() error = nil, want encode error")
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}

	var got []models.Metrics
	if err := json.Unmarshal(content, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}

	if got[0].ID != initialMetrics[0].ID {
		t.Fatalf("got ID = %q, want %q", got[0].ID, initialMetrics[0].ID)
	}

	if got[0].Value == nil || *got[0].Value != initialValue {
		t.Fatalf("got Value = %v, want %v", got[0].Value, initialValue)
	}
}
