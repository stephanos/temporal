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
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire"
	"go.temporal.io/server/tools/umpire/internal/execution"
	"go.temporal.io/server/tools/umpire/temporal/internal/delivery"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestConcurrentRunsRouteReorderedWorkflowAndNexusExactly(t *testing.T) {
	prepared := preparedRuntimeFixture(t, umpirespb.NEXUS_RESPONSE_KIND_SYNCHRONOUS)
	host, definition := runtimeTestHost(t, prepared)
	sessionA, carrierA, requestA := runtimeTestSession(t, host, definition, prepared, "run-a", "workflow-a")
	sessionB, carrierB, requestB := runtimeTestSession(t, host, definition, prepared, "run-b", "workflow-b")

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

	headerB, valueB, err := sessionB.preparedNexusDispatch(workflowB.activation, "start", nil, &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "b"}})
	require.NoError(t, err)
	headerA, valueA, err := sessionA.preparedNexusDispatch(workflowA.activation, "start", nil, &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "a"}})
	require.NoError(t, err)
	nexusB, err := host.admitNexus(t.Context(), "task-queue", delivery.NexusDelivery{Header: headerB, RequestID: "request-b"}, func() {})
	require.NoError(t, err)
	nexusA, err := host.admitNexus(t.Context(), "task-queue", delivery.NexusDelivery{Header: headerA, RequestID: "request-a"}, func() {})
	require.NoError(t, err)
	require.Same(t, sessionA, nexusA.session)
	require.Same(t, sessionB, nexusB.session)
	require.Equal(t, "a", valueA.GetText())
	require.Equal(t, "b", valueB.GetText())
	require.Equal(t, 4, host.routeAssociations)

	_, err = host.admitWorkflow(workflowDelivery(requestA, "crossed-run"))
	require.ErrorIs(t, err, delivery.ErrRouteConflict)
	_, err = host.admitNexus(t.Context(), "task-queue", delivery.NexusDelivery{Header: headerA, RequestID: "crossed-request"}, func() {})
	require.ErrorIs(t, err, delivery.ErrRouteConflict)
	_, err = carrierA.ParentTerminal(t.Context())
	require.NoError(t, err)
	_, err = carrierB.ParentTerminal(t.Context())
	require.NoError(t, err)
}

func TestAdmittedWorkflowUsesImmutableNexusDispatchAfterStop(t *testing.T) {
	prepared := preparedRuntimeFixture(t, umpirespb.NEXUS_RESPONSE_KIND_SYNCHRONOUS)
	host, definition := runtimeTestHost(t, prepared)
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
	header, value, err := session.preparedNexusDispatch(routed.activation, "start", nexus.Header{"user": "value"}, &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "request"}})
	require.NoError(t, err)
	require.Equal(t, "request", value.GetText())
	require.Equal(t, "value", header["user"])
	require.Len(t, canceler.cancellations, 1)
	require.Equal(t, workflowCancellation{workflowID: "workflow", runID: "temporal-run"}, canceler.cancellations[0])
	_, err = host.admitNexus(t.Context(), "task-queue", delivery.NexusDelivery{Header: header, RequestID: "request"}, func() {})
	require.ErrorIs(t, err, delivery.ErrRouteStale)
}

