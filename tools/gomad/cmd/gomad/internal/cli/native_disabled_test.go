//go:build !gomad_native

package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultBuildRefusesNativeWithoutFallback(t *testing.T) {
	t.Setenv("GOMAD_WASM_INSTALLATION", "")
	manifest := filepath.Join(t.TempDir(), "native.json")
	if err := os.WriteFile(manifest, []byte(`{"schema":"gomad.installation/v1","backend":"native"}`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, arguments := range [][]string{
		{"--installation", manifest, "explore", "exec", "--", "/bin/true"},
		{"explore", "exec", "--", "/bin/true"},
		{"--installation", manifest, "install"},
		{"__supervisor"},
	} {
		var stdout, stderr bytes.Buffer
		app := hostApplication()
		app.dispatch = func(string, io.Reader, io.Writer) error { return errors.New("unexpected private dispatch") }
		status := app.launch(arguments, &stdout, &stderr)
		if status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "native backend not built in") || !strings.Contains(stderr.String(), "gomad_native") {
			t.Errorf("%v: status=%d stdout=%q stderr=%q", arguments, status, stdout.String(), stderr.String())
		}
	}
}

func TestDefaultDoctorReportsNativeNotBuiltIn(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := Run([]string{"doctor", "--json", "--artifacts", t.TempDir()}, &stdout, &stderr)
	if status != 2 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "not built in") || !strings.Contains(stdout.String(), "gomad_native") || strings.Contains(stdout.String(), "unsupported; supported=") {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestDefaultUsageAndHistoricalInspection(t *testing.T) {
	var invalidOut, invalidErr bytes.Buffer
	if status := Run([]string{"inspect"}, &invalidOut, &invalidErr); status != 2 || strings.Contains(invalidErr.String(), "analyze") || strings.Contains(invalidErr.String(), "qualify-set") {
		t.Fatalf("inspect usage status=%d stdout=%q stderr=%q", status, invalidOut.String(), invalidErr.String())
	}
	var stdout, stderr bytes.Buffer
	if status := Run(nil, &stdout, &stderr); status != 2 || !strings.Contains(stderr.String(), "inspect") || strings.Contains(stderr.String(), "analyze") || strings.Contains(stderr.String(), "qualify-set") {
		t.Fatalf("usage status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	missing := filepath.Join(t.TempDir(), "historical")
	if status := Run([]string{"inspect", missing}, &stdout, &stderr); status != 2 || !strings.Contains(stderr.String(), "inspect "+missing) || strings.Contains(stderr.String(), "native backend not built in") {
		t.Fatalf("inspect status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}
