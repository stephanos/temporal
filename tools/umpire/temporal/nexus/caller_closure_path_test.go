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
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
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

func TestCallerClosurePortabilityResultRetainsTheBoundedProofMatrix(t *testing.T) {
	requireCallerClosureBoundedProofMatrix(t, callerClosureStableMeaning{
		ObservationEvaluationStatus: "accepted",
		SemanticStatus:              "satisfied",
	})
}

func requireCallerClosureBoundedProofMatrix(
	t *testing.T,
	localMeaning callerClosureStableMeaning,
) {
	t.Helper()
	limitFailure := errors.New("Limit N+1 rejected before runtime IO")
	semanticNonSuccessChecksum := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	localEvaluationOutcomeChecksum := "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	for _, testCase := range []struct {
		name               string
		outcome            callerClosurePathOutcome
		wantPhases         []string
		wantCapture        string
		wantOpenHandles    string
		wantMeaning        callerClosureStableMeaning
		wantToolingPhase   string
		wantToolingFailure error
		wantPortable       bool
	}{
		{
			name: "success at Limit N",
			outcome: callerClosurePortabilityOutcome(
				t, "succeeded", "complete", "closed", "", "accepted", "satisfied",
				localEvaluationOutcomeChecksum,
			),
			wantPhases: []string{
				"succeeded", "succeeded", "succeeded", "succeeded", "succeeded",
			},
			wantCapture: "closed", wantOpenHandles: "0", wantMeaning: localMeaning,
			wantPortable: true,
		},
		{
			name: "semantic non-success",
			outcome: callerClosurePortabilityOutcome(
				t, "succeeded", "complete", "closed", "", "accepted", "violated",
				semanticNonSuccessChecksum,
			),
			wantPhases: []string{
				"succeeded", "succeeded", "succeeded", "succeeded", "succeeded",
			},
			wantCapture: "closed", wantOpenHandles: "0",
			wantMeaning: callerClosureStableMeaning{
				ObservationEvaluationStatus: "accepted",
				SemanticStatus:              "violated",
			},
		},
		{
			name: "cancellation",
			outcome: callerClosurePortabilityOutcome(
				t, "incomplete", "complete", "closed", "canceled", "unknown", "incomplete", "",
			),
			wantPhases: []string{
				"succeeded", "canceled", "succeeded", "succeeded", "succeeded",
			},
			wantCapture: "closed", wantOpenHandles: "0",
			wantMeaning: callerClosureStableMeaning{
				ObservationEvaluationStatus: "unknown", SemanticStatus: "incomplete",
			},
		},
		{
			name: "timeout",
			outcome: callerClosurePortabilityOutcome(
				t, "incomplete", "complete", "partial", "timed-out", "unknown", "incomplete", "",
			),
			wantPhases: []string{
				"succeeded", "succeeded", "timed-out", "succeeded", "succeeded",
			},
			wantCapture: "partial", wantOpenHandles: "0",
			wantMeaning: callerClosureStableMeaning{
				ObservationEvaluationStatus: "unknown", SemanticStatus: "incomplete",
			},
		},
		{
			name: "Limit N+1",
			outcome: callerClosurePathOutcome{
				toolingPhase: "runner", toolingFailure: limitFailure,
			},
			wantToolingPhase: "runner", wantToolingFailure: limitFailure,
		},
		{
			name: "cleanup failure",
			outcome: callerClosurePortabilityOutcome(
				t, "failed", "failed", "failed", "failed", "unknown", "incomplete", "",
			),
			wantPhases: []string{
				"succeeded", "succeeded", "succeeded", "succeeded", "failed",
			},
			wantCapture: "failed", wantOpenHandles: "1",
			wantMeaning: callerClosureStableMeaning{
				ObservationEvaluationStatus: "unknown", SemanticStatus: "incomplete",
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			result := retainCallerClosurePortabilityResult(testCase.outcome)

			require.Equal(t, testCase.wantPhases, result.PhaseStatuses)
			require.Equal(t, testCase.wantCapture, result.CaptureStatus)
			require.Equal(t, testCase.wantOpenHandles, result.OpenHandleCount)
			require.Equal(t, testCase.wantMeaning, result.StableMeaning)
			require.Equal(t, testCase.wantToolingPhase, result.ToolingPhase)
			require.ErrorIs(t, result.ToolingFailure, testCase.wantToolingFailure)
			require.Equal(t, testCase.wantPortable, result.ProvesPortableSuccess(localMeaning))
		})
	}
}

func requireCallerClosureEqualResultMeaning(
	t *testing.T,
	local artifactv2.Result,
	ci artifactv2.Result,
) {
	t.Helper()
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
	path := newCallerClosurePath(t, "umpire.ci.caller-closure.limit-n-plus-one-1")
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

func callerClosurePortabilityOutcome(
	t *testing.T,
	operationalStatus string,
	cleanupStatus string,
	captureStatus string,
	terminalPhaseStatus string,
	observationStatus string,
	semanticStatus string,
	evaluationOutcomeChecksum string,
) callerClosurePathOutcome {
	t.Helper()
	output := callerClosurePathOutput(t, operationalStatus, cleanupStatus)
	run := output.ExperimentRun()
	rawEvidence := output.RawEvidence()
	rawEvidence.CaptureStatus = captureStatus
	switch terminalPhaseStatus {
	case "canceled":
		run.PhaseOutcomes[1].Status = terminalPhaseStatus
	case "timed-out":
		run.PhaseOutcomes[2].Status = terminalPhaseStatus
	case "failed":
		run.PhaseOutcomes[4].Status = terminalPhaseStatus
		run.Cleanup.OpenHandleCount = "1"
	default:
		require.Empty(t, terminalPhaseStatus)
	}
	output = umpireruntime.NewOutput(output.AdmittedSet(), run, rawEvidence)
	return callerClosurePathOutcome{
		execution: &output,
		evaluation: &callerClosureEvaluation{summary: callerClosureEvaluationSummary{
			OperationalStatus:           operationalStatus,
			ObservationEvaluationStatus: observationStatus,
			SemanticStatus:              semanticStatus,
			EvaluationOutcomeChecksum:   evaluationOutcomeChecksum,
		}},
	}
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
