package process

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunnerCapturesExitAndBothStreams(t *testing.T) {
	result, err := (Runner{}).Run(context.Background(), helperRequest(t, "streams", 1<<20))
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 7 || string(result.Stdout) != "stdout" || string(result.Stderr) != "stderr" || result.Truncated {
		t.Fatalf("result = %#v", result)
	}
}

func TestRunnerStopsAtOutputLimit(t *testing.T) {
	result, err := (Runner{}).Run(context.Background(), helperRequest(t, "overflow", 128))
	if !errors.Is(err, ErrOutputLimit) || !result.Truncated || len(result.Stdout) != 128 {
		t.Fatalf("result = %#v, error = %v", result, err)
	}
}

func TestRunnerCancelsBlockedProcessWithoutSleep(t *testing.T) {
	marker := t.TempDir() + "/started"
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	request := helperRequest(t, "block", 1<<20)
	request.Command = append(request.Command, marker)
	go func() {
		_, err := (Runner{}).Run(ctx, request)
		done <- err
	}()
	waitForFile(t, marker)
	cancel()
	if err := <-done; !errors.Is(err, ErrCancelled) {
		t.Fatalf("Run() error = %v, want cancellation", err)
	}
}

func TestRunnerRejectsInvalidBoundsBeforeLaunch(t *testing.T) {
	request := helperRequest(t, "streams", 1)
	request.Timeout = 0
	if _, err := (Runner{}).Run(context.Background(), request); err == nil {
		t.Fatal("zero timeout was accepted")
	}
	request.Timeout = time.Minute
	request.Command = nil
	if _, err := (Runner{}).Run(context.Background(), request); err == nil {
		t.Fatal("empty command was accepted")
	}
}

func TestRunnerDistinguishesTimeoutFromLaunchFailure(t *testing.T) {
	request := helperRequest(t, "block", 1<<20)
	request.Command = append(request.Command, filepath.Join(t.TempDir(), "marker"))
	request.Timeout = time.Millisecond
	if _, err := (Runner{}).Run(context.Background(), request); !errors.Is(err, ErrTimeout) {
		t.Fatalf("timeout error = %v", err)
	}
	request = helperRequest(t, "streams", 1<<20)
	request.Command = []string{filepath.Join(t.TempDir(), "missing")}
	if _, err := (Runner{}).Run(context.Background(), request); err == nil {
		t.Fatal("missing executable was accepted")
	}
}

func TestRunnerUsesOnlyTheExplicitEnvironment(t *testing.T) {
	t.Setenv("AGENTWORKFLOW_UNDECLARED_SECRET", "secret")
	request := helperRequest(t, "environment", 1<<20)
	result, err := (Runner{}).Run(context.Background(), request)
	if err != nil || len(result.Stdout) != 0 {
		t.Fatalf("isolated result=%q error=%v", result.Stdout, err)
	}
	request.Environment = []string{"AGENTWORKFLOW_UNDECLARED_SECRET=allowed"}
	result, err = (Runner{}).Run(context.Background(), request)
	if err != nil || string(result.Stdout) != "allowed" {
		t.Fatalf("explicit result=%q error=%v", result.Stdout, err)
	}
}

//nolint:errcheck,revive // Writes and exits intentionally exercise subprocess supervision.
func TestProcessHelper(t *testing.T) {
	separator := index(os.Args, "--")
	if separator < 0 || separator+1 >= len(os.Args) {
		return
	}
	mode := os.Args[separator+1]
	switch mode {
	case "streams":
		fmt.Fprint(os.Stdout, "stdout")
		fmt.Fprint(os.Stderr, "stderr")
		os.Exit(7)
	case "overflow":
		fmt.Fprint(os.Stdout, strings.Repeat("x", 1<<20))
		os.Exit(0)
	case "block":
		if separator+2 >= len(os.Args) {
			os.Exit(2)
		}
		if err := os.WriteFile(os.Args[separator+2], []byte("ready"), 0o600); err != nil {
			os.Exit(3)
		}
		for {
			runtime.Gosched()
		}
	case "environment":
		fmt.Fprint(os.Stdout, os.Getenv("AGENTWORKFLOW_UNDECLARED_SECRET"))
		os.Exit(0)
	default:
		os.Exit(4)
	}
}

func helperRequest(t *testing.T, mode string, limit int64) Request {
	t.Helper()
	return Request{
		Command: []string{os.Args[0], "-test.run=TestProcessHelper", "--", mode}, Directory: t.TempDir(),
		Environment: SelectEnvironment("GOCOVERDIR"), Timeout: time.Minute, MaxOutputBytes: limit,
	}
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", path)
		}
		runtime.Gosched()
	}
}

func index(values []string, target string) int {
	for index, value := range values {
		if value == target {
			return index
		}
	}
	return -1
}
