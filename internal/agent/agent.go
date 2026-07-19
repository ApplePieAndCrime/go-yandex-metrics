package internal_agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/hashutil"
	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
	workerPool "github.com/ApplePieAndCrime/go-yandex-metrics/internal/workerpool"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

// AgentMetrics содержит снимок метрик среды выполнения и операционной системы.
type AgentMetrics struct {
	PollCount      int64
	MemStats       runtime.MemStats
	RandomValue    float64
	TotalMemory    uint64
	FreeMemory     uint64
	CPUutilization []float64
}

// SafeMetrics обеспечивает конкурентно-безопасное хранение снимка метрик агента.
type SafeMetrics struct {
	mu   sync.RWMutex
	data AgentMetrics
}

var retryIntervals = []time.Duration{
	time.Second,
	3 * time.Second,
	5 * time.Second,
}

func (m *SafeMetrics) collect(randomFloat func() float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(&m.data.MemStats)
	m.data.PollCount++
	m.data.RandomValue = randomFloat()

	memStats, _ := mem.VirtualMemory()
	m.data.TotalMemory = memStats.Total
	m.data.FreeMemory = memStats.Free

	cpuStats, _ := cpu.Percent(0, true)
	m.data.CPUutilization = cpuStats
}

// Snapshot возвращает независимую копию текущих метрик агента.
func (m *SafeMetrics) Snapshot() AgentMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := m.data

	snapshot.CPUutilization = append(
		[]float64(nil),
		m.data.CPUutilization...,
	)

	return snapshot
}

// RunAgent запускает периодический сбор и отправку метрик на сервер.
func RunAgent(
	externalAddress string,
	pollCount int64,
	reportInterval int64,
	key string,
	rateLimit int,
) error {
	// Даём серверу время на запуск (2 секунды достаточно)
	time.Sleep(2 * time.Second)

	pool := workerPool.NewPool(rateLimit)
	pool.Start()

	defer func() {
		pool.Close()
		pool.Wait()
	}()

	currentPollInterval := time.Duration(pollCount) * time.Second
	currentReportInterval := time.Duration(reportInterval) * time.Second

	client := &http.Client{
		Timeout: currentReportInterval,
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}

	pollTicker := time.NewTicker(currentPollInterval)
	reportTicker := time.NewTicker(currentReportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	metrics := &SafeMetrics{}

	go func() {
		for range pollTicker.C {
			pool.AddTask(func() {
				metrics.collect(rand.Float64)
				log.Println("metrics collected")
			})
		}
	}()

	for range reportTicker.C {
		pool.AddTask(func() {
			snapshot := metrics.Snapshot()

			if _, err := sendAllMetricsWithRetry(client, externalAddress, &snapshot, time.Sleep, key); err != nil {
				pool.AddError(fmt.Errorf("metrics send error:%v", err))
				return
			}
			log.Println("metrics sent")
		})
	}

	pool.Wait()

	errors := pool.Errors()
	if len(errors) > 0 {
		return fmt.Errorf("errors: %v", errors)
	}

	return nil

}

// MemMetrics сопоставляет имени метрики её тип и строковое значение.
type MemMetrics map[string]struct {
	Type  string
	Value string
}

func sendAllMetrics(client *http.Client, baseUrl string, metrics *AgentMetrics, key string) ([]byte, error) {

	memMetrics := MemMetrics{
		"Alloc":         {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.Alloc)},
		"BuckHashSys":   {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.BuckHashSys)},
		"Frees":         {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.Frees)},
		"GCCPUFraction": {Type: "gauge", Value: fmt.Sprintf("%f", metrics.MemStats.GCCPUFraction)},
		"GCSys":         {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.GCSys)},
		"HeapAlloc":     {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.HeapAlloc)},
		"HeapIdle":      {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.HeapIdle)},
		"HeapInuse":     {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.HeapInuse)},
		"HeapObjects":   {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.HeapObjects)},
		"HeapReleased":  {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.HeapReleased)},
		"HeapSys":       {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.HeapSys)},
		"LastGC":        {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.LastGC)},
		"Lookups":       {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.Lookups)},
		"MCacheInuse":   {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.MCacheInuse)},
		"MCacheSys":     {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.MCacheSys)},
		"MSpanInuse":    {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.MSpanInuse)},
		"MSpanSys":      {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.MSpanSys)},
		"Mallocs":       {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.Mallocs)},
		"NextGC":        {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.NextGC)},
		"NumForcedGC":   {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.NumForcedGC)},
		"NumGC":         {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.NumGC)},
		"OtherSys":      {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.OtherSys)},
		"PauseTotalNs":  {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.PauseTotalNs)},
		"StackInuse":    {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.StackInuse)},
		"StackSys":      {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.StackSys)},
		"Sys":           {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.Sys)},
		"TotalAlloc":    {Type: "gauge", Value: fmt.Sprintf("%d", metrics.MemStats.TotalAlloc)},
		"RandomValue":   {Type: "gauge", Value: fmt.Sprintf("%f", metrics.RandomValue)},
		"PollCount":     {Type: "counter", Value: fmt.Sprintf("%d", metrics.PollCount)},
		"TotalMemory":   {Type: "gauge", Value: fmt.Sprintf("%d", metrics.TotalMemory)},
		"FreeMemory":    {Type: "gauge", Value: fmt.Sprintf("%d", metrics.FreeMemory)},
	}

	for i, cpuUtil := range metrics.CPUutilization {
		metricName := fmt.Sprintf("CPUutilization%d", i+1)
		memMetrics[metricName] = struct {
			Type  string
			Value string
		}{
			Type:  "gauge",
			Value: fmt.Sprintf("%f", cpuUtil),
		}
	}

	resp, _, err := SendRequestToServer(client, baseUrl, memMetrics, key)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return resBody, nil
}

