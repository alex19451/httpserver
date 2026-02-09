package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSendMetric(t *testing.T) {
	req, err := http.NewRequest("POST", "http://localhost:8080/update/gauge/test/10", nil)
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, "http://localhost:8080/update/gauge/test/10", req.URL.String())
}

func TestSendCounter(t *testing.T) {
	req, err := http.NewRequest("POST", "http://localhost:8080/update/counter/visits/1", nil)
	assert.NoError(t, err)
	assert.Equal(t, "POST", req.Method)
}

func TestURLFormat(t *testing.T) {
	url := "http://localhost:8080/update/counter/PollCount/5"
	req, err := http.NewRequest("POST", url, nil)
	assert.NoError(t, err)
	assert.Equal(t, url, req.URL.String())
}
