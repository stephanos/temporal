package gocommand

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/hostexec"
)

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
