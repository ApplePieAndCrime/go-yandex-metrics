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
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	// массив вида ["update", "counter", "test", "100"]
	paths := strings.Split(strings.Trim(req.URL.Path, "/"), "/")

	if len(paths) != 4 || paths[0] != "update" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	fmt.Printf("paths: %v\r\n", paths)

	metricType := paths[1]
	metricName := paths[2]
	metricValue := paths[3]

	fmt.Printf("metricType: %s metricName: %s metricValue: %s\r\n", metricType, metricName, metricValue)

	if metricName == "" || metricValue == "" {
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

	existingMetric, exists := storage.GetMetricsByID(metricName)
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
			existingMetric.Value = &value
			fmt.Printf("Gauge value: %d\r\n", *existingMetric.Value)
		}

	} else {
		newMetrics := models.NewMetrics(metricName, metricType, delta, value)
		storage.AddMetrics(newMetrics)
	}

	fmt.Printf("storage: %+v\r\n", storage.MetricsList)

	w.WriteHeader(http.StatusOK)
}
