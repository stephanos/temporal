package worker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/primitive"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestConcurrentRunsRouteReorderedWorkflowAndNexusExactly(t *testing.T) {
	prepared := preparedRuntimeFixture(t, replySynchronous)
	host, definition := runtimeTestDriver(t, prepared)
	sessionA, _, requestA := runtimeTestSession(t, host, definition, prepared, "run-a", "workflow-a")
	sessionB, _, requestB := runtimeTestSession(t, host, definition, prepared, "run-b", "workflow-b")

	workflowB, err := host.admitWorkflow(workflowDelivery(requestB, "temporal-run-b"))
	require.NoError(t, err)
	workflowA, err := host.admitWorkflow(workflowDelivery(requestA, "temporal-run-a"))
	require.NoError(t, err)
	require.Same(t, sessionA, workflowA.session)
	require.Same(t, sessionB, workflowB.session)
	require.NotEqual(t, workflowA.activation.Coordinate(), workflowB.activation.Coordinate())

	replayA, err := host.admitWorkflow(workflowDelivery(requestA, "temporal-run-a"))
	require.NoError(t, err)
	require.True(t, replayA.replay)
	require.Equal(t, workflowA.activation.Coordinate(), replayA.activation.Coordinate())

	headerB, err := sessionB.preparedNexusHeader(workflowB.activation, "start")
	require.NoError(t, err)
	headerA, err := sessionA.preparedNexusHeader(workflowA.activation, "start")
	require.NoError(t, err)
	nexusB, err := host.admitNexus(t.Context(), "task-queue", delivery.NexusDelivery{Header: headerB, RequestID: "request-b"}, func() {})
	require.NoError(t, err)
	nexusA, err := host.admitNexus(t.Context(), "task-queue", delivery.NexusDelivery{Header: headerA, RequestID: "request-a"}, func() {})
	require.NoError(t, err)
	require.Same(t, sessionA, nexusA.session)
	require.Same(t, sessionB, nexusB.session)
	require.Equal(t, 4, host.routeAssociations)

	_, err = host.admitWorkflow(workflowDelivery(requestA, "crossed-run"))
	require.ErrorIs(t, err, delivery.ErrRouteConflict)
	_, err = host.admitNexus(t.Context(), "task-queue", delivery.NexusDelivery{Header: headerA, RequestID: "crossed-request"}, func() {})
	require.ErrorIs(t, err, delivery.ErrRouteConflict)
	_, err = sessionA.parentTerminal(t.Context(), workflowA.activation)
	require.NoError(t, err)
	_, err = sessionB.parentTerminal(t.Context(), workflowB.activation)
	require.NoError(t, err)
}

func TestAdmittedWorkflowUsesImmutableNexusDispatchAfterStop(t *testing.T) {
	prepared := preparedRuntimeFixture(t, replySynchronous)
	host, definition := runtimeTestDriver(t, prepared)
	canceler := &recordingClient{}
	host.options.client = canceler
	session, _, request := runtimeTestSession(t, host, definition, prepared, "run", "workflow")
	routed, err := host.admitWorkflow(workflowDelivery(request, "temporal-run"))
	require.NoError(t, err)

	require.NoError(t, session.Close(t.Context()))
	replay, err := host.admitWorkflow(workflowDelivery(request, "temporal-run"))
	require.NoError(t, err)
	require.True(t, replay.replay)
	require.Equal(t, routed.activation.Coordinate(), replay.activation.Coordinate())
	header, err := session.preparedNexusHeader(routed.activation, "start", nexus.Header{"user": "value"})
	require.NoError(t, err)
	require.Equal(t, "value", header["user"])
	require.Len(t, canceler.cancellations, 1)
	require.Equal(t, workflowCancellation{workflowID: "workflow", runID: "temporal-run"}, canceler.cancellations[0])
	_, err = host.admitNexus(t.Context(), "task-queue", delivery.NexusDelivery{Header: header, RequestID: "request"}, func() {})
	require.ErrorIs(t, err, delivery.ErrRouteStale)
}

