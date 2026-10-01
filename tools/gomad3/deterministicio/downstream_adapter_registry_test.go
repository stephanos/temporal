package deterministicio

import (
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad3/target"
)

func TestProfileSelectsDownstreamAdapters(t *testing.T) {
	toolchainRoot, err := filepath.Abs(filepath.Join("..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	for _, consumer := range []struct{ fixture, module string }{
		{fixture: "sprig", module: sprigModulePath},
		{fixture: "validator", module: validatorModulePath},
		{fixture: "pebble", module: pebbleModulePath},
		{fixture: "cactusstatsd", module: cactusStatsDModulePath},
		{fixture: "memberlist", module: memberlistModulePath},
	} {
		t.Run(consumer.fixture, func(t *testing.T) {
			workingDirectory := t.TempDir()
			fixture := filepath.Join("testdata", consumer.fixture)
			entries, err := os.ReadDir(fixture)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if entry.IsDir() {
					t.Fatalf("unexpected directory in consumer fixture: %s", entry.Name())
				}
				contents, err := os.ReadFile(filepath.Join(fixture, entry.Name()))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(workingDirectory, entry.Name()), contents, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			spec, adapters, err := Default().PrepareTargetBuildAdapters(t.Context(), target.Spec{
				Kind: target.KindGoTest, Source: ".", WorkingDir: workingDirectory,
				PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot,
				BuildTags: []string{"gomad", "hashicorpmetrics", "integration", "test_dep"},
			})
			if err != nil {
				t.Fatal(err)
			}
			selected := false
			for _, adapter := range adapters {
				selected = selected || adapter.Module == consumer.module
			}
			if !selected {
				t.Fatalf("consumer adapter %s was not selected: %#v", consumer.module, adapters)
			}
			if _, err := target.ReviewCapabilities(t.Context(), spec); err != nil {
				t.Fatal(err)
			}
		})
	}
}
