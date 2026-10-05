package gocommand

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/hostexec"
)

type failingStdin struct{ err error }

func (reader failingStdin) Read([]byte) (int, error) { return 0, reader.err }

func TestCompatibilityInfrastructureFailurePreservesBoundedStderr(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	failure := errors.New("command cleanup failed")
	commandErr := errors.New("raw command outcome")
	runner := New(func(context.Context, hostexec.Request) (hostexec.Result, error) {
		return hostexec.Result{
			Cancelled: true, WatchdogTimeout: true, CommandError: commandErr,
			Stdout: hostexec.Output{Truncated: true},
			Stderr: hostexec.Output{RawBytes: []byte("diagnostic")},
		}, failure
	})
	result, err := runner.Compatibility(ctx, Request{Command: []string{"go"}, OutputLimit: 16})
	if err != failure || len(result.Stdout) != 0 || string(result.Stderr) != "diagnostic" {
		t.Fatalf("infrastructure result = %#v, %v; want original failure and bounded stderr", result, err)
	}
}

func TestCompatibilityStdinFailureWinsOverRealOverflow(t *testing.T) {
	for _, failure := range []error{errors.New("stdin read failed"), context.Canceled} {
		t.Run(failure.Error(), func(t *testing.T) {
			var raw hostexec.Result
			var infrastructure error
			runner := New(func(ctx context.Context, request hostexec.Request) (hostexec.Result, error) {
				request.Stdin = failingStdin{err: failure}
				raw, infrastructure = hostexec.Run(ctx, request)
				return raw, infrastructure
			})
			command := shell(t, "read value; printf '{\"valid\":true,\"large\":true}'; printf 'diagnostic' >&2; exit 0")
			result, err := runner.Compatibility(context.Background(), Request{Command: []string{command}, OutputLimit: 16})
			if infrastructure != failure || err != failure || raw.CommandError != nil {
				t.Fatalf("stdin failure = infrastructure %v, command %v, projected %v; want original infrastructure cause %v", infrastructure, raw.CommandError, err, failure)
			}
			if !raw.Stdout.Truncated || raw.ExitCode != 0 || !raw.GroupGone || len(result.Stdout) != 0 || string(result.Stderr) != "diagnostic" {
				t.Fatalf("stdin failure output = %#v, raw = %#v", result, raw)
			}
		})
	}
}

type blockedStdin struct {
	entered chan<- struct{}
	release <-chan struct{}
}

func (reader blockedStdin) Read([]byte) (int, error) {
	close(reader.entered)
	<-reader.release
	return 0, io.EOF
}

func TestCompatibilityCompletedExitSurvivesWatchdogWhileStdinCopyIsBlocked(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "leader.pid")
	command := shell(t, "printf '%s' \"$$\" > \"$1\"; printf 'diagnostic' >&2; exit 7")
	entered := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseCopy := func() { releaseOnce.Do(func() { close(release) }) }
	defer releaseCopy()
	var raw hostexec.Result
	var projected StructuredResult
	var outcome error
	runner := New(func(ctx context.Context, request hostexec.Request) (hostexec.Result, error) {
		request.Stdin = blockedStdin{entered: entered, release: release}
		var err error
		raw, err = hostexec.Run(ctx, request)
		return raw, err
	})
	const watchdog = 100 * time.Millisecond
	done := make(chan struct{})
	go func() {
		defer close(done)
		projected, outcome = runner.Compatibility(context.Background(), Request{Command: []string{command, marker}, Timeout: watchdog, OutputLimit: 128})
	}()
	limit := time.NewTimer(3 * time.Second)
	defer limit.Stop()
	select {
	case <-entered:
	case <-limit.C:
		t.Fatal("stdin copy did not enter")
	}
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	var pid string
	for pid == "" || processAlive(pid) {
		if data, err := os.ReadFile(marker); err == nil {
			pid = string(data)
		}
		select {
		case <-ticker.C:
		case <-limit.C:
			t.Fatal("leader was not reaped while stdin copy was blocked")
		}
	}
	timer := time.NewTimer(2 * watchdog)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-done:
		t.Fatal("command returned before blocked stdin copy was released")
	}
	releaseCopy()
	select {
	case <-done:
	case <-limit.C:
		t.Fatal("command did not return after stdin copy was released")
	}
	var exit *exec.ExitError
	if raw.WatchdogTimeout || outcome != raw.CommandError || !errors.As(outcome, &exit) || exit.ExitCode() != 7 || exit.Pid() != raw.PID {
		t.Fatalf("completed leader %s = raw %#v, projected %T %v; want original exit 7 without watchdog", pid, raw, outcome, outcome)
	}
	if len(projected.Stdout) != 0 || string(projected.Stderr) != "diagnostic" || !raw.GroupGone {
		t.Fatalf("completed output = %#v, raw = %#v", projected, raw)
	}
}

