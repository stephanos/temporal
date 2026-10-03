package execution

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/target"
)

// TestDiagnosticDrawCheckStopsHostTimedSeededDraw runs the draw_check fixture
// with a diagnostic trace. Without a fault, and with the fault switch's
// target-ordered draw, the run completes; with its host-timed form a
// host-timed path draws from the seeded stream and the runtime check stops the
// process before the fixture prints.
func TestDiagnosticDrawCheckStopsHostTimedSeededDraw(t *testing.T) {
	toolchainRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := target.Prepare(context.Background(), target.Spec{Kind: target.KindGoRun, Source: "./draw_check", WorkingDir: filepath.Join("..", "..", "..", "internal", "gomadtool", "conformance", "testdata"), PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot})
	if err != nil {
		t.Fatal(err)
	}
	implementation, err := choice.ImplementationIdentity(prepared.BuildKey)
	if err != nil {
		t.Fatal(err)
	}
	profile := deterministicio.Default()
	frame, err := profile.BootstrapFrame(prepared, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 7)
	if err != nil {
		t.Fatal(err)
	}
	launch := func(diagnostics bool, perturb string) (Result, error) {
		t.Helper()
		environment := []string{"GOMADSEED=7", "TZ=UTC", "GOMAD3_IO_PROFILE=" + profile.Name()}
		if perturb != "" {
			environment = append(environment, "GOMAD3_DIAGNOSTIC_PERTURB_DRAW="+perturb)
		}
		return Run(context.Background(), Spec{SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"}, BootstrapCommand: []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"}, Command: prepared.Path, Argv0: prepared.Argv[0], Dir: t.TempDir(), Env: environment, ExecutionTimeout: 10 * time.Second, TerminateGrace: time.Second, OutputLimit: 1024, World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: 7}, IO: &IOCapability{Config: frame, Transcript: &IOTranscriptCapability{Limit: 64 << 20}}, Choice: &ChoiceCapability{Mode: choice.ModeRecord, Profile: choice.Profile, ImplementationSHA256: implementation, Limit: 1 << 20}, Diagnostics: diagnostics})
	}
	run := func(diagnostics bool, perturb string) Result {
		t.Helper()
		result, err := launch(diagnostics, perturb)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	const message = "runtime: Gomad host-timed path drew from the seeded stream"
	off := run(false, "")
	plain := run(true, "")
	targetOrdered := run(true, "5")
	for name, result := range map[string]Result{"diagnostics off": off, "diagnostics on": plain, "target-ordered fault": targetOrdered} {
		if result.ExitCode != 0 || len(bytes.TrimSpace(result.Stdout.Bytes)) == 0 || strings.Contains(string(result.Stderr.Bytes), message) {
			t.Fatalf("%s: status %d, stdout %q, stderr %q", name, result.ExitCode, result.Stdout.Bytes, result.Stderr.Bytes)
		}
	}
	if !bytes.Equal(off.Stdout.Bytes, plain.Stdout.Bytes) || !bytes.Equal(off.ChoiceTrace.Trace.Bytes, plain.ChoiceTrace.Trace.Bytes) {
		t.Fatal("the diagnostic check changed behavior")
	}
	// The check stops the target before it closes its diagnostic trace, so
	// the launcher refuses the trace as incomplete.
	hostTimed, err := launch(true, "host-timed:5")
	if !errors.Is(err, choice.ErrDiagnosticIncomplete) {
		t.Fatalf("host-timed fault: error %v", err)
	}
	if hostTimed.ExitCode != 125 || !strings.Contains(string(hostTimed.Stderr.Bytes), message) || len(hostTimed.Stdout.Bytes) != 0 {
		t.Fatalf("host-timed fault: status %d, stdout %q, stderr %q", hostTimed.ExitCode, hostTimed.Stdout.Bytes, hostTimed.Stderr.Bytes)
	}
	// The host-timed form needs a diagnostic trace like the plain one; the
	// target exits during startup, before it opens its I/O transcript.
	rejected, err := launch(false, "host-timed:5")
	if err == nil || rejected.ExitCode != 2 || !strings.Contains(string(rejected.Stderr.Bytes), "runtime: invalid Gomad diagnostic trace configuration") {
		t.Fatalf("host-timed fault without diagnostics: error %v, status %d, stderr %q", err, rejected.ExitCode, rejected.Stderr.Bytes)
	}
}
