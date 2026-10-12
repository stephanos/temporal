package archcheck

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func fixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	files["go.mod"] = "module example.invalid/check\n\ngo 1.27.1\n"
	for name, source := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestPureStructure(t *testing.T) {
	for _, test := range []struct{ name, source, category string }{
		{"pure", "func F() int { return 1 }", ""},
		{"host-import", "import \"os\"; var _ = os.Getenv", "pure-import"},
		{"goroutine", "func F() { go func() {}() }", "pure-syntax"},
		{"time-select", "import \"time\"; func F() { select { case <-time.After(1): } }", "pure-syntax"},
		{"time-alias-select", "import clock \"time\"; func F() { timer := clock.NewTimer(1); select { case <-timer.C: } }", "pure-syntax"},
		{"time-declared-select", "import \"time\"; func F(timer *time.Timer) { select { case <-timer.C: } }", "pure-syntax"},
		{"syscall-errno", "import \"syscall\"; var _ = syscall.Errno(1); var _ = syscall.EINVAL", ""},
		{"syscall-host", "import \"syscall\"; var _ = syscall.Getpid", "pure-import"},
		{"syscall-dot", "import . \"syscall\"; var _ = EINVAL", "pure-import"},
		{"cgo", "import \"C\"", "pure-syntax"},
		{"impure-init", "func init() { host() }", "pure-syntax"},
		{"pure-init", "func pure() {}; func init() { pure() }", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := fixture(t, map[string]string{"world/pure.go": "package world\n" + test.source, "world/host.go": "package world\nfunc host() {}"})
			config := Config{Module: "example.invalid/check", PureFiles: func(path string) bool { return path == "world/pure.go" }, PureImports: []string{"time", "syscall"}}
			findings, err := Check(root, "go", Platform{runtime.GOOS, runtime.GOARCH}, config)
			if err != nil {
				t.Fatal(err)
			}
			if test.category == "" {
				if len(findings) != 0 {
					t.Fatalf("pure source rejected: %v", findings)
				}
			} else if !slices.ContainsFunc(findings, func(f Finding) bool { return f.Category == test.category && strings.Contains(f.Path, "pure.go") }) {
				t.Fatalf("missing %s: %v", test.category, findings)
			}
		})
	}
}

func TestConfigurationMustMatchSource(t *testing.T) {
	root := fixture(t, map[string]string{"world/pure.go": "package world"})
	_, err := Check(root, "go", Platform{runtime.GOOS, runtime.GOARCH}, Config{Module: "example.invalid/check", PureFiles: func(path string) bool { return path == "world/renamed.go" }})
	if err == nil || !strings.Contains(err.Error(), "pure") {
		t.Fatalf("unmatched pure set accepted: %v", err)
	}
}

func TestConfiguredEdgesAndOwners(t *testing.T) {
	root := fixture(t, map[string]string{"world/source.go": "package world\nimport _ \"example.invalid/check/backends/native\"", "backends/native/source.go": "package native"})
	config := Config{Module: "example.invalid/check", Owners: map[string][]string{"shared": {"example.invalid/check/world"}, "native": {"example.invalid/check/backends/native"}}, OwnerEdges: map[string][]string{}}
	findings, err := Check(root, "go", Platform{runtime.GOOS, runtime.GOARCH}, config)
	if err != nil {
		t.Fatal(err)
	}
	for _, category := range []string{"module-edge", "owner-edge"} {
		if !slices.ContainsFunc(findings, func(f Finding) bool { return f.Category == category && strings.Contains(f.Detail, "backends/native") }) {
			t.Fatalf("missing %s: %v", category, findings)
		}
	}
	config.Exceptions = []Exception{{From: "example.invalid/check/world", To: "example.invalid/check/backends/native", RemovalTask: "fn-164.5"}}
	config.OwnerEdges["shared"] = []string{"native"}
	findings, err = Check(root, "go", Platform{runtime.GOOS, runtime.GOARCH}, config)
	if err != nil || len(findings) != 0 {
		t.Fatalf("allowed edge rejected: %v, %v", findings, err)
	}
}

func TestSubtreeAndBuildTags(t *testing.T) {
	root := fixture(t, map[string]string{"backends/wasm/wasi/pure.go": "package wasi", "backends/wasm/wasi/ignored.go": "//go:build windows\n\npackage wasi\nimport _ \"os\"", "backends/wasm/wasi/tagged.go": "//go:build test_dep\n\npackage wasi\nimport _ \"os\""})
	findings, err := Check(filepath.Join(root, "backends/wasm"), "go", Platform{"linux", "amd64"}, Config{Module: "example.invalid/check", PureFiles: func(path string) bool { return strings.HasPrefix(path, "wasi/") }})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Category != "pure-import" || findings[0].Path != "wasi/tagged.go" {
		t.Fatalf("wrong scoped source set: %v", findings)
	}
}
