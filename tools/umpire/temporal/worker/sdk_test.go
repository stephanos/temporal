package worker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/testsuite"
	sdkworker "go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire"
	"go.temporal.io/server/tools/umpire/temporal/internal/delivery"
)

func TestSDKWorkflowInterpretsStartAwaitAndFinishWithArbitraryArguments(t *testing.T) {
	for _, valueOutcome := range []bool{false, true} {
		t.Run(fmt.Sprintf("value=%t", valueOutcome), func(t *testing.T) {
			prepared := preparedRuntimeFixture(t, umpirespb.NEXUS_RESPONSE_KIND_SYNCHRONOUS, func(program *umpirespb.Program) {
				if valueOutcome {
					program.Entrypoints[1].Nodes[2].Outcome = runtimeValueOutcomeSchema()
				}
			})
			host, definition := runtimeTestHost(t, prepared)
			host.options.namespace = "default-test-namespace"
			host.options.client = &recordingClient{}
			binding := WorkflowBinding{Namespace: "default-test-namespace", WorkflowID: "default-test-workflow-id", WorkflowType: "workflow-type", TaskQueue: "task-queue"}
			session, _, request := runtimeTestSessionWithBinding(t, host, definition, prepared, "run", "default-test-run-id", binding, SessionOptions{Bridge: newTestBridge()})

			var suite testsuite.WorkflowTestSuite
			environment := suite.NewTestWorkflowEnvironment()
			environment.SetWorkerOptions(sdkworker.Options{Interceptors: []interceptor.WorkerInterceptor{&sdkWorkerInterceptor{host: host, queue: "task-queue", registration: definition.registrations[0]}}})
			environment.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: binding.WorkflowID, TaskQueue: binding.TaskQueue})
			environment.SetHeader(request.GetHeader())
			operation := nexus.NewOperationReference[*umpirespb.Value, *umpirespb.Value]("operation")
			environment.OnNexusOperation(
				"service",
				operation,
				&umpirespb.Value{Value: &umpirespb.Value_Text{Text: "request"}},
				mock.Anything,
			).Return(&nexus.HandlerStartOperationResultSync[*umpirespb.Value]{Value: &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "done"}}}, nil)
			environment.RegisterDynamicWorkflow(host.dynamicWorkflow, workflow.DynamicRegisterOptions{})
			environment.ExecuteWorkflow("workflow-type", "untouched", 42, []byte("arguments"))
			require.NoError(t, environment.GetWorkflowError())
			var result umpirespb.Value
			require.NoError(t, environment.GetWorkflowResult(&result))
			require.Equal(t, "done", result.GetText())
			environment.AssertNexusOperationCalled(t, "service", "operation", &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "request"}}, mock.Anything)
			workflowReservation := reservationForEntrypoint(t, session, "workflow")
			workflowResult, err := workflowReservation.Wait(t.Context())
			require.NoError(t, err)
			require.Equal(t, umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, workflowResult.Outcome.GetStatus())
			handlerResult, err := reservationForEntrypoint(t, session, "handler").Wait(t.Context())
			require.NoError(t, err)
			require.Equal(t, umpirespb.INSTRUCTION_OUTCOME_STATUS_CANCELED, handlerResult.Outcome.GetStatus())
			require.Len(t, session.workflowAdmissions, 1)
		})
	}
}

