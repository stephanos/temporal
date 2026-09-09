//go:build test_dep && integration

package tests

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/serviceerror"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
)

const testpilotCleanupTimeout = 5 * time.Second

type testpilotLiveBinding struct {
	binding CaseBinding
	live    testpilotLiveCase
}

type testpilotLiveRunResult struct {
	environment int
	run         *testpilotpb.Run
	verdict     *testpilotpb.Verdict
	err         error
}

func TestTestpilotAsyncNexusCase(t *testing.T) {
	env := newTestpilotTestEnvironment(t)
	caseSource := loadTestpilotCase(t, "async-nexus")
	caseSnapshot := proto.CloneOf(caseSource)

	environments := []CaseBinding{
		{Identity: "async-nexus-profile", Namespace: "umpire-async-nexus-a", TaskQueue: "umpire-async-nexus-queue-a", NexusEndpoint: "umpire-async-nexus-endpoint-a", CreateEndpoint: true},
		{Identity: "async-nexus-profile", Namespace: "umpire-async-nexus-b", TaskQueue: "umpire-async-nexus-queue-b", NexusEndpoint: "umpire-async-nexus-endpoint-b", CreateEndpoint: true},
	}
	bindings := make([]testpilotLiveBinding, len(environments))
	for index, environment := range environments {
		live := bindCase(t, env, caseSource, environment)
		bindings[index] = testpilotLiveBinding{binding: environment, live: live}
	}

	require.True(t, proto.Equal(caseSnapshot, caseSource))
	require.True(t, proto.Equal(bindings[0].live.prepared.Snapshot(), bindings[1].live.prepared.Snapshot()))
	require.True(t, proto.Equal(bindings[0].live.prepared.Snapshot().GetContract(), bindings[1].live.prepared.Snapshot().GetContract()))
	require.True(t, proto.Equal(bindings[0].live.prepared.Snapshot().GetProvenance(), bindings[1].live.prepared.Snapshot().GetProvenance()))
	require.Equal(t, bindings[0].live.prepared.Snapshot().GetCaseId(), bindings[1].live.prepared.Snapshot().GetCaseId())
	require.Equal(t, bindings[0].live.prepared.Snapshot().GetProgram().GetProgramId(), bindings[1].live.prepared.Snapshot().GetProgram().GetProgramId())
	require.Equal(t, bindings[0].live.prepared.Snapshot().GetContract().GetContractId(), bindings[1].live.prepared.Snapshot().GetContract().GetContractId())
	require.NotEqual(t, bindings[0].live.prepared.Identity().Bindings, bindings[1].live.prepared.Identity().Bindings)

	results := make(chan testpilotLiveRunResult, len(bindings)*2)
	var runs sync.WaitGroup
	for index, binding := range bindings {
		for range 2 {
			runs.Go(func() {
				run, verdict, err := binding.live.prepared.Run(env.Context(), binding.live.driver)
				results <- testpilotLiveRunResult{environment: index, run: run, verdict: verdict, err: err}
			})
		}
	}
	runs.Wait()
	close(results)

	runIDs := make(map[string]struct{}, len(bindings)*2)
	for result := range results {
		require.NoError(t, result.err)
		require.Equal(t, testpilotpb.RUN_STATUS_COMPLETED, result.run.GetStatus())
		require.Equal(t, testpilotpb.CLEANUP_STATUS_SUCCEEDED, result.run.GetCleanup().GetStatus())
		require.Equal(t, testpilotpb.VERDICT_STATUS_SATISFIED, result.verdict.GetStatus())
		require.True(t, proto.Equal(result.verdict, result.run.GetVerdict()))
		require.Len(t, result.verdict.GetRules(), 1)
		require.Equal(t, testpilotpb.RULE_VERDICT_STATUS_SATISFIED, result.verdict.GetRules()[0].GetStatus())
		require.Len(t, result.verdict.GetRules()[0].GetSupportingEventSequences(), 3)
		requireCorrelatedNexusHistoryEvidence(t, result.run, result.verdict.GetSupportingEventSequences(), bindings[result.environment].binding.NexusEndpoint)
		require.NotContains(t, runIDs, result.run.GetRunId())
		runIDs[result.run.GetRunId()] = struct{}{}

		_, err := bindings[result.environment].live.client.DescribeWorkflowExecution(env.Context(), result.run.GetRunId(), "")
		require.NoError(t, err)
		_, err = bindings[1-result.environment].live.client.DescribeWorkflowExecution(env.Context(), result.run.GetRunId(), "")
		var notFound *serviceerror.NotFound
		require.ErrorAs(t, err, &notFound)
	}
	require.True(t, proto.Equal(caseSnapshot, caseSource))
	for _, binding := range bindings {
		require.Equal(t, binding.live.profile.EnvironmentBindings, binding.live.driver.Snapshot().EnvironmentBindings)
		require.True(t, proto.Equal(caseSnapshot, binding.live.prepared.Snapshot()))
	}
}

