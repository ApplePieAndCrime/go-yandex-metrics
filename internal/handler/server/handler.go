package handler

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/hashutil"
	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
	zipper "github.com/ApplePieAndCrime/go-yandex-metrics/internal/server/gzip"
	loggerMiddleware "github.com/ApplePieAndCrime/go-yandex-metrics/internal/server/logger"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/service"
	"github.com/go-chi/chi"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

type Handler struct {
	services *service.Service
	logger   *zap.SugaredLogger
	db       *sql.DB
	key      string
}

func NewHandler(services *service.Service, logger zap.SugaredLogger, db *sql.DB, key string) *Handler {
	return &Handler{services: services, logger: logger.With("component", "handler"), db: db, key: key}
}

func (h Handler) InitRoutes() *chi.Mux {
	r := chi.NewRouter()
	r.Use(loggerMiddleware.WithLogging(*h.logger))
	r.Use(zipper.ZipMiddleware)
	r.Use(h.HashResponseMiddleware)

	r.Route("/", func(r chi.Router) {
		r.Get("/value/{metricType}/{metricName}", h.GetMetricsByID)
		r.Post("/value/", h.GetMetricsByIDWithJSON)
		r.Post("/value", h.GetMetricsByIDWithJSON)
		r.Post("/updates/", h.BulkUpdateMetrics)
		r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateMetrics)
		r.Post("/update/", h.UpdateMetricsByJSON)
		r.Post("/update", h.UpdateMetricsByJSON)
		r.Get("/ping", h.Ping)
		r.Get("/", h.GetAllMetrics)
	})

	return r
}

