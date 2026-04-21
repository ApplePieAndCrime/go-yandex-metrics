package internal_agent

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

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
			assert.Equal(t, "/update", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.JSONEq(t, `{"id":"test","type":"counter","delta":100}`, string(body))

			var metric models.Metrics
			err = json.Unmarshal(body, &metric)
			require.NoError(t, err)

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

	resp, statusCode, err := SendRequestToServer(client, "http://example.com", models.Counter, "test", "100")
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

			var metric models.Metrics
			err = json.Unmarshal(body, &metric)
			require.NoError(t, err)

			if metric.ID == "PollCount" || metric.ID == "RandomValue" {
				sent[metric.ID] = append(sent[metric.ID], sentMetric{
					ID:    metric.ID,
					MType: metric.MType,
					Delta: metric.Delta,
					Value: metric.Value,
				})
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

	metrics := &AgentMetrics{}

	collectMetrics(metrics, randomFloat)
	_, errors := sendAllMetrics(client, "http://example.com", metrics)
	require.Empty(t, errors)

	collectMetrics(metrics, randomFloat)
	_, errors = sendAllMetrics(client, "http://example.com", metrics)
	require.Empty(t, errors)

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
