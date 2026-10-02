package runner

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	simulationengine "go.temporal.io/server/tools/gomad3/runner/internal/exploration/simulation"
	"go.temporal.io/server/tools/gomad3/runner/internal/minimizer"
)

func TestMinimizePublishesLinkedExactScheduleAndFaultReduction(t *testing.T) {
	artifactPath, _ := publishReplayArtifactForTarget(t, nil, replayArtifactTarget{Choices: true, Simulation: true, ForcedSimulation: true})
	parent, err := artifact.OpenArtifact(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	parentRecordHash := parent.Manifest.RecordHash
	parentFailureSignature := parent.Manifest.Outcome.FailureSignature
	if err := parent.Close(); err != nil {
		t.Fatal(err)
	}
	executor := &minimizationExecutor{}
	replayer := &minimizationReplayer{}

	result, err := Minimize(context.Background(), MinimizeSpec{
		ArtifactPath: artifactPath, OutputRoot: t.TempDir(), AttemptBudget: 16,
		ToolchainRoot: toolchainRoot(t), SupervisorCommand: []string{"unused"}, Executor: executor, Replayer: replayer,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Artifact.Path == artifactPath || result.Attempts == 0 || len(result.Accepted) != 1 {
		t.Fatalf("minimize result = %#v", result)
	}
	if result.Artifact.Manifest.RecordHash == parentRecordHash || result.Artifact.Manifest.Outcome.FailureSignature != parentFailureSignature {
		t.Fatalf("minimized identity = %#v", result.Artifact.Manifest)
	}
	minimization := result.Artifact.Manifest.Minimization
	if minimization == nil || minimization.ParentRecordHash != parentRecordHash || minimization.ParentFailureSignature != parentFailureSignature || minimization.OriginalForcedDecisions != 2 || minimization.FinalForcedDecisions != 1 || minimization.Predicate.ChoiceReplay != "exact" || minimization.Predicate.SimulationReplay != "exact" {
		t.Fatalf("minimization evidence = %#v", minimization)
	}
	if minimization.Accepted[0].Kind != "fault_entries" || minimization.Accepted[0].Removed[0].Dimension != "fault" {
		t.Fatalf("accepted reduction = %#v", minimization.Accepted)
	}
	if executor.calls != int(result.Attempts) || replayer.calls != 2 {
		t.Fatalf("candidate calls = %d, replay calls = %d", executor.calls, replayer.calls)
	}
	reopened, err := artifact.OpenArtifact(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Manifest.RecordHash != parentRecordHash || reopened.Manifest.Minimization != nil {
		t.Fatalf("parent artifact changed = %#v", reopened.Manifest)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestMinimizeRejectsSimulationArtifactWithoutExactChoiceTape(t *testing.T) {
	artifactPath, _ := publishReplayArtifactForTarget(t, nil, replayArtifactTarget{Simulation: true})
	_, err := Minimize(context.Background(), MinimizeSpec{
		ArtifactPath: artifactPath, OutputRoot: t.TempDir(), AttemptBudget: 16,
		ToolchainRoot: toolchainRoot(t), SupervisorCommand: []string{"unused"}, Executor: &minimizationExecutor{}, Replayer: &minimizationReplayer{},
	})
	if err == nil {
		t.Fatal("Minimize() accepted an artifact without an exact choice tape")
	}
}

// The fixture's run evaluates three candidates and accepts the second, so an
// interruption at the third executor call follows an accepted reduction, and
// the resumed run accepts nothing further.
const minimizationCallAfterAcceptedReduction = 3

var errMinimizationInterrupted = errors.New("minimization interrupted")

type minimizationOutcome struct {
	RecordHash    record.SHA256
	Changed       bool
	Attempts      uint64
	AttemptBudget uint64
	Accepted      []record.MinimizationReduction
	StopReason    string
}

func minimizationOutcomeOf(result MinimizeResult) minimizationOutcome {
	return minimizationOutcome{
		RecordHash: result.Artifact.Manifest.RecordHash, Changed: result.Changed, Attempts: result.Attempts,
		AttemptBudget: result.AttemptBudget, Accepted: result.Accepted, StopReason: result.StopReason,
	}
}

func minimizationParent(t *testing.T, environment ...record.Environment) string {
	t.Helper()
	artifactPath, _ := publishReplayArtifactForTarget(t, nil, replayArtifactTarget{Choices: true, Simulation: true, ForcedSimulation: true, Environment: environment})
	return artifactPath
}

func minimizationSpec(t *testing.T, artifactPath, outputRoot string) MinimizeSpec {
	t.Helper()
	return MinimizeSpec{
		ArtifactPath: artifactPath, OutputRoot: outputRoot, AttemptBudget: 16,
		ToolchainRoot: toolchainRoot(t), SupervisorCommand: []string{"unused"}, Executor: &minimizationExecutor{}, Replayer: &minimizationReplayer{},
	}
}

func uninterruptedMinimization(t *testing.T, artifactPath string) minimizationOutcome {
	t.Helper()
	result, err := Minimize(context.Background(), minimizationSpec(t, artifactPath, t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	return minimizationOutcomeOf(result)
}

func interruptAt(call int) func(int) error {
	return func(observed int) error {
		if observed == call {
			return errMinimizationInterrupted
		}
		return nil
	}
}

// interruptedMinimization returns an output root holding the state of a run
// that died in its third evaluation, after one accepted reduction.
func interruptedMinimization(t *testing.T, artifactPath string) string {
	t.Helper()
	outputRoot := t.TempDir()
	spec := minimizationSpec(t, artifactPath, outputRoot)
	spec.Executor = &minimizationExecutor{before: interruptAt(minimizationCallAfterAcceptedReduction)}
	if _, err := Minimize(context.Background(), spec); !errors.Is(err, errMinimizationInterrupted) {
		t.Fatalf("interrupted Minimize() error = %v", err)
	}
	return outputRoot
}

func publishedMinimizedArtifacts(t *testing.T, outputRoot string) []string {
	t.Helper()
	entries, err := os.ReadDir(outputRoot)
	if err != nil {
		t.Fatal(err)
	}
	var published []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "sha256-") {
			published = append(published, entry.Name())
		}
	}
	return published
}

func TestMinimizeResumeContinuesAfterAcceptedReductionWithoutRepeatingAttempts(t *testing.T) {
	artifactPath := minimizationParent(t)
	uninterrupted := uninterruptedMinimization(t, artifactPath)
	outputRoot := interruptedMinimization(t, artifactPath)

	spec := minimizationSpec(t, artifactPath, outputRoot)
	if _, err := Minimize(context.Background(), spec); !errors.Is(err, minimizer.ErrCheckpointExists) {
		t.Fatalf("Minimize() over existing state error = %v", err)
	}
	if calls := spec.Executor.(*minimizationExecutor).calls; calls != 0 {
		t.Fatalf("refused run evaluated %d candidates", calls)
	}

	spec.Resume = true
	result, err := Minimize(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if calls := spec.Executor.(*minimizationExecutor).calls; calls != int(uninterrupted.Attempts)-minimizationCallAfterAcceptedReduction+1 {
		t.Fatalf("resumed run evaluated %d candidates of %d attempts", calls, uninterrupted.Attempts)
	}
	if resumed := minimizationOutcomeOf(result); !resumed.Changed || !reflect.DeepEqual(resumed, uninterrupted) {
		t.Fatalf("resumed outcome = %#v, want %#v", resumed, uninterrupted)
	}
	if published := publishedMinimizedArtifacts(t, outputRoot); !reflect.DeepEqual(published, []string{filepath.Base(result.Artifact.Path)}) {
		t.Fatalf("published artifacts = %v, result = %s", published, result.Artifact.Path)
	}
	spec.Resume = false
	if _, err := Minimize(context.Background(), spec); err != nil {
		t.Fatalf("Minimize() after a completed run: %v", err)
	}
}

func TestMinimizeResumeRejectsStateOfAnotherRun(t *testing.T) {
	artifactPath := minimizationParent(t)
	outputRoot := interruptedMinimization(t, artifactPath)
	for _, test := range []struct {
		name   string
		change func(*MinimizeSpec)
		want   string
	}{
		{name: "changed parent artifact", change: func(spec *MinimizeSpec) {
			spec.ArtifactPath = minimizationParent(t, record.Environment{Name: "GOMAD_TEST_OTHER_PARENT", Value: "1"})
		}, want: "different parent artifact"},
		{name: "changed attempt budget", change: func(spec *MinimizeSpec) { spec.AttemptBudget++ }, want: "attempt budget 16, not 17"},
		{name: "no state", change: func(spec *MinimizeSpec) { spec.OutputRoot = t.TempDir() }, want: minimizer.ErrNoCheckpoint.Error()},
		{name: "no output directory", change: func(spec *MinimizeSpec) { spec.OutputRoot = filepath.Join(t.TempDir(), "absent") }, want: minimizer.ErrNoCheckpoint.Error()},
	} {
		t.Run(test.name, func(t *testing.T) {
			spec := minimizationSpec(t, artifactPath, outputRoot)
			spec.Resume = true
			test.change(&spec)
			_, err := Minimize(context.Background(), spec)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Minimize() error = %v, want %q", err, test.want)
			}
			if calls := spec.Executor.(*minimizationExecutor).calls; calls != 0 {
				t.Fatalf("rejected resume evaluated %d candidates", calls)
			}
		})
	}
}

func TestMinimizeResumeFailsClosedOnDamagedAcceptedArtifact(t *testing.T) {
	artifactPath := minimizationParent(t)
	for _, test := range []struct {
		name   string
		damage func(t *testing.T, accepted string)
	}{
		{name: "missing", damage: func(t *testing.T, accepted string) {
			if err := os.RemoveAll(accepted); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "corrupt payload", damage: func(t *testing.T, accepted string) {
			if err := os.WriteFile(filepath.Join(accepted, "stdout"), []byte("other stdout bytes"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			outputRoot := interruptedMinimization(t, artifactPath)
			accepted, err := filepath.Glob(filepath.Join(outputRoot, ".minimize", "accepted", "sha256-*"))
			if err != nil || len(accepted) != 1 {
				t.Fatalf("accepted artifacts = %v, %v", accepted, err)
			}
			test.damage(t, accepted[0])
			spec := minimizationSpec(t, artifactPath, outputRoot)
			spec.Resume = true
			if _, err := Minimize(context.Background(), spec); err == nil {
				t.Fatal("Minimize() resumed from a damaged accepted artifact")
			}
			if calls := spec.Executor.(*minimizationExecutor).calls; calls != 0 || len(publishedMinimizedArtifacts(t, outputRoot)) != 0 {
				t.Fatalf("damaged resume evaluated %d candidates, published %v", calls, publishedMinimizedArtifacts(t, outputRoot))
			}
		})
	}
}

func TestMinimizeExcludesConcurrentRunsOnOneOutputRoot(t *testing.T) {
	artifactPath := minimizationParent(t)
	for _, test := range []struct {
		name       string
		outputRoot func(t *testing.T) string
		resume     bool
	}{
		{name: "initial runs", outputRoot: func(t *testing.T) string { return t.TempDir() }},
		{name: "resumes", outputRoot: func(t *testing.T) string { return interruptedMinimization(t, artifactPath) }, resume: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			outputRoot := test.outputRoot(t)
			evaluating := make(chan struct{})
			release := make(chan struct{})
			holder := minimizationSpec(t, artifactPath, outputRoot)
			holder.Resume = test.resume
			holder.Executor = &minimizationExecutor{before: func(call int) error {
				if call == 1 {
					close(evaluating)
					<-release
				}
				return nil
			}}
			held := make(chan error, 1)
			go func() {
				_, err := Minimize(context.Background(), holder)
				held <- err
			}()
			<-evaluating
			contender := minimizationSpec(t, artifactPath, outputRoot)
			contender.Resume = test.resume
			_, err := Minimize(context.Background(), contender)
			close(release)
			if !errors.Is(err, hostfs.ErrContended) {
				t.Fatalf("concurrent Minimize() error = %v", err)
			}
			if calls := contender.Executor.(*minimizationExecutor).calls; calls != 0 {
				t.Fatalf("rejected run evaluated %d candidates", calls)
			}
			if err := <-held; err != nil {
				t.Fatalf("lock-holding Minimize() error = %v", err)
			}
		})
	}
}

func TestMinimizeResumeAfterFinalPublicationValidatesWithoutPublishingAgain(t *testing.T) {
	artifactPath := minimizationParent(t)
	uninterrupted := uninterruptedMinimization(t, artifactPath)
	outputRoot := t.TempDir()
	const finalValidationReplay = 2
	interrupted := minimizationSpec(t, artifactPath, outputRoot)
	interrupted.Replayer = &minimizationReplayer{before: interruptAt(finalValidationReplay)}
	if _, err := Minimize(context.Background(), interrupted); !errors.Is(err, errMinimizationInterrupted) {
		t.Fatalf("interrupted Minimize() error = %v", err)
	}
	published := publishedMinimizedArtifacts(t, outputRoot)
	if len(published) != 1 {
		t.Fatalf("published artifacts before resume = %v", published)
	}

	spec := minimizationSpec(t, artifactPath, outputRoot)
	spec.Resume = true
	result, err := Minimize(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if executor, replayer := spec.Executor.(*minimizationExecutor), spec.Replayer.(*minimizationReplayer); executor.calls != 0 || replayer.calls != 1 {
		t.Fatalf("resume evaluated %d candidates and replayed %d artifacts", executor.calls, replayer.calls)
	}
	if resumed := minimizationOutcomeOf(result); !reflect.DeepEqual(resumed, uninterrupted) {
		t.Fatalf("resumed outcome = %#v, want %#v", resumed, uninterrupted)
	}
	if after := publishedMinimizedArtifacts(t, outputRoot); !reflect.DeepEqual(after, published) || filepath.Base(result.Artifact.Path) != published[0] {
		t.Fatalf("published artifacts after resume = %v, before = %v, result = %s", after, published, result.Artifact.Path)
	}
}

type minimizationExecutor struct {
	mu    sync.Mutex
	calls int
	// before runs ahead of each call with its 1-based number; an error interrupts the run there.
	before func(call int) error
}

func (executor *minimizationExecutor) Run(_ context.Context, request execution.Spec) (execution.Result, error) {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	executor.calls++
	if executor.before != nil {
		if err := executor.before(executor.calls); err != nil {
			return execution.Result{}, err
		}
	}
	trace, err := minimizationChoiceTrace(request)
	if err != nil {
		return execution.Result{}, err
	}
	var retained struct {
		Overrides []struct {
			Dimension simulationengine.Dimension `json:"dimension"`
		} `json:"overrides"`
	}
	if err := json.Unmarshal(request.Simulation.ExplorationPlan, &retained); err != nil {
		return execution.Result{}, err
	}
	hasRuntime := false
	hasFault := false
	for _, override := range retained.Overrides {
		hasRuntime = hasRuntime || override.Dimension == simulationengine.DimensionRuntime
		hasFault = hasFault || override.Dimension == simulationengine.DimensionFault
	}
	failure := record.HashBytes([]byte("different failure"))
	if hasRuntime {
		failure = record.HashBytes([]byte("normalized replay failure"))
	}
	var decisions []simulationengine.Decision
	if hasFault {
		fault, decisionErr := simulationengine.CanonicalDecision(
			simulationengine.DimensionFault, 0, record.HashBytes([]byte("fault site")),
			[]record.SHA256{record.HashBytes([]byte("no fault")), record.HashBytes([]byte("drop"))}, 1,
		)
		if decisionErr != nil {
			return execution.Result{}, decisionErr
		}
		decisions = []simulationengine.Decision{fault}
	}
	record, err := json.Marshal(struct {
		Schema               string                      `json:"schema"`
		Seed                 uint64                      `json:"seed"`
		SpecSHA256           record.SHA256               `json:"spec_sha256"`
		Outcome              string                      `json:"outcome"`
		FailureIdentity      record.SHA256               `json:"failure_identity"`
		ExplorationPlan      json.RawMessage             `json:"exploration_plan"`
		ExplorationDecisions []simulationengine.Decision `json:"exploration_decisions,omitempty"`
		Identity             record.SHA256               `json:"identity"`
	}{
		Schema: "gomad3.cluster-record/v7", Seed: 7, SpecSHA256: record.HashBytes([]byte("simulation spec")),
		Outcome: "oracle_failed", FailureIdentity: failure, ExplorationPlan: request.Simulation.ExplorationPlan,
		ExplorationDecisions: decisions, Identity: record.HashBytes([]byte("simulation record")),
	})
	if err != nil {
		return execution.Result{}, err
	}
	empty := sha256.Sum256(nil)
	exitCode := 2
	return execution.Result{
		Termination: execution.TerminationExit, ExitCode: exitCode, GroupGone: true,
		Stdout: replayOutput("recorded stdout"), Stderr: replayOutput("recorded stderr"),
		IOTranscript: deterministicio.Transcript{SHA256: empty, Complete: true}, ChoiceTrace: trace,
		SimulationRecords: [][]byte{record},
	}, nil
}

func minimizationChoiceTrace(request execution.Spec) (execution.ChoiceTrace, error) {
	first := sha256.Sum256([]byte("first choice alternative"))
	second := sha256.Sum256([]byte("second choice alternative"))
	decision, err := choice.CanonicalDecision(0, choice.KindRunnable, 17, false, [][sha256.Size]byte{first, second}, second, 0)
	if err != nil {
		return execution.ChoiceTrace{}, err
	}
	trace, err := choice.BuildTrace([]choice.Record{decision.Record()}, choice.TerminalComplete)
	if err != nil {
		return execution.ChoiceTrace{}, err
	}
	return execution.ChoiceTrace{
		Profile: choice.Profile, ImplementationSHA256: request.Choice.ImplementationSHA256,
		Limit: request.Choice.Limit, Trace: trace,
	}, nil
}

type minimizationReplayer struct {
	calls  int
	before func(call int) error
}

func (replayer *minimizationReplayer) Replay(_ context.Context, spec ReplaySpec) (ReplayResult, error) {
	replayer.calls++
	if replayer.before != nil {
		if err := replayer.before(replayer.calls); err != nil {
			return ReplayResult{}, err
		}
	}
	opened, err := artifact.OpenArtifact(spec.ArtifactPath)
	if err != nil {
		return ReplayResult{}, err
	}
	detached := opened.Detached()
	if err := opened.Close(); err != nil {
		return ReplayResult{}, err
	}
	return ReplayResult{Artifact: detached, Verified: true, Match: true, ChoiceReplayStatus: ChoiceReplayExact}, nil
}
