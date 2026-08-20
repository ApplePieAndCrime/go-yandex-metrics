package server

import (
	"context"
	"errors"
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
	return SaveMetricsToFileWithContext(
		context.Background(),
		services,
		storeInterval,
		fileStoragePath,
		isRestore,
	)
}

// SaveMetricsToFileWithContext восстанавливает метрики, периодически сохраняет
// их и выполняет последнюю запись перед завершением по контексту.
func SaveMetricsToFileWithContext(
	ctx context.Context,
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
		defer close(errCh)

		var (
			ticker  *time.Ticker
			tickerC <-chan time.Time
		)
		if storeInterval > 0 {
			ticker = time.NewTicker(time.Duration(storeInterval) * time.Second)
			tickerC = ticker.C
			defer ticker.Stop()
		}

		for {
			select {
			case <-tickerC:
				if err := saveMetrics(services, fileStoragePath); err != nil {
					errCh <- err
					return
				}
			case <-ctx.Done():
				if err := saveMetrics(services, fileStoragePath); err != nil {
					errCh <- err
				}
				return
			}
		}
	}()

	return errCh
}

func saveMetrics(services service.Service, fileStoragePath string) error {
	storageMetricsList, err := services.GetAllMetrics()
	if err != nil {
		return fmt.Errorf("failed to get all metrics for file: %w", err)
	}

	producer, err := file_converter.NewProducer(fileStoragePath)
	if err != nil {
		return fmt.Errorf("producer: can't open file: %w", err)
	}

	writeErr := producer.WriteEvent(storageMetricsList)
	closeErr := producer.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return fmt.Errorf("producer write error: %w", err)
	}

	return nil
}
