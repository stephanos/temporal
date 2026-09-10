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
		// One derived monitor rule per operation, then the scoped bounded-response clause the same
		// history feeds through its lifted CorrelatedEvidence.
		require.Len(t, verdict.GetRules(), len(operations)+1)
		for index, operation := range operations {
			rule := verdict.GetRules()[index]
			require.Equal(t, testpilotpb.RULE_VERDICT_STATUS_SATISFIED, rule.GetStatus())
			requireCorrelatedNexusOperationEvidence(t, run, rule.GetSupportingEventSequences(), operation, environment.NexusEndpoint)
		}
		scoped := verdict.GetRules()[len(operations)]
		require.Equal(t, testpilotpb.RULE_VERDICT_STATUS_SATISFIED, scoped.GetStatus())
		requireLiftedCorrelatedEvidence(t, run, scoped.GetSupportingEventSequences(), operations)

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
		var historyEvent historypb.HistoryEvent
		require.NoError(t, observationValue(t, event, "history-event").GetMessageValue().UnmarshalTo(&historyEvent))
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

// requireLiftedCorrelatedEvidence reads the scoped clause's supporting Observations back out of the Run
// and checks they are the CorrelatedEvidence the history projection lifted: one scheduled and one
// completed value per operation, keyed by the scheduled event both sides name, in one dense
// zero-based source stream.
func requireLiftedCorrelatedEvidence(t testing.TB, run *testpilotpb.Run, sequences []int64, operations []string) {
	t.Helper()
	require.Len(t, sequences, 2*len(operations))
	kinds := make([]string, 0, len(sequences))
	keys := make(map[string][]string, len(operations))
	for index, sequence := range sequences {
		require.Positive(t, sequence)
		require.LessOrEqual(t, sequence, int64(len(run.GetEvents())))
		event := run.GetEvents()[sequence-1]
		require.Equal(t, "history", event.GetCoordinates().GetInstructionId())
		var evidence testpilotpb.CorrelatedEvidence
		require.NoError(t, observationValue(t, event, "correlated-evidence").GetMessageValue().UnmarshalTo(&evidence))
		require.EqualValues(t, index, evidence.GetIdentity().GetOrdinal())
		require.NotEmpty(t, evidence.GetOperation())
		kinds = append(kinds, evidence.GetKind())
		keys[evidence.GetOperation()] = append(keys[evidence.GetOperation()], evidence.GetKind())
	}
	expected := make([]string, 0, len(sequences))
	for _, operation := range operations {
		expected = append(expected,
			"temporal.nexus3.typed-nexus.evidence.scheduled-"+operation,
			"temporal.nexus3.typed-nexus.evidence.completed")
	}
	require.Equal(t, expected, kinds)
	// Both sides of one operation land under one key: the scheduled event a completion references.
	require.Len(t, keys, len(operations))
	for _, observed := range keys {
		require.Len(t, observed, 2)
	}
}

// observationValue reads exactly one declared Observation off a recorded Run Event.
func observationValue(t testing.TB, event *testpilotpb.RunEvent, observationID string) *testpilotpb.Value {
	t.Helper()
	for _, observation := range event.GetObservations() {
		if observation.GetObservationId() == observationID {
			return observation.GetValue()
		}
	}
	require.FailNowf(t, "missing declared Observation", "%s at sequence %d", observationID, event.GetSequence())
	return nil
}
