package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/repository"
)

var storage models.MemStorage

func Init() {
	repositoryResponse := repository.Init()
	storage = repositoryResponse.Storage
	http.HandleFunc("/update/", UpdateMetrics)
}

func UpdateMetrics(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	// массив вида ["", "update", "counter", "test", "100"]
	paths := strings.Split(req.URL.Path, "/")

	if len(paths) != 5 || paths[1] != "update" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	fmt.Printf("paths: %v\r\n", paths)

	metricType := paths[2]
	metricName := paths[3]
	metricValue := paths[4]

	fmt.Printf("metricType: %s metricName: %s metricValue: %s\r\n", metricType, metricName, metricValue)

	if metricName == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var delta int64
	var value float64

	switch metricType {
	case models.Gauge:
		converted, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		value = converted
	case models.Counter:
		converted, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		delta = converted
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	existingMetric, exists := storage.GetMetricsByID(metricName, metricType)
	if exists {
		switch existingMetric.MType {
		case models.Counter:
			if existingMetric.Delta == nil {
				existingMetric.Delta = &delta
			} else {
				*existingMetric.Delta += delta
			}
			fmt.Printf("Counter value: %d\r\n", *existingMetric.Delta)
		case models.Gauge:
			if existingMetric.Value == nil {
				existingMetric.Value = &value
			} else {
				*existingMetric.Value = value
			}
			fmt.Printf("Gauge value: %f\r\n", *existingMetric.Value)
		}

	} else {
		newMetrics := models.NewMetrics(metricName, metricType, delta, value)
		storage.AddMetrics(newMetrics)
	}

	fmt.Printf("storage: %+v\r\n", storage.MetricsList)

	w.WriteHeader(http.StatusOK)
}
