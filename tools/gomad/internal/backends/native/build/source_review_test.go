package build

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
)

func TestSourceReviewGoRootRejectsInvalidCompilerIdentity(t *testing.T) {
	t.Parallel()
	for _, test := range []struct{ name, output string }{
		{"wrong version", "go1.26.0\n/absolute/root\n"},
		{"missing root", "go1.27.1\n"},
		{"relative root", "go1.27.1\nrelative/root\n"},
		{"extra field", "go1.27.1\n/absolute/root\nextra\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			command := filepath.Join(t.TempDir(), "go")
			if err := os.WriteFile(command, fmt.Appendf(nil, "#!/bin/sh\nprintf '%%s' '%s'\n", test.output), 0o700); err != nil {
				t.Fatal(err)
			}
			if root, err := SourceReviewGoRoot(t.Context(), command); err == nil || root != "" {
				t.Fatalf("invalid compiler identity returned root %q, error %v", root, err)
			}
		})
	}
}

func TestSourceReviewSelectsDeclaredPlatformAndTags(t *testing.T) {
	t.Parallel()
	command := os.Getenv("GOMAD_STOCK_GO")
	if command == "" {
		t.Skip("set GOMAD_STOCK_GO to pinned stock Go")
	}
	directory := t.TempDir()
	for name, contents := range map[string]string{
		"go.mod":             "module example.test/source\n\ngo 1.27.1\n",
		"main.go":            "package main\nfunc main() {}\n",
		"boundary_darwin.go": "package main\nimport _ \"syscall\"\n",
		"boundary_linux.go":  "package main\nimport _ \"os/exec\"\n",
		"tagged.go":          "//go:build reviewed\n\npackage main\nimport _ \"os/signal\"\n",
		"foreign_arm64.s":    "// Declared arm64 source inventory.\n",
		"foreign_amd64.s":    "// Declared amd64 source inventory.\n",
	} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, test := range []struct {
		goos, goarch, source, foreign, forbidden string
	}{
		{"darwin", "arm64", "boundary_darwin.go", "foreign_arm64.s", "syscall"},
		{"linux", "amd64", "boundary_linux.go", "foreign_amd64.s", "os/exec"},
	} {
		t.Run(test.goos, func(t *testing.T) {
			t.Parallel()
			review, err := ReviewSourceCapabilities(t.Context(), target.Spec{Kind: target.KindGoRun, Source: ".", WorkingDir: directory, BuildTags: []string{"reviewed"}}, command, test.goos, test.goarch)
			if err != nil {
				t.Fatal(err)
			}
			var root target.CapabilityPackage
			for _, pkg := range review.Closure.Packages {
				if pkg.ImportPath == "example.test/source" {
					root = pkg
				}
			}
			names := make([]string, len(root.Sources))
			for index, source := range root.Sources {
				names[index] = source.Name
			}
			if !slices.Equal(names, []string{test.source, "main.go", "tagged.go"}) || len(root.ForeignSources) != 1 || root.ForeignSources[0].Name != test.foreign || !slices.Contains(root.Imports, test.forbidden) || !slices.Contains(root.Imports, "os/signal") {
				t.Fatalf("declared source inventory = %#v", root)
			}
			if len(review.Findings) < 3 || len(review.Closure.Compatibility) != 0 {
				t.Fatalf("unreviewed platform capabilities were admitted: %#v", review)
			}
		})
	}
}

func TestSourceReviewRejectsExecutionAndUndeclaredPlatforms(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, goos, goarch string
		mode               target.CapabilityMode
	}{
		{"unsupported platform", "linux", "arm64", target.CapabilityModeClosure},
		{"linked execution", "linux", "amd64", target.CapabilityModeLinked},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := ReviewSourceCapabilities(t.Context(), target.Spec{Kind: target.KindGoRun, CapabilityMode: test.mode}, "missing", test.goos, test.goarch)
			if !target.IsInvalidCapabilityReview(err) {
				t.Fatalf("source review admission = %v", err)
			}
		})
	}
}
