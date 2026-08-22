package execution_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/runner/internal/execution"
	"go.temporal.io/server/tools/gomadv3/target"
)

func TestBoundaryManifestSemanticCanaries(t *testing.T) {
	required := deterministicio.RequiredSemanticProbes()
	observed := make(map[string]struct{}, len(required))
	for _, fixture := range []struct {
		source         string
		capabilityMode target.CapabilityMode
	}{
		{source: "./io_filesystem"},
		{source: "./io_net"},
		{source: "./io_signal", capabilityMode: target.CapabilityModeGuarded},
		{source: "./io_user", capabilityMode: target.CapabilityModeGuarded},
	} {
		result := runBoundaryCanaryFixture(t, fixture.source, fixture.capabilityMode)
		coverage, err := deterministicio.DecodeSemanticCoverage(result.IOTranscript.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		for _, probe := range coverage.Probes {
			observed[probe] = struct{}{}
		}
	}
	var missing []string
	for _, probe := range required {
		if _, found := observed[probe]; !found {
			missing = append(missing, probe)
		}
	}
	if len(missing) != 0 {
		slices.Sort(missing)
		t.Fatalf("boundary manifest entries have no positive semantic canary: %v", missing)
	}
}

func runBoundaryCanaryFixture(t *testing.T, source string, capabilityMode target.CapabilityMode) execution.Result {
	t.Helper()
	toolchainRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := target.Prepare(context.Background(), target.Spec{
		Kind: target.KindGoRun, Source: source, WorkingDir: filepath.Join("..", "..", "..", "internal", "gomadtool", "conformance", "testdata"),
		PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot, CapabilityMode: capabilityMode,
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
	if result.Termination != execution.TerminationExit || result.ExitCode != 0 {
		t.Fatalf("boundary canary fixture %s result = %#v, stderr = %q", source, result, result.Stderr.Bytes)
	}
	return result
}
