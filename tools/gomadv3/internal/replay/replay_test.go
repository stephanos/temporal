package replay

import (
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"sort"
	"sync"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/artifact"
	"go.temporal.io/server/tools/gomadv3/internal/choicewire"
	"go.temporal.io/server/tools/gomadv3/internal/ioprofile"
	"go.temporal.io/server/tools/gomadv3/internal/iowire"
	"go.temporal.io/server/tools/gomadv3/internal/process"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/target"
	"go.temporal.io/server/tools/gomadv3/internal/worldrecord"
	"go.temporal.io/server/tools/gomadv3/world"
	worldchild "go.temporal.io/server/tools/gomadv3/world/child"
)

func TestReplayVerifiesThenRunsStoredTargetWithoutRebuilding(t *testing.T) {
	artifactPath, expected := replayArtifact(t)
	movedRoot := t.TempDir()
	movedPath := filepath.Join(movedRoot, "moved-artifact")
	if err := os.Rename(artifactPath, movedPath); err != nil {
		t.Fatal(err)
	}
	executor := &fakeReplayExecutor{result: expected}
	result, err := Replay(context.Background(), Config{
		ArtifactPath: movedPath, ToolchainRoot: toolchainRoot(t), SupervisorCommand: []string{"unused"}, Executor: executor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Match || result.Divergence != "" || executor.calls != 1 {
		t.Fatalf("replay result = %#v, calls = %d", result, executor.calls)
	}
	if executor.request.Command == filepath.Join(movedPath, "target") || filepath.Base(executor.request.Command) != "target" || filepath.Dir(executor.request.Command) != executor.request.Dir || executor.request.Env[0] != "GOMADSEED=7" || executor.request.Env[1] != "GOMADV3_IO_PROFILE=gomadv3-deterministic/v1" || executor.request.Env[2] != "TZ=UTC" {
		t.Fatalf("replay request = %#v", executor.request)
	}
}

func TestReplayEnvironmentExcludesChoiceControlVariables(t *testing.T) {
	environment := replayEnvironment([]record.Environment{
		{Name: "GOMADSEED", Value: "7"},
		{Name: "GOMADV3_CHOICE_PROFILE", Value: choicewire.Profile},
		{Name: "TZ", Value: "UTC"},
	})
	if !reflect.DeepEqual(environment, []string{"GOMADSEED=7", "TZ=UTC"}) {
		t.Fatalf("replay environment = %#v", environment)
	}
}

func TestReplayVerifyOnlyDoesNotStartTarget(t *testing.T) {
	artifactPath, _ := replayArtifact(t)
	executor := &fakeReplayExecutor{}
	result, err := Replay(context.Background(), Config{
		ArtifactPath: artifactPath, VerifyOnly: true, ToolchainRoot: toolchainRoot(t), SupervisorCommand: []string{"unused"}, Executor: executor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Verified || result.Match || executor.calls != 0 {
		t.Fatalf("verify result = %#v, calls = %d", result, executor.calls)
	}
}

func TestReplayRejectsUnavailableCompatibilityPackBeforeTargetStart(t *testing.T) {
	artifactPath, _ := publishReplayArtifactWithCompatibility(t, []record.CompatibilityPack{{
		ID: "unknown-pack", SHA256: record.HashBytes([]byte("unknown pack")),
	}})
	executor := &fakeReplayExecutor{}
	_, err := Replay(context.Background(), Config{
		ArtifactPath: artifactPath, VerifyOnly: true, ToolchainRoot: toolchainRoot(t), SupervisorCommand: []string{"unused"}, Executor: executor,
	})
	if err == nil || executor.calls != 0 {
		t.Fatalf("Replay() error = %v, calls = %d", err, executor.calls)
	}
}

func TestReplayRejectsChangedPayloadBeforeTargetStart(t *testing.T) {
	artifactPath, _ := replayArtifact(t)
	if err := os.Chmod(filepath.Join(artifactPath, "stdout"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifactPath, "stdout"), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	executor := &fakeReplayExecutor{}
	_, err := Replay(context.Background(), Config{
		ArtifactPath: artifactPath, ToolchainRoot: toolchainRoot(t), SupervisorCommand: []string{"unused"}, Executor: executor,
	})
	if err == nil || executor.calls != 0 {
		t.Fatalf("Replay() error = %v, calls = %d", err, executor.calls)
	}
}

func TestReplayRejectsFirstDivergentWorldTransitionBeforeTargetMutation(t *testing.T) {
	core, err := world.New(world.Config{Seed: 7, Limits: world.Limits{MaxRequests: 10, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 10, MaxPayloadBytes: 1024, MaxStringBytes: 64}})
	if err != nil {
		t.Fatal(err)
	}
	initial := core.Snapshot()
	if _, err := core.Register(world.Request{Kind: "read", Resource: world.ResourceID{Adapter: "memory", Kind: "cell", Key: "expected"}}); err != nil {
		t.Fatal(err)
	}
	bundle, err := worldrecord.ComposeRecording(world.Recording{
		Initial: initial,
		Final:   core.Snapshot(),
		Terminal: world.Terminal{
			Kind:   world.TerminalReplayDivergence,
			Detail: "recorded replay divergence",
		},
	}, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	artifactPath, _ := publishReplayArtifactForTarget(t, &bundle, replayArtifactTarget{
		Argv:          []string{"gomadv3-target", "-test.run=^TestWorldReplayTarget$"},
		Environment:   []record.Environment{{Name: "GOMADV3_WORLD_REPLAY_TARGET", Value: "diverge"}},
		OutcomeReason: "world_replay_divergence",
		IOTranscript:  recordReplayIOTranscript(t),
	})
	result, err := Replay(context.Background(), Config{
		ArtifactPath: artifactPath, ToolchainRoot: toolchainRoot(t),
		SupervisorCommand: []string{os.Args[0], "-test.run=TestIOReplaySupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestIOReplayBootstrapHelper"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Divergence != "world.terminal" {
		t.Fatalf("divergent transition reached target-visible mutation: %#v", result)
	}
}

func TestReplayExecutesMatchingWorldPlanThroughChildTransport(t *testing.T) {
	core, err := world.New(world.Config{Seed: 7, Limits: world.Limits{MaxRequests: 10, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 10, MaxPayloadBytes: 1024, MaxStringBytes: 64}})
	if err != nil {
		t.Fatal(err)
	}
	initial := core.Snapshot()
	if _, err := core.Register(world.Request{Kind: "read", Resource: world.ResourceID{Adapter: "memory", Kind: "cell", Key: "expected"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := core.Quiesce(); err != nil {
		t.Fatal(err)
	}
	bundle, err := worldrecord.ComposeRecording(world.Recording{Initial: initial, Final: core.Snapshot(), Terminal: world.Terminal{Kind: world.TerminalDeadlock}}, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	artifactPath, _ := publishReplayArtifactForTarget(t, &bundle, replayArtifactTarget{
		Argv:          []string{"gomadv3-target", "-test.run=^TestWorldReplayTarget$"},
		Environment:   []record.Environment{{Name: "GOMADV3_WORLD_REPLAY_TARGET", Value: "match"}},
		OutcomeReason: "world_deadlock",
		IOTranscript:  recordReplayIOTranscript(t),
	})
	result, err := Replay(context.Background(), Config{
		ArtifactPath: artifactPath, ToolchainRoot: toolchainRoot(t),
		SupervisorCommand: []string{os.Args[0], "-test.run=TestIOReplaySupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestIOReplayBootstrapHelper"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Match || result.Divergence != "" {
		t.Fatalf("replay result = %#v", result)
	}
}

func TestWorldReplayTarget(t *testing.T) {
	mode := os.Getenv("GOMADV3_WORLD_REPLAY_TARGET")
	if mode == "" {
		t.Skip("World replay target subprocess only")
	}
	core, err := world.New(world.Config{Seed: 7, Limits: world.Limits{MaxRequests: 10, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 10, MaxPayloadBytes: 1024, MaxStringBytes: 64}})
	if err != nil {
		os.Exit(10) //nolint:revive // This subprocess helper reports failure by exit status.
	}
	session, err := worldchild.Open(core)
	if err != nil {
		os.Exit(11) //nolint:revive // This subprocess helper reports failure by exit status.
	}
	key := "expected"
	stdout := "recorded stdout"
	if mode == "diverge" {
		key = "actual"
	}
	_, transitionErr := session.World().Register(world.Request{Kind: "read", Resource: world.ResourceID{Adapter: "memory", Kind: "cell", Key: key}})
	if transitionErr == nil {
		if mode == "diverge" {
			stdout = "target-visible mutation"
			transitionErr = fmt.Errorf("%w: replay plan was not attached", world.ErrReplayDivergence)
		} else {
			_, transitionErr = session.World().Quiesce()
		}
	}
	if transitionErr != nil {
		if err := session.FinishError(transitionErr); err != nil {
			os.Exit(13) //nolint:revive // This subprocess helper reports failure by exit status.
		}
	} else if err := session.Finish(); err != nil {
		os.Exit(14) //nolint:revive // This subprocess helper reports failure by exit status.
	}
	if _, err := net.Dial("tcp", "127.0.0.1:1"); err == nil {
		os.Exit(15) //nolint:revive // This subprocess helper reports failure by exit status.
	}
	if _, err := os.Stdout.WriteString(stdout); err != nil {
		os.Exit(16) //nolint:revive // This subprocess helper reports failure by exit status.
	}
	if _, err := os.Stderr.WriteString("recorded stderr"); err != nil {
		os.Exit(17) //nolint:revive // This subprocess helper reports failure by exit status.
	}
	os.Exit(2) //nolint:revive // This subprocess target intentionally returns a nonzero status.
}

func recordReplayIOTranscript(t *testing.T) []byte {
	t.Helper()
	targetPath, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	targetBytes, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	argv := []string{"gomadv3-target", "-test.run=^TestWorldReplayTarget$"}
	profile := ioprofile.Default()
	frame, err := profile.BootstrapFrame(target.Prepared{SHA256: string(record.HashBytes(targetBytes)), Argv: argv}, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 7)
	if err != nil {
		t.Fatal(err)
	}
	result, err := process.Run(context.Background(), process.Request{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestIOReplaySupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestIOReplayBootstrapHelper"},
		Command:           targetPath, Argv0: argv[0], Args: argv[1:], Dir: t.TempDir(),
		Env:        []string{"GOMADSEED=7", "GOMADV3_IO_PROFILE=" + profile.Name(), "GOMADV3_WORLD_REPLAY_TARGET=match", "TZ=UTC"},
		RunTimeout: 5 * time.Second, TerminateGrace: 100 * time.Millisecond, OutputLimit: 64,
		World: process.WorldCapability{RecordLimit: world.MaximumRecordingBytes, TransitionLimit: 1 << 20, Seed: 7},
		IO:    &process.IOCapability{Config: frame, Transcript: &process.IOTranscriptCapability{Limit: 64 << 20}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IOTranscript.Complete || result.IOTranscript.Records == 0 {
		t.Fatalf("recorded I/O transcript = %#v", result.IOTranscript)
	}
	return result.IOTranscript.Bytes
}

func TestReplayReportsFirstObservableDivergence(t *testing.T) {
	artifactPath, expected := replayArtifact(t)
	expected.Stdout = replayOutput("different stdout")
	executor := &fakeReplayExecutor{result: expected}
	result, err := Replay(context.Background(), Config{
		ArtifactPath: artifactPath, ToolchainRoot: toolchainRoot(t), SupervisorCommand: []string{"unused"}, Executor: executor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Match || result.Divergence != "stdout.full_sha256" {
		t.Fatalf("replay result = %#v", result)
	}
}

func TestReplayReportsIOTranscriptOrdinalBeforeOutcome(t *testing.T) {
	ordinal := uint64(4)
	observed := process.Result{IOTranscript: process.IOTranscript{ReplayDivergence: &ordinal}}
	if divergence := firstDivergence(record.Manifest{}, observed, nil); divergence != "io_profile.transcript.ordinal[4]" {
		t.Fatalf("firstDivergence() = %q", divergence)
	}
}

func TestReplayReportsFinalChoiceTraceMismatch(t *testing.T) {
	expectedDigest := sha256.Sum256([]byte("expected"))
	observedDigest := sha256.Sum256([]byte("observed"))
	manifest := record.Manifest{
		Outcome:       record.Outcome{Domain: "success", Reason: "success", Termination: "exit"},
		Streams:       record.Streams{Stdout: record.Stream{FullSHA256: record.SHA256FromSum(sha256.Sum256(nil))}, Stderr: record.Stream{FullSHA256: record.SHA256FromSum(sha256.Sum256(nil))}},
		ChoiceProfile: &record.ChoiceProfile{Name: choicewire.Profile, Trace: record.ChoiceTrace{SHA256: record.SHA256FromSum(expectedDigest), Records: 1, BranchingRecords: 1, Limit: 1 << 20}},
	}
	observed := process.Result{
		Termination: process.TerminationExit,
		Stdout:      replayOutput(""), Stderr: replayOutput(""),
		ChoiceTrace: process.ChoiceTrace{Profile: choicewire.Profile, Limit: 1 << 20, Trace: choicewire.Trace{SHA256: observedDigest, Summary: choicewire.Summary{Records: 1, Branching: 1, Terminal: choicewire.TerminalComplete}}},
	}
	if divergence := firstDivergence(manifest, observed, nil); divergence != "choice_profile.trace.sha256" {
		t.Fatalf("firstDivergence() = %q", divergence)
	}
}

func TestReplayRejectsUnexpectedWorldRecord(t *testing.T) {
	artifactPath, expected := replayArtifact(t)
	core, err := world.New(world.Config{Seed: 7, Limits: world.Limits{MaxRequests: 10, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 10, MaxPayloadBytes: 1024, MaxStringBytes: 64}})
	if err != nil {
		t.Fatal(err)
	}
	initial := core.Snapshot()
	if _, err := core.Quiesce(); err != nil {
		t.Fatal(err)
	}
	expected.WorldRecord, err = world.EncodeRecording(world.Recording{Initial: initial, Final: core.Snapshot(), Terminal: world.Terminal{Kind: world.TerminalIdle}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := Replay(context.Background(), Config{
		ArtifactPath: artifactPath, ToolchainRoot: toolchainRoot(t), SupervisorCommand: []string{"unused"}, Executor: &fakeReplayExecutor{result: expected},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Match || result.Divergence != "world.record" {
		t.Fatalf("replay result = %#v", result)
	}
}

func TestReplayPreflightValidatesConnectedWorldRecord(t *testing.T) {
	core, err := world.New(world.Config{Seed: 7, Limits: world.Limits{MaxRequests: 10, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 10, MaxPayloadBytes: 1024, MaxStringBytes: 64}})
	if err != nil {
		t.Fatal(err)
	}
	initial := core.Snapshot()
	if _, err := core.Quiesce(); err != nil {
		t.Fatal(err)
	}
	bundle, err := worldrecord.Compose(initial, core.Snapshot(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	artifactPath, expected := publishReplayArtifact(t, &bundle)
	result, err := Replay(context.Background(), Config{
		ArtifactPath: artifactPath, ToolchainRoot: toolchainRoot(t), SupervisorCommand: []string{"unused"}, Executor: &fakeReplayExecutor{result: expected},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Match {
		t.Fatalf("connected World replay = %#v", result)
	}
	changed := bundle
	changed.Manifest.Final.SemanticDigest = record.HashBytes([]byte("changed semantic digest"))
	changedPath, _ := publishReplayArtifact(t, &changed)
	executor := &fakeReplayExecutor{}
	if _, err := Replay(context.Background(), Config{
		ArtifactPath: changedPath, ToolchainRoot: toolchainRoot(t), SupervisorCommand: []string{"unused"}, Executor: executor,
	}); err == nil || executor.calls != 0 {
		t.Fatalf("changed World Replay() error = %v, calls = %d", err, executor.calls)
	}
}

type fakeReplayExecutor struct {
	mu      sync.Mutex
	calls   int
	request process.Request
	result  process.Result
}

func (executor *fakeReplayExecutor) Run(_ context.Context, request process.Request) (process.Result, error) {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	executor.calls++
	executor.request = request
	return executor.result, nil
}

func replayArtifact(t *testing.T) (string, process.Result) {
	return publishReplayArtifact(t, nil)
}

func publishReplayArtifact(t *testing.T, connected *worldrecord.Bundle) (string, process.Result) {
	return publishReplayArtifactWithWorldAndCompatibility(t, connected, []record.CompatibilityPack{})
}

func publishReplayArtifactWithCompatibility(t *testing.T, compatibility []record.CompatibilityPack) (string, process.Result) {
	return publishReplayArtifactWithWorldAndCompatibility(t, nil, compatibility)
}

func publishReplayArtifactWithWorldAndCompatibility(t *testing.T, connected *worldrecord.Bundle, compatibility []record.CompatibilityPack) (string, process.Result) {
	return publishReplayArtifactForTargetAndCompatibility(t, connected, compatibility, replayArtifactTarget{})
}

type replayArtifactTarget struct {
	Argv          []string
	Environment   []record.Environment
	OutcomeReason string
	IOTranscript  []byte
}

func publishReplayArtifactForTarget(t *testing.T, connected *worldrecord.Bundle, replayTarget replayArtifactTarget) (string, process.Result) {
	return publishReplayArtifactForTargetAndCompatibility(t, connected, []record.CompatibilityPack{}, replayTarget)
}

func publishReplayArtifactForTargetAndCompatibility(t *testing.T, connected *worldrecord.Bundle, compatibility []record.CompatibilityPack, replayTarget replayArtifactTarget) (string, process.Result) {
	t.Helper()
	targetPath, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	build, err := buildinfo.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	targetBytes, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := target.ReadToolchainIdentity(toolchainRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	stdout := replayOutput("recorded stdout")
	stderr := replayOutput("recorded stderr")
	ioTranscript := replayTarget.IOTranscript
	ioTranscriptSHA256 := sha256.Sum256(ioTranscript)
	profile := ioprofile.Default()
	exitCode := record.Uint64String(2)
	targetArgv := replayTarget.Argv
	if len(targetArgv) == 0 {
		targetArgv = []string{"gomadv3-target", "-test.run=none"}
	}
	environment := append([]record.Environment{
		{Name: "GOMADSEED", Value: "7"},
		{Name: "GOMADV3_IO_PROFILE", Value: profile.Name()},
		{Name: "TZ", Value: "UTC"},
	}, replayTarget.Environment...)
	sort.Slice(environment, func(i, j int) bool { return environment[i].Name < environment[j].Name })
	outcomeReason := replayTarget.OutcomeReason
	if outcomeReason == "" {
		outcomeReason = "nonzero_exit"
	}
	recordedWorld, payloads := record.NoneWorld()
	if connected != nil {
		recordedWorld = connected.Manifest
		payloads = connected.Payloads
	}
	input := artifact.Input{
		Manifest: record.Manifest{
			SchemaVersion: record.SchemaVersion, ArtifactKind: record.ArtifactTargetFailure, CreatedAt: "2026-08-10T12:00:00Z", BatchID: "replay-test",
			SelectionOrdinal: 0, Seed: 7, ReplayMode: record.ReplayExact,
			Runner:    record.Runner{RecordContract: record.RecordContract, RunnerBuild: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", HostOS: runtime.GOOS, HostArch: runtime.GOARCH},
			Toolchain: record.Toolchain{GoVersion: identity.GoVersion, BuildKey: identity.BuildKey, TargetGOOS: identity.TargetGOOS, TargetGOARCH: identity.TargetGOARCH},
			Target: record.Target{
				Kind: "go-test", Source: "replay fixture", SHA256: record.HashBytes(targetBytes), Size: record.Uint64String(len(targetBytes)),
				Argv: targetArgv, BuildTags: []string{"gomad_fixture"}, Adapters: []record.TargetAdapter{}, Compatibility: compatibility, BuildInfo: projectTestBuildInfo(build),
			},
			IOProfile: record.IOProfile{
				Name: profile.Name(), ImplementationSHA256: profile.ImplementationSHA256(), Inventory: string(profile.Inventory()), InventorySHA256: profile.InventorySHA256(),
				Transcript: &record.IOTranscript{Schema: "gomadv3.io-transcript/v1", SHA256: record.SHA256FromSum(ioTranscriptSHA256), Bytes: record.Uint64String(len(ioTranscript)), Records: record.Uint64String(len(ioTranscript) / iowire.TranscriptRecordBytes)},
			},
			Environment: environment,
			Limits:      record.Limits{RunTimeoutNanos: record.Uint64String(5 * time.Second), OverallTimeoutNanos: record.Uint64String(10 * time.Second), TerminateGraceNanos: record.Uint64String(100 * time.Millisecond), OutputBytes: 64, WorldTransitionBytes: 1 << 20, IOTranscriptBytes: 64 << 20},
			World:       recordedWorld,
			Outcome:     record.Outcome{Domain: "target", Reason: outcomeReason, Termination: "exit", ExitCode: &exitCode},
			Streams:     record.Streams{Stdout: replayStream(stdout), Stderr: replayStream(stderr)},
			Host:        record.Host{StartedAt: "2026-08-10T12:00:00Z", FinishedAt: "2026-08-10T12:00:01Z", ElapsedNanos: record.Uint64String(time.Second)},
		},
		TargetPath: targetPath, Stdout: stdout.Bytes, Stderr: stderr.Bytes, IOTranscript: ioTranscript, World: payloads,
	}
	published, err := (artifact.Store{Root: t.TempDir()}).Publish(input)
	if err != nil {
		t.Fatal(err)
	}
	result := process.Result{Termination: process.TerminationExit, ExitCode: 2, GroupGone: true, Stdout: stdout, Stderr: stderr, IOTranscript: process.IOTranscript{SHA256: ioTranscriptSHA256, Records: uint64(len(ioTranscript) / iowire.TranscriptRecordBytes), Complete: true}}
	if connected != nil {
		initial, decodeErr := world.DecodeSnapshot(connected.Payloads.Initial)
		if decodeErr != nil {
			t.Fatal(decodeErr)
		}
		final, decodeErr := world.DecodeSnapshot(connected.Payloads.Final)
		if decodeErr != nil {
			t.Fatal(decodeErr)
		}
		result.WorldRecord, decodeErr = world.EncodeRecording(world.Recording{
			Initial: initial, Final: final,
			Terminal: world.Terminal{Kind: world.TerminalKind(connected.Manifest.Terminal.Kind), Detail: connected.Manifest.Terminal.Detail},
		})
		if decodeErr != nil {
			t.Fatal(decodeErr)
		}
	}
	return published.Path, result
}

func replayOutput(value string) process.Output {
	data := []byte(value)
	digest := sha256.Sum256(data)
	return process.Output{Bytes: data, FullSHA256: digest, RetainedSHA256: digest, TotalBytes: uint64(len(data)), RetainedBytes: uint64(len(data))}
}

func replayStream(output process.Output) record.Stream {
	return record.Stream{
		FullSHA256: record.SHA256(fmt.Sprintf("sha256:%x", output.FullSHA256)), TotalBytes: record.Uint64String(output.TotalBytes),
		RetainedBytes: record.Uint64String(output.RetainedBytes), DiscardedBytes: record.Uint64String(output.DiscardedBytes), Truncated: output.Truncated,
	}
}

func projectTestBuildInfo(info *debug.BuildInfo) record.BuildInfo {
	settings := make([]record.BuildSetting, len(info.Settings))
	for index, setting := range info.Settings {
		settings[index] = record.BuildSetting{Key: setting.Key, Value: setting.Value}
	}
	sort.Slice(settings, func(i, j int) bool { return settings[i].Key < settings[j].Key })
	return record.BuildInfo{GoVersion: info.GoVersion, Path: info.Path, MainModule: info.Main.Path, Settings: settings}
}

func toolchainRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	return root
}
