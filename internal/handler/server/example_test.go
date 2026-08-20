package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	handler "github.com/ApplePieAndCrime/go-yandex-metrics/internal/handler/server"
	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/repository"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/service"
	"go.uber.org/zap"
)

func exampleRouter() http.Handler {
	storage := repository.NewMemoryStorage()
	metricsService := service.NewService(storage)
	metricsHandler := handler.NewHandler(metricsService, *zap.NewNop().Sugar(), nil, "", nil, nil)
	return metricsHandler.InitRoutes()
}

// ExampleHandler_UpdateMetrics показывает обновление счётчика через параметры URL.
func ExampleHandler_UpdateMetrics() {
	router := exampleRouter()

	updateRequest := httptest.NewRequest(http.MethodPost, "/update/counter/requests/3", nil)
	updateResponse := httptest.NewRecorder()
	router.ServeHTTP(updateResponse, updateRequest)

	valueRequest := httptest.NewRequest(http.MethodGet, "/value/counter/requests", nil)
	valueResponse := httptest.NewRecorder()
	router.ServeHTTP(valueResponse, valueRequest)

	fmt.Println(updateResponse.Code)
	fmt.Println(valueResponse.Body.String())

	// Output:
	// 200
	// 3
}

// ExampleHandler_UpdateMetricsByJSON показывает создание измерителя из JSON.
func ExampleHandler_UpdateMetricsByJSON() {
	router := exampleRouter()
	value := 42.5
	body, _ := json.Marshal(models.Metrics{
		ID:    "temperature",
		MType: models.Gauge,
		Value: &value,
	})

	request := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	var metric models.Metrics
	_ = json.Unmarshal(response.Body.Bytes(), &metric)
	fmt.Println(response.Code)
	fmt.Println(metric.ID, metric.MType, *metric.Value)

	// Output:
	// 200
	// temperature gauge 42.5
}

// ExampleHandler_BulkUpdateMetrics показывает пакетную отправку метрик.
func ExampleHandler_BulkUpdateMetrics() {
	router := exampleRouter()
	delta := int64(2)
	value := 128.25
	body, _ := json.Marshal([]models.Metrics{
		{ID: "jobs", MType: models.Counter, Delta: &delta},
		{ID: "memory", MType: models.Gauge, Value: &value},
	})

	request := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	responseBody, _ := io.ReadAll(response.Result().Body)

	var metrics []models.Metrics
	_ = json.Unmarshal(responseBody, &metrics)
	fmt.Println(response.Code)
	fmt.Println(metrics[0].ID, *metrics[0].Delta)
	fmt.Println(metrics[1].ID, *metrics[1].Value)

	// Output:
	// 200
	// jobs 2
	// memory 128.25
}
