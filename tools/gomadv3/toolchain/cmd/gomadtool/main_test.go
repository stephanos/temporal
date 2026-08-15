package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunBuildKey(t *testing.T) {
	root := t.TempDir()
	patch := filepath.Join(root, "gomad.patch")
	overlay := filepath.Join(root, "overlay")
	if err := os.MkdirAll(overlay, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(patch, []byte("patch"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(overlay, "gomad.go"), []byte("package gomad"), 0o600); err != nil {
		t.Fatal(err)
	}
	arguments := []string{
		"build-key", "--go-version=go1.26.4", "--archive-sha256=archive", "--patch=" + patch,
		"--overlay=" + overlay, "--host-os=darwin", "--host-arch=arm64", "--bootstrap-version=bootstrap",
		"--recipe-version=recipe", "--build-path=/usr/bin:/bin", "--bash-path=/bin/bash", "--bash-version=5.2",
	}
	first := runHosttool(t, arguments)
	if len(strings.TrimSpace(first)) != 64 {
		t.Fatalf("build key = %q", first)
	}
	if repeated := runHosttool(t, arguments); repeated != first {
		t.Fatalf("repeated key = %q, want %q", repeated, first)
	}
	if err := os.WriteFile(filepath.Join(overlay, "gomad.go"), []byte("package changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if changed := runHosttool(t, arguments); changed == first {
		t.Fatal("build key did not change with overlay contents")
	}
}

func TestRunBuildKeyRejectsIncompleteInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if status := run([]string{"build-key"}, &stdout, &stderr); status != 2 {
		t.Fatalf("status = %d, stderr = %q", status, stderr.String())
	}
}

func TestRunTestModeProjectsRegistryFields(t *testing.T) {
	for _, test := range []struct {
		arguments []string
		want      string
	}{
		{arguments: []string{"test-mode", "--mode=test", "--output=tiers"}, want: "test-builder\ntest-runtime\ntest-upstream\n"},
		{arguments: []string{"test-mode", "--mode=test-runtime", "--output=success"}, want: "gomadv3 runtime tier passed\n"},
	} {
		var stdout, stderr bytes.Buffer
		if status := run(test.arguments, &stdout, &stderr); status != 0 || stderr.Len() != 0 || stdout.String() != test.want {
			t.Fatalf("run(%q) = %d, stdout %q, stderr %q", test.arguments, status, stdout.String(), stderr.String())
		}
	}
}

func TestRunPatchValidateChecksCurrentModule(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if status := run([]string{"patch-validate", "--root=" + root}, &stdout, &stderr); status != 0 || stderr.Len() != 0 {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout.String(), stderr.String())
	}
}

func TestRunScriptValidateChecksCurrentModule(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if status := run([]string{"script-validate", "--root=" + root}, &stdout, &stderr); status != 0 || stderr.Len() != 0 {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout.String(), stderr.String())
	}
}

func TestRunPatchCommandsRejectIncompleteInput(t *testing.T) {
	for _, arguments := range [][]string{{"patch-validate"}, {"patch-materialize", "--root=/tmp"}, {"patch-regenerate", "--root=/tmp"}} {
		var stdout, stderr bytes.Buffer
		if status := run(arguments, &stdout, &stderr); status != 2 {
			t.Fatalf("run(%q) status = %d, stderr = %q", arguments, status, stderr.String())
		}
	}
}

func TestRunToolchainBuildRejectsIncompleteInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if status := run([]string{"toolchain-build"}, &stdout, &stderr); status != 2 {
		t.Fatalf("status = %d, stderr = %q", status, stderr.String())
	}
}

func TestRunTestRejectsIncompleteInput(t *testing.T) {
	for _, arguments := range [][]string{
		{"test", "--mode=test-upstream"},
		{"test", "--root=/tmp", "--mode=test-interception", "--go=/tmp/go"},
	} {
		var stdout, stderr bytes.Buffer
		if status := run(arguments, &stdout, &stderr); status != 2 {
			t.Fatalf("run(%q) status = %d, stderr = %q", arguments, status, stderr.String())
		}
	}
}

func TestBootstrapGofmtResolvesInstalledTool(t *testing.T) {
	gofmt, err := bootstrapGofmt("")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(gofmt) != "gofmt" {
		t.Fatalf("bootstrapGofmt() = %q", gofmt)
	}
}

func TestRunCheckedRunWritesCompatibilityResult(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	status := run([]string{
		"checked-run", "5", "7", "fixture seed=1 mode=unit iteration=0", root, "--",
		"/bin/sh", "-c", "printf output; printf error >&2; exit 7",
	}, &stdout, &stderr)
	if status != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout.String(), stderr.String())
	}
	for name, want := range map[string]string{
		"stdout": "output", "stderr": "error", "status": "7\n", "timed-out": "0\n", "output-truncated": "0\n",
	} {
		contents, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(contents) != want {
			t.Fatalf("%s = %q, want %q", name, contents, want)
		}
	}
}

func TestRunCheckedRunDistinguishesExit124FromTimeout(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	status := run([]string{"checked-run", "5", "124", "false timeout", root, "--", "/bin/sh", "-c", "exit 124"}, &stdout, &stderr)
	if status != 1 || !strings.Contains(stderr.String(), "status 124 was not a timeout") {
		t.Fatalf("status = %d, stderr = %q", status, stderr.String())
	}
}

func runHosttool(t *testing.T, arguments []string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if status := run(arguments, &stdout, &stderr); status != 0 {
		t.Fatalf("run() status = %d, stderr = %q", status, stderr.String())
	}
	return stdout.String()
}
