package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileObserverNotifyAppendsJSONLine(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "audit.log")
	observer := NewFileObserver(filePath)

	err := observer.Notify(context.Background(), Event{
		Timestamp: 12345678,
		Metrics:   []string{"Alloc", "Frees"},
		IPAddress: "192.168.0.42",
	})
	require.NoError(t, err)

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.Len(t, lines, 1)

	var event Event
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &event))
	assert.Equal(t, int64(12345678), event.Timestamp)
	assert.Equal(t, []string{"Alloc", "Frees"}, event.Metrics)
	assert.Equal(t, "192.168.0.42", event.IPAddress)
}

func TestHTTPObserverNotifyPostsJSON(t *testing.T) {
	client := &stubHTTPClient{
		t: t,
		do: func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, http.MethodPost, req.Method)
			assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
			assert.Equal(t, "https://audit.example.com/events", req.URL.String())

			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)

			var received Event
			require.NoError(t, json.Unmarshal(body, &received))
			assert.Equal(t, int64(12345678), received.Timestamp)
			assert.Equal(t, []string{"Alloc"}, received.Metrics)
			assert.Equal(t, "192.168.0.42", received.IPAddress)

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(nil)),
			}, nil
		},
	}

	observer := NewHTTPObserver("https://audit.example.com/events", client)
	err := observer.Notify(context.Background(), Event{
		Timestamp: 12345678,
		Metrics:   []string{"Alloc"},
		IPAddress: "192.168.0.42",
	})
	require.NoError(t, err)
}

func TestNewEventExtractsForwardedIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/update/", nil)
	req.Header.Set("X-Forwarded-For", "192.168.0.42, 10.0.0.1")
	req.RemoteAddr = "127.0.0.1:8080"

	event := NewEvent([]string{"Alloc"}, req)

	assert.Equal(t, "192.168.0.42", event.IPAddress)
	assert.Equal(t, []string{"Alloc"}, event.Metrics)
}

type stubHTTPClient struct {
	t  *testing.T
	do func(req *http.Request) (*http.Response, error)
}

func (s *stubHTTPClient) Do(req *http.Request) (*http.Response, error) {
	s.t.Helper()
	return s.do(req)
}
