package backend

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestProviderLifecycleRejectsNativeRequestsWithoutLaunching(t *testing.T) {
	provider, err := New(portableProviderOptions(wasi.StockProfile))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Describe(t.Context(), backends.Configuration{Target: target.Spec{Backend: "native"}}); err == nil {
		t.Fatal("WASM provider admitted native target preparation")
	}
	if _, err := provider.Evidence(t.Context(), backends.EvidenceRequest{Transcript: []byte("native")}); err == nil {
		t.Fatal("WASM provider admitted native transcript evidence")
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reader.Close() })
	_, err = provider.Execute(t.Context(), backends.ExecutionRequest{Coordinator: &backends.CoordinatorRequest{Stdout: writer}})
	if err == nil || !strings.Contains(err.Error(), "coordinator") {
		t.Fatalf("native coordinator request error = %v", err)
	}
	if _, err := writer.Stat(); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("transferred coordinator writer remained open: %v", err)
	}
}

func TestProviderLifecyclePreservesCancellation(t *testing.T) {
	provider, err := New(portableProviderOptions(wasi.StockProfile))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := provider.Describe(ctx, backends.Configuration{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Describe cancellation = %v", err)
	}
	if _, err := provider.Evidence(ctx, backends.EvidenceRequest{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Evidence cancellation = %v", err)
	}
}
