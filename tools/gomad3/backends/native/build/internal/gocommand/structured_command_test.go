package gocommand

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"reflect"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/hostexec"
)

func TestStructuredCommandSeparatesOperationAndCommandOutcomes(t *testing.T) {
	raw := errors.New("raw command outcome")
	infrastructure := errors.New("command infrastructure")
	for _, test := range []struct {
		name                     string
		cancel, stdout, stderr   bool
		watchdog                 bool
		operation                error
		command                  error
		wantOperation            error
		wantStream               string
		wantWatchdog, wantOutput bool
	}{
		{name: "success", wantOutput: true},
		{name: "command", command: raw, wantOutput: true},
		{name: "caller cancellation", cancel: true, command: raw, wantOutput: true},
		{name: "stdout overflow", stdout: true, command: raw, wantStream: "stdout"},
		{name: "stderr overflow", stderr: true, command: raw, wantStream: "stderr"},
		{name: "dual overflow", stdout: true, stderr: true, command: raw, wantStream: "stdout"},
		{name: "infrastructure before overflow", stdout: true, operation: infrastructure, command: raw, wantOperation: infrastructure},
		{name: "watchdog", watchdog: true, command: raw, wantWatchdog: true},
		{name: "overflow before watchdog", stdout: true, watchdog: true, command: raw, wantStream: "stdout"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if test.cancel {
				cancel()
			}
			request := Request{Command: []string{"go", "mod", "download", "-json", "example.com/m@v1.0.0"}, Dir: "outside", Env: []string{"CGO_ENABLED=0"}, OutputLimit: 64}
			runner := New(func(actual context.Context, got hostexec.Request) (hostexec.Result, error) {
				want := hostexec.Request{Command: request.Command, Dir: request.Dir, Env: request.Env, Timeout: 15 * time.Minute, TerminateGrace: 100 * time.Millisecond, OutputLimit: request.OutputLimit, PreserveCommandError: true}
				if actual != ctx || !reflect.DeepEqual(got, want) {
					t.Fatalf("request = %#v, want %#v", got, want)
				}
				return hostexec.Result{Stdout: hostexec.Output{RawBytes: []byte(`{"Error":"module refused"}`), Truncated: test.stdout}, Stderr: hostexec.Output{RawBytes: []byte("diagnostic"), Truncated: test.stderr}, CommandError: test.command, WatchdogTimeout: test.watchdog, Cancelled: test.cancel}, test.operation
			})
			result, err := runner.StructuredCommand(ctx, request)
			var overflow *OverflowError
			var watchdog *WatchdogError
			if test.wantOutput {
				want := StructuredCommandResult{Stdout: []byte(`{"Error":"module refused"}`), Stderr: []byte("diagnostic"), CommandError: test.command}
				if err != nil || !reflect.DeepEqual(result, want) {
					t.Fatalf("complete outcome = %#v, %v, want %#v", result, err, want)
				}
			} else {
				if err == nil || result.Stdout != nil || result.CommandError != nil || errors.Is(err, infrastructure) != (test.wantOperation != nil) || errors.As(err, &overflow) != (test.wantStream != "") || errors.As(err, &watchdog) != test.wantWatchdog {
					t.Fatalf("operation failure = %#v, %T %v", result, err, err)
				}
				if overflow != nil && (overflow.Stream != test.wantStream || overflow.Limit != request.OutputLimit) {
					t.Fatalf("overflow = %#v", overflow)
				}
				if watchdog != nil && (watchdog.Timeout != 15*time.Minute || !errors.Is(watchdog, raw)) {
					t.Fatalf("watchdog = %#v", watchdog)
				}
				wantStderr := []byte("diagnostic")
				if test.stderr {
					wantStderr = nil
				}
				if !reflect.DeepEqual(result.Stderr, wantStderr) {
					t.Fatalf("stderr = %q, want %q", result.Stderr, wantStderr)
				}
			}
		})
	}
}

