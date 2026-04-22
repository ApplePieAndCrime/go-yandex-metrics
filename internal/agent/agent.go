package internal_agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"

	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
)

type AgentMetrics struct {
	PollCount   int64
	MemStats    runtime.MemStats
	RandomValue float64
}

func collectMetrics(metrics *AgentMetrics, randomFloat func() float64) {
	runtime.ReadMemStats(&metrics.MemStats)
	metrics.PollCount++
	metrics.RandomValue = randomFloat()
}

func RunAgent(externalAddress *string, pollCount *int64, reportInterval *int64) {
	currentPollInterval := time.Duration(*pollCount) * time.Second
	currentReportInterval := time.Duration(*reportInterval) * time.Second

	client := &http.Client{
		Timeout: currentReportInterval,
	}

	pollTicker := time.NewTicker(currentPollInterval)
	reportTicker := time.NewTicker(currentReportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	metrics := &AgentMetrics{}
	var mu sync.RWMutex

	go func() {
		for range pollTicker.C {
			mu.Lock()
			collectMetrics(metrics, rand.Float64)
			mu.Unlock()
			fmt.Println("metrics collected")
		}
	}()

	for range reportTicker.C {
		mu.RLock()
		snapshot := *metrics
		mu.RUnlock()

		sendAllMetrics(client, *externalAddress, &snapshot)
		fmt.Println("metrics sent")
	}
}

func sendAllMetrics(client *http.Client, baseUrl string, metrics *AgentMetrics) ([]string, []string) {

	memMetrics := map[string]struct {
		Type  string
		Value string
	}{
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
	}
	resBody := []string{}
	errors := []string{}
	for metricName, metric := range memMetrics {
		resp, _, err := SendRequestToServer(client, baseUrl, metric.Type, metricName, metric.Value)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Error sending mem metric %s: %v\n", metricName, err))
			continue
		}

		resBody = append(resBody, fmt.Sprintf("%v", resp))
		resp.Body.Close()
	}

	return resBody, errors
}

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

func SendRequestToServer(client *http.Client, baseUrl string, metricType string, metricName string, metricValue string) (*http.Response, int, error) {
	url := fmt.Sprintf("%s/update/", baseUrl)

	payload := models.Metrics{
		ID:    metricName,
		MType: metricType,
	}

	switch metricType {
	case models.Counter:
		delta, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return nil, http.StatusBadRequest, err
		}
		payload.Delta = &delta
	case models.Gauge:
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return nil, http.StatusBadRequest, err
		}
		payload.Value = &value
	default:
		return nil, http.StatusBadRequest, fmt.Errorf("unsupported metric type: %s", metricType)
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

	resp, err := client.Do(req)

	if err != nil {
		fmt.Println(err)
		return nil, http.StatusInternalServerError, err
	}

	defer resp.Body.Close()

	out, err := Unzip(resp)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	resp.Body = io.NopCloser(bytes.NewReader(out))

	return resp, resp.StatusCode, nil
}
