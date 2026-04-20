package server

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alex19451/httpserver/internal/config"
	"github.com/alex19451/httpserver/internal/models"
	"github.com/alex19451/httpserver/internal/signature"
	"github.com/alex19451/httpserver/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

type Server struct {
	cfg    *config.ServerConfig
	db     *storage.Storage
	logger zerolog.Logger
}

func New(cfg *config.ServerConfig, db *storage.Storage, logger zerolog.Logger) *Server {
	return &Server{
		cfg:    cfg,
		db:     db,
		logger: logger,
	}
}

func (s *Server) Run() error {
	if s.cfg.Restore && s.cfg.FileStoragePath != "" {
		if err := s.db.LoadFromFile(); err != nil {
			s.logger.Error().Err(err).Msg("error loading from file")
		}
	}

	if s.cfg.StoreInterval == 0 && s.cfg.FileStoragePath != "" {
		s.logger.Info().Msg("sync save mode enabled")
	} else if s.cfg.StoreInterval > 0 && s.cfg.FileStoragePath != "" {
		go func() {
			ticker := time.NewTicker(time.Duration(s.cfg.StoreInterval) * time.Second)
			defer ticker.Stop()

			for range ticker.C {
				if err := s.db.SaveToFile(); err != nil {
					s.logger.Error().Err(err).Msg("error saving to file")
				} else {
					s.logger.Info().Msg("metrics saved to file")
				}
			}
		}()
	}

	r := chi.NewRouter()

	r.Use(LoggingMiddleware(s.logger))
	r.Use(GzipMiddleware)                 // Сначала сжатие
	r.Use(SignatureMiddleware(s.cfg.Key)) // Потом подпись

	r.Post("/update/{type}/{name}/{value}", s.update)
	r.Get("/value/{type}/{name}", s.getValue)

	r.Post("/update/", s.updateJSON)
	r.Post("/updates/", s.batchUpdate)
	r.Post("/value/", s.valueJSON)

	r.Get("/ping", s.ping)
	r.Get("/", s.getAll)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	s.logger.Info().
		Str("address", s.cfg.Address).
		Int("store_interval", s.cfg.StoreInterval).
		Str("file_path", s.cfg.FileStoragePath).
		Bool("restore", s.cfg.Restore).
		Str("database_dsn", s.cfg.DatabaseDSN).
		Bool("has_key", s.cfg.Key != "").
		Msg("server starting")

	return http.ListenAndServe(s.cfg.Address, r)
}

