package wasi

import (
	"testing"
	"testing/fstest"
)

func TestImplementationIdentityBindsCompiledModelDigest(t *testing.T) {
	const expected = "sha256:04d18f1c559ba4dd7896e343332e0181d93657dc979e2f5bd43f485ec6fe2441"
	if got := ImplementationSHA256(); got != expected {
		t.Fatalf("compiled model identity = %q, want %q", got, expected)
	}
}

func TestImplementationIdentityFailsClosedOnMissingSource(t *testing.T) {
	if got := implementationSHA256(fstest.MapFS{}); got != "" {
		t.Fatalf("missing implementation source returned valid identity %q", got)
	}
}
