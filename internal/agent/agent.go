package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"time"

	"github.com/alex19451/httpserver/internal/config"
	"github.com/alex19451/httpserver/internal/models"
	"github.com/rs/zerolog"
)

type Agent struct {
	cfg    *config.AgentConfig
	logger zerolog.Logger
}

func New(cfg *config.AgentConfig, logger zerolog.Logger) *Agent {
	return &Agent{
		cfg:    cfg,
		logger: logger,
	}
}

func (a *Agent) Run() {
	pollInterval := time.Duration(a.cfg.PollInterval) * time.Second
	reportInterval := time.Duration(a.cfg.ReportInterval) * time.Second

	a.logger.Info().
		Str("address", a.cfg.Address).
		Dur("poll_interval", pollInterval).
		Dur("report_interval", reportInterval).
		Msg("agent started")

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
			a.logger.Info().Msg("sending metrics")
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
		if err := a.sendAll(pollCount, mem); err == nil {
			return
		}
		a.logger.Warn().
			Dur("backoff", backoff).
			Msg("failed to send metrics, retrying")
		time.Sleep(backoff)
	}
	a.logger.Error().Msg("failed to send metrics after all retries")
}

func (a *Agent) sendAll(pollCount int, mem runtime.MemStats) error {
	pollCountValue := int64(pollCount)
	if err := a.sendJSON("counter", "PollCount", &pollCountValue, nil); err != nil {
		return err
	}

	randomValue := rand.Float64()
	if err := a.sendJSON("gauge", "RandomValue", nil, &randomValue); err != nil {
		return err
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
		val := value
		if err := a.sendJSON("gauge", name, nil, &val); err != nil {
			return err
		}
	}

	return nil
}

func (a *Agent) sendJSON(mtype, name string, delta *int64, value *float64) error {
	url := fmt.Sprintf("http://%s/update/", a.cfg.Address)

	metrics := models.Metrics{
		ID:    name,
		MType: mtype,
		Delta: delta,
		Value: value,
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("marshal metric %s: %w", name, err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return fmt.Errorf("compress data for %s: %w", name, err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("close gzip for %s: %w", name, err)
	}

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return fmt.Errorf("create request for %s: %w", name, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send metric %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("response for %s: %d", name, resp.StatusCode)
	}

	reader := resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return fmt.Errorf("create gzip reader for %s: %w", name, err)
		}
		defer gz.Close()
		reader = gz
	}

	var respMetrics models.Metrics
	if err := json.NewDecoder(reader).Decode(&respMetrics); err != nil {
		return fmt.Errorf("decode response for %s: %w", name, err)
	}

	a.logger.Debug().
		Str("metric", name).
		Str("type", mtype).
		Msg("metric sent successfully")

	return nil
}
