package execution_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/runner/internal/execution"
	"go.temporal.io/server/tools/gomadv3/target"
)

func TestVirtualTCPCloseAfterWriteDeliversFinalBytes(t *testing.T) {
	runVirtualTCPCase(t, "close-after-write", 10*time.Second)
}

func TestVirtualTCPAcceptsQueuedConnectionBeforeClose(t *testing.T) {
	runVirtualTCPCase(t, "accept-close", 10*time.Second)
}

func TestVirtualTCPRejectsWritesAfterHalfClose(t *testing.T) {
	runVirtualTCPCase(t, "write-close", 10*time.Second)
}

func TestVirtualTCPRejectsQueuedReadsAfterClose(t *testing.T) {
	runVirtualTCPCase(t, "read-close", 10*time.Second)
}

func TestVirtualTCPRejectsQueuedReadsAfterCloseRead(t *testing.T) {
	runVirtualTCPCase(t, "read-close-read", 10*time.Second)
}

func TestVirtualTCPAcceptsQueuedConnectionBeforeDeadline(t *testing.T) {
	runVirtualTCPCase(t, "accept-deadline", 10*time.Second)
}

func TestVirtualTCPWritesToAvailableBufferBeforeDeadline(t *testing.T) {
	runVirtualTCPCase(t, "write-deadline", 10*time.Second)
}

func TestVirtualTCPCanceledBlockedDialDoesNotConnect(t *testing.T) {
	runVirtualTCPCase(t, "dial-cancel", 10*time.Second)
}

func TestVirtualTCPClosedListenerRejectsBlockedDial(t *testing.T) {
	runVirtualTCPCase(t, "dial-close", 10*time.Second)
}

func TestVirtualTCPReportsPortExhaustion(t *testing.T) {
	runVirtualTCPCase(t, "port-exhaustion", 30*time.Second)
}

func runVirtualTCPCase(t *testing.T, testCase string, timeout time.Duration) {
	t.Helper()
	toolchainRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := target.Prepare(context.Background(), target.Spec{
		Kind: target.KindGoRun, Source: "./io_net_races", WorkingDir: filepath.Join("..", "..", "..", "internal", "gomadtool", "conformance", "testdata"),
		PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	profile := deterministicio.Default()
	const seed = 1
	frame, err := profile.BootstrapFrame(prepared, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", seed)
	if err != nil {
		t.Fatal(err)
	}
	result, err := execution.Run(context.Background(), execution.Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestEntropySupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestEntropyBootstrapHelper"},
		Command:           prepared.Path, Args: []string{testCase}, Argv0: prepared.Argv[0], Dir: t.TempDir(), Env: []string{"GOMADV3_IO_PROFILE=" + profile.Name(), "GOMADSEED=1", "TZ=UTC"},
		ExecutionTimeout: timeout, TerminateGrace: time.Second, OutputLimit: 4096,
		World: execution.WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: seed},
		IO:    &execution.IOCapability{Config: frame, Transcript: &execution.IOTranscriptCapability{Limit: 64 << 20}},
	})
	if err != nil {
		t.Fatalf("case %s seed %d: %v, stdout = %q, stderr = %q", testCase, seed, err, result.Stdout.Bytes, result.Stderr.Bytes)
	}
	if result.Termination != execution.TerminationExit || result.ExitCode != 0 || string(result.Stdout.Bytes) != "ok\n" {
		t.Fatalf("case %s seed %d: stdout = %q, stderr = %q, termination = %s/%d", testCase, seed, result.Stdout.Bytes, result.Stderr.Bytes, result.Termination, result.ExitCode)
	}
}

func TestProfileTCPUsesInMemoryLoopback(t *testing.T) {
	toolchainRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := target.Prepare(context.Background(), target.Spec{
		Kind: target.KindGoRun, Source: "./io_net", WorkingDir: filepath.Join("..", "..", "..", "internal", "gomadtool", "conformance", "testdata"),
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
	result, err := execution.Run(context.Background(), execution.Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestEntropySupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestEntropyBootstrapHelper"},
		Command:           prepared.Path, Argv0: prepared.Argv[0], Dir: t.TempDir(), Env: []string{"GOMADV3_IO_PROFILE=" + profile.Name(), "GOMADSEED=7", "TZ=UTC"},
		ExecutionTimeout: 10 * time.Second, TerminateGrace: time.Second, OutputLimit: 4096,
		World: execution.WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: 7},
		IO:    &execution.IOCapability{Config: frame, Transcript: &execution.IOTranscriptCapability{Limit: 64 << 20}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Termination != execution.TerminationExit || result.ExitCode != 0 || string(result.Stdout.Bytes) != "ok\n" {
		t.Fatalf("result = %#v, stderr = %q", result, result.Stderr.Bytes)
	}
}
