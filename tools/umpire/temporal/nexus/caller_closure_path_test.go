package nexus_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	"go.temporal.io/server/tools/umpire/runner"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
	"go.temporal.io/server/tools/umpire/temporal/nexus"
)

func TestCallerClosurePathTraversesEveryStageExactlyOnce(t *testing.T) {
	calls := []string{}
	output := callerClosurePathOutput(t, "succeeded", "complete")
	path := callerClosurePath{
		checkSubject: func() error {
			calls = append(calls, "subject")
			return nil
		},
		admit: func() (artifact.AdmittedSet, error) {
			calls = append(calls, "admission")
			return artifact.AdmittedSet{}, nil
		},
		run: func(context.Context, artifact.AdmittedSet) (umpireruntime.Output, error) {
			calls = append(calls, "runner")
			return output, nil
		},
		evaluate: func(context.Context, artifact.AdmittedSet) (callerClosureEvaluation, error) {
			calls = append(calls, "run-evaluation")
			return callerClosureEvaluation{summary: callerClosureEvaluationSummary{
				OperationalStatus:           "succeeded",
				ObservationEvaluationStatus: "accepted",
				SemanticStatus:              "satisfied",
			}}, nil
		},
	}

	outcome := runCallerClosurePath(context.Background(), path)

	require.Equal(t, []string{"subject", "admission", "runner", "run-evaluation"}, calls)
	require.NoError(t, outcome.toolingFailure)
	require.NotNil(t, outcome.execution)
	require.NotNil(t, outcome.evaluation)
}

func TestCallerClosurePathRejectsPreflightBeforeRunnerIO(t *testing.T) {
	preflightFailure := errors.New("preflight failed")
	for _, testCase := range []struct {
		name         string
		checkSubject func() error
		admit        func() (artifact.AdmittedSet, error)
	}{
		{
			name:         "subject",
			checkSubject: func() error { return preflightFailure },
			admit:        func() (artifact.AdmittedSet, error) { return artifact.AdmittedSet{}, nil },
		},
		{
			name:         "admission",
			checkSubject: func() error { return nil },
			admit:        func() (artifact.AdmittedSet, error) { return artifact.AdmittedSet{}, preflightFailure },
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			runnerCalls := 0
			evaluationCalls := 0
			outcome := runCallerClosurePath(context.Background(), callerClosurePath{
				checkSubject: testCase.checkSubject,
				admit:        testCase.admit,
				run: func(context.Context, artifact.AdmittedSet) (umpireruntime.Output, error) {
					runnerCalls++
					return umpireruntime.Output{}, nil
				},
				evaluate: func(context.Context, artifact.AdmittedSet) (callerClosureEvaluation, error) {
					evaluationCalls++
					return callerClosureEvaluation{}, nil
				},
			})

			require.ErrorIs(t, outcome.toolingFailure, preflightFailure)
			require.Equal(t, "admission", outcome.toolingPhase)
			require.Equal(t, 0, runnerCalls)
			require.Equal(t, 0, evaluationCalls)
			require.Nil(t, outcome.execution)
			require.Nil(t, outcome.evaluation)
		})
	}
}

