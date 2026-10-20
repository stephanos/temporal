package sweep

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSweepPreservesPackageCompilationBetweenUnits(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	marker := filepath.Join(dirsFor(f.options.Work, "./tests/alpha").preparation, "compiled-module")
	f.outcomes["TestAlpha"] = func(attempt int, artifacts string) (Completed, map[string]ArtifactFacts) {
		if err := os.WriteFile(marker, []byte("compiled once"), 0o600); err != nil {
			t.Fatal(err)
		}
		return passing("TestAlpha")(attempt, artifacts)
	}
	f.outcomes["TestBeta"] = func(attempt int, artifacts string) (Completed, map[string]ArtifactFacts) {
		if data, err := os.ReadFile(marker); err != nil || string(data) != "compiled once" {
			t.Fatalf("package compilation discarded between units: %q %v", data, err)
		}
		return passing("TestBeta")(attempt, artifacts)
	}
	if findings, err := f.sweeper().Run(t.Context()); err != nil || len(findings) != 0 {
		t.Fatalf("sweep: %v %v", findings, err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("package compilation not cleaned after package: %v", err)
	}
}
