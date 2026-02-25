package server

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/alex19451/httpserver/internal/config"
	"github.com/alex19451/httpserver/internal/models"
	"github.com/alex19451/httpserver/internal/storage"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	cfg *config.ServerConfig
	db  *storage.Storage
}

func New(cfg *config.ServerConfig, db *storage.Storage) *Server {
	return &Server{cfg: cfg, db: db}
}

func (s *Server) Run() error {
	r := chi.NewRouter()

	r.Use(LoggingMiddleware)
	r.Use(GzipMiddleware)

	r.Post("/update/{type}/{name}/{value}", s.update)
	r.Get("/value/{type}/{name}", s.getValue)

	r.Post("/update/", s.updateJSON)
	r.Post("/value/", s.valueJSON)

	r.Get("/", s.getAll)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	fmt.Printf("Server starting on http://%s\n", s.cfg.Address)
	return http.ListenAndServe(s.cfg.Address, r)
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
		s.db.Gauges[name] = val
		w.WriteHeader(http.StatusOK)
	} else if metricType == "counter" {
		val, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		s.db.Counters[name] += val
		w.WriteHeader(http.StatusOK)
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
		s.db.Gauges[metrics.ID] = *metrics.Value

		resp := models.Metrics{
			ID:    metrics.ID,
			MType: metrics.MType,
			Value: metrics.Value,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)

	} else if metrics.MType == "counter" {
		if metrics.Delta == nil {
			http.Error(w, "delta is required for counter", http.StatusBadRequest)
			return
		}
		s.db.Counters[metrics.ID] += *metrics.Delta

		delta := s.db.Counters[metrics.ID]
		resp := models.Metrics{
			ID:    metrics.ID,
			MType: metrics.MType,
			Delta: &delta,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)

	} else {
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}
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
		val, ok := s.db.Gauges[metrics.ID]
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
		json.NewEncoder(w).Encode(resp)

	} else if metrics.MType == "counter" {
		val, ok := s.db.Counters[metrics.ID]
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
		json.NewEncoder(w).Encode(resp)

	} else {
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}
}

func (s *Server) getValue(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")

	if metricType == "gauge" {
		if val, ok := s.db.Gauges[name]; ok {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(strconv.FormatFloat(val, 'f', -1, 64)))
			return
		}
	} else if metricType == "counter" {
		if val, ok := s.db.Counters[name]; ok {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(strconv.FormatInt(val, 10)))
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
}

func (s *Server) getAll(w http.ResponseWriter, r *http.Request) {
	html := `<html><body><h1>Metrics</h1><h2>Gauges</h2><ul>`

	for name, val := range s.db.Gauges {
		html += fmt.Sprintf("<li>%s: %f</li>", name, val)
	}
	html += `</ul><h2>Counters</h2><ul>`

	for name, val := range s.db.Counters {
		html += fmt.Sprintf("<li>%s: %d</li>", name, val)
	}
	html += `</ul></body></html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
