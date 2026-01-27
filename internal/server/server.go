package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/alex19451/httpserver/internal/config"
	"github.com/alex19451/httpserver/internal/storage"
)

type Server struct {
	cfg *config.ServerConfig
	db  *storage.Storage
}

func New(cfg *config.ServerConfig, db *storage.Storage) *Server {
	return &Server{cfg: cfg, db: db}
}

func (s *Server) Run() error {
	http.HandleFunc("/update/", s.update)
	http.HandleFunc("/value/", s.getValue)
	http.HandleFunc("/", s.getAll)

	fmt.Printf("Server starting on http://%s\n", s.cfg.Address)
	return http.ListenAndServe(s.cfg.Address, nil)
}

func (s *Server) update(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(405)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 4 || parts[0] != "update" {
		w.WriteHeader(404)
		return
	}

	metricType := parts[1]
	name := parts[2]
	value := parts[3]

	if metricType == "gauge" {
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		s.db.Gauges[name] = val
	} else if metricType == "counter" {
		val, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		s.db.Counters[name] += val
	} else {
		w.WriteHeader(400)
		return
	}

	w.WriteHeader(200)
}

func (s *Server) getValue(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "value" {
		w.WriteHeader(404)
		return
	}

	metricType := parts[1]
	name := parts[2]

	if metricType == "gauge" {
		if val, ok := s.db.Gauges[name]; ok {
			w.Write([]byte(strconv.FormatFloat(val, 'f', -1, 64)))
			return
		}
	} else if metricType == "counter" {
		if val, ok := s.db.Counters[name]; ok {
			w.Write([]byte(strconv.FormatInt(val, 10)))
			return
		}
	}

	w.WriteHeader(404)
}

func (s *Server) getAll(w http.ResponseWriter, r *http.Request) {
	html := `<html><body><h1>Metrics</h1>`

	html += `<h2>Gauges</h2><ul>`
	for name, val := range s.db.Gauges {
		html += fmt.Sprintf("<li>%s: %f</li>", name, val)
	}
	html += `</ul>`

	html += `<h2>Counters</h2><ul>`
	for name, val := range s.db.Counters {
		html += fmt.Sprintf("<li>%s: %d</li>", name, val)
	}
	html += `</ul></body></html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
