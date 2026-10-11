package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompressionStartupInventoryAcceptsPinnedSources(t *testing.T) {
	for _, path := range []string{"compress/gzip", "hash/crc32"} {
		t.Run(path, func(t *testing.T) {
			a := reflectSummaryAnalysis(t, "package record\nfunc Root(){}\n")
			dir := filepath.Join(filepath.Dir(a.program.Metadata["reflect"].Dir), path)
			a.checkStartupSource(Package{ImportPath: path, Standard: true, Dir: dir}, 0)
			if len(a.findings) != 0 {
				t.Fatalf("reviewed standard startup rejected: %v", a.findings)
			}
		})
	}
}

func TestCompressionStartupInventoryRejectsChangedSources(t *testing.T) {
	for _, path := range []string{"compress/gzip", "hash/crc32"} {
		t.Run(path, func(t *testing.T) {
			a := effectResolverAnalysis(effectResolverProgram(t, "package record\nfunc Root(){}\n"))
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "changed.go"), []byte("package changed\n"), 0600); err != nil {
				t.Fatal(err)
			}
			a.checkStartupSource(Package{ImportPath: path, Standard: true, Dir: dir}, 0)
			if len(a.findings) != 1 {
				t.Fatalf("changed source accepted: %v", a.findings)
			}
			for _, finding := range a.findings {
				if finding.Category != "unresolved-effect" || !strings.Contains(finding.Detail, "pinned standard startup source changed: "+path) {
					t.Fatalf("wrong source rejection: %v", finding)
				}
			}
		})
	}
}

func TestCompressionStartupInventoryMissingEntryFailsClosed(t *testing.T) {
	a := effectResolverAnalysis(effectResolverProgram(t, "package record\nfunc Root(){}\n"))
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "unreviewed.go"), []byte("package unreviewed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a.checkStartupSource(Package{ImportPath: "example.invalid/unreviewed", Standard: true, Dir: dir}, 0)
	if len(a.findings) != 1 {
		t.Fatalf("unreviewed startup accepted: %v", a.findings)
	}
	for _, finding := range a.findings {
		if finding.Category != "unresolved-effect" || !strings.Contains(finding.Detail, "standard startup source has no pinned inventory: example.invalid/unreviewed") {
			t.Fatalf("missing inventory misclassified as changed source: %v", finding)
		}
	}
}

func TestCompressionStartupInventoryRejectsNonstandardIdentity(t *testing.T) {
	for _, path := range []string{"compress/gzip", "hash/crc32"} {
		t.Run(path, func(t *testing.T) {
			a := reflectSummaryAnalysis(t, "package record\nfunc Root(){}\n")
			dir := filepath.Join(filepath.Dir(a.program.Metadata["reflect"].Dir), path)
			a.checkStartupSource(Package{ImportPath: path, Dir: dir}, 0)
			if len(a.findings) != 1 {
				t.Fatalf("nonstandard identity accepted: %v", a.findings)
			}
			for _, finding := range a.findings {
				if finding.Category != "unresolved-effect" || !strings.Contains(finding.Detail, "standard startup requires standard-library identity: "+path) {
					t.Fatalf("wrong identity rejection: %v", finding)
				}
			}
		})
	}
}

func TestCompressionStartupInventoryDoesNotHideDependencyInitializer(t *testing.T) {
	for _, path := range []string{"compress/gzip", "hash/crc32"} {
		t.Run(path, func(t *testing.T) {
			program := effectResolverProgram(t, "package record\nfunc Root(){}\nfunc init(){go sideEffect()}\nfunc sideEffect(){}\n")
			const original = "example.invalid/resolver/record"
			pkg := program.Packages[original]
			metadata := pkg.Metadata
			metadata.ImportPath = path
			pkg.Metadata = metadata
			delete(program.Packages, original)
			delete(program.Metadata, original)
			program.Packages[path], program.Metadata[path] = pkg, metadata
			a := effectResolverAnalysis(program)
			a.initializers()
			if len(a.findings) != 1 {
				t.Fatalf("dependency initializer was not traced: %v", a.findings)
			}
			for _, finding := range a.findings {
				if finding.Category != "host-effect" || !strings.Contains(finding.Detail, "goroutine start") {
					t.Fatalf("dependency initializer was hidden by standard package spelling: %v", finding)
				}
			}
		})
	}
}
