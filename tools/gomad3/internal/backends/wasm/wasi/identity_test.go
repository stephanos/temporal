package wasi

import (
	"testing"
	"testing/fstest"
)

func TestImplementationIdentityBindsCompiledModelDigest(t *testing.T) {
	const expected = "sha256:5ae9c8f10115a4f3dba31b1f8be30e2568161029d65e727d5ca9ba86d5dff425"
	if got := ImplementationSHA256(); got != expected {
		t.Fatalf("compiled model identity = %q, want %q", got, expected)
	}
}

func TestImplementationIdentityFailsClosedOnMissingSource(t *testing.T) {
	if got := implementationSHA256(fstest.MapFS{}); got != "" {
		t.Fatalf("missing implementation source returned valid identity %q", got)
	}
}

func TestImplementationIdentityBindsStrictJSONOwner(t *testing.T) {
	baseline := implementationSHA256WithStrictJSON(implementationSources, "sha256:strict-json")
	changed := implementationSHA256WithStrictJSON(implementationSources, "sha256:changed")
	if baseline == "" || changed == baseline {
		t.Fatalf("strict JSON owner did not rotate model identity: %q / %q", baseline, changed)
	}
	if got := implementationSHA256WithStrictJSON(implementationSources, ""); got != "" {
		t.Fatalf("missing strict JSON identity returned valid model identity %q", got)
	}
}