func TestSDKWorkflowReplayerCompletesAnUnfinishedAdmission(t *testing.T) {
	prepared := preparedRuntimeFixture(t, umpirespb.NEXUS_RESPONSE_KIND_SYNCHRONOUS)
	host, definition := runtimeTestHost(t, prepared)
	host.options.namespace = "ReplayNamespace"
	host.options.client = &recordingClient{}
	binding := WorkflowBinding{Namespace: "ReplayNamespace", WorkflowID: "replay-workflow", WorkflowType: "workflow-type", TaskQueue: "task-queue"}
	session, _, request := runtimeTestSessionWithBinding(t, host, definition, prepared, "run", "replay-run", binding, SessionOptions{Bridge: newTestBridge()})

	routed, err := host.admitWorkflow(workflowDelivery(request, "replay-run"))
	require.NoError(t, err)
	header, _, err := session.preparedNexusDispatch(routed.activation, "start", nil, &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "request"}})
	require.NoError(t, err)
	history := workflowReplayHistory(t, binding, request.GetHeader(), header)
	replayer, err := sdkworker.NewWorkflowReplayerWithOptions(sdkworker.WorkflowReplayerOptions{Interceptors: []interceptor.WorkerInterceptor{
		&sdkWorkerInterceptor{host: host, queue: binding.TaskQueue, registration: definition.registrations[0]},
	}})
	require.NoError(t, err)
	replayer.RegisterDynamicWorkflow(host.dynamicWorkflow, workflow.DynamicRegisterOptions{})
	options := sdkworker.ReplayWorkflowHistoryOptions{OriginalExecution: workflow.Execution{ID: binding.WorkflowID, RunID: "replay-run"}}

	require.NoError(t, replayer.ReplayWorkflowHistoryWithOptions(nil, &historypb.History{Events: history[:5]}, options))
	require.False(t, routed.admission.terminal)
	require.NoError(t, replayer.ReplayWorkflowHistoryWithOptions(nil, &historypb.History{Events: history}, options))
	require.True(t, routed.admission.terminal)
	workflowResult, err := reservationForEntrypoint(t, session, "workflow").Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, workflowResult.Outcome.GetStatus())
	handlerResult, err := reservationForEntrypoint(t, session, "handler").Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, umpirespb.INSTRUCTION_OUTCOME_STATUS_CANCELED, handlerResult.Outcome.GetStatus())
	require.Len(t, session.workflowAdmissions, 1)
}

func TestSDKDynamicWorkflowRejectsForeignWorkflowTypeBeforeAdmission(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	environment := suite.NewTestWorkflowEnvironment()
	environment.SetWorkerOptions(sdkworker.Options{Interceptors: []interceptor.WorkerInterceptor{&sdkWorkerInterceptor{queue: "task-queue", registration: queueRegistration{queue: "task-queue", workflows: []string{"workflow-type"}}}}})
	environment.SetStartWorkflowOptions(client.StartWorkflowOptions{TaskQueue: "task-queue"})
	environment.RegisterDynamicWorkflow(func(workflow.Context, converter.EncodedValues) error { return nil }, workflow.DynamicRegisterOptions{})
	environment.ExecuteWorkflow("foreign-workflow", "arbitrary", 42)
	require.Error(t, environment.GetWorkflowError())
}

func TestSDKNexusInboundRoutesRedeliveryThroughLedger(t *testing.T) {
	for _, valueOutcome := range []bool{false, true} {
		t.Run(fmt.Sprintf("value=%t", valueOutcome), func(t *testing.T) {
			prepared := preparedRuntimeFixture(t, umpirespb.NEXUS_RESPONSE_KIND_SYNCHRONOUS, func(program *umpirespb.Program) {
				if valueOutcome {
					program.Entrypoints[2].Nodes[0].Outcome = runtimeValueOutcomeSchema()
				}
			})
			host, definition := runtimeTestHost(t, prepared)
			session, _, request := runtimeTestSession(t, host, definition, prepared, "run", "workflow")
			workflowRoute, err := host.admitWorkflow(workflowDelivery(request, "temporal-run"))
			require.NoError(t, err)
			header, value, err := session.preparedNexusDispatch(workflowRoute.activation, "start", nil, &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "request"}})
			require.NoError(t, err)

			operation := &genericNexusOperation{queue: "task-queue", service: "service", operation: "operation"}
			service := nexus.NewService("service")
			require.NoError(t, service.Register(operation))
			registry := nexus.NewServiceRegistry()
			require.NoError(t, registry.Register(service))
			handler, err := registry.NewHandler()
			require.NoError(t, err)
			terminal := &registeredNexusTerminal{handler: handler, service: "service", operation: "operation"}
			inbound := (&sdkWorkerInterceptor{host: host, queue: "task-queue", registration: definition.registrations[0]}).InterceptNexusOperation(t.Context(), terminal)
			input := interceptor.NexusStartOperationInput{Input: value, Options: nexus.StartOperationOptions{Header: header, RequestID: "request-id"}}
			result, err := inbound.StartOperation(t.Context(), input)
			require.NoError(t, err)
			require.Equal(t, "accepted", result.(*nexus.HandlerStartOperationResultSync[*umpirespb.Value]).Value.GetText())
			_, err = reservationForEntrypoint(t, session, "handler").Wait(t.Context())
			require.NoError(t, err)
			replay, err := inbound.StartOperation(t.Context(), input)
			require.NoError(t, err)
			require.Equal(t, "accepted", replay.(*nexus.HandlerStartOperationResultSync[*umpirespb.Value]).Value.GetText())
			require.Len(t, session.nexusAdmissions, 1)
			_, err = inbound.StartOperation(t.Context(), interceptor.NexusStartOperationInput{Input: value, Options: nexus.StartOperationOptions{Header: header, RequestID: "crossed"}})
			require.Error(t, err)
		})
	}
}