func TestCallerClosurePathRetainsIndependentOutcomes(t *testing.T) {
	runnerFailure := errors.New("runner unavailable")
	evaluationFailure := errors.New("checker unavailable")
	for _, testCase := range []struct {
		name                        string
		operationalStatus           string
		cleanupStatus               string
		evaluationOperationalStatus string
		observationStatus           string
		semanticStatus              string
		runnerError                 error
		evaluationError             error
		wantToolingPhase            string
		wantExecution               bool
	}{
		{
			name: "cancellation", operationalStatus: "canceled", cleanupStatus: "complete",
			evaluationOperationalStatus: "canceled",
			observationStatus:           "incomplete", semanticStatus: "incomplete", wantExecution: true,
		},
		{
			name: "semantic non-success", operationalStatus: "succeeded", cleanupStatus: "complete",
			evaluationOperationalStatus: "succeeded",
			observationStatus:           "accepted", semanticStatus: "violated", wantExecution: true,
		},
		{
			name: "cleanup failure", operationalStatus: "failed", cleanupStatus: "failed",
			evaluationOperationalStatus: "failed",
			observationStatus:           "incomplete", semanticStatus: "incomplete", wantExecution: true,
		},
		{
			name: "runner tooling failure", operationalStatus: "succeeded", cleanupStatus: "complete",
			runnerError: runnerFailure, wantToolingPhase: "runner",
		},
		{
			name: "evaluation tooling failure", operationalStatus: "succeeded", cleanupStatus: "complete",
			evaluationError: evaluationFailure, wantToolingPhase: "Run Evaluation", wantExecution: true,
		},
		{
			name: "evaluation status drift", operationalStatus: "succeeded", cleanupStatus: "complete",
			evaluationOperationalStatus: "failed", observationStatus: "incomplete",
			semanticStatus: "incomplete", wantToolingPhase: "Run Evaluation", wantExecution: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			output := callerClosurePathOutput(t, testCase.operationalStatus, testCase.cleanupStatus)
			outcome := runCallerClosurePath(context.Background(), callerClosurePath{
				checkSubject: func() error { return nil },
				admit:        func() (artifact.AdmittedSet, error) { return artifact.AdmittedSet{}, nil },
				run: func(context.Context, artifact.AdmittedSet) (umpireruntime.Output, error) {
					if testCase.runnerError != nil {
						return umpireruntime.Output{}, testCase.runnerError
					}
					return output, nil
				},
				evaluate: func(context.Context, artifact.AdmittedSet) (callerClosureEvaluation, error) {
					if testCase.evaluationError != nil {
						return callerClosureEvaluation{}, testCase.evaluationError
					}
					return callerClosureEvaluation{summary: callerClosureEvaluationSummary{
						OperationalStatus:           testCase.evaluationOperationalStatus,
						ObservationEvaluationStatus: testCase.observationStatus,
						SemanticStatus:              testCase.semanticStatus,
					}}, nil
				},
			})

			if !testCase.wantExecution {
				require.ErrorIs(t, outcome.toolingFailure, testCase.runnerError)
				require.Equal(t, testCase.wantToolingPhase, outcome.toolingPhase)
				require.Nil(t, outcome.execution)
				require.Nil(t, outcome.evaluation)
				return
			}
			require.NotNil(t, outcome.execution)
			require.Equal(t, testCase.operationalStatus, outcome.execution.ExperimentRun().OperationalStatus)
			require.Equal(t, testCase.cleanupStatus, outcome.execution.ExperimentRun().Cleanup.Status)
			if testCase.evaluationError != nil {
				require.ErrorIs(t, outcome.toolingFailure, testCase.evaluationError)
				require.Equal(t, testCase.wantToolingPhase, outcome.toolingPhase)
				require.Nil(t, outcome.evaluation)
				return
			}
			if testCase.evaluationOperationalStatus != testCase.operationalStatus {
				require.ErrorContains(t, outcome.toolingFailure, "operational status drifted")
				require.Equal(t, testCase.wantToolingPhase, outcome.toolingPhase)
				require.Nil(t, outcome.evaluation)
				return
			}
			require.NoError(t, outcome.toolingFailure)
			require.NotNil(t, outcome.evaluation)
			require.Equal(t, testCase.evaluationOperationalStatus, outcome.evaluation.summary.OperationalStatus)
			require.Equal(t, testCase.observationStatus, outcome.evaluation.summary.ObservationEvaluationStatus)
			require.Equal(t, testCase.semanticStatus, outcome.evaluation.summary.SemanticStatus)
		})
	}
}

type callerClosureStableMeaning struct {
	ObservationEvaluationStatus string
	SemanticStatus              string
}

type callerClosurePortabilityResult struct {
	OperationalStatus           string
	EvaluationOperationalStatus string
	PhaseStatuses               []string
	CleanupStatus               string
	OpenHandleCount             string
	CaptureStatus               string
	EvaluationOutcomeChecksum   string
	StableMeaning               callerClosureStableMeaning
	ToolingPhase                string
	ToolingFailure              error
}

