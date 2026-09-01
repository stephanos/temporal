package executor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/evaluationcontract"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	"go.temporal.io/server/tools/umpire/portableevaluation"
	"go.temporal.io/server/tools/umpire/runner"
	"google.golang.org/protobuf/proto"
)

const fixtureRunIdentity = "umpire.local.caller-closure.closed-fixture"

type executeResult struct {
	response *umpirespb.ExecuteResponse
	err      error
}

func TestExecuteCompletesTheAdmittedContractThroughOneInterface(t *testing.T) {
	executor := newExecutor(
		nil,
		fixtureRunner(t, "normal"),
		portableevaluation.Evaluate,
		func() string { return fixtureRunIdentity },
	)

	response, err := executor.Execute(context.Background(), fixtureRequest(t, "normal"))

	require.NoError(t, err)
	result := response.GetResult()
	require.Equal(t, fixtureRunIdentity, result.GetRunIdentity())
	require.Equal(t, umpirespb.TOOLING_STATUS_SUCCEEDED, result.GetToolingStatus())
	require.Equal(t, umpirespb.OPERATIONAL_STATUS_SUCCEEDED, result.GetOperationalStatus())
	require.Equal(t, umpirespb.OBSERVATION_STATUS_ACCEPTED, result.GetObservation().GetStatus())
	require.Equal(t, umpirespb.IMPLEMENTATION_LINK_STATUS_APPLIED, result.GetImplementationLink().GetStatus())
	require.Equal(t, umpirespb.EVALUATION_STATUS_SATISFIED, result.GetSemanticStatus())
	require.Equal(t, umpirespb.CLEANUP_STATUS_COMPLETE, result.GetCleanupStatus())
	require.Equal(t, umpirespb.CANARY_DECISION_PASS, result.GetDecision())
}

func TestExecutePreservesIndependentStatusesAtTheEvidenceRecordBoundary(t *testing.T) {
	records := int64(len(fixtureRawEvidence(t, "normal").Facts))
	tests := []struct {
		name            string
		limit           int64
		wantObservation umpirespb.ObservationStatus
		wantLink        umpirespb.ImplementationLinkStatus
		wantSemantic    umpirespb.EvaluationStatus
		wantDecision    umpirespb.CanaryDecision
	}{
		{
			name: "exact N", limit: records,
			wantObservation: umpirespb.OBSERVATION_STATUS_ACCEPTED,
			wantLink:        umpirespb.IMPLEMENTATION_LINK_STATUS_APPLIED,
			wantSemantic:    umpirespb.EVALUATION_STATUS_SATISFIED,
			wantDecision:    umpirespb.CANARY_DECISION_PASS,
		},
		{
			name: "N plus one", limit: records - 1,
			wantObservation: umpirespb.OBSERVATION_STATUS_UNKNOWN,
			wantLink:        umpirespb.IMPLEMENTATION_LINK_STATUS_NOT_EVALUATED,
			wantSemantic:    umpirespb.EVALUATION_STATUS_INCOMPLETE,
			wantDecision:    umpirespb.CANARY_DECISION_INCONCLUSIVE,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executor := newExecutor(
				nil,
				fixtureRunner(t, "normal"),
				portableevaluation.Evaluate,
				func() string { return fixtureRunIdentity },
			)
			request := fixtureRequestWithContract(t, "normal", func(contract *umpirespb.EvaluationContract) {
				contract.Limits.MaxEvidenceRecords = test.limit
				for _, cardinality := range contract.Observation.Profile.Cardinalities {
					if cardinality.Maximum > test.limit {
						cardinality.Maximum = test.limit
					}
				}
			})

			response, err := executor.Execute(context.Background(), request)

			require.NoError(t, err)
			result := response.GetResult()
			require.Equal(t, umpirespb.TOOLING_STATUS_SUCCEEDED, result.GetToolingStatus())
			require.Equal(t, umpirespb.OPERATIONAL_STATUS_SUCCEEDED, result.GetOperationalStatus())
			require.Equal(t, test.wantObservation, result.GetObservation().GetStatus())
			require.Equal(t, test.wantLink, result.GetImplementationLink().GetStatus())
			require.Equal(t, test.wantSemantic, result.GetSemanticStatus())
			require.Equal(t, umpirespb.CLEANUP_STATUS_COMPLETE, result.GetCleanupStatus())
			require.Equal(t, test.wantDecision, result.GetDecision())
		})
	}
}

