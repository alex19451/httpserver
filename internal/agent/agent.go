package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"runtime"
	"time"

	"github.com/alex19451/httpserver/internal/config"
	"github.com/alex19451/httpserver/internal/models"
	"github.com/alex19451/httpserver/internal/retry"
	"github.com/alex19451/httpserver/internal/signature"
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
		Bool("has_key", a.cfg.Key != "").
		Msg("agent started")

	a.waitForServer()

	count := 1
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	a.logger.Info().Msg("sending initial metrics")
	if err := retry.DoWithRetry(func() error {
		return a.sendBatch(count, mem)
	}); err != nil {
		a.logger.Error().Err(err).Msg("failed to send initial metrics")
	} else {
		a.logger.Info().Msg("initial metrics sent successfully")
	}

	pollTicker := time.NewTicker(pollInterval)
	reportTicker := time.NewTicker(reportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	for {
		select {
		case <-pollTicker.C:
			count++
			runtime.ReadMemStats(&mem)

		case <-reportTicker.C:
			a.logger.Info().Msg("sending metrics")
			if err := retry.DoWithRetry(func() error {
				return a.sendBatch(count, mem)
			}); err != nil {
				a.logger.Error().Err(err).Msg("failed to send metrics")
			} else {
				a.logger.Info().Msg("metrics sent successfully")
			}
		}
	}
}

func (a *Agent) waitForServer() {
	url := fmt.Sprintf("http://%s/ping", a.cfg.Address)
	client := &http.Client{Timeout: 2 * time.Second}

	backoffs := []time.Duration{1 * time.Second, 2 * time.Second, 3 * time.Second, 5 * time.Second}

	for i, backoff := range backoffs {
		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			a.logger.Info().Msg("server is ready")
			return
		}
		if resp != nil {
			resp.Body.Close()
		}

		if i < len(backoffs)-1 {
			a.logger.Warn().Dur("backoff", backoff).Msg("waiting for server")
			ticker := time.NewTicker(backoff)
			<-ticker.C
			ticker.Stop()
		}
	}

	a.logger.Warn().Msg("server may not be ready, continuing anyway")
}

func (a *Agent) sendBatch(pollCount int, mem runtime.MemStats) error {
	var metrics []models.Metrics

	pollCountValue := int64(pollCount)
	metrics = append(metrics, models.Metrics{
		ID:    "PollCount",
		MType: "counter",
		Delta: &pollCountValue,
	})

	randomValue := rand.Float64()
	metrics = append(metrics, models.Metrics{
		ID:    "RandomValue",
		MType: "gauge",
		Value: &randomValue,
	})

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
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: "gauge",
			Value: &val,
		})
	}

	return a.sendBatchRequest(metrics)
}

func (a *Agent) sendBatchRequest(metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	url := fmt.Sprintf("http://%s/updates/", a.cfg.Address)

	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("marshal batch: %w", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return fmt.Errorf("compress batch: %w", err)
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

	if a.cfg.Key != "" {
		hash := signature.CalculateHash(buf.Bytes(), a.cfg.Key)
		req.Header.Set("HashSHA256", hash)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send batch: %w", err)
	}
	defer resp.Body.Close()

	if a.cfg.Key != "" {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read response body: %w", err)
		}
		expectedHash := resp.Header.Get("HashSHA256")
		if expectedHash != "" && !signature.VerifyHash(bodyBytes, expectedHash, a.cfg.Key) {
			return fmt.Errorf("invalid response signature")
		}
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("batch status: %d", resp.StatusCode)
	}

	return nil
}
