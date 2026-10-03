package cli

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeInstalledExecutable(t *testing.T, mode os.FileMode) (string, string) {
	t.Helper()
	root := t.TempDir()
	for _, directory := range []string{filepath.Join(root, ".bin"), filepath.Join(root, ".toolchain")} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	executable := filepath.Join(root, ".bin", "gomad")
	if err := os.WriteFile(executable, []byte("runner"), mode); err != nil {
		t.Fatal(err)
	}
	return executable, filepath.Join(root, ".toolchain")
}

func fixedApplication(executable string, environment map[string]string) application {
	return application{
		executable:  func() (string, error) { return executable, nil },
		environment: func(name string) string { return environment[name] },
	}
}

func TestApplicationInstallResolvesToolchainRunnerBuildAndChildModes(t *testing.T) {
	executable, adjacent := writeInstalledExecutable(t, 0o700)
	digest := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte("runner")))
	for _, test := range []struct {
		name        string
		explicit    string
		environment map[string]string
		want        string
	}{
		{name: "adjacent", want: adjacent},
		{name: "environment", environment: map[string]string{toolchainEnvironment: "/environment/toolchain"}, want: "/environment/toolchain"},
		{name: "explicit before environment", explicit: "/explicit/toolchain", environment: map[string]string{toolchainEnvironment: "/environment/toolchain"}, want: "/explicit/toolchain"},
	} {
		t.Run(test.name, func(t *testing.T) {
			installed, err := fixedApplication(executable, test.environment).install(test.explicit)
			if err != nil {
				t.Fatal(err)
			}
			if want := (installation{toolchainRoot: test.want, executable: executable, runnerBuild: digest}); installed != want {
				t.Fatalf("installation = %#v, want %#v", installed, want)
			}
			if got := strings.Join(installed.supervisorCommand(), " "); got != executable+" __supervisor" {
				t.Fatalf("supervisor command = %q", got)
			}
			if got := strings.Join(installed.coordinatorCommand(), " "); got != executable+" __coordinator" {
				t.Fatalf("coordinator command = %q", got)
			}
		})
	}
}

func TestApplicationInstallFailsClosed(t *testing.T) {
	executable, _ := writeInstalledExecutable(t, 0o600)
	for _, test := range []struct {
		name string
		app  application
		want string
	}{
		{
			name: "executable unavailable",
			app:  application{executable: func() (string, error) { return "", errors.New("no executable") }, environment: func(string) string { return "" }},
			want: "resolve gomad executable: no executable",
		},
		{
			name: "invalid environment root",
			app:  fixedApplication(executable, map[string]string{toolchainEnvironment: "relative"}),
			want: `resolve Gomad installation: GOMAD3_TOOLCHAIN_DIR toolchain root must be an absolute non-root clean path: "relative"`,
		},
		{
			name: "runner not executable",
			app:  fixedApplication(executable, nil),
			want: "hash gomad executable: Runner is not a regular executable: " + executable,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := test.app.install(""); err == nil || err.Error() != test.want {
				t.Fatalf("install error = %v, want %s", err, test.want)
			}
		})
	}
}

func TestApplicationDispatchesPrivateModesOverProcessStreams(t *testing.T) {
	input, output := strings.NewReader("request"), new(bytes.Buffer)
	var dispatched []string
	app := application{privateInput: input, privateOutput: output, dispatch: func(mode string, in io.Reader, out io.Writer) error {
		if in != input || out != output {
			t.Fatalf("private mode %s streams = %v, %v", mode, in, out)
		}
		dispatched = append(dispatched, mode)
		if mode == coordinatorMode {
			return errors.New("decode coordinator request: EOF")
		}
		return nil
	}}
	for _, test := range []struct {
		mode   string
		result commandResult
	}{
		{mode: supervisorMode, result: commandResult{status: 0}},
		{mode: targetBootstrapMode, result: commandResult{status: 0}},
		{mode: coordinatorMode, result: commandResult{status: 3, stderr: "decode coordinator request: EOF\n"}},
	} {
		got := runCommand(func(stdout, stderr *bytes.Buffer) int { return app.run([]string{test.mode, "ignored"}, stdout, stderr) })
		if got != test.result {
			t.Fatalf("private mode %s = %s, want %s", test.mode, got, test.result)
		}
	}
	if strings.Join(dispatched, ",") != "__supervisor,__target_bootstrap,__coordinator" {
		t.Fatalf("dispatched = %q", dispatched)
	}
}

// TestApplicationResolvesOnlyAfterInputValidation pins that the construction
// path is consulted after a command validates its input, so an invalid
// request is never reported as an unusable installation.
func TestApplicationResolvesOnlyAfterInputValidation(t *testing.T) {
	calls := 0
	app := application{executable: func() (string, error) {
		calls++
		return "", errors.New("no executable")
	}, environment: func(string) string { return "" }}
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return app.run([]string{"replay"}, stdout, stderr)
	}), (commandResult{status: 2, stderr: usage}); got != want || calls != 0 {
		t.Fatalf("invalid replay = %s calls=%d", got, calls)
	}
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return app.run([]string{"replay", "/artifact"}, stdout, stderr)
	}), (commandResult{status: 3, stderr: "resolve gomad executable: no executable\n"}); got != want || calls != 1 {
		t.Fatalf("replay = %s calls=%d, want %s", got, calls, want)
	}
	if got := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return app.run([]string{"qualify-set", "--manifest=/missing.json", "--working-dir=/repo", "--check"}, stdout, stderr)
	}); got.status != 2 || calls != 1 || !strings.HasPrefix(got.stderr, "load qualification manifest: ") {
		t.Fatalf("qualify-set check = %s calls=%d", got, calls)
	}
	// Doctor names the executable before it parses its flags.
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return app.run([]string{"doctor", "--unknown"}, stdout, stderr)
	}), (commandResult{status: 3, stderr: "resolve gomad executable: no executable\n"}); got != want || calls != 2 {
		t.Fatalf("doctor = %s calls=%d, want %s", got, calls, want)
	}
}
