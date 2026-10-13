package main

import (
	"bytes"
	"errors"
	"io"
	"maps"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

type compatibilitySourceWriter struct {
	output    io.Writer
	attempted bytes.Buffer
	calls     int
	err       error
}

func (writer *compatibilitySourceWriter) Write(data []byte) (int, error) {
	writer.calls++
	if _, err := writer.attempted.Write(data); err != nil {
		return 0, err
	}
	n, err := writer.output.Write(data)
	writer.err = err
	return n, err
}

func TestCompatibilityPackSourceUsagePreservesBytesAndWriteFailure(t *testing.T) {
	const usage = "usage: gomadtool compatibility-pack discover|review|refresh|generate|check|qualify [flags]\n"
	for _, test := range []struct {
		name      string
		arguments []string
		closed    bool
		status    int
	}{
		{name: "no command", status: 2},
		{name: "unknown command", arguments: []string{"source-unknown"}, status: 2},
		{name: "no command closed writer", closed: true, status: 1},
		{name: "unknown command closed writer", arguments: []string{"source-unknown"}, closed: true, status: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			writer := compatibilitySourceWriter{output: &stderr}
			if test.closed {
				file, err := os.CreateTemp(t.TempDir(), "closed-output")
				if err != nil {
					t.Fatal(err)
				}
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
				writer.output = file
			}
			status := runCompatibilityPack(test.arguments, &stdout, &writer)
			if status != test.status || writer.calls != 1 || writer.attempted.String() != usage || stdout.Len() != 0 {
				t.Errorf("status=%d want=%d, calls=%d, attempted=%q, stdout=%q", status, test.status, writer.calls, writer.attempted.String(), stdout.String())
			}
			if test.closed {
				if !errors.Is(writer.err, os.ErrClosed) || stderr.Len() != 0 {
					t.Fatalf("closed writer error=%v, diagnostics=%q", writer.err, stderr.String())
				}
			} else if writer.err != nil || stderr.String() != usage {
				t.Fatalf("ordinary writer error=%v, diagnostics=%q", writer.err, stderr.String())
			}
		})
	}
}

func TestCompatibilityPackPathsHonorAnExternalRoot(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()

	_, compatibilityRoot, request, err := resolveCompatibilityPackPaths(root, "", "backends/native/compatibilitypack/requests/x.json")
	if err != nil || compatibilityRoot != filepath.Join(root, "backends", "native", "compatibilitypack") || request != filepath.Join(root, "backends", "native", "compatibilitypack", "requests", "x.json") {
		t.Fatalf("default paths = %q, %q, %v", compatibilityRoot, request, err)
	}

	_, compatibilityRoot, request, err = resolveCompatibilityPackPaths(root, external, "requests/x.json")
	if err != nil || compatibilityRoot != external || request != filepath.Join(external, "requests", "x.json") {
		t.Fatalf("external paths = %q, %q, %v", compatibilityRoot, request, err)
	}
	if !pathWithin(compatibilityRoot, request) {
		t.Fatal("external request is not below the external root")
	}

	if _, _, _, err := resolveCompatibilityPackPaths(root, external, filepath.Join(root, "requests", "x.json")); err == nil {
		t.Fatal("a request outside the external root was accepted")
	}
	for _, override := range []string{"relative/root", external + "/../" + filepath.Base(external)} {
		if _, _, _, err := resolveCompatibilityPackPaths(root, override, "requests/x.json"); err == nil {
			t.Fatalf("--compatibility-root %q was accepted", override)
		}
	}
}

func TestCompatibilityPathBaseKeepsDefaultPathsRootRelative(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	if base := compatibilityPathBase(root, filepath.Join(root, "backends", "native", "compatibilitypack"), ""); base != root {
		t.Fatalf("default base = %q, want %q", base, root)
	}
	if base := compatibilityPathBase(root, external, external); base != external {
		t.Fatalf("external base = %q, want %q", base, external)
	}
}

func TestRunCompatibilityPackUsageStatusPreservation(t *testing.T) {
	const usage = "usage: gomadtool compatibility-pack discover|review|refresh|generate|check|qualify [flags]\n"
	for _, test := range []struct {
		name      string
		arguments []string
	}{
		{name: "missing subcommand", arguments: []string{"compatibility-pack"}},
		{name: "unknown subcommand", arguments: []string{"compatibility-pack", "--not-a-flag"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			maintainerWrite(t, root, "sentinels/keep", "unchanged publication sentinel\n")
			t.Chdir(root)
			for _, failed := range []bool{false, true} {
				name := "healthy stderr"
				if failed {
					name = "EBADF stderr"
				}
				t.Run(name, func(t *testing.T) {
					before := maintainerSnapshot(t, root)
					var stdout, stderr bytes.Buffer
					output := compatibilitySourceWriter{output: &stdout}
					diagnostics := compatibilitySourceWriter{output: &stderr}
					wantStatus := 2
					var failedOutput, failedDiagnostics *refreshOutput
					if failed {
						failedOutput, failedDiagnostics = newRefreshOutput(t, 0), newRefreshOutput(t, 0)
						output.output, diagnostics.output = failedOutput, failedDiagnostics
						wantStatus = 1
					}
					status := run(test.arguments, &output, &diagnostics)
					if status != wantStatus {
						t.Errorf("status = %d, want %d", status, wantStatus)
					}
					if diagnostics.calls != 1 || diagnostics.attempted.String() != usage {
						t.Fatalf("stderr calls = %d, attempted = %q, want one usage write", diagnostics.calls, diagnostics.attempted.String())
					}
					if output.calls != 0 || output.attempted.Len() != 0 || output.err != nil || stdout.Len() != 0 {
						t.Fatalf("stdout calls = %d, attempted = %q, error = %v, delivered = %q", output.calls, output.attempted.String(), output.err, stdout.String())
					}
					if failed {
						if !errors.Is(diagnostics.err, syscall.EBADF) || !errors.Is(failedDiagnostics.err, syscall.EBADF) || stderr.Len() != 0 || failedDiagnostics.Len() != 0 || failedOutput.err != nil || failedOutput.Len() != 0 {
							t.Fatalf("failed stderr error = %v, underlying = %v, delivered = %q; stdout error = %v", diagnostics.err, failedDiagnostics.err, stderr.String(), failedOutput.err)
						}
						for _, file := range []*os.File{failedOutput.file, failedDiagnostics.file} {
							contents, err := os.ReadFile(file.Name())
							if err != nil {
								t.Fatal(err)
							}
							if len(contents) != 0 {
								t.Fatalf("read-only output received bytes = %q", contents)
							}
						}
					} else if diagnostics.err != nil || stderr.String() != usage {
						t.Fatalf("healthy stderr error = %v, delivered = %q, want %q", diagnostics.err, stderr.String(), usage)
					}
					if after := maintainerSnapshot(t, root); !maps.Equal(before, after) {
						t.Fatal("usage changed publication bytes or file presence")
					}
					t.Logf("status=%d stderr-calls=%d attempted=%q stderr-error=%v stdout-calls=%d stdout-attempted=%q publication unchanged", status, diagnostics.calls, diagnostics.attempted.String(), diagnostics.err, output.calls, output.attempted.String())
				})
			}
		})
	}
}
