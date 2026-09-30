package execution_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	"go.temporal.io/server/tools/gomad3/target"
)

// TestProfileNetworkBindContract runs net_bind under the deterministic profile:
// an unspecified host binds the in-memory loopback listener, listeners and
// addresses keep their concrete net types, and a closed port can be bound
// again, with the same outcome and transcript for every seed.
func TestProfileNetworkBindContract(t *testing.T) {
	toolchainRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := target.Prepare(context.Background(), target.Spec{
		Kind: target.KindGoRun, Source: "./net_bind", WorkingDir: filepath.Join("..", "..", "..", "internal", "gomadtool", "conformance", "testdata"),
		PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	profile := deterministicio.Default()
	var transcripts []string
	for _, seed := range []uint64{1, 999} {
		frame, err := profile.BootstrapFrame(prepared, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", seed)
		if err != nil {
			t.Fatal(err)
		}
		result, err := execution.Run(context.Background(), execution.Spec{
			SupervisorCommand: []string{os.Args[0], "-test.run=TestEntropySupervisorHelper"},
			BootstrapCommand:  []string{os.Args[0], "-test.run=TestEntropyBootstrapHelper"},
			Command:           prepared.Path, Argv0: prepared.Argv[0], Dir: t.TempDir(), Env: []string{"GOMAD3_IO_PROFILE=" + profile.Name(), fmt.Sprintf("GOMADSEED=%d", seed), "TZ=UTC"},
			ExecutionTimeout: 10 * time.Second, TerminateGrace: time.Second, OutputLimit: 4096,
			World: execution.WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: seed},
			IO:    &execution.IOCapability{Config: frame, Transcript: &execution.IOTranscriptCapability{Limit: 64 << 20}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.ExitCode != 0 || result.Termination != execution.TerminationExit || string(result.Stdout.Bytes) != "net-bind ok\n" {
			t.Fatalf("seed %d result = %#v, stdout = %q, stderr = %q", seed, result.Termination, result.Stdout.Bytes, result.Stderr.Bytes)
		}
		if !result.IOTranscript.Complete || result.IOTranscript.Records == 0 {
			t.Fatalf("seed %d I/O transcript = %#v", seed, result.IOTranscript)
		}
		transcripts = append(transcripts, string(result.IOTranscript.Bytes))
	}
	if transcripts[0] != transcripts[1] {
		t.Fatal("net_bind transcript changed with the schedule seed")
	}
}
