package runner

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"sync"
	"testing"

	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/combinedfrontier"
	"go.temporal.io/server/tools/gomadv3/runner/internal/execution"
)

func TestMinimizePublishesLinkedExactScheduleAndFaultReduction(t *testing.T) {
	artifactPath, _ := publishReplayArtifactForTarget(t, nil, replayArtifactTarget{Choices: true, Simulation: true, ForcedSimulation: true})
	parent, err := evidence.OpenArtifact(artifactPath)
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
	reopened, err := evidence.OpenArtifact(artifactPath)
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

type minimizationExecutor struct {
	mu    sync.Mutex
	calls int
}

func (executor *minimizationExecutor) Run(_ context.Context, request execution.Spec) (execution.Result, error) {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	executor.calls++
	trace, err := minimizationChoiceTrace(request)
	if err != nil {
		return execution.Result{}, err
	}
	var retained struct {
		Overrides []struct {
			Dimension combinedfrontier.Dimension `json:"dimension"`
		} `json:"overrides"`
	}
	if err := json.Unmarshal(request.Simulation.ExplorationPlan, &retained); err != nil {
		return execution.Result{}, err
	}
	hasRuntime := false
	hasFault := false
	for _, override := range retained.Overrides {
		hasRuntime = hasRuntime || override.Dimension == combinedfrontier.DimensionRuntime
		hasFault = hasFault || override.Dimension == combinedfrontier.DimensionFault
	}
	failure := evidence.HashBytes([]byte("different failure"))
	if hasRuntime {
		failure = evidence.HashBytes([]byte("normalized replay failure"))
	}
	var decisions []combinedfrontier.Decision
	if hasFault {
		fault, decisionErr := combinedfrontier.CanonicalDecision(
			combinedfrontier.DimensionFault, 0, evidence.HashBytes([]byte("fault site")),
			[]evidence.SHA256{evidence.HashBytes([]byte("no fault")), evidence.HashBytes([]byte("drop"))}, 1,
		)
		if decisionErr != nil {
			return execution.Result{}, decisionErr
		}
		decisions = []combinedfrontier.Decision{fault}
	}
	record, err := json.Marshal(struct {
		Schema               string                      `json:"schema"`
		Seed                 uint64                      `json:"seed"`
		SpecSHA256           evidence.SHA256             `json:"spec_sha256"`
		Outcome              string                      `json:"outcome"`
		FailureIdentity      evidence.SHA256             `json:"failure_identity"`
		ExplorationPlan      json.RawMessage             `json:"exploration_plan"`
		ExplorationDecisions []combinedfrontier.Decision `json:"exploration_decisions,omitempty"`
		Identity             evidence.SHA256             `json:"identity"`
	}{
		Schema: "gomadv3.cluster-record/v7", Seed: 7, SpecSHA256: evidence.HashBytes([]byte("simulation spec")),
		Outcome: "oracle_failed", FailureIdentity: failure, ExplorationPlan: request.Simulation.ExplorationPlan,
		ExplorationDecisions: decisions, Identity: evidence.HashBytes([]byte("simulation record")),
	})
	if err != nil {
		return execution.Result{}, err
	}
	empty := sha256.Sum256(nil)
	exitCode := 2
	return execution.Result{
		Termination: execution.TerminationExit, ExitCode: exitCode, GroupGone: true,
		Stdout: replayOutput("recorded stdout"), Stderr: replayOutput("recorded stderr"),
		IOTranscript: execution.IOTranscript{SHA256: empty, Complete: true}, ChoiceTrace: trace,
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
	calls int
}

func (replayer *minimizationReplayer) Replay(_ context.Context, spec ReplaySpec) (ReplayResult, error) {
	replayer.calls++
	opened, err := evidence.OpenArtifact(spec.ArtifactPath)
	if err != nil {
		return ReplayResult{}, err
	}
	detached := opened.Detached()
	if err := opened.Close(); err != nil {
		return ReplayResult{}, err
	}
	return ReplayResult{Artifact: detached, Verified: true, Match: true, ChoiceReplayStatus: ChoiceReplayExact}, nil
}