func retainCallerClosurePortabilityResult(
	outcome callerClosurePathOutcome,
) callerClosurePortabilityResult {
	result := callerClosurePortabilityResult{
		ToolingPhase: outcome.toolingPhase, ToolingFailure: outcome.toolingFailure,
	}
	if outcome.execution != nil {
		run := outcome.execution.ExperimentRun()
		result.OperationalStatus = run.OperationalStatus
		result.PhaseStatuses = make([]string, len(run.PhaseOutcomes))
		for index, phase := range run.PhaseOutcomes {
			result.PhaseStatuses[index] = phase.Status
		}
		result.CleanupStatus = run.Cleanup.Status
		result.OpenHandleCount = string(run.Cleanup.OpenHandleCount)
		result.CaptureStatus = outcome.execution.RawEvidence().CaptureStatus
	}
	if outcome.evaluation != nil {
		result.EvaluationOperationalStatus = outcome.evaluation.summary.OperationalStatus
		result.EvaluationOutcomeChecksum = outcome.evaluation.summary.EvaluationOutcomeChecksum
		result.StableMeaning = callerClosureStableMeaning{
			ObservationEvaluationStatus: outcome.evaluation.summary.ObservationEvaluationStatus,
			SemanticStatus:              outcome.evaluation.summary.SemanticStatus,
		}
	}
	return result
}

func (result callerClosurePortabilityResult) ProvesPortableSuccess(
	local callerClosureStableMeaning,
) bool {
	if result.ToolingFailure != nil || result.OperationalStatus != "succeeded" ||
		result.EvaluationOperationalStatus != "succeeded" ||
		result.CleanupStatus != "complete" || result.OpenHandleCount != "0" ||
		result.CaptureStatus != "closed" || result.EvaluationOutcomeChecksum == "" ||
		result.StableMeaning != local {
		return false
	}
	for _, status := range result.PhaseStatuses {
		if status != "succeeded" {
			return false
		}
	}
	return len(result.PhaseStatuses) == 5
}

func requireCallerClosureSuccessfulPortabilityResult(
	t *testing.T,
	outcome callerClosurePathOutcome,
) callerClosurePortabilityResult {
	t.Helper()
	require.NoError(t, outcome.toolingFailure, outcome.toolingPhase)
	require.NotNil(t, outcome.execution)
	require.NotNil(t, outcome.evaluation)
	require.NotEmpty(t, outcome.evaluation.admitted.Identity())
	run := outcome.execution.ExperimentRun()
	require.Equal(t, "succeeded", run.OperationalStatus)
	require.Equal(t, []string{
		"succeeded", "succeeded", "succeeded", "succeeded", "succeeded",
	}, retainCallerClosurePortabilityResult(outcome).PhaseStatuses)
	require.Equal(t, "complete", run.Cleanup.Status)
	require.EqualValues(t, "0", run.Cleanup.OpenHandleCount)
	require.Equal(t, "closed", outcome.execution.RawEvidence().CaptureStatus)
	sources := outcome.execution.RawEvidence().Sources
	sourceDefinitionIDs := make([]string, len(sources))
	for index, source := range sources {
		sourceDefinitionIDs[index] = source.SourceDefinitionID
	}
	require.Equal(t, []string{
		umpireruntime.EvidenceSourceCleanup,
		umpireruntime.EvidenceSourceControlReceipt,
		umpireruntime.EvidenceSourceHistory,
		umpireruntime.EvidenceSourceParticipantOutput,
	}, sourceDefinitionIDs)
	require.Empty(t, run.KnownGaps)
	require.Empty(t, outcome.execution.RawEvidence().KnownGaps)
	require.Equal(t, "succeeded", outcome.evaluation.summary.OperationalStatus)
	require.Equal(t, "accepted", outcome.evaluation.summary.ObservationEvaluationStatus)
	require.Equal(t, "satisfied", outcome.evaluation.summary.SemanticStatus)
	execution, ok := outcome.execution.AdmittedSet().Execution()
	require.True(t, ok)
	require.EqualValues(t, "3584", execution.RuntimeConfiguration().PhaseLimits[2].MaxRecords)
	return retainCallerClosurePortabilityResult(outcome)
}

