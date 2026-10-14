package execution_test

import (
	"context"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	nativebuild "go.temporal.io/server/tools/gomad/internal/backends/native/build"
	"go.temporal.io/server/tools/gomad/internal/backends/native/execution"
	"go.temporal.io/server/tools/gomad/internal/backends/native/transcript"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
)

func simulationReplayTarget(t *testing.T) target.Spec {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return target.Spec{
		Kind: target.KindGoRun, Source: "./internal/model/cluster/testdata/simulation_replay", WorkingDir: root,
		BuildTags: []string{"test_dep", "gomad3_toolchain"}, PreparationRoot: t.TempDir(), ToolchainRoot: filepath.Join(root, ".toolchain"),
	}
}

func TestSimulationReplayTargetResolvesInGomadModule(t *testing.T) {
	spec := simulationReplayTarget(t)
	command := exec.CommandContext(t.Context(), "go", "list", "-mod=readonly", "-tags", strings.Join(spec.BuildTags, ","), "-f", "{{.Module.Path}}|{{.ImportPath}}", spec.Source)
	command.Dir = spec.WorkingDir
	command.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOWORK=off", "GOFLAGS=")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("resolve simulation replay target: %v: %s", err, output)
	}
	const expected = "go.temporal.io/server/tools/gomad|go.temporal.io/server/tools/gomad/internal/model/cluster/testdata/simulation_replay"
	if strings.TrimSpace(string(output)) != expected {
		t.Fatalf("simulation replay target ownership = %q, want %q", output, expected)
	}
}

func TestSimulationReplayPayloadDivergence(t *testing.T) {
	prepared, err := nativebuild.Prepare(context.Background(), simulationReplayTarget(t))
	if err != nil {
		t.Fatal(err)
	}
	profile := transcript.Default()
	const seed = 89
	const transcriptLimit = 64 << 20
	frame, err := profile.BootstrapFrame(prepared, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", seed)
	if err != nil {
		t.Fatal(err)
	}
	run := func(t *testing.T, payload string, composed bool, expected []byte) execution.Result {
		t.Helper()
		if err := nativebuild.VerifyPrepared(prepared); err != nil {
			t.Fatal(err)
		}
		request := execution.Spec{
			SupervisorCommand: []string{os.Args[0], "-test.run=TestEntropySupervisorHelper"},
			BootstrapCommand:  []string{os.Args[0], "-test.run=TestEntropyBootstrapHelper"},
			Command:           prepared.Path, Args: prepared.Argv[1:], Argv0: prepared.Argv[0], Dir: t.TempDir(),
			Env: []string{"GOMADSEED=89", "TZ=UTC"}, ExecutionTimeout: 20 * time.Second, TerminateGrace: time.Second, OutputLimit: 4096,
			World:      execution.WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: seed},
			Simulation: &execution.SimulationCapability{Role: execution.SimulationRoleCoordinator},
		}
		if composed {
			request.Env = append(request.Env, "GOMAD3_IO_PROFILE="+profile.Name(), "GOMAD3_SIMULATION_REPLAY_PAYLOAD="+payload)
			request.IO = &execution.IOCapability{Config: frame, Transcript: &execution.IOTranscriptCapability{
				Limit: transcriptLimit, Replay: expected != nil, Expected: expected,
			}}
		}
		result, runErr := execution.Run(context.Background(), request)
		if runErr != nil {
			t.Fatalf("simulation execution: %v, stderr=%q", runErr, result.Stderr.Bytes)
		}
		return result
	}
	completed := func(t *testing.T, result execution.Result) {
		t.Helper()
		if result.WatchdogTimeout || result.Cancelled || result.Termination != execution.TerminationExit || result.ExitCode != 0 || string(result.Stdout.Bytes) != "ok\n" {
			t.Fatalf("simulation result=%#v, stderr=%q", result, result.Stderr.Bytes)
		}
	}
	t.Run("direct-network-only-waiver", func(t *testing.T) {
		result := run(t, "", false, nil)
		completed(t, result)
		if result.IOTranscript.Records != 0 || len(result.IOTranscript.Bytes) != 0 {
			t.Fatalf("network-only control unexpectedly recorded generic I/O: %#v", result.IOTranscript)
		}
	})
	t.Run("composed-generic-io", func(t *testing.T) {
		original := run(t, "one", true, nil)
		completed(t, original)
		changed := run(t, "two", true, nil)
		completed(t, changed)
		for _, result := range []execution.Result{original, changed} {
			if !result.IOTranscript.Complete || result.IOTranscript.Records == 0 || len(result.IOTranscript.Bytes) == 0 || uint64(len(result.IOTranscript.Bytes)) > transcriptLimit {
				t.Fatalf("incomplete or unbounded generic I/O transcript: %#v", result.IOTranscript)
			}
		}
		before, err := transcript.DecodeTranscript(original.IOTranscript.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		after, err := transcript.DecodeTranscript(changed.IOTranscript.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		if len(before) != len(after) {
			t.Fatalf("same-length payload changed operation count: %d != %d", len(before), len(after))
		}
		var mismatch *uint64
		for index, expected := range before {
			actual := after[index]
			if expected == actual {
				continue
			}
			if mismatch == nil {
				ordinal := expected.Ordinal
				mismatch = &ordinal
				if expected.Name != "net.write" || expected.Count != 3 || expected.ContentHash != sha256.Sum256([]byte("one")) || actual.ContentHash != sha256.Sum256([]byte("two")) {
					t.Fatalf("first mismatch is not the actual same-length stream write: expected=%#v actual=%#v", expected, actual)
				}
			}
			actual.ContentHash = expected.ContentHash
			if actual != expected {
				t.Fatalf("payload substitution changed operation structure at %d: expected=%#v actual=%#v", index, expected, actual)
			}
		}
		if mismatch == nil {
			t.Fatal("actual changed stream payload did not change generic I/O content evidence")
		}
		matching := run(t, "one", true, original.IOTranscript.Bytes)
		completed(t, matching)
		if !matching.IOTranscript.Complete || matching.IOTranscript.ReplayDivergence != nil {
			t.Fatalf("matching replay=%#v", matching.IOTranscript)
		}
		diverged := run(t, "two", true, original.IOTranscript.Bytes)
		if diverged.WatchdogTimeout || diverged.Cancelled || diverged.ExitCode == 0 || diverged.IOTranscript.ReplayDivergence == nil || *diverged.IOTranscript.ReplayDivergence != *mismatch {
			t.Fatalf("changed payload replay did not reject at generic I/O ordinal %d: result=%#v", *mismatch, diverged)
		}
	})
}
