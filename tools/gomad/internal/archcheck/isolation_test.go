package archcheck_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/archcheck"
	"golang.org/x/mod/modfile"
)

func TestNativeModulePinsMatchParent(t *testing.T) {
	parentBytes, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	childBytes, err := os.ReadFile("internal/backends/native/go.mod")
	if err != nil {
		t.Fatal(err)
	}
	parent, err := modfile.Parse("go.mod", parentBytes, nil)
	if err != nil {
		t.Fatal(err)
	}
	child, err := modfile.Parse("native/go.mod", childBytes, nil)
	if err != nil {
		t.Fatal(err)
	}
	if parent.Go.Version != child.Go.Version {
		t.Fatalf("native Go pin %s differs from parent %s", child.Go.Version, parent.Go.Version)
	}
	pins := map[string]string{}
	for _, required := range parent.Require {
		if required.Mod.Path != modulePath+"/internal/backends/native" {
			pins[required.Mod.Path] = required.Mod.Version
		}
	}
	for _, required := range child.Require {
		if required.Mod.Path == modulePath {
			continue
		}
		if pins[required.Mod.Path] != required.Mod.Version {
			t.Errorf("native dependency %s@%s differs from parent pin %s", required.Mod.Path, required.Mod.Version, pins[required.Mod.Path])
		}
		delete(pins, required.Mod.Path)
	}
	if len(pins) != 0 {
		t.Errorf("native module omitted parent pins: %v", pins)
	}
	parentSum, err := os.ReadFile("go.sum")
	if err != nil {
		t.Fatal(err)
	}
	childSum, err := os.ReadFile("internal/backends/native/go.sum")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(parentSum, childSum) {
		t.Fatal("native and parent dependency checksums differ")
	}
}

func TestDefaultDependencyClosureRejectsNativeImports(t *testing.T) {
	for _, testFile := range []bool{false, true} {
		root := t.TempDir()
		name := "source.go"
		if testFile {
			name = "source_test.go"
		}
		files := map[string]string{
			"go.mod":                             "module " + modulePath + "\n\ngo 1.27.1\nrequire " + modulePath + "/internal/backends/native v0.0.0\nreplace " + modulePath + "/internal/backends/native => ./internal/backends/native\n",
			"internal/backends/native/go.mod":    "module " + modulePath + "/internal/backends/native\n\ngo 1.27.1\n",
			"internal/backends/native/source.go": "package native\n",
			"internal/bridge/source.go":          "package bridge\nimport _ \"" + modulePath + "/internal/backends/native\"\n",
			"cmd/gomad/main.go":                  "package main\nfunc main() {}\n",
			"cmd/gomad/" + name:                  "package main\nimport _ \"" + modulePath + "/internal/bridge\"\n",
		}
		for name, source := range files {
			path := filepath.Join(root, name)
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
		}
		_, findings, err := archcheck.Discover(root, sourceGoCommand(), archcheck.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH})
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, finding := range findings {
			if finding.Category == "default-native-dependency" {
				found = true
			}
		}
		if found == archcheck.NativeSelected {
			t.Errorf("test-only=%t selected=%t native closure finding=%t: %v", testFile, archcheck.NativeSelected, found, findings)
		}
	}
}

func TestFixtureHelpersCannotEnterProductionImports(t *testing.T) {
	for _, imported := range []string{modulePath + "/internal/backends/testfixture", modulePath + "/internal/evidence/record/testfixture"} {
		if archcheck.ModuleMayImport(modulePath, modulePath+"/internal/runner", imported) {
			t.Errorf("production runner may import test helper %s", imported)
		}
	}
}