func sendAllMetricsWithRetry(client *http.Client, baseUrl string, metrics *AgentMetrics, sleep func(time.Duration), key string) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt <= len(retryIntervals); attempt++ {
		resBody, err := sendAllMetrics(client, baseUrl, metrics, key)
		if err == nil {
			return resBody, nil
		}

		lastErr = err
		if !isRetriableRequestError(err) || attempt == len(retryIntervals) {
			return nil, err
		}

		sleep(retryIntervals[attempt])
	}

	return nil, lastErr
}

func isRetriableRequestError(err error) bool {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}

	var netErr net.Error
	return errors.As(err, &netErr)
}

// Unzip читает тело HTTP-ответа и при необходимости распаковывает gzip.
func Unzip(resp *http.Response) ([]byte, error) {
	var reader io.Reader = resp.Body

	if resp.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		reader = gz
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// UpdatedMetrics содержит разобранные параметры обновления метрики.
type UpdatedMetrics struct {
	metricType  string
	metricName  string
	metricValue string
}

// SendRequestToServer отправляет набор метрик на эндпоинт пакетного обновления.
func SendRequestToServer(client *http.Client, baseUrl string, updatedBodies MemMetrics, key string) (*http.Response, int, error) {
	url := fmt.Sprintf("%s/updates/", baseUrl)

	var payload []models.Metrics
	for metricName, updatedBody := range updatedBodies {

		updatedMetrics := models.Metrics{
			ID:    metricName,
			MType: updatedBody.Type,
		}

		switch updatedBody.Type {
		case models.Counter:
			delta, err := strconv.ParseInt(updatedBody.Value, 10, 64)
			if err != nil {
				return nil, http.StatusBadRequest, err
			}
			updatedMetrics.Delta = &delta
		case models.Gauge:
			value, err := strconv.ParseFloat(updatedBody.Value, 64)
			if err != nil {
				return nil, http.StatusBadRequest, err
			}
			updatedMetrics.Value = &value
		default:
			return nil, http.StatusBadRequest, fmt.Errorf("unsupported metric type: %s", updatedBody.Type)
		}
		payload = append(payload, updatedMetrics)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")

	if key != "" {
		req.Header.Set(hashutil.HeaderName, hashutil.SignBody(body, key))
	}

	resp, err := client.Do(req)

	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("send request: %w", err)
	}

	defer resp.Body.Close()

	out, err := Unzip(resp)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	resp.Body = io.NopCloser(bytes.NewReader(out))

	return resp, resp.StatusCode, nil
}
