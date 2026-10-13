package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/artifact"
	runnerbackend "go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/backends/native"
	wasmbackend "go.temporal.io/server/tools/gomad3/backends/wasm/backend"
	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
	"go.temporal.io/server/tools/gomad3/target"
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
			wantConfig := native.Config{ToolchainRoot: test.want, SupervisorCommand: []string{executable, supervisorMode}, CoordinatorCommand: []string{executable, coordinatorMode}, BootstrapCommand: []string{executable, targetBootstrapMode}, RunnerBuild: digest}
			if got := nativeConfiguration(installed); !reflect.DeepEqual(got, wantConfig) {
				t.Fatalf("native provider configuration = %#v, want %#v", got, wantConfig)
			}
		})
	}
}

func TestApplicationRetainsInstallationIdentityAndPrivateCommands(t *testing.T) {
	executable, root := writeInstalledExecutable(t, 0o700)
	app := fixedApplication(executable, nil)
	app.cache = new(installationCache)
	first, err := app.install(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(executable); err != nil {
		t.Fatal(err)
	}
	second, err := app.install(root)
	if err != nil || second != first {
		t.Fatalf("cached installation = %#v, %v; want %#v", second, err, first)
	}
	otherRoot := filepath.Join(t.TempDir(), "toolchain")
	third, err := app.install(otherRoot)
	if err != nil || third.toolchainRoot != otherRoot || third.executable != first.executable || third.runnerBuild != first.runnerBuild {
		t.Fatalf("installation with another root = %#v, %v; want cached executable and build with %q", third, err, otherRoot)
	}
	if got := strings.Join(nativeConfiguration(second).SupervisorCommand, " "); got != executable+" __supervisor" {
		t.Fatalf("supervisor command = %q", got)
	}
	if got := strings.Join(nativeConfiguration(second).CoordinatorCommand, " "); got != executable+" __coordinator" {
		t.Fatalf("coordinator command = %q", got)
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
	if got := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return app.run([]string{"doctor", "--unknown"}, stdout, stderr)
	}); got.status != 2 || calls != 1 || !strings.Contains(got.stderr, "flag provided but not defined: -unknown") {
		t.Fatalf("invalid doctor = %s calls=%d", got, calls)
	}
	if got, want := runCommand(func(stdout, stderr *bytes.Buffer) int {
		return app.run([]string{"doctor", "--json"}, stdout, stderr)
	}), (commandResult{status: 3, stderr: "resolve gomad executable: no executable\n"}); got != want || calls != 2 {
		t.Fatalf("valid doctor = %s calls=%d, want %s", got, calls, want)
	}
}

type commandBackend struct{}

func externalCapabilities() runnerbackend.Capabilities {
	return wasmbackend.CapabilitiesForProfile(wasi.CooperativeProfile)
}

func RunWithBackend(arguments []string, stdout, stderr io.Writer, resolve func() (BackendInstallation, error)) int {
	return RunWithBackendReplay(arguments, stdout, stderr, resolve, nil)
}

func RunWithBackendReplay(arguments []string, stdout, stderr io.Writer, resolve func() (BackendInstallation, error), replay func(context.Context, runner.ReplaySpec) (runner.ReplayResult, bool, error)) int {
	app := hostApplication()
	if resolve == nil {
		if _, err := fmt.Fprintln(stderr, "external backend resolver is required"); err != nil {
			return 3
		}
		return 2
	}
	app.backend, app.replayHook = resolve, replay
	capabilities := externalCapabilities()
	app.capabilities = &capabilities
	return app.run(arguments, stdout, stderr)
}

func (commandBackend) Capabilities() runnerbackend.Capabilities { return externalCapabilities() }
func (commandBackend) Describe(context.Context, runnerbackend.Configuration) (runnerbackend.Description, error) {
	return runnerbackend.Description{}, errors.New("unused")
}
func (commandBackend) Execute(context.Context, runnerbackend.ExecutionRequest) (runnerbackend.ExecutionResult, error) {
	return runnerbackend.ExecutionResult{}, errors.New("unused")
}
func (commandBackend) Evidence(context.Context, runnerbackend.EvidenceRequest) (runnerbackend.Evidence, error) {
	return runnerbackend.Evidence{}, errors.New("unused")
}

func (commandBackend) Prepare(context.Context, target.Spec) (target.Prepared, error) {
	return target.Prepared{}, errors.New("unused")
}

