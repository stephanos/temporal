package measure

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/measurement"
)

func TestFixtureStagingCreatesMissingDestinationParents(t *testing.T) {
	t.Parallel()
	source, destination := t.TempDir(), t.TempDir()
	if err := os.Mkdir(filepath.Join(source, "owner"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "owner", "fixture.go"), []byte("package fixture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"direct", "missing/nested/fixture"} {
		if err := temporalStageFixture(source, "owner", destination, target); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(destination, target, "fixture.go"))
		if err != nil || string(data) != "package fixture\n" {
			t.Fatalf("staged data %q, error %v", data, err)
		}
		if err := temporalStageFixture(source, "owner", destination, target); err != nil {
			t.Fatalf("existing destination: %v", err)
		}
	}
}

func TestProducersMatchCommand(t *testing.T) {
	t.Parallel()
	names := make([]string, 0, len(producers))
	for name := range producers {
		names = append(names, name)
	}
	slices.Sort(names)
	if !slices.Equal(names, measurement.Names()) {
		t.Fatalf("producers %v, command %v", names, measurement.Names())
	}
}

func TestRunRefusesImplicitInputsBeforeCollection(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		command string
		options Options
	}{
		{name: "unknown", command: "unknown"},
		{name: "relative-repository", command: "namespace-sample", options: Options{Repo: "relative"}},
		{name: "missing-identities", command: "namespace-sample", options: Options{Repo: t.TempDir()}},
		{name: "inherited-input", command: "namespace-sample", options: Options{Repo: t.TempDir(), Environ: []string{"GOMAD_WASM_NAMESPACE_SAMPLE_ROOT=/evidence"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := Run(t.Context(), test.command, test.options); !errors.Is(err, ErrRefused) {
				t.Fatalf("expected admission refusal, got %v", err)
			}
		})
	}
}

func TestRunCancellationBeforeCollection(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := Run(ctx, "namespace-sample", Options{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func TestCollectionRetainsReportsAndCleanupFailures(t *testing.T) {
	t.Parallel()
	collectionError, cleanupError := errors.New("collection failure"), errors.New("report failure")
	scope := &runScope{name: "collection", options: Options{Log: io.Discard}}
	ctx := context.WithValue(t.Context(), runKey{}, scope)
	var reports []string
	err := scope.execute(ctx, func(ctx context.Context) error {
		addCleanup(ctx, func(ctx context.Context) error {
			if !failed(ctx) {
				t.Error("partial collection failure was absent when writing the final report")
			}
			reports = append(reports, "final")
			return cleanupError
		})
		passed, err := runStep(ctx, "failed-sample", func(context.Context) error { return collectionError })
		if passed || err != nil {
			t.Fatalf("failed sample should permit retained collection: passed=%t err=%v", passed, err)
		}
		reports = append(reports, "next-sample")
		return nil
	})
	if !errors.Is(err, collectionError) || !errors.Is(err, cleanupError) {
		t.Fatalf("collection or final report failure lost: %v", err)
	}
	if !slices.Equal(reports, []string{"next-sample", "final"}) {
		t.Fatalf("report sequence %v", reports)
	}
}
