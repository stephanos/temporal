package cli

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/runner"
)

func TestReplayOutputVerifyOnly(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(map[bool]string{false: "healthy", true: "EBADF"}[failed], func(t *testing.T) {
			stdout := terminalDiagnostics(t, map[int]bool{1: failed})
			stderr := terminalDiagnostics(t, nil)
			installation := newFakeInstallation()
			calls := 0
			var observed runner.ReplaySpec
			dependencies := installation.replayDependencies(func(ctx context.Context, spec runner.ReplaySpec) (runner.ReplayResult, error) {
				calls++
				observed = spec
				if ctx != context.Background() || len(stdout.attempts) != 0 || len(stderr.attempts) != 0 || !reflect.DeepEqual(*installation.requested, []string{"/bundle"}) {
					t.Fatal("replay callback did not follow installation and precede reporting")
				}
				return runner.ReplayResult{Artifact: artifact.Artifact{Path: "/verified artifact/%s"}, Verified: true}, nil
			})
			status := runReplayWith([]string{"--toolchain-root=/bundle", "--observed=/observed", "--verify-only", "/requested-artifact"}, stdout, stderr, dependencies)
			want := runner.ReplaySpec{ArtifactPath: "/requested-artifact", VerifyOnly: true, ToolchainRoot: "/toolchain", ObservedDir: "/observed", SupervisorCommand: []string{"/bin/gomad", "__supervisor"}}
			if calls != 1 || !reflect.DeepEqual(observed, want) {
				t.Fatalf("replay calls = %d, request = %#v, want %#v", calls, observed, want)
			}
			checkTerminalDiagnostics(t, stdout, []string{"gomad: verified /verified artifact/%s\n"})
			checkTerminalDiagnostics(t, stderr, nil)
			wantStatus := 0
			if failed {
				wantStatus = 3
			}
			if status != wantStatus {
				t.Fatalf("status = %d, want %d after completed verification", status, wantStatus)
			}
		})
	}
}

func TestReplayOutputEarlierErrors(t *testing.T) {
	for _, test := range []struct {
		name       string
		installErr error
		replayErr  error
		status     int
		calls      int
		diagnostic string
	}{
		{"installation", errFakeInstallation, nil, 3, 0, "resolve Gomad installation: fake installation is unavailable\n"},
		{"preflight", nil, &runner.ReplayPreflightError{Err: errors.New("toolchain changed")}, 2, 1, "incompatible replay artifact: toolchain changed\n"},
		{"replay", nil, errors.New("supervisor failed"), 3, 1, "supervisor failed\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			stdout := terminalDiagnostics(t, map[int]bool{1: true})
			stderr := terminalDiagnostics(t, nil)
			installation := newFakeInstallation().failing(test.installErr)
			calls := 0
			dependencies := installation.replayDependencies(func(_ context.Context, spec runner.ReplaySpec) (runner.ReplayResult, error) {
				calls++
				want := runner.ReplaySpec{ArtifactPath: "/artifact", VerifyOnly: true, ToolchainRoot: "/toolchain", SupervisorCommand: []string{"/bin/gomad", "__supervisor"}}
				if !reflect.DeepEqual(spec, want) || len(stdout.attempts) != 0 || len(stderr.attempts) != 0 {
					t.Fatalf("replay request = %#v, want %#v before reporting", spec, want)
				}
				return runner.ReplayResult{}, test.replayErr
			})
			status := runReplayWith([]string{"--toolchain-root=/bundle", "--verify-only", "/artifact"}, stdout, stderr, dependencies)
			if status != test.status || calls != test.calls || !reflect.DeepEqual(*installation.requested, []string{"/bundle"}) {
				t.Fatalf("status = %d, calls = %d, installation requests = %q", status, calls, *installation.requested)
			}
			checkTerminalDiagnostics(t, stdout, nil)
			checkTerminalDiagnostics(t, stderr, []string{test.diagnostic})
		})
	}
}
