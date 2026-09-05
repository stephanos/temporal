package delivery

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type fakeReservation struct {
	identity     umpire.ReservationIdentity
	activation   umpire.Coordinate
	consume      func(context.Context) (umpire.Coordinate, error)
	done         chan struct{}
	finishOnce   sync.Once
	result       umpire.EffectResult
	waitErr      error
	cancel       func(context.Context) error
	consumeCount atomic.Int64
	cancelCount  atomic.Int64
}

func newFakeReservation(identity umpire.ReservationIdentity) *fakeReservation {
	return &fakeReservation{
		identity:   identity,
		activation: umpire.Coordinate{RunID: identity.Origin.RunID, EntrypointID: identity.EntrypointID, ActivationID: identity.ID},
		done:       make(chan struct{}),
		result:     umpire.EffectResult{Outcome: &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}},
	}
}

func (h *fakeReservation) Identity() umpire.ReservationIdentity { return h.identity }
func (h *fakeReservation) Consume(ctx context.Context) (umpire.Coordinate, error) {
	h.consumeCount.Add(1)
	if h.consume != nil {
		return h.consume(ctx)
	}
	return h.activation, nil
}
func (h *fakeReservation) Wait(ctx context.Context) (umpire.EffectResult, error) {
	select {
	case <-ctx.Done():
		return umpire.EffectResult{}, ctx.Err()
	case <-h.done:
		return cloneEffectResult(h.result), h.waitErr
	}
}
func (h *fakeReservation) Cancel(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	h.cancelCount.Add(1)
	if h.cancel != nil {
		return h.cancel(ctx)
	}
	return nil
}
func (h *fakeReservation) Drain(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-h.done:
		return nil
	}
}
func (h *fakeReservation) finish() { h.finishOnce.Do(func() { close(h.done) }) }

type fixture struct {
	ledger   *Ledger
	origin   umpire.Coordinate
	plan     umpire.ReservationCarrierPlan
	binding  WorkflowBinding
	workflow *fakeReservation
	handler  *fakeReservation
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
	origin := umpire.Coordinate{RunID: runID, EntrypointID: "controller", ActivationID: "controller.0", InstructionID: "start-workflow", Attempt: 1}
	plan := umpire.ReservationCarrierPlan{
		EndpointRoleID: "temporal",
		Method:         startWorkflowPath,
		Reservations: []umpire.ReservationTopology{
			{EntrypointID: "workflow", Context: umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW, Count: 1},
			{EntrypointID: "handler", Context: umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER, Count: 1},
		},
		Routes: []umpire.ReservationRoute{{WorkflowEntrypointID: "workflow", WorkflowOrdinal: 0, SourceInstructionID: "start-nexus", HandlerEntrypointID: "handler", HandlerOrdinal: 0}},
	}
	workflow := newFakeReservation(umpire.ReservationIdentity{Origin: origin, EntrypointID: "workflow", Ordinal: 0, ID: sessionID + "-workflow"})
	handler := newFakeReservation(umpire.ReservationIdentity{Origin: origin, EntrypointID: "handler", Ordinal: 0, ID: sessionID + "-handler"})
	retainedHandler, err := ledger.RetainReservation(context.Background(), handler)
	require.NoError(t, err)
	retainedWorkflow, err := ledger.RetainReservation(context.Background(), workflow)
	require.NoError(t, err)
	bundle, err := ledger.CreateBundle(context.Background(), origin, plan, WorkflowBinding{Namespace: "namespace", WorkflowID: "workflow-id", WorkflowType: "workflow-type", TaskQueue: "task-queue"}, []umpire.ReservationHandle{retainedHandler, retainedWorkflow})
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

func cloneEffectResult(result umpire.EffectResult) umpire.EffectResult {
	return umpire.EffectResult{Outcome: proto.CloneOf(result.Outcome), Response: proto.Clone(result.Response)}
}

func requireContextError(t *testing.T, err error) {
	t.Helper()
	require.True(t, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded), err)
}