// Replays of one admission race its completion: every replay shares the Session's admission, even
// after the ledger has released the parent, and the reservation completes once.
func TestConcurrentReplayAndCompletionOfOneAdmission(t *testing.T) {
	prepared := preparedRuntimeFixture(t, replySynchronous)
	host, definition := runtimeTestDriver(t, prepared)
	session, _, request := runtimeTestSession(t, host, definition, prepared, "run", "workflow")
	routed, err := host.admitWorkflow(workflowDelivery(request, "temporal-run"))
	require.NoError(t, err)

	const racers = 8
	failed := errors.New("activation failed")
	replays := make(chan routedWorkflow, racers)
	errs := make(chan error, 2*racers)
	var wg sync.WaitGroup
	for i := range racers {
		wg.Go(func() {
			replay, err := host.admitWorkflow(workflowDelivery(request, "temporal-run"))
			errs <- err
			replays <- replay
		})
		wg.Go(func() {
			var executionErr error
			if i%2 == 1 {
				executionErr = failed
			}
			if err := session.completeWorkflow(routed.admission, routed.activation, executionErr); !errors.Is(err, executionErr) {
				errs <- err
			}
		})
	}
	wg.Wait()
	close(errs)
	close(replays)
	for err := range errs {
		require.NoError(t, err)
	}
	for replay := range replays {
		require.True(t, replay.replay)
		require.Same(t, routed.admission, replay.admission)
		require.Equal(t, routed.activation.Coordinate(), replay.activation.Coordinate())
	}
	require.True(t, routed.admission.terminal)
	raw, err := session.rawReservation(routed.activation.Reservation().ID)
	require.NoError(t, err)
	result, err := raw.Wait(t.Context())
	if err != nil {
		require.ErrorIs(t, err, failed)
		require.Equal(t, sdkFailureOutcome(failed).GetStatus(), result.Outcome.GetStatus())
	} else {
		require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, result.Outcome.GetStatus())
	}
}

func TestCreateCarrierRejectsForeignPhysicalWorkflowBinding(t *testing.T) {
	prepared := preparedRuntimeFixture(t, replySynchronous)
	tests := map[string]delivery.WorkflowBinding{
		"namespace": {Namespace: "foreign", WorkflowID: "workflow", WorkflowType: "workflow-type", TaskQueue: "task-queue"},
		"type":      {Namespace: "namespace", WorkflowID: "workflow", WorkflowType: "foreign", TaskQueue: "task-queue"},
		"queue":     {Namespace: "namespace", WorkflowID: "workflow", WorkflowType: "workflow-type", TaskQueue: "foreign"},
	}
	for name, binding := range tests {
		t.Run(name, func(t *testing.T) {
			host, definition := runtimeTestDriver(t, prepared)
			session, origin, handles := runtimeReservedSession(t, host, definition, "run")
			plan, exists := prepared.ReservationCarrier("controller", "call")
			require.True(t, exists)
			_, err := session.CreateCarrier(t.Context(), origin, plan, binding, handles)
			require.ErrorIs(t, err, ErrInvalid)
			require.Empty(t, session.carriers)
			require.Zero(t, host.routeAssociations)
		})
	}
}

func TestNewRejectsProfileLimitsBeforeRetainedStateAllocation(t *testing.T) {
	catalog, err := testpilot.NewCatalog(testsupport.DescriptorClosure(workflowservice.File_temporal_api_workflowservice_v1_service_proto))
	require.NoError(t, err)
	limits := proto.CloneOf(preparedRuntimeFixture(t, replySynchronous).Limits())
	limits.MaxRunEvents = 100001
	_, err = New(Options{
		Profile: testpilot.ProfileSpec{Identity: "profile", Catalog: catalog, ProgramLimits: limits},
		Client:  &recordingClient{}, WorkerRoleID: "worker",
	})
	require.ErrorIs(t, err, ErrInvalid)
}

