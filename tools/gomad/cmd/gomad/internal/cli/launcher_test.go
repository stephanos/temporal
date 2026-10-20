package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLauncherRoutesTemporalSetupBeforeInstallation(t *testing.T) {
	for _, entry := range os.Environ() {
		key, value, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "GOMAD_") {
			t.Setenv(key, value)
			if err := os.Unsetenv(key); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, tc := range []struct {
		name, guard, want string
		ambient           bool
	}{
		{"explicit inputs", "", "--temporal-package", false},
		{"ambient guard", "1", "unset GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD; Temporal setup accepts explicit inputs only", true},
		{"empty ambient guard", "", "unset GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD; Temporal setup accepts explicit inputs only", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.ambient {
				t.Setenv("GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD", tc.guard)
			}
			var stdout, stderr bytes.Buffer
			status := Run([]string{"install", "--temporal-checkout", "/checkout", "--output", filepath.Join(t.TempDir(), "installation.json")}, &stdout, &stderr)
			if status != 2 || !strings.Contains(stderr.String(), tc.want) || stdout.Len() != 0 {
				t.Fatalf("Temporal setup was not routed before installation: status=%d stderr=%s", status, stderr.String())
			}
		})
	}
}

func TestLauncherWASMInstallationKeepsTopLevelFlagErrors(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "installation.json")
	if err := os.WriteFile(manifest, []byte(`{"schema":"gomad-wasm.installation/v1"}`), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if status := Run([]string{"--installation=" + manifest, "--not-a-global-flag"}, &stdout, &stderr); status != 2 || stdout.Len() != 0 || !strings.HasPrefix(stderr.String(), "flag provided but not defined: -not-a-global-flag\nUsage of gomad:\n") {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestUnifiedLauncherRejectsNativeFlagsForWASMInstallation(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "installation.json")
	if err := os.WriteFile(manifest, []byte(`{"schema":"gomad-wasm.installation/v1","options":{"Config":{"Profile":"gomad.wasi-cooperative/v1"}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, option := range []string{"--toolchain-root=/native", "--clock-tick=forward", "--io-ro-mount=/host=/guest", "--guide"} {
		var stdout, stderr bytes.Buffer
		status := Run([]string{"--installation", manifest, "explore", option, "go-run", "."}, &stdout, &stderr)
		if status != 2 || !strings.Contains(stderr.String(), "flag provided but not defined") || !strings.Contains(stderr.String(), "Usage of gomad explore:") || strings.Contains(stderr.String(), "installed WASM helper") || strings.Contains(stderr.String(), "panic") {
			t.Errorf("option=%s status=%d stdout=%q stderr=%q", option, status, stdout.String(), stderr.String())
		}
	}
	var stdout, stderr bytes.Buffer
	status := Run([]string{"--installation", manifest, "explore", "--help"}, &stdout, &stderr)
	if status != 2 || !strings.Contains(stderr.String(), "Usage of gomad explore:") || strings.Contains(stderr.String(), "-toolchain-root") || strings.Contains(stderr.String(), "-clock-tick") || strings.Contains(stderr.String(), "-io-ro-mount") || !strings.Contains(stderr.String(), "-choices") {
		t.Fatalf("WASM help status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	status = Run([]string{"--installation", manifest, "--help"}, &stdout, &stderr)
	if status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "install") || !strings.Contains(stderr.String(), "minimize") || strings.Contains(stderr.String(), "toolchain-root") || strings.Contains(stderr.String(), "clock-tick") {
		t.Fatalf("selected launcher help status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestUnifiedLauncherRejectsUnknownManifestBackend(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "installation.json")
	if err := os.WriteFile(manifest, []byte(`{"backend":"nope"}`), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	status := Run([]string{"--installation", manifest, "explore", "go-run", "."}, &stdout, &stderr)
	if status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "nope") || !strings.Contains(stderr.String(), "backend") {
		t.Fatalf("unknown backend status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestUnifiedLauncherDefersMalformedManifestForSourceFreeCommands(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "broken-installation.json")
	if err := os.WriteFile(manifest, []byte(`{"schema":`), 0600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(t.TempDir(), "missing-retained")
	for _, test := range []struct {
		arguments []string
		status    int
		wantPath  bool
	}{
		{[]string{"--help"}, 2, false}, {[]string{"explore", "--help"}, 2, false}, {[]string{"inspect", missing}, 2, true}, {[]string{"recover", missing}, 2, true}, {[]string{"report", "--result", missing}, 3, true},
	} {
		var stdout, stderr bytes.Buffer
		status := Run(append([]string{"--installation", manifest}, test.arguments...), &stdout, &stderr)
		if status != test.status || stdout.Len() != 0 || strings.Contains(stderr.String(), "installation header") || strings.Contains(stderr.String(), "broken-installation.json") || test.wantPath && !strings.Contains(stderr.String(), missing) {
			t.Fatalf("source-free arguments=%q status=%d stdout=%q stderr=%q", test.arguments, status, stdout.String(), stderr.String())
		}
	}
	var stdout, stderr bytes.Buffer
	status := Run([]string{"--installation", manifest, "explore", "go-run", "."}, &stdout, &stderr)
	if status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "installation header") {
		t.Fatalf("installation-dependent status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestOperatorRejectsUnsupportedCommandsWithoutInstallation(t *testing.T) {
	t.Setenv("GOMAD_WASM_INSTALLATION", "/missing/manifest")
	for _, arguments := range [][]string{{"plan"}, {"minimize"}, {"explore", "--count=0", "go-run", "."}, {"discover", "extra"}, {"install", "--output=x", "--profile=unknown"}} {
		var stdout, stderr bytes.Buffer
		if status := Run(arguments, &stdout, &stderr); status != 2 || strings.Contains(stderr.String(), "/missing/manifest") {
			t.Fatalf("%q: status=%d stdout=%q stderr=%q", arguments, status, stdout.String(), stderr.String())
		}
	}
}

func TestUnifiedLauncherRoutesReportWithoutInstallation(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "missing-installation.json")
	result := filepath.Join(t.TempDir(), "missing-result.json")
	var stdout, stderr bytes.Buffer
	status := Run([]string{"--installation", manifest, "report", "--result", result}, &stdout, &stderr)
	if status != 3 || stdout.Len() != 0 || !strings.Contains(stderr.String(), result) || strings.Contains(stderr.String(), manifest) || strings.Contains(stderr.String(), "unknown gomad command") {
		t.Fatalf("source-free report status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestUnifiedLauncherHistoricalChoicesWithoutInstallation(t *testing.T) {
	t.Setenv("GOMAD_WASM_INSTALLATION", "")
	missing := filepath.Join(t.TempDir(), "historical")
	var stdout, stderr bytes.Buffer
	status := Run([]string{"inspect", "--choices", missing}, &stdout, &stderr)
	if status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "inspect "+missing) || strings.Contains(stderr.String(), "flag provided but not defined") || strings.Contains(stderr.String(), "native backend not built in") {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestUnifiedLauncherRoutesUnambiguousWASMCommandsBeforeInstallation(t *testing.T) {
	t.Setenv("GOMAD_WASM_INSTALLATION", "")
	missing := filepath.Join(t.TempDir(), "missing-installation.json")
	for _, prefix := range [][]string{nil, {"--installation", missing}} {
		for _, test := range []struct {
			arguments []string
			status    int
			want      string
		}{
			{[]string{"discover"}, 3, "WASM installation"},
			{[]string{"discover", "extra"}, 2, "discover accepts no command arguments"},
			{[]string{"install"}, 2, "install requires --output FILE"},
			{[]string{"install", "--unknown"}, 2, "flag provided but not defined: -unknown"},
		} {
			var stdout, stderr bytes.Buffer
			status := Run(append(append([]string(nil), prefix...), test.arguments...), &stdout, &stderr)
			if status != test.status || stdout.Len() != 0 || !strings.Contains(stderr.String(), test.want) || strings.Contains(stderr.String(), "native backend not built in") {
				t.Errorf("prefix=%v arguments=%v status=%d stdout=%q stderr=%q", prefix, test.arguments, status, stdout.String(), stderr.String())
			}
		}
	}
}

func TestOperatorMinimizeUsesSharedValidationAndInstallation(t *testing.T) {
	manifest := "/missing/minimize-installation"
	t.Setenv("GOMAD_WASM_INSTALLATION", manifest)
	for _, test := range []struct {
		arguments []string
		status    int
		want      string
	}{
		{[]string{"minimize"}, 2, "usage:"},
		{[]string{"minimize", "--attempt-budget=0", "/failure"}, 2, "usage:"},
		{[]string{"minimize", "--max-bytes=0", "/failure"}, 2, "invalid byte size"},
		{[]string{"minimize", "--toolchain-root=/native", "/failure"}, 2, "flag provided but not defined"},
		{[]string{"minimize", "--toolchain-root=", "/failure"}, 2, "flag provided but not defined"},
		{[]string{"minimize", "--resume", "--attempt-budget=4", "--max-bytes=2MiB", "/failure"}, 3, manifest},
	} {
		t.Run(strings.Join(test.arguments, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status := Run(test.arguments, &stdout, &stderr)
			if status != test.status || stdout.Len() != 0 || !strings.Contains(stderr.String(), test.want) || test.status == 2 && strings.Contains(stderr.String(), manifest) {
				t.Fatalf("status=%d stdout=%q stderr=%q; want status=%d %q", status, stdout.String(), stderr.String(), test.status, test.want)
			}
		})
	}
}

func TestOperatorUsageIncludesCompatibleMinimization(t *testing.T) {
	t.Setenv("GOMAD_WASM_INSTALLATION", "/missing/manifest")
	var stdout, stderr bytes.Buffer
	if status := Run(nil, &stdout, &stderr); status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "minimize") {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}
