package execution_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	"go.temporal.io/server/tools/gomad3/target"
)

// A runtime lock that outlasts the waiter's spin is host timing. The sampling
// draw lock2 makes before it sleeps once came from the seeded stream when the
// waiting M held the P, so the stream stood one draw further in the executions
// where the scheduler lock was contended that long. No single execution is
// certain to contend, so the test compares several, run together so that they
// also compete for the host's processors.
func TestProfileSchedulerLockContentionLeavesSeededStreamInPlace(t *testing.T) {
	const executions = 12
	toolchainRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := target.Prepare(context.Background(), target.Spec{
		Kind: target.KindGoRun, Source: "./io_handoff_contention", WorkingDir: filepath.Join("..", "..", "..", "internal", "gomadtool", "conformance", "testdata"),
		PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	profile := deterministicio.Default()
	frame, err := profile.BootstrapFrame(prepared, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 7)
	if err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	outputs := make([]string, executions)
	failures := make([]string, executions)
	var group sync.WaitGroup
	for index := range executions {
		directory := t.TempDir()
		group.Add(1)
		go func() {
			defer group.Done()
			result, err := execution.Run(context.Background(), execution.Spec{
				SupervisorCommand: []string{os.Args[0], "-test.run=TestEntropySupervisorHelper"}, BootstrapCommand: []string{os.Args[0], "-test.run=TestEntropyBootstrapHelper"},
				Command: prepared.Path, Argv0: prepared.Argv[0], Dir: directory, Env: []string{"GOMAD3_IO_PROFILE=" + profile.Name(), "GOMADSEED=7", "TZ=UTC"},
				ExecutionTimeout: time.Minute, TerminateGrace: time.Second, OutputLimit: 4096,
				World: execution.WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: 7},
				IO: &execution.IOCapability{Config: frame, Transcript: &execution.IOTranscriptCapability{Limit: 64 << 20},
					ReadOnlyMount: &execution.ReadOnlyMountCapability{Mappings: []readonlymount.Mapping{{Source: source, Target: "/mounted"}}, Limits: readonlymount.DefaultLimits()}},
			})
			outputs[index] = string(result.Stdout.Bytes)
			if err != nil || result.ExitCode != 0 || !strings.HasPrefix(outputs[index], "type-assertion cache grew at ") {
				failures[index] = fmt.Sprintf("execution.Run() error = %v, result = %#v, stderr = %q", err, result, result.Stderr.Bytes)
			}
		}()
	}
	group.Wait()
	for index := range executions {
		if failures[index] != "" {
			t.Fatalf("execution %d: %s", index, failures[index])
		}
		if outputs[index] != outputs[0] {
			t.Fatalf("execution %d placed the seeded stream differently:\n%s%s", index, outputs[0], outputs[index])
		}
	}
}
