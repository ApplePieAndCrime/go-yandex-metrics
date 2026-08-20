package internal_agent

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/cryptoutil"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/hashutil"
	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunAgentStopsWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.NoError(t, RunAgent(ctx, "http://example.com", 1, 1, "", 1, ""))
}

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
			assert.NotNil(t, net.ParseIP(r.Header.Get("X-Real-IP")))

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
	}, "", nil)
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
	}, "test-key", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, statusCode)
}

func TestSendRequestToServerEncryptsBody(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			body, readErr := io.ReadAll(r.Body)
			require.NoError(t, readErr)
			assert.NotContains(t, string(body), `"id":"test"`)

			decrypted, decryptErr := cryptoutil.Decrypt(privateKey, body)
			require.NoError(t, decryptErr)
			assert.JSONEq(t, `[{"id":"test","type":"counter","delta":100}]`, string(decrypted))
			assert.Equal(t, hashutil.SignBody(decrypted, "test-key"), r.Header.Get(hashutil.HeaderName))

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	resp, statusCode, err := SendRequestToServer(client, "http://example.com", MemMetrics{
		"test": {Type: models.Counter, Value: "100"},
	}, "test-key", &privateKey.PublicKey)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, statusCode)
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
	_, err := sendAllMetrics(client, "http://example.com", &snapshot, "", nil)
	require.NoError(t, err)

	safeMetrics.collect(randomFloat)
	snapshot2 := safeMetrics.Snapshot()
	_, err = sendAllMetrics(client, "http://example.com", &snapshot2, "", nil)
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
	}, "", nil)
	require.NoError(t, err)

	assert.Equal(t, 4, attempts)
	assert.Equal(t, []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}, sleeps)
}

func TestSendAllMetricsWithRetryDoesNotRetryNonTransportErrors(t *testing.T) {
	_, _, err := SendRequestToServer(&http.Client{}, "http://example.com", MemMetrics{
		"broken": {Type: "unsupported", Value: "1"},
	}, "", nil)
	require.Error(t, err)
	assert.False(t, isRetriableRequestError(err))
}

func TestSendAllMetricsWithRetryRetriesServerError(t *testing.T) {
	originalIntervals := retryIntervals
	retryIntervals = []time.Duration{0}
	t.Cleanup(func() {
		retryIntervals = originalIntervals
	})

	attempts := 0
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			attempts++
			statusCode := http.StatusServiceUnavailable
			if attempts == 2 {
				statusCode = http.StatusOK
			}

			return &http.Response{
				StatusCode: statusCode,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	_, err := sendAllMetricsWithRetry(
		client,
		"http://example.com",
		&AgentMetrics{},
		time.Sleep,
		"",
		nil,
	)
	require.NoError(t, err)
	assert.Equal(t, 2, attempts)
}

func TestFlushUnsentMetricsSendsLatestSnapshotOnce(t *testing.T) {
	var requests int
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			requests++

			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			var metrics []models.Metrics
			require.NoError(t, json.Unmarshal(body, &metrics))

			var pollCount *int64
			for _, metric := range metrics {
				if metric.ID == "PollCount" {
					pollCount = metric.Delta
					break
				}
			}
			require.NotNil(t, pollCount)
			assert.Equal(t, int64(1), *pollCount)

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	metrics := &SafeMetrics{
		data:     AgentMetrics{PollCount: 1},
		revision: 1,
	}

	sent, err := sendUnsentMetrics(client, "http://example.com", metrics, "", nil)
	require.NoError(t, err)
	assert.True(t, sent)

	sent, err = sendUnsentMetrics(client, "http://example.com", metrics, "", nil)
	require.NoError(t, err)
	assert.False(t, sent)
	assert.Equal(t, 1, requests)
}

func TestFlushUnsentMetricsSerializesConcurrentReports(t *testing.T) {
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	var requests int
	var requestsMu sync.Mutex

	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			requestsMu.Lock()
			requests++
			if requests == 1 {
				close(requestStarted)
			}
			requestsMu.Unlock()

			<-releaseRequest
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	metrics := &SafeMetrics{
		data:     AgentMetrics{PollCount: 1},
		revision: 1,
	}

	var wg sync.WaitGroup
	type result struct {
		sent bool
		err  error
	}
	results := make(chan result, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sent, err := sendUnsentMetrics(client, "http://example.com", metrics, "", nil)
			results <- result{sent: sent, err: err}
		}()
	}

	<-requestStarted
	close(releaseRequest)
	wg.Wait()
	close(results)
	sentCount := 0
	for result := range results {
		require.NoError(t, result.err)
		if result.sent {
			sentCount++
		}
	}
	assert.Equal(t, 1, sentCount)

	requestsMu.Lock()
	defer requestsMu.Unlock()
	assert.Equal(t, 1, requests)
}