func TestExecuteWaitsForExplicitSourceClosure(t *testing.T) {
	closure := make(chan struct{})
	returned := make(chan executeResult, 1)
	run := fixtureRunner(t, "normal")
	request := fixtureRequest(t, "normal")
	executor := newExecutor(nil, func(
		ctx context.Context,
		input artifact.AdmittedSet,
		binding runner.InputBinding,
		runIdentity string,
		adapter runner.Adapter,
	) (runOutcome, error) {
		select {
		case <-closure:
			return run(ctx, input, binding, runIdentity, adapter)
		case <-ctx.Done():
			return runOutcome{reusable: true}, ctx.Err()
		}
	}, portableevaluation.Evaluate, func() string { return fixtureRunIdentity })

	go func() {
		response, err := executor.Execute(context.Background(), request)
		returned <- executeResult{response: response, err: err}
	}()
	require.Never(t, func() bool { return len(returned) != 0 }, 50*time.Millisecond, time.Millisecond)

	close(closure)
	result := <-returned
	require.NoError(t, result.err)
	require.Equal(t, umpirespb.CANARY_DECISION_PASS, result.response.GetResult().GetDecision())
}

func TestExecuteSequentialRunsReceiveFreshIdentities(t *testing.T) {
	var identitiesMu sync.Mutex
	identities := []string{}
	run := func(
		_ context.Context,
		_ artifact.AdmittedSet,
		_ runner.InputBinding,
		runIdentity string,
		_ runner.Adapter,
	) (runOutcome, error) {
		identitiesMu.Lock()
		identities = append(identities, runIdentity)
		identitiesMu.Unlock()
		return successfulOutcome(runIdentity), nil
	}
	executor := newExecutor(nil, run, successfulEvaluation, sequentialIdentities())

	first, err := executor.Execute(context.Background(), fixtureRequest(t, "normal"))
	require.NoError(t, err)
	second, err := executor.Execute(context.Background(), fixtureRequest(t, "normal"))
	require.NoError(t, err)

	require.Equal(t, umpirespb.CANARY_DECISION_PASS, first.GetResult().GetDecision())
	require.Equal(t, umpirespb.CANARY_DECISION_PASS, second.GetResult().GetDecision())
	require.Equal(t, []string{"umpire.executor.test.1", "umpire.executor.test.2"}, identities)
	require.NotEqual(t, first.GetResult().GetRunIdentity(), second.GetResult().GetRunIdentity())
}

func TestExecuteRejectsOverlapBeforeRunnerIO(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan executeResult, 1)
	var runCalls atomic.Int32
	request := fixtureRequest(t, "normal")
	executor := newExecutor(nil, func(
		_ context.Context,
		_ artifact.AdmittedSet,
		_ runner.InputBinding,
		runIdentity string,
		_ runner.Adapter,
	) (runOutcome, error) {
		runCalls.Add(1)
		close(entered)
		<-release
		return successfulOutcome(runIdentity), nil
	}, successfulEvaluation, sequentialIdentities())

	go func() {
		response, err := executor.Execute(context.Background(), request)
		firstDone <- executeResult{response: response, err: err}
	}()
	<-entered

	overlap, err := executor.Execute(context.Background(), fixtureRequest(t, "normal"))
	require.NoError(t, err)
	require.Equal(t, umpirespb.TOOLING_STATUS_BUSY, overlap.GetResult().GetToolingStatus())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, overlap.GetResult().GetDecision())
	require.Equal(t, int32(1), runCalls.Load())

	close(release)
	first := <-firstDone
	require.NoError(t, first.err)
	require.Equal(t, umpirespb.CANARY_DECISION_PASS, first.response.GetResult().GetDecision())
}

func TestExecuteCancellationDoesNotExposeIdleBeforeCleanup(t *testing.T) {
	runStarted := make(chan struct{})
	cleanupStarted := make(chan struct{})
	cleanupRelease := make(chan struct{})
	firstDone := make(chan executeResult, 1)
	var runCalls atomic.Int32
	request := fixtureRequest(t, "normal")
	executor := newExecutor(nil, func(
		ctx context.Context,
		_ artifact.AdmittedSet,
		_ runner.InputBinding,
		runIdentity string,
		_ runner.Adapter,
	) (runOutcome, error) {
		if runCalls.Add(1) == 1 {
			close(runStarted)
			<-ctx.Done()
			close(cleanupStarted)
			<-cleanupRelease
			return runOutcome{reusable: true}, ctx.Err()
		}
		return successfulOutcome(runIdentity), nil
	}, successfulEvaluation, sequentialIdentities())
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		response, err := executor.Execute(ctx, request)
		firstDone <- executeResult{response: response, err: err}
	}()
	<-runStarted
	cancel()
	<-cleanupStarted

	overlap, err := executor.Execute(context.Background(), fixtureRequest(t, "normal"))
	require.NoError(t, err)
	require.Equal(t, umpirespb.TOOLING_STATUS_BUSY, overlap.GetResult().GetToolingStatus())
	require.Equal(t, int32(1), runCalls.Load())

	close(cleanupRelease)
	canceled := <-firstDone
	require.NoError(t, canceled.err)
	require.Equal(t, umpirespb.TOOLING_STATUS_CANCELED, canceled.response.GetResult().GetToolingStatus())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, canceled.response.GetResult().GetDecision())

	next, err := executor.Execute(context.Background(), fixtureRequest(t, "normal"))
	require.NoError(t, err)
	require.Equal(t, umpirespb.CANARY_DECISION_PASS, next.GetResult().GetDecision())
	require.Equal(t, int32(2), runCalls.Load())
}

