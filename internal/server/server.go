package server

import (
	"fmt"
	"os"
	"time"

	// models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/server/file_converter"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/server/logger"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/service"
)

// func ReadMetricsJson(jsonBody []byte) (*models.Metrics, error) {
// 	var metrics models.Metrics
// 	// buf := bytes.NewBuffer(jsonBody)
// 	body, err := io.ReadAll(jsonBody)

// 	if err := json.Unmarshal(buf.Bytes(), &metrics); err != nil {
// 		return nil, err
// 	}
// 	return &metrics, nil
// }

func RunServer(services service.Service, storeInterval int64, fileStoragePath string, isRestore bool) {
	logger.Sugar.Infof("work with file %s started by interval %d", fileStoragePath, storeInterval)

	intervalTicker := time.NewTicker(time.Duration(storeInterval) * time.Second)
	defer intervalTicker.Stop()

	for range intervalTicker.C {
		file, err := os.OpenFile(fileStoragePath, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			fmt.Errorf("can't open file: %w", err)
		}
		defer file.Close()

		if isRestore {
			var metricsList []models.Metrics

			consumer, err := file_converter.NewConsumer(fileStoragePath)
			if err != nil {
				fmt.Errorf("consumer: can't open file: %w", err)
			}
			metricsList, err = consumer.ReadMetricsList()
			if err != nil {
				fmt.Errorf("decoder error: %w", err)
			}
			consumer.Close()

			for _, metrics := range metricsList {
				services.CreateOrUpdateMetrics(metrics)
			}
		}

		storageMetricsList := services.GetAllMetrics()

		producer, err := file_converter.NewProducer(fileStoragePath)
		if err != nil {
			fmt.Errorf("producer: can't open file: %w", err)
		}

		if err := producer.WriteEvent(storageMetricsList); err != nil {
			fmt.Errorf("producer write error: %w", err)
		}

		producer.Close()
	}
}
