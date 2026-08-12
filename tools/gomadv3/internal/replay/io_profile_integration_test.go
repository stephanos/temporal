package replay

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/artifact"
	"go.temporal.io/server/tools/gomadv3/internal/ioprofile"
	"go.temporal.io/server/tools/gomadv3/internal/process"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/romount"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

func TestIOProfileFailureArtifactReplaysExactly(t *testing.T) {
	toolchain := toolchainRoot(t)
	prepared, err := target.Prepare(context.Background(), target.Spec{
		Kind: target.KindGoTest, Source: "./io_failure", WorkingDir: filepath.Join("..", "..", "testdata"),
		Args: []string{"-test.run=^TestDeterministicIOFailure$"}, BuildTags: []string{"test_dep"}, PreparationRoot: t.TempDir(), ToolchainRoot: toolchain,
	})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := ioprofile.Resolve(ioprofile.Deterministic)
	if err != nil {
		t.Fatal(err)
	}
	const runnerBuild = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	frame, err := profile.BootstrapFrame(prepared, runnerBuild, 7)
	if err != nil {
		t.Fatal(err)
	}
	environment := []string{"GOMADSEED=7", "GOMADV3_IO_PROFILE=" + profile.Name(), "TZ=UTC"}
	observed, err := process.Run(context.Background(), process.Request{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestIOReplaySupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestIOReplayBootstrapHelper"},
		Command:           prepared.Path, Argv0: prepared.Argv[0], Args: prepared.Argv[1:], Dir: t.TempDir(), Env: environment,
		RunTimeout: 10 * time.Second, TerminateGrace: time.Second, OutputLimit: 1 << 20,
		World: process.WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: 7},
		IO:    &process.IOCapability{Config: frame, Transcript: &process.IOTranscriptCapability{Limit: 64 << 20}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if observed.Termination != process.TerminationExit || observed.ExitCode == 0 || !observed.IOTranscript.Complete {
		t.Fatalf("fixture result = %#v", observed)
	}
	exitCode := record.Uint64String(observed.ExitCode)
	recordedWorld, worldPayloads := record.NoneWorld()
	manifest := record.Manifest{
		SchemaVersion: record.SchemaVersion, ArtifactKind: record.ArtifactTargetFailure, CreatedAt: "2026-08-11T12:00:00Z", BatchID: "io-replay-test",
		SelectionOrdinal: 0, Seed: 7, ReplayMode: record.ReplayExact,
		Runner:    record.Runner{RecordContract: "gomadv3.run-record/v1", RunnerBuild: runnerBuild, HostOS: runtime.GOOS, HostArch: runtime.GOARCH},
		Toolchain: record.Toolchain{GoVersion: prepared.GoVersion, BuildKey: prepared.BuildKey, TargetGOOS: prepared.TargetGOOS, TargetGOARCH: prepared.TargetGOARCH},
		Target: record.Target{
			Kind: string(prepared.Kind), Source: prepared.Source, SHA256: record.SHA256(prepared.SHA256), Size: record.Uint64String(prepared.Size),
			Argv: prepared.Argv, BuildTags: prepared.BuildTags, BuildInfo: prepared.BuildInfo,
		},
		IOProfile: record.IOProfile{
			Name: profile.Name(), ImplementationSHA256: profile.ImplementationSHA256(), Inventory: string(profile.Inventory()), InventorySHA256: profile.InventorySHA256(),
			Transcript: &record.IOTranscript{Schema: "gomadv3.io-transcript/v1", File: "io/transcript.bin", SHA256: transcriptSHA256(observed.IOTranscript.SHA256), Bytes: record.Uint64String(len(observed.IOTranscript.Bytes)), Records: record.Uint64String(observed.IOTranscript.Records)},
		},
		Environment: []record.Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "GOMADV3_IO_PROFILE", Value: profile.Name()}, {Name: "TZ", Value: "UTC"}},
		Limits: record.Limits{
			RunTimeoutNanos: record.Uint64String(10 * time.Second), OverallTimeoutNanos: record.Uint64String(time.Minute), TerminateGraceNanos: record.Uint64String(time.Second),
			OutputBytes: 1 << 20, WorldTransitionBytes: 1 << 20, IOTranscriptBytes: 64 << 20,
		},
		World:   recordedWorld,
		Outcome: record.Outcome{Domain: "target", Reason: "nonzero_exit", Termination: string(observed.Termination), ExitCode: &exitCode},
		Streams: record.Streams{Stdout: replayStream(observed.Stdout), Stderr: replayStream(observed.Stderr)},
		Host:    record.Host{StartedAt: "2026-08-11T12:00:00Z", FinishedAt: "2026-08-11T12:00:01Z", ElapsedNanos: record.Uint64String(time.Second)},
	}
	published, err := (artifact.Store{Root: t.TempDir()}).Publish(artifact.Input{
		Manifest: manifest, TargetPath: prepared.Path, Stdout: observed.Stdout.Bytes, Stderr: observed.Stderr.Bytes,
		IOTranscript: observed.IOTranscript.Bytes, World: worldPayloads,
	})
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := Replay(context.Background(), Config{
		ArtifactPath: published.Path, ToolchainRoot: toolchain,
		SupervisorCommand: []string{os.Args[0], "-test.run=TestIOReplaySupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestIOReplayBootstrapHelper"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Match || replayed.Divergence != "" {
		t.Fatalf("replay result = %#v", replayed)
	}
}

func TestReadOnlyMountFailureArtifactReplaysAfterHostRemoval(t *testing.T) {
	toolchain := toolchainRoot(t)
	prepared, err := target.Prepare(context.Background(), target.Spec{
		Kind: target.KindGoRun, Source: "./io_ro_mount_failure", WorkingDir: filepath.Join("..", "..", "testdata"),
		PreparationRoot: t.TempDir(), ToolchainRoot: toolchain,
	})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := ioprofile.Resolve(ioprofile.Deterministic)
	if err != nil {
		t.Fatal(err)
	}
	const runnerBuild = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	frame, err := profile.BootstrapFrame(prepared, runnerBuild, 7)
	if err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "schema.sql"), []byte("select 1;\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	mappings := []romount.Mapping{{Source: source, Target: "/mounted"}}
	limits := romount.DefaultLimits()
	environment := []string{"GOMADSEED=7", "GOMADV3_IO_PROFILE=" + profile.Name(), "TZ=UTC"}
	observed, err := process.Run(context.Background(), process.Request{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestIOReplaySupervisorHelper"}, BootstrapCommand: []string{os.Args[0], "-test.run=TestIOReplayBootstrapHelper"},
		Command: prepared.Path, Argv0: prepared.Argv[0], Dir: t.TempDir(), Env: environment,
		RunTimeout: 10 * time.Second, TerminateGrace: time.Second, OutputLimit: 1 << 20,
		World: process.WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: 7},
		IO: &process.IOCapability{Config: frame, Transcript: &process.IOTranscriptCapability{Limit: 64 << 20},
			ReadOnlyMount: &process.ReadOnlyMountCapability{Mappings: mappings, Limits: limits}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if observed.ExitCode != 2 || string(observed.Stdout.Bytes) != "select 1;\n" || len(observed.IOROMounts.Entries) != 1 {
		t.Fatalf("fixture result = %#v", observed)
	}
	mountArtifact, err := romount.EncodeArtifact(mappings, limits, observed.IOROMounts)
	if err != nil {
		t.Fatal(err)
	}
	exitCode := record.Uint64String(observed.ExitCode)
	recordedWorld, worldPayloads := record.NoneWorld()
	manifest := record.Manifest{
		SchemaVersion: record.SchemaVersion, ArtifactKind: record.ArtifactTargetFailure, CreatedAt: "2026-08-12T12:00:00Z", BatchID: "mount-replay-test",
		SelectionOrdinal: 0, Seed: 7, ReplayMode: record.ReplayExact,
		Runner:    record.Runner{RecordContract: "gomadv3.run-record/v1", RunnerBuild: runnerBuild, HostOS: runtime.GOOS, HostArch: runtime.GOARCH},
		Toolchain: record.Toolchain{GoVersion: prepared.GoVersion, BuildKey: prepared.BuildKey, TargetGOOS: prepared.TargetGOOS, TargetGOARCH: prepared.TargetGOARCH},
		Target: record.Target{
			Kind: string(prepared.Kind), Source: prepared.Source, SHA256: record.SHA256(prepared.SHA256), Size: record.Uint64String(prepared.Size),
			Argv: prepared.Argv, BuildTags: prepared.BuildTags, BuildInfo: prepared.BuildInfo,
		},
		IOProfile: record.IOProfile{
			Name: profile.Name(), ImplementationSHA256: profile.ImplementationSHA256(), Inventory: string(profile.Inventory()), InventorySHA256: profile.InventorySHA256(),
			Transcript:     &record.IOTranscript{Schema: "gomadv3.io-transcript/v1", File: "io/transcript.bin", SHA256: transcriptSHA256(observed.IOTranscript.SHA256), Bytes: record.Uint64String(len(observed.IOTranscript.Bytes)), Records: record.Uint64String(observed.IOTranscript.Records)},
			ReadOnlyMounts: &mountArtifact.Manifest,
		},
		Environment: []record.Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "GOMADV3_IO_PROFILE", Value: profile.Name()}, {Name: "TZ", Value: "UTC"}},
		Limits: record.Limits{
			RunTimeoutNanos: record.Uint64String(10 * time.Second), OverallTimeoutNanos: record.Uint64String(time.Minute), TerminateGraceNanos: record.Uint64String(time.Second),
			OutputBytes: 1 << 20, WorldTransitionBytes: 1 << 20, IOTranscriptBytes: 64 << 20,
		},
		World: recordedWorld, Outcome: record.Outcome{Domain: "target", Reason: "nonzero_exit", Termination: string(observed.Termination), ExitCode: &exitCode},
		Streams: record.Streams{Stdout: replayStream(observed.Stdout), Stderr: replayStream(observed.Stderr)},
		Host:    record.Host{StartedAt: "2026-08-12T12:00:00Z", FinishedAt: "2026-08-12T12:00:01Z", ElapsedNanos: record.Uint64String(time.Second)},
	}
	published, err := (artifact.Store{Root: t.TempDir()}).Publish(artifact.Input{
		Manifest: manifest, TargetPath: prepared.Path, Stdout: observed.Stdout.Bytes, Stderr: observed.Stderr.Bytes,
		IOTranscript: observed.IOTranscript.Bytes, ReadOnlyMounts: &mountArtifact, World: worldPayloads,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(source); err != nil {
		t.Fatal(err)
	}
	replayed, err := Replay(context.Background(), Config{
		ArtifactPath: published.Path, ToolchainRoot: toolchain,
		SupervisorCommand: []string{os.Args[0], "-test.run=TestIOReplaySupervisorHelper"}, BootstrapCommand: []string{os.Args[0], "-test.run=TestIOReplayBootstrapHelper"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Match || replayed.Divergence != "" {
		t.Fatalf("replay result = %#v", replayed)
	}
}

func TestIOReplaySupervisorHelper(t *testing.T) {
	if os.Getenv("GOMADV3_PROCESS_SUPERVISOR") != "1" {
		t.Skip("supervisor subprocess only")
	}
	if err := process.SupervisorMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	os.Exit(0)
}

func TestIOReplayBootstrapHelper(t *testing.T) {
	if os.Getenv("GOMADV3_TARGET_BOOTSTRAP") != "1" {
		t.Skip("target bootstrap subprocess only")
	}
	if err := process.BootstrapMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	os.Exit(0)
}

func transcriptSHA256(value [sha256.Size]byte) record.SHA256 {
	return record.SHA256("sha256:" + hex.EncodeToString(value[:]))
}