func TestExecuteDeadlineIsInconclusiveAndReusableAfterCertainCleanup(t *testing.T) {
	var runCalls atomic.Int32
	executor := newExecutor(nil, func(
		ctx context.Context,
		_ artifact.AdmittedSet,
		_ runner.InputBinding,
		runIdentity string,
		_ runner.Adapter,
	) (runOutcome, error) {
		if runCalls.Add(1) == 1 {
			<-ctx.Done()
			return runOutcome{reusable: true}, ctx.Err()
		}
		return successfulOutcome(runIdentity), nil
	}, successfulEvaluation, sequentialIdentities())
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	deadline, err := executor.Execute(ctx, fixtureRequest(t, "normal"))
	require.NoError(t, err)
	require.Equal(t, umpirespb.TOOLING_STATUS_CANCELED, deadline.GetResult().GetToolingStatus())
	require.Equal(t, umpirespb.OPERATIONAL_STATUS_INCOMPLETE, deadline.GetResult().GetOperationalStatus())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, deadline.GetResult().GetDecision())

	next, err := executor.Execute(context.Background(), fixtureRequest(t, "normal"))
	require.NoError(t, err)
	require.Equal(t, umpirespb.CANARY_DECISION_PASS, next.GetResult().GetDecision())
}

func TestExecuteAppliesTheContractDeadlineAcrossExecutionAndEvaluation(t *testing.T) {
	executor := newExecutor(nil, func(
		ctx context.Context,
		_ artifact.AdmittedSet,
		_ runner.InputBinding,
		_ string,
		_ runner.Adapter,
	) (runOutcome, error) {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 100*time.Millisecond {
			return runOutcome{reusable: true}, errors.New("contract deadline missing")
		}
		<-ctx.Done()
		return runOutcome{reusable: true}, ctx.Err()
	}, successfulEvaluation, sequentialIdentities())
	request := fixtureRequestWithContract(t, "normal", func(contract *umpirespb.EvaluationContract) {
		contract.Limits.MaxTotalDurationMilliseconds = 1
	})

	response, err := executor.Execute(context.Background(), request)

	require.NoError(t, err)
	require.Equal(t, umpirespb.TOOLING_STATUS_CANCELED, response.GetResult().GetToolingStatus())
	require.Equal(t, umpirespb.OPERATIONAL_STATUS_INCOMPLETE, response.GetResult().GetOperationalStatus())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, response.GetResult().GetDecision())
}

func TestExecutePoisonsAfterUncertainCleanup(t *testing.T) {
	var runCalls atomic.Int32
	executor := newExecutor(nil, func(
		_ context.Context,
		_ artifact.AdmittedSet,
		_ runner.InputBinding,
		runIdentity string,
		_ runner.Adapter,
	) (runOutcome, error) {
		runCalls.Add(1)
		outcome := successfulOutcome(runIdentity)
		outcome.cleanupStatus = umpirespb.CLEANUP_STATUS_INCOMPLETE
		outcome.reusable = false
		return outcome, nil
	}, successfulEvaluation, sequentialIdentities())

	uncertain, err := executor.Execute(context.Background(), fixtureRequest(t, "normal"))
	require.NoError(t, err)
	require.Equal(t, umpirespb.CLEANUP_STATUS_INCOMPLETE, uncertain.GetResult().GetCleanupStatus())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, uncertain.GetResult().GetDecision())

	poisoned, err := executor.Execute(context.Background(), fixtureRequest(t, "normal"))
	require.NoError(t, err)
	require.Equal(t, umpirespb.TOOLING_STATUS_POISONED, poisoned.GetResult().GetToolingStatus())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, poisoned.GetResult().GetDecision())
	require.Equal(t, int32(1), runCalls.Load())
}

