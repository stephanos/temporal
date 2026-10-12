package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
	runnerbackend "go.temporal.io/server/tools/gomad3/runner/backend"
	"go.temporal.io/server/tools/gomad3/target"
)

type commandBackend struct{}

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
			if spec.ArtifactPath != "/artifact" || spec.ObservedDir != "/observed" || spec.Backend != nil || spec.ToolchainRoot != "" || len(spec.SupervisorCommand) != 0 {
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
	if campaign.Backend != provider || campaign.Target.Backend != "wasm" || campaign.Target.ToolchainRoot != "" || len(campaign.CoordinatorCommand) != 0 || len(campaign.SupervisorCommand) != 0 || len(campaign.Target.Args) != 2 || campaign.Target.Args[0] != "-test.run=Test Name" || campaign.Target.Args[1] != "literal;$value" {
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
		{[]string{"minimize", "--toolchain-root=/native", "/failure"}, "--toolchain-root is unavailable"},
		{[]string{"minimize", "--toolchain-root=", "/failure"}, "--toolchain-root is unavailable"},
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