func requireCallerClosureBoundedProofMatrix(
	t *testing.T,
	localMeaning callerClosureStableMeaning,
	outcomes []callerClosurePathOutcome,
) {
	t.Helper()
	testCases := []struct {
		name               string
		wantOperational    string
		wantPhases         []string
		wantCapture        string
		wantOpenHandles    string
		wantMeaning        callerClosureStableMeaning
		wantToolingPhase   string
		wantToolingFailure bool
		wantEvaluation     bool
		wantChecksum       bool
		wantPortable       bool
	}{
		{
			name:            "success at Limit N",
			wantOperational: "succeeded",
			wantPhases: []string{
				"succeeded", "succeeded", "succeeded", "succeeded", "succeeded",
			},
			wantCapture: "closed", wantOpenHandles: "0", wantMeaning: localMeaning,
			wantEvaluation: true, wantChecksum: true, wantPortable: true,
		},
		{
			name:            "semantic non-success",
			wantOperational: "succeeded",
			wantPhases: []string{
				"succeeded", "succeeded", "succeeded", "succeeded", "succeeded",
			},
			wantCapture: "closed", wantOpenHandles: "0",
			wantMeaning: callerClosureStableMeaning{
				ObservationEvaluationStatus: "accepted",
				SemanticStatus:              "violated",
			}, wantEvaluation: true, wantChecksum: true,
		},
		{
			name: "cancellation", wantOperational: "incomplete",
			wantPhases: []string{
				"succeeded", "canceled", "succeeded", "succeeded", "succeeded",
			},
			wantCapture: "closed", wantOpenHandles: "0",
			wantMeaning: callerClosureStableMeaning{
				ObservationEvaluationStatus: "unknown", SemanticStatus: "incomplete",
			}, wantEvaluation: true,
		},
		{
			name: "timeout", wantOperational: "incomplete",
			wantPhases: []string{
				"succeeded", "succeeded", "timed-out", "succeeded", "succeeded",
			},
			wantCapture: "partial", wantOpenHandles: "0",
			wantMeaning: callerClosureStableMeaning{
				ObservationEvaluationStatus: "unknown", SemanticStatus: "incomplete",
			}, wantEvaluation: true,
		},
		{
			name:             "Limit N+1",
			wantToolingPhase: "runner", wantToolingFailure: true,
		},
		{
			name: "cleanup failure", wantOperational: "failed",
			wantPhases: []string{
				"succeeded", "succeeded", "succeeded", "succeeded", "failed",
			},
			wantCapture: "failed", wantOpenHandles: "0",
			wantMeaning: callerClosureStableMeaning{
				ObservationEvaluationStatus: "unknown", SemanticStatus: "incomplete",
			}, wantEvaluation: true,
		},
	}
	require.Len(t, outcomes, len(testCases))
	for index, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := retainCallerClosurePortabilityResult(outcomes[index])

			require.Equal(t, testCase.wantOperational, result.OperationalStatus)
			require.Equal(t, testCase.wantPhases, result.PhaseStatuses)
			require.Equal(t, testCase.wantCapture, result.CaptureStatus)
			require.Equal(t, testCase.wantOpenHandles, result.OpenHandleCount)
			require.Equal(t, testCase.wantMeaning, result.StableMeaning)
			require.Equal(t, testCase.wantToolingPhase, result.ToolingPhase)
			if testCase.wantToolingFailure {
				require.Error(t, result.ToolingFailure)
			} else {
				require.NoError(t, result.ToolingFailure)
			}
			if testCase.wantEvaluation {
				require.NotNil(t, outcomes[index].evaluation)
			}
			if testCase.wantChecksum {
				require.NotEmpty(t, result.EvaluationOutcomeChecksum)
			} else {
				require.Empty(t, result.EvaluationOutcomeChecksum)
			}
			require.Equal(t, testCase.wantPortable, result.ProvesPortableSuccess(localMeaning))
		})
	}
}

type callerClosureControlledPhase string

const (
	callerClosureCanceledRealization callerClosureControlledPhase = "canceled-realization"
	callerClosureTimedOutObservation callerClosureControlledPhase = "timed-out-observation"
	callerClosureFailedCleanup       callerClosureControlledPhase = "failed-cleanup"
)

