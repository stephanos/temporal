package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPinImpactReportsUnselectedPinsWithoutChangingCandidate(t *testing.T) {
	directory := t.TempDir()
	module := []byte("module example.com/pin-test\n\ngo 1.27.1\n")
	sums := []byte("")
	modPath := filepath.Join(directory, "go.mod")
	if err := os.WriteFile(modPath, module, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "go.sum"), sums, 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	status := run([]string{"pin-impact", "--root", "../..", "--candidate", modPath, "--baseline", modPath, "--format", "json"}, &stdout, &stderr)
	if status != 0 {
		t.Fatalf("status %d: %s", status, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"schema":"gomad3.pin-impact/v1"`) || !strings.Contains(stdout.String(), `"status":"not_selected"`) {
		t.Fatalf("missing report: %s", stdout.String())
	}
	got, err := os.ReadFile(modPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(module, got) {
		t.Fatal("candidate module mutated")
	}
	got, err = os.ReadFile(filepath.Join(directory, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sums, got) {
		t.Fatal("candidate sums mutated")
	}
}

func TestPinImpactExitStatuses(t *testing.T) {
	directory := t.TempDir()
	modulePath := filepath.Join(directory, "go.mod")
	for _, test := range []struct {
		name, module, sum, root string
		want                    int
	}{
		{"unaffected", "module example.com/pins\n\ngo 1.27.1\nrequire github.com/getsentry/sentry-go v0.46.0\n", "github.com/getsentry/sentry-go v0.46.0 h1:mbdDaarbUdOt9X+dx6kDdntkShLEX3/+KyOsVDTPDj0=\n", "../..", 0},
		{"invalidated", "module example.com/pins\n\ngo 1.27.1\nrequire github.com/getsentry/sentry-go v0.47.0\n", "", "../..", 1},
		{"unknown", "module example.com/pins\n\ngo 1.27.1\nrequire github.com/getsentry/sentry-go v0.46.0\n", "", "../..", 1},
		{"invalid", "malformed module", "", "../..", 2},
		{"missing module", "go 1.27.1\n", "", "../..", 2},
		{"infrastructure", "module example.com/pins\n\ngo 1.27.1\n", "", directory, 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := os.WriteFile(modulePath, []byte(test.module), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(directory, "go.sum"), []byte(test.sum), 0600); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			if got := run([]string{"pin-impact", "--root", test.root, "--candidate", modulePath, "--baseline", modulePath, "--format", "json"}, &stdout, &stderr); got != test.want {
				t.Fatalf("want status %d got %d: %s", test.want, got, stderr.String())
			}
			if test.name == "unknown" && !strings.Contains(stdout.String(), `"status":"unknown"`) {
				t.Fatal("missing unknown result")
			}
		})
	}
}