func TestCreateCarrierRejectsForeignPhysicalWorkflowBinding(t *testing.T) {
	prepared := preparedRuntimeFixture(t, umpirespb.NEXUS_RESPONSE_KIND_SYNCHRONOUS)
	tests := map[string]WorkflowBinding{
		"namespace": {Namespace: "foreign", WorkflowID: "workflow", WorkflowType: "workflow-type", TaskQueue: "task-queue"},
		"type":      {Namespace: "namespace", WorkflowID: "workflow", WorkflowType: "foreign", TaskQueue: "task-queue"},
		"queue":     {Namespace: "namespace", WorkflowID: "workflow", WorkflowType: "workflow-type", TaskQueue: "foreign"},
	}
	for name, binding := range tests {
		t.Run(name, func(t *testing.T) {
			host, definition := runtimeTestHost(t, prepared)
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
	catalog, err := umpire.NewCatalog(descriptorClosure(workflowservice.File_temporal_api_workflowservice_v1_service_proto))
	require.NoError(t, err)
	limits := proto.CloneOf(preparedRuntimeFixture(t, umpirespb.NEXUS_RESPONSE_KIND_SYNCHRONOUS).Snapshot().GetLimits())
	limits.MaxRunEvents = 100001
	_, err = New(Options{
		Profile: umpire.ProfileSpec{Identity: "profile", Catalog: catalog, ProgramLimits: limits},
		Client:  &recordingClient{}, Namespace: "namespace", WorkerRoleID: "worker",
		TaskQueues: []RoleBinding{{RoleID: "queue", Value: "task-queue"}},
	})
	require.ErrorIs(t, err, ErrInvalid)
}

func TestAsyncCompletionAuthorityIsOpaqueReplaySafeAndLateBounded(t *testing.T) {
	prepared := preparedRuntimeFixture(t, umpirespb.NEXUS_RESPONSE_KIND_ASYNCHRONOUS)
	host, definition := runtimeTestHost(t, prepared)
	bridge := newTestBridge()
	factoryCalls := 0
	var captured CompletionInfo
	options := SessionOptions{
		Bridge: bridge,
		NewCompletionCapability: func(_ context.Context, _ umpire.Coordinate, info CompletionInfo) (umpire.OpaqueCapability, error) {
			factoryCalls++
			captured = cloneCompletionInfo(info)
			return &struct{ run string }{run: "run"}, nil
		},
	}
	session, carrier, request := runtimeTestSessionWithOptions(t, host, definition, prepared, "run", "workflow", options)
	workflowRoute, err := host.admitWorkflow(workflowDelivery(request, "temporal-run"))
	require.NoError(t, err)
	dispatchHeader, dispatchValue, err := session.preparedNexusDispatch(workflowRoute.activation, "start", nil, &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "request"}})
	require.NoError(t, err)
	nexusRoute, err := host.admitNexus(t.Context(), "task-queue", delivery.NexusDelivery{Header: dispatchHeader, RequestID: "request-id"}, func() {})
	require.NoError(t, err)

	result, err := session.executeNexus(t.Context(), nexusRoute.activation, dispatchValue, nexus.StartOperationOptions{CallbackURL: "https://callback.invalid/private", CallbackHeader: nexus.Header{"authorization": "secret"}, RequestID: "request-id"})
	require.NoError(t, err)
	require.Equal(t, "request-id", result.(*nexus.HandlerStartOperationResultAsync).OperationToken)
	require.Equal(t, 1, factoryCalls)
	require.Equal(t, "https://callback.invalid/private", captured.URL)
	require.Equal(t, nexus.Header{"authorization": "secret"}, captured.Header)
	require.True(t, bridge.published)
	require.Equal(t, "capability", bridge.slot)

	replay, err := session.ledger.AdmitNexus(t.Context(), delivery.NexusDelivery{Header: dispatchHeader, RequestID: "request-id"})
	require.NoError(t, err)
	require.True(t, replay.Replay())
	_, err = session.executeNexus(t.Context(), replay, dispatchValue, nexus.StartOperationOptions{CallbackURL: "https://crossed.invalid", RequestID: "request-id"})
	require.NoError(t, err)
	require.Equal(t, 1, factoryCalls)
	require.Equal(t, "https://callback.invalid/private", captured.URL)

	_, err = carrier.ParentTerminal(t.Context())
	require.NoError(t, err)
	session.finishActivation(workflowRoute.activation, &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, nil)
	session.finishActivation(nexusRoute.activation, &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, nil)
	require.NoError(t, session.Close(t.Context()))
	_, err = host.admitNexus(t.Context(), "task-queue", delivery.NexusDelivery{Header: dispatchHeader, RequestID: "request-id"}, func() {})
	require.ErrorIs(t, err, delivery.ErrRouteStale)
	require.LessOrEqual(t, session.diagnostics, int(session.definition.snapshot.GetLimits().GetMaxRunEvents()))
}

func TestAsyncCompletionCannotPublishAfterClose(t *testing.T) {
	prepared := preparedRuntimeFixture(t, umpirespb.NEXUS_RESPONSE_KIND_ASYNCHRONOUS)
	host, definition := runtimeTestHost(t, prepared)
	bridge := newTestBridge()
	factoryStarted := make(chan struct{})
	factoryProceed := make(chan struct{})
	options := SessionOptions{
		Bridge: bridge,
		NewCompletionCapability: func(context.Context, umpire.Coordinate, CompletionInfo) (umpire.OpaqueCapability, error) {
			close(factoryStarted)
			<-factoryProceed
			return &struct{}{}, nil
		},
	}
	session, _, request := runtimeTestSessionWithOptions(t, host, definition, prepared, "run", "workflow", options)
	workflowRoute, err := host.admitWorkflow(workflowDelivery(request, "temporal-run"))
	require.NoError(t, err)
	session.finishActivation(workflowRoute.activation, &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, nil)
	dispatchHeader, dispatchValue, err := session.preparedNexusDispatch(workflowRoute.activation, "start", nil, &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "request"}})
	require.NoError(t, err)
	activationCtx, cancelActivation := context.WithCancel(t.Context())
	nexusRoute, err := host.admitNexus(activationCtx, "task-queue", delivery.NexusDelivery{Header: dispatchHeader, RequestID: "request-id"}, cancelActivation)
	require.NoError(t, err)

	result := make(chan error, 1)
	go func() {
		_, err := session.executeNexus(activationCtx, nexusRoute.activation, dispatchValue, nexus.StartOperationOptions{CallbackURL: "https://callback.invalid/private", RequestID: "request-id"})
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
	prepared := preparedRuntimeFixture(t, umpirespb.NEXUS_RESPONSE_KIND_ASYNCHRONOUS)
	host, definition := runtimeTestHost(t, prepared)
	options := SessionOptions{
		Bridge: newTestBridge(),
		NewCompletionCapability: func(context.Context, umpire.Coordinate, CompletionInfo) (umpire.OpaqueCapability, error) {
			panic("fault")
		},
	}
	session, _, request := runtimeTestSessionWithOptions(t, host, definition, prepared, "run", "workflow", options)
	workflowRoute, err := host.admitWorkflow(workflowDelivery(request, "temporal-run"))
	require.NoError(t, err)
	dispatchHeader, dispatchValue, err := session.preparedNexusDispatch(workflowRoute.activation, "start", nil, &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "request"}})
	require.NoError(t, err)
	nexusRoute, err := host.admitNexus(t.Context(), "task-queue", delivery.NexusDelivery{Header: dispatchHeader, RequestID: "request-id"}, func() {})
	require.NoError(t, err)

	_, err = session.executeNexus(t.Context(), nexusRoute.activation, dispatchValue, nexus.StartOperationOptions{RequestID: "request-id"})
	require.EqualError(t, err, "nexus handler activation panicked")
	replay, err := session.ledger.AdmitNexus(t.Context(), delivery.NexusDelivery{Header: dispatchHeader, RequestID: "request-id"})
	require.NoError(t, err)
	_, err = session.executeNexus(t.Context(), replay, dispatchValue, nexus.StartOperationOptions{RequestID: "request-id"})
	require.EqualError(t, err, "nexus handler activation panicked")
}