func TestSDKAdmittedWorkflowUsesCachedDispatchWhenStopRacesNextCommand(t *testing.T) {
	prepared := preparedRuntimeFixture(t, umpirespb.NEXUS_RESPONSE_KIND_SYNCHRONOUS)
	host, definition := runtimeTestHost(t, prepared)
	host.options.namespace = "default-test-namespace"
	host.options.client = &recordingClient{}
	binding := WorkflowBinding{Namespace: "default-test-namespace", WorkflowID: "default-test-workflow-id", WorkflowType: "workflow-type", TaskQueue: "task-queue"}
	session, _, request := runtimeTestSessionWithBinding(t, host, definition, prepared, "run", "default-test-run-id", binding, SessionOptions{Bridge: newTestBridge()})
	entered, proceed := make(chan struct{}), make(chan struct{})

	var suite testsuite.WorkflowTestSuite
	environment := suite.NewTestWorkflowEnvironment()
	environment.SetWorkerOptions(sdkworker.Options{Interceptors: []interceptor.WorkerInterceptor{&sdkWorkerInterceptor{host: host, queue: "task-queue", registration: definition.registrations[0]}}})
	environment.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: binding.WorkflowID, TaskQueue: binding.TaskQueue})
	environment.SetHeader(request.GetHeader())
	operation := nexus.NewOperationReference[*umpirespb.Value, *umpirespb.Value]("operation")
	environment.OnNexusOperation("service", operation, mock.Anything, mock.Anything).Return(&nexus.HandlerStartOperationResultSync[*umpirespb.Value]{Value: &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "done"}}}, nil)
	environment.RegisterDynamicWorkflow(func(ctx workflow.Context, arguments converter.EncodedValues) (*umpirespb.Value, error) {
		close(entered)
		<-proceed
		return host.dynamicWorkflow(ctx, arguments)
	}, workflow.DynamicRegisterOptions{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		environment.ExecuteWorkflow("workflow-type", "untouched")
	}()
	<-entered
	require.NoError(t, session.Close(t.Context()))
	close(proceed)
	<-done
	require.Error(t, environment.GetWorkflowError())
	environment.AssertNexusOperationCalled(t, "service", "operation", &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "request"}}, mock.Anything)

	replayEnvironment := suite.NewTestWorkflowEnvironment()
	replayEnvironment.SetWorkerOptions(sdkworker.Options{Interceptors: []interceptor.WorkerInterceptor{&sdkWorkerInterceptor{host: host, queue: "task-queue", registration: definition.registrations[0]}}})
	replayEnvironment.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: binding.WorkflowID, TaskQueue: binding.TaskQueue})
	replayEnvironment.SetHeader(request.GetHeader())
	replayEnvironment.OnNexusOperation("service", operation, mock.Anything, mock.Anything).Return(&nexus.HandlerStartOperationResultSync[*umpirespb.Value]{Value: &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "done"}}}, nil)
	replayEnvironment.RegisterDynamicWorkflow(host.dynamicWorkflow, workflow.DynamicRegisterOptions{})
	replayEnvironment.ExecuteWorkflow("workflow-type", "untouched")
	require.NoError(t, replayEnvironment.GetWorkflowError())
	require.Len(t, session.workflowAdmissions, 1)
}

func TestSDKFailedTriggerRejectsWorkflowBeforeExecution(t *testing.T) {
	prepared := preparedRuntimeFixture(t, umpirespb.NEXUS_RESPONSE_KIND_SYNCHRONOUS)
	host, definition := runtimeTestHost(t, prepared)
	host.options.namespace = "default-test-namespace"
	binding := WorkflowBinding{Namespace: "default-test-namespace", WorkflowID: "default-test-workflow-id", WorkflowType: "workflow-type", TaskQueue: "task-queue"}
	_, _, request := runtimeTestSessionWithDisposition(t, host, definition, prepared, "run", "default-test-run-id", binding, SessionOptions{Bridge: newTestBridge()}, delivery.TriggerNonSuccess)

	var suite testsuite.WorkflowTestSuite
	environment := suite.NewTestWorkflowEnvironment()
	environment.SetWorkerOptions(sdkworker.Options{Interceptors: []interceptor.WorkerInterceptor{&sdkWorkerInterceptor{host: host, queue: "task-queue", registration: definition.registrations[0]}}})
	environment.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: binding.WorkflowID, TaskQueue: binding.TaskQueue})
	environment.SetHeader(request.GetHeader())
	environment.RegisterDynamicWorkflow(host.dynamicWorkflow, workflow.DynamicRegisterOptions{})
	environment.ExecuteWorkflow("workflow-type", "untouched")
	require.Error(t, environment.GetWorkflowError())
}

