//go:build test_dep && integration

package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	testpilotfixture "go.temporal.io/server/tests/testcore/testpilot"
	"google.golang.org/protobuf/proto"
)

// TestTestpilotTypedUnaryCase drives the generated StartWorkflowExecution example on the shared
// Driver. The modeled requirement is that the workflow type the request submitted is the workflow
// type the WorkflowExecutionStarted event records, so the Run's evidence is read back here from the
// declared Observation rather than restated by the test.
func TestTestpilotTypedUnaryCase(t *testing.T) {
	env := newTestpilotTestEnvironment(t)
	caseSnapshot := proto.CloneOf(loadTestpilotCase(t, "typed-unary"))

	binding := CaseBinding{
		Identity: "typed-unary-profile", Namespace: "umpire-typed-unary", TaskQueue: "umpire-typed-unary-queue",
	}
	// The single-shot happy path: runCase loads the fixture, derives the Profile, provisions,
	// prepares and runs once.
	run, verdict := runCaseWithBinding(t, env, "typed-unary", binding)
	require.Equal(t, testpilotpb.RUN_STATUS_COMPLETED, run.GetStatus())
	require.Equal(t, testpilotpb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
	requireSubmittedWorkflowTypeEvidence(t, run, verdict.GetSupportingEventSequences(), binding.TaskQueue)

	caseSource := loadTestpilotCase(t, "typed-unary")
	repeat := CaseBinding{
		Identity: "typed-unary-profile", Namespace: "umpire-typed-unary-repeat", TaskQueue: "umpire-typed-unary-queue-repeat",
	}
	live := bindCase(t, env, caseSource, repeat)

	runIDs := make(map[string]struct{}, 2)
	for range 2 {
		run, verdict, err := live.prepared.Run(env.Context(), live.driver)
		require.NoError(t, err)
		require.Equal(t, testpilotpb.RUN_STATUS_COMPLETED, run.GetStatus())
		require.Equal(t, testpilotpb.CLEANUP_STATUS_SUCCEEDED, run.GetCleanup().GetStatus())
		require.Equal(t, testpilotpb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
		require.True(t, proto.Equal(verdict, run.GetVerdict()))
		require.Len(t, verdict.GetRules(), 1)
		require.Equal(t, testpilotpb.RULE_VERDICT_STATUS_SATISFIED, verdict.GetRules()[0].GetStatus())
		require.Len(t, verdict.GetRules()[0].GetSupportingEventSequences(), 1)
		requireSubmittedWorkflowTypeEvidence(t, run, verdict.GetSupportingEventSequences(), repeat.TaskQueue)

		require.NotContains(t, runIDs, run.GetRunId())
		runIDs[run.GetRunId()] = struct{}{}
		_, err = live.client.DescribeWorkflowExecution(env.Context(), run.GetRunId(), "")
		require.NoError(t, err)
	}

	require.True(t, proto.Equal(caseSnapshot, caseSource))
	require.True(t, proto.Equal(caseSnapshot, live.prepared.Snapshot()))
	require.Equal(t, live.profile.EnvironmentBindings, live.driver.Snapshot().EnvironmentBindings)
}

// requireSubmittedWorkflowTypeEvidence reads the supporting Observation back out of the Run and
// checks it is the started event whose recorded workflow type the clause compared.
func requireSubmittedWorkflowTypeEvidence(t testing.TB, run *testpilotpb.Run, sequences []int64, taskQueue string) {
	t.Helper()
	require.Len(t, sequences, 1)
	sequence := sequences[0]
	require.Positive(t, sequence)
	require.LessOrEqual(t, sequence, int64(len(run.GetEvents())))
	event := run.GetEvents()[sequence-1]
	require.Equal(t, "controller", event.GetCoordinates().GetEntrypointId())
	require.Equal(t, "history", event.GetCoordinates().GetInstructionId())
	require.Len(t, event.GetObservations(), 1)
	require.Equal(t, "history-event", event.GetObservations()[0].GetObservationId())

	var historyEvent historypb.HistoryEvent
	require.NoError(t, event.GetObservations()[0].GetValue().GetMessageValue().UnmarshalTo(&historyEvent))
	require.Equal(t, enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED, historyEvent.GetEventType())
	started := historyEvent.GetWorkflowExecutionStartedEventAttributes()
	require.Equal(t, testpilotfixture.TypedUnaryWorkflowType, started.GetWorkflowType().GetName())
	require.Equal(t, taskQueue, started.GetTaskQueue().GetName())
}
