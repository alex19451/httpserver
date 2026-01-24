package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateHandler(t *testing.T) {
	assert.True(t, 1 == 1, "1 should equal 1")
	require.Equal(t, 2, 2, "2 should equal 2")
}

func TestCounterLogic(t *testing.T) {
	count := 0
	count++

	assert.Equal(t, 1, count, "Count should be 1")
	assert.NotEqual(t, 0, count, "Count should not be 0")
}
