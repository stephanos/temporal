package temporal

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport/facadetest"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestCompositeSessionKeepsControllerAndWorkerAuthoritySeparate(t *testing.T) {
	var closes []string
	closing := func(name string) func(context.Context) error {
		return func(context.Context) error {
			closes = append(closes, name)
			return nil
		}
	}
	controller := &testsupport.Session{HandleBridge: &recordingBridge{}, OnInvokeRPC: completedRPC, OnQuarantine: acceptQuarantine, OnClose: closing("controller")}
	workers := &testsupport.Session{OnReserve: func(context.Context, testpilot.ReservationRequest) ([]testpilot.ReservationHandle, error) {
		return []testpilot.ReservationHandle{testsupport.NewReservation(testpilot.ReservationIdentity{})}, nil
	}, OnQuarantine: acceptQuarantine, OnClose: closing("worker")}
	session := newPreparedCompositeSession(controller, workers, preparedConformanceProgram(t))
	origin := testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "call", Attempt: 1}

	reservations, err := session.Reserve(t.Context(), testpilot.ReservationRequest{Origin: origin, EntrypointID: "workflow", Count: 1})
	require.NoError(t, err)
	require.Len(t, reservations, 1)
	_, err = session.InvokeRPC(t.Context(), origin, "endpoint", nil, nil)
	require.NoError(t, err)
	bridge, err := session.Bridge(t.Context())
	require.NoError(t, err)
	require.Same(t, controller.HandleBridge, bridge)
	require.NoError(t, session.Quarantine(t.Context(), reservations[0]))
	require.NoError(t, session.Close(t.Context()))
	require.Equal(t, 1, workers.Calls("Reserve"))
	require.Equal(t, 1, workers.Calls("Quarantine"))
	require.Zero(t, controller.Calls("Reserve"))
	require.Zero(t, controller.Calls("Quarantine"))
	require.Equal(t, 1, controller.Calls("InvokeRPC"))
	require.Equal(t, []string{"worker", "controller"}, closes)
}

// A fault is worker authority, so the composite hands it to the worker Session; a Program with no
// worker use has no worker Session and therefore no queue to stop.
func TestCompositeSessionRoutesFaultsToTheWorker(t *testing.T) {
	workers := &testsupport.Session{OnInjectFault: func(context.Context, testpilot.Coordinate, string, testpilotspb.FaultKind) (testpilot.EffectHandle, error) {
		return testsupport.Completed(testpilot.EffectResult{}), nil
	}}
	program := preparedConformanceProgram(t)
	session := newPreparedCompositeSession(&testsupport.Session{HandleBridge: &recordingBridge{}}, workers, program)
	origin := testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "stop", Attempt: 1}
	_, err := session.InjectFault(t.Context(), origin, "queue", testpilotspb.FAULT_KIND_WORKER_STOP)
	require.NoError(t, err)
	require.Equal(t, 1, workers.Calls("InjectFault"))

	workerless := newPreparedCompositeSession(&testsupport.Session{HandleBridge: &recordingBridge{}}, nil, program)
	_, err = workerless.InjectFault(t.Context(), origin, "queue", testpilotspb.FAULT_KIND_WORKER_STOP)
	require.ErrorIs(t, err, ErrInvalid)
}

func completedRPC(context.Context, testpilot.Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (testpilot.EffectHandle, error) {
	return testsupport.Completed(testpilot.EffectResult{}), nil
}

func acceptQuarantine(context.Context, testpilot.EffectHandle) error { return nil }

// preparedConformanceProgram is the satisfied conformance Case's Program as preparation hands it to
// a Driver. It declares no reservation carrier at the coordinates these tests invoke from.
func preparedConformanceProgram(t *testing.T) testpilot.PreparedProgram {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("..", "testdata", "case-runtime-conformance", "satisfied", "case.json"))
	require.NoError(t, err)
	source, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)
	catalog, err := NewWorkflowServiceCatalog()
	require.NoError(t, err)
	profile, err := DeriveProfile(source, catalog, Environment{Identity: "composite"})
	require.NoError(t, err)
	prepared, err := testpilot.Prepare(source, profile)
	require.NoError(t, err)
	return facadetest.Capture(t, prepared)
}

func TestWorkerQuarantineCompletionFollowsRawHandle(t *testing.T) {
	raw := &testsupport.Effect{WaitErr: errors.New("terminal worker failure")}
	completed := make(chan struct{})
	require.NoError(t, quarantineWorkerHandle(t.Context(), raw, func() { close(completed) }))
	select {
	case <-completed:
		t.Fatal("worker quarantine released before the raw handle completed")
	default:
	}
	raw.Complete()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	select {
	case <-completed:
	case <-ctx.Done():
		require.NoError(t, ctx.Err())
	}
	canceled, cancelCanceled := context.WithCancel(t.Context())
	cancelCanceled()
	require.ErrorIs(t, quarantineWorkerHandle(canceled, raw, func() {}), context.Canceled)
	require.ErrorIs(t, quarantineWorkerHandle(t.Context(), nil, func() {}), ErrInvalid)
}

