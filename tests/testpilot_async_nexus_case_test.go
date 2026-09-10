//go:build test_dep && integration

package tests

import (
	"os"
	"path/filepath"
	"strconv"
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
		// One rule verdict per scoped clause the checked Property lowered into, each answered by the
		// two recorded Nexus events the projection admitted as this operation's semantic steps.
		require.Len(t, result.verdict.GetRules(), 3)
		for _, rule := range result.verdict.GetRules() {
			require.Equal(t, testpilotpb.RULE_VERDICT_STATUS_SATISFIED, rule.GetStatus())
			require.Equal(t, result.verdict.GetSupportingEventSequences(), rule.GetSupportingEventSequences())
		}
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

// requireCorrelatedNexusHistoryEvidence reads the supporting evidence back out of the Run. Each
// supporting event carries both the history event it projected and the CorrelatedEvidence the same
// projection lifted from it, and the two agree: the operation key every value carries is the
// scheduled event both the started and the completed event name.
func requireCorrelatedNexusHistoryEvidence(t testing.TB, run *testpilotpb.Run, sequences []int64, endpoint string) {
	t.Helper()
	require.Len(t, sequences, 2)
	events := make([]*historypb.HistoryEvent, 0, len(sequences))
	keys := make([]string, 0, len(sequences))
	kinds := make([]string, 0, len(sequences))
	for _, sequence := range sequences {
		event := runEventAt(t, run, sequence)
		require.Equal(t, "controller", event.GetCoordinates().GetEntrypointId())
		require.Equal(t, "history", event.GetCoordinates().GetInstructionId())
		var historyEvent historypb.HistoryEvent
		require.NoError(t, observationValue(t, event, "history-event").GetMessageValue().UnmarshalTo(&historyEvent))
		events = append(events, &historyEvent)
		var evidence testpilotpb.CorrelatedEvidence
		require.NoError(t, observationValue(t, event, "correlated-evidence").GetMessageValue().UnmarshalTo(&evidence))
		keys = append(keys, evidence.GetOperation())
		kinds = append(kinds, evidence.GetKind())
	}
	require.Equal(t, []enumspb.EventType{
		enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED,
		enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED,
	}, []enumspb.EventType{events[0].GetEventType(), events[1].GetEventType()})
	require.Equal(t, []string{
		"temporal.nexus.success.evidence.started", "temporal.nexus.success.evidence.completed",
	}, kinds)

	scheduledID := events[0].GetNexusOperationStartedEventAttributes().GetScheduledEventId()
	require.Positive(t, scheduledID)
	require.Equal(t, scheduledID, events[1].GetNexusOperationCompletedEventAttributes().GetScheduledEventId())
	require.Equal(t, []string{strconv.FormatInt(scheduledID, 10), strconv.FormatInt(scheduledID, 10)}, keys)
	requestID := events[0].GetNexusOperationStartedEventAttributes().GetRequestId()
	require.NotEmpty(t, requestID)
	require.Equal(t, requestID, events[1].GetNexusOperationCompletedEventAttributes().GetRequestId())
	requireScheduledNexusEndpoint(t, run, scheduledID, requestID, endpoint)
}

// requireScheduledNexusEndpoint finds the scheduled event the two supporting events name. It
// supports no clause of its own -- the model's operation is already scheduled when it starts -- so
// it is read from the Run rather than from the Verdict.
func requireScheduledNexusEndpoint(t testing.TB, run *testpilotpb.Run, scheduledID int64, requestID, endpoint string) {
	t.Helper()
	for _, event := range run.GetEvents() {
		for _, observation := range event.GetObservations() {
			if observation.GetObservationId() != "history-event" {
				continue
			}
			var historyEvent historypb.HistoryEvent
			require.NoError(t, observation.GetValue().GetMessageValue().UnmarshalTo(&historyEvent))
			if historyEvent.GetEventId() != scheduledID {
				continue
			}
			scheduled := historyEvent.GetNexusOperationScheduledEventAttributes()
			require.Equal(t, endpoint, scheduled.GetEndpoint())
			require.Equal(t, requestID, scheduled.GetRequestId())
			return
		}
	}
	require.FailNow(t, "scheduled Nexus event not recorded")
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
