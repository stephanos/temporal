package ioprofile

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/process"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

func TestBoundaryManifestSemanticCanaries(t *testing.T) {
	observed := make(map[string]struct{}, len(generatedBoundaryProbes))
	for _, fixture := range []string{"./io_filesystem", "./io_net"} {
		result := runBoundaryCanaryFixture(t, fixture)
		coverage, err := DecodeSemanticCoverage(result.IOTranscript.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		for _, probe := range coverage.Probes {
			observed[probe] = struct{}{}
		}
	}
	var missing []string
	for _, probe := range generatedBoundaryProbes {
		if _, found := observed[probe.Name]; !found {
			missing = append(missing, probe.Name)
		}
	}
	if len(missing) != 0 {
		slices.Sort(missing)
		t.Fatalf("boundary manifest entries have no positive semantic canary: %v", missing)
	}
}

func runBoundaryCanaryFixture(t *testing.T, source string) process.Result {
	t.Helper()
	toolchainRoot, err := filepath.Abs(filepath.Join("..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := target.Prepare(context.Background(), target.Spec{
		Kind: target.KindGoRun, Source: source, WorkingDir: filepath.Join("..", "..", "testdata"),
		PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	profile := Default()
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
	if result.Termination != process.TerminationExit || result.ExitCode != 0 {
		t.Fatalf("boundary canary fixture %s result = %#v, stderr = %q", source, result, result.Stderr.Bytes)
	}
	return result
}
