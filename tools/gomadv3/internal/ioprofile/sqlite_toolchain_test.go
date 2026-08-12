package ioprofile

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/process"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

func TestProfileSQLiteUsesVirtualTimeAndEntropy(t *testing.T) {
	toolchainRoot, err := filepath.Abs(filepath.Join("..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	moduleCacheBytes, err := exec.Command(filepath.Join(toolchainRoot, "bin", "go"), "env", "GOMODCACHE").Output()
	if err != nil {
		t.Fatal(err)
	}
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	profile, err := Resolve(Deterministic)
	if err != nil {
		t.Fatal(err)
	}
	spec, _, err := profile.PrepareBuildOverlay(target.Spec{
		Kind: target.KindGoRun, Source: "./tools/gomadv3/root_testdata/io_sqlite", WorkingDir: repositoryRoot,
		PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot,
	}, strings.TrimSpace(string(moduleCacheBytes)))
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := target.Prepare(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	frame, err := profile.BootstrapFrame(prepared, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 7)
	if err != nil {
		t.Fatal(err)
	}
	result, err := process.Run(context.Background(), process.Request{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestEntropySupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestEntropyBootstrapHelper"},
		Command:           prepared.Path, Argv0: prepared.Argv[0], Dir: t.TempDir(), Env: []string{"GOMADV3_IO_PROFILE=" + profile.Name(), "GOMADSEED=7", "TZ=UTC"},
		RunTimeout: 10 * time.Second, TerminateGrace: time.Second, OutputLimit: 4096,
		World: process.WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: 7},
		IO:    &process.IOCapability{Config: frame, Transcript: &process.IOTranscriptCapability{Limit: 64 << 20}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Termination != process.TerminationExit || result.ExitCode != 0 || string(result.Stdout.Bytes) != "42 2000-01-01 00:00:00\n" {
		t.Fatalf("result = %#v, stderr = %q", result, result.Stderr.Bytes)
	}
}
