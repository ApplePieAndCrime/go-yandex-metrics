package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
	logger "github.com/ApplePieAndCrime/go-yandex-metrics/internal/server"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/service"
	"github.com/go-chi/chi"
)

type Handler struct {
	services *service.Service
}

func NewHandler(services *service.Service) *Handler {
	return &Handler{services: services}
}

func (h Handler) InitRoutes() *chi.Mux {
	r := chi.NewRouter()
	r.Route("/", func(r chi.Router) {
		r.Get("/", logger.WithLogging(h.GetAllMetrics))
		r.Get("/value/{metricType}/{metricName}", logger.WithLogging(h.GetMetricsByID))
		r.Get("/value", logger.WithLogging(h.GetMetricsByIDWithJSON))
		r.Post("/update/{metricType}/{metricName}/{metricValue}", logger.WithLogging(h.UpdateMetrics))
		r.Post("/update", logger.WithLogging(h.UpdateMetricsByJSON))
	})

	return r
}

func (h *Handler) GetAllMetrics(w http.ResponseWriter, req *http.Request) {
	metricsList := h.services.GetAllMetrics()

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	for _, metric := range metricsList {
		io.WriteString(w, fmt.Sprintf("%+v\n", metric))
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
	existingMetric, exists := h.services.GetMetricsByID(metricName, metricType)

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	switch existingMetric.MType {
	case models.Counter:
		io.WriteString(w, strconv.FormatInt(*existingMetric.Delta, 10))
	case models.Gauge:
		io.WriteString(w, strconv.FormatFloat(*existingMetric.Value, 'f', -1, 64))
	}
}

func (h *Handler) GetMetricsByIDWithJSON(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var metrics models.Metrics
	var buf bytes.Buffer
	// читаем тело запроса
	_, err := buf.ReadFrom(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// десериализуем JSON в Visitor
	if err = json.Unmarshal(buf.Bytes(), &metrics); err != nil {
		fmt.Println("err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if metrics.ID == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	existingMetric, exists := h.services.GetMetricsByID(metrics.ID, metrics.MType)

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)

	resp, err := json.Marshal(existingMetric)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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

	existingMetric, exists := h.services.GetMetricsByID(metricName, metricType)
	if exists {
		h.services.UpdateMetrics(existingMetric, delta, value)
		w.WriteHeader(http.StatusOK)

		resp, err := json.Marshal(existingMetric)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(resp)
	} else {
		newMetrics := h.services.CreateMetrics(metricName, metricType, delta, value)
		w.WriteHeader(http.StatusOK)

		resp, err := json.Marshal(newMetrics)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(resp)
	}

	fmt.Printf("storage: %+v\r\n", h.services.GetAllMetrics())
}

func (h *Handler) UpdateMetricsByJSON(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var metrics models.Metrics
	var buf bytes.Buffer

	_, err := buf.ReadFrom(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err = json.Unmarshal(buf.Bytes(), &metrics); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
			fmt.Println("metrics: ", metrics, "metrics delta:", *metrics.Delta)
			return
		}
	case models.Gauge:
		if metrics.Value == nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Println("metrics: ", metrics, "metrics value:", *metrics.Value)
			return
		}
	}

	existingMetric, exists := h.services.GetMetricsByID(metrics.ID, metrics.MType)
	if exists {
		var delta int64
		var value float64

		if metrics.Delta != nil {
			delta = *metrics.Delta
		}
		if metrics.Value != nil {
			value = *metrics.Value
		}

		h.services.UpdateMetrics(existingMetric, delta, value)
		w.WriteHeader(http.StatusOK)

		resp, err := json.Marshal(existingMetric)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

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

	newMetrics := h.services.CreateMetrics(metrics.ID, metrics.MType, delta, value)
	w.WriteHeader(http.StatusOK)

	resp, err := json.Marshal(newMetrics)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(resp)

	fmt.Printf("storage: %+v\r\n", h.services.GetAllMetrics())
}
