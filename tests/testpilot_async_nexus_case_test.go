//go:build test_dep && integration

package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/nexus/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/sdk/client"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/namespace"
	"go.temporal.io/server/common/testing/testpilot"
	testpilotdriver "go.temporal.io/server/tests/testcore/testpilot"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

func TestTestpilotAsyncNexusCase(t *testing.T) {
	env := newTestpilotTestEnvironment(t)
	caseSource := loadTestpilotCase(t, "async-nexus")
	catalog, err := testpilotdriver.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	profile := testpilot.ProfileSpec{
		Identity: "temporal-async-nexus-case-profile",
		Catalog:  catalog,
		Roles: []testpilot.RolePolicy{
			{
				ID: "temporal.workflow-service", Kind: testpilotpb.ROLE_KIND_ENDPOINT,
				Methods: []string{
					"/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution",
					"/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory",
				},
				ReservationCarriers: []testpilot.ReservationCarrierPolicy{{
					Method: "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution",
					Shapes: []testpilot.ReservationCarrierShape{
						{Context: testpilotpb.ENTRYPOINT_KIND_WORKFLOW, MaximumCount: 1},
						{Context: testpilotpb.ENTRYPOINT_KIND_NEXUS_HANDLER, MaximumCount: 1},
					},
				}},
			},
			{ID: "temporal.worker", Kind: testpilotpb.ROLE_KIND_WORKER},
			{ID: "temporal.task-queue", Kind: testpilotpb.ROLE_KIND_TASK_QUEUE},
			{ID: "temporal.nexus-endpoint", Kind: testpilotpb.ROLE_KIND_ENDPOINT},
		},
		Capabilities: []testpilot.Capability{
			testpilot.InvokeRPC, testpilot.AwaitSlot, testpilot.CompleteNexusOperation,
			testpilot.StartNexusOperation, testpilot.Await, testpilot.Finish, testpilot.RespondNexus,
		},
		ProgramLimits:  proto.CloneOf(caseSource.GetProgram().GetLimits()),
		ContractLimits: proto.CloneOf(caseSource.GetContract().GetLimits()),
	}
	prepared, err := testpilot.Prepare(caseSource, profile)
	require.NoError(t, err)
	_, err = env.RegisterNamespace(namespace.Name("default"), 1, enumspb.ARCHIVAL_STATE_DISABLED, "", "")
	require.NoError(t, err)
	caseClient, err := client.Dial(client.Options{HostPort: env.FrontendGRPCAddress(), Namespace: "default"})
	require.NoError(t, err)
	t.Cleanup(caseClient.Close)

	const taskQueue = "umpire-async-nexus-workflow-queue"
	const endpointName = "umpire-async-nexus-endpoint"
	const cleanupTimeout = 5 * time.Second
	created, err := env.OperatorClient().CreateNexusEndpoint(env.Context(), &operatorservice.CreateNexusEndpointRequest{
		Spec: &nexus.EndpointSpec{
			Name: endpointName,
			Target: &nexus.EndpointTarget{Variant: &nexus.EndpointTarget_Worker_{
				Worker: &nexus.EndpointTarget_Worker{Namespace: "default", TaskQueue: taskQueue},
			}},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
		defer cancel()
		_, err := env.OperatorClient().DeleteNexusEndpoint(ctx, &operatorservice.DeleteNexusEndpointRequest{
			Id: created.GetEndpoint().GetId(), Version: created.GetEndpoint().GetVersion(),
		})
		require.NoError(t, err)
	})

	driver, err := testpilotdriver.New(testpilotdriver.Options{
		Profile: profile,
		ServerEndpoints: map[string]testpilotdriver.Endpoint{
			"temporal.workflow-service": {Target: env.FrontendGRPCAddress(), Credentials: insecure.NewCredentials()},
		},
		SystemCallbackBaseURL: "http://" + env.HttpAPIAddress(),
		SDKClient:             caseClient, Namespace: "default", WorkerRoleID: "temporal.worker",
		TaskQueues:        []testpilotdriver.RoleBinding{{RoleID: "temporal.task-queue", Value: taskQueue}},
		NexusEndpoints:    []testpilotdriver.RoleBinding{{RoleID: "temporal.nexus-endpoint", Value: endpointName}},
		WorkerStopTimeout: cleanupTimeout,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
		defer cancel()
		require.NoError(t, driver.Close(ctx))
	})

	run, verdict, err := prepared.Run(env.Context(), driver)
	require.NoError(t, err)
	require.Equal(t, testpilotpb.RUN_STATUS_COMPLETED, run.GetStatus())
	require.Equal(t, testpilotpb.CLEANUP_STATUS_SUCCEEDED, run.GetCleanup().GetStatus())
	require.Equal(t, testpilotpb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
	require.True(t, proto.Equal(verdict, run.GetVerdict()))
	require.Len(t, verdict.GetRules(), 1)
	require.Equal(t, testpilotpb.RULE_VERDICT_STATUS_SATISFIED, verdict.GetRules()[0].GetStatus())
	require.Len(t, verdict.GetRules()[0].GetSupportingEventSequences(), 3)
	requireCorrelatedNexusHistoryEvidence(t, run, verdict.GetSupportingEventSequences())
}

func loadTestpilotCase(t testing.TB, name string) *testpilotpb.Case {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("testcore", "testpilot", "testdata", name+"-case.json"))
	require.NoError(t, err)
	decoded, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)
	return decoded
}

func requireCorrelatedNexusHistoryEvidence(t testing.TB, run *testpilotpb.Run, sequences []int64) {
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
	requestID := events[0].GetNexusOperationScheduledEventAttributes().GetRequestId()
	require.Positive(t, scheduledID)
	require.NotEmpty(t, requestID)
	require.Equal(t, scheduledID, events[1].GetNexusOperationStartedEventAttributes().GetScheduledEventId())
	require.Equal(t, requestID, events[1].GetNexusOperationStartedEventAttributes().GetRequestId())
	require.Equal(t, scheduledID, events[2].GetNexusOperationCompletedEventAttributes().GetScheduledEventId())
	require.Equal(t, requestID, events[2].GetNexusOperationCompletedEventAttributes().GetRequestId())
}
