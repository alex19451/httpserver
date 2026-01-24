package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAgent(t *testing.T) {
	assert.True(t, true, "true should be true")
	assert.False(t, false, "false should be false")
}

func TestReportInterval(t *testing.T) {
	assert.Equal(t, 0, 5%5, "Should send when count=5")
	assert.Equal(t, 0, 10%5, "Should send when count=10")
	assert.NotEqual(t, 0, 6%5, "Should not send when count=6")
}
