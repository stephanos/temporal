//go:build gomad_native

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