func callerClosureBoundedProofOutcomes(
	ctx context.Context,
	t *testing.T,
	success callerClosurePathOutcome,
) []callerClosurePathOutcome {
	t.Helper()
	return []callerClosurePathOutcome{
		success,
		callerClosureDuplicateDeliveryOutcome(ctx, t),
		callerClosureControlledOutcome(ctx, t, callerClosureCanceledRealization),
		callerClosureControlledOutcome(ctx, t, callerClosureTimedOutObservation),
		callerClosureLimitNPlusOneOutcome(ctx, t),
		callerClosureControlledOutcome(ctx, t, callerClosureFailedCleanup),
	}
}

func callerClosureControlledOutcome(
	ctx context.Context,
	t *testing.T,
	phase callerClosureControlledPhase,
) callerClosurePathOutcome {
	t.Helper()
	runIdentity := "umpire.ci.caller-closure." + string(phase) + "-1"
	path := newCallerClosurePath(t, runIdentity, callerClosureControlledAdapter{
		t: t, phase: phase,
	})
	return runCallerClosurePath(ctx, path)
}

func callerClosureDuplicateDeliveryOutcome(
	ctx context.Context,
	t *testing.T,
) callerClosurePathOutcome {
	t.Helper()
	input := admitCallerClosureInputAt(t, "caller-closure-duplicate-delivery-input-set")
	const runIdentity = "umpire.ci.caller-closure.semantic-non-success-1"
	path := callerClosurePath{
		checkSubject: func() error { return nil },
		admit: func() (artifact.AdmittedSet, error) {
			return input, nil
		},
		run: func(ctx context.Context, admitted artifact.AdmittedSet) (umpireruntime.Output, error) {
			return runner.Run(
				ctx, admitted, callerClosureDuplicateDeliveryBinding(), runIdentity, nexus.Binding{},
			)
		},
		evaluate: func(ctx context.Context, admitted artifact.AdmittedSet) (callerClosureEvaluation, error) {
			return runCallerClosureEvaluation(t, ctx, admitted)
		},
	}
	return runCallerClosurePath(ctx, path)
}

func admitCallerClosureInputAt(t *testing.T, name string) artifact.AdmittedSet {
	t.Helper()
	files := make(map[string][]byte, 3)
	for _, relative := range []string{
		"manifest.json",
		"artifacts/experiment.json",
		"artifacts/runtime-configuration.json",
	} {
		encoded, err := os.ReadFile(filepath.Join("testdata", name, filepath.FromSlash(relative)))
		require.NoError(t, err)
		files[relative] = encoded
	}
	admitted, err := artifact.AdmitSetFiles(files)
	require.NoError(t, err)
	return admitted
}

func callerClosureDuplicateDeliveryBinding() runner.InputBinding {
	return runner.InputBinding{
		ArtifactSetIdentity:                     "umpire.artifact-set.2a6c3ef5fbd3b7dfba1acbe2c9ffc5ec3072b19daf50d3d63bd16b122fc2bd68",
		ArtifactSetChecksum:                     "sha256:3ddabf041e499ee0b7e970cac3900b8d6306ec9009e92924ef7b9ea0f584a5f8",
		ManifestSHA256:                          "sha256:96cf1869d444e1db25f9999ea3d3928f5c07308b8c7f387b570027f5f69b5f4b",
		ExperimentArtifactChecksum:              "sha256:09091758defd5ce50cc9acbba23a5c8499da4eef9b6e36878ac989ddea87fedf",
		ExperimentBehaviorFingerprint:           "sha256:eb6c9391f0bbd82effc5793d4b0650c3b01f2471b5f05838cdec7377a5931a91",
		RuntimeConfigurationArtifactChecksum:    "sha256:440c0632b911571e4efb34c96fb4c4c7096fbd52f23900ed4784e037370063cf",
		RuntimeConfigurationBehaviorFingerprint: "sha256:d88670a6766c2ef9037c82183f00c1c42179a7578c3c4c07714eadb5540750c0",
		AuthorityRequiredCapabilityDefinitionIDs: []string{
			"umpire.runtime.capability.complete-workflow-history-read",
			"umpire.runtime.capability.ephemeral-server-lifecycle",
			"umpire.runtime.capability.sdk-worker-lifecycle",
		},
	}
}

