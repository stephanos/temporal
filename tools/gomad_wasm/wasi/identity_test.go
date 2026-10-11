package wasi

import (
	"testing"
	"testing/fstest"
)

func TestImplementationIdentityPreservesCompiledModelDigest(t *testing.T) {
	const expected = "sha256:26d1d15593e3c28e5ff6e74154d100439b9c6c497d6677d9570961cca2f7978a"
	if got := ImplementationSHA256(); got != expected {
		t.Fatalf("compiled model identity = %q, want %q", got, expected)
	}
}

func TestImplementationIdentityFailsClosedOnMissingSource(t *testing.T) {
	if got := implementationSHA256(fstest.MapFS{}); got != "" {
		t.Fatalf("missing implementation source returned valid identity %q", got)
	}
}
