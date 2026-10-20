//go:build gomad_fixture

package io_failure

import (
	"os"
	"testing"
)

// TestDeterministicIOFailure performs modeled I/O and then fails so the
// runner records a non-zero exit with a complete I/O transcript.
func TestDeterministicIOFailure(t *testing.T) {
	if err := os.WriteFile("state.txt", []byte("pending"), 0o644); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile("state.txt")
	if err != nil {
		t.Fatal(err)
	}
	t.Fatalf("deterministic failure after reading %q", contents)
}
