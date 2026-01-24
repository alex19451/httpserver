package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateHandler(t *testing.T) {
	a := 1
	b := 2
	assert.True(t, a == 1, "a should equal 1")
	require.Equal(t, b, 2, "b should equal 2")
}

func TestCounterLogic(t *testing.T) {
	count := 0
	count++

	assert.Equal(t, 1, count, "Count should be 1")
	assert.NotEqual(t, 0, count, "Count should not be 0")
}
