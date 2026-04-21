package internal_agent_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	agent "github.com/ApplePieAndCrime/go-yandex-metrics/internal/agent"
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

	resp, statusCode, err := agent.SendRequestToServer(client, "http://example.com", models.Counter, "test", "100")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, statusCode)
}
