package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/alex19451/httpserver/internal/config"
	"github.com/alex19451/httpserver/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type Server struct {
	cfg *config.ServerConfig
	db  *storage.Storage
}

func New(cfg *config.ServerConfig, db *storage.Storage) *Server {
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.TextFormatter{})

	return &Server{cfg: cfg, db: db}
}

func (s *Server) Run() error {
	r := chi.NewRouter()

	r.Use(LoggingMiddleware)

	r.Post("/update/{type}/{name}/{value}", s.update)
	r.Get("/value/{type}/{name}", s.getValue)
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
