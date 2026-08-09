package internal_agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rsa"
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

	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/cryptoutil"
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
	mu           sync.RWMutex
	sendMu       sync.Mutex
	data         AgentMetrics
	revision     uint64
	sentRevision uint64
}

var retryIntervals = []time.Duration{
	time.Second,
	3 * time.Second,
	5 * time.Second,
}

func (m *SafeMetrics) collect(randomFloat func() float64) error {
	memStats, memErr := mem.VirtualMemory()
	cpuStats, cpuErr := cpu.Percent(0, true)

	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(&m.data.MemStats)
	m.data.PollCount++
	m.data.RandomValue = randomFloat()
	m.revision++

	if memErr == nil && memStats != nil {
		m.data.TotalMemory = memStats.Total
		m.data.FreeMemory = memStats.Free
	}

	if cpuErr == nil {
		m.data.CPUutilization = append(m.data.CPUutilization[:0], cpuStats...)
	}

	return errors.Join(memErr, cpuErr)
}

// Snapshot возвращает независимую копию текущих метрик агента.
func (m *SafeMetrics) Snapshot() AgentMetrics {
	snapshot, _ := m.snapshotWithRevision()
	return snapshot
}

func (m *SafeMetrics) snapshotWithRevision() (AgentMetrics, uint64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := m.data

	snapshot.CPUutilization = append(
		[]float64(nil),
		m.data.CPUutilization...,
	)

	return snapshot, m.revision
}

func (m *SafeMetrics) markSent(revision uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if revision > m.sentRevision {
		m.sentRevision = revision
	}
}

func (m *SafeMetrics) hasUnsent(revision uint64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return revision > m.sentRevision
}

// RunAgent запускает периодический сбор и отправку метрик на сервер.
func RunAgent(
	ctx context.Context,
	externalAddress string,
	pollCount int64,
	reportInterval int64,
	key string,
	rateLimit int,
	cryptoKeyPath string,
) error {
	var publicKey *rsa.PublicKey
	if cryptoKeyPath != "" {
		var err error
		publicKey, err = cryptoutil.LoadPublicKey(cryptoKeyPath)
		if err != nil {
			return fmt.Errorf("load agent crypto key: %w", err)
		}
	}

	// Даём серверу время на запуск, но не задерживаем штатное завершение агента.
	startupTimer := time.NewTimer(2 * time.Second)
	select {
	case <-ctx.Done():
		if !startupTimer.Stop() {
			<-startupTimer.C
		}
		return nil
	case <-startupTimer.C:
	}

	pool := workerPool.NewPool(rateLimit)
	pool.Start()

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

	for {
		select {
		case <-ctx.Done():
			pool.Close()
			pool.Wait()

			flushed, err := sendUnsentMetrics(client, externalAddress, metrics, key, publicKey)
			if err != nil {
				return fmt.Errorf("send metrics during shutdown: %w", err)
			}
			if flushed {
				log.Println("metrics sent during shutdown")
			}
			return nil

		case <-pollTicker.C:
			pool.AddTask(func() {
				if err := metrics.collect(rand.Float64); err != nil {
					log.Printf("metrics collection error: %v", err)
					return
				}
				log.Println("metrics collected")
			})

		case <-reportTicker.C:
			pool.AddTask(func() {
				sent, err := sendUnsentMetrics(client, externalAddress, metrics, key, publicKey)
				if err != nil {
					log.Printf("metrics send error: %v", err)
					return
				}
				if sent {
					log.Println("metrics sent")
				}
			})
		}
	}
}

func sendUnsentMetrics(
	client *http.Client,
	externalAddress string,
	metrics *SafeMetrics,
	key string,
	publicKey *rsa.PublicKey,
) (bool, error) {
	// Мьютекс намеренно удерживается во время HTTP-запроса и ретраев: отправки
	// выполняются последовательно даже при rateLimit > 1. Здесь корректный
	// порядок снимков важнее параллельной отправки; сбор метрик не блокируется.
	metrics.sendMu.Lock()
	defer metrics.sendMu.Unlock()

	snapshot, revision := metrics.snapshotWithRevision()
	if !metrics.hasUnsent(revision) {
		return false, nil
	}

	if _, err := sendAllMetricsWithRetry(client, externalAddress, &snapshot, time.Sleep, key, publicKey); err != nil {
		return false, err
	}

	metrics.markSent(revision)
	return true, nil
}

// MemMetrics сопоставляет имени метрики её тип и строковое значение.
type MemMetrics map[string]struct {
	Type  string
	Value string
}

func sendAllMetrics(client *http.Client, baseUrl string, metrics *AgentMetrics, key string, publicKey *rsa.PublicKey) ([]byte, error) {

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

	resp, _, err := SendRequestToServer(client, baseUrl, memMetrics, key, publicKey)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &httpStatusError{statusCode: resp.StatusCode}
	}

	return resBody, nil
}

type httpStatusError struct {
	statusCode int
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("server returned status %d", e.statusCode)
}

func sendAllMetricsWithRetry(client *http.Client, baseUrl string, metrics *AgentMetrics, sleep func(time.Duration), key string, publicKey *rsa.PublicKey) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt <= len(retryIntervals); attempt++ {
		resBody, err := sendAllMetrics(client, baseUrl, metrics, key, publicKey)
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
	var statusErr *httpStatusError
	if errors.As(err, &statusErr) {
		return statusErr.statusCode == http.StatusTooManyRequests ||
			statusErr.statusCode >= http.StatusInternalServerError
	}

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
func SendRequestToServer(client *http.Client, baseUrl string, updatedBodies MemMetrics, key string, publicKey *rsa.PublicKey) (*http.Response, int, error) {
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

	requestBody := body
	if publicKey != nil {
		requestBody, err = cryptoutil.Encrypt(publicKey, body)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("encrypt request: %w", err)
		}
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(requestBody))
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
