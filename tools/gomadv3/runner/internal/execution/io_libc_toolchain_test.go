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

func TestModerncLibcUsesDeterministicFilesystem(t *testing.T) {
	toolchainRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory, err := filepath.Abs(filepath.Join("..", "..", "..", "toolchain", "internal", "conformance", "testdata", "libc_adapter"))
	if err != nil {
		t.Fatal(err)
	}
	preparationRoot := t.TempDir()
	moduleCache, err := target.ReadModuleCache(context.Background(), toolchainRoot)
	if err != nil {
		t.Fatal(err)
	}
	profile := deterministicio.Default()
	spec, adapters, err := profile.PrepareBuildAdapters(target.Spec{
		Kind: target.KindGoRun, Source: ".", WorkingDir: workingDirectory,
		PreparationRoot: preparationRoot, ToolchainRoot: toolchainRoot,
	}, moduleCache)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := target.Prepare(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	prepared.Adapters = recordAdapters(adapters)
	frame, err := profile.BootstrapFrame(prepared, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 7)
	if err != nil {
		t.Fatal(err)
	}
	runDirectory := t.TempDir()
	result, err := execution.Run(context.Background(), execution.Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestEntropySupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestEntropyBootstrapHelper"},
		Command:           prepared.Path, Argv0: prepared.Argv[0], Dir: runDirectory,
		Env:        []string{"GOMADV3_IO_PROFILE=" + profile.Name(), "GOMADSEED=7", "TZ=UTC"},
		RunTimeout: 10 * time.Second, TerminateGrace: time.Second, OutputLimit: 4096,
		World: execution.WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: 7},
		IO:    &execution.IOCapability{Config: frame, Transcript: &execution.IOTranscriptCapability{Limit: 64 << 20}},
	})
	if err != nil {
		t.Fatalf("execution.Run() error = %v, result = %#v, stderr = %q", err, result, result.Stderr.Bytes)
	}
	if result.Termination != execution.TerminationExit || result.ExitCode != 0 || string(result.Stdout.Bytes) != "ok\n" {
		t.Fatalf("result = %#v, stderr = %q", result, result.Stderr.Bytes)
	}
	if _, err := os.Stat(filepath.Join(runDirectory, "workspace")); !os.IsNotExist(err) {
		t.Fatalf("libc adapter created host filesystem state: %v", err)
	}
}