func TestRetainedBackendReplayUsesSharedGrammarAndClassificationWithoutInstallation(t *testing.T) {
	for _, test := range []struct {
		domain string
		match  bool
		status int
	}{
		{"success", true, 0},
		{"target", true, 1},
		{"success", false, 1},
	} {
		var stdout, stderr bytes.Buffer
		called := false
		status := RunWithBackendReplay([]string{"replay", "--observed=/observed", "/artifact"}, &stdout, &stderr, func() (BackendInstallation, error) {
			t.Fatal("retained replay resolved an installation")
			return BackendInstallation{}, nil
		}, func(_ context.Context, spec runner.ReplaySpec) (runner.ReplayResult, bool, error) {
			called = true
			if spec.ArtifactPath != "/artifact" || spec.ObservedDir != "/observed" || spec.Backend != nil || spec.ToolchainRoot != "" {
				t.Fatalf("retained replay changed shared arguments: %+v", spec)
			}
			return runner.ReplayResult{Match: test.match, Artifact: artifact.Artifact{Path: "/artifact", Manifest: record.ExecutionRecord{Outcome: record.Outcome{Domain: test.domain}}}}, true, nil
		})
		if !called || status != test.status || stderr.Len() != 0 || !strings.HasPrefix(stdout.String(), "gomad: reproduced=") {
			t.Fatalf("domain=%s match=%t status=%d called=%t stdout=%q stderr=%q", test.domain, test.match, status, called, stdout.String(), stderr.String())
		}
	}
}

func TestRetainedReplayHookPreservesSyntaxAndUnhandledInstallationPath(t *testing.T) {
	for _, arguments := range [][]string{{"replay"}, {"replay", "--unknown", "/artifact"}} {
		var stdout, stderr bytes.Buffer
		status := RunWithBackendReplay(arguments, &stdout, &stderr, func() (BackendInstallation, error) {
			t.Fatal("invalid syntax resolved installation")
			return BackendInstallation{}, nil
		}, func(context.Context, runner.ReplaySpec) (runner.ReplayResult, bool, error) {
			t.Fatal("invalid syntax reached retained hook")
			return runner.ReplayResult{}, false, nil
		})
		if status != 2 {
			t.Fatalf("arguments=%q status=%d", arguments, status)
		}
	}
	resolve := func() (BackendInstallation, error) {
		return BackendInstallation{}, errors.New("unavailable installed helper")
	}
	var originalOut, originalErr, hookedOut, hookedErr bytes.Buffer
	original := RunWithBackend([]string{"replay", "/single-guest"}, &originalOut, &originalErr, resolve)
	hooked := RunWithBackendReplay([]string{"replay", "/single-guest"}, &hookedOut, &hookedErr, resolve, func(context.Context, runner.ReplaySpec) (runner.ReplayResult, bool, error) {
		return runner.ReplayResult{}, false, nil
	})
	if original != 3 || hooked != original || originalOut.String() != hookedOut.String() || originalErr.String() != hookedErr.String() {
		t.Fatalf("unhandled changed normal replay: statuses=%d/%d stdout=%q/%q stderr=%q/%q", original, hooked, originalOut.String(), hookedOut.String(), originalErr.String(), hookedErr.String())
	}
}

func TestRetainedReplayHookUsesExistingPreflightAndInfrastructureErrors(t *testing.T) {
	for _, test := range []struct {
		err    error
		status int
	}{
		{&runner.ReplayPreflightError{Err: errors.New("unsupported retained replay option")}, 2},
		{errors.New("invalid retained helper evidence"), 3},
	} {
		var stdout, stderr bytes.Buffer
		status := RunWithBackendReplay([]string{"replay", "--verify-only", "/artifact"}, &stdout, &stderr, func() (BackendInstallation, error) {
			t.Fatal("handled error resolved installation")
			return BackendInstallation{}, nil
		}, func(_ context.Context, spec runner.ReplaySpec) (runner.ReplayResult, bool, error) {
			if !spec.VerifyOnly {
				t.Fatal("shared parser lost verify-only")
			}
			return runner.ReplayResult{}, true, test.err
		})
		if status != test.status || stdout.Len() != 0 || !strings.Contains(stderr.String(), test.err.Error()) {
			t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
		}
	}
}
func (commandBackend) ValidatePrepared(target.Spec, target.Prepared, []string) error {
	return errors.New("unused")
}
func (commandBackend) ValidateReplay(context.Context, *artifact.Opened) (target.Prepared, error) {
	return target.Prepared{}, errors.New("unused")
}
func (commandBackend) Run(context.Context, runnerbackend.Request) (runnerbackend.Result, error) {
	return runnerbackend.Result{}, errors.New("unused")
}