func TestCompatibilityPreservesRawExitAndRejectsFailedStdout(t *testing.T) {
	command := shell(t, "printf '{\"valid\":true}'; printf 'diagnostic' >&2; exit 7")
	result, err := Default().Compatibility(context.Background(), Request{Command: []string{command}, Dir: t.TempDir(), OutputLimit: 1024})
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 7 || len(exit.Stderr) != 0 {
		t.Fatalf("compatibility error = %T %v, want actual exit 7", err, err)
	}
	if len(result.Stdout) != 0 || string(result.Stderr) != "diagnostic" {
		t.Fatalf("compatibility result = %#v", result)
	}
}

func TestCompatibilityPreservesLegacyStartupPrecedence(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Default().Compatibility(ctx, Request{Command: []string{"fn109-absent-command"}, OutputLimit: 1024})
	var lookup *exec.Error
	if !errors.As(err, &lookup) || !errors.Is(err, exec.ErrNotFound) || errors.Is(err, context.Canceled) {
		t.Fatalf("compatibility missing PATH command = %T %v", err, err)
	}
}

func TestCompatibilityStartupErrorsAndDirectories(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "missing")
	nonexec := filepath.Join(root, "nonexec")
	invalid := filepath.Join(root, "invalid")
	for path, mode := range map[string]os.FileMode{nonexec: 0o600, invalid: 0o700} {
		if err := os.WriteFile(path, []byte("invalid executable\n"), mode); err != nil {
			t.Fatal(err)
		}
	}
	for _, test := range []struct {
		name, command, dir, op, path string
		cause                        error
		text                         string
	}{
		{name: "missing", command: missing, op: "fork/exec", path: missing, cause: syscall.ENOENT},
		{name: "permission", command: nonexec, op: "fork/exec", path: nonexec, cause: syscall.EACCES},
		{name: "format", command: invalid, op: "fork/exec", path: invalid, cause: syscall.ENOEXEC},
		{name: "directory", command: "/bin/sh", dir: missing, op: "chdir", path: missing, cause: syscall.ENOENT},
		{name: "empty", text: "exec: no command"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := Default().Compatibility(context.Background(), Request{Command: []string{test.command}, Dir: test.dir, OutputLimit: 128})
			if test.text != "" {
				if err == nil || err.Error() != test.text {
					t.Fatalf("error = %v", err)
				}
				return
			}
			var pathErr *os.PathError
			if !errors.As(err, &pathErr) || pathErr.Op != test.op || pathErr.Path != test.path || !errors.Is(err, test.cause) || err != pathErr {
				t.Fatalf("startup error = %T %v, want original %s %s %v", err, err, test.op, test.path, test.cause)
			}
		})
	}
	for _, expired := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		want := context.Canceled
		if expired {
			ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			defer cancel()
			want = context.DeadlineExceeded
		}
		for _, dir := range []string{"", missing} {
			for _, command := range []string{missing, "fn109-absent-command", ""} {
				_, err := Default().Compatibility(ctx, Request{Command: []string{command}, Dir: dir, OutputLimit: 128})
				if command == missing && err != want || command == "fn109-absent-command" && !errors.Is(err, exec.ErrNotFound) || command == "" && (err == nil || err.Error() != "exec: no command") {
					t.Fatalf("pre-start command %s dir %s = %v, context = %v", command, dir, err, want)
				}
			}
		}
	}
	command := shell(t, "printf '%s' \"$PWD\"")
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(cwd, filepath.Dir(command))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ command, dir, want string }{
		{command, "", cwd}, {command, ".", cwd},
		{"./go", filepath.Dir(command), filepath.Dir(command)}, {"./go", relative, filepath.Dir(command)},
	} {
		result, err := Default().Compatibility(context.Background(), Request{Command: []string{test.command}, Dir: test.dir, OutputLimit: 4096})
		if err != nil || string(result.Stdout) != test.want {
			t.Fatalf("cwd result = %#v, %v, want %s", result, err, test.want)
		}
	}
}

func TestCompatibilityBoundsRealStreamsBeforeRawOutcome(t *testing.T) {
	for _, test := range []struct {
		name, script, stream string
		stderr               string
	}{
		{"stdout-exit", "printf '{\"ok\":true}'; printf 'err' >&2; exit 7", "stdout", "err"},
		{"stderr-exit", "printf '{}'; printf 'long diagnostic' >&2; exit 7", "stderr", ""},
		{"both", "printf '{\"ok\":true}'; printf 'long diagnostic' >&2", "stdout", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := Default().Compatibility(context.Background(), Request{Command: []string{shell(t, test.script)}, OutputLimit: 8})
			var overflow *OverflowError
			if !errors.As(err, &overflow) || overflow.Stream != test.stream || len(result.Stdout) != 0 || string(result.Stderr) != test.stderr {
				t.Fatalf("bounded result = %#v, %v", result, err)
			}
		})
	}
}

