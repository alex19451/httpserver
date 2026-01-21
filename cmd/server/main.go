package main

import (
	"net/http"
	"strconv"
	"strings"
)

var gauges = make(map[string]float64)
var counters = make(map[string]int64)

func updateHandler(w http.ResponseWriter, r *http.Request) {
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

func main() {
	http.HandleFunc("/update/", updateHandler)

	http.ListenAndServe(":8080", nil)
}