// A carried StartWorkflow request names its whole binding; a request that omits a binding field or
// leaves one empty is the composite Driver's invalid input.
func TestCarrierBindingRejectsIncompleteStartRequests(t *testing.T) {
	complete := func() *workflowservice.StartWorkflowExecutionRequest {
		return &workflowservice.StartWorkflowExecutionRequest{Namespace: "namespace", WorkflowId: "workflow-id", WorkflowType: &commonpb.WorkflowType{Name: "workflow-type"}, TaskQueue: &taskqueuepb.TaskQueue{Name: "task-queue"}}
	}
	binding, err := carrierBinding(complete())
	require.NoError(t, err)
	require.Equal(t, delivery.WorkflowBinding{Namespace: "namespace", WorkflowID: "workflow-id", WorkflowType: "workflow-type", TaskQueue: "task-queue"}, binding)

	for name, mutate := range map[string]func(*workflowservice.StartWorkflowExecutionRequest){
		"namespace":             func(request *workflowservice.StartWorkflowExecutionRequest) { request.Namespace = "" },
		"workflow id":           func(request *workflowservice.StartWorkflowExecutionRequest) { request.WorkflowId = "" },
		"workflow type name":    func(request *workflowservice.StartWorkflowExecutionRequest) { request.WorkflowType.Name = "" },
		"task queue name":       func(request *workflowservice.StartWorkflowExecutionRequest) { request.TaskQueue.Name = "" },
		"missing workflow type": func(request *workflowservice.StartWorkflowExecutionRequest) { request.WorkflowType = nil },
	} {
		t.Run(name, func(t *testing.T) {
			request := complete()
			mutate(request)
			_, err := carrierBinding(request)
			require.ErrorIs(t, err, ErrInvalid)
		})
	}
}

func TestCarrierEffectFinalizesWithFreshBoundAndRetries(t *testing.T) {
	terminal := &recordingTerminalCarrier{failures: 1, admissible: true}
	effect := &carrierEffect{
		EffectHandle:   &testsupport.Effect{},
		carrier:        terminal,
		cleanupTimeout: time.Second,
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := effect.Wait(canceled)
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorContains(t, err, "injected terminal failure")
	require.Equal(t, []delivery.TriggerStatus{delivery.TriggerUncertain}, terminal.dispositions)
	require.Equal(t, []error{nil}, terminal.contextErrors)
	require.True(t, terminal.admissible)

	require.NoError(t, effect.Cancel(t.Context()))
	require.Equal(t, []delivery.TriggerStatus{delivery.TriggerUncertain, delivery.TriggerUncertain}, terminal.dispositions)
	require.Equal(t, []error{nil, nil}, terminal.contextErrors)
	require.False(t, terminal.admissible)
}

func TestCarrierEffectDrainFinalizesCompletedResult(t *testing.T) {
	terminal := &recordingTerminalCarrier{admissible: true}
	effect := &carrierEffect{
		EffectHandle: testsupport.Completed(testpilot.EffectResult{
			Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE},
		}),
		carrier: terminal, cleanupTimeout: time.Second,
	}
	require.NoError(t, effect.Drain(t.Context()))
	require.Equal(t, []delivery.TriggerStatus{delivery.TriggerNonSuccess}, terminal.dispositions)
	require.False(t, terminal.admissible)
}

type recordingTerminalCarrier struct {
	failures       int
	admissible     bool
	dispositions   []delivery.TriggerStatus
	contextErrors  []error
	pinnedResponse *workflowservice.StartWorkflowExecutionResponse
}

func (c *recordingTerminalCarrier) PinStartResponse(ctx context.Context, response *workflowservice.StartWorkflowExecutionResponse) error {
	c.contextErrors = append(c.contextErrors, ctx.Err())
	c.pinnedResponse = proto.CloneOf(response)
	return nil
}
func (c *recordingTerminalCarrier) TriggerTerminal(ctx context.Context, disposition delivery.TriggerStatus) (int, error) {
	c.contextErrors = append(c.contextErrors, ctx.Err())
	c.dispositions = append(c.dispositions, disposition)
	if c.failures > 0 {
		c.failures--
		return 0, errors.New("injected terminal failure")
	}
	c.admissible = false
	return 0, nil
}

type recordingBridge struct{}

func (*recordingBridge) Publish(context.Context, testpilot.Coordinate, string, testpilot.OpaqueHandle) error {
	return nil
}
func (*recordingBridge) Await(context.Context, string) error { return nil }
func (*recordingBridge) Consume(context.Context, string) (testpilot.OpaqueHandle, error) {
	return struct{}{}, nil
}
