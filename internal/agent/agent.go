package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"time"

	"github.com/alex19451/httpserver/internal/config"
	"github.com/alex19451/httpserver/internal/models"
)

type Agent struct {
	cfg *config.AgentConfig
}

func New(cfg *config.AgentConfig) *Agent {
	return &Agent{cfg: cfg}
}

func (a *Agent) Run() {
	pollInterval := time.Duration(a.cfg.PollInterval) * time.Second
	reportInterval := time.Duration(a.cfg.ReportInterval) * time.Second

	fmt.Printf("Agent started\n")
	fmt.Printf("Server address: %s\n", a.cfg.Address)
	fmt.Printf("Poll interval: %v\n", pollInterval)
	fmt.Printf("Report interval: %v\n", reportInterval)

	count := 0
	pollTicker := time.NewTicker(pollInterval)
	reportTicker := time.NewTicker(reportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	var mem runtime.MemStats

	for {
		select {
		case <-pollTicker.C:
			count++
			runtime.ReadMemStats(&mem)

		case <-reportTicker.C:
			fmt.Println("Sending metrics")
			a.sendAllWithBackoff(count, mem)
		}
	}
}

func (a *Agent) sendAllWithBackoff(pollCount int, mem runtime.MemStats) {
	backoffSchedule := []time.Duration{
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
	}

	for _, backoff := range backoffSchedule {
		if a.sendAll(pollCount, mem) {
			return
		}
		fmt.Printf("Failed to send metrics, retrying in %v\n", backoff)
		time.Sleep(backoff)
	}
	fmt.Println("Failed to send metrics after all retries")
}

func (a *Agent) sendAll(pollCount int, mem runtime.MemStats) bool {
	success := true

	if !a.sendJSON("counter", "PollCount", int64(pollCount), nil) {
		success = false
	}

	randomValue := rand.Float64()
	if !a.sendJSON("gauge", "RandomValue", nil, &randomValue) {
		success = false
	}

	runtimeMetrics := map[string]float64{
		"Alloc":         float64(mem.Alloc),
		"BuckHashSys":   float64(mem.BuckHashSys),
		"Frees":         float64(mem.Frees),
		"GCCPUFraction": mem.GCCPUFraction,
		"GCSys":         float64(mem.GCSys),
		"HeapAlloc":     float64(mem.HeapAlloc),
		"HeapIdle":      float64(mem.HeapIdle),
		"HeapInuse":     float64(mem.HeapInuse),
		"HeapObjects":   float64(mem.HeapObjects),
		"HeapReleased":  float64(mem.HeapReleased),
		"HeapSys":       float64(mem.HeapSys),
		"LastGC":        float64(mem.LastGC),
		"Lookups":       float64(mem.Lookups),
		"MCacheInuse":   float64(mem.MCacheInuse),
		"MCacheSys":     float64(mem.MCacheSys),
		"MSpanInuse":    float64(mem.MSpanInuse),
		"MSpanSys":      float64(mem.MSpanSys),
		"Mallocs":       float64(mem.Mallocs),
		"NextGC":        float64(mem.NextGC),
		"NumForcedGC":   float64(mem.NumForcedGC),
		"NumGC":         float64(mem.NumGC),
		"OtherSys":      float64(mem.OtherSys),
		"PauseTotalNs":  float64(mem.PauseTotalNs),
		"StackInuse":    float64(mem.StackInuse),
		"StackSys":      float64(mem.StackSys),
		"Sys":           float64(mem.Sys),
		"TotalAlloc":    float64(mem.TotalAlloc),
	}

	for name, value := range runtimeMetrics {
		if !a.sendJSON("gauge", name, nil, &value) {
			success = false
		}
	}

	return success
}

func (a *Agent) sendJSON(mtype, name string, delta *int64, value *float64) bool {
	url := fmt.Sprintf("http://%s/update/", a.cfg.Address)

	metrics := models.Metrics{
		ID:    name,
		MType: mtype,
		Delta: delta,
		Value: value,
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		fmt.Printf("Error marshaling metric %s: %v\n", name, err)
		return false
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		fmt.Printf("Error creating request for %s: %v\n", name, err)
		return false
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error sending metric %s: %v\n", name, err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error response for %s: %d\n", name, resp.StatusCode)
		return false
	}

	var respMetrics models.Metrics
	if err := json.NewDecoder(resp.Body).Decode(&respMetrics); err != nil {
		fmt.Printf("Error decoding response for %s: %v\n", name, err)
		return false
	}

	fmt.Printf("Metric %s sent successfully\n", name)
	return true
}
