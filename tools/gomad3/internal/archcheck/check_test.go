package archcheck

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestPureImportAllowlistUsesExactPaths(t *testing.T) {
	const allowed = "github.com/go-json-experiment/json/jsontext"
	const nested = allowed + "/internal/jsonwire"
	file, err := parser.ParseFile(token.NewFileSet(), "pure.go", "package pure\nimport (\n_ \""+allowed+"\"\n_ \""+nested+"\"\n)", parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	findings := syntax("pure.go", file, []string{allowed}, map[string]bool{}, map[string]bool{})
	if len(findings) != 1 || findings[0].Category != "pure-import" || findings[0].Detail != nested {
		t.Fatalf("exact pure import edge findings = %v", findings)
	}
}

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

func TestPureCgoRejectedWhenDisabled(t *testing.T) {
	t.Setenv("CGO_ENABLED", "0")
	root := fixture(t, map[string]string{
		"world/pure.go":             "package world\nimport \"C\"",
		"world/host.go":             "package world\nfunc host(){}",
		"world/inactive_windows.go": "package world\nimport \"os\";var _=os.Getenv",
	})
	config := Config{Module: "example.invalid/check", PureFiles: func(path string) bool { return strings.HasPrefix(path, "world/") && path != "world/host.go" }}
	findings, err := Check(root, "go", Platform{"linux", "amd64"}, config)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Category != "pure-syntax" || findings[0].Path != "world/pure.go" {
		t.Fatalf("inactive cgo must be rejected without activating other platform files: %v", findings)
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
	config.OwnerEdges["shared"] = []string{"native"}
	findings, err = Check(root, "go", Platform{runtime.GOOS, runtime.GOARCH}, config)
	if err != nil || len(findings) != 1 || findings[0].Category != "module-edge" {
		t.Fatalf("owner edge rejected or module edge admitted: %v, %v", findings, err)
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

func TestBackendNamespaceOwners(t *testing.T) {
	const module = "example.invalid/check"
	root := fixture(t, map[string]string{
		"backends/source.go":        "package backends",
		"backends/native/source.go": "package native;import _ \"example.invalid/check/backends\"",
		"backends/wasm/source.go":   "package wasm;import _ \"example.invalid/check/backends\"",
		"backends/shadow/source.go": "package shadow",
	})
	packages, err := List(root, "go", Platform{runtime.GOOS, runtime.GOARCH}, false)
	if err != nil {
		t.Fatal(err)
	}
	findings := PackageEdges(module, packages)
	if len(findings) != 1 || findings[0].Category != "ownerless" || findings[0].Path != module+"/backends/shadow" {
		t.Fatalf("shadow namespace must not inherit the neutral protocol owner: %v", findings)
	}
	if OwnerMayImport(module, "cli", "wasm", module+"/cmd/gomad/internal/cli", module+"/backends/wasm/command/unchecked") {
		t.Fatal("CLI descendant wiring admitted")
	}
}

func TestCapturedMayUseOnlyThePublicHostFSFacade(t *testing.T) {
	const module = "example.invalid/check"
	if !OwnerMayImport(module, "captured", "hostfs", module+"/captured", module+"/hostfs") {
		t.Fatal("captured cannot use the public hostfs facade")
	}
	if OwnerMayImport(module, "captured", "hostfs", module+"/captured", module+"/internal/hostfs") {
		t.Fatal("captured may bypass the public hostfs facade")
	}
}
