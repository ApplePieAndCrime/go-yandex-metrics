package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/repository"
	"github.com/go-chi/chi"
)

var storage models.MemStorage

func Init() *chi.Mux {
	repositoryResponse := repository.Init()
	storage = repositoryResponse.Storage
	r := chi.NewRouter()
	r.Route("/", func(r chi.Router) {
		r.Get("/", GetAllMetrics)
		r.Get("/value/{metricType}/{metricName}", GetMetricsByID)
		r.Post("/update/{metricType}/{metricName}/{metricValue}", UpdateMetrics)
	})

	return r
}

func GetAllMetrics(w http.ResponseWriter, req *http.Request) {
	metricsList := storage.GetAllMetrics()

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/html")

	for _, metric := range metricsList {
		io.WriteString(w, fmt.Sprintf("%+v\n", metric))
	}
}

func GetMetricsByID(w http.ResponseWriter, req *http.Request) {
	metricName := chi.URLParam(req, "metricName")
	metricType := chi.URLParam(req, "metricType")

	if metricName == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	existingMetric, exists := storage.GetMetricsByID(metricName, metricType)

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	switch existingMetric.MType {
	case models.Counter:
		io.WriteString(w, strconv.FormatInt(*existingMetric.Delta, 10))
	case models.Gauge:
		io.WriteString(w, strconv.FormatFloat(*existingMetric.Value, 'f', -1, 64))
	}
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
		io.WriteString(w, fmt.Sprintf("%+v", existingMetric))
	} else {
		newMetrics := models.NewMetrics(metricName, metricType, delta, value)
		storage.AddMetrics(newMetrics)
		io.WriteString(w, fmt.Sprintf("%+v", newMetrics))
	}

	fmt.Printf("storage: %+v\r\n", storage.MetricsList)

	w.WriteHeader(http.StatusOK)
}
