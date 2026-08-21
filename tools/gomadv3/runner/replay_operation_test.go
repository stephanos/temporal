package runner

import (
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/binary"
	"encoding/json"
	"errors"
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

	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/campaignstore"
	"go.temporal.io/server/tools/gomadv3/runner/internal/combinedfrontier"
	"go.temporal.io/server/tools/gomadv3/runner/internal/execution"
	"go.temporal.io/server/tools/gomadv3/runner/internal/simulationexploration"
	"go.temporal.io/server/tools/gomadv3/target"
	"go.temporal.io/server/tools/gomadv3/world"
	worldtarget "go.temporal.io/server/tools/gomadv3/world/target"
)

func TestReplayVerifiesThenRunsStoredTargetWithoutRebuilding(t *testing.T) {
	artifactPath, expected := replayArtifact(t)
	movedRoot := t.TempDir()
	movedPath := filepath.Join(movedRoot, "moved-artifact")
	if err := os.Rename(artifactPath, movedPath); err != nil {
		t.Fatal(err)
	}
	executor := &fakeReplayExecutor{result: expected}
	result, err := Replay(context.Background(), ReplaySpec{
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
	environment := replayEnvironment([]evidence.Environment{
		{Name: "GOMADSEED", Value: "7"},
		{Name: "GOMADV3_CHOICE_PROFILE", Value: choice.Profile},
		{Name: "TZ", Value: "UTC"},
	})
	if !reflect.DeepEqual(environment, []string{"GOMADSEED=7", "TZ=UTC"}) {
		t.Fatalf("replay environment = %#v", environment)
	}
}

func TestReplayAutomaticallySuppliesExactChoiceTape(t *testing.T) {
	artifactPath, expected := publishReplayArtifactForTarget(t, nil, replayArtifactTarget{Choices: true})
	executor := &fakeReplayExecutor{result: expected}
	result, err := Replay(context.Background(), ReplaySpec{
		ArtifactPath: artifactPath, ToolchainRoot: toolchainRoot(t), SupervisorCommand: []string{"unused"}, Executor: executor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Match || result.ChoiceReplayStatus != ChoiceReplayExact {
		t.Fatalf("replay result = %#v", result)
	}
	if executor.request.Choice == nil || executor.request.Choice.Mode != choice.ModeReplay || executor.request.Choice.ReplayPlan == nil || len(executor.request.Choice.ReplayPlan.Decisions) != 1 {
		t.Fatalf("replay choice capability = %#v", executor.request.Choice)
	}
}

func TestReplayAutomaticallySuppliesExactSimulationExplorationTape(t *testing.T) {
	artifactPath, expected := publishReplayArtifactForTarget(t, nil, replayArtifactTarget{Choices: true, Simulation: true})
	opened, err := evidence.OpenArtifact(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	profile := opened.Manifest.SimulationProfile
	if profile == nil {
		t.Fatal("replay fixture omitted simulation exploration evidence")
	}
	plan, err := evidence.ReadPayload(opened, profile.Plan.File, uint64(profile.Plan.Bytes))
	if err != nil {
		t.Fatal(err)
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
	executor := &fakeReplayExecutor{result: expected}
	result, err := Replay(context.Background(), ReplaySpec{
		ArtifactPath: artifactPath, ToolchainRoot: toolchainRoot(t), SupervisorCommand: []string{"unused"}, Executor: executor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Match || result.Divergence != "" {
		t.Fatalf("replay result = %#v", result)
	}
	if executor.request.Simulation == nil || executor.request.Simulation.Role != execution.SimulationRoleCoordinator || !reflect.DeepEqual(executor.request.Simulation.ExplorationPlan, plan) || executor.request.Simulation.ExplorationRecordLimit != uint64(profile.Record.Limit) || executor.request.Simulation.ExplorationRecordCount != 1 {
		t.Fatalf("replay simulation capability = %#v", executor.request.Simulation)
	}
}

func TestReplayReportsChangedSimulationExplorationRecord(t *testing.T) {
	artifactPath, observed := publishReplayArtifactForTarget(t, nil, replayArtifactTarget{Choices: true, Simulation: true})
	observed.SimulationRecords = [][]byte{[]byte("changed simulation record")}
	result, err := Replay(context.Background(), ReplaySpec{
		ArtifactPath: artifactPath, ToolchainRoot: toolchainRoot(t), SupervisorCommand: []string{"unused"}, Executor: &fakeReplayExecutor{result: observed},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Match || result.Divergence != "simulation_profile.record.sha256" {
		t.Fatalf("replay result = %#v", result)
	}
}

func TestReplayReportsTypedChoiceDivergenceBeforeOrdinaryComparison(t *testing.T) {
	artifactPath, _ := publishReplayArtifactForTarget(t, nil, replayArtifactTarget{Choices: true})
	executor := &fakeReplayExecutor{err: &execution.ChoiceReplayDivergenceError{Divergence: choice.Divergence{Ordinal: 3, Reason: choice.DivergenceSite}}}
	result, err := Replay(context.Background(), ReplaySpec{
		ArtifactPath: artifactPath, ToolchainRoot: toolchainRoot(t), SupervisorCommand: []string{"unused"}, Executor: executor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Match || result.ChoiceReplayStatus != ChoiceReplayDiverged || result.Divergence != "choice_profile.divergence.ordinal[3].site" || executor.calls != 1 {
		t.Fatalf("replay result = %#v, calls = %d", result, executor.calls)
	}
}

func TestReplayPreservesInfrastructureErrorJoinedWithChoiceDivergence(t *testing.T) {
	artifactPath, _ := publishReplayArtifactForTarget(t, nil, replayArtifactTarget{Choices: true})
	cleanupErr := errors.New("cleanup failed")
	executor := &fakeReplayExecutor{err: errors.Join(
		&execution.ChoiceReplayDivergenceError{Divergence: choice.Divergence{Ordinal: 3, Reason: choice.DivergenceSite}},
		cleanupErr,
	)}
	_, err := Replay(context.Background(), ReplaySpec{
		ArtifactPath: artifactPath, ToolchainRoot: toolchainRoot(t), SupervisorCommand: []string{"unused"}, Executor: executor,
	})
	if !errors.Is(err, cleanupErr) {
		t.Fatalf("Replay() error = %v", err)
	}
}

func TestReplayDoesNotExecuteLegacyChoiceArtifact(t *testing.T) {
	artifactPath, _ := replayArtifact(t)
	opened, err := evidence.OpenArtifact(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	manifest := opened.Manifest
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
	payload := make([]byte, 48)
	binary.BigEndian.PutUint64(payload[:8], 0)
	payload[8] = byte(choice.KindRunnable)
	payload[9] = byte(choice.FlagDecision)
	binary.BigEndian.PutUint32(payload[12:16], 2)
	binary.BigEndian.PutUint32(payload[16:20], 1)
	binary.BigEndian.PutUint64(payload[24:32], 17)
	manifest.SchemaVersion = evidence.PreviousSchemaVersion
	manifest.Runner.RecordContract = evidence.PreviousRecordContract
	manifest.Target.CapabilityMode = ""
	manifest.Target.CapabilityManifest = nil
	manifest.ChoiceProfile = &evidence.ChoiceProfile{
		Name: choice.LegacyProfile, ImplementationSHA256: evidence.HashBytes([]byte("legacy choice implementation")),
		Trace: evidence.ChoiceTrace{
			Schema: "gomadv3.choice-trace/v1", File: "choices.bin", SHA256: evidence.HashBytes(payload), Bytes: 48,
			Records: 1, BranchingRecords: 1, TerminalState: "complete", Limit: 112,
		},
	}
	manifest.Limits.ChoiceTraceBytes = 112
	manifest.Environment = append(manifest.Environment, evidence.Environment{Name: "GOMADV3_CHOICE_PROFILE", Value: choice.LegacyProfile})
	sort.Slice(manifest.Environment, func(i, j int) bool { return manifest.Environment[i].Name < manifest.Environment[j].Name })
	manifest.Files = append(manifest.Files, evidence.File{Path: "choices.bin", Mode: "0600", Size: 48, SHA256: evidence.HashBytes(payload)})
	sort.Slice(manifest.Files, func(i, j int) bool { return manifest.Files[i].Path < manifest.Files[j].Path })
	_, manifestBytes, err := evidence.FinalizeExecutionRecord(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifactPath, "choices.bin"), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifactPath, "manifest.json"), manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	executor := &fakeReplayExecutor{}
	result, err := Replay(context.Background(), ReplaySpec{
		ArtifactPath: artifactPath, ToolchainRoot: toolchainRoot(t), SupervisorCommand: []string{"unused"}, Executor: executor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Match || result.ChoiceReplayStatus != ChoiceReplayUnavailable || result.Divergence != "choice_profile.replay_unavailable" || executor.calls != 0 {
		t.Fatalf("legacy replay result = %#v, calls = %d", result, executor.calls)
	}
}

func TestReplayVerifyOnlyDoesNotStartTarget(t *testing.T) {
	artifactPath, _ := replayArtifact(t)
	executor := &fakeReplayExecutor{}
	result, err := Replay(context.Background(), ReplaySpec{
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
	artifactPath, _ := publishReplayArtifactWithCompatibility(t, []evidence.CompatibilityPack{{
		ID: "unknown-pack", SHA256: evidence.HashBytes([]byte("unknown pack")),
	}})
	executor := &fakeReplayExecutor{}
	_, err := Replay(context.Background(), ReplaySpec{
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
	_, err := Replay(context.Background(), ReplaySpec{
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
	bundle, err := execution.ComposeRecording(world.Recording{
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
		Argv:          []string{"gomadv3-target", "-test.run=^TestWorldReplayDivergentTarget$"},
		OutcomeReason: "world_replay_divergence",
		IOTranscript:  recordReplayIOTranscript(t),
	})
	result, err := Replay(context.Background(), ReplaySpec{
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
	bundle, err := execution.ComposeRecording(world.Recording{Initial: initial, Final: core.Snapshot(), Terminal: world.Terminal{Kind: world.TerminalDeadlock}}, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	artifactPath, _ := publishReplayArtifactForTarget(t, &bundle, replayArtifactTarget{
		Argv:          []string{"gomadv3-target", "-test.run=^TestWorldReplayMatchingTarget$"},
		OutcomeReason: "world_deadlock",
		IOTranscript:  recordReplayIOTranscript(t),
	})
	result, err := Replay(context.Background(), ReplaySpec{
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

func TestWorldReplayDivergentTarget(_ *testing.T) {
	if !worldReplayTargetSelected("TestWorldReplayDivergentTarget") {
		return
	}
	runWorldReplayTarget("diverge")
}

func TestWorldReplayMatchingTarget(_ *testing.T) {
	if !worldReplayTargetSelected("TestWorldReplayMatchingTarget") {
		return
	}
	runWorldReplayTarget("match")
}

func worldReplayTargetSelected(name string) bool {
	want := "-test.run=^" + name + "$"
	for _, argument := range os.Args {
		if argument == want {
			return true
		}
	}
	return false
}

func runWorldReplayTarget(mode string) {
	core, err := world.New(world.Config{Seed: 7, Limits: world.Limits{MaxRequests: 10, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 10, MaxPayloadBytes: 1024, MaxStringBytes: 64}})
	if err != nil {
		os.Exit(10) //nolint:revive // This subprocess helper reports failure by exit status.
	}
	session, err := worldtarget.Open(core)
	if err != nil {
		os.Exit(11) //nolint:revive // This subprocess helper reports failure by exit status.
	}
	key := "expected"
	stdout := "recorded stdout"
	if mode == "diverge" {
		key = "actual"
	}
	_, transitionErr := session.Model().Register(world.Request{Kind: "read", Resource: world.ResourceID{Adapter: "memory", Kind: "cell", Key: key}})
	if transitionErr == nil {
		if mode == "diverge" {
			stdout = "target-visible mutation"
			transitionErr = fmt.Errorf("%w: replay plan was not attached", world.ErrReplayDivergence)
		} else {
			_, transitionErr = session.Model().Quiesce()
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
	argv := []string{"gomadv3-target", "-test.run=^TestWorldReplayMatchingTarget$"}
	profile := deterministicio.Default()
	frame, err := profile.BootstrapFrame(target.Prepared{SHA256: string(evidence.HashBytes(targetBytes)), Argv: argv}, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 7)
	if err != nil {
		t.Fatal(err)
	}
	result, err := execution.Run(context.Background(), execution.Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestIOReplaySupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestIOReplayBootstrapHelper"},
		Command:           targetPath, Argv0: argv[0], Args: argv[1:], Dir: t.TempDir(),
		Env:        []string{"GOMADSEED=7", "GOMADV3_IO_PROFILE=" + profile.Name(), "TZ=UTC"},
		RunTimeout: 5 * time.Second, TerminateGrace: 100 * time.Millisecond, OutputLimit: 64,
		World: execution.WorldCapability{RecordLimit: world.MaximumRecordingBytes, TransitionLimit: 1 << 20, Seed: 7},
		IO:    &execution.IOCapability{Config: frame, Transcript: &execution.IOTranscriptCapability{Limit: 64 << 20}},
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
	result, err := Replay(context.Background(), ReplaySpec{
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
	observed := execution.Result{IOTranscript: execution.IOTranscript{ReplayDivergence: &ordinal}}
	if divergence := replayDivergence(evidence.ExecutionRecord{}, observed, nil); divergence != "io_profile.transcript.ordinal[4]" {
		t.Fatalf("replayDivergence() = %q", divergence)
	}
}

func TestReplayReportsFinalChoiceTraceMismatch(t *testing.T) {
	expectedDigest := sha256.Sum256([]byte("expected"))
	observedDigest := sha256.Sum256([]byte("observed"))
	manifest := evidence.ExecutionRecord{
		Outcome:       evidence.Outcome{Domain: "success", Reason: "success", Termination: "exit"},
		Streams:       evidence.Streams{Stdout: evidence.Stream{FullSHA256: evidence.SHA256FromSum(sha256.Sum256(nil))}, Stderr: evidence.Stream{FullSHA256: evidence.SHA256FromSum(sha256.Sum256(nil))}},
		ChoiceProfile: &evidence.ChoiceProfile{Name: choice.Profile, Trace: evidence.ChoiceTrace{SHA256: evidence.SHA256FromSum(expectedDigest), Records: 1, BranchingRecords: 1, Limit: 1 << 20}},
	}
	observed := execution.Result{
		Termination: execution.TerminationExit,
		Stdout:      replayOutput(""), Stderr: replayOutput(""),
		ChoiceTrace: execution.ChoiceTrace{Profile: choice.Profile, Limit: 1 << 20, Trace: choice.Trace{SHA256: observedDigest, Summary: choice.Summary{Records: 1, Branching: 1, Terminal: choice.TerminalComplete}}},
	}
	if divergence := replayDivergence(manifest, observed, nil); divergence != "choice_profile.trace.sha256" {
		t.Fatalf("replayDivergence() = %q", divergence)
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
	result, err := Replay(context.Background(), ReplaySpec{
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
	bundle, err := execution.Compose(initial, core.Snapshot(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	artifactPath, expected := publishReplayArtifact(t, &bundle)
	result, err := Replay(context.Background(), ReplaySpec{
		ArtifactPath: artifactPath, ToolchainRoot: toolchainRoot(t), SupervisorCommand: []string{"unused"}, Executor: &fakeReplayExecutor{result: expected},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Match {
		t.Fatalf("connected World replay = %#v", result)
	}
	changed := bundle
	changed.Manifest.Final.SemanticDigest = evidence.HashBytes([]byte("changed semantic digest"))
	changedPath, _ := publishReplayArtifact(t, &changed)
	executor := &fakeReplayExecutor{}
	if _, err := Replay(context.Background(), ReplaySpec{
		ArtifactPath: changedPath, ToolchainRoot: toolchainRoot(t), SupervisorCommand: []string{"unused"}, Executor: executor,
	}); err == nil || executor.calls != 0 {
		t.Fatalf("changed World Replay() error = %v, calls = %d", err, executor.calls)
	}
}

type fakeReplayExecutor struct {
	mu      sync.Mutex
	calls   int
	request execution.Spec
	result  execution.Result
	err     error
}

func (executor *fakeReplayExecutor) Run(_ context.Context, request execution.Spec) (execution.Result, error) {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	executor.calls++
	executor.request = request
	return executor.result, executor.err
}

func replayArtifact(t *testing.T) (string, execution.Result) {
	return publishReplayArtifact(t, nil)
}

func publishReplayArtifact(t *testing.T, connected *execution.Bundle) (string, execution.Result) {
	return publishReplayArtifactWithWorldAndCompatibility(t, connected, []evidence.CompatibilityPack{})
}

func publishReplayArtifactWithCompatibility(t *testing.T, compatibility []evidence.CompatibilityPack) (string, execution.Result) {
	return publishReplayArtifactWithWorldAndCompatibility(t, nil, compatibility)
}

func publishReplayArtifactWithWorldAndCompatibility(t *testing.T, connected *execution.Bundle, compatibility []evidence.CompatibilityPack) (string, execution.Result) {
	return publishReplayArtifactForTargetAndCompatibility(t, connected, compatibility, replayArtifactTarget{})
}

type replayArtifactTarget struct {
	Argv             []string
	Environment      []evidence.Environment
	OutcomeReason    string
	IOTranscript     []byte
	Choices          bool
	Simulation       bool
	ForcedSimulation bool
}

func publishReplayArtifactForTarget(t *testing.T, connected *execution.Bundle, replayTarget replayArtifactTarget) (string, execution.Result) {
	return publishReplayArtifactForTargetAndCompatibility(t, connected, []evidence.CompatibilityPack{}, replayTarget)
}

func publishReplayArtifactForTargetAndCompatibility(t *testing.T, connected *execution.Bundle, compatibility []evidence.CompatibilityPack, replayTarget replayArtifactTarget) (string, execution.Result) {
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
	ioTranscriptRecords, err := deterministicio.TranscriptRecordCount(ioTranscript)
	if err != nil {
		t.Fatal(err)
	}
	profile := deterministicio.Default()
	exitCode := evidence.Uint64String(2)
	targetArgv := replayTarget.Argv
	if len(targetArgv) == 0 {
		targetArgv = []string{"gomadv3-target", "-test.run=none"}
	}
	environment := append([]evidence.Environment{
		{Name: "GOMADSEED", Value: "7"},
		{Name: "GOMADV3_IO_PROFILE", Value: profile.Name()},
		{Name: "TZ", Value: "UTC"},
	}, replayTarget.Environment...)
	sort.Slice(environment, func(i, j int) bool { return environment[i].Name < environment[j].Name })
	outcomeReason := replayTarget.OutcomeReason
	if outcomeReason == "" {
		outcomeReason = "nonzero_exit"
	}
	recordedWorld, payloads := evidence.NoneWorld()
	if connected != nil {
		recordedWorld = connected.Manifest
		payloads = connected.Payloads
	}
	input := campaignstore.ArtifactInput{
		Manifest: evidence.ExecutionRecord{
			SchemaVersion: evidence.SchemaVersion, ArtifactKind: evidence.ArtifactTargetFailure, CreatedAt: "2026-08-10T12:00:00Z", CampaignID: "replay-test",
			SelectionOrdinal: 0, Seed: 7, ReplayMode: evidence.ReplayExact,
			Runner:    evidence.Runner{RecordContract: evidence.RecordContract, RunnerBuild: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", HostOS: runtime.GOOS, HostArch: runtime.GOARCH},
			Toolchain: evidence.Toolchain{GoVersion: identity.GoVersion, BuildKey: identity.BuildKey, TargetGOOS: identity.TargetGOOS, TargetGOARCH: identity.TargetGOARCH},
			Target: evidence.Target{
				Kind: "go-test", Source: "replay fixture", SHA256: evidence.HashBytes(targetBytes), Size: evidence.Uint64String(len(targetBytes)),
				Argv: targetArgv, BuildTags: []string{"gomad_fixture"}, Adapters: []evidence.TargetAdapter{}, Compatibility: compatibility, BuildInfo: projectTestBuildInfo(build),
			},
			IOProfile: evidence.IOProfile{
				Name: profile.Name(), ImplementationSHA256: evidence.SHA256(profile.ImplementationSHA256()), Inventory: string(profile.Inventory()), InventorySHA256: evidence.SHA256(profile.InventorySHA256()),
				Transcript: &evidence.IOTranscript{Schema: "gomadv3.io-transcript/v1", SHA256: evidence.SHA256FromSum(ioTranscriptSHA256), Bytes: evidence.Uint64String(len(ioTranscript)), Records: evidence.Uint64String(ioTranscriptRecords)},
			},
			Environment: environment,
			Limits:      evidence.Limits{RunTimeoutNanos: evidence.Uint64String(5 * time.Second), OverallTimeoutNanos: evidence.Uint64String(10 * time.Second), TerminateGraceNanos: evidence.Uint64String(100 * time.Millisecond), OutputBytes: 64, WorldTransitionBytes: 1 << 20, IOTranscriptBytes: 64 << 20},
			World:       recordedWorld,
			Outcome:     evidence.Outcome{Domain: "target", Reason: outcomeReason, Termination: "exit", ExitCode: &exitCode},
			Streams:     evidence.Streams{Stdout: replayStream(stdout), Stderr: replayStream(stderr)},
			Host:        evidence.Host{StartedAt: "2026-08-10T12:00:00Z", FinishedAt: "2026-08-10T12:00:01Z", ElapsedNanos: evidence.Uint64String(time.Second)},
		},
		TargetPath: targetPath, Stdout: stdout.Bytes, Stderr: stderr.Bytes, IOTranscript: ioTranscript, World: payloads,
	}
	var recordedChoices execution.ChoiceTrace
	if replayTarget.Choices {
		implementation, identityErr := choice.ImplementationIdentity(identity.BuildKey)
		if identityErr != nil {
			t.Fatal(identityErr)
		}
		first := sha256.Sum256([]byte("first choice alternative"))
		second := sha256.Sum256([]byte("second choice alternative"))
		decision, decisionErr := choice.CanonicalDecision(0, choice.KindRunnable, 17, false, [][sha256.Size]byte{first, second}, second, 0)
		if decisionErr != nil {
			t.Fatal(decisionErr)
		}
		choiceTrace, decodeErr := choice.BuildTrace([]choice.Record{decision.Record()}, choice.TerminalComplete)
		if decodeErr != nil {
			t.Fatal(decodeErr)
		}
		choicePayload := append([]byte(nil), choiceTrace.Bytes...)
		choiceLimit, limitErr := choice.TraceBytes(1)
		if limitErr != nil {
			t.Fatal(limitErr)
		}
		targetSHA256, targetErr := input.Manifest.Target.SHA256.Bytes()
		if targetErr != nil {
			t.Fatal(targetErr)
		}
		executionIdentity := choice.ExecutionIdentity{
			TargetSHA256: targetSHA256, ToolchainBuildKey: identity.BuildKey,
			GOOS: identity.TargetGOOS, GOARCH: identity.TargetGOARCH, ImplementationSHA256: implementation,
		}
		tape, tapeErr := choice.ProjectReplayPlan(choiceTrace, executionIdentity)
		if tapeErr != nil {
			t.Fatal(tapeErr)
		}
		input.Manifest.ChoiceProfile = &evidence.ChoiceProfile{
			Name: choice.Profile, ImplementationSHA256: evidence.SHA256FromSum(implementation),
			Trace: evidence.ChoiceTrace{
				Schema: "gomadv3.choice-trace/v2", SHA256: evidence.SHA256FromSum(choiceTrace.SHA256), Bytes: evidence.Uint64String(len(choiceTrace.Bytes)),
				Records: 1, BranchingRecords: 1, TerminalState: "complete", Limit: evidence.Uint64String(choiceLimit),
				TapeSHA256: evidence.SHA256FromSum(tape.SHA256), Decisions: 1,
			},
		}
		input.Manifest.Limits.ChoiceTraceBytes = evidence.Uint64String(choiceLimit)
		input.Manifest.Environment = append(input.Manifest.Environment, evidence.Environment{Name: "GOMADV3_CHOICE_PROFILE", Value: choice.Profile})
		sort.Slice(input.Manifest.Environment, func(i, j int) bool { return input.Manifest.Environment[i].Name < input.Manifest.Environment[j].Name })
		input.ChoiceTrace = choicePayload
		recordedChoices = execution.ChoiceTrace{Profile: choice.Profile, ImplementationSHA256: implementation, Limit: choiceLimit, Trace: choiceTrace}
	}
	var recordedSimulationRecords [][]byte
	if replayTarget.Simulation {
		config := combinedfrontier.Config{
			ExecutionSHA256: evidence.HashBytes([]byte("replay simulation execution")), ControllerSHA256: combinedfrontier.ImplementationSHA256(), BaseSeed: 7,
			Parallel: 1, MaxRuns: 1, MaxForcedDecisions: 1, MaxFrontierBytes: 1 << 20, MaxResultBytes: 1 << 20, FailureBudget: 1,
			Limits: combinedfrontier.DimensionLimits{Runtime: 1, Scenario: 1, Network: 1, Storage: 1, Fault: 1, Crash: 1},
		}
		state, stateErr := combinedfrontier.New(config)
		if stateErr != nil {
			t.Fatal(stateErr)
		}
		round, ok := state.NextRound()
		if !ok {
			t.Fatal("simulation replay root candidate is unavailable")
		}
		candidate := round.Candidates[0]
		var runtimeDecisions []combinedfrontier.Decision
		var explorationDecisions []combinedfrontier.Decision
		if replayTarget.ForcedSimulation {
			if !replayTarget.Choices {
				t.Fatal("forced simulation fixture requires choices")
			}
			targetSHA256, targetErr := input.Manifest.Target.SHA256.Bytes()
			if targetErr != nil {
				t.Fatal(targetErr)
			}
			implementation, implementationErr := choice.ImplementationIdentity(identity.BuildKey)
			if implementationErr != nil {
				t.Fatal(implementationErr)
			}
			executionIdentity := choice.ExecutionIdentity{
				TargetSHA256: targetSHA256, ToolchainBuildKey: identity.BuildKey, GOOS: identity.TargetGOOS,
				GOARCH: identity.TargetGOARCH, ImplementationSHA256: implementation,
			}
			exactTape, tapeErr := choice.ProjectReplayPlan(recordedChoices.Trace, executionIdentity)
			if tapeErr != nil {
				t.Fatal(tapeErr)
			}
			runtimeDecisions, tapeErr = simulationexploration.RuntimeDecisions(exactTape)
			if tapeErr != nil {
				t.Fatal(tapeErr)
			}
			prefix, prefixErr := choice.BuildForcedRankPrefix(exactTape, 0, runtimeDecisions[0].Selected)
			if prefixErr != nil {
				t.Fatal(prefixErr)
			}
			runtimeOverride, overrideErr := combinedfrontier.CanonicalForcedDecision(combinedfrontier.ForcedDecision{
				Dimension: combinedfrontier.DimensionRuntime, Ordinal: runtimeDecisions[0].Ordinal,
				SiteSHA256: runtimeDecisions[0].SiteSHA256, Alternatives: uint32(len(runtimeDecisions[0].Alternatives)),
				AlternativeSetSHA256: runtimeDecisions[0].AlternativeSetSHA256, Selected: runtimeDecisions[0].Selected,
				SelectedSHA256: runtimeDecisions[0].Alternatives[runtimeDecisions[0].Selected], Control: prefix.Bytes,
			})
			if overrideErr != nil {
				t.Fatal(overrideErr)
			}
			faultSite := evidence.HashBytes([]byte("fault site"))
			faultAlternatives := []evidence.SHA256{evidence.HashBytes([]byte("no fault")), evidence.HashBytes([]byte("drop"))}
			faultBaseline, decisionErr := combinedfrontier.CanonicalDecision(combinedfrontier.DimensionFault, 0, faultSite, faultAlternatives, 0)
			if decisionErr != nil {
				t.Fatal(decisionErr)
			}
			faultOverride, overrideErr := combinedfrontier.ForceDecision(faultBaseline, 1)
			if overrideErr != nil {
				t.Fatal(overrideErr)
			}
			candidate, overrideErr = combinedfrontier.CanonicalCandidate(config, []combinedfrontier.ForcedDecision{runtimeOverride, faultOverride}, "")
			if overrideErr != nil {
				t.Fatal(overrideErr)
			}
			faultObserved, decisionErr := combinedfrontier.CanonicalDecision(combinedfrontier.DimensionFault, 0, faultSite, faultAlternatives, 1)
			if decisionErr != nil {
				t.Fatal(decisionErr)
			}
			explorationDecisions = []combinedfrontier.Decision{faultObserved}
		}
		plan, planErr := simulationexploration.PlanForCandidate(config, candidate)
		if planErr != nil {
			t.Fatal(planErr)
		}
		record, recordErr := json.Marshal(struct {
			Schema               string                      `json:"schema"`
			Seed                 uint64                      `json:"seed"`
			SpecSHA256           evidence.SHA256             `json:"spec_sha256"`
			Outcome              string                      `json:"outcome"`
			FailureIdentity      evidence.SHA256             `json:"failure_identity"`
			ExplorationPlan      json.RawMessage             `json:"exploration_plan"`
			ExplorationDecisions []combinedfrontier.Decision `json:"exploration_decisions,omitempty"`
			Identity             evidence.SHA256             `json:"identity"`
		}{
			Schema: "gomadv3.cluster-record/v7", Seed: 7, SpecSHA256: evidence.HashBytes([]byte("simulation spec")), Outcome: "oracle_failed",
			FailureIdentity: evidence.HashBytes([]byte("normalized replay failure")), ExplorationPlan: plan,
			ExplorationDecisions: explorationDecisions, Identity: evidence.HashBytes([]byte("simulation record")),
		})
		if recordErr != nil {
			t.Fatal(recordErr)
		}
		profile, profileErr := simulationexploration.ProjectArtifact(config, candidate, plan, record, runtimeDecisions, 1<<20)
		if profileErr != nil {
			t.Fatal(profileErr)
		}
		input.Manifest.SimulationProfile = &profile
		input.Simulation = &campaignstore.SimulationPayloads{Plan: plan, Record: record}
		recordedSimulationRecords = [][]byte{record}
	}
	published, err := campaignstore.PublishArtifact(evidence.Store{Root: t.TempDir()}, input)
	if err != nil {
		t.Fatal(err)
	}
	result := execution.Result{Termination: execution.TerminationExit, ExitCode: 2, GroupGone: true, Stdout: stdout, Stderr: stderr, IOTranscript: execution.IOTranscript{SHA256: ioTranscriptSHA256, Records: ioTranscriptRecords, Complete: true}, ChoiceTrace: recordedChoices, SimulationRecords: recordedSimulationRecords}
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

func replayOutput(value string) execution.Output {
	data := []byte(value)
	digest := sha256.Sum256(data)
	return execution.Output{Bytes: data, FullSHA256: digest, RetainedSHA256: digest, TotalBytes: uint64(len(data)), RetainedBytes: uint64(len(data))}
}

func replayStream(output execution.Output) evidence.Stream {
	return evidence.Stream{
		FullSHA256: evidence.SHA256(fmt.Sprintf("sha256:%x", output.FullSHA256)), TotalBytes: evidence.Uint64String(output.TotalBytes),
		RetainedBytes: evidence.Uint64String(output.RetainedBytes), DiscardedBytes: evidence.Uint64String(output.DiscardedBytes), Truncated: output.Truncated,
	}
}

func projectTestBuildInfo(info *debug.BuildInfo) evidence.BuildInfo {
	settings := make([]evidence.BuildSetting, len(info.Settings))
	for index, setting := range info.Settings {
		settings[index] = evidence.BuildSetting{Key: setting.Key, Value: setting.Value}
	}
	sort.Slice(settings, func(i, j int) bool { return settings[i].Key < settings[j].Key })
	return evidence.BuildInfo{GoVersion: info.GoVersion, Path: info.Path, MainModule: info.Main.Path, Settings: settings}
}

func toolchainRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	return root
}
