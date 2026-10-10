package wasi

import (
	"testing"
	"testing/fstest"
)

func TestImplementationIdentityPreservesCompiledModelDigest(t *testing.T) {
	const expected = "sha256:c218a9071d4ee6ea2771b9126b31e770684e2dfabf7091b73c266a12b0a8bf8b"
	if got := ImplementationSHA256(); got != expected {
		t.Fatalf("compiled model identity = %q, want %q", got, expected)
	}
}

func TestImplementationIdentityFailsClosedOnMissingSource(t *testing.T) {
	if got := implementationSHA256(fstest.MapFS{}); got != "" {
		t.Fatalf("missing implementation source returned valid identity %q", got)
	}
}