func TestExecuteAdmissionFailuresAreTypedPreIOAndDoNotPoison(t *testing.T) {
	var runCalls atomic.Int32
	executor := newExecutor(nil, func(
		_ context.Context,
		_ artifact.AdmittedSet,
		_ runner.InputBinding,
		runIdentity string,
		_ runner.Adapter,
	) (runOutcome, error) {
		runCalls.Add(1)
		return successfulOutcome(runIdentity), nil
	}, successfulEvaluation, sequentialIdentities())

	tests := []struct {
		name string
		edit func(*umpirespb.ExecuteRequest)
		want umpirespb.ToolingStatus
	}{
		{
			name: "invalid contract",
			edit: func(request *umpirespb.ExecuteRequest) {
				request.EvaluationContract = []byte("not a protobuf contract")
			},
			want: umpirespb.TOOLING_STATUS_INVALID_CONTRACT,
		},
		{
			name: "crossed input",
			edit: func(request *umpirespb.ExecuteRequest) {
				request.Input = fixtureRequest(t, "duplicate-delivery").GetInput()
			},
			want: umpirespb.TOOLING_STATUS_INVALID_INPUT,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := fixtureRequest(t, "normal")
			test.edit(request)

			response, err := executor.Execute(context.Background(), request)

			require.NoError(t, err)
			require.Equal(t, test.want, response.GetResult().GetToolingStatus())
			require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, response.GetResult().GetDecision())
			require.Equal(t, int32(0), runCalls.Load())
		})
	}

	accepted, err := executor.Execute(context.Background(), fixtureRequest(t, "normal"))
	require.NoError(t, err)
	require.Equal(t, umpirespb.CANARY_DECISION_PASS, accepted.GetResult().GetDecision())
	require.Equal(t, int32(1), runCalls.Load())
}

func TestExecutePoisonsWhenAStartedRunLosesCleanupCertainty(t *testing.T) {
	var runCalls atomic.Int32
	executor := newExecutor(nil, func(
		_ context.Context,
		_ artifact.AdmittedSet,
		_ runner.InputBinding,
		_ string,
		_ runner.Adapter,
	) (runOutcome, error) {
		runCalls.Add(1)
		return runOutcome{reusable: false}, errors.New("run closure unavailable")
	}, successfulEvaluation, sequentialIdentities())

	failed, err := executor.Execute(context.Background(), fixtureRequest(t, "normal"))
	require.NoError(t, err)
	require.Equal(t, umpirespb.TOOLING_STATUS_INTERNAL_ERROR, failed.GetResult().GetToolingStatus())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, failed.GetResult().GetDecision())

	poisoned, err := executor.Execute(context.Background(), fixtureRequest(t, "normal"))
	require.NoError(t, err)
	require.Equal(t, umpirespb.TOOLING_STATUS_POISONED, poisoned.GetResult().GetToolingStatus())
	require.Equal(t, int32(1), runCalls.Load())
}

func TestExecuteKeepsPreStartRunnerFailuresInternalAndReusable(t *testing.T) {
	var runCalls atomic.Int32
	executor := newExecutor(nil, func(
		_ context.Context,
		_ artifact.AdmittedSet,
		_ runner.InputBinding,
		runIdentity string,
		_ runner.Adapter,
	) (runOutcome, error) {
		if runCalls.Add(1) == 1 {
			return runOutcome{reusable: true}, errors.New("authority preflight failed")
		}
		return successfulOutcome(runIdentity), nil
	}, successfulEvaluation, sequentialIdentities())

	failed, err := executor.Execute(context.Background(), fixtureRequest(t, "normal"))
	require.NoError(t, err)
	require.Equal(t, umpirespb.TOOLING_STATUS_INTERNAL_ERROR, failed.GetResult().GetToolingStatus())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, failed.GetResult().GetDecision())

	next, err := executor.Execute(context.Background(), fixtureRequest(t, "normal"))
	require.NoError(t, err)
	require.Equal(t, umpirespb.CANARY_DECISION_PASS, next.GetResult().GetDecision())
	require.Equal(t, int32(2), runCalls.Load())
}

func fixtureRequest(t *testing.T, name string) *umpirespb.ExecuteRequest {
	t.Helper()
	contract, err := os.ReadFile(filepath.Join("..", "portableevaluation", "testdata", name, "contract.pb"))
	require.NoError(t, err)
	inputRoot := filepath.Join("..", "temporal", "nexus", "testdata", "caller-closure-input-set")
	if name == "duplicate-delivery" {
		inputRoot = filepath.Join(
			"..", "temporal", "nexus", "testdata", "caller-closure-duplicate-delivery-input-set",
		)
	}
	experiment, err := os.ReadFile(filepath.Join(inputRoot, "artifacts", "experiment.json"))
	require.NoError(t, err)
	configuration, err := os.ReadFile(filepath.Join(inputRoot, "artifacts", "runtime-configuration.json"))
	require.NoError(t, err)
	return &umpirespb.ExecuteRequest{
		EvaluationContract: contract,
		Input: &umpirespb.EvaluationInput{
			Experiment: experiment, RuntimeConfig: configuration,
		},
	}
}

