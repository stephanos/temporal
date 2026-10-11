package backend

import (
	"bytes"
	"testing"

	"go.temporal.io/server/tools/gomad_wasm/wasi"
)

func TestModelEvidencePreservesRawDefaultAndBindsEffects(t *testing.T) {
	plain := wasi.Result{Transcript: []byte("wasi")}
	encoded, err := modelEvidence(plain, 1024)
	if err != nil || !bytes.Equal(encoded, plain.Transcript) {
		t.Fatalf("default bytes changed: %q %v", encoded, err)
	}
	modeled := plain
	modeled.StorageTranscript = []byte("committed effects")
	first, err := modelEvidence(modeled, 1024)
	if err != nil {
		t.Fatal(err)
	}
	modeled.StorageTranscript = []byte("lost effects")
	second, err := modelEvidence(modeled, 1024)
	if err != nil || bytes.Equal(first, second) {
		t.Fatalf("storage effects absent from generic backend identity: %q %v", second, err)
	}
	modeled.NetworkTranscript = []byte("network effects")
	third, err := modelEvidence(modeled, 1024)
	if err != nil || bytes.Equal(second, third) {
		t.Fatalf("network effects absent from identity: %q %v", third, err)
	}
	if _, err := modelEvidence(modeled, 2); err == nil {
		t.Fatal("oversized modeled evidence accepted")
	}
}
