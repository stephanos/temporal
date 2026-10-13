package runner

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	backend "go.temporal.io/server/tools/gomad3/internal/backends"
	"go.temporal.io/server/tools/gomad3/internal/backends/target"
	"go.temporal.io/server/tools/gomad3/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/host/fs"
	"go.temporal.io/server/tools/gomad3/internal/model/choice"
	"go.temporal.io/server/tools/gomad3/internal/runner/internal/exploration"
	"go.temporal.io/server/tools/gomad3/internal/runner/internal/minimizer"
)

func TestMinimizePublishesLinkedExactScheduleAndFaultReduction(t *testing.T) {
	artifactPath, _ := publishReplayArtifactForTarget(t, nil, replayArtifactTarget{Choices: true, Simulation: true, ForcedSimulation: true})
	parent, err := artifact.OpenArtifact(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	parentRecordHash := parent.Manifest().RecordHash
	parentFailureSignature := parent.Manifest().Outcome.FailureSignature
	if err := parent.Close(); err != nil {
		t.Fatal(err)
	}
	executor := &minimizationExecutor{}
	replayer := &minimizationReplayer{}

	result, err := minimizeWith(context.Background(), MinimizeSpec{
		ArtifactPath: artifactPath, OutputRoot: t.TempDir(), AttemptBudget: 16,
		ToolchainRoot: toolchainRoot(t), Backend: replayNativeProvider(t), Replayer: replayer,
	}, executionDependencies{executor: executor},
	)
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
	if reopened.Manifest().RecordHash != parentRecordHash || reopened.Manifest().Minimization != nil {
		t.Fatalf("parent artifact changed = %#v", reopened.Manifest())
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestMinimizeRejectsSimulationArtifactWithoutExactChoiceTape(t *testing.T) {
	artifactPath, _ := publishReplayArtifactForTarget(t, nil, replayArtifactTarget{Simulation: true})
	_, err := minimizeWith(context.Background(), MinimizeSpec{
		ArtifactPath: artifactPath, OutputRoot: t.TempDir(), AttemptBudget: 16,
		ToolchainRoot: toolchainRoot(t), Backend: replayNativeProvider(t), Replayer: &minimizationReplayer{},
	}, executionDependencies{executor: &minimizationExecutor{}},
	)
	if err == nil {
		t.Fatal("minimizeWith() accepted an artifact without an exact choice tape")
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

// otherMinimizationParent returns a parent whose record hash differs from minimizationParent's.
func otherMinimizationParent(t *testing.T) string {
	t.Helper()
	return minimizationParent(t, record.Environment{Name: "GOMAD_TEST_OTHER_PARENT", Value: "1"})
}

func minimizationSpec(t *testing.T, artifactPath, outputRoot string) (MinimizeSpec, executionDependencies) {
	t.Helper()
	return MinimizeSpec{
		ArtifactPath: artifactPath, OutputRoot: outputRoot, AttemptBudget: 16,
		ToolchainRoot: toolchainRoot(t), Backend: replayNativeProvider(t), Replayer: &minimizationReplayer{},
	}, executionDependencies{executor: &minimizationExecutor{}}
}

func uninterruptedMinimization(t *testing.T, artifactPath string) minimizationOutcome {
	t.Helper()
	spec, dependencies := minimizationSpec(t, artifactPath, t.TempDir())
	result, err := minimizeWith(context.Background(), spec, dependencies)
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
	spec, specDependencies := minimizationSpec(t, artifactPath, outputRoot)
	specDependencies.executor = &minimizationExecutor{before: interruptAt(minimizationCallAfterAcceptedReduction)}
	if _, err := minimizeWith(context.Background(), spec, specDependencies); !errors.Is(err, errMinimizationInterrupted) {
		t.Fatalf("interrupted minimizeWith() error = %v", err)
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

	spec, specDependencies := minimizationSpec(t, artifactPath, outputRoot)
	if _, err := minimizeWith(context.Background(), spec, specDependencies); !errors.Is(err, minimizer.ErrCheckpointExists) {
		t.Fatalf("minimizeWith() over existing state error = %v", err)
	}
	if calls := specDependencies.executor.(*minimizationExecutor).calls; calls != 0 {
		t.Fatalf("refused run evaluated %d candidates", calls)
	}

	spec.Resume = true
	result, err := minimizeWith(context.Background(), spec, specDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if calls := specDependencies.executor.(*minimizationExecutor).calls; calls != int(uninterrupted.Attempts)-minimizationCallAfterAcceptedReduction+1 {
		t.Fatalf("resumed run evaluated %d candidates of %d attempts", calls, uninterrupted.Attempts)
	}
	if resumed := minimizationOutcomeOf(result); !resumed.Changed || !reflect.DeepEqual(resumed, uninterrupted) {
		t.Fatalf("resumed outcome = %#v, want %#v", resumed, uninterrupted)
	}
	if published := publishedMinimizedArtifacts(t, outputRoot); !reflect.DeepEqual(published, []string{filepath.Base(result.Artifact.Path)}) {
		t.Fatalf("published artifacts = %v, result = %s", published, result.Artifact.Path)
	}
	spec.Resume = false
	if _, err := minimizeWith(context.Background(), spec, specDependencies); err != nil {
		t.Fatalf("minimizeWith() after a completed run: %v", err)
	}
}

func TestMinimizeKeepsStatePerParentArtifactInOneOutputRoot(t *testing.T) {
	artifactPath, otherPath := minimizationParent(t), otherMinimizationParent(t)
	uninterrupted, otherUninterrupted := uninterruptedMinimization(t, artifactPath), uninterruptedMinimization(t, otherPath)
	if uninterrupted.RecordHash == otherUninterrupted.RecordHash {
		t.Fatalf("both parents minimize to record %s", uninterrupted.RecordHash)
	}
	outputRoot := interruptedMinimization(t, artifactPath)

	otherSpec, otherDependencies := minimizationSpec(t, otherPath, outputRoot)
	otherResult, err := minimizeWith(context.Background(), otherSpec, otherDependencies)
	if err != nil {
		t.Fatalf("minimizeWith() of another parent beside interrupted state: %v", err)
	}
	if other := minimizationOutcomeOf(otherResult); !reflect.DeepEqual(other, otherUninterrupted) {
		t.Fatalf("other parent outcome = %#v, want %#v", other, otherUninterrupted)
	}

	spec, specDependencies := minimizationSpec(t, artifactPath, outputRoot)
	spec.Resume = true
	result, err := minimizeWith(context.Background(), spec, specDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if calls := specDependencies.executor.(*minimizationExecutor).calls; calls != int(uninterrupted.Attempts)-minimizationCallAfterAcceptedReduction+1 {
		t.Fatalf("resumed run evaluated %d candidates of %d attempts", calls, uninterrupted.Attempts)
	}
	if resumed := minimizationOutcomeOf(result); !reflect.DeepEqual(resumed, uninterrupted) {
		t.Fatalf("resumed outcome = %#v, want %#v", resumed, uninterrupted)
	}
	want := []string{filepath.Base(result.Artifact.Path), filepath.Base(otherResult.Artifact.Path)}
	sort.Strings(want)
	if published := publishedMinimizedArtifacts(t, outputRoot); !reflect.DeepEqual(published, want) {
		t.Fatalf("published artifacts = %v, want %v", published, want)
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
			spec.ArtifactPath = otherMinimizationParent(t)
		}, want: minimizer.ErrNoCheckpoint.Error()},
		{name: "changed attempt budget", change: func(spec *MinimizeSpec) { spec.AttemptBudget++ }, want: "attempt budget 16, not 17"},
		{name: "no state", change: func(spec *MinimizeSpec) { spec.OutputRoot = t.TempDir() }, want: minimizer.ErrNoCheckpoint.Error()},
		{name: "no output directory", change: func(spec *MinimizeSpec) { spec.OutputRoot = filepath.Join(t.TempDir(), "absent") }, want: minimizer.ErrNoCheckpoint.Error()},
	} {
		t.Run(test.name, func(t *testing.T) {
			spec, specDependencies := minimizationSpec(t, artifactPath, outputRoot)
			spec.Resume = true
			test.change(&spec)
			_, err := minimizeWith(context.Background(), spec, specDependencies)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("minimizeWith() error = %v, want %q", err, test.want)
			}
			if calls := specDependencies.executor.(*minimizationExecutor).calls; calls != 0 {
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
			accepted, err := filepath.Glob(filepath.Join(outputRoot, ".minimize", "sha256-*", "accepted", "sha256-*"))
			if err != nil || len(accepted) != 1 {
				t.Fatalf("accepted artifacts = %v, %v", accepted, err)
			}
			test.damage(t, accepted[0])
			spec, specDependencies := minimizationSpec(t, artifactPath, outputRoot)
			spec.Resume = true
			if _, err := minimizeWith(context.Background(), spec, specDependencies); err == nil {
				t.Fatal("minimizeWith() resumed from a damaged accepted artifact")
			}
			if calls := specDependencies.executor.(*minimizationExecutor).calls; calls != 0 || len(publishedMinimizedArtifacts(t, outputRoot)) != 0 {
				t.Fatalf("damaged resume evaluated %d candidates, published %v", calls, publishedMinimizedArtifacts(t, outputRoot))
			}
		})
	}
}

func TestMinimizeExcludesConcurrentRunsOfOneParentOnOneOutputRoot(t *testing.T) {
	artifactPath, otherPath := minimizationParent(t), otherMinimizationParent(t)
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
			holder, holderDependencies := minimizationSpec(t, artifactPath, outputRoot)
			holder.Resume = test.resume
			holderDependencies.executor = &minimizationExecutor{before: func(call int) error {
				if call == 1 {
					close(evaluating)
					<-release
				}
				return nil
			}}
			held := make(chan error, 1)
			go func() {
				_, err := minimizeWith(context.Background(), holder, holderDependencies)
				held <- err
			}()
			<-evaluating
			contender, contenderDependencies := minimizationSpec(t, artifactPath, outputRoot)
			contender.Resume = test.resume
			_, err := minimizeWith(context.Background(), contender, contenderDependencies)
			otherSpec, otherDependencies := minimizationSpec(t, otherPath, outputRoot)
			_, otherErr := minimizeWith(context.Background(), otherSpec, otherDependencies)
			close(release)
			if !errors.Is(err, hostfs.ErrContended) {
				t.Fatalf("concurrent minimizeWith() error = %v", err)
			}
			if otherErr != nil {
				t.Fatalf("concurrent minimizeWith() of another parent: %v", otherErr)
			}
			if calls := contenderDependencies.executor.(*minimizationExecutor).calls; calls != 0 {
				t.Fatalf("rejected run evaluated %d candidates", calls)
			}
			if err := <-held; err != nil {
				t.Fatalf("lock-holding minimizeWith() error = %v", err)
			}
		})
	}
}

func TestMinimizeResumeAfterFinalPublicationValidatesWithoutPublishingAgain(t *testing.T) {
	artifactPath := minimizationParent(t)
	uninterrupted := uninterruptedMinimization(t, artifactPath)
	outputRoot := t.TempDir()
	const finalValidationReplay = 2
	interrupted, interruptedDependencies := minimizationSpec(t, artifactPath, outputRoot)
	interrupted.Replayer = &minimizationReplayer{before: interruptAt(finalValidationReplay)}
	if _, err := minimizeWith(context.Background(), interrupted, interruptedDependencies); !errors.Is(err, errMinimizationInterrupted) {
		t.Fatalf("interrupted minimizeWith() error = %v", err)
	}
	published := publishedMinimizedArtifacts(t, outputRoot)
	if len(published) != 1 {
		t.Fatalf("published artifacts before resume = %v", published)
	}

	spec, specDependencies := minimizationSpec(t, artifactPath, outputRoot)
	spec.Resume = true
	result, err := minimizeWith(context.Background(), spec, specDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if executor, replayer := specDependencies.executor.(*minimizationExecutor), spec.Replayer.(*minimizationReplayer); executor.calls != 0 || replayer.calls != 1 {
		t.Fatalf("resume evaluated %d candidates and replayed %d artifacts", executor.calls, replayer.calls)
	}
	if resumed := minimizationOutcomeOf(result); !reflect.DeepEqual(resumed, uninterrupted) {
		t.Fatalf("resumed outcome = %#v, want %#v", resumed, uninterrupted)
	}
	if after := publishedMinimizedArtifacts(t, outputRoot); !reflect.DeepEqual(after, published) || filepath.Base(result.Artifact.Path) != published[0] {
		t.Fatalf("published artifacts after resume = %v, before = %v, result = %s", after, published, result.Artifact.Path)
	}
}

func TestMinimizeResumeAfterUnrecordedFinalPublicationDoesNotPublishAgain(t *testing.T) {
	artifactPath := minimizationParent(t)
	uninterrupted := uninterruptedMinimization(t, artifactPath)
	outputRoot := t.TempDir()

	// A run killed between the final publication and the checkpoint that
	// records it leaves the published artifact and a state without it.
	spec, dependencies := minimizationSpec(t, artifactPath, outputRoot)
	session, err := openMinimizationSession(context.Background(), spec, dependencies)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.reduce(context.Background()); err != nil {
		t.Fatal(err)
	}
	killed := session.workspace.Checkpoint()
	if _, err := session.publishAccepted(context.Background(), killed); err != nil {
		t.Fatal(err)
	}
	if persisted := session.workspace.Checkpoint(); !reflect.DeepEqual(persisted, killed) || persisted.Published != nil {
		t.Fatalf("state after the unrecorded publication = %#v, want %#v", persisted, killed)
	}
	if err := session.close(); err != nil {
		t.Fatal(err)
	}
	published := publishedMinimizedArtifacts(t, outputRoot)
	if len(published) != 1 {
		t.Fatalf("published artifacts before resume = %v", published)
	}

	spec, specDependencies := minimizationSpec(t, artifactPath, outputRoot)
	spec.Resume = true
	result, err := minimizeWith(context.Background(), spec, specDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if executor, replayer := specDependencies.executor.(*minimizationExecutor), spec.Replayer.(*minimizationReplayer); executor.calls != 0 || replayer.calls != 1 {
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

func (executor *minimizationExecutor) Run(_ context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	executor.calls++
	if executor.before != nil {
		if err := executor.before(executor.calls); err != nil {
			return backend.ExecutionResult{}, err
		}
	}
	trace, err := minimizationChoiceTrace(request)
	if err != nil {
		return backend.ExecutionResult{}, err
	}
	var retained struct {
		Overrides []struct {
			Dimension exploration.Dimension `json:"dimension"`
		} `json:"overrides"`
	}
	if err := json.Unmarshal(request.Simulation.ExplorationPlan, &retained); err != nil {
		return backend.ExecutionResult{}, err
	}
	hasRuntime := false
	hasFault := false
	for _, override := range retained.Overrides {
		hasRuntime = hasRuntime || override.Dimension == exploration.DimensionRuntime
		hasFault = hasFault || override.Dimension == exploration.DimensionFault
	}
	failure := record.HashBytes([]byte("different failure"))
	if hasRuntime {
		failure = record.HashBytes([]byte("normalized replay failure"))
	}
	var decisions []exploration.Decision
	if hasFault {
		fault, decisionErr := exploration.CanonicalDecision(
			exploration.DimensionFault, 0, record.HashBytes([]byte("fault site")),
			[]record.SHA256{record.HashBytes([]byte("no fault")), record.HashBytes([]byte("drop"))}, 1,
		)
		if decisionErr != nil {
			return backend.ExecutionResult{}, decisionErr
		}
		decisions = []exploration.Decision{fault}
	}
	record, err := json.Marshal(struct {
		Schema               string                 `json:"schema"`
		Seed                 uint64                 `json:"seed"`
		SpecSHA256           record.SHA256          `json:"spec_sha256"`
		Outcome              string                 `json:"outcome"`
		FailureIdentity      record.SHA256          `json:"failure_identity"`
		ExplorationPlan      json.RawMessage        `json:"exploration_plan"`
		ExplorationDecisions []exploration.Decision `json:"exploration_decisions,omitempty"`
		Identity             record.SHA256          `json:"identity"`
	}{
		Schema: "gomad3.cluster-record/v8", Seed: 7, SpecSHA256: record.HashBytes([]byte("simulation spec")),
		Outcome: "oracle_failed", FailureIdentity: failure, ExplorationPlan: request.Simulation.ExplorationPlan,
		ExplorationDecisions: decisions, Identity: record.HashBytes([]byte("simulation record")),
	})
	if err != nil {
		return backend.ExecutionResult{}, err
	}
	empty := sha256.Sum256(nil)
	exitCode := 2
	return backend.ExecutionResult{
		Termination: backend.TerminationExit, ExitCode: exitCode, GroupGone: true,
		Stdout: replayOutput("recorded stdout"), Stderr: replayOutput("recorded stderr"),
		IOTranscript: backend.Transcript{SHA256: empty, Complete: true}, ChoiceTrace: trace,
		SimulationRecords: [][]byte{record},
	}, nil
}

func minimizationChoiceTrace(request backend.ExecutionRequest) (backend.ChoiceTrace, error) {
	first := sha256.Sum256([]byte("first choice alternative"))
	second := sha256.Sum256([]byte("second choice alternative"))
	decision, err := choice.CanonicalDecision(0, choice.KindRunnable, 17, false, [][sha256.Size]byte{first, second}, second, 0)
	if err != nil {
		return backend.ChoiceTrace{}, err
	}
	trace, err := choice.BuildTrace([]choice.Record{decision.Record()}, choice.TerminalComplete)
	if err != nil {
		return backend.ChoiceTrace{}, err
	}
	return backend.ChoiceTrace{
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
	detached := opened.Snapshot()
	if err := opened.Close(); err != nil {
		return ReplayResult{}, err
	}
	return ReplayResult{Artifact: detached, Verified: true, Match: true, ChoiceReplayStatus: ChoiceReplayExact}, nil
}

type prefixFailureProvider struct {
	boundaryProvider
	controls       []int
	replayMismatch bool
	environment    []string
}

type cancellationMinimizationProvider struct {
	*prefixFailureProvider
	cancel      context.CancelFunc
	cancelAt    int
	validations int
	observed    error
}

func (p *prefixFailureProvider) Execute(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	return executeBoundaryFixture(ctx, p, request)
}

func (p *cancellationMinimizationProvider) ValidateReplay(ctx context.Context, _ *artifact.Opened) (target.Prepared, error) {
	p.validations++
	if p.validations == p.cancelAt {
		p.cancel()
	}
	p.observed = ctx.Err()
	return p.prepared, p.observed
}

func TestBackendMinimizePreservesCancellationThroughRetainedPreflightAndPreparation(t *testing.T) {
	for _, cancelAt := range []int{1, 2} {
		t.Run([]string{"preflight", "preparation"}[cancelAt-1], func(t *testing.T) {
			parent, provider := backendMinimizationParent(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			observed := &cancellationMinimizationProvider{prefixFailureProvider: provider, cancel: cancel, cancelAt: cancelAt}
			_, err := Minimize(ctx, MinimizeSpec{Backend: observed, ArtifactPath: parent.Path, OutputRoot: t.TempDir(), AttemptBudget: 8})
			if !errors.Is(err, context.Canceled) || !errors.Is(observed.observed, context.Canceled) || observed.validations != cancelAt || len(provider.controls) != 0 {
				t.Fatalf("retained validation lost caller cancellation: err=%v observed=%v validations=%d controls=%v", err, observed.observed, observed.validations, provider.controls)
			}
		})
	}
}

func (p *prefixFailureProvider) Run(ctx context.Context, request backend.Request) (backend.Result, error) {
	if err := ctx.Err(); err != nil {
		return backend.Result{}, err
	}
	if p.err != nil {
		return backend.Result{}, p.err
	}
	if request.Choice == nil {
		return backend.Result{}, errors.New("missing choice controls")
	}
	length := 0
	if request.Choice.Tape != nil {
		length = len(request.Choice.Tape.Decisions)
	}
	p.controls = append(p.controls, length)
	if request.Target.Backend == nil || !reflect.DeepEqual(request.Target.BackendPayloads, p.prepared.BackendPayloads) || !reflect.DeepEqual(request.Target.Argv, p.prepared.Argv) || !reflect.DeepEqual(request.Environment, p.environment) || request.Seed != 7 || request.Timeout != time.Second || request.OutputBytes != 4096 || request.TranscriptBytes != 64<<20 {
		return backend.Result{}, errors.New("minimization changed retained backend inputs or limits")
	}
	result := p.result
	if p.replayMismatch && request.Choice.Mode == choice.ModeReplay {
		result.Evidence = []byte("different replay observations")
	}
	if length == 0 {
		result.ExitCode = 0
		result.Stdout = []byte("success")
	}
	return result, nil
}

func backendMinimizationParent(t *testing.T) (artifact.Artifact, *prefixFailureProvider) {
	t.Helper()
	prepared := newFakePreparer(t).prepared
	prepared.Argv = []string{"gomad3-target", "fixed", "arguments"}
	provenance := []byte("immutable model and faults")
	prepared.Backend = &record.BackendMetadata{Name: "fixture-cooperative", ReplayMode: record.ReplayExact, Provenance: record.BackendPayload{
		Schema: "fixture/v1", File: "backend/provenance.json", SHA256: record.HashBytes(provenance), Bytes: record.Uint64String(len(provenance)),
	}}
	prepared.BackendPayloads = []target.BackendPayload{{Reference: prepared.Backend.Provenance, Data: provenance}}
	prepared.BuildInfo.Settings = []record.BuildSetting{}
	prepared.CapabilityMode = target.CapabilityModeClosure
	config, _ := testConfig(t, nil, nil, "7", PolicyAll, 1)
	config.ExecutionTimeout, config.OutputLimit, config.ChoiceTraceLimit = time.Second, 4096, 4096
	config.Target.Backend, config.Backend = "fixture-cooperative", &boundaryProvider{}
	_, environment, err := validateConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	trace := completeChoiceTrace(t, prepared.BuildKey, 4096, []choice.Record{
		{Ordinal: 0, Kind: choice.KindRunnable, Flags: choice.FlagDecision, Alternatives: 2, Selected: 0},
		{Ordinal: 1, Kind: choice.KindRunnable, Flags: choice.FlagDecision, Alternatives: 2, Selected: 0},
		{Ordinal: 2, Kind: choice.KindRunnable, Flags: choice.FlagDecision, Alternatives: 2, Selected: 0},
	})
	completion := runCompletion{job: runJob{seed: 7}, startedAt: time.Unix(1, 0), finishedAt: time.Unix(2, 0), result: processResult(17, "failure", "")}
	completion.result.BackendEvidence, completion.result.ChoiceTrace = []byte("model observations"), trace
	world := noneWorldBundle()
	manifest, err := manifestForRun(campaignRequestFromSpec(config), prepared, environment, completion, backend.Classify(completion.result, false, world.Manifest.Terminal), "fixture", world.Manifest, nil)
	if err != nil {
		t.Fatal(err)
	}
	published, err := artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, executionArtifactInput(manifest, prepared, completion.result, nil, world))
	if err != nil {
		t.Fatal(err)
	}
	provider := &prefixFailureProvider{environment: replayEnvironment(manifest.Environment), boundaryProvider: boundaryProvider{prepared: prepared, result: backend.Result{
		Termination: backend.Exit, Reaped: true, ExitCode: 17, Stdout: []byte("failure"), Evidence: []byte("model observations"), ChoiceTrace: trace.Trace, ImplementationSHA256: trace.ImplementationSHA256,
	}}}
	return published, provider
}

func TestBackendMinimizePreservesEssentialPrefixAndDetachedEvidence(t *testing.T) {
	parent, provider := backendMinimizationParent(t)
	result, err := Minimize(t.Context(), MinimizeSpec{Backend: provider, ArtifactPath: parent.Path, OutputRoot: t.TempDir(), AttemptBudget: 8})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Attempts != 2 || len(result.Accepted) != 1 || !reflect.DeepEqual(provider.controls, []int{0, 1, 3, 3}) {
		t.Fatalf("effective prefix order/reductions: result=%+v controls=%v", result, provider.controls)
	}
	evidence := result.Artifact.Manifest.Minimization
	if evidence == nil || evidence.Schema != "gomad3.minimization/choice-prefix/v1" || evidence.OriginalForcedDecisions != 3 || evidence.FinalForcedDecisions != 1 || evidence.Predicate.ChoiceReplay != "exact" || evidence.Predicate.SimulationReplay != "not_present" || result.Artifact.Manifest.SimulationProfile != nil {
		t.Fatalf("choice-only evidence: %+v", evidence)
	}
	replay, err := Replay(t.Context(), ReplaySpec{Backend: provider, ArtifactPath: result.Artifact.Path})
	if err != nil || !replay.Match || replay.ChoiceReplayStatus != ChoiceReplayExact || result.Artifact.Manifest.Outcome.FailureSignature != parent.Manifest.Outcome.FailureSignature {
		t.Fatalf("fresh retained replay: %+v err=%v", replay, err)
	}
	opened, err := artifact.OpenArtifact(parent.Path)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Manifest().RecordHash != parent.Manifest.RecordHash {
		t.Fatal("parent changed")
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestBackendMinimizeResumeUsesCommittedPrefixAndRetainedPayloads(t *testing.T) {
	parent, provider := backendMinimizationParent(t)
	baseline, err := Minimize(t.Context(), MinimizeSpec{Backend: provider, ArtifactPath: parent.Path, OutputRoot: t.TempDir(), AttemptBudget: 8})
	if err != nil {
		t.Fatal(err)
	}
	provider.controls = nil
	root := filepath.Join(t.TempDir(), "minimized")
	spec := MinimizeSpec{Backend: provider, ArtifactPath: parent.Path, OutputRoot: root, AttemptBudget: 8}
	session, err := openMinimizationSession(t.Context(), spec, executionDependencies{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.close(); err != nil {
			t.Error(err)
		}
	})
	if _, err := session.reduce(t.Context()); err != nil {
		t.Fatal(err)
	}
	if session.workspace.Checkpoint().Accepted == nil {
		t.Fatal("missing committed reduction")
	}
	if err := session.close(); err != nil {
		t.Fatal(err)
	}
	provider.controls, spec.Resume = nil, true
	resumed, err := Minimize(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(minimizationOutcomeOf(baseline), minimizationOutcomeOf(resumed)) || !reflect.DeepEqual(provider.controls, []int{3}) {
		t.Fatalf("resume repeated committed work or lost evidence: %+v controls=%v", resumed, provider.controls)
	}
}

func TestBackendMinimizeBindsEffectiveControlsToRetainedTape(t *testing.T) {
	for _, name := range []string{"prefix digest", "removed identity"} {
		t.Run(name, func(t *testing.T) {
			parent, provider := backendMinimizationParent(t)
			result, err := Minimize(t.Context(), MinimizeSpec{Backend: provider, ArtifactPath: parent.Path, OutputRoot: t.TempDir(), AttemptBudget: 8})
			if err != nil {
				t.Fatal(err)
			}
			manifest := result.Artifact.Manifest
			if name == "prefix digest" {
				manifest.Minimization.ChoicePrefix.Final.SHA256 = record.HashBytes([]byte("forged effective tape"))
				identity, err := manifest.Minimization.ChoicePrefix.Final.CandidateSHA256()
				if err != nil {
					t.Fatal(err)
				}
				manifest.Minimization.FinalCandidateSHA256 = identity
				manifest.Minimization.Accepted[0].AfterSHA256 = identity
			} else {
				manifest.Minimization.Accepted[0].Removed[0].Identity = record.HashBytes([]byte("not the removed choice"))
			}
			manifest, encoded, err := record.FinalizeExecutionRecord(manifest)
			if err != nil {
				t.Fatalf("self-consistent forged record rejected without tape: %v", err)
			}
			world := noneWorldBundle()
			payloads := append([]target.BackendPayload(nil), provider.prepared.BackendPayloads...)
			payloads = append(payloads, target.BackendPayload{Reference: *manifest.Target.Backend.Evidence, Data: provider.result.Evidence})
			_, err = artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, artifact.ArtifactInput{Manifest: manifest, TargetPath: provider.prepared.Path, Stdout: provider.result.Stdout, ChoiceTrace: provider.result.ChoiceTrace.Bytes, BackendPayloads: payloads, World: world.Payloads})
			if err == nil || !strings.Contains(err.Error(), "retained tape") {
				t.Fatalf("forged controls published: %v", err)
			}
			if err := os.WriteFile(filepath.Join(result.Artifact.Path, "manifest.json"), encoded, 0600); err != nil {
				t.Fatal(err)
			}
			opened, err := artifact.OpenArtifact(result.Artifact.Path)
			if opened != nil {
				_ = opened.Close()
			}
			if err == nil || !strings.Contains(err.Error(), "retained tape") {
				t.Fatalf("forged controls opened: %v", err)
			}
		})
	}
}

func TestBackendMinimizeRejectsInvalidEvidenceShape(t *testing.T) {
	parent, provider := backendMinimizationParent(t)
	result, err := Minimize(t.Context(), MinimizeSpec{Backend: provider, ArtifactPath: parent.Path, OutputRoot: t.TempDir(), AttemptBudget: 8})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*record.Minimization)
	}{
		{"count", func(m *record.Minimization) { m.ChoicePrefix.Final.Length++ }},
		{"non-suffix", func(m *record.Minimization) { m.Accepted[0].Removed[0].Ordinal = 0 }},
		{"native schema", func(m *record.Minimization) { m.Schema = "gomad3.minimization/v1" }},
		{"fake simulation replay", func(m *record.Minimization) { m.Predicate.SimulationReplay = "exact" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opened, err := artifact.OpenArtifact(result.Artifact.Path)
			if err != nil {
				t.Fatal(err)
			}
			manifest := opened.Manifest()
			if err := opened.Close(); err != nil {
				t.Fatal(err)
			}
			tc.change(manifest.Minimization)
			if _, _, err := record.FinalizeExecutionRecord(manifest); err == nil {
				t.Fatal("invalid minimization evidence admitted")
			}
		})
	}
}

func TestBackendMinimizeInfrastructureFailureIsNotCandidateRejection(t *testing.T) {
	parent, provider := backendMinimizationParent(t)
	provider.err = errors.New("helper infrastructure unavailable")
	_, err := Minimize(t.Context(), MinimizeSpec{Backend: provider, ArtifactPath: parent.Path, OutputRoot: t.TempDir(), AttemptBudget: 8})
	if !errors.Is(err, provider.err) || len(provider.controls) != 0 {
		t.Fatalf("infrastructure error consumed attempt: %v controls=%v", err, provider.controls)
	}
}

func TestBackendMinimizeRejectsIncompatibleArtifactsBeforeExecution(t *testing.T) {
	for _, name := range []string{"observed", "success", "watchdog", "missing choices"} {
		t.Run(name, func(t *testing.T) {
			parent, provider := backendMinimizationParent(t)
			world := noneWorldBundle()
			payloads := append([]target.BackendPayload(nil), provider.prepared.BackendPayloads...)
			payloads = append(payloads, target.BackendPayload{Reference: *parent.Manifest.Target.Backend.Evidence, Data: provider.result.Evidence})
			input := artifact.ArtifactInput{Manifest: parent.Manifest, TargetPath: provider.prepared.Path, Stdout: provider.result.Stdout, ChoiceTrace: provider.result.ChoiceTrace.Bytes, BackendPayloads: payloads, World: world.Payloads}
			switch name {
			case "observed":
				input.Manifest.Target.Backend.ReplayMode, input.Manifest.ReplayMode = record.ReplayObserved, record.ReplayObserved
				input.Manifest.ChoiceProfile, input.ChoiceTrace = nil, nil
			case "success":
				code := record.Uint64String(0)
				input.Manifest.ArtifactKind = record.ArtifactSuccess
				input.Manifest.Outcome.Domain, input.Manifest.Outcome.Reason, input.Manifest.Outcome.ExitCode = "success", "zero_exit", &code
			case "watchdog":
				deadline := "execution"
				input.Manifest.ArtifactKind, input.Manifest.ReplayMode = record.ArtifactWatchdogTimeout, record.ReplayDiagnostic
				input.Manifest.Outcome = record.Outcome{Domain: "watchdog", Reason: "watchdog_timeout", Termination: "timeout", Deadline: &deadline}
				input.Manifest.ChoiceProfile, input.ChoiceTrace = nil, nil
			case "missing choices":
				input.Manifest.ChoiceProfile, input.ChoiceTrace = nil, nil
			}
			if input.Manifest.ChoiceProfile == nil {
				input.Manifest.Limits.ChoiceTraceBytes = 0
				environment := make([]record.Environment, 0, len(input.Manifest.Environment))
				for _, entry := range input.Manifest.Environment {
					if entry.Name != "GOMAD3_CHOICE_PROFILE" {
						environment = append(environment, entry)
					}
				}
				input.Manifest.Environment = environment
			}
			incompatible, err := artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, input)
			if err != nil {
				t.Fatal(err)
			}
			_, err = Minimize(t.Context(), MinimizeSpec{Backend: provider, ArtifactPath: incompatible.Path, OutputRoot: t.TempDir(), AttemptBudget: 8})
			if err == nil || len(provider.controls) != 0 {
				t.Fatalf("incompatible artifact executed: %v controls=%v", err, provider.controls)
			}
		})
	}
}

func TestBackendMinimizeRejectsChangedFailureOrReplayEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*prefixFailureProvider)
	}{
		{"outcome", func(p *prefixFailureProvider) { p.result.ExitCode = 18 }},
		{"output", func(p *prefixFailureProvider) { p.result.Stdout = []byte("other failure") }},
		{"backend observations", func(p *prefixFailureProvider) { p.result.Evidence = []byte("changed fixed model effects") }},
		{"fresh replay", func(p *prefixFailureProvider) { p.replayMismatch = true }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent, provider := backendMinimizationParent(t)
			tc.change(provider)
			result, err := Minimize(t.Context(), MinimizeSpec{Backend: provider, ArtifactPath: parent.Path, OutputRoot: t.TempDir(), AttemptBudget: 8})
			if err != nil || result.Changed || result.Artifact.Manifest.RecordHash != parent.Manifest.RecordHash || len(result.Accepted) != 0 {
				t.Fatalf("changed failure accepted: result=%+v err=%v", result, err)
			}
		})
	}
}

func TestMinimizeKeepsOneTargetCopyInItsOutputRoot(t *testing.T) {
	artifactPath := minimizationParent(t)
	targetPath, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	target, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	outputRoot := t.TempDir()
	replayer := &sharedTargetReplayer{ArtifactReplayer: &minimizationReplayer{}}
	spec, specDependencies := minimizationSpec(t, artifactPath, outputRoot)
	specDependencies.executor = &minimizationExecutor{before: interruptAt(minimizationCallAfterAcceptedReduction)}
	spec.Replayer = replayer
	if _, err := minimizeWith(context.Background(), spec, specDependencies); !errors.Is(err, errMinimizationInterrupted) {
		t.Fatalf("interrupted minimizeWith() error = %v", err)
	}
	paths, accepted := retainedTargets(t, outputRoot, target)
	if accepted != 1 || len(paths) != 2 || distinctFiles(t, paths) != 1 {
		t.Fatalf("interrupted run holds %d accepted artifacts in %v, want one sharing the pool entry", accepted, paths)
	}
	specDependencies.executor = &minimizationExecutor{}
	spec.Resume = true
	result, err := minimizeWith(context.Background(), spec, specDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if result.Artifact.TargetSharing != artifact.TargetShared {
		t.Fatalf("minimized artifact target sharing = %q", result.Artifact.TargetSharing)
	}
	if replayer.replayed < 2 || len(replayer.private) != 0 {
		t.Fatalf("%d replayed candidates, private target copies in %v", replayer.replayed, replayer.private)
	}
	paths, minimized := retainedTargets(t, outputRoot, target)
	if minimized != 1 || len(paths) != 2 || distinctFiles(t, paths) != 1 {
		t.Fatalf("completed run holds %d minimized artifacts in %v, want one sharing the pool entry", minimized, paths)
	}
}