func (s *Server) ping(w http.ResponseWriter, r *http.Request) {
	if err := s.db.Ping(); err != nil {
		s.logger.Error().Err(err).Msg("ping failed")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) update(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")
	value := chi.URLParam(r, "value")

	if name == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if metricType == "gauge" {
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := s.db.UpdateGauge(name, val); err != nil {
			s.logger.Error().Err(err).Msg("update gauge failed")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

		if s.cfg.StoreInterval == 0 && s.cfg.FileStoragePath != "" {
			if err := s.db.SaveToFile(); err != nil {
				s.logger.Error().Err(err).Msg("error saving to file")
			}
		}

	} else if metricType == "counter" {
		val, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if _, err := s.db.UpdateCounter(name, val); err != nil {
			s.logger.Error().Err(err).Msg("update counter failed")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

		if s.cfg.StoreInterval == 0 && s.cfg.FileStoragePath != "" {
			if err := s.db.SaveToFile(); err != nil {
				s.logger.Error().Err(err).Msg("error saving to file")
			}
		}

	} else {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

func (s *Server) updateJSON(w http.ResponseWriter, r *http.Request) {
	body := r.Body
	if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer gz.Close()
		body = gz
	}

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	var metrics models.Metrics
	if err := json.NewDecoder(body).Decode(&metrics); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if metrics.ID == "" || metrics.MType == "" {
		http.Error(w, "id and type are required", http.StatusBadRequest)
		return
	}

	if metrics.MType == "gauge" {
		if metrics.Value == nil {
			http.Error(w, "value is required for gauge", http.StatusBadRequest)
			return
		}
		if err := s.db.UpdateGauge(metrics.ID, *metrics.Value); err != nil {
			s.logger.Error().Err(err).Msg("update gauge failed")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if s.cfg.StoreInterval == 0 && s.cfg.FileStoragePath != "" {
			if err := s.db.SaveToFile(); err != nil {
				s.logger.Error().Err(err).Msg("error saving to file")
			}
		}

		resp := models.Metrics{
			ID:    metrics.ID,
			MType: metrics.MType,
			Value: metrics.Value,
		}
		w.Header().Set("Content-Type", "application/json")

		if s.cfg.Key != "" {
			respBytes, _ := json.Marshal(resp)
			hash := signature.CalculateHash(respBytes, s.cfg.Key)
			w.Header().Set("HashSHA256", hash)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)

	} else if metrics.MType == "counter" {
		if metrics.Delta == nil {
			http.Error(w, "delta is required for counter", http.StatusBadRequest)
			return
		}
		total, err := s.db.UpdateCounter(metrics.ID, *metrics.Delta)
		if err != nil {
			s.logger.Error().Err(err).Msg("update counter failed")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if s.cfg.StoreInterval == 0 && s.cfg.FileStoragePath != "" {
			if err := s.db.SaveToFile(); err != nil {
				s.logger.Error().Err(err).Msg("error saving to file")
			}
		}

		resp := models.Metrics{
			ID:    metrics.ID,
			MType: metrics.MType,
			Delta: &total,
		}
		w.Header().Set("Content-Type", "application/json")

		if s.cfg.Key != "" {
			respBytes, _ := json.Marshal(resp)
			hash := signature.CalculateHash(respBytes, s.cfg.Key)
			w.Header().Set("HashSHA256", hash)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)

	} else {
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}
}

func (s *Server) batchUpdate(w http.ResponseWriter, r *http.Request) {
	body := r.Body
	if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer gz.Close()
		body = gz
	}

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	var metrics []models.Metrics
	if err := json.NewDecoder(body).Decode(&metrics); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(metrics) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	if s.db.IsDB() {
		if err := s.db.BatchUpdate(metrics); err != nil {
			s.logger.Error().Err(err).Msg("batch update failed")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	} else {
		for _, metric := range metrics {
			if metric.MType == "gauge" {
				if metric.Value == nil {
					http.Error(w, fmt.Sprintf("value required for gauge %s", metric.ID), http.StatusBadRequest)
					return
				}
				s.db.UpdateGauge(metric.ID, *metric.Value)
			} else if metric.MType == "counter" {
				if metric.Delta == nil {
					http.Error(w, fmt.Sprintf("delta required for counter %s", metric.ID), http.StatusBadRequest)
					return
				}
				s.db.UpdateCounter(metric.ID, *metric.Delta)
			} else {
				http.Error(w, fmt.Sprintf("invalid type %s", metric.MType), http.StatusBadRequest)
				return
			}
		}
	}

	if s.cfg.StoreInterval == 0 && s.cfg.FileStoragePath != "" {
		if err := s.db.SaveToFile(); err != nil {
			s.logger.Error().Err(err).Msg("error saving to file")
		}
	}

	w.Header().Set("Content-Type", "application/json")

	if s.cfg.Key != "" {
		respBytes, _ := json.Marshal(map[string]string{"status": "ok"})
		hash := signature.CalculateHash(respBytes, s.cfg.Key)
		w.Header().Set("HashSHA256", hash)
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) valueJSON(w http.ResponseWriter, r *http.Request) {
	body := r.Body
	if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer gz.Close()
		body = gz
	}

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	var metrics models.Metrics
	if err := json.NewDecoder(body).Decode(&metrics); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if metrics.ID == "" || metrics.MType == "" {
		http.Error(w, "id and type are required", http.StatusBadRequest)
		return
	}

	if metrics.MType == "gauge" {
		val, ok, err := s.db.GetGauge(metrics.ID)
		if err != nil {
			s.logger.Error().Err(err).Msg("get gauge failed")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !ok {
			http.Error(w, "metric not found", http.StatusNotFound)
			return
		}

		resp := models.Metrics{
			ID:    metrics.ID,
			MType: metrics.MType,
			Value: &val,
		}

		w.Header().Set("Content-Type", "application/json")

		if s.cfg.Key != "" {
			respBytes, _ := json.Marshal(resp)
			hash := signature.CalculateHash(respBytes, s.cfg.Key)
			w.Header().Set("HashSHA256", hash)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
		return

	} else if metrics.MType == "counter" {
		val, ok, err := s.db.GetCounter(metrics.ID)
		if err != nil {
			s.logger.Error().Err(err).Msg("get counter failed")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !ok {
			http.Error(w, "metric not found", http.StatusNotFound)
			return
		}

		resp := models.Metrics{
			ID:    metrics.ID,
			MType: metrics.MType,
			Delta: &val,
		}

		w.Header().Set("Content-Type", "application/json")

		if s.cfg.Key != "" {
			respBytes, _ := json.Marshal(resp)
			hash := signature.CalculateHash(respBytes, s.cfg.Key)
			w.Header().Set("HashSHA256", hash)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
		return

	} else {
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}
}

func (s *Server) getValue(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")

	if metricType == "gauge" {
		val, ok, err := s.db.GetGauge(name)
		if err != nil {
			s.logger.Error().Err(err).Msg("get gauge failed")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(strconv.FormatFloat(val, 'f', -1, 64)))

	} else if metricType == "counter" {
		val, ok, err := s.db.GetCounter(name)
		if err != nil {
			s.logger.Error().Err(err).Msg("get counter failed")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(strconv.FormatInt(val, 10)))

	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *Server) getAll(w http.ResponseWriter, r *http.Request) {
	gauges, counters, err := s.db.GetAll()
	if err != nil {
		s.logger.Error().Err(err).Msg("get all metrics failed")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	html := `<html><body><h1>Metrics</h1><h2>Gauges</h2><ul>`

	for name, val := range gauges {
		html += fmt.Sprintf("<li>%s: %f</li>", name, val)
	}
	html += `</ul><h2>Counters</h2><ul>`

	for name, val := range counters {
		html += fmt.Sprintf("<li>%s: %d</li>", name, val)
	}
	html += `</ul></body></html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
