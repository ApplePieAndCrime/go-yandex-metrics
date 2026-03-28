package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	handler "github.com/ApplePieAndCrime/go-yandex-metrics/internal/handler/server"
	"github.com/stretchr/testify/assert"
)

func TestGUpdateMetrics(t *testing.T) {
	type want struct {
		code        int
		contentType string
	}

	tests := []struct {
		name   string
		method string
		url    string
		want   want
	}{
		{
			name:   "валидный запрос",
			method: http.MethodPost,
			url:    "/update/counter/test/100",
			want: want{
				code:        http.StatusOK,
				contentType: "text/plain",
			},
		},
		{
			name:   "неправильный метод",
			method: http.MethodGet,
			url:    "/update/counter/test/100",
			want: want{
				code:        http.StatusNotFound,
				contentType: "text/plain",
			},
		},
		{
			name:   "неправильный метод",
			method: http.MethodPost,
			url:    "/update/counter/test/100.2",
			want: want{
				code:        http.StatusBadRequest,
				contentType: "text/plain",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.url, nil)
			w := httptest.NewRecorder()
			handler.UpdateMetrics(w, request)

			result := w.Result()
			assert.Equal(t, tt.want.code, result.StatusCode)
			assert.Equal(t, tt.want.contentType, result.Header.Get("Content-Type"))
		})
	}
}
