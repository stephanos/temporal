package io_failure

import (
	"os"
	"testing"
)

func TestDeterministicIOFailure(t *testing.T) {
	if err := os.MkdirAll("../escape", 0o755); err != nil {
		t.Fatalf("deterministic I/O fixture: %v", err)
	}
	t.Fatal("Gomad accepted a path outside the in-memory root")
}
