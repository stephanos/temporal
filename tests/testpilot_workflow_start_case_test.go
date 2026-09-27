//go:build test_dep && integration

package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	testpilotfixture "go.temporal.io/server/tests/testcore/testpilot"
)

// TestTestpilotWorkflowStartCase drives the workflow-start Model's Case on the shared Driver. The
// modeled requirement is a field relation: the workflow type the request submitted is the workflow
// type the WorkflowExecutionStarted event records, so the Run's evidence is read back here from the
// declared Observation rather than restated by the test.
func TestTestpilotWorkflowStartCase(t *testing.T) {
	env := newTestpilotTestEnvironment(t)

	binding := CaseBinding{
		Identity: "workflow-start-profile", Namespace: "umpire-workflow-start", TaskQueue: "umpire-workflow-start-queue",
	}
	// The single-shot happy path: runCapturedCaseWithBinding loads the fixture, derives the
	// Profile, provisions, prepares and runs once.
	run, verdict := runCapturedCaseWithBinding(t, env, testpilotfixture.WorkflowStartFixture, binding)
	require.Equal(t, testpilotpb.RUN_DISPOSITION_COMPLETED, run.GetDisposition())
	require.Equal(t, testpilotpb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
	requireSubmittedWorkflowTypeEvidence(t, run, verdict, binding.TaskQueue)

	caseSource := loadTestpilotCase(t, testpilotfixture.WorkflowStartFixture)
	repeat := CaseBinding{
		Identity: "workflow-start-profile", Namespace: "umpire-workflow-start-repeat", TaskQueue: "umpire-workflow-start-queue-repeat",
	}
	live := bindCase(t, env, caseSource, repeat)
	requireRepeatedRuns(t, env, testpilotfixture.WorkflowStartFixture, caseSource, []testpilotLiveCase{live}, 2, false,
		func(_ int, run *testpilotpb.Run, verdict *testpilotpb.Verdict) {
			requireRunSatisfied(t, "", run, verdict)
			requireSubmittedWorkflowTypeEvidence(t, run, verdict, repeat.TaskQueue)
		})
}

// requireSubmittedWorkflowTypeEvidence reads the relation's supporting Observation back out of the
// Run and checks it is the started event whose recorded workflow type the rule compared. The
// Verdict carries the relation's monitor rule beside the correlated rules the Query's Property
// lowered into; the monitor rule's one supporting event is the one read here.
func requireSubmittedWorkflowTypeEvidence(t testing.TB, run *testpilotpb.Run, verdict *testpilotpb.Verdict, taskQueue string) {
	t.Helper()
	var sequences []int64
	for _, rule := range verdict.GetRules() {
		require.Equal(t, testpilotpb.RULE_VERDICT_STATUS_SATISFIED, rule.GetStatus(), rule.GetRuleId())
		if rule.GetRuleId() == testpilotfixture.WorkflowStartRuleID {
			sequences = rule.GetSupportingEventSequences()
		}
	}
	require.Len(t, sequences, 1)
	event := runEventAt(t, run, sequences[0])
	require.Equal(t, "controller", event.GetCoordinates().GetEntrypointId())
	require.Equal(t, "history", event.GetCoordinates().GetInstructionId())

	var historyEvent historypb.HistoryEvent
	require.NoError(t, observationValue(t, event, "history-event").GetMessageValue().UnmarshalTo(&historyEvent))
	require.Equal(t, enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED, historyEvent.GetEventType())
	started := historyEvent.GetWorkflowExecutionStartedEventAttributes()
	require.Equal(t, testpilotfixture.WorkflowStartWorkflowType, started.GetWorkflowType().GetName())
	require.Equal(t, taskQueue, started.GetTaskQueue().GetName())
}
