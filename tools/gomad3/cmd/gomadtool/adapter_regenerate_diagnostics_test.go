package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestRunAdapterRegenerateDiagnosticFailuresPreserveStatus(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, diagnostic string
		arguments        []string
		status           int
	}{
		{name: "flag", arguments: []string{"--unknown"}, status: 2, diagnostic: "flag provided but not defined"},
		{name: "help", arguments: []string{"--help"}, status: 2, diagnostic: "Usage of gomadtool adapter-regenerate:"},
		{name: "positional", arguments: []string{"unexpected"}, status: 2, diagnostic: adapterRegenerateUsage},
		{name: "approval aliases", arguments: []string{"--approve=a", "--approve-review=b"}, status: 2, diagnostic: "must name the same digest"},
		{name: "stage approval", arguments: []string{"--stage-only"}, status: 2, diagnostic: "requires an approval digest"},
		{name: "recover", arguments: []string{"--recover", "--root", file}, status: 3, diagnostic: "not a directory"},
		{name: "missing Go", arguments: []string{"--go", filepath.Join(root, "missing-go")}, status: 3, diagnostic: "requires the pinned go command"},
		{name: "verify usage", arguments: []string{"--verify", "--module=google.golang.org/grpc"}, status: 2, diagnostic: adapterRegenerateUsage},
		{name: "verify failure", arguments: []string{"--verify", "--module=example.test/unknown", "--module-dir", root}, status: 1, diagnostic: "no adapter is registered"},
		{name: "module usage", arguments: nil, status: 2, diagnostic: adapterRegenerateUsage + "\nregenerable adapters: "},
		{name: "input failure", arguments: []string{"--root", root, "--module=example.test/unknown", "--version=v1.0.0"}, status: 2, diagnostic: "no adapter is pinned"},
		{name: "download failure", arguments: []string{"--root", root, "--module=github.com/getsentry/sentry-go", "--version=v0.46.1"}, status: 3, diagnostic: "module lookup disabled by GOPROXY=off"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("GOPROXY", "off")
			for _, failed := range []bool{false, true} {
				var stdout, stderr bytes.Buffer
				status := 0
				if failed {
					diagnostics := newRefreshOutput(t, 0)
					status = runAdapterRegenerate(test.arguments, &stdout, diagnostics)
					if !errors.Is(diagnostics.err, syscall.EBADF) {
						t.Fatalf("stderr error = %v, want EBADF", diagnostics.err)
					}
				} else {
					status = runAdapterRegenerate(test.arguments, &stdout, &stderr)
					if !strings.Contains(stderr.String(), test.diagnostic) {
						t.Fatalf("stderr = %q, want %q", stderr.String(), test.diagnostic)
					}
				}
				if status != test.status || stdout.Len() != 0 {
					t.Fatalf("failed=%t status=%d stdout=%q, want %d", failed, status, &stdout, test.status)
				}
			}
		})
	}
	t.Run("missing working directory", func(t *testing.T) {
		directory := t.TempDir()
		t.Chdir(directory)
		t.Setenv("PWD", "")
		if err := os.Remove(directory); err != nil {
			t.Fatal(err)
		}
		_, workingDirectoryError := os.Getwd()
		if !errors.Is(workingDirectoryError, os.ErrNotExist) {
			t.Fatalf("deleted working directory error = %v", workingDirectoryError)
		}
		var stdout, stderr bytes.Buffer
		if status := runAdapterRegenerate(nil, &stdout, &stderr); status != 2 || stderr.String() != workingDirectoryError.Error()+"\n" || stdout.Len() != 0 {
			t.Fatalf("status=%d stderr=%q", status, &stderr)
		}
		diagnostics := newRefreshOutput(t, 0)
		if status := runAdapterRegenerate(nil, &stdout, diagnostics); status != 2 || !errors.Is(diagnostics.err, syscall.EBADF) || stdout.Len() != 0 {
			t.Fatalf("status=%d stderr error=%v", status, diagnostics.err)
		}
	})
}

func TestRunAdapterRegenerateOutputFailuresPreserveStatus(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{"human", "json"} {
		t.Run(format, func(t *testing.T) {
			arguments := []string{"--root", root, "--module=github.com/Masterminds/sprig/v3", "--version=v3.2.3"}
			if format == "json" {
				arguments = append(arguments, "--json")
			}
			var healthy, diagnostic bytes.Buffer
			if status := runAdapterRegenerate(arguments, &healthy, &diagnostic); status != 0 || healthy.Len() == 0 || diagnostic.Len() != 0 {
				t.Fatalf("healthy %s: status=%d stdout=%q stderr=%q", format, status, &healthy, &diagnostic)
			}
			for _, failedDiagnostic := range []bool{false, true} {
				output := newRefreshOutput(t, 0)
				var stderr bytes.Buffer
				status := 0
				if failedDiagnostic {
					failed := newRefreshOutput(t, 0)
					status = runAdapterRegenerate(arguments, struct{ io.Writer }{output}, failed)
					if !errors.Is(failed.err, syscall.EBADF) {
						t.Fatalf("failed %s diagnostic = %v", format, failed.err)
					}
				} else {
					status = runAdapterRegenerate(arguments, struct{ io.Writer }{output}, &stderr)
					if !strings.Contains(stderr.String(), syscall.EBADF.Error()) {
						t.Fatalf("failed %s stdout diagnostic = %q", format, &stderr)
					}
				}
				if status != 3 || !errors.Is(output.err, syscall.EBADF) {
					t.Fatalf("failed %s output: status=%d error=%v", format, status, output.err)
				}
			}
		})
	}
}