func TestSDKConcurrentRunsAdmitReorderedWorkflowDelivery(t *testing.T) {
	prepared := preparedRuntimeFixture(t, umpirespb.NEXUS_RESPONSE_KIND_SYNCHRONOUS)
	host, definition := runtimeTestHost(t, prepared)
	host.options.namespace = "default-test-namespace"
	host.options.client = &recordingClient{}
	bindingA := WorkflowBinding{Namespace: "default-test-namespace", WorkflowID: "workflow-a", WorkflowType: "workflow-type", TaskQueue: "task-queue"}
	bindingB := WorkflowBinding{Namespace: "default-test-namespace", WorkflowID: "workflow-b", WorkflowType: "workflow-type", TaskQueue: "task-queue"}
	sessionA, _, requestA := runtimeTestSessionWithBinding(t, host, definition, prepared, "run-a", "default-test-run-id", bindingA, SessionOptions{Bridge: newTestBridge()})
	sessionB, _, requestB := runtimeTestSessionWithBinding(t, host, definition, prepared, "run-b", "default-test-run-id", bindingB, SessionOptions{Bridge: newTestBridge()})
	enteredB, proceedB := make(chan struct{}), make(chan struct{})

	newEnvironment := func(binding WorkflowBinding, request *commonpb.Header, dynamic func(workflow.Context, converter.EncodedValues) (*umpirespb.Value, error)) *testsuite.TestWorkflowEnvironment {
		var suite testsuite.WorkflowTestSuite
		environment := suite.NewTestWorkflowEnvironment()
		environment.SetWorkerOptions(sdkworker.Options{Interceptors: []interceptor.WorkerInterceptor{&sdkWorkerInterceptor{host: host, queue: "task-queue", registration: definition.registrations[0]}}})
		environment.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: binding.WorkflowID, TaskQueue: binding.TaskQueue})
		environment.SetHeader(request)
		operation := nexus.NewOperationReference[*umpirespb.Value, *umpirespb.Value]("operation")
		environment.OnNexusOperation("service", operation, mock.Anything, mock.Anything).Return(&nexus.HandlerStartOperationResultSync[*umpirespb.Value]{Value: &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "done"}}}, nil)
		environment.RegisterDynamicWorkflow(dynamic, workflow.DynamicRegisterOptions{})
		return environment
	}
	environmentB := newEnvironment(bindingB, requestB.GetHeader(), func(ctx workflow.Context, arguments converter.EncodedValues) (*umpirespb.Value, error) {
		close(enteredB)
		<-proceedB
		return host.dynamicWorkflow(ctx, arguments)
	})
	doneB := make(chan error, 1)
	go func() {
		environmentB.ExecuteWorkflow("workflow-type", "b")
		doneB <- environmentB.GetWorkflowError()
	}()
	select {
	case <-t.Context().Done():
		require.FailNow(t, "run B was not admitted")
	case <-enteredB:
	}
	environmentA := newEnvironment(bindingA, requestA.GetHeader(), host.dynamicWorkflow)
	environmentA.ExecuteWorkflow("workflow-type", "a")
	require.NoError(t, environmentA.GetWorkflowError())
	close(proceedB)
	require.NoError(t, <-doneB)
	require.Len(t, sessionA.workflowAdmissions, 1)
	require.Len(t, sessionB.workflowAdmissions, 1)
}

func TestBoundedTextPreservesUTF8(t *testing.T) {
	result := boundedText(strings.Repeat("a", 1023) + "é")
	require.LessOrEqual(t, len(result), 1024)
	require.True(t, utf8.ValidString(result))
}

