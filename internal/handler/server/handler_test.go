package handler_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	handler "github.com/ApplePieAndCrime/go-yandex-metrics/internal/handler/server"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/repository"
	logger "github.com/ApplePieAndCrime/go-yandex-metrics/internal/server"
	"github.com/stretchr/testify/assert"
)

func TestGetAllMetrics(t *testing.T) {
	repo := repository.Init()
	repo.Storage.AddMetrics(repo.Storage.NewMetrics("testCounter", "counter", 10, 0))
	handler.Init(repo)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.GetAllMetrics(w, req)

	res := w.Result()
	body, _ := io.ReadAll(res.Body)

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "text/html", res.Header.Get("Content-Type"))
	assert.Contains(t, string(body), "testCounter")
}

func TestGetMetricsByID(t *testing.T) {
	type want struct {
		code        int
		contentType string
		body        string
	}

	tests := []struct {
		name   string
		method string
		url    string
		want   want
	}{
		{
			name:   "валидный тест",
			method: http.MethodGet,
			url:    "/value/counter/testCounter",
			want: want{
				code:        http.StatusOK,
				contentType: "text/plain",
				body:        "10",
			},
		},
		{
			name:   "неправильный метод",
			method: http.MethodPost,
			url:    "/value/counter/testCounter",
			want: want{
				code:        http.StatusMethodNotAllowed,
				contentType: "",
				body:        "",
			},
		},
		{
			name:   "metricName не существует",
			method: http.MethodGet,
			url:    "/value/counter/testCounter2",
			want: want{
				code:        http.StatusNotFound,
				contentType: "text/plain",
				body:        "",
			},
		},
	}
	logger.LoggerInitialize()

	repo := repository.Init()
	repo.Storage.AddMetrics(repo.Storage.NewMetrics("testCounter", "counter", 10, 0))

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.url, nil)
		w := httptest.NewRecorder()

		router := handler.Init(repo)
		router.ServeHTTP(w, req)

		res := w.Result()
		body, _ := io.ReadAll(res.Body)

		assert.Equal(t, tt.want.code, res.StatusCode)
		assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
		assert.Contains(t, tt.want.body, string(body))

	}

}

func TestUpdateMetrics(t *testing.T) {
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
				code:        http.StatusMethodNotAllowed,
				contentType: "",
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

	logger.LoggerInitialize()

	repo := repository.Init()
	router := handler.Init(repo)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			result := w.Result()
			assert.Equal(t, tt.want.code, result.StatusCode)
			assert.Equal(t, tt.want.contentType, result.Header.Get("Content-Type"))
		})
	}
}
