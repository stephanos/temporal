package runner

import (
	"context"
	"crypto/sha256"
	"debug/buildinfo"
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

	"go.temporal.io/server/tools/gomadv3/artifact"
	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/internal/hostexec"
	"go.temporal.io/server/tools/gomadv3/record"
	"go.temporal.io/server/tools/gomadv3/runner/internal/execution"
	simulationengine "go.temporal.io/server/tools/gomadv3/runner/internal/exploration/simulation"
	simulationrecord "go.temporal.io/server/tools/gomadv3/runner/internal/exploration/simulationrecord"
	"go.temporal.io/server/tools/gomadv3/target"
	"go.temporal.io/server/tools/gomadv3/world"
	worldprocess "go.temporal.io/server/tools/gomadv3/world/process"
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
	environment := replayEnvironment([]record.Environment{
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
	opened, err := artifact.OpenArtifact(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	profile := opened.Manifest.SimulationProfile
	if profile == nil {
		t.Fatal("replay fixture omitted simulation exploration evidence")
	}
	plan, err := artifact.ReadPayload(opened, profile.Plan.File, uint64(profile.Plan.Bytes))
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
	artifactPath, _ := publishReplayArtifactWithCompatibility(t, []record.CompatibilityPack{{
		ID: "unknown-pack", SHA256: record.HashBytes([]byte("unknown pack")),
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
	session, err := worldprocess.Open(core)
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
	frame, err := profile.BootstrapFrame(target.Prepared{SHA256: string(record.HashBytes(targetBytes)), Argv: argv}, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 7)
	if err != nil {
		t.Fatal(err)
	}
	result, err := execution.Run(context.Background(), execution.Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestIOReplaySupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestIOReplayBootstrapHelper"},
		Command:           targetPath, Argv0: argv[0], Args: argv[1:], Dir: t.TempDir(),
		Env:              []string{"GOMADSEED=7", "GOMADV3_IO_PROFILE=" + profile.Name(), "TZ=UTC"},
		ExecutionTimeout: 5 * time.Second, TerminateGrace: 100 * time.Millisecond, OutputLimit: 64,
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
	observed := execution.Result{IOTranscript: deterministicio.Transcript{ReplayDivergence: &ordinal}}
	if divergence := replayDivergence(record.ExecutionRecord{}, observed, nil); divergence != "io_profile.transcript.ordinal[4]" {
		t.Fatalf("replayDivergence() = %q", divergence)
	}
}

func TestReplayReportsFinalChoiceTraceMismatch(t *testing.T) {
	expectedDigest := sha256.Sum256([]byte("expected"))
	observedDigest := sha256.Sum256([]byte("observed"))
	manifest := record.ExecutionRecord{
		Outcome:       record.Outcome{Domain: "success", Reason: "success", Termination: "exit"},
		Streams:       record.Streams{Stdout: record.Stream{FullSHA256: record.SHA256FromSum(sha256.Sum256(nil))}, Stderr: record.Stream{FullSHA256: record.SHA256FromSum(sha256.Sum256(nil))}},
		ChoiceProfile: &record.ChoiceProfile{Name: choice.Profile, Trace: record.ChoiceTrace{SHA256: record.SHA256FromSum(expectedDigest), Records: 1, BranchingRecords: 1, Limit: 1 << 20}},
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
	changed.Manifest.Final.SemanticDigest = record.HashBytes([]byte("changed semantic digest"))
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
	return publishReplayArtifactWithWorldAndCompatibility(t, connected, []record.CompatibilityPack{})
}

func publishReplayArtifactWithCompatibility(t *testing.T, compatibility []record.CompatibilityPack) (string, execution.Result) {
	return publishReplayArtifactWithWorldAndCompatibility(t, nil, compatibility)
}

func publishReplayArtifactWithWorldAndCompatibility(t *testing.T, connected *execution.Bundle, compatibility []record.CompatibilityPack) (string, execution.Result) {
	return publishReplayArtifactForTargetAndCompatibility(t, connected, compatibility, replayArtifactTarget{})
}

type replayArtifactTarget struct {
	Argv             []string
	Environment      []record.Environment
	OutcomeReason    string
	IOTranscript     []byte
	Choices          bool
	Simulation       bool
	ForcedSimulation bool
}

func publishReplayArtifactForTarget(t *testing.T, connected *execution.Bundle, replayTarget replayArtifactTarget) (string, execution.Result) {
	return publishReplayArtifactForTargetAndCompatibility(t, connected, []record.CompatibilityPack{}, replayTarget)
}

func publishReplayArtifactForTargetAndCompatibility(t *testing.T, connected *execution.Bundle, compatibility []record.CompatibilityPack, replayTarget replayArtifactTarget) (string, execution.Result) {
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
	input := artifact.ArtifactInput{
		Manifest: record.ExecutionRecord{
			SchemaVersion: record.SchemaVersion, ArtifactKind: record.ArtifactTargetFailure, CreatedAt: "2026-08-10T12:00:00Z", CampaignID: "replay-test",
			SelectionOrdinal: 0, Seed: 7, ReplayMode: record.ReplayExact,
			Runner:    record.Runner{RecordContract: record.RecordContract, RunnerBuild: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", HostOS: runtime.GOOS, HostArch: runtime.GOARCH},
			Toolchain: record.Toolchain{GoVersion: identity.GoVersion, BuildKey: identity.BuildKey, TargetGOOS: identity.TargetGOOS, TargetGOARCH: identity.TargetGOARCH},
			Target: record.Target{
				Kind: "go-test", Source: "replay fixture", SHA256: record.HashBytes(targetBytes), Size: record.Uint64String(len(targetBytes)),
				Argv: targetArgv, BuildTags: []string{"gomad_fixture"}, Adapters: []record.TargetAdapter{}, Compatibility: compatibility, BuildInfo: projectTestBuildInfo(build),
			},
			IOProfile: record.IOProfile{
				Name: profile.Name(), ImplementationSHA256: record.SHA256(profile.ImplementationSHA256()), Inventory: string(profile.Inventory()), InventorySHA256: record.SHA256(profile.InventorySHA256()),
				Transcript: &record.IOTranscript{Schema: "gomadv3.io-transcript/v1", SHA256: record.SHA256FromSum(ioTranscriptSHA256), Bytes: record.Uint64String(len(ioTranscript)), Records: record.Uint64String(ioTranscriptRecords)},
			},
			Environment: environment,
			Limits:      record.Limits{ExecutionTimeoutNanos: record.Uint64String(5 * time.Second), OverallTimeoutNanos: record.Uint64String(10 * time.Second), TerminateGraceNanos: record.Uint64String(100 * time.Millisecond), OutputBytes: 64, WorldTransitionBytes: 1 << 20, IOTranscriptBytes: 64 << 20},
			World:       recordedWorld,
			Outcome:     record.Outcome{Domain: "target", Reason: outcomeReason, Termination: "exit", ExitCode: &exitCode},
			Streams:     record.Streams{Stdout: replayStream(stdout), Stderr: replayStream(stderr)},
			Host:        record.Host{StartedAt: "2026-08-10T12:00:00Z", FinishedAt: "2026-08-10T12:00:01Z", ElapsedNanos: record.Uint64String(time.Second)},
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
		input.Manifest.ChoiceProfile = &record.ChoiceProfile{
			Name: choice.Profile, ImplementationSHA256: record.SHA256FromSum(implementation),
			Trace: record.ChoiceTrace{
				Schema: "gomadv3.choice-trace/v2", SHA256: record.SHA256FromSum(choiceTrace.SHA256), Bytes: record.Uint64String(len(choiceTrace.Bytes)),
				Records: 1, BranchingRecords: 1, TerminalState: "complete", Limit: record.Uint64String(choiceLimit),
				TapeSHA256: record.SHA256FromSum(tape.SHA256), Decisions: 1,
			},
		}
		input.Manifest.Limits.ChoiceTraceBytes = record.Uint64String(choiceLimit)
		input.Manifest.Environment = append(input.Manifest.Environment, record.Environment{Name: "GOMADV3_CHOICE_PROFILE", Value: choice.Profile})
		sort.Slice(input.Manifest.Environment, func(i, j int) bool { return input.Manifest.Environment[i].Name < input.Manifest.Environment[j].Name })
		input.ChoiceTrace = choicePayload
		recordedChoices = execution.ChoiceTrace{Profile: choice.Profile, ImplementationSHA256: implementation, Limit: choiceLimit, Trace: choiceTrace}
	}
	var recordedSimulationRecords [][]byte
	if replayTarget.Simulation {
		config := simulationengine.Config{
			ExecutionSHA256: record.HashBytes([]byte("replay simulation execution")), ControllerSHA256: simulationengine.ImplementationSHA256(), BaseSeed: 7,
			Parallel: 1, MaxExecutions: 1, MaxForcedDecisions: 1, MaxExplorationBytes: 1 << 20, MaxResultBytes: 1 << 20, FailureBudget: 1,
			Limits: simulationengine.DimensionLimits{Runtime: 1, Scenario: 1, Network: 1, Storage: 1, Fault: 1, Crash: 1},
		}
		state, stateErr := simulationengine.New(config)
		if stateErr != nil {
			t.Fatal(stateErr)
		}
		round, ok := state.NextRound()
		if !ok {
			t.Fatal("simulation replay root candidate is unavailable")
		}
		candidate := round.Candidates[0]
		var runtimeDecisions []simulationengine.Decision
		var explorationDecisions []simulationengine.Decision
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
			runtimeDecisions, tapeErr = simulationrecord.RuntimeDecisions(exactTape)
			if tapeErr != nil {
				t.Fatal(tapeErr)
			}
			prefix, prefixErr := choice.BuildForcedRankPrefix(exactTape, 0, runtimeDecisions[0].Selected)
			if prefixErr != nil {
				t.Fatal(prefixErr)
			}
			runtimeOverride, overrideErr := simulationengine.CanonicalForcedDecision(simulationengine.ForcedDecision{
				Dimension: simulationengine.DimensionRuntime, Ordinal: runtimeDecisions[0].Ordinal,
				SiteSHA256: runtimeDecisions[0].SiteSHA256, Alternatives: uint32(len(runtimeDecisions[0].Alternatives)),
				AlternativeSetSHA256: runtimeDecisions[0].AlternativeSetSHA256, Selected: runtimeDecisions[0].Selected,
				SelectedSHA256: runtimeDecisions[0].Alternatives[runtimeDecisions[0].Selected], Control: prefix.Bytes,
			})
			if overrideErr != nil {
				t.Fatal(overrideErr)
			}
			faultSite := record.HashBytes([]byte("fault site"))
			faultAlternatives := []record.SHA256{record.HashBytes([]byte("no fault")), record.HashBytes([]byte("drop"))}
			faultBaseline, decisionErr := simulationengine.CanonicalDecision(simulationengine.DimensionFault, 0, faultSite, faultAlternatives, 0)
			if decisionErr != nil {
				t.Fatal(decisionErr)
			}
			faultOverride, overrideErr := simulationengine.ForceDecision(faultBaseline, 1)
			if overrideErr != nil {
				t.Fatal(overrideErr)
			}
			candidate, overrideErr = simulationengine.CanonicalCandidate(config, []simulationengine.ForcedDecision{runtimeOverride, faultOverride}, "")
			if overrideErr != nil {
				t.Fatal(overrideErr)
			}
			faultObserved, decisionErr := simulationengine.CanonicalDecision(simulationengine.DimensionFault, 0, faultSite, faultAlternatives, 1)
			if decisionErr != nil {
				t.Fatal(decisionErr)
			}
			explorationDecisions = []simulationengine.Decision{faultObserved}
		}
		plan, planErr := simulationrecord.PlanForCandidate(config, candidate)
		if planErr != nil {
			t.Fatal(planErr)
		}
		record, recordErr := json.Marshal(struct {
			Schema               string                      `json:"schema"`
			Seed                 uint64                      `json:"seed"`
			SpecSHA256           record.SHA256               `json:"spec_sha256"`
			Outcome              string                      `json:"outcome"`
			FailureIdentity      record.SHA256               `json:"failure_identity"`
			ExplorationPlan      json.RawMessage             `json:"exploration_plan"`
			ExplorationDecisions []simulationengine.Decision `json:"exploration_decisions,omitempty"`
			Identity             record.SHA256               `json:"identity"`
		}{
			Schema: "gomadv3.cluster-record/v7", Seed: 7, SpecSHA256: record.HashBytes([]byte("simulation spec")), Outcome: "oracle_failed",
			FailureIdentity: record.HashBytes([]byte("normalized replay failure")), ExplorationPlan: plan,
			ExplorationDecisions: explorationDecisions, Identity: record.HashBytes([]byte("simulation record")),
		})
		if recordErr != nil {
			t.Fatal(recordErr)
		}
		profile, profileErr := simulationrecord.ProjectArtifact(config, candidate, plan, record, runtimeDecisions, 1<<20)
		if profileErr != nil {
			t.Fatal(profileErr)
		}
		input.Manifest.SimulationProfile = &profile
		input.Simulation = &artifact.SimulationPayloads{Plan: plan, Record: record}
		recordedSimulationRecords = [][]byte{record}
	}
	published, err := artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, input)
	if err != nil {
		t.Fatal(err)
	}
	result := execution.Result{Termination: execution.TerminationExit, ExitCode: 2, GroupGone: true, Stdout: stdout, Stderr: stderr, IOTranscript: deterministicio.Transcript{SHA256: ioTranscriptSHA256, Records: ioTranscriptRecords, Complete: true}, ChoiceTrace: recordedChoices, SimulationRecords: recordedSimulationRecords}
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

func replayOutput(value string) hostexec.Output {
	data := []byte(value)
	digest := sha256.Sum256(data)
	return hostexec.Output{Bytes: data, FullSHA256: digest, RetainedSHA256: digest, TotalBytes: uint64(len(data)), RetainedBytes: uint64(len(data))}
}

func replayStream(output hostexec.Output) record.Stream {
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
	root, err := filepath.Abs(filepath.Join("..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	return root
}