type callerClosureControlledAdapter struct {
	t     *testing.T
	phase callerClosureControlledPhase
}

func (adapter callerClosureControlledAdapter) CheckRequest(
	admitted artifact.AdmittedSet,
	runIdentity string,
) (umpireruntime.CheckedRunRequest, error) {
	return (nexus.Binding{}).CheckRequest(admitted, runIdentity)
}

func (callerClosureControlledAdapter) EnvironmentFactory() umpireruntime.EnvironmentFactory {
	return (nexus.Binding{}).EnvironmentFactory()
}

func (adapter callerClosureControlledAdapter) NewParticipant(
	request umpireruntime.CheckedRunRequest,
) (umpireruntime.Participant, error) {
	participant, err := (nexus.Binding{}).NewParticipant(request)
	if err != nil {
		return nil, err
	}
	return callerClosureControlledParticipant{
		t: adapter.t, phase: adapter.phase, delegate: participant,
	}, nil
}

func (callerClosureControlledAdapter) ValidateOutput(
	request umpireruntime.CheckedRunRequest,
	output umpireruntime.Output,
) error {
	return (nexus.Binding{}).ValidateOutput(request, output)
}

type callerClosureControlledParticipant struct {
	t        *testing.T
	phase    callerClosureControlledPhase
	delegate umpireruntime.Participant
}

func (participant callerClosureControlledParticipant) Prepare(
	ctx context.Context,
	environment umpireruntime.Environment,
	command umpireruntime.Command,
) umpireruntime.Receipt {
	return participant.delegate.Prepare(ctx, environment, command)
}

func (participant callerClosureControlledParticipant) Realize(
	ctx context.Context,
	environment umpireruntime.Environment,
	command umpireruntime.Command,
) umpireruntime.Receipt {
	receipt := participant.delegate.Realize(ctx, environment, command)
	if participant.phase == callerClosureCanceledRealization {
		return callerClosureReceiptWithStatus(participant.t, receipt, umpireruntime.ReceiptCanceled)
	}
	return receipt
}

func (participant callerClosureControlledParticipant) Observe(
	ctx context.Context,
	environment umpireruntime.Environment,
	command umpireruntime.Command,
) umpireruntime.Receipt {
	receipt := participant.delegate.Observe(ctx, environment, command)
	if participant.phase == callerClosureTimedOutObservation {
		<-ctx.Done()
	}
	return receipt
}

func (participant callerClosureControlledParticipant) Cleanup(
	ctx context.Context,
	environment umpireruntime.Environment,
	command umpireruntime.Command,
) umpireruntime.Receipt {
	receipt := participant.delegate.Cleanup(ctx, environment, command)
	if participant.phase == callerClosureFailedCleanup {
		return callerClosureReceiptWithStatus(participant.t, receipt, umpireruntime.ReceiptFailed)
	}
	return receipt
}

func callerClosureReceiptWithStatus(
	t *testing.T,
	receipt umpireruntime.Receipt,
	status umpireruntime.ReceiptStatus,
) umpireruntime.Receipt {
	t.Helper()
	var (
		controlled umpireruntime.Receipt
		err        error
	)
	if receipt.ControlAttempted() {
		controlled, err = umpireruntime.NewControlReceipt(
			receipt.Command(), status, receipt.Facts(),
			receipt.AcquiredResources(), receipt.ReleasedResources(),
		)
	} else {
		controlled, err = umpireruntime.NewReceipt(
			receipt.Command(), status, receipt.Facts(),
			receipt.AcquiredResources(), receipt.ReleasedResources(),
		)
	}
	require.NoError(t, err)
	return controlled
}

