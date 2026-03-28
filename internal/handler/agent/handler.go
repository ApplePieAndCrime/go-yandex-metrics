package handler

import (
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"time"
)

func Init() {
	http.HandleFunc("/", GetMetrics)
}

func GetMetrics(w http.ResponseWriter, req *http.Request) {
	var m runtime.MemStats
	polInterval := 2
	reportInterval := 10
	runtime.ReadMemStats(&m)
	client := &http.Client{
		Timeout: time.Duration(reportInterval) * time.Second,
	}

	// for gauge
	gaugeMetrics := map[string]float64{
		"Alloc":         float64(m.Alloc),
		"BuckHashSys":   float64(m.BuckHashSys),
		"Frees":         float64(m.Frees),
		"GCCPUFraction": float64(m.GCCPUFraction),
		"GCSys":         float64(m.GCSys),
		"HeapAlloc":     float64(m.HeapAlloc),
		"HeapInuse":     float64(m.HeapInuse),
		"HeapObjects":   float64(m.HeapObjects),
		"HeapReleased":  float64(m.HeapReleased),
		"LastGC":        float64(m.LastGC),
		"Lookups":       float64(m.Lookups),
		"MCacheInuse":   float64(m.MCacheInuse),
		"MCacheSys":     float64(m.MCacheSys),
		"MSpanInuse":    float64(m.MSpanInuse),
		"MSpanSys":      float64(m.MSpanSys),
		"Mallocs":       float64(m.Mallocs),
		"NextGC":        float64(m.NextGC),
		"NumForcedGC":   float64(m.NumForcedGC),
		"NumGC":         float64(m.NumGC),
		"OtherSys":      float64(m.OtherSys),
		"PauseTotalNs":  float64(m.PauseTotalNs),
		"StackInuse":    float64(m.StackInuse),
		"StackSys":      float64(m.StackSys),
		"Sys":           float64(m.Sys),
		"TotalAlloc":    float64(m.TotalAlloc),
	}
	for metricName, metricValue := range gaugeMetrics {
		SendRequestToServer(*client, "gauge", metricName, fmt.Sprintf("%f", metricValue))
	}

	SendRequestToServer(*client, "counter", "PollCount", fmt.Sprintf("%d", polInterval))
	SendRequestToServer(*client, "gauge", "RandomValue", fmt.Sprintf("%d", rand.Intn(100)))

}

func SendRequestToServer(client http.Client, metricType string, metricName string, metricValue string) {
	url := fmt.Sprintf("http://localhost:8080/update/%s/%s/%s", metricType, metricName, metricValue)
	_, err := client.Post(url, "text/plain", nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	return
}