func TestStructuredCommandKeepsActualExitAndRejectsPartialStreams(t *testing.T) {
	for _, test := range []struct {
		name, body, stream string
		wantExit           int
	}{
		{name: "complete nonzero", body: "printf '{\"Error\":\"module refused\"}'; printf diagnostic >&2; exit 7", wantExit: 7},
		{name: "stdout exact capacity", body: "printf '%064d' 0"},
		{name: "stderr exact capacity", body: "printf '%064d' 0 >&2"},
		{name: "stdout limit plus one", body: "printf '%065d' 0; exit 7", stream: "stdout"},
		{name: "stderr limit plus one", body: "printf '%065d' 0 >&2; exit 7", stream: "stderr"},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := Default().StructuredCommand(t.Context(), Request{Command: []string{shell(t, test.body)}, Dir: t.TempDir(), OutputLimit: 64})
			var overflow *OverflowError
			if test.stream != "" {
				if !errors.As(err, &overflow) || overflow.Stream != test.stream || result.Stdout != nil || result.CommandError != nil {
					t.Fatalf("partial streams exposed = %#v, %v", result, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var exit *exec.ExitError
			if test.wantExit != 0 {
				if !errors.As(result.CommandError, &exit) || exit.ProcessState == nil || exit.ExitCode() != test.wantExit || string(result.Stdout) != `{"Error":"module refused"}` || string(result.Stderr) != "diagnostic" {
					t.Fatalf("actual command outcome = %#v", result)
				}
			} else if result.CommandError != nil || len(result.Stdout)+len(result.Stderr) != 64 {
				t.Fatalf("exact capacity = %#v", result)
			}
		})
	}
}

func TestStructuredCommandOutputErrorKeepsIdentityAndDetachesStderr(t *testing.T) {
	result, err := Default().StructuredCommand(t.Context(), Request{Command: []string{shell(t, "printf diagnostic >&2; exit 7")}, Dir: t.TempDir(), OutputLimit: 64})
	if err != nil {
		t.Fatal(err)
	}
	exit, ok := result.CommandError.(*exec.ExitError)
	if !ok || exit.ProcessState == nil || exit.Stderr != nil {
		t.Fatalf("raw exit = %T %v", result.CommandError, result.CommandError)
	}
	state := exit.ProcessState
	if projected := result.OutputError(); projected != result.CommandError || string(exit.Stderr) != "diagnostic" || exit.ProcessState != state {
		t.Fatalf("projected exit = %T %v", projected, projected)
	}
	result.Stderr[0] = 'x'
	if string(exit.Stderr) != "diagnostic" {
		t.Fatal("projected stderr aliases captured bytes")
	}
	exit.Stderr = nil
	wrapped := fmt.Errorf("outer: %w", exit)
	result.CommandError = wrapped
	if projected := result.OutputError(); projected != wrapped || exit.Stderr != nil {
		t.Fatalf("projection traversed unrelated wrapper = %T %v", projected, projected)
	}
}

func TestStructuredCommandWatchdogRefusesCompleteCapturedStdout(t *testing.T) {
	result, err := Default().StructuredCommand(t.Context(), Request{
		Command: []string{shell(t, "printf '{\"Error\":\"module refused\"}'; printf diagnostic >&2; exec /bin/sleep 30")},
		Dir:     t.TempDir(), Timeout: 100 * time.Millisecond, OutputLimit: 64,
	})
	var watchdog *WatchdogError
	var exit *exec.ExitError
	if !errors.As(err, &watchdog) || watchdog.Timeout != 100*time.Millisecond || !errors.As(watchdog, &exit) || exit.ProcessState == nil || exit.ExitCode() != -1 || result.Stdout != nil || result.CommandError != nil || string(result.Stderr) != "diagnostic" {
		t.Fatalf("watchdog exposed structured data = %#v, %T %v", result, err, err)
	}
}

func TestStructuredCommandOutputErrorMatchesActualOutputStderr(t *testing.T) {
	for _, size := range []int{65535, 65536, 65537, 4 * 1024 * 1024} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			command := shell(t, fmt.Sprintf("printf '%%0%dd' 0 >&2; exit 7", size))
			_, original := exec.CommandContext(t.Context(), command).Output()
			originalExit, ok := original.(*exec.ExitError)
			if !ok || originalExit.ProcessState == nil {
				t.Fatalf("original command = %T %v", original, original)
			}
			result, err := Default().StructuredCommand(t.Context(), Request{Command: []string{command}, OutputLimit: 4 * 1024 * 1024})
			if err != nil {
				t.Fatal(err)
			}
			exit, ok := result.CommandError.(*exec.ExitError)
			if !ok || exit.ProcessState == nil || len(result.Stderr) != size {
				t.Fatalf("complete raw outcome = %T %v", result.CommandError, result.CommandError)
			}
			state := exit.ProcessState
			if projected := result.OutputError(); projected != exit || exit.ProcessState != state || !bytes.Equal(exit.Stderr, originalExit.Stderr) {
				t.Fatalf("stderr projection = %d bytes, want original %d", len(exit.Stderr), len(originalExit.Stderr))
			}
			result.Stderr[0] = 'x'
			if !bytes.Equal(exit.Stderr, originalExit.Stderr) {
				t.Fatal("stderr projection aliases captured input")
			}
		})
	}
}
