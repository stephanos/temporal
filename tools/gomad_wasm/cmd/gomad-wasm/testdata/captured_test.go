package io_failure

import (
	"errors"
	"os"
	"testing"
)

var executions int

func TestCapturedInputFailure(t *testing.T) {
	executions++
	if executions != 1 {
		t.Fatalf("guest heap leaked between executions: %d", executions)
	}
	if _, err := os.Stat("/tmp/operator-fresh-state"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("modeled file state leaked between executions: %v", err)
	}
	if err := os.WriteFile("/tmp/operator-fresh-state", []byte("guest-local state"), 0600); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("/inputs/control")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "immutable operator input" {
		t.Fatalf("captured input changed: %q", data)
	}
	t.Fatalf("intentional failure after captured input %q", data)
}
