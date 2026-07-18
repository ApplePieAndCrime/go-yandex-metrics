package internal_agent

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/hashutil"
	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestSendRequestToServer(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/updates/", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.JSONEq(t, `[{"id":"test","type":"counter","delta":100}]`, string(body))

			var metrics []models.Metrics
			err = json.Unmarshal(body, &metrics)
			require.NoError(t, err)
			require.Len(t, metrics, 1)

			metric := metrics[0]

			require.NotNil(t, metric.Delta)
			assert.Equal(t, "test", metric.ID)
			assert.Equal(t, models.Counter, metric.MType)
			assert.Equal(t, int64(100), *metric.Delta)
			assert.Nil(t, metric.Value)

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	resp, statusCode, err := SendRequestToServer(client, "http://example.com", MemMetrics{
		"test": {Type: models.Counter, Value: "100"},
	}, "")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, statusCode)
}

func TestSendRequestToServerSetsHashHeader(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.Equal(t, hashutil.SignBody(body, "test-key"), r.Header.Get(hashutil.HeaderName))

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	resp, statusCode, err := SendRequestToServer(client, "http://example.com", MemMetrics{
		"test": {Type: models.Counter, Value: "100"},
	}, "test-key")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, statusCode)
}

func TestCollectAndSendMetricsChangesBetweenReports(t *testing.T) {
	type sentMetric struct {
		ID    string
		MType string
		Delta *int64
		Value *float64
	}

	sent := make(map[string][]sentMetric)
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			var metrics []models.Metrics
			err = json.Unmarshal(body, &metrics)
			require.NoError(t, err)

			for _, metric := range metrics {
				if metric.ID == "PollCount" || metric.ID == "RandomValue" {
					sent[metric.ID] = append(sent[metric.ID], sentMetric{
						ID:    metric.ID,
						MType: metric.MType,
						Delta: metric.Delta,
						Value: metric.Value,
					})
				}
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	randomValues := []float64{0.25, 0.75}
	randomIndex := 0
	randomFloat := func() float64 {
		value := randomValues[randomIndex]
		randomIndex++
		return value
	}

	safeMetrics := &SafeMetrics{}

	safeMetrics.collect(randomFloat)
	snapshot := safeMetrics.Snapshot()
	_, err := sendAllMetrics(client, "http://example.com", &snapshot, "")
	require.NoError(t, err)

	safeMetrics.collect(randomFloat)
	snapshot2 := safeMetrics.Snapshot()
	_, err = sendAllMetrics(client, "http://example.com", &snapshot2, "")
	require.NoError(t, err)

	require.Len(t, sent["PollCount"], 2)
	require.Len(t, sent["RandomValue"], 2)

	require.NotNil(t, sent["PollCount"][0].Delta)
	require.NotNil(t, sent["PollCount"][1].Delta)
	assert.Equal(t, int64(1), *sent["PollCount"][0].Delta)
	assert.Equal(t, int64(2), *sent["PollCount"][1].Delta)

	require.NotNil(t, sent["RandomValue"][0].Value)
	require.NotNil(t, sent["RandomValue"][1].Value)
	assert.Equal(t, 0.25, *sent["RandomValue"][0].Value)
	assert.Equal(t, 0.75, *sent["RandomValue"][1].Value)
	assert.NotEqual(t, *sent["RandomValue"][0].Value, *sent["RandomValue"][1].Value)
}

func TestSendAllMetricsWithRetry(t *testing.T) {
	attempts := 0
	var sleeps []time.Duration

	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			attempts++
			if attempts < 4 {
				return nil, &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("temporary connection failure")}
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	metrics := &AgentMetrics{}

	_, err := sendAllMetricsWithRetry(client, "http://example.com", metrics, func(delay time.Duration) {
		sleeps = append(sleeps, delay)
	}, "")
	require.NoError(t, err)

	assert.Equal(t, 4, attempts)
	assert.Equal(t, []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}, sleeps)
}

func TestSendAllMetricsWithRetryDoesNotRetryNonTransportErrors(t *testing.T) {
	_, _, err := SendRequestToServer(&http.Client{}, "http://example.com", MemMetrics{
		"broken": {Type: "unsupported", Value: "1"},
	}, "")
	require.Error(t, err)
	assert.False(t, isRetriableRequestError(err))
}