func fixtureRequestWithContract(
	t *testing.T,
	name string,
	modify func(*umpirespb.EvaluationContract),
) *umpirespb.ExecuteRequest {
	t.Helper()
	request := fixtureRequest(t, name)
	contract, err := evaluationcontract.Admit(request.GetEvaluationContract())
	require.NoError(t, err)
	contract = proto.CloneOf(contract)
	contract.ArtifactChecksum = nil
	modify(contract)
	canonical, err := evaluationcontract.CanonicalProtoJSON(contract)
	require.NoError(t, err)
	request.EvaluationContract, err = evaluationcontract.Pack(canonical)
	require.NoError(t, err)
	return request
}

func fixtureRunner(t *testing.T, name string) executeRun {
	t.Helper()
	rawEvidence := fixtureRawEvidence(t, name)
	return func(
		_ context.Context,
		_ artifact.AdmittedSet,
		_ runner.InputBinding,
		runIdentity string,
		_ runner.Adapter,
	) (runOutcome, error) {
		if rawEvidence.RunIdentity != runIdentity {
			return runOutcome{}, errors.New("fixture run identity mismatch")
		}
		return runOutcome{
			rawEvidence:       rawEvidence,
			runBinding:        rawEvidence.Run,
			sourceClosures:    closures(rawEvidence.Sources),
			operationalStatus: umpirespb.OPERATIONAL_STATUS_SUCCEEDED,
			cleanupStatus:     umpirespb.CLEANUP_STATUS_COMPLETE,
			reusable:          true,
		}, nil
	}
}

func fixtureRawEvidence(t *testing.T, name string) artifactv2.RawEvidence {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join(
		"..", "portableevaluation", "testdata", name, "raw-evidence.json",
	))
	require.NoError(t, err)
	rawEvidence, err := artifact.DecodeRawEvidenceV2(encoded)
	require.NoError(t, err)
	return rawEvidence
}

func successfulOutcome(runIdentity string) runOutcome {
	return runOutcome{
		rawEvidence:       artifactv2.RawEvidence{RunIdentity: runIdentity},
		operationalStatus: umpirespb.OPERATIONAL_STATUS_SUCCEEDED,
		cleanupStatus:     umpirespb.CLEANUP_STATUS_COMPLETE,
		reusable:          true,
	}
}

func successfulEvaluation(
	_ context.Context,
	request portableevaluation.Request,
) *umpirespb.EvaluationResult {
	decision := umpirespb.CANARY_DECISION_PASS
	if request.CleanupStatus != umpirespb.CLEANUP_STATUS_COMPLETE ||
		request.OperationalStatus != umpirespb.OPERATIONAL_STATUS_SUCCEEDED {
		decision = umpirespb.CANARY_DECISION_INCONCLUSIVE
	}
	return &umpirespb.EvaluationResult{
		RunIdentity:       request.ExpectedRunIdentity,
		ToolingStatus:     umpirespb.TOOLING_STATUS_SUCCEEDED,
		OperationalStatus: request.OperationalStatus,
		Observation: &umpirespb.ObservationEvaluationResult{
			Status: umpirespb.OBSERVATION_STATUS_ACCEPTED,
		},
		ImplementationLink: &umpirespb.ImplementationLinkResult{
			Status: umpirespb.IMPLEMENTATION_LINK_STATUS_APPLIED,
		},
		SemanticStatus: umpirespb.EVALUATION_STATUS_SATISFIED,
		CleanupStatus:  request.CleanupStatus,
		Decision:       decision,
	}
}

func sequentialIdentities() func() string {
	var next atomic.Int32
	return func() string {
		return "umpire.executor.test." + string(rune('0'+next.Add(1)))
	}
}

func closures(sources []artifactv2.RawEvidenceSource) []artifactv2.SourceClosure {
	result := make([]artifactv2.SourceClosure, len(sources))
	for index, source := range sources {
		result[index] = artifactv2.SourceClosure{
			SourceDefinitionID: source.SourceDefinitionID,
			Status:             source.Status,
			RecordCount:        source.FactCount,
			ByteCount:          source.ByteCount,
		}
	}
	return result
}
