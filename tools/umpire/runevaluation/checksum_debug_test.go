package runevaluation

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	"go.temporal.io/server/tools/umpire/internal/runtimeengine"
)

func TestDebugAcceptedOutcomeChecksums(t *testing.T) {
	for _, semanticStatus := range []string{"satisfied", "violated"} {
		input := acceptedCallerClosureExecutionFixture(t, "succeeded")
		execution, ok := input.Execution()
		require.True(t, ok)
		request, err := newCheckerRequest(execution)
		require.NoError(t, err)
		response := acceptedCallerClosureResponse(t, request, semanticStatus)
		response.EvaluationOutcomeChecksum = nil
		evidence, err := sealEvaluationEvidence(request, response)
		require.NoError(t, err)
		run := execution.ExperimentRun()
		result := artifactv2.Result{
			FormatVersion:               artifactv2.ResultFormat,
			RunIdentity:                 request.RunIdentity,
			BehaviorFingerprint:         checkerBehaviorFingerprint,
			Experiment:                  request.Experiment,
			RuntimeConfiguration:        request.RuntimeConfiguration,
			Run:                         request.Run,
			RawEvidence:                 request.RawEvidence,
			Evidence:                    artifactv2.EvidenceArtifactBinding(evidence),
			OperationalStatus:           runtimeengine.OperationalStatus(run),
			ObservationEvaluationStatus: response.ObservationEvaluationStatus,
			ImplementationLink:          callerClosureImplementationLink(),
			ImplementationLinkStatus:    "applied",
			PropertyVerdicts:            response.PropertyVerdicts,
			QuerySummary:                response.QuerySummary,
			SemanticStatus:              response.SemanticStatus,
			Limits:                      callerClosureStagedLimits(),
			KnownGaps:                   response.ResultKnownGaps,
			CleanupStatus:               run.Cleanup.Status,
			Provenance:                  evaluationProvenance(),
		}
		checksum, err := artifactv2.ExpectedEvaluationOutcomeChecksum(
			result, evidence, execution.Experiment(),
		)
		require.NoError(t, err)
		t.Log(semanticStatus, checksum)
	}
}
