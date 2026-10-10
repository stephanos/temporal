package backend

import (
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad3/target"
)

func TestTestSourceChangesInvalidateCompilationCache(t *testing.T) {
	provider := integrationProvider(t)
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "go.mod"), []byte("module cache.fixture\n\ngo 1.27.1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(source, "fixture_test.go")
	firstSource := []byte("package fixture\nimport \"testing\"\nfunc TestCache(t *testing.T){t.Log(\"first\")}\n")
	if err := os.WriteFile(file, firstSource, 0600); err != nil {
		t.Fatal(err)
	}
	spec := target.Spec{Backend: Name, Kind: target.KindGoTest, Source: ".", WorkingDir: source, PreparationRoot: filepath.Join(t.TempDir(), "first"), BuildTags: []string{"test_dep"}}
	first, err := provider.Prepare(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("package fixture\nimport \"testing\"\nfunc TestCache(t *testing.T){t.Log(\"second\")}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	spec.PreparationRoot = filepath.Join(t.TempDir(), "second")
	second, err := provider.Prepare(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if first.BuildKey == second.BuildKey || first.SHA256 == second.SHA256 {
		t.Fatal("changed selected test source reused stale compilation cache")
	}
	spec.PreparationRoot = filepath.Join(t.TempDir(), "third")
	third, err := provider.Prepare(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if third.BuildKey != second.BuildKey || third.SHA256 != second.SHA256 {
		t.Fatal("unchanged source did not reuse verified cache identity")
	}
	module := filepath.Join(provider.options.CacheRoot, third.BuildKey, "module.wasm")
	if err := os.Chmod(module, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(module, []byte("corrupt module"), 0600); err != nil {
		t.Fatal(err)
	}
	spec.PreparationRoot = filepath.Join(t.TempDir(), "corrupt")
	if _, err := provider.Prepare(t.Context(), spec); err == nil {
		t.Fatal("corrupt module compilation cache accepted")
	}
}
