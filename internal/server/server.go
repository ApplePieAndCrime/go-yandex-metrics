package server

import (
	"fmt"
	"time"

	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/server/file_converter"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/service"
)

// SaveMetricsToFile восстанавливает метрики из файла и запускает их периодическое сохранение.
func SaveMetricsToFile(
	services service.Service,
	storeInterval int64,
	fileStoragePath string,
	isRestore bool,
) <-chan error {
	errCh := make(chan error, 1)

	if isRestore {
		consumer, err := file_converter.NewConsumer(fileStoragePath)
		if err != nil {
			errCh <- fmt.Errorf("consumer: can't open file: %w", err)
			close(errCh)
			return errCh
		}
		defer consumer.Close()

		metricsList, err := consumer.ReadMetricsList()
		if err != nil {
			errCh <- fmt.Errorf("decoder error: %w", err)
			close(errCh)
			return errCh
		}

		for _, metrics := range metricsList {
			services.CreateOrUpdateMetrics(metrics)
		}
	}

	go func() {
		ticker := time.NewTicker(time.Duration(storeInterval) * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			storageMetricsList, err := services.GetAllMetrics()
			if err != nil {
				errCh <- fmt.Errorf("failed to get all metrics for file: %w", err)
				return
			}

			producer, err := file_converter.NewProducer(fileStoragePath)
			if err != nil {
				errCh <- fmt.Errorf("producer: can't open file: %w", err)
				return
			}

			if err := producer.WriteEvent(storageMetricsList); err != nil {
				producer.Close()
				errCh <- fmt.Errorf("producer write error: %w", err)
				return
			}

			producer.Close()
		}
	}()

	return errCh
}
