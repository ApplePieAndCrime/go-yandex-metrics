package handler_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	handler "github.com/ApplePieAndCrime/go-yandex-metrics/internal/handler/server"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/hashutil"
	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
	repository "github.com/ApplePieAndCrime/go-yandex-metrics/internal/repository"
	logger "github.com/ApplePieAndCrime/go-yandex-metrics/internal/server/logger"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/service"
	"github.com/stretchr/testify/assert"
)

func newTestRouter(key string) http.Handler {
	loggerSugar := logger.LoggerInitialize()

	repo := repository.NewMemoryStorage()

	delta := int64(10)
	repo.SaveMetrics(context.Background(), models.Metrics{
		ID:    "testCounter",
		MType: models.Counter,
		Delta: &delta,
	})

	services := service.NewService(repo)
	handlers := handler.NewHandler(services, loggerSugar, nil, key)

	return handlers.InitRoutes()
}

func TestGetAllMetrics(t *testing.T) {

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	newTestRouter("").ServeHTTP(w, req)

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

	router := newTestRouter("")

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

	router := newTestRouter("")

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

func TestBulkUpdateMetrics(t *testing.T) {
	type want struct {
		code int
	}

	tests := []struct {
		name string
		body []models.Metrics
		want want
	}{
		{
			name: "create counter",
			body: []models.Metrics{
				{
					ID:    "jsonCounter",
					MType: models.Counter,
					Delta: int64Ptr(5),
				},
			},
			want: want{code: http.StatusOK},
		},
		{
			name: "invalid counter without delta",
			body: []models.Metrics{
				{
					ID:    "jsonCounter",
					MType: models.Counter,
				},
			},
			want: want{code: http.StatusBadRequest},
		},
		{
			name: "create gauge",
			body: []models.Metrics{
				{
					ID:    "jsonGauge",
					MType: models.Gauge,
					Value: float64Ptr(3.14),
				},
			},
			want: want{code: http.StatusOK},
		},
	}

	router := newTestRouter("")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.body)
			assert.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
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

	router := newTestRouter("")

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

	router := newTestRouter("")

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

func TestUpdateMetricsByJSONRejectsInvalidHash(t *testing.T) {
	router := newTestRouter("test-key")

	body, err := json.Marshal(models.Metrics{
		ID:    "jsonCounter",
		MType: models.Counter,
		Delta: int64Ptr(5),
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hashutil.HeaderName, "broken-hash")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	result := w.Result()
	assert.Equal(t, http.StatusBadRequest, result.StatusCode)
}

func TestUpdateMetricsByJSONAllowsMissingHashHeader(t *testing.T) {
	router := newTestRouter("test-key")

	body, err := json.Marshal(models.Metrics{
		ID:    "jsonCounter",
		MType: models.Counter,
		Delta: int64Ptr(5),
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	result := w.Result()
	assert.Equal(t, http.StatusOK, result.StatusCode)
}

func TestGetMetricsByIDWithJSONReturnsNotFoundForEmptyBody(t *testing.T) {
	router := newTestRouter("")

	req := httptest.NewRequest(http.MethodPost, "/value/", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	result := w.Result()
	assert.Equal(t, http.StatusNotFound, result.StatusCode)
	assert.Equal(t, "application/json", result.Header.Get("Content-Type"))
}

func TestUpdateMetricsByJSONAcceptsGzipRequest(t *testing.T) {
	router := newTestRouter("")

	body, err := json.Marshal(models.Metrics{
		ID:    "gzipGauge",
		MType: models.Gauge,
		Value: float64Ptr(3.14),
	})
	assert.NoError(t, err)

	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	_, err = gz.Write(body)
	assert.NoError(t, err)
	assert.NoError(t, gz.Close())

	req := httptest.NewRequest(http.MethodPost, "/update/", &compressed)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	result := w.Result()
	assert.Equal(t, http.StatusOK, result.StatusCode)
}

func TestUpdateMetricsByJSONAcceptsGzipBodyWithoutContentEncoding(t *testing.T) {
	router := newTestRouter("")

	body, err := json.Marshal(models.Metrics{
		ID:    "gzipGauge",
		MType: models.Gauge,
		Value: float64Ptr(3.14),
	})
	assert.NoError(t, err)

	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	_, err = gz.Write(body)
	assert.NoError(t, err)
	assert.NoError(t, gz.Close())

	req := httptest.NewRequest(http.MethodPost, "/update/", &compressed)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	result := w.Result()
	assert.Equal(t, http.StatusOK, result.StatusCode)
}

func TestUpdateThenGetMetricsByIDWithJSONReturnsValue(t *testing.T) {
	router := newTestRouter("")

	updateBody, err := json.Marshal(models.Metrics{
		ID:    "jsonGauge",
		MType: models.Gauge,
		Value: float64Ptr(3.14),
	})
	assert.NoError(t, err)

	updateReq := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRecorder := httptest.NewRecorder()
	router.ServeHTTP(updateRecorder, updateReq)
	assert.Equal(t, http.StatusOK, updateRecorder.Result().StatusCode)

	valueBody, err := json.Marshal(models.Metrics{
		ID:    "jsonGauge",
		MType: models.Gauge,
	})
	assert.NoError(t, err)

	valueReq := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader(valueBody))
	valueReq.Header.Set("Content-Type", "application/json")
	valueRecorder := httptest.NewRecorder()
	router.ServeHTTP(valueRecorder, valueReq)

	result := valueRecorder.Result()
	responseBody, err := io.ReadAll(result.Body)
	assert.NoError(t, err)

	var metric models.Metrics
	assert.NoError(t, json.Unmarshal(responseBody, &metric))
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.NotNil(t, metric.Value)
	assert.Nil(t, metric.Delta)
}

func TestGetMetricsByIDWithJSONSetsResponseHash(t *testing.T) {
	router := newTestRouter("test-key")

	requestBody, err := json.Marshal(models.Metrics{
		ID:    "testCounter",
		MType: models.Counter,
	})
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hashutil.HeaderName, hashutil.SignBody(requestBody, "test-key"))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	result := w.Result()
	responseBody, err := io.ReadAll(result.Body)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Equal(t, hashutil.SignBody(responseBody, "test-key"), result.Header.Get(hashutil.HeaderName))
}

func int64Ptr(v int64) *int64 {
	return &v
}

func float64Ptr(v float64) *float64 {
	return &v
}
