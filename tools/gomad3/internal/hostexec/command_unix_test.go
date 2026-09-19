//go:build unix

package hostexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestRunCapturesSuccessfulCommand(t *testing.T) {
	result, err := Run(context.Background(), testRequest("printf 'standard output'; printf 'standard error' >&2"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Termination != TerminationExit || result.ExitCode != 0 || result.WatchdogTimeout || result.Cancelled || !result.GroupGone {
		t.Fatalf("result = %#v", result)
	}
	if string(result.Stdout.Bytes) != "standard output" || string(result.Stderr.Bytes) != "standard error" {
		t.Fatalf("stdout = %q, stderr = %q", result.Stdout.Bytes, result.Stderr.Bytes)
	}
}

func TestRunSuppliesExplicitStandardInput(t *testing.T) {
	request := testRequest("read value; printf '%s' \"$value\"")
	request.Stdin = bytes.NewBufferString("input value\n")
	result, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if string(result.Stdout.Bytes) != "input value" {
		t.Fatalf("stdout = %q", result.Stdout.Bytes)
	}
}

func TestRunReturnsNonzeroExitAsData(t *testing.T) {
	result, err := Run(context.Background(), testRequest("exit 124"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Termination != TerminationExit || result.ExitCode != 124 || result.WatchdogTimeout {
		t.Fatalf("result = %#v", result)
	}
}

func TestRunClassifiesSignalExit(t *testing.T) {
	result, err := Run(context.Background(), testRequest("kill -TERM $$"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Termination != TerminationSignal || result.Signal != "terminated" || result.WatchdogTimeout {
		t.Fatalf("result = %#v", result)
	}
}

func TestRunTimesOutAndRemovesTermIgnoringDescendant(t *testing.T) {
	pidPath := filepath.Join(t.TempDir(), "descendant.pid")
	request := testRequest("trap '' TERM; sleep 30 & printf '%s\\n' $! >\"$1\"; wait")
	request.Command = append(request.Command, "sh", pidPath)
	request.Timeout = 150 * time.Millisecond
	request.TerminateGrace = 50 * time.Millisecond
	result, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !result.WatchdogTimeout || result.Cancelled || !result.GroupGone {
		t.Fatalf("result = %#v", result)
	}
	contents, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(contents)))
	if err != nil {
		t.Fatal(err)
	}
	requireProcessGone(t, pid)
}

func TestRunClassifiesContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	request := testRequest("sleep 30")
	request.Timeout = 5 * time.Second
	result, err := Run(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if result.WatchdogTimeout || !result.Cancelled || !result.GroupGone {
		t.Fatalf("result = %#v", result)
	}
}

func TestRunBoundsEachOutputStream(t *testing.T) {
	request := testRequest("printf 'abcdefghijklmnop'; printf 'ABCDEFGHIJKLMNOP' >&2")
	request.OutputLimit = 8
	result, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Stdout.Truncated || result.Stdout.RetainedBytes != 8 || result.Stdout.TotalBytes != 16 || !result.Stderr.Truncated || result.Stderr.RetainedBytes != 8 || result.Stderr.TotalBytes != 16 {
		t.Fatalf("stdout = %#v, stderr = %#v", result.Stdout, result.Stderr)
	}
}

func TestRunRejectsInvalidRequest(t *testing.T) {
	for name, mutate := range map[string]func(*Request){
		"command":           func(request *Request) { request.Command = nil },
		"working-directory": func(request *Request) { request.Dir = "" },
		"timeout":           func(request *Request) { request.Timeout = 0 },
		"termination-grace": func(request *Request) { request.TerminateGrace = -1 },
		"output-limit":      func(request *Request) { request.OutputLimit = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			request := testRequest("exit 0")
			mutate(&request)
			if _, err := Run(context.Background(), request); err == nil {
				t.Fatal("Run() accepted invalid request")
			}
		})
	}
}

func TestRunSupportsConcurrentCommands(t *testing.T) {
	const count = 8
	var group sync.WaitGroup
	errorsFound := make(chan error, count)
	for index := 0; index < count; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			result, err := Run(context.Background(), testRequest(fmt.Sprintf("printf %d", index)))
			if err == nil && string(result.Stdout.Bytes) != strconv.Itoa(index) {
				err = fmt.Errorf("stdout = %q, want %d", result.Stdout.Bytes, index)
			}
			errorsFound <- err
		}(index)
	}
	group.Wait()
	close(errorsFound)
	for err := range errorsFound {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func testRequest(script string) Request {
	return Request{
		Command:        []string{"/bin/sh", "-c", script},
		Dir:            "/",
		Env:            append(os.Environ(), "LC_ALL=C"),
		Timeout:        5 * time.Second,
		TerminateGrace: 100 * time.Millisecond,
		OutputLimit:    1 << 20,
	}
}

func requireProcessGone(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		err := syscall.Kill(pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("process %d remains after command cleanup", pid)
}
