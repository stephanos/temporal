package wasi

import (
	"testing"
	"testing/fstest"
)

func TestImplementationIdentityBindsCompiledModelDigest(t *testing.T) {
	const expected = "sha256:4f0d27b92506aa39ef168279ddce40f58dd700edf575dc2655f0fb3c7429dcc1"
	if got := ImplementationSHA256(); got != expected {
		t.Fatalf("compiled model identity = %q, want %q", got, expected)
	}
}

func TestImplementationIdentityFailsClosedOnMissingSource(t *testing.T) {
	if got := implementationSHA256(fstest.MapFS{}); got != "" {
		t.Fatalf("missing implementation source returned valid identity %q", got)
	}
}