func TestTestpilotAsyncNexusCaseMissingRemoteEndpoint(t *testing.T) {
	env := newTestpilotTestEnvironment(t)
	caseSource := loadTestpilotCase(t, "async-nexus")
	// The endpoint the Case binds is deliberately not created, so the Nexus operation never
	// completes and the Run closes incomplete and inconclusive.
	live := bindCase(t, env, caseSource, CaseBinding{
		Identity: "async-nexus-profile", Namespace: "umpire-async-nexus-missing",
		TaskQueue: "umpire-async-nexus-queue-missing", NexusEndpoint: "umpire-async-nexus-endpoint-missing",
		CreateEndpoint: false,
	})

	run, verdict, err := live.prepared.Run(env.Context(), live.driver)
	require.NoError(t, err)
	require.Equal(t, testpilotpb.RUN_STATUS_INCOMPLETE, run.GetStatus())
	require.Equal(t, testpilotpb.CLEANUP_STATUS_SUCCEEDED, run.GetCleanup().GetStatus())
	require.Equal(t, testpilotpb.VERDICT_STATUS_INCONCLUSIVE, verdict.GetStatus())
	requireRunHasOutcome(t, run, "await-completion-authority", testpilotpb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT)
	_, err = live.client.DescribeWorkflowExecution(env.Context(), run.GetRunId(), "")
	require.NoError(t, err)
}

func loadTestpilotCase(t testing.TB, name string) *testpilotpb.Case {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("testcore", "testpilot", "testdata", name+"-case.json"))
	require.NoError(t, err)
	decoded, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)
	return decoded
}

func requireCorrelatedNexusHistoryEvidence(t testing.TB, run *testpilotpb.Run, sequences []int64, endpoint string) {
	t.Helper()
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
		enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED,
		enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED,
	}, []enumspb.EventType{events[0].GetEventType(), events[1].GetEventType(), events[2].GetEventType()})
	scheduledID := events[0].GetEventId()
	scheduled := events[0].GetNexusOperationScheduledEventAttributes()
	requestID := scheduled.GetRequestId()
	require.Equal(t, endpoint, scheduled.GetEndpoint())
	require.Positive(t, scheduledID)
	require.NotEmpty(t, requestID)
	require.Equal(t, scheduledID, events[1].GetNexusOperationStartedEventAttributes().GetScheduledEventId())
	require.Equal(t, requestID, events[1].GetNexusOperationStartedEventAttributes().GetRequestId())
	require.Equal(t, scheduledID, events[2].GetNexusOperationCompletedEventAttributes().GetScheduledEventId())
	require.Equal(t, requestID, events[2].GetNexusOperationCompletedEventAttributes().GetRequestId())
}

func requireRunHasOutcome(t testing.TB, run *testpilotpb.Run, instructionID string, status testpilotpb.InstructionOutcomeStatus) {
	t.Helper()
	for _, event := range run.GetEvents() {
		if event.GetCoordinates().GetInstructionId() == instructionID && event.GetOutcome().GetStatus() == status {
			return
		}
	}
	require.Fail(t, "Run does not contain expected instruction outcome", "instruction: %s, status: %s", instructionID, status)
}
