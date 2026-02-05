package main

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdateHandler(t *testing.T) {
	req := httptest.NewRequest("POST", "/update/gauge/test/10.5", nil)
	rr := httptest.NewRecorder()

	assert.NotNil(t, req)
	assert.NotNil(t, rr)
}

func TestValueHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/value/gauge/temp", nil)
	rr := httptest.NewRecorder()

	assert.NotNil(t, req)
	assert.NotNil(t, rr)
}
