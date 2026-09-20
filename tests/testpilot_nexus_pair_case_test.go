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

// nexusPairCleanupTimeout gives the two Nexus handler entrypoints room to stop within the
// Program's own declared cleanup budget.
const nexusPairCleanupTimeout = 15 * time.Second

// TestTestpilotNexusPairCase drives the pair Model's Case on the shared Driver. The workflow
// schedules two instances of the operation on one endpoint and awaits both; each instance's
// capture rule retains the scheduled event that records its own operation name and requires the
// completion that references exactly that scheduled event, so the Run's evidence is read back here
// from the declared Observation rather than restated by the test.
func TestTestpilotNexusPairCase(t *testing.T) {
	env := newTestpilotTestEnvironment(t)
	caseSource := loadTestpilotCase(t, testpilotfixture.NexusPairFixture)
	caseSnapshot := proto.CloneOf(caseSource)

	environment := CaseBinding{
		Identity:       "nexus-pair-profile",
		Namespace:      "umpire-nexus-pair",
		TaskQueue:      "umpire-nexus-pair-queue",
		NexusEndpoint:  "umpire-nexus-pair-endpoint",
		CreateEndpoint: true,
		CleanupTimeout: nexusPairCleanupTimeout,
	}
	live := bindCase(t, env, caseSource, environment)

	rules := []string{testpilotfixture.NexusPairFirstRuleID, testpilotfixture.NexusPairSecondRuleID}
	operations := []string{testpilotfixture.NexusPairFirstOperation, testpilotfixture.NexusPairSecondOperation}
	runIDs := make(map[string]struct{}, 2)
	for range 2 {
		run, verdict, err := live.prepared.Run(env.Context(), live.driver)
		require.NoError(t, err)
		require.Equal(t, testpilotpb.RUN_DISPOSITION_COMPLETED, run.GetDisposition(), "diagnostics: %v", run.GetDiagnostics())
		require.Equal(t, testpilotpb.CLEANUP_STATUS_SUCCEEDED, run.GetCleanup().GetStatus())
		require.Equal(t, testpilotpb.VERDICT_STATUS_SATISFIED, verdict.GetStatus(), "diagnostics: %v", run.GetDiagnostics())
		require.True(t, proto.Equal(verdict, run.GetVerdict()))
		// One capture rule per instance, then the scoped clauses the Query's Property lowered into.
		require.Greater(t, len(verdict.GetRules()), len(rules))
		for index, ruleID := range rules {
			rule := verdict.GetRules()[index]
			require.Equal(t, ruleID, rule.GetRuleId())
			require.Equal(t, testpilotpb.RULE_VERDICT_STATUS_SATISFIED, rule.GetStatus())
			requireCorrelatedNexusPairEvidence(t, run, rule.GetSupportingEventSequences(), operations[index], environment.NexusEndpoint)
		}
		for _, rule := range verdict.GetRules()[len(rules):] {
			require.Equal(t, testpilotpb.RULE_VERDICT_STATUS_SATISFIED, rule.GetStatus())
			require.NotEmpty(t, rule.GetSupportingEventSequences())
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

// requireCorrelatedNexusPairEvidence reads one capture rule's supporting Observations back out of
// the Run and checks they are the scheduled event that records the rule's own operation and the
// completion that references exactly that scheduled event.
func requireCorrelatedNexusPairEvidence(t testing.TB, run *testpilotpb.Run, sequences []int64, operation, endpoint string) {
	t.Helper()
	require.Len(t, sequences, 2)
	events := make([]*historypb.HistoryEvent, 0, len(sequences))
	for _, sequence := range sequences {
		event := runEventAt(t, run, sequence)
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
	require.Equal(t, testpilotfixture.NexusPairService, scheduled.GetService())
	require.Equal(t, endpoint, scheduled.GetEndpoint())
	require.Positive(t, events[0].GetEventId())
	require.Equal(t, events[0].GetEventId(), events[1].GetNexusOperationCompletedEventAttributes().GetScheduledEventId())
	require.Equal(t, scheduled.GetRequestId(), events[1].GetNexusOperationCompletedEventAttributes().GetRequestId())
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
