package archcheck_test

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/archcheck"
)

// A retained package that imports a retired host package is refused even when
// the retired package is reintroduced; retained WASM neighbors stay admitted.
func TestInventoryRefusesRetainedImportsOfRetiredHostPackages(t *testing.T) {
	root := t.TempDir()
	const module = "example.invalid/retired"
	files := map[string]string{
		"go.mod":                    "module " + module + "\n\ngo 1.27.1\n",
		"internal/runner/source.go": "package runner\n",
		"internal/backends/wasm/backend/source.go":   "package backend\n",
		"internal/backends/wasm/toolchain/source.go": "package toolchain\n",
		"internal/model/world/source.go":             "package world\nimport _ \"" + module + "/internal/runner\"\n",
		"internal/model/clock/source.go":             "package clock\nimport _ \"" + module + "/internal/backends/wasm/backend\"\n",
		"internal/backends/wasm/cluster/source.go":   "package cluster\nimport _ \"" + module + "/internal/backends/wasm/toolchain\"\n",
		"internal/support/fs/source.go":              "package fs\n",
		"internal/model/capture/source.go":           "package capture\nimport _ \"" + module + "/internal/support/fs\"\n",
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
	var got []string
	for _, finding := range findings {
		if finding.Category == "retired-host-import" {
			got = append(got, finding.Path)
		}
	}
	slices.Sort(got)
	want := []string{module + "/internal/model/clock", module + "/internal/model/world"}
	if !slices.Equal(got, want) {
		t.Fatalf("retired-host-import findings = %v, want %v\n%v", got, want, findings)
	}
	for relative, retired := range map[string]bool{
		"cmd/gomad": true, "cmd/gomad/internal/cli": true, "internal/runner/options": true,
		"internal/evidence/record": true, "internal/evidence/qualification": true, "internal/backends": true,
		"internal/backends/wasm": true, "internal/backends/wasm/wasi": true, "internal/host/fs": true,
		"internal/backends/native/build":   true,
		"internal/backends/wasm/toolchain": false, "internal/backends/wasm/cluster": false,
		"internal/backends/wasm/internal/stockgo": false, "internal/evidence/qualification/corpus/basic/network": false,
		"internal/model/wasi": false, "internal/support/fs": false, "cmd/gomadx": false,
	} {
		if archcheck.RetiredHostPackage(relative) != retired {
			t.Errorf("RetiredHostPackage(%q) = %v, want %v", relative, !retired, retired)
		}
	}
}
