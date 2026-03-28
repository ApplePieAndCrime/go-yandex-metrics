package handler_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	handler "github.com/ApplePieAndCrime/go-yandex-metrics/internal/handler/agent"
	"github.com/stretchr/testify/assert"
)

func TestGetMetrics(t *testing.T) {

	tests := []struct {
		name      string
		method    string
		url       string
		resStatus int
	}{
		{
			name:      "валидный запрос",
			method:    http.MethodGet,
			url:       "/",
			resStatus: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.url, nil)
			w := httptest.NewRecorder()
			handler.GetMetrics(w, request)

			result := w.Result()
			fmt.Printf("result: %+v\n", result)
			assert.Equal(t, tt.resStatus, result.StatusCode)
		})
	}
}
