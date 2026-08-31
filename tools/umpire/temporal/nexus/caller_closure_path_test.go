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
