package archcheck_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/archcheck"
)

// Any nested module the parent replaces with an in-tree directory (today the
// native backend) must stay out of the default dependency closure.
func TestDefaultDependencyClosureRejectsNativeImports(t *testing.T) {
	for _, testFile := range []bool{false, true} {
		root := t.TempDir()
		name := "source.go"
		if testFile {
			name = "source_test.go"
		}
		files := map[string]string{
			"go.mod":                             "module " + modulePath + "\n\ngo 1.27.1\nrequire " + modulePath + "/internal/backends/legacy v0.0.0\nreplace " + modulePath + "/internal/backends/legacy => ./internal/backends/legacy\n",
			"internal/backends/legacy/go.mod":    "module " + modulePath + "/internal/backends/legacy\n\ngo 1.27.1\n",
			"internal/backends/legacy/source.go": "package legacy\n",
			"internal/bridge/source.go":          "package bridge\nimport _ \"" + modulePath + "/internal/backends/legacy\"\n",
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
