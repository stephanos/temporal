package architecture

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoveryRejectsOmittedSource(t *testing.T) {
	root := t.TempDir()
	for name, contents := range map[string]string{
		"go.mod":            "module example.invalid/architecture\n\ngo 1.27.1\n",
		".hidden/source.go": "package hidden\n",
	} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	_, findings, err := Discover(root, "go", Platform{"linux", "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		if finding.Category == "uncovered-source" {
			return
		}
	}
	t.Fatalf("hidden production source escaped inventory: %v", findings)
}

func TestRequiredModuleNeedsItsOwnSource(t *testing.T) {
	root := t.TempDir()
	for name, source := range map[string]string{"go.mod": "module example.invalid/architecture\n\ngo 1.27.1\n", "deterministicio/testdata/cactusstatsd/go.mod": "module example.invalid/fixture\n\ngo 1.27.1\n", "deterministicio/testdata/other/source.go": "package other\n"} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	_, findings, err := Discover(root, "go", Platform{"linux", "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		if finding.Category == "stale-exclusion" && finding.Path == "deterministicio/testdata/cactusstatsd/go.mod" {
			return
		}
	}
	t.Fatalf("classified nested module without source escaped: %v", findings)
}

func TestDiscoveryRejectsDirectorySymlink(t *testing.T) {
	root := t.TempDir()
	linked := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.invalid/architecture\n\ngo 1.27.1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(linked, "source.go"), []byte("package hidden\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(linked, filepath.Join(root, "hidden")); err != nil {
		t.Fatal(err)
	}
	_, findings, err := Discover(root, "go", Platform{"linux", "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		if finding.Category == "uncovered-source" && finding.Path == "hidden" {
			return
		}
	}
	t.Fatalf("linked source directory escaped: %v", findings)
}