func TestAsyncCompletionAuthorityIsOpaqueReplaySafeAndLateBounded(t *testing.T) {
	prepared := preparedRuntimeFixture(t, replyAsynchronous)
	host, definition := runtimeTestDriver(t, prepared)
	bridge := newTestBridge()
	factoryCalls := 0
	var captured testpilot.HandleEffect
	options := SessionOptions{
		Bridge: bridge,
		NewHandle: func(_ context.Context, _ testpilot.Coordinate, effect testpilot.HandleEffect) (testpilot.OpaqueHandle, error) {
			factoryCalls++
			captured = effect
			return &struct{ run string }{run: "run"}, nil
		},
	}
	session, _, request := runtimeTestSessionWithOptions(t, host, definition, prepared, "run", "workflow", options)
	workflowRoute, err := host.admitWorkflow(workflowDelivery(request, "temporal-run"))
	require.NoError(t, err)
	dispatchHeader, err := session.preparedNexusHeader(workflowRoute.activation, "start")
	require.NoError(t, err)
	nexusRoute, err := host.admitNexus(t.Context(), "task-queue", delivery.NexusDelivery{Header: dispatchHeader, RequestID: "request-id"}, func() {})
	require.NoError(t, err)

	result, err := session.executeNexus(t.Context(), nexusRoute.activation, nil, nexus.StartOperationOptions{CallbackURL: "https://callback.invalid/private", CallbackHeader: nexus.Header{"authorization": "secret"}, RequestID: "request-id"})
	require.NoError(t, err)
	require.Equal(t, "request-id", result.(*nexus.HandlerStartOperationResultAsync).OperationToken)
	require.Equal(t, 1, factoryCalls)
	require.NotNil(t, captured)
	require.True(t, bridge.published)
	require.Equal(t, "handle", bridge.slot)

	replay, err := session.ledger.AdmitNexus(t.Context(), delivery.NexusDelivery{Header: dispatchHeader, RequestID: "request-id"})
	require.NoError(t, err)
	require.True(t, replay.Replay())
	_, err = session.executeNexus(t.Context(), replay, nil, nexus.StartOperationOptions{CallbackURL: "https://crossed.invalid", RequestID: "request-id"})
	require.NoError(t, err)
	require.Equal(t, 1, factoryCalls)

	_, err = session.parentTerminal(t.Context(), workflowRoute.activation)
	require.NoError(t, err)
	session.finishActivation(workflowRoute.activation, &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, nil)
	session.finishActivation(nexusRoute.activation, &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, nil)
	require.NoError(t, session.Close(t.Context()))
	_, err = host.admitNexus(t.Context(), "task-queue", delivery.NexusDelivery{Header: dispatchHeader, RequestID: "request-id"}, func() {})
	require.ErrorIs(t, err, delivery.ErrRouteStale)
	require.LessOrEqual(t, session.diagnostics, int(session.definition.limits.GetMaxRunEvents()))
}

func TestAsyncCompletionCannotPublishAfterClose(t *testing.T) {
	prepared := preparedRuntimeFixture(t, replyAsynchronous)
	host, definition := runtimeTestDriver(t, prepared)
	bridge := newTestBridge()
	factoryStarted := make(chan struct{})
	factoryProceed := make(chan struct{})
	options := SessionOptions{
		Bridge: bridge,
		NewHandle: func(context.Context, testpilot.Coordinate, testpilot.HandleEffect) (testpilot.OpaqueHandle, error) {
			close(factoryStarted)
			<-factoryProceed
			return &struct{}{}, nil
		},
	}
	session, _, request := runtimeTestSessionWithOptions(t, host, definition, prepared, "run", "workflow", options)
	workflowRoute, err := host.admitWorkflow(workflowDelivery(request, "temporal-run"))
	require.NoError(t, err)
	session.finishActivation(workflowRoute.activation, &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, nil)
	dispatchHeader, err := session.preparedNexusHeader(workflowRoute.activation, "start")
	require.NoError(t, err)
	activationCtx, cancelActivation := context.WithCancel(t.Context())
	nexusRoute, err := host.admitNexus(activationCtx, "task-queue", delivery.NexusDelivery{Header: dispatchHeader, RequestID: "request-id"}, cancelActivation)
	require.NoError(t, err)

	result := make(chan error, 1)
	go func() {
		_, err := session.executeNexus(activationCtx, nexusRoute.activation, nil, nexus.StartOperationOptions{CallbackURL: "https://callback.invalid/private", RequestID: "request-id"})
		result <- err
	}()
	<-factoryStarted
	require.NoError(t, session.Close(t.Context()))
	close(factoryProceed)
	require.ErrorIs(t, <-result, ErrClosed)
	require.False(t, bridge.published)
	require.Equal(t, 1, session.diagnostics)
}

