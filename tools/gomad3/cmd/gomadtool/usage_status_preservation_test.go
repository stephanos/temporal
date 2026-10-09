package main

import (
	"bytes"
	"errors"
	"maps"
	"os"
	"syscall"
	"testing"
)

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
