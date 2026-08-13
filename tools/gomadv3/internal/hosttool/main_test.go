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

func runHosttool(t *testing.T, arguments []string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if status := run(arguments, &stdout, &stderr); status != 0 {
		t.Fatalf("run() status = %d, stderr = %q", status, stderr.String())
	}
	return stdout.String()
}
