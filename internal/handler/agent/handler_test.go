package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	agent_handler "github.com/ApplePieAndCrime/go-yandex-metrics/internal/handler/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendRequestToServer(t *testing.T) {

	tests := []struct {
		name       string
		method     string
		url        string
		statusCode int
	}{
		{
			name:       "валидный запрос",
			url:        "/update/counter/test/100",
			statusCode: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer ts.Close()

			client := ts.Client()

			resp, err, statusCode := agent_handler.SendRequestToServer(client, ts.URL, "counter", "test", "100")
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.statusCode, statusCode)
		})
	}
}
