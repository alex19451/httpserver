package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

var gauges = make(map[string]float64)
var counters = make(map[string]int64)

func oldUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	if r.Header.Get("Content-Type") != "text/plain" {
		w.WriteHeader(400)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	if len(parts) != 4 || parts[0] != "update" {
		w.WriteHeader(404)
		return
	}

	metricType := parts[1]
	metricName := parts[2]
	metricValue := parts[3]

	if metricName == "" {
		w.WriteHeader(404)
		return
	}

	if metricType == "gauge" {
		val, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		gauges[metricName] = val
	} else if metricType == "counter" {
		val, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		counters[metricName] += val
	} else {
		w.WriteHeader(400)
		return
	}

	w.WriteHeader(200)
}

func newUpdateHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")
	value := chi.URLParam(r, "value")

	if name == "" {
		w.WriteHeader(404)
		return
	}

	if metricType == "gauge" {
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		gauges[name] = val
	} else if metricType == "counter" {
		val, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		counters[name] += val
	} else {
		w.WriteHeader(400)
		return
	}

	w.WriteHeader(200)
}

func valueHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")

	if metricType == "gauge" {
		if val, exists := gauges[name]; exists {
			w.WriteHeader(200)
			w.Write([]byte(strconv.FormatFloat(val, 'f', -1, 64)))
			return
		}
	} else if metricType == "counter" {
		if val, exists := counters[name]; exists {
			w.WriteHeader(200)
			w.Write([]byte(strconv.FormatInt(val, 10)))
			return
		}
	}

	w.WriteHeader(404)
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
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

func main() {
	r := chi.NewRouter()

	r.Post("/update/{type}/{name}/{value}", newUpdateHandler)
	r.Get("/value/{type}/{name}", valueHandler)
	r.Get("/", mainHandler)

	http.HandleFunc("/update/", oldUpdateHandler)

	fmt.Println("Server: http://localhost:8080")

	http.Handle("/", r)
	http.ListenAndServe(":8080", nil)
}
