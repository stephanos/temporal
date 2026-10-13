package backend

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/backends"
	"go.temporal.io/server/tools/gomad3/internal/backends/target"
	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/wasi"
	captured "go.temporal.io/server/tools/gomad3/internal/evidence/artifact/capture"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
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

func TestProviderLifecycleRejectsCommandAuthorityMismatch(t *testing.T) {
	for _, request := range []backends.ExecutionRequest{
		{Command: "arbitrary-host-command"},
		{Args: []string{"without-prepared-argv"}},
		{Argv0: "without-prepared-argv"},
		{Target: target.Prepared{Path: "sealed-module", Argv: []string{"guest", "expected"}}, Args: []string{"changed"}},
	} {
		if _, err := observedRequest(request); err == nil || !strings.Contains(err.Error(), "prepared target") {
			t.Fatalf("request %#v error = %v", request, err)
		}
	}
}

func TestProviderLifecycleDoesNotCloseBorrowedJournal(t *testing.T) {
	provider, err := New(portableProviderOptions(wasi.StockProfile))
	if err != nil {
		t.Fatal(err)
	}
	journal, err := os.Create(filepath.Join(t.TempDir(), "stdout"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = journal.Close() })
	if _, err := provider.Execute(t.Context(), backends.ExecutionRequest{StdoutHead: journal}); err == nil {
		t.Fatal("missing prepared target was accepted")
	}
	if _, err := journal.Stat(); err != nil {
		t.Fatalf("provider closed borrowed journal: %v", err)
	}
}

func TestProviderLifecycleDistinguishesEmptyAndNativeIORequests(t *testing.T) {
	for _, test := range []struct {
		name        string
		mount       *backends.ReadOnlyMountCapability
		unsupported bool
	}{
		{"absent", nil, false},
		{"campaign-default", &backends.ReadOnlyMountCapability{Limits: captured.DefaultLimits()}, false},
		{"replay-default", &backends.ReadOnlyMountCapability{Limits: captured.DefaultLimits(), Replay: &captured.Snapshot{}}, false},
		{"mapping", &backends.ReadOnlyMountCapability{Mappings: []captured.Mapping{{Source: "/host", Target: "/guest"}}}, true},
		{"captured-entry", &backends.ReadOnlyMountCapability{Replay: &captured.Snapshot{Entries: []captured.Entry{{Path: "/guest"}}}}, true},
		{"negative-lookup", &backends.ReadOnlyMountCapability{Replay: &captured.Snapshot{NotExist: []string{"/guest"}}}, true},
		{"captured-request", &backends.ReadOnlyMountCapability{Replay: &captured.Snapshot{Requests: 1}}, true},
		{"captured-bytes", &backends.ReadOnlyMountCapability{Replay: &captured.Snapshot{TotalBytes: 1}}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := backends.ExecutionRequest{Seed: 7, OutputLimit: 1024, Diagnostics: true, IO: &backends.IOCapability{Profile: record.IOProfile{Name: Name}, Transcript: &backends.IOTranscriptCapability{Limit: 4096}, ReadOnlyMount: test.mount}}
			observed, err := observedRequest(request)
			if test.unsupported {
				if err == nil || !strings.Contains(err.Error(), "native I/O") {
					t.Fatalf("unsupported request error = %v", err)
				}
				return
			}
			if err != nil || observed.Seed != request.Seed || observed.OutputBytes != request.OutputLimit || observed.TranscriptBytes != 4096 || !observed.Diagnostics {
				t.Fatalf("inert native fields changed observed request: %#v, %v", observed, err)
			}
		})
	}
	for _, transcript := range []*backends.IOTranscriptCapability{{Replay: true}, {Expected: []byte{1}}} {
		if _, err := observedRequest(backends.ExecutionRequest{IO: &backends.IOCapability{Transcript: transcript}}); err == nil {
			t.Fatal("native transcript replay admitted")
		}
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
