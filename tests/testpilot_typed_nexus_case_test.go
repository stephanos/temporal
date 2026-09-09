//go:build test_dep && integration

package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	testpilotfixture "go.temporal.io/server/tests/testcore/testpilot"
	"google.golang.org/protobuf/proto"
)

// typedNexusCleanupTimeout gives the two Nexus handler entrypoints room to stop within the
// Program's own declared cleanup budget.
const typedNexusCleanupTimeout = 15 * time.Second

// TestTestpilotTypedNexusOperationsCase drives the two-operation Nexus example on the shared
// Driver. The workflow schedules both declared operations on one endpoint and awaits both; each
// derived Contract rule retains the scheduled event of its own operation identity and requires the
// completion that references exactly that scheduled event, so the Run's evidence is read back here
// from the declared Observation rather than restated by the test.
func TestTestpilotTypedNexusOperationsCase(t *testing.T) {
	env := newTestpilotTestEnvironment(t)
	caseSource := loadTestpilotCase(t, "typed-nexus")
	caseSnapshot := proto.CloneOf(caseSource)

	environment := CaseBinding{
		Identity:      "typed-nexus-profile",
		Namespace:     "umpire-typed-nexus",
		TaskQueue:     "umpire-typed-nexus-queue",
		NexusEndpoint: "umpire-typed-nexus-endpoint",

		CreateEndpoint: true,
		CleanupTimeout: typedNexusCleanupTimeout,
	}
	live := bindCase(t, env, caseSource, environment)

	operations := []string{testpilotfixture.TypedNexusFirstOperation, testpilotfixture.TypedNexusSecondOperation}
	runIDs := make(map[string]struct{}, 2)
	for range 2 {
		run, verdict, err := live.prepared.Run(env.Context(), live.driver)
		require.NoError(t, err)
		require.Equal(t, testpilotpb.RUN_STATUS_COMPLETED, run.GetStatus())
		require.Equal(t, testpilotpb.CLEANUP_STATUS_SUCCEEDED, run.GetCleanup().GetStatus())
		require.Equal(t, testpilotpb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
		require.True(t, proto.Equal(verdict, run.GetVerdict()))
		require.Len(t, verdict.GetRules(), len(operations))
		for index, rule := range verdict.GetRules() {
			require.Equal(t, testpilotpb.RULE_VERDICT_STATUS_SATISFIED, rule.GetStatus())
			requireCorrelatedNexusOperationEvidence(t, run, rule.GetSupportingEventSequences(), operations[index], environment.NexusEndpoint)
		}

		require.NotContains(t, runIDs, run.GetRunId())
		runIDs[run.GetRunId()] = struct{}{}
		_, err = live.client.DescribeWorkflowExecution(env.Context(), run.GetRunId(), "")
		require.NoError(t, err)
	}

	require.True(t, proto.Equal(caseSnapshot, caseSource))
	require.True(t, proto.Equal(caseSnapshot, live.prepared.Snapshot()))
	require.Equal(t, live.profile.EnvironmentBindings, live.driver.Snapshot().EnvironmentBindings)
}

// requireCorrelatedNexusOperationEvidence reads one rule's supporting Observations back out of the
// Run and checks they are the scheduled event of the rule's own operation identity and the
// completion that references exactly that scheduled event.
func requireCorrelatedNexusOperationEvidence(t testing.TB, run *testpilotpb.Run, sequences []int64, operation, endpoint string) {
	t.Helper()
	require.Len(t, sequences, 2)
	events := make([]*historypb.HistoryEvent, 0, len(sequences))
	for _, sequence := range sequences {
		require.Positive(t, sequence)
		require.LessOrEqual(t, sequence, int64(len(run.GetEvents())))
		event := run.GetEvents()[sequence-1]
		require.Equal(t, "controller", event.GetCoordinates().GetEntrypointId())
		require.Equal(t, "history", event.GetCoordinates().GetInstructionId())
		require.Len(t, event.GetObservations(), 1)
		require.Equal(t, "history-event", event.GetObservations()[0].GetObservationId())
		var historyEvent historypb.HistoryEvent
		require.NoError(t, event.GetObservations()[0].GetValue().GetMessageValue().UnmarshalTo(&historyEvent))
		events = append(events, &historyEvent)
	}

	require.Equal(t, []enumspb.EventType{
		enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED,
		enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED,
	}, []enumspb.EventType{events[0].GetEventType(), events[1].GetEventType()})
	scheduled := events[0].GetNexusOperationScheduledEventAttributes()
	require.Equal(t, operation, scheduled.GetOperation())
	require.Equal(t, testpilotfixture.TypedNexusService, scheduled.GetService())
	require.Equal(t, endpoint, scheduled.GetEndpoint())
	require.Positive(t, events[0].GetEventId())
	require.Equal(t, events[0].GetEventId(), events[1].GetNexusOperationCompletedEventAttributes().GetScheduledEventId())
	require.Equal(t, scheduled.GetRequestId(), events[1].GetNexusOperationCompletedEventAttributes().GetRequestId())
}