func TestNexusPanicCompletesReplayWaiters(t *testing.T) {
	prepared := preparedRuntimeFixture(t, replyAsynchronous)
	host, definition := runtimeTestDriver(t, prepared)
	options := SessionOptions{
		Bridge: newTestBridge(),
		NewHandle: func(context.Context, testpilot.Coordinate, testpilot.HandleEffect) (testpilot.OpaqueHandle, error) {
			panic("fault")
		},
	}
	session, _, request := runtimeTestSessionWithOptions(t, host, definition, prepared, "run", "workflow", options)
	workflowRoute, err := host.admitWorkflow(workflowDelivery(request, "temporal-run"))
	require.NoError(t, err)
	dispatchHeader, err := session.preparedNexusHeader(workflowRoute.activation, "start")
	require.NoError(t, err)
	nexusRoute, err := host.admitNexus(t.Context(), "task-queue", delivery.NexusDelivery{Header: dispatchHeader, RequestID: "request-id"}, func() {})
	require.NoError(t, err)

	_, err = session.executeNexus(t.Context(), nexusRoute.activation, nil, nexus.StartOperationOptions{CallbackURL: "https://callback.invalid/private", RequestID: "request-id"})
	require.EqualError(t, err, "nexus handler activation panicked")
	replay, err := session.ledger.AdmitNexus(t.Context(), delivery.NexusDelivery{Header: dispatchHeader, RequestID: "request-id"})
	require.NoError(t, err)
	_, err = session.executeNexus(t.Context(), replay, nil, nexus.StartOperationOptions{CallbackURL: "https://callback.invalid/private", RequestID: "request-id"})
	require.EqualError(t, err, "nexus handler activation panicked")
}

func TestNexusCanceledEvaluationPreventsResponse(t *testing.T) {
	prepared := preparedRuntimeFixture(t, replySynchronous)
	host, definition := runtimeTestDriver(t, prepared)
	session, _, request := runtimeTestSession(t, host, definition, prepared, "run", "workflow")
	workflowRoute, err := host.admitWorkflow(workflowDelivery(request, "temporal-run"))
	require.NoError(t, err)
	header, err := session.preparedNexusHeader(workflowRoute.activation, "start")
	require.NoError(t, err)
	routed, err := host.admitNexus(t.Context(), "task-queue", delivery.NexusDelivery{Header: header, RequestID: "request-id"}, func() {})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	result, err := session.interpretNexus(ctx, routed.activation, nexus.StartOperationOptions{}, &nexusResult{})
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, result.kind)
	require.Nil(t, result.raw)
	require.Empty(t, result.token)
}

