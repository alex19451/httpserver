package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAgent(t *testing.T) {
	assert.True(t, true)
	assert.False(t, false)
}

func TestReportInterval(t *testing.T) {
	a := 5
	b := 10
	c := 6

	assert.Equal(t, 0, a%5)
	assert.Equal(t, 0, b%5)
	assert.Equal(t, 1, c%5)
}
