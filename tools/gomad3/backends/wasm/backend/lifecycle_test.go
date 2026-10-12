package backend

import (
	"context"
	"encoding/json"
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

func TestProviderLifecycleRetainsExecutionObservers(t *testing.T) {
	provider, err := New(portableProviderOptions(wasi.StockProfile))
	if err != nil {
		t.Fatal(err)
	}
	counted := &countedProvider{Provider: provider}
	if _, err := counted.Execute(t.Context(), backends.ExecutionRequest{}); err == nil {
		t.Fatal("missing prepared target was accepted")
	}
	if counted.calls != 1 {
		t.Fatalf("execution observer calls = %d, want 1", counted.calls)
	}
}

func TestProviderLifecycleEmptyCoveragePreservesCanonicalBytes(t *testing.T) {
	provider, err := New(portableProviderOptions(wasi.StockProfile))
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := provider.Evidence(t.Context(), backends.EvidenceRequest{})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(evidence.Coverage)
	if err != nil {
		t.Fatal(err)
	}
	const expected = `{"schema":"gomad3.semantic-coverage/v1","digest":"sha256:b14c6b04e0e4bd4b4431562e77b1b78eb7940a9cc505e47a8df9f713f4058356","probes":[]}`
	if string(data) != expected {
		t.Fatalf("empty coverage = %s, want %s", data, expected)
	}
}

func executeObservedTestProvider(ctx context.Context, request backends.ExecutionRequest, run func(context.Context, backends.Request) (backends.Result, error)) (backends.ExecutionResult, error) {
	converted, err := observedRequest(request)
	if err != nil {
		return backends.ExecutionResult{}, err
	}
	observed, err := run(ctx, converted)
	if err != nil {
		return backends.ExecutionResult{}, err
	}
	return backends.ProjectObservedResult(converted, observed, request.StdoutHead, request.StderrHead)
}

func (p *countedProvider) Execute(ctx context.Context, request backends.ExecutionRequest) (backends.ExecutionResult, error) {
	return executeObservedTestProvider(ctx, request, p.Run)
}

func (p *minimizationProvider) Execute(ctx context.Context, request backends.ExecutionRequest) (backends.ExecutionResult, error) {
	return executeObservedTestProvider(ctx, request, p.Run)
}

func (p *interruptRoundProvider) Execute(ctx context.Context, request backends.ExecutionRequest) (backends.ExecutionResult, error) {
	return executeObservedTestProvider(ctx, request, p.Run)
}

func (p *resumeBoundaryProvider) Execute(ctx context.Context, request backends.ExecutionRequest) (backends.ExecutionResult, error) {
	return executeObservedTestProvider(ctx, request, p.Run)
}
