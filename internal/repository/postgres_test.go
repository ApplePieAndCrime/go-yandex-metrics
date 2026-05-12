package repository

import (
	"errors"
	"testing"
	"time"

	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRetriablePostgresError(t *testing.T) {
	assert.True(t, isRetriablePostgresError(&pgconn.PgError{Code: "08006"}))
	assert.False(t, isRetriablePostgresError(&pgconn.PgError{Code: "23505"}))
	assert.False(t, isRetriablePostgresError(errors.New("plain error")))
}

func TestWithRetryValueRetriesConnectionException(t *testing.T) {
	originalSleep := retrySleep
	t.Cleanup(func() {
		retrySleep = originalSleep
	})

	var sleeps []time.Duration
	retrySleep = func(delay time.Duration) {
		sleeps = append(sleeps, delay)
	}

	attempts := 0
	expected := &models.Metrics{ID: "Alloc", MType: models.Gauge}

	metric, err := withRetryValue(func() (*models.Metrics, error) {
		attempts++
		if attempts < 4 {
			return nil, &pgconn.PgError{Code: "08006", Message: "connection failure"}
		}

		return expected, nil
	})
	require.NoError(t, err)

	assert.Equal(t, 4, attempts)
	assert.Same(t, expected, metric)
	assert.Equal(t, []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}, sleeps)
}

func TestWithRetryValueDoesNotRetryNonRetriableError(t *testing.T) {
	originalSleep := retrySleep
	t.Cleanup(func() {
		retrySleep = originalSleep
	})

	retrySleep = func(time.Duration) {
		t.Fatal("sleep should not be called for non-retriable errors")
	}

	attempts := 0
	_, err := withRetryValue(func() (*models.Metrics, error) {
		attempts++
		return nil, &pgconn.PgError{Code: "23505", Message: "unique violation"}
	})
	require.Error(t, err)
	assert.Equal(t, 1, attempts)
}
