package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeLauncherReportsLeadingFlagsAsUnknownCommands(t *testing.T) {
	for _, argument := range []string{"--not-a-global-flag", "-h", "--help"} {
		var stdout, stderr bytes.Buffer
		if status := Run([]string{argument}, &stdout, &stderr); status != 2 || stdout.Len() != 0 || stderr.String() != fmt.Sprintf("unknown gomad command %q\n%s", argument, usage) {
			t.Fatalf("%s: status=%d stdout=%q stderr=%q", argument, status, stdout.String(), stderr.String())
		}
		if status := Run([]string{argument}, &stdout, failingWriter{}); status != 2 || stdout.Len() != 0 {
			t.Fatalf("%s: failed diagnostic writer: status=%d stdout=%q", argument, status, stdout.String())
		}
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

func TestLauncherNativeInstallationPrecedence(t *testing.T) {
	base := t.TempDir()
	executable := filepath.Join(base, "bin", "gomad")
	if err := os.MkdirAll(filepath.Dir(executable), 0o755); err != nil {
		t.Fatal(err)
	}
	adjacent := filepath.Join(base, "bin", "gomad-install.json")
	explicit := filepath.Join(base, "explicit.json")
	cliRoot, environmentRoot := filepath.Join(base, "cli"), filepath.Join(base, "environment")
	write := func(path, contents string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(explicit, `{"schema":"gomad.installation/v1","toolchain_root":"explicit"}`)
	for _, test := range []struct {
		name, adjacent, environment string
		arguments                   []string
		want                        string
	}{
		{name: "adjacent", adjacent: `{"schema":"gomad.installation/v1","toolchain_root":"toolchain"}`, want: `"installation_source":"manifest","toolchain_root":"` + filepath.Join(base, "bin", "toolchain") + `"`},
		{name: "adjacent under environment", adjacent: `{"schema":"gomad.installation/v1","toolchain_root":"toolchain"}`, environment: environmentRoot, want: `"installation_source":"environment","toolchain_root":"` + environmentRoot + `"`},
		{name: "malformed adjacent under environment", adjacent: `{"schema":`, environment: environmentRoot, want: `"installation_source":"environment","toolchain_root":"` + environmentRoot + `"`},
		{name: "malformed adjacent under cli", adjacent: `{"schema":`, environment: environmentRoot, arguments: []string{"--toolchain-root=" + cliRoot}, want: `"installation_source":"cli","toolchain_root":"` + cliRoot + `"`},
		{name: "malformed adjacent", adjacent: `{"schema":`, want: "resolve Gomad installation: decode installation manifest: unexpected EOF\n"},
		{name: "backslash adjacent root", adjacent: `{"schema":"gomad.installation/v1","toolchain_root":"lib\\toolchain"}`, want: "resolve Gomad installation: installation manifest identity is invalid\n"},
		{name: "explicit", arguments: []string{"--installation", explicit}, want: `"installation_source":"manifest","toolchain_root":"` + filepath.Join(base, "explicit") + `"`},
		{name: "explicit under environment", arguments: []string{"--installation", explicit}, environment: environmentRoot, want: `"installation_source":"environment","toolchain_root":"` + environmentRoot + `"`},
		{name: "explicit under cli", arguments: []string{"--installation", explicit, "--toolchain-root=" + cliRoot}, environment: environmentRoot, want: `"installation_source":"cli","toolchain_root":"` + cliRoot + `"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := os.RemoveAll(adjacent); err != nil {
				t.Fatal(err)
			}
			if test.adjacent != "" {
				write(adjacent, test.adjacent)
			}
			app := hostApplication()
			app.executable = func() (string, error) { return executable, nil }
			app.environment = func(name string) string {
				if name == toolchainEnvironment {
					return test.environment
				}
				return ""
			}
			arguments := test.arguments
			if len(arguments) != 0 && arguments[0] == "--installation" {
				arguments = append(append([]string{}, arguments[:2]...), append([]string{"doctor", "--json", "--artifacts", filepath.Join(base, "artifacts")}, arguments[2:]...)...)
			} else {
				arguments = append([]string{"doctor", "--json", "--artifacts", filepath.Join(base, "artifacts")}, arguments...)
			}
			var stdout, stderr bytes.Buffer
			app.launch(arguments, &stdout, &stderr)
			if !strings.Contains(stdout.String()+stderr.String(), test.want) {
				t.Fatalf("stdout=%q stderr=%q; want %q", stdout.String(), stderr.String(), test.want)
			}
		})
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
