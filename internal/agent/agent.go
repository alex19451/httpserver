package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"time"

	"github.com/alex19451/httpserver/internal/config"
	"github.com/alex19451/httpserver/internal/models"
	"github.com/alex19451/httpserver/internal/retry"
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
			if err := retry.DoWithRetry(func() error {
				return a.sendAll(count, mem)
			}); err != nil {
				a.logger.Error().Err(err).Msg("failed to send metrics")
			} else {
				a.logger.Info().Msg("metrics sent successfully")
			}
		}
	}
}

func (a *Agent) sendAll(pollCount int, mem runtime.MemStats) error {
	var errs []error

	pollCountValue := int64(pollCount)
	if err := a.sendJSON("counter", "PollCount", &pollCountValue, nil); err != nil {
		errs = append(errs, fmt.Errorf("send PollCount: %w", err))
	}

	randomValue := rand.Float64()
	if err := a.sendJSON("gauge", "RandomValue", nil, &randomValue); err != nil {
		errs = append(errs, fmt.Errorf("send RandomValue: %w", err))
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
			errs = append(errs, fmt.Errorf("send %s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
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
		return fmt.Errorf("marshal: %w", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return fmt.Errorf("compress: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("close gzip: %w", err)
	}

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status: %d", resp.StatusCode)
	}

	reader := resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return fmt.Errorf("create gzip reader: %w", err)
		}
		defer gz.Close()
		reader = gz
	}

	var respMetrics models.Metrics
	if err := json.NewDecoder(reader).Decode(&respMetrics); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}
