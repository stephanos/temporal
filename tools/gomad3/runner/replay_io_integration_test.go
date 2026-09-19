package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestIOProfileFailureArtifactReplaysExactly(t *testing.T) {
	toolchain := toolchainRoot(t)
	prepared, err := target.Prepare(context.Background(), target.Spec{
		Kind: target.KindGoTest, Source: "./io_failure", WorkingDir: filepath.Join("..", "internal", "gomadtool", "conformance", "testdata"),
		Args: []string{"-test.run=^TestDeterministicIOFailure$"}, BuildTags: []string{"gomad_fixture"}, PreparationRoot: t.TempDir(), ToolchainRoot: toolchain,
	})
	if err != nil {
		t.Fatal(err)
	}
	profile := deterministicio.Default()
	const runnerBuild = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	frame, err := profile.BootstrapFrame(prepared, runnerBuild, 7)
	if err != nil {
		t.Fatal(err)
	}
	environment := []string{"GOMAD3_IO_PROFILE=" + profile.Name(), "GOMADSEED=7", "TZ=UTC"}
	observed, err := execution.Run(context.Background(), execution.Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestIOReplaySupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestIOReplayBootstrapHelper"},
		Command:           prepared.Path, Argv0: prepared.Argv[0], Args: prepared.Argv[1:], Dir: t.TempDir(), Env: environment,
		ExecutionTimeout: 10 * time.Second, TerminateGrace: time.Second, OutputLimit: 1 << 20,
		World: execution.WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: 7},
		IO:    &execution.IOCapability{Config: frame, Transcript: &execution.IOTranscriptCapability{Limit: 64 << 20}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if observed.Termination != execution.TerminationExit || observed.ExitCode == 0 || !observed.IOTranscript.Complete {
		t.Fatalf("fixture result = %#v", observed)
	}
	exitCode := record.Uint64String(observed.ExitCode)
	recordedWorld, worldPayloads := record.NoneWorld()
	manifest := record.ExecutionRecord{
		SchemaVersion: record.SchemaVersion, ArtifactKind: record.ArtifactTargetFailure, CreatedAt: "2026-08-11T12:00:00Z", CampaignID: "io-replay-test",
		SelectionOrdinal: 0, Seed: 7, ReplayMode: record.ReplayExact,
		Runner:    record.Runner{RecordContract: record.RecordContract, RunnerBuild: runnerBuild, HostOS: runtime.GOOS, HostArch: runtime.GOARCH},
		Toolchain: record.Toolchain{GoVersion: prepared.GoVersion, BuildKey: prepared.BuildKey, TargetGOOS: prepared.TargetGOOS, TargetGOARCH: prepared.TargetGOARCH},
		Target: record.Target{
			Kind: string(prepared.Kind), Source: prepared.Source, SHA256: record.SHA256(prepared.SHA256), Size: record.Uint64String(prepared.Size),
			Argv: prepared.Argv, BuildTags: prepared.BuildTags, Adapters: prepared.Adapters, Compatibility: prepared.Compatibility, BuildInfo: prepared.BuildInfo,
		},
		IOProfile: record.IOProfile{
			Name: profile.Name(), ImplementationSHA256: record.SHA256(profile.ImplementationSHA256()), Inventory: string(profile.Inventory()), InventorySHA256: record.SHA256(profile.InventorySHA256()),
			Transcript: &record.IOTranscript{Schema: "gomad3.io-transcript/v1", File: "io/transcript.bin", SHA256: transcriptSHA256(observed.IOTranscript.SHA256), Bytes: record.Uint64String(len(observed.IOTranscript.Bytes)), Records: record.Uint64String(observed.IOTranscript.Records)},
		},
		Environment: []record.Environment{{Name: "GOMAD3_IO_PROFILE", Value: profile.Name()}, {Name: "GOMADSEED", Value: "7"}, {Name: "TZ", Value: "UTC"}},
		Limits: record.Limits{
			ExecutionTimeoutNanos: record.Uint64String(10 * time.Second), OverallTimeoutNanos: record.Uint64String(time.Minute), TerminateGraceNanos: record.Uint64String(time.Second),
			OutputBytes: 1 << 20, WorldTransitionBytes: 1 << 20, IOTranscriptBytes: 64 << 20,
		},
		World:   recordedWorld,
		Outcome: record.Outcome{Domain: "target", Reason: "nonzero_exit", Termination: string(observed.Termination), ExitCode: &exitCode},
		Streams: record.Streams{Stdout: replayStream(observed.Stdout), Stderr: replayStream(observed.Stderr)},
		Host:    record.Host{StartedAt: "2026-08-11T12:00:00Z", FinishedAt: "2026-08-11T12:00:01Z", ElapsedNanos: record.Uint64String(time.Second)},
	}
	published, err := artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, artifact.ArtifactInput{
		Manifest: manifest, TargetPath: prepared.Path, Stdout: observed.Stdout.Bytes, Stderr: observed.Stderr.Bytes,
		IOTranscript: observed.IOTranscript.Bytes, World: worldPayloads,
	})
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := Replay(context.Background(), ReplaySpec{
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

func TestLinkedCapabilityArtifactRevalidatesRetainedExecutableBeforeReplay(t *testing.T) {
	module := t.TempDir()
	if err := os.WriteFile(filepath.Join(module, "go.mod"), []byte("module example.com/linkedreplay\n\ngo 1.26\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(module, "main.go"), []byte("package main\nimport \"os/exec\"\nfunc unused() { _, _ = exec.Command(\"unused\").Output() }\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	toolchain := toolchainRoot(t)
	prepared, err := target.Prepare(context.Background(), target.Spec{
		Kind: target.KindGoRun, Source: ".", WorkingDir: module, PreparationRoot: t.TempDir(), ToolchainRoot: toolchain,
		CapabilityMode: target.CapabilityModeLinked,
	})
	if err != nil {
		t.Fatal(err)
	}
	profile := deterministicio.Default()
	worldRecord, worldPayloads := record.NoneWorld()
	exitCode := record.Uint64String(2)
	manifest := record.ExecutionRecord{
		SchemaVersion: record.SchemaVersion, ArtifactKind: record.ArtifactTargetFailure, CreatedAt: "2026-08-15T12:00:00Z", CampaignID: "linked-replay-test",
		SelectionOrdinal: 0, Seed: 7, ReplayMode: record.ReplayExact,
		Runner:    record.Runner{RecordContract: record.RecordContract, RunnerBuild: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", HostOS: runtime.GOOS, HostArch: runtime.GOARCH},
		Toolchain: prepared.RecordToolchain(), Target: prepared.RecordTarget(),
		IOProfile:   record.IOProfile{Name: profile.Name(), ImplementationSHA256: record.SHA256(profile.ImplementationSHA256()), Inventory: string(profile.Inventory()), InventorySHA256: record.SHA256(profile.InventorySHA256())},
		Environment: []record.Environment{{Name: "GOMAD3_IO_PROFILE", Value: profile.Name()}, {Name: "GOMADSEED", Value: "7"}, {Name: "TZ", Value: "UTC"}},
		Limits:      record.Limits{ExecutionTimeoutNanos: record.Uint64String(time.Second), OverallTimeoutNanos: record.Uint64String(time.Minute), OutputBytes: 64, WorldTransitionBytes: 64},
		World:       worldRecord, Outcome: record.Outcome{Domain: "target", Reason: "nonzero_exit", Termination: "exit", ExitCode: &exitCode},
		Streams: record.Streams{
			Stdout: record.Stream{FullSHA256: record.HashBytes(nil)},
			Stderr: record.Stream{FullSHA256: record.HashBytes(nil)},
		},
		Host: record.Host{StartedAt: "2026-08-15T12:00:00Z", FinishedAt: "2026-08-15T12:00:01Z", ElapsedNanos: record.Uint64String(time.Second)},
	}
	published, err := artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, artifact.ArtifactInput{
		Manifest: manifest, TargetPath: prepared.Path, Stdout: []byte{}, Stderr: []byte{}, World: worldPayloads,
	})
	if err != nil {
		t.Fatal(err)
	}
	if published.Manifest.Target.CapabilityManifest == nil {
		t.Fatal("published artifact omitted the linked capability manifest")
	}
	opened, err := artifact.OpenArtifact(published.Path)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := artifact.ReadPayload(opened, "target-capabilities.json", uint64(published.Manifest.Target.CapabilityManifest.Bytes))
	closeErr := opened.Close()
	if err != nil || closeErr != nil {
		t.Fatal(errors.Join(err, closeErr))
	}
	if string(payload) != string(prepared.CapabilityManifest.Payload) {
		t.Fatal("retained capability payload differs from the embedded record")
	}
	result, err := Replay(context.Background(), ReplaySpec{ArtifactPath: published.Path, VerifyOnly: true, ToolchainRoot: toolchain})
	if err != nil || !result.Verified {
		t.Fatalf("Replay() = %#v, %v", result, err)
	}
	if err := os.WriteFile(filepath.Join(published.Path, "target-capabilities.json"), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Replay(context.Background(), ReplaySpec{ArtifactPath: published.Path, VerifyOnly: true, ToolchainRoot: toolchain}); err == nil {
		t.Fatal("Replay() accepted a changed retained capability payload")
	}
}

func TestReadOnlyMountFailureArtifactReplaysAfterHostRemoval(t *testing.T) {
	toolchain := toolchainRoot(t)
	prepared, err := target.Prepare(context.Background(), target.Spec{
		Kind: target.KindGoRun, Source: "./io_ro_mount_failure", WorkingDir: filepath.Join("..", "internal", "gomadtool", "conformance", "testdata"),
		PreparationRoot: t.TempDir(), ToolchainRoot: toolchain,
	})
	if err != nil {
		t.Fatal(err)
	}
	profile := deterministicio.Default()
	const runnerBuild = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	frame, err := profile.BootstrapFrame(prepared, runnerBuild, 7)
	if err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "schema.sql"), []byte("select 1;\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	mappings := []readonlymount.Mapping{{Source: source, Target: "/mounted"}}
	limits := readonlymount.DefaultLimits()
	environment := []string{"GOMAD3_IO_PROFILE=" + profile.Name(), "GOMADSEED=7", "TZ=UTC"}
	observed, err := execution.Run(context.Background(), execution.Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestIOReplaySupervisorHelper"}, BootstrapCommand: []string{os.Args[0], "-test.run=TestIOReplayBootstrapHelper"},
		Command: prepared.Path, Argv0: prepared.Argv[0], Dir: t.TempDir(), Env: environment,
		ExecutionTimeout: 10 * time.Second, TerminateGrace: time.Second, OutputLimit: 1 << 20,
		World: execution.WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: 7},
		IO: &execution.IOCapability{Config: frame, Transcript: &execution.IOTranscriptCapability{Limit: 64 << 20},
			ReadOnlyMount: &execution.ReadOnlyMountCapability{Mappings: mappings, Limits: limits}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if observed.ExitCode != 2 || string(observed.Stdout.Bytes) != "select 1;\n" || len(observed.IOROMounts.Entries) != 1 {
		t.Fatalf("fixture result = %#v", observed)
	}
	mountArtifact, err := readonlymount.EncodeCapturedInputs(mappings, limits, observed.IOROMounts)
	if err != nil {
		t.Fatal(err)
	}
	exitCode := record.Uint64String(observed.ExitCode)
	recordedWorld, worldPayloads := record.NoneWorld()
	manifest := record.ExecutionRecord{
		SchemaVersion: record.SchemaVersion, ArtifactKind: record.ArtifactTargetFailure, CreatedAt: "2026-08-12T12:00:00Z", CampaignID: "mount-replay-test",
		SelectionOrdinal: 0, Seed: 7, ReplayMode: record.ReplayExact,
		Runner:    record.Runner{RecordContract: record.RecordContract, RunnerBuild: runnerBuild, HostOS: runtime.GOOS, HostArch: runtime.GOARCH},
		Toolchain: record.Toolchain{GoVersion: prepared.GoVersion, BuildKey: prepared.BuildKey, TargetGOOS: prepared.TargetGOOS, TargetGOARCH: prepared.TargetGOARCH},
		Target: record.Target{
			Kind: string(prepared.Kind), Source: prepared.Source, SHA256: record.SHA256(prepared.SHA256), Size: record.Uint64String(prepared.Size),
			Argv: prepared.Argv, BuildTags: prepared.BuildTags, Adapters: prepared.Adapters, Compatibility: prepared.Compatibility, BuildInfo: prepared.BuildInfo,
		},
		IOProfile: record.IOProfile{
			Name: profile.Name(), ImplementationSHA256: record.SHA256(profile.ImplementationSHA256()), Inventory: string(profile.Inventory()), InventorySHA256: record.SHA256(profile.InventorySHA256()),
			Transcript:     &record.IOTranscript{Schema: "gomad3.io-transcript/v1", File: "io/transcript.bin", SHA256: transcriptSHA256(observed.IOTranscript.SHA256), Bytes: record.Uint64String(len(observed.IOTranscript.Bytes)), Records: record.Uint64String(observed.IOTranscript.Records)},
			ReadOnlyMounts: pointerToReadOnlyMounts(replayRecordedCapturedInputs(mountArtifact.Manifest)),
		},
		Environment: []record.Environment{{Name: "GOMAD3_IO_PROFILE", Value: profile.Name()}, {Name: "GOMADSEED", Value: "7"}, {Name: "TZ", Value: "UTC"}},
		Limits: record.Limits{
			ExecutionTimeoutNanos: record.Uint64String(10 * time.Second), OverallTimeoutNanos: record.Uint64String(time.Minute), TerminateGraceNanos: record.Uint64String(time.Second),
			OutputBytes: 1 << 20, WorldTransitionBytes: 1 << 20, IOTranscriptBytes: 64 << 20,
		},
		World: recordedWorld, Outcome: record.Outcome{Domain: "target", Reason: "nonzero_exit", Termination: string(observed.Termination), ExitCode: &exitCode},
		Streams: record.Streams{Stdout: replayStream(observed.Stdout), Stderr: replayStream(observed.Stderr)},
		Host:    record.Host{StartedAt: "2026-08-12T12:00:00Z", FinishedAt: "2026-08-12T12:00:01Z", ElapsedNanos: record.Uint64String(time.Second)},
	}
	published, err := artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, artifact.ArtifactInput{
		Manifest: manifest, TargetPath: prepared.Path, Stdout: observed.Stdout.Bytes, Stderr: observed.Stderr.Bytes,
		IOTranscript: observed.IOTranscript.Bytes, ReadOnlyMounts: &mountArtifact, World: worldPayloads,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(source); err != nil {
		t.Fatal(err)
	}
	replayed, err := Replay(context.Background(), ReplaySpec{
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
	if os.Getenv("GOMAD3_PROCESS_SUPERVISOR") != "1" {
		t.Skip("supervisor subprocess only")
	}
	if err := execution.SupervisorMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	os.Exit(0)
}

func TestIOReplayBootstrapHelper(t *testing.T) {
	if os.Getenv("GOMAD3_TARGET_BOOTSTRAP") != "1" {
		t.Skip("target bootstrap subprocess only")
	}
	if err := execution.BootstrapMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	os.Exit(0)
}

func transcriptSHA256(value [sha256.Size]byte) record.SHA256 {
	return record.SHA256("sha256:" + hex.EncodeToString(value[:]))
}

func pointerToReadOnlyMounts(mounts record.ReadOnlyMounts) *record.ReadOnlyMounts {
	return &mounts
}

func replayRecordedCapturedInputs(manifest readonlymount.CapturedInputsManifest) record.ReadOnlyMounts {
	return record.ReadOnlyMounts{
		Schema: manifest.Schema, File: manifest.File, SHA256: record.SHA256(manifest.SHA256), Bytes: record.Uint64String(manifest.Bytes),
		Entries: record.Uint64String(manifest.Entries), NotExist: record.Uint64String(manifest.NotExist), TotalBytes: record.Uint64String(manifest.TotalBytes), Mappings: append([]string(nil), manifest.Mappings...),
		Limits: record.ReadOnlyMountLimits{
			PathBytes: record.Uint64String(manifest.Limits.PathBytes), Requests: record.Uint64String(manifest.Limits.Requests), Files: record.Uint64String(manifest.Limits.Files),
			DirectoryEntries: record.Uint64String(manifest.Limits.DirectoryEntries), SingleFileBytes: record.Uint64String(manifest.Limits.SingleFileBytes), TotalBytes: record.Uint64String(manifest.Limits.TotalBytes),
		},
	}
}
