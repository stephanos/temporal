package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestRunPinImpactDiagnosticFailuresPreservePrimaryStatus(t *testing.T) {
	fixture := newPinImpactFixture(t)
	t.Setenv("GOMAD3_BOOTSTRAP_GO", fixture.goCommand)
	baseline, malformed := t.TempDir(), t.TempDir()
	writePinImpactModule(t, baseline, fixture.baseline)
	if err := os.WriteFile(filepath.Join(malformed, "go.mod"), []byte("module\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(malformed, "go.sum"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	moduleArgs := []string{"--root", fixture.root, "--module", baseline, "--baseline-module", baseline, "--go", fixture.goCommand}
	fileArgs := []string{"--root", fixture.root, "--candidate", filepath.Join(baseline, "go.mod"), "--baseline", filepath.Join(baseline, "go.mod")}
	baselineBefore, malformedBefore := readModuleSnapshot(t, baseline), readModuleSnapshot(t, malformed)
	for _, test := range []struct {
		name       string
		arguments  []string
		proxy      string
		stdout     bool
		want       int
		diagnostic string
	}{
		{name: "unknown flag", arguments: []string{"--not-a-flag"}, want: 2, diagnostic: "flag provided but not defined: -not-a-flag"},
		{name: "help", arguments: []string{"--help"}, want: 2, diagnostic: "Usage of gomadtool pin-impact:"},
		{name: "positional argument", arguments: []string{"unexpected"}, want: 2, diagnostic: pinImpactUsage},
		{name: "unknown format", arguments: []string{"--format=invalid"}, want: 2, diagnostic: pinImpactUsage},
		{name: "both baseline flags", arguments: []string{"--baseline-module=baseline", "--baseline-ref=HEAD"}, want: 2, diagnostic: pinImpactUsage},
		{name: "partial file arguments", arguments: []string{"--baseline=go.mod"}, want: 2, diagnostic: pinImpactUsage},
		{name: "mixed file arguments", arguments: []string{"--baseline=go.mod", "--candidate=go.mod", "--module=module"}, want: 2, diagnostic: pinImpactUsage},
		{name: "missing candidate", arguments: append([]string{"--module", filepath.Join(baseline, "missing")}, "--baseline-module", baseline), want: 2, diagnostic: "read module file:"},
		{name: "missing baseline", arguments: []string{"--module", baseline, "--baseline-module", filepath.Join(baseline, "missing")}, want: 2, diagnostic: "read module file:"},
		{name: "malformed candidate", arguments: []string{"--root", fixture.root, "--module", malformed, "--baseline-module", baseline, "--go", fixture.goCommand}, want: 2, diagnostic: "candidate"},
		{name: "missing go command", arguments: []string{"--module", baseline, "--baseline-module", baseline, "--go", filepath.Join(baseline, "missing-go")}, want: 3, diagnostic: "gomad3 pin impact requires a go command"},
		{name: "unreachable proxy", arguments: moduleArgs, proxy: "off", want: 3, diagnostic: "module"},
		{name: "module publication", arguments: append(append([]string{}, moduleArgs...), "--output", baseline), want: 3, diagnostic: "publish replacement: rename "},
		{name: "module stdout", arguments: append(append([]string{}, moduleArgs...), "--json"), stdout: true, want: 3, diagnostic: "bad file descriptor"},
		{name: "file malformed candidate", arguments: []string{"--root", fixture.root, "--candidate", filepath.Join(malformed, "go.mod"), "--baseline", filepath.Join(baseline, "go.mod")}, want: 2, diagnostic: "candidate"},
		{name: "file publication", arguments: append(append([]string{}, fileArgs...), "--output", baseline), want: 3, diagnostic: "publish replacement: rename "},
		{name: "file stdout", arguments: fileArgs, stdout: true, want: 3, diagnostic: "bad file descriptor"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.proxy != "" {
				t.Setenv("GOPROXY", test.proxy)
			}
			for _, failed := range []bool{false, true} {
				var stdout, stderr bytes.Buffer
				var status int
				arguments := append([]string{"pin-impact"}, test.arguments...)
				if failed {
					diagnostics := newRefreshOutput(t, 0)
					if test.stdout {
						output := newRefreshOutput(t, 0)
						status = run(arguments, output, diagnostics)
						if !errors.Is(output.err, syscall.EBADF) {
							t.Fatalf("stdout error = %v, want EBADF", output.err)
						}
					} else {
						status = run(arguments, &stdout, diagnostics)
					}
					if !errors.Is(diagnostics.err, syscall.EBADF) {
						t.Fatalf("stderr error = %v, want EBADF", diagnostics.err)
					}
				} else {
					if test.stdout {
						status = run(arguments, newRefreshOutput(t, 0), &stderr)
					} else {
						status = run(arguments, &stdout, &stderr)
					}
					if !strings.Contains(stderr.String(), test.diagnostic) {
						t.Fatalf("stderr = %q, want %q", stderr.String(), test.diagnostic)
					}
				}
				if status != test.want || stdout.Len() != 0 {
					t.Fatalf("failed=%t status=%d stdout=%q, want primary status %d", failed, status, &stdout, test.want)
				}
			}
		})
	}
	if readModuleSnapshot(t, baseline) != baselineBefore || readModuleSnapshot(t, malformed) != malformedBefore {
		t.Fatal("diagnostic failures changed input module files")
	}
}
