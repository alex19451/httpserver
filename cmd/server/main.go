package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
)

var gauges = make(map[string]float64)
var counters = make(map[string]int64)

var serverAddress string

func updateHandler(w http.ResponseWriter, r *http.Request) {
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
		gauges[name] = val
		w.WriteHeader(http.StatusOK)
	} else if metricType == "counter" {
		val, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		counters[name] += val
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

func valueHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")

	if metricType == "gauge" {
		if val, exists := gauges[name]; exists {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(strconv.FormatFloat(val, 'f', -1, 64)))
			return
		}
	} else if metricType == "counter" {
		if val, exists := counters[name]; exists {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(strconv.FormatInt(val, 10)))
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
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
	flag.StringVar(&serverAddress, "a", "localhost:8080", "HTTP server endpoint address")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	flag.Parse()

	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: unknown arguments: %v\n", flag.Args())
		flag.Usage()
		os.Exit(1)
	}

	r := chi.NewRouter()

	r.Post("/update/{type}/{name}/{value}", updateHandler)
	r.Get("/value/{type}/{name}", valueHandler)
	r.Get("/", mainHandler)

	fmt.Printf("Server starting on http://%s\n", serverAddress)

	err := http.ListenAndServe(serverAddress, r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Server startup error: %v\n", err)
		os.Exit(1)
	}
}
