package runner

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio"
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
	prepared.Adapters = executionAdapters(adapters)
	if err := profile.ValidatePreparedTarget(preparedSpec, prepared, environment); err != nil {
		return target.Prepared{}, err
	}
	return prepared, nil
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
		prepared, err := legacyPreparationForEquivalence(t.Context(), target.Spec{
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