func reservationForEntrypoint(t *testing.T, session *Session, entrypoint string) *reservation {
	t.Helper()
	for _, reservation := range session.reservations {
		if reservation.Identity().EntrypointID == entrypoint {
			return reservation
		}
	}
	require.FailNow(t, "missing reservation", entrypoint)
	return nil
}

func workflowAdmissionForTest(t *testing.T, session *Session) *workflowAdmission {
	t.Helper()
	require.Len(t, session.workflowAdmissions, 1)
	for _, admission := range session.workflowAdmissions {
		return admission
	}
	require.FailNow(t, "missing workflow admission")
	return nil
}

func workflowReplayHistory(t *testing.T, binding WorkflowBinding, header *commonpb.Header, nexusHeader nexus.Header) []*historypb.HistoryEvent {
	t.Helper()
	dataConverter := converter.GetDefaultDataConverter()
	arguments, err := dataConverter.ToPayloads("untouched", 42, []byte("arguments"))
	require.NoError(t, err)
	nexusInput, err := dataConverter.ToPayload(&umpirespb.Value{Value: &umpirespb.Value_Text{Text: "request"}})
	require.NoError(t, err)
	nexusResult, err := dataConverter.ToPayload(&umpirespb.Value{Value: &umpirespb.Value_Text{Text: "done"}})
	require.NoError(t, err)
	workflowResult, err := dataConverter.ToPayloads(&umpirespb.Value{Value: &umpirespb.Value_Text{Text: "done"}})
	require.NoError(t, err)
	return []*historypb.HistoryEvent{
		{
			EventId: 1, EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
			Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{
				WorkflowType: &commonpb.WorkflowType{Name: binding.WorkflowType}, TaskQueue: &taskqueuepb.TaskQueue{Name: binding.TaskQueue},
				Input: arguments, Header: header, OriginalExecutionRunId: "replay-run", WorkflowId: binding.WorkflowID,
			}},
		},
		{EventId: 2, EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED, Attributes: &historypb.HistoryEvent_WorkflowTaskScheduledEventAttributes{WorkflowTaskScheduledEventAttributes: &historypb.WorkflowTaskScheduledEventAttributes{}}},
		{EventId: 3, EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED, Attributes: &historypb.HistoryEvent_WorkflowTaskStartedEventAttributes{WorkflowTaskStartedEventAttributes: &historypb.WorkflowTaskStartedEventAttributes{ScheduledEventId: 2}}},
		{EventId: 4, EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED, Attributes: &historypb.HistoryEvent_WorkflowTaskCompletedEventAttributes{WorkflowTaskCompletedEventAttributes: &historypb.WorkflowTaskCompletedEventAttributes{ScheduledEventId: 2, StartedEventId: 3}}},
		{
			EventId: 5, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED,
			Attributes: &historypb.HistoryEvent_NexusOperationScheduledEventAttributes{NexusOperationScheduledEventAttributes: &historypb.NexusOperationScheduledEventAttributes{
				Endpoint: "endpoint", Service: "service", Operation: "operation", Input: nexusInput, NexusHeader: nexusHeader,
				WorkflowTaskCompletedEventId: 4, RequestId: "nexus-request",
			}},
		},
		{EventId: 6, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED, Attributes: &historypb.HistoryEvent_NexusOperationCompletedEventAttributes{NexusOperationCompletedEventAttributes: &historypb.NexusOperationCompletedEventAttributes{ScheduledEventId: 5, Result: nexusResult, RequestId: "nexus-request"}}},
		{EventId: 7, EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED, Attributes: &historypb.HistoryEvent_WorkflowTaskScheduledEventAttributes{WorkflowTaskScheduledEventAttributes: &historypb.WorkflowTaskScheduledEventAttributes{}}},
		{EventId: 8, EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED, Attributes: &historypb.HistoryEvent_WorkflowTaskStartedEventAttributes{WorkflowTaskStartedEventAttributes: &historypb.WorkflowTaskStartedEventAttributes{ScheduledEventId: 7}}},
		{EventId: 9, EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED, Attributes: &historypb.HistoryEvent_WorkflowTaskCompletedEventAttributes{WorkflowTaskCompletedEventAttributes: &historypb.WorkflowTaskCompletedEventAttributes{ScheduledEventId: 7, StartedEventId: 8}}},
		{EventId: 10, EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED, Attributes: &historypb.HistoryEvent_WorkflowExecutionCompletedEventAttributes{WorkflowExecutionCompletedEventAttributes: &historypb.WorkflowExecutionCompletedEventAttributes{Result: workflowResult, WorkflowTaskCompletedEventId: 9}}},
	}
}