func requireCallerClosureEqualResultMeaning(
	t *testing.T,
	local artifactv2.Result,
	ci artifactv2.Result,
) {
	t.Helper()
	require.Equal(t, local.FormatVersion, ci.FormatVersion)
	require.Equal(t, local.BehaviorFingerprint, ci.BehaviorFingerprint)
	require.Equal(t, local.OperationalStatus, ci.OperationalStatus)
	require.Equal(t, local.ObservationEvaluationStatus, ci.ObservationEvaluationStatus)
	require.Equal(t, local.ImplementationLink, ci.ImplementationLink)
	require.Equal(t, local.ImplementationLinkStatus, ci.ImplementationLinkStatus)
	requireCallerClosureEqualPropertyVerdicts(t, local.PropertyVerdicts, ci.PropertyVerdicts)
	require.Equal(t, local.QuerySummary.QueryDefinitionID, ci.QuerySummary.QueryDefinitionID)
	require.Equal(t, local.QuerySummary.Status, ci.QuerySummary.Status)
	require.Equal(t, local.QuerySummary.QueryLimits, ci.QuerySummary.QueryLimits)
	require.Equal(t, local.QuerySummary.RequiredPropertyDefinitionIDs,
		ci.QuerySummary.RequiredPropertyDefinitionIDs)
	requireCallerClosureEqualPropertyVerdicts(t,
		local.QuerySummary.PropertyVerdicts, ci.QuerySummary.PropertyVerdicts)
	require.Equal(t, local.QuerySummary.MissingPropertyDefinitionIDs,
		ci.QuerySummary.MissingPropertyDefinitionIDs)
	require.Equal(t, local.QuerySummary.DuplicatePropertyDefinitionIDs,
		ci.QuerySummary.DuplicatePropertyDefinitionIDs)
	require.Equal(t, local.QuerySummary.UnexpectedPropertyDefinitionIDs,
		ci.QuerySummary.UnexpectedPropertyDefinitionIDs)
	require.Equal(t, local.QuerySummary.DivergentPropertyDefinitionIDs,
		ci.QuerySummary.DivergentPropertyDefinitionIDs)
	require.Equal(t, local.QuerySummary.WrongQueryResultDefinitionIDs,
		ci.QuerySummary.WrongQueryResultDefinitionIDs)
	require.Equal(t, local.SemanticStatus, ci.SemanticStatus)
	require.Equal(t, local.Limits, ci.Limits)
	require.Equal(t, local.KnownGaps, ci.KnownGaps)
	require.Equal(t, local.CleanupStatus, ci.CleanupStatus)
}

func requireCallerClosureDistinctResultTransport(
	t *testing.T,
	local artifactv2.Result,
	ci artifactv2.Result,
) {
	t.Helper()
	require.NotEqual(t, local.RunIdentity, ci.RunIdentity)
	require.NotEqual(t, local.QuerySummary.TraceIDs, ci.QuerySummary.TraceIDs)
	require.Len(t, ci.PropertyVerdicts, len(local.PropertyVerdicts))
	for index, localVerdict := range local.PropertyVerdicts {
		ciVerdict := ci.PropertyVerdicts[index]
		require.NotEqual(t, localVerdict.TraceID, ciVerdict.TraceID)
		require.Len(t, ciVerdict.Clauses, len(localVerdict.Clauses))
		for clauseIndex, localClause := range localVerdict.Clauses {
			require.NotEqual(t, localClause.EvidenceLinks, ciVerdict.Clauses[clauseIndex].EvidenceLinks)
		}
	}
}

