//go:build test_dep && integration

package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/sdk/client"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/namespace"
	"go.temporal.io/server/common/testing/testpilot"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	testpilotfixture "go.temporal.io/server/tests/testcore/testpilot"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

// TestTestpilotTypedUnaryCase drives the generated StartWorkflowExecution example on the shared
// Driver. The modeled requirement is that the workflow type the request submitted is the workflow
// type the WorkflowExecutionStarted event records, so the Run's evidence is read back here from the
// declared Observation rather than restated by the test.
func TestTestpilotTypedUnaryCase(t *testing.T) {
	env := newTestpilotTestEnvironment(t)
	caseSource := loadTestpilotCase(t, "typed-unary")
	caseSnapshot := proto.CloneOf(caseSource)
	catalog, err := testpilotdriver.NewWorkflowServiceCatalog()
	require.NoError(t, err)

	environment := testpilotfixture.TypedUnaryEnvironment{
		Namespace: "umpire-typed-unary", TaskQueue: "umpire-typed-unary-queue",
	}
	_, err = env.RegisterNamespace(namespace.Name(environment.Namespace), 1, enumspb.ARCHIVAL_STATE_DISABLED, "", "")
	require.NoError(t, err)

	profile := testpilotfixture.TypedUnaryProfile(catalog, caseSource, environment)
	frozen := profile.Snapshot()
	expectedProfile := frozen.Snapshot()
	prepared, err := testpilot.Prepare(caseSource, frozen)
	require.NoError(t, err)
	caseClient, err := client.Dial(client.Options{HostPort: env.FrontendGRPCAddress(), Namespace: environment.Namespace})
	require.NoError(t, err)
	t.Cleanup(caseClient.Close)
	driver, err := testpilotdriver.New(testpilotdriver.Options{
		Profile: frozen,
		ServerEndpoints: map[string]testpilotdriver.Endpoint{
			"temporal.workflow-service": {Target: env.FrontendGRPCAddress(), Credentials: insecure.NewCredentials()},
		},
		SystemCallbackBaseURL: "http://" + env.HttpAPIAddress(),
		SDKClient:             caseClient,
		WorkerRoleID:          "temporal.worker",
		WorkerStopTimeout:     testpilotCleanupTimeout,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), testpilotCleanupTimeout)
		defer cancel()
		require.NoError(t, driver.Close(ctx))
	})

	runIDs := make(map[string]struct{}, 2)
	for range 2 {
		run, verdict, err := prepared.Run(env.Context(), driver)
		require.NoError(t, err)
		require.Equal(t, testpilotpb.RUN_STATUS_COMPLETED, run.GetStatus())
		require.Equal(t, testpilotpb.CLEANUP_STATUS_SUCCEEDED, run.GetCleanup().GetStatus())
		require.Equal(t, testpilotpb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
		require.True(t, proto.Equal(verdict, run.GetVerdict()))
		require.Len(t, verdict.GetRules(), 1)
		require.Equal(t, testpilotpb.RULE_VERDICT_STATUS_SATISFIED, verdict.GetRules()[0].GetStatus())
		require.Len(t, verdict.GetRules()[0].GetSupportingEventSequences(), 1)
		requireSubmittedWorkflowTypeEvidence(t, run, verdict.GetSupportingEventSequences())

		require.NotContains(t, runIDs, run.GetRunId())
		runIDs[run.GetRunId()] = struct{}{}
		_, err = caseClient.DescribeWorkflowExecution(env.Context(), run.GetRunId(), "")
		require.NoError(t, err)
	}

	require.True(t, proto.Equal(caseSnapshot, caseSource))
	require.True(t, proto.Equal(caseSnapshot, prepared.Snapshot()))
	require.Equal(t, expectedProfile.EnvironmentBindings, driver.Snapshot().EnvironmentBindings)
}

// requireSubmittedWorkflowTypeEvidence reads the supporting Observation back out of the Run and
// checks it is the started event whose recorded workflow type the clause compared.
func requireSubmittedWorkflowTypeEvidence(t testing.TB, run *testpilotpb.Run, sequences []int64) {
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
	require.Equal(t, "umpire-typed-unary-queue", started.GetTaskQueue().GetName())
}