func (h *Handler) Ping(w http.ResponseWriter, req *http.Request) {
	if h.db == nil {
		h.logger.Errorf("database is nil")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err := h.db.Ping()
	if err != nil {
		h.logger.Errorf("database ping error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.logger.Infoln("Successfully connected to PostgreSQL!")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetAllMetrics(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	metricsList, err := h.services.GetAllMetrics()
	if err != nil {
		h.logger.Errorln("error fetching all metrics: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, metric := range metricsList {
		_, _ = io.WriteString(w, fmt.Sprintf("%+v\n", metric))
	}
}

func (h *Handler) GetMetricsByID(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	metricName := chi.URLParam(req, "metricName")
	metricType := chi.URLParam(req, "metricType")

	if metricName == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	existingMetric, exists, err := h.services.GetMetricsByID(metricName, metricType)

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}

	switch existingMetric.MType {
	case models.Counter:
		_, _ = io.WriteString(w, strconv.FormatInt(*existingMetric.Delta, 10))
	case models.Gauge:
		_, _ = io.WriteString(w, strconv.FormatFloat(*existingMetric.Value, 'f', -1, 64))
	}
}

func (h *Handler) GetMetricsByIDWithJSON(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var metrics models.Metrics
	body, err := h.readAndVerifyRequestBody(req)
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(body) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err = json.Unmarshal(body, &metrics); err != nil {
		h.logger.Errorln("unmarshal err: ", err)
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.logger.Infoln("metrics: ", metrics)

	if metrics.MType == "" || metrics.ID == "" {
		fmt.Println("test error")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	existingMetrics, exists, err := h.services.GetMetricsByID(metrics.ID, metrics.MType)
	fmt.Println("exists", exists)
	fmt.Println("existingMetric", existingMetrics)

	if err != nil || !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	resp, err := json.Marshal(*existingMetrics)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (h *Handler) UpdateMetrics(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	metricType := chi.URLParam(req, "metricType")
	metricName := chi.URLParam(req, "metricName")
	metricValue := chi.URLParam(req, "metricValue")

	fmt.Printf("metricType: %s metricName: %s metricValue: %s\r\n", metricType, metricName, metricValue)

	if metricType == "" || metricName == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var delta int64
	var value float64

	switch metricType {
	case models.Gauge:
		converted, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		value = converted
	case models.Counter:
		converted, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		delta = converted
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	existingMetric, exists, err := h.services.GetMetricsByID(metricName, metricType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if exists {
		h.services.UpdateMetrics(existingMetric, delta, value)
		resp, err := json.Marshal(existingMetric)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(resp)
	} else {
		newMetrics, createErr := h.services.CreateMetrics(metricName, metricType, delta, value)
		if createErr != nil {
			http.Error(w, createErr.Error(), http.StatusInternalServerError)
			return
		}
		resp, err := json.Marshal(newMetrics)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(resp)
	}
}

func (h *Handler) UpdateMetricsByJSON(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var metrics models.Metrics
	body, err := h.readAndVerifyRequestBody(req)
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err = json.Unmarshal(body, &metrics); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if metrics.MType == "" || metrics.ID == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if metrics.MType != models.Counter && metrics.MType != models.Gauge {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch metrics.MType {
	case models.Counter:
		if metrics.Delta == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	case models.Gauge:
		if metrics.Value == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	existingMetric, exists, err := h.services.GetMetricsByID(metrics.ID, metrics.MType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if exists {
		var delta int64
		var value float64

		if metrics.Delta != nil {
			delta = *metrics.Delta
			fmt.Println("metrics: ", metrics, "metrics delta:", *metrics.Delta)
		}
		if metrics.Value != nil {
			value = *metrics.Value
			fmt.Println("metrics: ", metrics, "metrics value:", *metrics.Value)
		}

		h.services.UpdateMetrics(existingMetric, delta, value)

		resp, err := json.Marshal(existingMetric)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(resp)
		return
	}

	var delta int64
	var value float64

	if metrics.Delta != nil {
		delta = *metrics.Delta
	}
	if metrics.Value != nil {
		value = *metrics.Value
	}

	newMetrics, createErr := h.services.CreateMetrics(metrics.ID, metrics.MType, delta, value)

	if createErr != nil {
		http.Error(w, createErr.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(newMetrics)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (h *Handler) BulkUpdateMetrics(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var metricsList []models.Metrics
	body, err := h.readAndVerifyRequestBody(req)
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err = json.Unmarshal(body, &metricsList); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(metricsList) == 0 {
		writeError(w, "Пустой массив", http.StatusBadRequest)
		return
	}
	var updatedMetricsList []models.Metrics

	for _, metrics := range metricsList {
		if metrics.MType != models.Counter && metrics.MType != models.Gauge || (metrics.Delta == nil && metrics.Value == nil) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		updatedMetrics, err := h.services.CreateOrUpdateMetrics(metrics)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		updatedMetricsList = append(updatedMetricsList, *updatedMetrics)
	}

	resp, err := json.Marshal(updatedMetricsList)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (h *Handler) readAndVerifyRequestBody(req *http.Request) ([]byte, error) {
	defer req.Body.Close()

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	if isGzipBody(body) {
		gz, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		defer gz.Close()

		body, err = io.ReadAll(gz)
		if err != nil {
			return nil, err
		}
	}

	if h.key == "" {
		return body, nil
	}

	receivedHash := req.Header.Get(hashutil.HeaderName)
	if receivedHash != "" && !hashutil.IsValid(body, h.key, receivedHash) {
		return nil, fmt.Errorf("invalid request hash")
	}

	return body, nil
}

func isGzipBody(body []byte) bool {
	return len(body) >= 2 && body[0] == 0x1f && body[1] == 0x8b
}

func writeError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	w.Write([]byte(message))
}

type hashResponseWriter struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func newHashResponseWriter() *hashResponseWriter {
	return &hashResponseWriter{
		header: make(http.Header),
		status: http.StatusOK,
	}
}

func (w *hashResponseWriter) Header() http.Header {
	return w.header
}

func (w *hashResponseWriter) Write(body []byte) (int, error) {
	return w.body.Write(body)
}

func (w *hashResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
}

func (h Handler) HashResponseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if h.key == "" {
			next.ServeHTTP(w, req)
			return
		}

		buffered := newHashResponseWriter()
		next.ServeHTTP(buffered, req)

		for key, values := range buffered.Header() {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		w.Header().Set(hashutil.HeaderName, hashutil.HashBody(buffered.body.Bytes(), h.key))
		w.WriteHeader(buffered.status)
		_, _ = w.Write(buffered.body.Bytes())
	})
}
