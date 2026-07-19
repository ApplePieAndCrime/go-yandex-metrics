package models

const (
	// Counter обозначает тип метрики-счётчика, значение которой накапливается.
	Counter = "counter"
	// Gauge обозначает тип метрики-измерителя, значение которой заменяется при обновлении.
	Gauge = "gauge"
)

// Metrics описывает метрику, передаваемую между агентом, сервером и хранилищем.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

// MemStorage представляет сериализуемый список метрик в памяти.
type MemStorage struct {
	MetricsList []Metrics
}
