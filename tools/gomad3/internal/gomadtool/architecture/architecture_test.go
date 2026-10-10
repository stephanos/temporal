package architecture

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestDiscoveryInventoriesRuntimeHostPackages(t *testing.T) {
	root := writeArchitectureSources(t, map[string]string{
		"toolchain/runtime/testdata/vfdpointer/source.go":      "package runner\n",
		"toolchain/runtime/testdata/vfdpointer/source_test.go": "package runner\nimport \"testing\"\nfunc TestRunner(t *testing.T) {}\n",
		"toolchain/runtime/testdata/vfdnative/source.go":       "package runner\n",
		"toolchain/runtime/testdata/vfdnative/source_test.go":  "package runner\nimport \"testing\"\nfunc TestRunner(t *testing.T) {}\n",
	})
	inventory, findings, err := Discover(root, "go", Platform{"linux", "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"toolchain/runtime/testdata/vfdnative/source.go",
		"toolchain/runtime/testdata/vfdnative/source_test.go",
		"toolchain/runtime/testdata/vfdpointer/source.go",
		"toolchain/runtime/testdata/vfdpointer/source_test.go",
	}
	if !slices.Equal(inventory.Sources, want) {
		t.Fatalf("source inventory = %v, want %v", inventory.Sources, want)
	}
	for _, finding := range findings {
		if finding.Category != "stale-exclusion" {
			t.Fatalf("host packages escaped discovery: %+v", finding)
		}
	}
	for _, dependencies := range []bool{false, true} {
		packages, err := List(root, "go", Platform{"linux", "amd64"}, dependencies)
		if err != nil {
			t.Fatal(err)
		}
		for _, directory := range []string{"toolchain/runtime/testdata/vfdpointer", "toolchain/runtime/testdata/vfdnative"} {
			index := slices.IndexFunc(packages, func(pkg Package) bool { return pkg.Dir == filepath.Join(root, directory) })
			if index == -1 {
				t.Fatalf("List(dependencies=%t) omitted %s", dependencies, directory)
			}
			pkg := packages[index]
			if Owner("example.invalid/architecture", pkg.ImportPath) != "toolchain" || !slices.Equal(pkg.GoFiles, []string{"source.go"}) || !slices.Equal(pkg.TestGoFiles, []string{"source_test.go"}) {
				t.Fatalf("host package lost ownership or source metadata: %+v", pkg)
			}
		}
	}
}

func TestDiscoveryRejectsUnclassifiedRuntimeHostSource(t *testing.T) {
	for _, tc := range []struct{ path, category string }{
		{"toolchain/runtime/testdata/unknown/source.go", "uncovered-source"},
		{"toolchain/runtime/testdata/vfdpointerextra/source.go", "uncovered-source"},
		{"toolchain/runtime/testdata/vfdpointer/nested/source.go", "uncovered-source"},
		{"toolchain/runtime/testdata/vfdpointer/.hidden/source.go", "uncovered-source"},
		{"toolchain/runtime/testdata/vfdpointer/_hidden/source.go", "uncovered-source"},
		{"toolchain/runtime/testdata/vfdpointer/go.mod", "unclassified-module"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			contents := "package runner\n"
			if strings.HasSuffix(tc.path, "go.mod") {
				contents = "module example.invalid/nested\n\ngo 1.27.1\n"
			}
			root := writeArchitectureSources(t, map[string]string{tc.path: contents})
			_, findings, err := Discover(root, "go", Platform{"linux", "amd64"})
			if err != nil {
				t.Fatal(err)
			}
			if !slices.ContainsFunc(findings, func(finding Finding) bool { return finding.Category == tc.category && finding.Path == tc.path }) {
				t.Fatalf("unclassified runtime source escaped: %v", findings)
			}
		})
	}
}

func TestDiscoveryReportsRuntimeHostPackageErrors(t *testing.T) {
	directory := "toolchain/runtime/testdata/vfdpointer"
	root := writeArchitectureSources(t, map[string]string{directory + "/source.go": "package runner\nimport _ \"example.invalid/architecture/missing\"\n"})
	_, findings, err := Discover(root, "go", Platform{"linux", "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(findings, func(finding Finding) bool {
		return finding.Category == "package-error" && finding.Path == directory && strings.Contains(finding.Detail, "example.invalid/architecture/missing")
	}) {
		t.Fatalf("runtime host package import error escaped: %v", findings)
	}
}

func writeArchitectureSources(t *testing.T, sources map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if sources == nil {
		sources = map[string]string{}
	}
	sources["go.mod"] = "module example.invalid/architecture\n\ngo 1.27.1\n"
	for name, contents := range sources {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestDiscoveryRejectsRuntimeHostSourceSymlinks(t *testing.T) {
	for _, directory := range []bool{false, true} {
		t.Run(fmt.Sprintf("directory=%t", directory), func(t *testing.T) {
			root := writeArchitectureSources(t, nil)
			linked := t.TempDir()
			source := filepath.Join(linked, "source.go")
			if err := os.WriteFile(source, []byte("package runner\n"), 0600); err != nil {
				t.Fatal(err)
			}
			path := "toolchain/runtime/testdata/vfdpointer"
			target := linked
			if !directory {
				path += "/source.go"
				target = source
			}
			if err := os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, filepath.Join(root, path)); err != nil {
				t.Fatal(err)
			}
			_, findings, err := Discover(root, "go", Platform{"linux", "amd64"})
			if err != nil {
				t.Fatal(err)
			}
			if !slices.ContainsFunc(findings, func(finding Finding) bool { return finding.Category == "uncovered-source" && finding.Path == path }) {
				t.Fatalf("runtime host source symlink escaped: %v", findings)
			}
		})
	}
}

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
