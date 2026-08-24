package quality

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/agentworkflow/internal/workspace"
)

func TestChecksClassifyPassFailureAndOptionalResultsInOrder(t *testing.T) {
	candidate := t.TempDir()
	results, err := Run(context.Background(), candidate, []Check{
		{Name: "pass", Command: helperCommand("pass"), Required: true},
		{Name: "fail", Command: helperCommand("fail"), Required: false},
	}, testOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].Outcome != "passed" || results[1].Outcome != "failed" || results[1].ExitCode != 9 {
		t.Fatalf("results = %#v", results)
	}
}

func TestCheckDetectsUnexpectedCandidateMutation(t *testing.T) {
	candidate := t.TempDir()
	results, err := Run(context.Background(), candidate, []Check{{
		Name: "mutate", Command: helperCommand("mutate"), Required: true,
	}}, testOptions())
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Outcome != "mutated" || results[0].BeforeHash == results[0].AfterHash {
		t.Fatalf("result = %#v", results[0])
	}
}

func TestCheckClassifiesOutputCapacity(t *testing.T) {
	candidate := t.TempDir()
	options := testOptions()
	options.MaxOutputBytes = 32
	results, err := Run(context.Background(), candidate, []Check{{
		Name: "overflow", Command: helperCommand("overflow"), Required: true,
	}}, options)
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Outcome != "capacity-exhausted" || !results[0].Truncated {
		t.Fatalf("result = %#v", results[0])
	}
}

func TestCheckRejectsEscapingDirectoryBeforeLaunch(t *testing.T) {
	_, err := Run(context.Background(), t.TempDir(), []Check{{
		Name: "escape", Command: helperCommand("pass"), Directory: "../outside", Required: true,
	}}, testOptions())
	if err == nil {
		t.Fatal("escaping check directory was accepted")
	}
}

func TestChecksClassifyTimeoutAndLaunchFailure(t *testing.T) {
	candidate := t.TempDir()
	results, err := Run(context.Background(), candidate, []Check{{
		Name: "timeout", Command: helperCommand("block"), Timeout: time.Millisecond, Required: true,
	}}, testOptions())
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Outcome != "timed-out" {
		t.Fatalf("timeout result = %#v", results[0])
	}
	results, err = Run(context.Background(), candidate, []Check{{
		Name: "missing", Command: []string{filepath.Join(candidate, "does-not-exist")}, Required: true,
	}}, testOptions())
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Outcome != "infrastructure-failed" {
		t.Fatalf("launch result = %#v", results[0])
	}
}

func TestChecksRejectInvalidEnvironmentAndCommand(t *testing.T) {
	options := testOptions()
	options.Environment = []string{"BAD=VALUE"}
	if _, err := Run(context.Background(), t.TempDir(), nil, options); err == nil {
		t.Fatal("invalid environment name was accepted")
	}
	if _, err := Run(context.Background(), t.TempDir(), []Check{{Name: "empty"}}, testOptions()); err == nil {
		t.Fatal("empty check command was accepted")
	}
}

//nolint:errcheck,revive // Writes and exits intentionally model a direct-check subprocess.
func TestQualityHelper(t *testing.T) {
	separator := argumentIndex(os.Args, "--")
	if separator < 0 || separator+1 >= len(os.Args) {
		return
	}
	switch os.Args[separator+1] {
	case "pass":
		fmt.Fprint(os.Stdout, "passed")
		os.Exit(0)
	case "fail":
		fmt.Fprint(os.Stderr, "failed")
		os.Exit(9)
	case "mutate":
		if err := os.WriteFile(filepath.Join("generated.txt"), []byte("changed"), 0o600); err != nil {
			os.Exit(3)
		}
		os.Exit(0)
	case "overflow":
		fmt.Fprint(os.Stdout, strings.Repeat("x", 1<<20))
		os.Exit(0)
	case "block":
		for {
			runtime.Gosched()
		}
	default:
		os.Exit(4)
	}
}

func helperCommand(mode string) []string {
	return []string{os.Args[0], "-test.run=TestQualityHelper", "--", mode}
}

func testOptions() Options {
	return Options{
		DefaultTimeout: time.Minute, MaxOutputBytes: 1 << 20,
		Environment: []string{"PATH"}, Snapshot: workspace.Options{MaxBytes: 1 << 20, MaxFiles: 100},
	}
}

func argumentIndex(values []string, target string) int {
	for index, value := range values {
		if value == target {
			return index
		}
	}
	return -1
}