func requireCallerClosureEqualPropertyVerdicts(
	t *testing.T,
	local []artifactv2.PropertyVerdict,
	ci []artifactv2.PropertyVerdict,
) {
	t.Helper()
	require.Len(t, ci, len(local))
	for index, localVerdict := range local {
		ciVerdict := ci[index]
		require.Equal(t, localVerdict.QueryDefinitionID, ciVerdict.QueryDefinitionID)
		require.Equal(t, localVerdict.PropertyDefinitionID, ciVerdict.PropertyDefinitionID)
		require.Equal(t, localVerdict.PropertyBehaviorFingerprint,
			ciVerdict.PropertyBehaviorFingerprint)
		require.Equal(t, localVerdict.Status, ciVerdict.Status)
		require.Equal(t, localVerdict.QueryLimits, ciVerdict.QueryLimits)
		require.Equal(t, localVerdict.EvidenceLimit, ciVerdict.EvidenceLimit)
		require.Equal(t, localVerdict.ProvenanceDefinitionIDs,
			ciVerdict.ProvenanceDefinitionIDs)
		require.Equal(t, localVerdict.Diagnostic, ciVerdict.Diagnostic)
		require.Len(t, ciVerdict.Clauses, len(localVerdict.Clauses))
		for clauseIndex, localClause := range localVerdict.Clauses {
			ciClause := ciVerdict.Clauses[clauseIndex]
			require.Equal(t, localClause.PropertyDefinitionID, ciClause.PropertyDefinitionID)
			require.Equal(t, localClause.ClauseDefinitionID, ciClause.ClauseDefinitionID)
			require.Equal(t, localClause.Status, ciClause.Status)
			require.Equal(t, localClause.Coordinates, ciClause.Coordinates)
			require.Equal(t, localClause.QueryLimits, ciClause.QueryLimits)
			require.Equal(t, localClause.PropertyLimit, ciClause.PropertyLimit)
			require.Equal(t, localClause.EvidenceLimit, ciClause.EvidenceLimit)
			require.Equal(t, localClause.ProvenanceDefinitionIDs,
				ciClause.ProvenanceDefinitionIDs)
		}
	}
}

func callerClosureLimitNPlusOneOutcome(
	ctx context.Context,
	t *testing.T,
) callerClosurePathOutcome {
	t.Helper()
	path := newCallerClosurePath(
		t, "umpire.ci.caller-closure.limit-n-plus-one-1", nexus.Binding{},
	)
	input, err := path.admit()
	require.NoError(t, err)
	executable, ok := input.Executable()
	require.True(t, ok)
	configuration := executable.RuntimeConfiguration()
	require.Len(t, configuration.PhaseLimits, 5)
	configuration.PhaseLimits[2].MaxRecords = artifactv2.NaturalFromUint64(3585)
	configuration, err = artifactv2.SealRuntimeConfiguration(configuration)
	require.NoError(t, err)
	experiment, err := artifact.EncodeExperimentV2(executable.Experiment())
	require.NoError(t, err)
	encodedConfiguration, err := artifact.EncodeRuntimeConfigurationV2(configuration)
	require.NoError(t, err)
	input, err = artifact.AdmitSet([]artifact.SetMember{
		{Path: "artifacts/experiment.json", Encoded: experiment},
		{Path: "artifacts/runtime-configuration.json", Encoded: encodedConfiguration},
	})
	require.NoError(t, err)
	path.admit = func() (artifact.AdmittedSet, error) {
		return input, nil
	}
	return runCallerClosurePath(ctx, path)
}

func TestCallerClosureEvaluationPreservesSemanticNonSuccess(t *testing.T) {
	output := callerClosurePathOutput(t, "succeeded", "complete")
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	evaluation, err := runCallerClosureEvaluation(t, ctx, output.AdmittedSet())

	require.NoError(t, err)
	require.Equal(t, "succeeded", evaluation.summary.OperationalStatus)
	require.Equal(t, "accepted", evaluation.summary.ObservationEvaluationStatus)
	require.Equal(t, "violated", evaluation.summary.SemanticStatus)
	require.NotEmpty(t, evaluation.admitted.Identity())
}

func callerClosurePathOutput(
	t *testing.T,
	operationalStatus string,
	cleanupStatus string,
) umpireruntime.Output {
	t.Helper()
	root := "testdata/caller-closure-duplicate-delivery-run-set"
	files := make(map[string][]byte, 5)
	for _, relative := range []string{
		"manifest.json",
		"artifacts/experiment.json",
		"artifacts/runtime-configuration.json",
		"artifacts/experiment-run.json",
		"artifacts/raw-evidence.json",
	} {
		encoded, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		require.NoError(t, err)
		files[relative] = encoded
	}
	admitted, err := artifact.AdmitSetFiles(files)
	require.NoError(t, err)
	execution, ok := admitted.Execution()
	require.True(t, ok)
	run := execution.ExperimentRun()
	run.OperationalStatus = operationalStatus
	run.Cleanup.Status = cleanupStatus
	return umpireruntime.NewOutput(admitted, run, execution.RawEvidence())
}
