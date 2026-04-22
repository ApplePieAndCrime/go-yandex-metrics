package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	handler "github.com/ApplePieAndCrime/go-yandex-metrics/internal/handler/server"
	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/repository"
	logger "github.com/ApplePieAndCrime/go-yandex-metrics/internal/server/logger"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/service"
	"github.com/stretchr/testify/assert"
)

func newTestRouter() http.Handler {
	logger.LoggerInitialize()
	repo := repository.NewRepository()
	repo.Storage.AddMetrics(*repo.Storage.NewMetrics("testCounter", models.Counter, 10, 0))

	services := service.NewService(repo)
	handlers := handler.NewHandler(services)

	return handlers.InitRoutes()
}

func TestGetAllMetrics(t *testing.T) {

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	newTestRouter().ServeHTTP(w, req)

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
			name:   "valid request",
			method: http.MethodGet,
			url:    "/value/counter/testCounter",
			want: want{
				code:        http.StatusOK,
				contentType: "text/plain",
				body:        "10",
			},
		},
		{
			name:   "wrong method",
			method: http.MethodPost,
			url:    "/value/counter/testCounter",
			want: want{
				code:        http.StatusMethodNotAllowed,
				contentType: "",
				body:        "",
			},
		},
		{
			name:   "unknown metric",
			method: http.MethodGet,
			url:    "/value/counter/testCounter2",
			want: want{
				code:        http.StatusNotFound,
				contentType: "text/plain",
				body:        "",
			},
		},
	}

	router := newTestRouter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			res := w.Result()
			body, _ := io.ReadAll(res.Body)

			assert.Equal(t, tt.want.code, res.StatusCode)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
			assert.Contains(t, string(body), tt.want.body)
		})
	}
}

func TestGetMetricsByIDByJSON(t *testing.T) {
	type want struct {
		code int
	}

	tests := []struct {
		name string
		body models.Metrics
		want want
	}{
		{
			name: "get existing counter",
			body: models.Metrics{
				ID:    "testCounter",
				MType: models.Counter,
			},
			want: want{code: http.StatusOK},
		},
		{
			name: "get unknown gauge",
			body: models.Metrics{
				ID:    "jsonGauge",
				MType: models.Gauge,
			},
			want: want{code: http.StatusNotFound},
		},
		{
			name: "invalid counter without MType",
			body: models.Metrics{
				ID: "jsonCounter",
			},
			want: want{code: http.StatusNotFound},
		},
	}

	router := newTestRouter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.body)
			assert.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			result := w.Result()
			assert.Equal(t, tt.want.code, result.StatusCode)
			assert.Equal(t, "application/json", result.Header.Get("Content-Type"))
		})
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
			name:   "valid request",
			method: http.MethodPost,
			url:    "/update/counter/test/100",
			want: want{
				code:        http.StatusOK,
				contentType: "text/plain",
			},
		},
		{
			name:   "wrong method",
			method: http.MethodGet,
			url:    "/update/counter/test/100",
			want: want{
				code:        http.StatusMethodNotAllowed,
				contentType: "",
			},
		},
		{
			name:   "invalid counter value",
			method: http.MethodPost,
			url:    "/update/counter/test/100.2",
			want: want{
				code:        http.StatusBadRequest,
				contentType: "text/plain",
			},
		},
	}

	router := newTestRouter()

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

func TestUpdateMetricsByJSON(t *testing.T) {
	type want struct {
		code int
	}

	tests := []struct {
		name string
		body models.Metrics
		want want
	}{
		{
			name: "create counter",
			body: models.Metrics{
				ID:    "jsonCounter",
				MType: models.Counter,
				Delta: int64Ptr(5),
			},
			want: want{code: http.StatusOK},
		},
		{
			name: "invalid counter without delta",
			body: models.Metrics{
				ID:    "jsonCounter",
				MType: models.Counter,
			},
			want: want{code: http.StatusBadRequest},
		},
		{
			name: "create gauge",
			body: models.Metrics{
				ID:    "jsonGauge",
				MType: models.Gauge,
				Value: float64Ptr(3.14),
			},
			want: want{code: http.StatusOK},
		},
	}

	router := newTestRouter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.body)
			assert.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			result := w.Result()
			assert.Equal(t, tt.want.code, result.StatusCode)
			assert.Equal(t, "application/json", result.Header.Get("Content-Type"))
		})
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}

func float64Ptr(v float64) *float64 {
	return &v
}
