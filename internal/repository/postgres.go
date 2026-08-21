package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
	"github.com/jackc/pgx/v5/pgconn"
)

// PostgresStorage хранит метрики в базе данных PostgreSQL.
type PostgresStorage struct {
	db *sql.DB
}

type rowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

var retrySleep = time.Sleep

// NewPostgresStorage создаёт хранилище, использующее указанное подключение к базе данных.
func NewPostgresStorage(db *sql.DB) *PostgresStorage {
	return &PostgresStorage{db: db}
}

// GetMetricsByID возвращает метрику по имени и типу, повторяя запрос при временном сбое соединения.
func (s *PostgresStorage) GetMetricsByID(ctx context.Context, id string, mType string) (*models.Metrics, bool, error) {
	return withRetry(func() (*models.Metrics, bool, error) {
		row := s.db.QueryRowContext(
			ctx,
			`SELECT id, mtype, delta, value FROM metrics WHERE id = $1 AND mtype = $2`,
			id,
			mType,
		)

		var metric models.Metrics
		err := row.Scan(&metric.ID, &metric.MType, &metric.Delta, &metric.Value)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		if err != nil {
			return nil, false, err
		}

		return &metric, true, nil
	})
}

// GetAllMetrics возвращает все метрики из базы данных.
func (s *PostgresStorage) GetAllMetrics(ctx context.Context) ([]models.Metrics, error) {
	return withRetryValue(func() ([]models.Metrics, error) {
		rows, err := s.db.QueryContext(ctx, `SELECT id, mtype, delta, value FROM metrics`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var metrics []models.Metrics

		for rows.Next() {
			var metric models.Metrics
			if err := rows.Scan(&metric.ID, &metric.MType, &metric.Delta, &metric.Value); err != nil {
				return nil, err
			}
			metrics = append(metrics, metric)
		}

		if err := rows.Err(); err != nil {
			return nil, err
		}

		return metrics, nil
	})
}

// SaveMetrics создаёт или обновляет метрику в базе данных.
func (s *PostgresStorage) SaveMetrics(ctx context.Context, metrics models.Metrics) (*models.Metrics, error) {
	if err := validateMetric(metrics); err != nil {
		return nil, err
	}
	return withRetryValue(func() (*models.Metrics, error) {
		return saveMetric(ctx, s.db, metrics)
	})
}

// SaveMetricsBatch атомарно сохраняет пакет метрик в одной транзакции.
func (s *PostgresStorage) SaveMetricsBatch(ctx context.Context, metrics []models.Metrics) ([]models.Metrics, error) {
	for _, metric := range metrics {
		if err := validateMetric(metric); err != nil {
			return nil, err
		}
	}

	return withRetryValue(func() ([]models.Metrics, error) {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}

		saved := make([]models.Metrics, 0, len(metrics))
		for _, metric := range metrics {
			savedMetric, err := saveMetric(ctx, tx, metric)
			if err != nil {
				return nil, errors.Join(err, tx.Rollback())
			}
			saved = append(saved, *savedMetric)
		}

		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return saved, nil
	})
}

func saveMetric(ctx context.Context, querier rowQuerier, metric models.Metrics) (*models.Metrics, error) {
	var row *sql.Row
	switch metric.MType {
	case models.Counter:
		row = querier.QueryRowContext(
			ctx,
			`
			INSERT INTO metrics (id, mtype, delta)
			VALUES ($1, $2, $3)
			ON CONFLICT (id, mtype)
			DO UPDATE SET delta = metrics.delta + EXCLUDED.delta
			RETURNING id, mtype, delta, value
			`,
			metric.ID,
			metric.MType,
			metric.Delta,
		)
	case models.Gauge:
		row = querier.QueryRowContext(
			ctx,
			`
			INSERT INTO metrics (id, mtype, value)
			VALUES ($1, $2, $3)
			ON CONFLICT (id, mtype)
			DO UPDATE SET value = EXCLUDED.value
			RETURNING id, mtype, delta, value
			`,
			metric.ID,
			metric.MType,
			metric.Value,
		)
	default:
		return nil, fmt.Errorf("unsupported metric type %q", metric.MType)
	}

	var saved models.Metrics
	if err := row.Scan(&saved.ID, &saved.MType, &saved.Delta, &saved.Value); err != nil {
		return nil, err
	}
	return &saved, nil
}

func withRetry(operation func() (*models.Metrics, bool, error)) (*models.Metrics, bool, error) {
	var lastMetric *models.Metrics
	var lastExists bool
	var lastErr error

	for attempt := 0; attempt <= len(postgresRetryIntervals); attempt++ {
		metric, exists, err := operation()
		if err == nil {
			return metric, exists, nil
		}

		lastMetric = metric
		lastExists = exists
		lastErr = err
		if !isRetriablePostgresError(err) || attempt == len(postgresRetryIntervals) {
			return lastMetric, lastExists, err
		}

		retrySleep(postgresRetryIntervals[attempt])
	}

	return lastMetric, lastExists, lastErr
}

func withRetryValue[T any](operation func() (T, error)) (T, error) {
	var zero T
	var lastValue T
	var lastErr error

	for attempt := 0; attempt <= len(postgresRetryIntervals); attempt++ {
		value, err := operation()
		if err == nil {
			return value, nil
		}

		lastValue = value
		lastErr = err
		if !isRetriablePostgresError(err) || attempt == len(postgresRetryIntervals) {
			return lastValue, err
		}

		retrySleep(postgresRetryIntervals[attempt])
	}

	return zero, lastErr
}

var postgresRetryIntervals = []time.Duration{
	time.Second,
	3 * time.Second,
	5 * time.Second,
}

func isRetriablePostgresError(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return strings.HasPrefix(pgErr.Code, "08")
}