func TestCompatibilityCallerLifetimeKillsLeaderBeforeDescendants(t *testing.T) {
	for _, mode := range []string{"cancel", "deadline", "watchdog", "cancel-overflow", "cancel-stderr-overflow"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			marker := filepath.Join(root, "ready")
			stdout := "{\"ok\":true}"
			if mode == "cancel-stderr-overflow" {
				stdout = "{}"
			}
			command := shell(t, "(trap '' TERM; exec sleep 1000) & child=$!; printf 'diagnostic' >&2; printf '"+stdout+"'; printf '%s' \"$child\" > \"$1\"; wait")
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			ack := make(chan string, 1)
			go func() {
				ticker := time.NewTicker(time.Millisecond)
				defer ticker.Stop()
				for {
					if data, err := os.ReadFile(marker); err == nil && len(data) > 0 {
						ack <- string(data)
						if mode == "cancel" || mode == "cancel-overflow" || mode == "cancel-stderr-overflow" {
							cancel()
						}
						return
					}
					select {
					case <-ctx.Done():
						ack <- ""
						return
					case <-ticker.C:
					}
				}
			}()
			var raw hostexec.Result
			runner := New(func(ctx context.Context, request hostexec.Request) (hostexec.Result, error) {
				var err error
				raw, err = hostexec.Run(ctx, request)
				return raw, err
			})
			request := Request{Command: []string{command, marker}, OutputLimit: 128}
			if mode == "watchdog" {
				request.Timeout = time.Second
			}
			if mode == "cancel-overflow" || mode == "cancel-stderr-overflow" {
				request.OutputLimit = 8
			}
			result, err := runner.Compatibility(ctx, request)
			pid := <-ack
			if pid == "" || processAlive(pid) || !raw.GroupGone || len(result.Stdout) != 0 {
				t.Fatalf("termination evidence = %#v, %v, child %s", raw, err, pid)
			}
			var exit *exec.ExitError
			if !errors.As(raw.CommandError, &exit) || exit.Pid() != raw.PID || exit.ExitCode() != -1 || exit.ProcessState.Sys().(syscall.WaitStatus).Signal() != syscall.SIGKILL {
				t.Fatalf("raw leader error = %T %v", raw.CommandError, raw.CommandError)
			}
			if mode == "cancel-overflow" || mode == "cancel-stderr-overflow" {
				var overflow *OverflowError
				stream := "stdout"
				if mode == "cancel-stderr-overflow" {
					stream = "stderr"
				}
				if !errors.As(err, &overflow) || overflow.Stream != stream {
					t.Fatalf("cancel overflow = %v", err)
				}
			} else {
				if string(result.Stderr) != "diagnostic" || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("caller result = %#v, %v", result, err)
				}
				if mode == "watchdog" {
					var watchdog *WatchdogError
					if !errors.As(err, &watchdog) || !raw.WatchdogTimeout || watchdog.Cause != raw.CommandError {
						t.Fatalf("watchdog = %v", err)
					}
				} else if err != raw.CommandError || raw.WatchdogTimeout {
					t.Fatalf("caller cancellation replaced raw error: %v", err)
				}
			}
		})
	}
}

func TestCompatibilityKeepsCompletedOutcomeAndCleanupPrecedence(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	command := shell(t, "printf '{}'; kill -TERM $$")
	var raw error
	runner := New(func(ctx context.Context, request hostexec.Request) (hostexec.Result, error) {
		result, err := hostexec.Run(ctx, request)
		raw = result.CommandError
		cancel()
		return result, err
	})
	result, err := runner.Compatibility(ctx, Request{Command: []string{command}, OutputLimit: 128})
	var exit *exec.ExitError
	if err != raw || !errors.As(err, &exit) || exit.ExitCode() != -1 || exit.ProcessState.Sys().(syscall.WaitStatus).Signal() != syscall.SIGTERM || len(result.Stdout) != 0 {
		t.Fatalf("completed outcome = %#v, %v", result, err)
	}
	cleanup := fmt.Errorf("cleanup failed")
	broken := New(func(context.Context, hostexec.Request) (hostexec.Result, error) {
		return hostexec.Result{Cancelled: true, WatchdogTimeout: true, CommandError: raw,
			Stdout: hostexec.Output{Truncated: true}, Stderr: hostexec.Output{Truncated: true}}, cleanup
	})
	result, err = broken.Compatibility(ctx, Request{Command: []string{command}, OutputLimit: 8})
	if err != cleanup || len(result.Stdout) != 0 {
		t.Fatalf("cleanup precedence = %#v, %v", result, err)
	}
}

