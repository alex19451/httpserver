package main

import (
	"testing"
)

func TestServer(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Run("simple", func(t *testing.T) {
		if 2+2 != 4 {
			t.Fail()
		}
	})
}