func TestExternalBackendUsesSharedCampaignGrammar(t *testing.T) {
	provider := commandBackend{}
	var stdout, stderr bytes.Buffer
	request, status, ok := parseCampaignRequest(exploreCampaign, []string{"--json", "--seeds=7", "--parallel=1", "--build-tag=test_dep", "go-test", "./pkg", "--", "-test.run=Test Name", "literal;$value"}, &stdout, &stderr, exploreDependencies{
		install: func(string) (installation, error) {
			return installation{backend: provider, backendName: "wasm", runnerBuild: "sha256:runner"}, nil
		},
		workingDirectory: func() (string, error) { return "/workspace", nil },
	})
	if !ok || status != 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	campaign := request.campaign
	if campaign.Backend != provider || campaign.Target.Backend != "wasm" || campaign.Target.ToolchainRoot != "" || len(campaign.Target.Args) != 2 || campaign.Target.Args[0] != "-test.run=Test Name" || campaign.Target.Args[1] != "literal;$value" {
		t.Fatalf("external campaign routing: %#v", campaign)
	}
}

func TestExternalBackendRejectsCommandsBeforeResolvingInstallation(t *testing.T) {
	for _, arguments := range [][]string{{"plan"}, {"analyze", "go-run", "./pkg"}, {"doctor"}, {"qualify"}, {"minimize"}, {supervisorMode}, {"explore", "--count=0", "go-run", "./pkg"}, {"replay"}, {"resume"}} {
		var stdout, stderr bytes.Buffer
		status := RunWithBackend(arguments, &stdout, &stderr, func() (BackendInstallation, error) {
			t.Fatal("resolved installation for unsupported command or invalid syntax")
			return BackendInstallation{}, nil
		})
		if status != 2 {
			t.Fatalf("%q: status=%d stdout=%q stderr=%q", arguments, status, stdout.String(), stderr.String())
		}
	}
}

func TestExternalBackendInstallationAvoidsNativeToolchainDiscovery(t *testing.T) {
	app := hostApplication()
	app.environment = func(string) string { t.Fatal("consulted native toolchain environment"); return "" }
	app.backend = func() (BackendInstallation, error) {
		return BackendInstallation{Name: "wasm", Provider: commandBackend{}}, nil
	}
	installed, err := app.install("")
	if err != nil || installed.backend == nil || installed.backendName != "wasm" || !strings.HasPrefix(installed.runnerBuild, "sha256:") || installed.toolchainRoot != "" {
		t.Fatalf("installation=%#v err=%v", installed, err)
	}
	if _, err := app.install("/native"); err == nil {
		t.Fatal("external backend silently accepted native toolchain selection")
	}
}

func TestExternalBackendMinimizeUsesSharedValidationBeforeResolution(t *testing.T) {
	for _, test := range []struct {
		arguments []string
		want      string
	}{
		{[]string{"minimize"}, "usage:"},
		{[]string{"minimize", "/failure", "extra"}, "usage:"},
		{[]string{"minimize", "--attempt-budget=0", "/failure"}, "usage:"},
		{[]string{"minimize", "--attempt-budget=18446744073709551616", "/failure"}, "invalid value"},
		{[]string{"minimize", "--max-bytes=0", "/failure"}, "invalid byte size"},
		{[]string{"minimize", "--unknown", "/failure"}, "flag provided but not defined"},
		{[]string{"minimize", "--toolchain-root=/native", "/failure"}, "flag provided but not defined"},
		{[]string{"minimize", "--toolchain-root=", "/failure"}, "flag provided but not defined"},
	} {
		t.Run(strings.Join(test.arguments, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status := RunWithBackend(test.arguments, &stdout, &stderr, func() (BackendInstallation, error) {
				t.Fatal("invalid minimization input resolved an installation")
				return BackendInstallation{}, nil
			})
			if status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), test.want) {
				t.Fatalf("status=%d stdout=%q stderr=%q; want %q", status, stdout.String(), stderr.String(), test.want)
			}
		})
	}
}

func TestExternalBackendMinimizeReachesInstallationAfterValidGrammar(t *testing.T) {
	var stdout, stderr bytes.Buffer
	resolved := false
	status := RunWithBackend([]string{"minimize", "--resume", "--attempt-budget=4", "--max-bytes=2MiB", "/failure"}, &stdout, &stderr, func() (BackendInstallation, error) {
		resolved = true
		return BackendInstallation{}, errors.New("selected external installation unavailable")
	})
	if status != 3 || !resolved || stdout.Len() != 0 || stderr.String() != "selected external installation unavailable\n" {
		t.Fatalf("status=%d resolved=%t stdout=%q stderr=%q", status, resolved, stdout.String(), stderr.String())
	}
}