func TestStopRejectsDelayedAndUnreservedDelivery(t *testing.T) {
	prepared := preparedRuntimeFixture(t, replySynchronous)
	host, definition := runtimeTestDriver(t, prepared)
	session, _, request := runtimeTestSession(t, host, definition, prepared, "run", "workflow")
	require.NoError(t, session.Close(t.Context()))
	_, err := host.admitWorkflow(workflowDelivery(request, "temporal-run"))
	require.ErrorIs(t, err, delivery.ErrRouteStale)
	_, err = session.Reserve(t.Context(), testpilot.ReservationRequest{Origin: testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller", InstructionID: "call", Attempt: 1}, EntrypointID: "workflow", Count: 1})
	require.ErrorIs(t, err, ErrClosed)
	foreign := &commonpb.Header{Fields: map[string]*commonpb.Payload{"foreign": {Data: []byte("route")}}}
	_, err = host.admitWorkflow(delivery.WorkflowDelivery{Header: foreign, Namespace: "namespace", WorkflowID: "workflow", WorkflowType: "workflow-type", TaskQueue: "task-queue", TemporalRunID: "temporal-run"})
	require.Error(t, err)
}

func runtimeTestDriver(t *testing.T, prepared testpilot.PreparedProgram) (*Driver, programDefinition) {
	t.Helper()
	completion, err := newCompletionTransport(nil, "", prepared.Limits())
	require.NoError(t, err)
	host := &Driver{
		mu: primitive.NewMutex(), sessions: make(map[string]*Session),
		options: hostOptions{
			profile: testpilot.ProfileSpec{Roles: []testpilot.RolePolicy{
				{ID: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, Methods: []string{primitive.StartWorkflowPath}},
				{ID: "worker", Kind: testpilotspb.ROLE_KIND_WORKER},
				{ID: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE},
				{ID: "nexus-endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT},
			}},
			workerRoleID: "worker", maximum: 16, diagnostics: 16, requestBytes: 64 << 10, now: time.Now, completion: completion,
		},
	}
	definition, err := host.prepareDefinitionResources(prepared.Snapshot(), prepared.Limits(), prepared.Entrypoints(), prepared.Roles(), true)
	require.NoError(t, err)
	return host, definition
}

func runtimeReservedSession(t *testing.T, host *Driver, definition programDefinition, runID string) (*Session, testpilot.Coordinate, []testpilot.ReservationHandle) {
	t.Helper()
	session, err := newSession(host, runID, "session-"+runID, definition, SessionOptions{Bridge: newTestBridge()})
	require.NoError(t, err)
	require.NoError(t, host.mu.LockContext(t.Context(), ErrInvalid))
	host.sessions[runID] = session
	host.mu.Unlock()
	origin := testpilot.Coordinate{RunID: runID, EntrypointID: "controller", ActivationID: "controller-1", InstructionID: "call", Attempt: 1}
	var handles []testpilot.ReservationHandle
	for _, entrypoint := range []string{"workflow", "handler"} {
		reserved, err := session.Reserve(t.Context(), testpilot.ReservationRequest{Origin: origin, EntrypointID: entrypoint, Count: 1})
		require.NoError(t, err)
		handles = append(handles, reserved...)
	}
	return session, origin, handles
}

func runtimeTestSession(t *testing.T, host *Driver, definition programDefinition, prepared testpilot.PreparedProgram, runID, workflowID string) (*Session, *Carrier, *workflowservice.StartWorkflowExecutionRequest) {
	t.Helper()
	return runtimeTestSessionWithOptions(t, host, definition, prepared, runID, workflowID, SessionOptions{Bridge: newTestBridge()})
}

func runtimeTestSessionWithOptions(t *testing.T, host *Driver, definition programDefinition, prepared testpilot.PreparedProgram, runID, workflowID string, options SessionOptions) (*Session, *Carrier, *workflowservice.StartWorkflowExecutionRequest) {
	t.Helper()
	return runtimeTestSessionWithBinding(t, host, definition, prepared, runID, "temporal-"+runID, delivery.WorkflowBinding{Namespace: "namespace", WorkflowID: workflowID, WorkflowType: "workflow-type", TaskQueue: "task-queue"}, options)
}

func runtimeTestSessionWithBinding(t *testing.T, host *Driver, definition programDefinition, prepared testpilot.PreparedProgram, runID, temporalRunID string, binding delivery.WorkflowBinding, options SessionOptions) (*Session, *Carrier, *workflowservice.StartWorkflowExecutionRequest) {
	t.Helper()
	return runtimeTestSessionWithDisposition(t, host, definition, prepared, runID, temporalRunID, binding, options, delivery.TriggerSucceeded)
}

func runtimeTestSessionWithDisposition(t *testing.T, host *Driver, definition programDefinition, prepared testpilot.PreparedProgram, runID, temporalRunID string, binding delivery.WorkflowBinding, options SessionOptions, disposition delivery.TriggerStatus) (*Session, *Carrier, *workflowservice.StartWorkflowExecutionRequest) {
	t.Helper()
	session, err := newSession(host, runID, "session-"+runID, definition, options)
	require.NoError(t, err)
	require.NoError(t, host.mu.LockContext(t.Context(), ErrInvalid))
	host.sessions[runID] = session
	host.mu.Unlock()
	origin := testpilot.Coordinate{RunID: runID, EntrypointID: "controller", ActivationID: "controller-1", InstructionID: "call", Attempt: 1}
	plan, exists := prepared.ReservationCarrier("controller", "call")
	require.True(t, exists)
	var handles []testpilot.ReservationHandle
	for _, reservation := range plan.Reservations {
		reserved, err := session.Reserve(t.Context(), testpilot.ReservationRequest{Origin: origin, EntrypointID: reservation.EntrypointID, Count: reservation.Count})
		require.NoError(t, err)
		handles = append(handles, reserved...)
	}
	carrier, err := session.CreateCarrier(t.Context(), origin, plan, binding, handles)
	require.NoError(t, err)
	request := &workflowservice.StartWorkflowExecutionRequest{Namespace: binding.Namespace, WorkflowId: binding.WorkflowID, WorkflowType: &commonpb.WorkflowType{Name: binding.WorkflowType}, TaskQueue: &taskqueuepb.TaskQueue{Name: binding.TaskQueue}}
	method := prepared.Entrypoints()[0].Instructions()[0].Method()
	wire, err := proto.Marshal(request)
	require.NoError(t, err)
	dynamicRequest := dynamicpb.NewMessage(method.Input())
	require.NoError(t, proto.Unmarshal(wire, dynamicRequest))
	preparedRequest, err := carrier.PrepareRPC(t.Context(), "endpoint", method, dynamicRequest, 64<<10)
	require.NoError(t, err)
	wire, err = proto.Marshal(preparedRequest)
	require.NoError(t, err)
	require.NoError(t, proto.Unmarshal(wire, request))
	require.NoError(t, carrier.PinStartResponse(t.Context(), &workflowservice.StartWorkflowExecutionResponse{RunId: temporalRunID}))
	_, err = carrier.TriggerTerminal(t.Context(), disposition)
	require.NoError(t, err)
	return session, carrier, request
}

func workflowDelivery(request *workflowservice.StartWorkflowExecutionRequest, temporalRunID string) delivery.WorkflowDelivery {
	return delivery.WorkflowDelivery{Header: request.GetHeader(), Namespace: request.GetNamespace(), WorkflowID: request.GetWorkflowId(), WorkflowType: request.GetWorkflowType().GetName(), TaskQueue: request.GetTaskQueue().GetName(), TemporalRunID: temporalRunID}
}

type testBridge struct {
	mu         sync.Mutex
	published  bool
	coordinate testpilot.Coordinate
	slot       string
	handle     testpilot.OpaqueHandle
}

func newTestBridge() *testBridge { return &testBridge{} }

func (b *testBridge) Publish(_ context.Context, coordinate testpilot.Coordinate, slot string, handle testpilot.OpaqueHandle) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.published {
		return errors.New("conflicting publication")
	}
	b.published, b.coordinate, b.slot, b.handle = true, coordinate, slot, handle
	return nil
}

func (b *testBridge) Await(ctx context.Context, slot string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.published || b.slot != slot {
		return errors.New("not ready")
	}
	return ctx.Err()
}

func (b *testBridge) Consume(ctx context.Context, slot string) (testpilot.OpaqueHandle, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !b.published || b.slot != slot {
		return nil, errors.New("not ready")
	}
	handle := b.handle
	b.handle = nil
	return handle, nil
}

type workflowCancellation struct {
	workflowID string
	runID      string
}

type recordingClient struct {
	client.Client
	cancellations []workflowCancellation
}

func (c *recordingClient) CancelWorkflow(ctx context.Context, workflowID, runID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.cancellations = append(c.cancellations, workflowCancellation{workflowID: workflowID, runID: runID})
	return nil
}
