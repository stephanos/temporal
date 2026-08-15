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

	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/campaignstore"
	"go.temporal.io/server/tools/gomadv3/runner/internal/execution"
	"go.temporal.io/server/tools/gomadv3/target"
)

func TestIOProfileFailureArtifactReplaysExactly(t *testing.T) {
	toolchain := toolchainRoot(t)
	prepared, err := target.Prepare(context.Background(), target.Spec{
		Kind: target.KindGoTest, Source: "./io_failure", WorkingDir: filepath.Join("..", "toolchain", "internal", "conformance", "testdata"),
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
	environment := []string{"GOMADSEED=7", "GOMADV3_IO_PROFILE=" + profile.Name(), "TZ=UTC"}
	observed, err := execution.Run(context.Background(), execution.Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestIOReplaySupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestIOReplayBootstrapHelper"},
		Command:           prepared.Path, Argv0: prepared.Argv[0], Args: prepared.Argv[1:], Dir: t.TempDir(), Env: environment,
		RunTimeout: 10 * time.Second, TerminateGrace: time.Second, OutputLimit: 1 << 20,
		World: execution.WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: 7},
		IO:    &execution.IOCapability{Config: frame, Transcript: &execution.IOTranscriptCapability{Limit: 64 << 20}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if observed.Termination != execution.TerminationExit || observed.ExitCode == 0 || !observed.IOTranscript.Complete {
		t.Fatalf("fixture result = %#v", observed)
	}
	exitCode := evidence.Uint64String(observed.ExitCode)
	recordedWorld, worldPayloads := evidence.NoneWorld()
	manifest := evidence.ExecutionRecord{
		SchemaVersion: evidence.SchemaVersion, ArtifactKind: evidence.ArtifactTargetFailure, CreatedAt: "2026-08-11T12:00:00Z", CampaignID: "io-replay-test",
		SelectionOrdinal: 0, Seed: 7, ReplayMode: evidence.ReplayExact,
		Runner:    evidence.Runner{RecordContract: evidence.RecordContract, RunnerBuild: runnerBuild, HostOS: runtime.GOOS, HostArch: runtime.GOARCH},
		Toolchain: evidence.Toolchain{GoVersion: prepared.GoVersion, BuildKey: prepared.BuildKey, TargetGOOS: prepared.TargetGOOS, TargetGOARCH: prepared.TargetGOARCH},
		Target: evidence.Target{
			Kind: string(prepared.Kind), Source: prepared.Source, SHA256: evidence.SHA256(prepared.SHA256), Size: evidence.Uint64String(prepared.Size),
			Argv: prepared.Argv, BuildTags: prepared.BuildTags, Adapters: prepared.Adapters, Compatibility: prepared.Compatibility, BuildInfo: prepared.BuildInfo,
		},
		IOProfile: evidence.IOProfile{
			Name: profile.Name(), ImplementationSHA256: evidence.SHA256(profile.ImplementationSHA256()), Inventory: string(profile.Inventory()), InventorySHA256: evidence.SHA256(profile.InventorySHA256()),
			Transcript: &evidence.IOTranscript{Schema: "gomadv3.io-transcript/v1", File: "io/transcript.bin", SHA256: transcriptSHA256(observed.IOTranscript.SHA256), Bytes: evidence.Uint64String(len(observed.IOTranscript.Bytes)), Records: evidence.Uint64String(observed.IOTranscript.Records)},
		},
		Environment: []evidence.Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "GOMADV3_IO_PROFILE", Value: profile.Name()}, {Name: "TZ", Value: "UTC"}},
		Limits: evidence.Limits{
			RunTimeoutNanos: evidence.Uint64String(10 * time.Second), OverallTimeoutNanos: evidence.Uint64String(time.Minute), TerminateGraceNanos: evidence.Uint64String(time.Second),
			OutputBytes: 1 << 20, WorldTransitionBytes: 1 << 20, IOTranscriptBytes: 64 << 20,
		},
		World:   recordedWorld,
		Outcome: evidence.Outcome{Domain: "target", Reason: "nonzero_exit", Termination: string(observed.Termination), ExitCode: &exitCode},
		Streams: evidence.Streams{Stdout: replayStream(observed.Stdout), Stderr: replayStream(observed.Stderr)},
		Host:    evidence.Host{StartedAt: "2026-08-11T12:00:00Z", FinishedAt: "2026-08-11T12:00:01Z", ElapsedNanos: evidence.Uint64String(time.Second)},
	}
	published, err := campaignstore.PublishArtifact(evidence.Store{Root: t.TempDir()}, campaignstore.ArtifactInput{
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
	worldRecord, worldPayloads := evidence.NoneWorld()
	exitCode := evidence.Uint64String(2)
	manifest := evidence.ExecutionRecord{
		SchemaVersion: evidence.SchemaVersion, ArtifactKind: evidence.ArtifactTargetFailure, CreatedAt: "2026-08-15T12:00:00Z", CampaignID: "linked-replay-test",
		SelectionOrdinal: 0, Seed: 7, ReplayMode: evidence.ReplayExact,
		Runner:    evidence.Runner{RecordContract: evidence.RecordContract, RunnerBuild: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", HostOS: runtime.GOOS, HostArch: runtime.GOARCH},
		Toolchain: prepared.RecordToolchain(), Target: prepared.RecordTarget(),
		IOProfile:   evidence.IOProfile{Name: profile.Name(), ImplementationSHA256: evidence.SHA256(profile.ImplementationSHA256()), Inventory: string(profile.Inventory()), InventorySHA256: evidence.SHA256(profile.InventorySHA256())},
		Environment: []evidence.Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "GOMADV3_IO_PROFILE", Value: profile.Name()}, {Name: "TZ", Value: "UTC"}},
		Limits:      evidence.Limits{RunTimeoutNanos: evidence.Uint64String(time.Second), OverallTimeoutNanos: evidence.Uint64String(time.Minute), OutputBytes: 64, WorldTransitionBytes: 64},
		World:       worldRecord, Outcome: evidence.Outcome{Domain: "target", Reason: "nonzero_exit", Termination: "exit", ExitCode: &exitCode},
		Streams: evidence.Streams{
			Stdout: evidence.Stream{FullSHA256: evidence.HashBytes(nil)},
			Stderr: evidence.Stream{FullSHA256: evidence.HashBytes(nil)},
		},
		Host: evidence.Host{StartedAt: "2026-08-15T12:00:00Z", FinishedAt: "2026-08-15T12:00:01Z", ElapsedNanos: evidence.Uint64String(time.Second)},
	}
	published, err := campaignstore.PublishArtifact(evidence.Store{Root: t.TempDir()}, campaignstore.ArtifactInput{
		Manifest: manifest, TargetPath: prepared.Path, Stdout: []byte{}, Stderr: []byte{}, World: worldPayloads,
	})
	if err != nil {
		t.Fatal(err)
	}
	if published.Manifest.Target.CapabilityManifest == nil {
		t.Fatal("published artifact omitted the linked capability manifest")
	}
	opened, err := evidence.OpenArtifact(published.Path)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := evidence.ReadPayload(opened, "target-capabilities.json", uint64(published.Manifest.Target.CapabilityManifest.Bytes))
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
		Kind: target.KindGoRun, Source: "./io_ro_mount_failure", WorkingDir: filepath.Join("..", "toolchain", "internal", "conformance", "testdata"),
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
	mappings := []deterministicio.Mapping{{Source: source, Target: "/mounted"}}
	limits := deterministicio.DefaultLimits()
	environment := []string{"GOMADSEED=7", "GOMADV3_IO_PROFILE=" + profile.Name(), "TZ=UTC"}
	observed, err := execution.Run(context.Background(), execution.Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestIOReplaySupervisorHelper"}, BootstrapCommand: []string{os.Args[0], "-test.run=TestIOReplayBootstrapHelper"},
		Command: prepared.Path, Argv0: prepared.Argv[0], Dir: t.TempDir(), Env: environment,
		RunTimeout: 10 * time.Second, TerminateGrace: time.Second, OutputLimit: 1 << 20,
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
	mountArtifact, err := deterministicio.EncodeCapturedInputs(mappings, limits, observed.IOROMounts)
	if err != nil {
		t.Fatal(err)
	}
	exitCode := evidence.Uint64String(observed.ExitCode)
	recordedWorld, worldPayloads := evidence.NoneWorld()
	manifest := evidence.ExecutionRecord{
		SchemaVersion: evidence.SchemaVersion, ArtifactKind: evidence.ArtifactTargetFailure, CreatedAt: "2026-08-12T12:00:00Z", CampaignID: "mount-replay-test",
		SelectionOrdinal: 0, Seed: 7, ReplayMode: evidence.ReplayExact,
		Runner:    evidence.Runner{RecordContract: evidence.RecordContract, RunnerBuild: runnerBuild, HostOS: runtime.GOOS, HostArch: runtime.GOARCH},
		Toolchain: evidence.Toolchain{GoVersion: prepared.GoVersion, BuildKey: prepared.BuildKey, TargetGOOS: prepared.TargetGOOS, TargetGOARCH: prepared.TargetGOARCH},
		Target: evidence.Target{
			Kind: string(prepared.Kind), Source: prepared.Source, SHA256: evidence.SHA256(prepared.SHA256), Size: evidence.Uint64String(prepared.Size),
			Argv: prepared.Argv, BuildTags: prepared.BuildTags, Adapters: prepared.Adapters, Compatibility: prepared.Compatibility, BuildInfo: prepared.BuildInfo,
		},
		IOProfile: evidence.IOProfile{
			Name: profile.Name(), ImplementationSHA256: evidence.SHA256(profile.ImplementationSHA256()), Inventory: string(profile.Inventory()), InventorySHA256: evidence.SHA256(profile.InventorySHA256()),
			Transcript:     &evidence.IOTranscript{Schema: "gomadv3.io-transcript/v1", File: "io/transcript.bin", SHA256: transcriptSHA256(observed.IOTranscript.SHA256), Bytes: evidence.Uint64String(len(observed.IOTranscript.Bytes)), Records: evidence.Uint64String(observed.IOTranscript.Records)},
			ReadOnlyMounts: pointerToReadOnlyMounts(replayRecordedCapturedInputs(mountArtifact.Manifest)),
		},
		Environment: []evidence.Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "GOMADV3_IO_PROFILE", Value: profile.Name()}, {Name: "TZ", Value: "UTC"}},
		Limits: evidence.Limits{
			RunTimeoutNanos: evidence.Uint64String(10 * time.Second), OverallTimeoutNanos: evidence.Uint64String(time.Minute), TerminateGraceNanos: evidence.Uint64String(time.Second),
			OutputBytes: 1 << 20, WorldTransitionBytes: 1 << 20, IOTranscriptBytes: 64 << 20,
		},
		World: recordedWorld, Outcome: evidence.Outcome{Domain: "target", Reason: "nonzero_exit", Termination: string(observed.Termination), ExitCode: &exitCode},
		Streams: evidence.Streams{Stdout: replayStream(observed.Stdout), Stderr: replayStream(observed.Stderr)},
		Host:    evidence.Host{StartedAt: "2026-08-12T12:00:00Z", FinishedAt: "2026-08-12T12:00:01Z", ElapsedNanos: evidence.Uint64String(time.Second)},
	}
	published, err := campaignstore.PublishArtifact(evidence.Store{Root: t.TempDir()}, campaignstore.ArtifactInput{
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
	if os.Getenv("GOMADV3_PROCESS_SUPERVISOR") != "1" {
		t.Skip("supervisor subprocess only")
	}
	if err := execution.SupervisorMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	os.Exit(0)
}

func TestIOReplayBootstrapHelper(t *testing.T) {
	if os.Getenv("GOMADV3_TARGET_BOOTSTRAP") != "1" {
		t.Skip("target bootstrap subprocess only")
	}
	if err := execution.BootstrapMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	os.Exit(0)
}

func transcriptSHA256(value [sha256.Size]byte) evidence.SHA256 {
	return evidence.SHA256("sha256:" + hex.EncodeToString(value[:]))
}

func pointerToReadOnlyMounts(mounts evidence.ReadOnlyMounts) *evidence.ReadOnlyMounts {
	return &mounts
}

func replayRecordedCapturedInputs(manifest deterministicio.CapturedInputsManifest) evidence.ReadOnlyMounts {
	return evidence.ReadOnlyMounts{
		Schema: manifest.Schema, File: manifest.File, SHA256: evidence.SHA256(manifest.SHA256), Bytes: evidence.Uint64String(manifest.Bytes),
		Entries: evidence.Uint64String(manifest.Entries), NotExist: evidence.Uint64String(manifest.NotExist), TotalBytes: evidence.Uint64String(manifest.TotalBytes), Mappings: append([]string(nil), manifest.Mappings...),
		Limits: evidence.ReadOnlyMountLimits{
			PathBytes: evidence.Uint64String(manifest.Limits.PathBytes), Requests: evidence.Uint64String(manifest.Limits.Requests), Files: evidence.Uint64String(manifest.Limits.Files),
			DirectoryEntries: evidence.Uint64String(manifest.Limits.DirectoryEntries), SingleFileBytes: evidence.Uint64String(manifest.Limits.SingleFileBytes), TotalBytes: evidence.Uint64String(manifest.Limits.TotalBytes),
		},
	}
}
