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
	assert.Equal(t, 0, 5%5)
	assert.Equal(t, 0, 10%5)
	assert.Equal(t, 1, 6%5)
}