func TestStopRejectsDelayedAndUnreservedDelivery(t *testing.T) {
	prepared := preparedRuntimeFixture(t, umpirespb.NEXUS_RESPONSE_KIND_SYNCHRONOUS)
	host, definition := runtimeTestHost(t, prepared)
	session, _, request := runtimeTestSession(t, host, definition, prepared, "run", "workflow")
	require.NoError(t, session.Close(t.Context()))
	_, err := host.admitWorkflow(workflowDelivery(request, "temporal-run"))
	require.ErrorIs(t, err, delivery.ErrRouteStale)
	_, err = session.Reserve(t.Context(), umpire.ReservationRequest{Origin: umpire.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller", InstructionID: "call", Attempt: 1}, EntrypointID: "workflow", Count: 1})
	require.ErrorIs(t, err, ErrClosed)
	foreign := &commonpb.Header{Fields: map[string]*commonpb.Payload{"foreign": {Data: []byte("route")}}}
	_, err = host.admitWorkflow(delivery.WorkflowDelivery{Header: foreign, Namespace: "namespace", WorkflowID: "workflow", WorkflowType: "workflow-type", TaskQueue: "task-queue", TemporalRunID: "temporal-run"})
	require.Error(t, err)
}

func runtimeTestHost(t *testing.T, prepared *execution.PreparedProgram) (*Host, programDefinition) {
	t.Helper()
	host := &Host{
		mu: newContextMutex(), sessions: make(map[string]*Session),
		options: hostOptions{namespace: "namespace", workerRoleID: "worker", taskQueues: map[string]string{"queue": "task-queue"}, endpoints: map[string]string{"endpoint": "endpoint"}, maximum: 16, diagnostics: 16, requestBytes: 64 << 10, now: time.Now},
	}
	definition, err := host.prepareDefinitionPlans(prepared.Snapshot(), prepared.Entrypoints())
	require.NoError(t, err)
	return host, definition
}

func runtimeReservedSession(t *testing.T, host *Host, definition programDefinition, runID string) (*Session, umpire.Coordinate, []umpire.ReservationHandle) {
	t.Helper()
	session, err := newSession(host, runID, "session-"+runID, definition, SessionOptions{Bridge: newTestBridge()})
	require.NoError(t, err)
	require.NoError(t, host.mu.lock(t.Context()))
	host.sessions[runID] = session
	host.mu.unlock()
	origin := umpire.Coordinate{RunID: runID, EntrypointID: "controller", ActivationID: "controller-1", InstructionID: "call", Attempt: 1}
	var handles []umpire.ReservationHandle
	for _, entrypoint := range []string{"workflow", "handler"} {
		reserved, err := session.Reserve(t.Context(), umpire.ReservationRequest{Origin: origin, EntrypointID: entrypoint, Count: 1})
		require.NoError(t, err)
		handles = append(handles, reserved...)
	}
	return session, origin, handles
}

func runtimeTestSession(t *testing.T, host *Host, definition programDefinition, prepared *execution.PreparedProgram, runID, workflowID string) (*Session, *Carrier, *workflowservice.StartWorkflowExecutionRequest) {
	t.Helper()
	return runtimeTestSessionWithOptions(t, host, definition, prepared, runID, workflowID, SessionOptions{Bridge: newTestBridge()})
}

func runtimeTestSessionWithOptions(t *testing.T, host *Host, definition programDefinition, prepared *execution.PreparedProgram, runID, workflowID string, options SessionOptions) (*Session, *Carrier, *workflowservice.StartWorkflowExecutionRequest) {
	t.Helper()
	return runtimeTestSessionWithBinding(t, host, definition, prepared, runID, "temporal-"+runID, WorkflowBinding{Namespace: "namespace", WorkflowID: workflowID, WorkflowType: "workflow-type", TaskQueue: "task-queue"}, options)
}

func runtimeTestSessionWithBinding(t *testing.T, host *Host, definition programDefinition, prepared *execution.PreparedProgram, runID, temporalRunID string, binding WorkflowBinding, options SessionOptions) (*Session, *Carrier, *workflowservice.StartWorkflowExecutionRequest) {
	t.Helper()
	return runtimeTestSessionWithDisposition(t, host, definition, prepared, runID, temporalRunID, binding, options, delivery.TriggerSucceeded)
}

func runtimeTestSessionWithDisposition(t *testing.T, host *Host, definition programDefinition, prepared *execution.PreparedProgram, runID, temporalRunID string, binding WorkflowBinding, options SessionOptions, disposition delivery.TriggerDisposition) (*Session, *Carrier, *workflowservice.StartWorkflowExecutionRequest) {
	t.Helper()
	session, err := newSession(host, runID, "session-"+runID, definition, options)
	require.NoError(t, err)
	require.NoError(t, host.mu.lock(t.Context()))
	host.sessions[runID] = session
	host.mu.unlock()
	origin := umpire.Coordinate{RunID: runID, EntrypointID: "controller", ActivationID: "controller-1", InstructionID: "call", Attempt: 1}
	var handles []umpire.ReservationHandle
	for _, entrypoint := range []string{"workflow", "handler"} {
		reserved, err := session.Reserve(t.Context(), umpire.ReservationRequest{Origin: origin, EntrypointID: entrypoint, Count: 1})
		require.NoError(t, err)
		handles = append(handles, reserved...)
	}
	plan, exists := prepared.ReservationCarrier("controller", "call")
	require.True(t, exists)
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
	coordinate umpire.Coordinate
	slot       string
	capability umpire.OpaqueCapability
}

func newTestBridge() *testBridge { return &testBridge{} }

func (b *testBridge) Publish(_ context.Context, coordinate umpire.Coordinate, slot string, capability umpire.OpaqueCapability) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.published {
		return errors.New("conflicting publication")
	}
	b.published, b.coordinate, b.slot, b.capability = true, coordinate, slot, capability
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

func (b *testBridge) Consume(ctx context.Context, slot string) (umpire.OpaqueCapability, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !b.published || b.slot != slot {
		return nil, errors.New("not ready")
	}
	capability := b.capability
	b.capability = nil
	return capability, nil
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
