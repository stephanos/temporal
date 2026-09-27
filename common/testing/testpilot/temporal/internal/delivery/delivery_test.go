package delivery

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/primitive"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// binding keeps the codec tests, the route wire golden among them, compiling unedited.
type binding = WorkflowBinding

type fixture struct {
	ledger   *Ledger
	origin   testpilot.Coordinate
	plan     testpilot.ReservationCarrierPlan
	binding  WorkflowBinding
	workflow *testsupport.Reservation
	handler  *testsupport.Reservation
	bundle   Bundle
}

func newFixture(t *testing.T, runID, sessionID string) *fixture {
	t.Helper()
	ledger, err := New(Config{
		RunID:     runID,
		SessionID: sessionID,
		Limits:    Limits{MaxRoutes: 8, MaxHeaderBytes: 4096, MaxHandles: 8, MaxDiagnostics: 8},
	})
	require.NoError(t, err)
	origin := testpilot.Coordinate{RunID: runID, EntrypointID: "controller", ActivationID: "controller.0", InstructionID: "start-workflow", Attempt: 1}
	plan := testpilot.ReservationCarrierPlan{
		EndpointRoleID: "temporal",
		Method:         primitive.StartWorkflowPath,
		Reservations: []testpilot.ReservationTopology{
			{EntrypointID: "workflow", Kind: testpilot.WorkflowEntrypoint, Count: 1},
			{EntrypointID: "handler", Kind: testpilot.NexusHandlerEntrypoint, Count: 1},
		},
		Routes: []testpilot.ReservationRoute{{WorkflowEntrypointID: "workflow", WorkflowOrdinal: 0, SourceInstructionID: "start-nexus", HandlerEntrypointID: "handler", HandlerOrdinal: 0}},
	}
	workflow := testsupport.NewReservation(testpilot.ReservationIdentity{Origin: origin, EntrypointID: "workflow", Ordinal: 0, ID: sessionID + "-workflow"})
	handler := testsupport.NewReservation(testpilot.ReservationIdentity{Origin: origin, EntrypointID: "handler", Ordinal: 0, ID: sessionID + "-handler"})
	retainedHandler, err := ledger.RetainReservation(context.Background(), handler)
	require.NoError(t, err)
	retainedWorkflow, err := ledger.RetainReservation(context.Background(), workflow)
	require.NoError(t, err)
	bundle, err := ledger.CreateBundle(context.Background(), origin, plan, WorkflowBinding{Namespace: "namespace", WorkflowID: "workflow-id", WorkflowType: "workflow-type", TaskQueue: "task-queue"}, []testpilot.ReservationHandle{retainedHandler, retainedWorkflow})
	require.NoError(t, err)
	return &fixture{ledger: ledger, origin: origin, plan: plan, binding: WorkflowBinding{Namespace: "namespace", WorkflowID: "workflow-id", WorkflowType: "workflow-type", TaskQueue: "task-queue"}, workflow: workflow, handler: handler, bundle: bundle}
}

func startMethod(t *testing.T) protoreflect.MethodDescriptor {
	t.Helper()
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName("temporal.api.workflowservice.v1.WorkflowService.StartWorkflowExecution")
	require.NoError(t, err)
	method, ok := descriptor.(protoreflect.MethodDescriptor)
	require.True(t, ok)
	return method
}

func workflowRequest(f *fixture) *workflowservice.StartWorkflowExecutionRequest {
	return &workflowservice.StartWorkflowExecutionRequest{
		Namespace:    f.binding.Namespace,
		WorkflowId:   f.binding.WorkflowID,
		WorkflowType: &commonpb.WorkflowType{Name: f.binding.WorkflowType},
		TaskQueue:    &taskqueuepb.TaskQueue{Name: f.binding.TaskQueue},
		RequestId:    "application-request-id",
	}
}

func workflowHeader(t *testing.T, f *fixture) *commonpb.Header {
	t.Helper()
	prepared, err := f.ledger.PrepareRPC(context.Background(), &f.bundle, "temporal", startMethod(t), workflowRequest(f), 1<<20)
	require.NoError(t, err)
	return prepared.(*workflowservice.StartWorkflowExecutionRequest).Header
}

func admitWorkflow(t *testing.T, f *fixture, temporalRunID string) Activation {
	t.Helper()
	activation, err := f.ledger.AdmitWorkflow(context.Background(), WorkflowDelivery{
		Header:        workflowHeader(t, f),
		Namespace:     f.binding.Namespace,
		WorkflowID:    f.binding.WorkflowID,
		WorkflowType:  f.binding.WorkflowType,
		TaskQueue:     f.binding.TaskQueue,
		TemporalRunID: temporalRunID,
	})
	require.NoError(t, err)
	return activation
}

func requireContextError(t *testing.T, err error) {
	t.Helper()
	require.True(t, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded), err)
}