func TestCompatibilityUsesFiniteDefaultWatchdogAndPositiveCapacity(t *testing.T) {
	var received hostexec.Request
	runner := New(func(_ context.Context, request hostexec.Request) (hostexec.Result, error) {
		received = request
		return hostexec.Result{Stdout: hostexec.Output{RawBytes: []byte("{}")}}, nil
	})
	result, err := runner.Compatibility(context.Background(), Request{Command: []string{"go"}, OutputLimit: 17})
	if err != nil || string(result.Stdout) != "{}" || received.Timeout != 15*time.Minute || received.OutputLimit != 17 || !received.PreserveCommandError {
		t.Fatalf("compatibility defaults = %#v, %#v, %v", received, result, err)
	}
	_, err = Default().Compatibility(context.Background(), Request{Command: []string{"/bin/sh", "-c", "exit 0"}})
	if err == nil || err.Error() != "output limit must be positive" {
		t.Fatalf("zero capacity error = %v", err)
	}
}

func TestStructuredRejectsOverflowBeforeReturningData(t *testing.T) {
	command := shell(t, "printf '{\"valid\":true}'; printf 'x' >&2")
	_, err := Default().Structured(context.Background(), Request{Command: []string{command}, Dir: t.TempDir(), OutputLimit: 8})
	var overflow *OverflowError
	if !errors.As(err, &overflow) || overflow.Stream != "stdout" {
		t.Fatalf("Structured() error = %T %v, want stdout overflow", err, err)
	}
}

func TestDiagnosticBoundsLongOutputAndKeepsFullHashes(t *testing.T) {
	command := shell(t, "printf 'head'; printf '%0200d' 0; printf 'tail' >&2; exit 1")
	result, err := Default().Diagnostic(context.Background(), Request{Command: []string{command}, Dir: t.TempDir(), OutputLimit: 16})
	var exit *ExitError
	if !errors.As(err, &exit) || exit.Code != 1 {
		t.Fatalf("Diagnostic() error = %T %v, want exit status 1", err, err)
	}
	if !result.Stdout.Truncated || result.Stdout.TotalBytes != 204 || result.Stdout.FullSHA256 != sha256.Sum256([]byte("head"+strings.Repeat("0", 200))) {
		t.Fatalf("stdout capture = %#v", result.Stdout)
	}
	if result.Stderr.Truncated || string(result.Stderr.RawBytes) != "tail" {
		t.Fatalf("stderr capture = %#v", result.Stderr)
	}
}

func TestStructuredCancellationRemovesDescendant(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	command := shell(t, "sleep 1000 & child=$!; printf '%s' \"$child\" > \"$1\"; wait")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go func() {
		for {
			if _, err := os.Stat(pidFile); err == nil {
				cancel()
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Millisecond):
			}
		}
	}()
	_, err := Default().Structured(ctx, Request{Command: []string{command, pidFile}, Dir: t.TempDir(), OutputLimit: 1024})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Structured() error = %v, want cancellation", err)
	}
	data, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	if processAlive(string(data)) {
		t.Fatalf("descendant %s survived cancellation", data)
	}
}

func TestDiagnosticRetainsOutputOnWatchdogAndPropagatesCleanupFailure(t *testing.T) {
	request := Request{Command: []string{"go", "build"}, Dir: t.TempDir(), OutputLimit: 8}
	watchdog := New(func(context.Context, hostexec.Request) (hostexec.Result, error) {
		return hostexec.Result{WatchdogTimeout: true, Stdout: hostexec.Output{Bytes: []byte("partial")}}, nil
	})
	result, err := watchdog.Diagnostic(context.Background(), request)
	if !errors.Is(err, context.DeadlineExceeded) || string(result.Stdout.Bytes) != "partial" {
		t.Fatalf("Diagnostic() = %#v, %v", result, err)
	}
	cleanup := errors.New("process group cleanup failed")
	broken := New(func(context.Context, hostexec.Request) (hostexec.Result, error) {
		return hostexec.Result{}, cleanup
	})
	_, err = broken.Diagnostic(context.Background(), request)
	if !errors.Is(err, cleanup) || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Diagnostic() error = %v, want cleanup failure", err)
	}
}

func shell(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "go")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func processAlive(text string) bool {
	pid, err := strconv.Atoi(text)
	if err != nil {
		return true
	}
	return !errors.Is(syscall.Kill(pid, 0), syscall.ESRCH)
}
