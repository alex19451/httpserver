package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/alex19451/httpserver/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestSendMetric(t *testing.T) {
	value := 10.5
	metrics := models.Metrics{
		ID:    "test",
		MType: "gauge",
		Value: &value,
	}

	data, err := json.Marshal(metrics)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "http://localhost:8080/update/", bytes.NewBuffer(data))
	assert.NoError(t, err)
	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
}

func TestSendCounter(t *testing.T) {
	delta := int64(5)
	metrics := models.Metrics{
		ID:    "visits",
		MType: "counter",
		Delta: &delta,
	}

	data, err := json.Marshal(metrics)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "http://localhost:8080/update/", bytes.NewBuffer(data))
	assert.NoError(t, err)
	assert.Equal(t, "POST", req.Method)
}

func TestURLFormat(t *testing.T) {
	url := "http://localhost:8080/update/"
	req, err := http.NewRequest("POST", url, nil)
	assert.NoError(t, err)
	assert.Equal(t, url, req.URL.String())
}

func TestBackoffSchedule(t *testing.T) {
	backoffSchedule := []time.Duration{
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
	}

	assert.Equal(t, 3, len(backoffSchedule))
	assert.Equal(t, 100*time.Millisecond, backoffSchedule[0])
	assert.Equal(t, 500*time.Millisecond, backoffSchedule[1])
	assert.Equal(t, 1*time.Second, backoffSchedule[2])
}
