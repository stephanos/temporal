package ioprofile

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/process"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

func TestProfileEntropyIsIndependentOfScheduleSeed(t *testing.T) {
	toolchainRoot, err := filepath.Abs(filepath.Join("..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := target.Prepare(context.Background(), target.Spec{
		Kind: target.KindGoRun, Source: "./io_entropy", WorkingDir: filepath.Join("..", "..", "testdata"),
		PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	symbols, err := exec.Command(filepath.Join(toolchainRoot, "bin", "go"), "tool", "nm", prepared.Path).Output()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(symbols), "runtime.gomadIOConfigFrame") {
		t.Fatalf("prepared target omitted the Gomad runtime configuration frame (toolchain=%s build=%s target=%s runtime=%v gomadio=%v GOROOT=%q)", toolchainRoot, prepared.BuildKey, prepared.Path, strings.Contains(string(symbols), "runtime.gomadInit"), strings.Contains(string(symbols), "internal/gomadio"), os.Getenv("GOROOT"))
	}
	profile := Default()
	outputs := make([]string, 2)
	transcripts := make([]process.IOTranscript, 2)
	for index, seed := range []uint64{1, 999} {
		frame, frameErr := profile.BootstrapFrame(prepared, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", seed)
		if frameErr != nil {
			t.Fatal(frameErr)
		}
		result, runErr := process.Run(context.Background(), process.Request{
			SupervisorCommand: []string{os.Args[0], "-test.run=TestEntropySupervisorHelper"},
			BootstrapCommand:  []string{os.Args[0], "-test.run=TestEntropyBootstrapHelper"},
			Command:           prepared.Path, Argv0: prepared.Argv[0], Dir: t.TempDir(), Env: []string{"GOMADV3_IO_PROFILE=" + profile.Name(), fmt.Sprintf("GOMADSEED=%d", seed), "TZ=UTC"},
			RunTimeout: 10 * time.Second, TerminateGrace: time.Second, OutputLimit: 1024,
			World: process.WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: seed},
			IO:    &process.IOCapability{Config: frame, Transcript: &process.IOTranscriptCapability{Limit: 64 << 20}},
		})
		if runErr != nil {
			t.Fatal(runErr)
		}
		if result.ExitCode != 0 || result.Termination != process.TerminationExit {
			t.Fatalf("seed %d result = %#v, stderr = %q", seed, result, result.Stderr.Bytes)
		}
		if !result.IOTranscript.Complete || result.IOTranscript.Records == 0 || len(result.IOTranscript.Bytes) == 0 {
			t.Fatalf("seed %d I/O transcript = %#v", seed, result.IOTranscript)
		}
		outputs[index] = string(result.Stdout.Bytes)
		transcripts[index] = result.IOTranscript
	}
	if outputs[0] != outputs[1] {
		t.Fatalf("profile entropy changed with schedule seed: %q != %q", outputs[0], outputs[1])
	}
	if digest := sha256.Sum256([]byte(outputs[0])); digest == ([sha256.Size]byte{}) {
		t.Fatal("empty entropy output digest")
	}
	frame, err := profile.BootstrapFrame(prepared, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 1)
	if err != nil {
		t.Fatal(err)
	}
	runReplay := func(expected []byte) process.Result {
		t.Helper()
		result, runErr := process.Run(context.Background(), process.Request{
			SupervisorCommand: []string{os.Args[0], "-test.run=TestEntropySupervisorHelper"},
			BootstrapCommand:  []string{os.Args[0], "-test.run=TestEntropyBootstrapHelper"},
			Command:           prepared.Path, Argv0: prepared.Argv[0], Dir: t.TempDir(), Env: []string{"GOMADV3_IO_PROFILE=" + profile.Name(), "GOMADSEED=1", "TZ=UTC"},
			RunTimeout: 10 * time.Second, TerminateGrace: time.Second, OutputLimit: 1024,
			World: process.WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: 1},
			IO:    &process.IOCapability{Config: frame, Transcript: &process.IOTranscriptCapability{Limit: 64 << 20, Replay: true, Expected: expected}},
		})
		if runErr != nil {
			t.Fatal(runErr)
		}
		return result
	}
	replayed := runReplay(transcripts[0].Bytes)
	if replayed.Termination != process.TerminationExit || replayed.ExitCode != 0 || replayed.IOTranscript.ReplayDivergence != nil {
		t.Fatalf("matching replay result = %#v", replayed)
	}
	changed := append([]byte(nil), transcripts[0].Bytes...)
	changed[0] ^= 1
	diverged := runReplay(changed)
	if diverged.IOTranscript.ReplayDivergence == nil || *diverged.IOTranscript.ReplayDivergence != 0 {
		t.Fatalf("divergent replay result = %#v", diverged)
	}
}

func TestEntropySupervisorHelper(t *testing.T) {
	if os.Getenv("GOMADV3_PROCESS_SUPERVISOR") != "1" {
		t.Skip("supervisor subprocess only")
	}
	if err := process.SupervisorMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	os.Exit(0)
}

func TestEntropyBootstrapHelper(t *testing.T) {
	if os.Getenv("GOMADV3_TARGET_BOOTSTRAP") != "1" {
		t.Skip("target bootstrap subprocess only")
	}
	if err := process.BootstrapMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	os.Exit(0)
}
