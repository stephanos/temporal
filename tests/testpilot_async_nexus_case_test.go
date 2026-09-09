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
	"go.temporal.io/sdk/client"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/tests/testcore"
	testpilotfixture "go.temporal.io/server/tests/testcore/testpilot"
	"google.golang.org/protobuf/proto"
)

const testpilotCleanupTimeout = 5 * time.Second

type testpilotLiveBinding struct {
	environment testpilotfixture.AsyncNexusEnvironment
	profile     testpilot.ProfileSpec
	prepared    *testpilot.PreparedCase
	client      client.Client
	driver      *testpilotdriver.Driver
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
	catalog, err := testpilotdriver.NewWorkflowServiceCatalog()
	require.NoError(t, err)

	environments := []testpilotfixture.AsyncNexusEnvironment{
		{Namespace: "umpire-async-nexus-a", TaskQueue: "umpire-async-nexus-queue-a", NexusEndpoint: "umpire-async-nexus-endpoint-a"},
		{Namespace: "umpire-async-nexus-b", TaskQueue: "umpire-async-nexus-queue-b", NexusEndpoint: "umpire-async-nexus-endpoint-b"},
	}
	bindings := make([]testpilotLiveBinding, len(environments))
	for index, environment := range environments {
		bindings[index] = newTestpilotLiveBinding(t, env, catalog, caseSource, environment, true)
	}

	require.True(t, proto.Equal(caseSnapshot, caseSource))
	require.True(t, proto.Equal(bindings[0].prepared.Snapshot(), bindings[1].prepared.Snapshot()))
	require.True(t, proto.Equal(bindings[0].prepared.Snapshot().GetContract(), bindings[1].prepared.Snapshot().GetContract()))
	require.True(t, proto.Equal(bindings[0].prepared.Snapshot().GetProvenance(), bindings[1].prepared.Snapshot().GetProvenance()))
	require.Equal(t, bindings[0].prepared.Snapshot().GetCaseId(), bindings[1].prepared.Snapshot().GetCaseId())
	require.Equal(t, bindings[0].prepared.Snapshot().GetProgram().GetProgramId(), bindings[1].prepared.Snapshot().GetProgram().GetProgramId())
	require.Equal(t, bindings[0].prepared.Snapshot().GetContract().GetContractId(), bindings[1].prepared.Snapshot().GetContract().GetContractId())
	require.NotEqual(t, bindings[0].prepared.Identity().Bindings, bindings[1].prepared.Identity().Bindings)

	results := make(chan testpilotLiveRunResult, len(bindings)*2)
	var runs sync.WaitGroup
	for index, binding := range bindings {
		for range 2 {
			runs.Go(func() {
				run, verdict, err := binding.prepared.Run(env.Context(), binding.driver)
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
		requireCorrelatedNexusHistoryEvidence(t, result.run, result.verdict.GetSupportingEventSequences(), bindings[result.environment].environment.NexusEndpoint)
		require.NotContains(t, runIDs, result.run.GetRunId())
		runIDs[result.run.GetRunId()] = struct{}{}

		_, err := bindings[result.environment].client.DescribeWorkflowExecution(env.Context(), result.run.GetRunId(), "")
		require.NoError(t, err)
		_, err = bindings[1-result.environment].client.DescribeWorkflowExecution(env.Context(), result.run.GetRunId(), "")
		var notFound *serviceerror.NotFound
		require.ErrorAs(t, err, &notFound)
	}
	require.True(t, proto.Equal(caseSnapshot, caseSource))
	for _, binding := range bindings {
		require.Equal(t, binding.profile.EnvironmentBindings, binding.driver.Snapshot().EnvironmentBindings)
		require.True(t, proto.Equal(caseSnapshot, binding.prepared.Snapshot()))
	}
}

func TestTestpilotAsyncNexusCaseMissingRemoteEndpoint(t *testing.T) {
	env := newTestpilotTestEnvironment(t)
	caseSource := loadTestpilotCase(t, "async-nexus")
	catalog, err := testpilotdriver.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	binding := newTestpilotLiveBinding(t, env, catalog, caseSource, testpilotfixture.AsyncNexusEnvironment{
		Namespace: "umpire-async-nexus-missing", TaskQueue: "umpire-async-nexus-queue-missing",
		NexusEndpoint: "umpire-async-nexus-endpoint-missing",
	}, false)

	run, verdict, err := binding.prepared.Run(env.Context(), binding.driver)
	require.NoError(t, err)
	require.Equal(t, testpilotpb.RUN_STATUS_INCOMPLETE, run.GetStatus())
	require.Equal(t, testpilotpb.CLEANUP_STATUS_SUCCEEDED, run.GetCleanup().GetStatus())
	require.Equal(t, testpilotpb.VERDICT_STATUS_INCONCLUSIVE, verdict.GetStatus())
	requireRunHasOutcome(t, run, "await-completion-authority", testpilotpb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT)
	_, err = binding.client.DescribeWorkflowExecution(env.Context(), run.GetRunId(), "")
	require.NoError(t, err)
}

func newTestpilotLiveBinding(
	t *testing.T,
	env *testcore.TestEnv,
	catalog *testpilot.Catalog,
	caseSource *testpilotpb.Case,
	environment testpilotfixture.AsyncNexusEnvironment,
	createEndpoint bool,
) testpilotLiveBinding {
	t.Helper()
	profile := testpilotfixture.AsyncNexusProfile(catalog, caseSource, environment)
	frozenEnvironment := requireTestpilotProfileEnvironment(t, profile.Snapshot())
	resources := testpilotLiveResources{
		Namespace: frozenEnvironment.Namespace, TaskQueue: frozenEnvironment.TaskQueue,
	}
	if createEndpoint {
		resources.NexusEndpoint = frozenEnvironment.NexusEndpoint
	}
	live := newTestpilotLiveCase(t, env, caseSource, profile, resources, testpilotCleanupTimeout)
	return testpilotLiveBinding{
		environment: frozenEnvironment, profile: live.profile, prepared: live.prepared,
		client: live.client, driver: live.driver,
	}
}

func requireTestpilotProfileEnvironment(t testing.TB, profile testpilot.ProfileSpec) testpilotfixture.AsyncNexusEnvironment {
	t.Helper()
	bindings := make(map[string]string, len(profile.EnvironmentBindings))
	for _, binding := range profile.EnvironmentBindings {
		require.NotContains(t, bindings, binding.ID)
		bindings[binding.ID] = binding.Value
	}
	result := testpilotfixture.AsyncNexusEnvironment{
		Namespace:     bindings[testpilotfixture.AsyncNexusWorkerNamespaceBindingID],
		TaskQueue:     bindings[testpilotfixture.AsyncNexusTaskQueueBindingID],
		NexusEndpoint: bindings[testpilotfixture.AsyncNexusEndpointBindingID],
	}
	require.NotEmpty(t, result.Namespace)
	require.NotEmpty(t, result.TaskQueue)
	require.NotEmpty(t, result.NexusEndpoint)
	return result
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