type registeredNexusTerminal struct {
	interceptor.NexusOperationInboundInterceptorBase
	handler            nexus.Handler
	service, operation string
}

func (i *registeredNexusTerminal) StartOperation(ctx context.Context, input interceptor.NexusStartOperationInput) (nexus.HandlerStartOperationResult[any], error) {
	payload, err := converter.GetDefaultDataConverter().ToPayload(input.Input)
	if err != nil {
		return nil, err
	}
	lazy := nexus.NewLazyValue(&registeredNexusPayloadSerializer{payload: payload}, &nexus.Reader{ReadCloser: io.NopCloser(bytes.NewReader(nil))})
	ctx = nexus.WithHandlerContext(ctx, nexus.HandlerInfo{Service: i.service, Operation: i.operation, Header: input.Options.Header})
	return i.handler.StartOperation(ctx, i.service, i.operation, lazy, input.Options)
}

type registeredNexusPayloadSerializer struct {
	payload *commonpb.Payload
}

func (s *registeredNexusPayloadSerializer) Deserialize(_ *nexus.Content, value any) error {
	return converter.GetDefaultDataConverter().FromPayload(s.payload, value)
}

func (*registeredNexusPayloadSerializer) Serialize(any) (*nexus.Content, error) {
	return nil, ErrInvalid
}

func TestSDKAwaitUsesItsOwnTimeout(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		start, await, completion time.Duration
		status                   umpirespb.InstructionOutcomeStatus
	}{
		{"await expires first", 10 * time.Second, time.Second, 5 * time.Second, umpirespb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT},
		{"start expires first", time.Second, 10 * time.Second, 5 * time.Second, umpirespb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT},
		{"completion before await", 10 * time.Second, 5 * time.Second, time.Second, umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prepared := preparedRuntimeFixture(t, umpirespb.NEXUS_RESPONSE_KIND_SYNCHRONOUS, func(program *umpirespb.Program) {
				program.Entrypoints[1].Nodes[0].Bounds.TimeoutMilliseconds = tc.start.Milliseconds()
				program.Entrypoints[1].Nodes[1].Bounds.TimeoutMilliseconds = tc.await.Milliseconds()
			})
			_, definition := runtimeTestHost(t, prepared)
			var suite testsuite.WorkflowTestSuite
			environment := suite.NewTestWorkflowEnvironment()
			operation := nexus.NewOperationReference[*umpirespb.Value, *umpirespb.Value]("operation")
			environment.OnNexusOperation("service", operation, mock.Anything, mock.Anything).Return(&nexus.HandlerStartOperationResultAsync{OperationToken: "token"}, nil)
			require.NoError(t, environment.RegisterNexusAsyncOperationCompletion("service", "operation", "token", &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "done"}}, nil, tc.completion))
			environment.ExecuteWorkflow(func(ctx workflow.Context) (int32, error) {
				entry := definition.entries["workflow"].plan
				i := workflowInterpreter{session: &Session{definition: definition}, ctx: ctx, values: newActivationValues(entry.ID(), entry.RuntimeWorkLimit()), futures: make(map[string]workflow.NexusOperationFuture)}
				start, err := instructionAt(entry, 0)
				if err != nil {
					return 0, err
				}
				if err := i.startNexus(start, &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "request"}}); err != nil {
					return 0, err
				}
				await, err := instructionAt(entry, 1)
				if err != nil {
					return 0, err
				}
				before := workflow.Now(ctx)
				if err := i.awaitNexus(await); err != nil {
					return 0, err
				}
				expected := min(tc.start, tc.await, tc.completion)
				if elapsed := workflow.Now(ctx).Sub(before); elapsed != expected {
					return 0, fmt.Errorf("await elapsed %s, want %s", elapsed, expected)
				}
				return i.values.lookup(umpire.ValueReference{Kind: umpire.OutcomeReference, Entrypoint: "workflow", ID: "await", Field: int32(umpirespb.INSTRUCTION_OUTCOME_FIELD_STATUS)}).GetEnumValue().GetNumber(), nil
			})
			require.NoError(t, environment.GetWorkflowError())
			var status int32
			require.NoError(t, environment.GetWorkflowResult(&status))
			require.EqualValues(t, tc.status, status)
		})
	}
}
