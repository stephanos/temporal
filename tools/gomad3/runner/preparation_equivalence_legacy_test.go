package runner

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/internal/preparation"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
)

type preparationEquivalenceSnapshot struct {
	Fresh           target.Prepared `json:"fresh"`
	Cache           target.Prepared `json:"cache"`
	FreshProjection record.Target   `json:"fresh_projection"`
	CacheProjection record.Target   `json:"cache_projection"`
}

func legacyPreparationForEquivalence(ctx context.Context, spec target.Spec, environment []string) (target.Prepared, error) {
	profile := deterministicio.Default()
	preparedSpec, adapters, err := profile.PrepareTargetBuildAdapters(ctx, spec)
	if err != nil {
		return target.Prepared{}, err
	}
	prepared, err := target.Prepare(ctx, preparedSpec)
	if err != nil {
		return target.Prepared{}, err
	}
	prepared.Adapters = make([]record.TargetAdapter, len(adapters))
	for index, adapter := range adapters {
		prepared.Adapters[index] = record.TargetAdapter{Module: adapter.Module, Version: adapter.Version, Sum: adapter.Sum}
	}
	if err := profile.ValidatePreparedTarget(preparedSpec, prepared, environment); err != nil {
		return target.Prepared{}, err
	}
	return prepared, nil
}

func preparationForEquivalence(ctx context.Context, spec target.Spec, environment []string) (target.Prepared, error) {
	if os.Getenv("GOMAD3_PREPARATION_USE_OWNER") == "1" {
		return preparation.Prepare(ctx, preparation.Request{Target: spec, Environment: environment})
	}
	return legacyPreparationForEquivalence(ctx, spec, environment)
}

func TestLegacyPreparationFixedInputSnapshot(t *testing.T) {
	path := os.Getenv("GOMAD3_PREPARATION_SNAPSHOT")
	if path == "" {
		t.Skip("fixed-input snapshot requested separately")
	}
	fixture := os.Getenv("GOMAD3_PREPARATION_FIXTURE")
	if fixture == "" {
		t.Fatal("fixed-input fixture is required")
	}
	result := preparationEquivalenceSnapshot{}
	for index, name := range []string{"fresh", "cache"} {
		if err := os.MkdirAll(filepath.Join(filepath.Dir(path), name), 0o700); err != nil {
			t.Fatal(err)
		}
		prepared, err := preparationForEquivalence(t.Context(), target.Spec{
			Kind: target.KindGoRun, Source: ".", WorkingDir: fixture,
			PreparationRoot: filepath.Join(filepath.Dir(path), name), ToolchainRoot: toolchainRoot(t),
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		prepared.Path = "<prepared-path>"
		if index == 0 {
			result.Fresh = prepared
			result.FreshProjection = prepared.RecordTarget()
		} else {
			result.Cache = prepared
			result.CacheProjection = prepared.RecordTarget()
		}
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyAdapterPreparationFixedInputSnapshot(t *testing.T) {
	path := os.Getenv("GOMAD3_PREPARATION_ADAPTER_SNAPSHOT")
	if path == "" {
		t.Skip("fixed-input adapter snapshot requested separately")
	}
	fixture, err := filepath.Abs(filepath.Join("..", "deterministicio", "testdata", "sprig"))
	if err != nil {
		t.Fatal(err)
	}
	result := preparationEquivalenceSnapshot{}
	for index, name := range []string{"fresh", "cache"} {
		if err := os.MkdirAll(filepath.Join(filepath.Dir(path), name), 0o700); err != nil {
			t.Fatal(err)
		}
		prepared, err := preparationForEquivalence(t.Context(), target.Spec{
			Kind: target.KindGoTest, Source: ".", WorkingDir: fixture,
			PreparationRoot: filepath.Join(filepath.Dir(path), name), ToolchainRoot: toolchainRoot(t),
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		prepared.Path = "<prepared-path>"
		if index == 0 {
			result.Fresh = prepared
			result.FreshProjection = prepared.RecordTarget()
		} else {
			result.Cache = prepared
			result.CacheProjection = prepared.RecordTarget()
		}
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}
