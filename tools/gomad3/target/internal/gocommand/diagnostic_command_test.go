package gocommand

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/hostexec"
)

func TestDiagnosticCommandCapturesOneOrderedBoundedStream(t *testing.T) {
	for _, test := range []struct {
		name, body, output string
		exit               int
	}{
		{name: "ordered success", body: "printf 'stderr first\\n' >&2; printf 'stdout second\\n'", output: "stderr first\nstdout second\n"},
		{name: "ordered nonzero", body: "printf 'stderr first\\n' >&2; printf 'stdout second\\n'; exit 7", output: "stderr first\nstdout second\n", exit: 7},
		{name: "ordered signal", body: "printf 'stderr first\\n' >&2; printf 'stdout second\\n'; kill -KILL $$", output: "stderr first\nstdout second\n", exit: -1},
		{name: "long diagnostic", body: "printf HEAD >&2; printf '%0100d' 0; printf TAIL >&2; exit 7", output: "HEAD" + strings.Repeat("0", 100) + "TAIL", exit: 7},
		{name: "combined exact capacity", body: "printf '%032d' 0 >&2; printf '%032d' 0", output: strings.Repeat("0", 64)},
		{name: "combined limit plus one", body: "printf '%032d' 0 >&2; printf '%033d' 0", output: strings.Repeat("0", 65)},
	} {
		t.Run(test.name, func(t *testing.T) {
			var raw hostexec.Result
			runner := New(func(ctx context.Context, request hostexec.Request) (hostexec.Result, error) {
				if !request.PreserveCommandError || !request.CombinedOutput || request.Timeout != 15*time.Minute || request.TerminateGrace != 100*time.Millisecond || request.OutputLimit != 64 {
					t.Fatalf("diagnostic policy = %#v", request)
				}
				var err error
				raw, err = hostexec.Run(ctx, request)
				return raw, err
			})
			result, err := runner.DiagnosticCommand(t.Context(), Request{Command: []string{shell(t, test.body)}, Dir: t.TempDir(), OutputLimit: 64})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(result.Output, raw.Stdout) || raw.Stderr.TotalBytes != 0 || len(raw.Stderr.Bytes) != 0 || !raw.GroupGone || result.Output.TotalBytes != uint64(len(test.output)) || result.Output.FullSHA256 != sha256.Sum256([]byte(test.output)) || result.Output.Truncated != (len(test.output) > 64) {
				t.Fatalf("ordered capture = %#v, raw=%#v", result, raw)
			}
			if len(test.output) <= 64 {
				if string(result.Output.Bytes) != test.output || string(result.Output.RawBytes) != test.output {
					t.Fatalf("ordinary diagnostic bytes = %q", result.Output.Bytes)
				}
			} else {
				rawBytes := test.output[:48] + test.output[len(test.output)-16:]
				diagnostic := test.output[:48] + fmt.Sprintf("\n--- gomad3 output truncated: %d bytes discarded ---\n", len(test.output)-64) + test.output[len(test.output)-16:]
				if string(result.Output.RawBytes) != rawBytes || string(result.Output.Bytes) != diagnostic || result.Output.RetainedBytes != 64 || result.Output.DiscardedBytes != uint64(len(test.output)-64) || result.Output.RetainedSHA256 != sha256.Sum256([]byte(diagnostic)) {
					t.Fatalf("bounded head/tail and annotation = %#v", result.Output)
				}
			}
			var exit *exec.ExitError
			if test.exit != 0 {
				if !errors.As(result.CommandError, &exit) || exit.ProcessState == nil || exit.ExitCode() != test.exit || result.CommandError != raw.CommandError || exit.Stderr != nil {
					t.Fatalf("actual raw outcome = %T %v", result.CommandError, result.CommandError)
				}
			} else if result.CommandError != nil {
				t.Fatal(result.CommandError)
			}
		})
	}
}

func TestDiagnosticCommandLegacyInjectionAndOperationPrecedence(t *testing.T) {
	infra := errors.New("infrastructure")
	raw := errors.New("raw outcome")
	for _, test := range []struct {
		name      string
		result    hostexec.Result
		operation error
		want      error
		watchdog  bool
		cancel    bool
	}{
		{name: "legacy compiler", result: hostexec.Result{Termination: hostexec.TerminationExit, ExitCode: 7, Stderr: hostexec.Output{Bytes: []byte("compiler failed")}}},
		{name: "legacy cancellation", result: hostexec.Result{Cancelled: true}, want: context.Canceled},
		{name: "legacy watchdog", result: hostexec.Result{WatchdogTimeout: true}, want: context.DeadlineExceeded, watchdog: true},
		{name: "raw outcome before synthetic flags", result: hostexec.Result{Cancelled: true, CommandError: raw}, want: raw},
		{name: "completed late cancellation", result: hostexec.Result{Termination: hostexec.TerminationExit}, cancel: true},
		{name: "infrastructure before watchdog", result: hostexec.Result{WatchdogTimeout: true, CommandError: raw}, operation: infra, want: infra},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if test.cancel {
				cancel()
			}
			runner := New(func(context.Context, hostexec.Request) (hostexec.Result, error) { return test.result, test.operation })
			result, err := runner.DiagnosticCommand(ctx, Request{Command: []string{"go", "build"}, Dir: "fixture", OutputLimit: 64})
			if test.name == "legacy compiler" {
				var exit *ExitError
				if err != nil || !errors.As(result.CommandError, &exit) || exit.Code != 7 || string(result.Output.Bytes) != "compiler failed" || result.Output.FullSHA256 != ([32]byte{}) {
					t.Fatalf("legacy injection = %#v, %v", result, err)
				}
				return
			}
			var watchdog *WatchdogError
			if test.operation != nil || test.watchdog {
				if !errors.Is(err, test.want) || errors.As(err, &watchdog) != test.watchdog {
					t.Fatalf("operation precedence = %#v, %v", result, err)
				}
			} else if err != nil || result.CommandError != test.want {
				t.Fatalf("command outcome = %#v, %v", result, err)
			}
		})
	}
}

func TestDiagnosticCommandWatchdogKeepsRealRawOutcomeAndCapture(t *testing.T) {
	result, err := Default().DiagnosticCommand(t.Context(), Request{Command: []string{shell(t, "printf partial >&2; exec /bin/sleep 30")}, Dir: t.TempDir(), Timeout: 100 * time.Millisecond, OutputLimit: 64})
	var watchdog *WatchdogError
	var exit *exec.ExitError
	if !errors.As(err, &watchdog) || !errors.As(watchdog, &exit) || exit.ProcessState == nil || result.CommandError != watchdog.Cause || string(result.Output.Bytes) != "partial" || result.Output.FullSHA256 != sha256.Sum256([]byte("partial")) {
		t.Fatalf("watchdog capture/outcome = %#v, %T %v", result, err, err)
	}
}
