package worker

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	nexuspb "go.temporal.io/api/nexus/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/testsuite"
	sdkworker "go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

func jsonPayload(t *testing.T, value any) *commonpb.Payload {
	t.Helper()
	payload, err := converter.GetDefaultDataConverter().ToPayload(value)
	require.NoError(t, err)
	return payload
}

// typedProfile admits the typed instructions beside the untyped ones the runtime fixture uses, and
// binds the worker's namespace to the one the SDK test environment runs under.
func typedProfile(profile *testpilot.ProfileSpec) {
	profile.Opcodes = append(profile.Opcodes, testpilot.WorkflowCommand, testpilot.NexusHandlerReply, testpilot.NexusOperationCompletion)
	profile.CommandTypes = CommandTypes()
	for index := range profile.EnvironmentBindings {
		if profile.EnvironmentBindings[index].ID == "namespace" {
			profile.EnvironmentBindings[index].Value = "default-test-namespace"
		}
	}
}

// scheduleCommandProgram replaces the workflow's untyped start with a schedule command carrying
// the given attributes.
func scheduleCommandProgram(t *testing.T, attributes *commandpb.ScheduleNexusOperationCommandAttributes) func(*testpilotspb.Program) {
	t.Helper()
	return func(program *testpilotspb.Program) {
		program.Entrypoints[1].Instructions[0].Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_WorkflowCommand{WorkflowCommand: &testpilotspb.WorkflowCommand{Command: &commandpb.Command{
			CommandType: enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION,
			Attributes:  &commandpb.Command_ScheduleNexusOperationCommandAttributes{ScheduleNexusOperationCommandAttributes: attributes},
		}}}}
	}
}

// A schedule command reaches the SDK as workflow.ExecuteNexusOperation: its carried payload is the
// operation's input, unconverted; its Nexus header travels beside the Driver's routing header; and
// the handler's payload comes back whole as the Await's value.
func TestSDKWorkflowIssuesTheCarriedScheduleCommand(t *testing.T) {
	request := jsonPayload(t, "request")
	done := jsonPayload(t, "done")
	prepared := preparedRuntimeFixtureWithProfile(t, testpilotspb.NEXUS_RESPONSE_KIND_SYNCHRONOUS, typedProfile, scheduleCommandProgram(t, &commandpb.ScheduleNexusOperationCommandAttributes{
		Endpoint: "nexus-endpoint", Service: "service", Operation: "operation", Input: request,
		ScheduleToCloseTimeout: durationpb.New(7 * time.Second), ScheduleToStartTimeout: durationpb.New(3 * time.Second), StartToCloseTimeout: durationpb.New(5 * time.Second),
		NexusHeader: map[string]string{"x-case": "carried"},
	}))
	host, definition := runtimeTestDriver(t, prepared)
	host.options.client = &recordingClient{}
	binding := WorkflowBinding{Namespace: "default-test-namespace", WorkflowID: "typed-workflow", WorkflowType: "workflow-type", TaskQueue: "task-queue"}
	session, _, start := runtimeTestSessionWithBinding(t, host, definition, prepared, "run", "default-test-run-id", binding, SessionOptions{Bridge: newTestBridge()})

	var suite testsuite.WorkflowTestSuite
	environment := suite.NewTestWorkflowEnvironment()
	environment.SetWorkerOptions(sdkworker.Options{Interceptors: []interceptor.WorkerInterceptor{&sdkWorkerInterceptor{host: host, queue: "task-queue", registration: definition.registrations[0]}}})
	environment.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: binding.WorkflowID, TaskQueue: binding.TaskQueue})
	environment.SetHeader(start.GetHeader())
	operation := nexus.NewOperationReference[converter.RawValue, converter.RawValue]("operation")
	var options workflow.NexusOperationOptions
	var input converter.RawValue
	environment.OnNexusOperation("service", operation, mock.Anything, mock.Anything).Run(func(arguments mock.Arguments) {
		input = arguments.Get(1).(converter.RawValue)
		options = arguments.Get(2).(workflow.NexusOperationOptions)
	}).Return(&nexus.HandlerStartOperationResultSync[converter.RawValue]{Value: converter.NewRawValue(done)}, nil)
	environment.RegisterDynamicWorkflow(host.dynamicWorkflow, workflow.DynamicRegisterOptions{})
	environment.ExecuteWorkflow("workflow-type", "untouched")
	require.NoError(t, environment.GetWorkflowError())

	var result testpilotspb.Value
	require.NoError(t, environment.GetWorkflowResult(&result))
	var awaited commonpb.Payload
	require.NoError(t, result.GetMessageValue().UnmarshalTo(&awaited))
	require.True(t, proto.Equal(done, &awaited))
	require.True(t, proto.Equal(request, input.Payload()))
	require.Equal(t, 7*time.Second, options.ScheduleToCloseTimeout)
	require.Equal(t, 3*time.Second, options.ScheduleToStartTimeout)
	require.Equal(t, 5*time.Second, options.StartToCloseTimeout)
	workflowResult, err := reservationForEntrypoint(t, session, "workflow").Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, workflowResult.Outcome.GetStatus())
}

// A schedule command's Nexus header travels beside the Driver's routing header, and a Case header
// that spells the routing header's own name is rejected rather than overwritten.
func TestPreparedNexusHeaderCarriesTheCaseHeader(t *testing.T) {
	prepared := preparedRuntimeFixtureWithProfile(t, testpilotspb.NEXUS_RESPONSE_KIND_SYNCHRONOUS, typedProfile, scheduleCommandProgram(t, &commandpb.ScheduleNexusOperationCommandAttributes{
		Endpoint: "nexus-endpoint", Service: "service", Operation: "operation", NexusHeader: map[string]string{"x-case": "carried"},
	}))
	host, definition := runtimeTestDriver(t, prepared)
	session, _, request := runtimeTestSessionWithBinding(t, host, definition, prepared, "run", "temporal-run", WorkflowBinding{Namespace: "default-test-namespace", WorkflowID: "workflow", WorkflowType: "workflow-type", TaskQueue: "task-queue"}, SessionOptions{Bridge: newTestBridge()})
	workflowRoute, err := host.admitWorkflow(workflowDelivery(request, "temporal-run"))
	require.NoError(t, err)
	header, err := session.preparedNexusHeader(workflowRoute.activation, "start", nexus.Header{"x-case": "carried"})
	require.NoError(t, err)
	require.Equal(t, "carried", header.Get("x-case"))
	require.NotEmpty(t, header.Get("temporal-testpilot-reserved-nexus-v1"))
	_, err = session.preparedNexusHeader(workflowRoute.activation, "start", nexus.Header{"temporal-testpilot-reserved-nexus-v1": "forged"})
	require.ErrorIs(t, err, delivery.ErrReservedHeader)
}

// The carried schedule-to-close timeout bounds the operation, not the instruction's own timeout:
// a completion that arrives after it times the Await out although the Await's limit is longer.
func TestSDKScheduleCommandCarriesItsOwnTimeouts(t *testing.T) {
	prepared := preparedRuntimeFixtureWithProfile(t, testpilotspb.NEXUS_RESPONSE_KIND_SYNCHRONOUS, typedProfile, scheduleCommandProgram(t, &commandpb.ScheduleNexusOperationCommandAttributes{
		Endpoint: "nexus-endpoint", Service: "service", Operation: "operation", ScheduleToCloseTimeout: durationpb.New(time.Second),
	}), func(program *testpilotspb.Program) {
		program.Entrypoints[1].Instructions[0].Limits.Timeout = &testpilotspb.InstructionLimits_TimeoutMilliseconds{TimeoutMilliseconds: 10000}
		program.Entrypoints[1].Instructions[1].Limits.Timeout = &testpilotspb.InstructionLimits_TimeoutMilliseconds{TimeoutMilliseconds: 10000}
	})
	host, definition := runtimeTestDriver(t, prepared)
	host.options.client = &recordingClient{}
	binding := WorkflowBinding{Namespace: "default-test-namespace", WorkflowID: "timeout-workflow", WorkflowType: "workflow-type", TaskQueue: "task-queue"}
	_, _, start := runtimeTestSessionWithBinding(t, host, definition, prepared, "run", "default-test-run-id", binding, SessionOptions{Bridge: newTestBridge()})

	var suite testsuite.WorkflowTestSuite
	environment := suite.NewTestWorkflowEnvironment()
	environment.SetWorkerOptions(sdkworker.Options{Interceptors: []interceptor.WorkerInterceptor{&sdkWorkerInterceptor{host: host, queue: "task-queue", registration: definition.registrations[0]}}})
	environment.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: binding.WorkflowID, TaskQueue: binding.TaskQueue})
	environment.SetHeader(start.GetHeader())
	operation := nexus.NewOperationReference[converter.RawValue, converter.RawValue]("operation")
	environment.OnNexusOperation("service", operation, mock.Anything, mock.Anything).Return(&nexus.HandlerStartOperationResultAsync{OperationToken: "token"}, nil)
	require.NoError(t, environment.RegisterNexusAsyncOperationCompletion("service", "operation", "token", converter.NewRawValue(jsonPayload(t, "late")), nil, 5*time.Second))
	environment.RegisterDynamicWorkflow(host.dynamicWorkflow, workflow.DynamicRegisterOptions{})
	before := environment.Now()
	environment.ExecuteWorkflow("workflow-type", "untouched")
	// The finish is guarded on the await succeeding, so the workflow ends without a Finish.
	require.Error(t, environment.GetWorkflowError())
	require.Equal(t, time.Second, environment.Now().Sub(before))
}

// Each typed reply reaches the SDK as the handler's return: a synchronous payload unconverted, an
// asynchronous reply through the completion authority the Driver publishes under the Driver's
// own token, a handler error with its type, message and retry behavior, and a failed start as the
// operation error it denotes.
func TestSessionAnswersTypedReplies(t *testing.T) {
	answer := jsonPayload(t, "answered")
	for _, tc := range []struct {
		name  string
		reply *testpilotspb.NexusHandlerReply
		check func(t *testing.T, result nexus.HandlerStartOperationResult[any], err error, bridge *testBridge)
	}{
		{"synchronous payload", &testpilotspb.NexusHandlerReply{Reply: &testpilotspb.NexusHandlerReply_Response{Response: &nexuspb.StartOperationResponse{Variant: &nexuspb.StartOperationResponse_SyncSuccess{SyncSuccess: &nexuspb.StartOperationResponse_Sync{Payload: answer}}}}},
			func(t *testing.T, result nexus.HandlerStartOperationResult[any], err error, bridge *testBridge) {
				require.NoError(t, err)
				raw, ok := result.(*nexus.HandlerStartOperationResultSync[any]).Value.(converter.RawValue)
				require.True(t, ok)
				require.True(t, proto.Equal(answer, raw.Payload()))
				require.False(t, bridge.published)
			}},
		{"asynchronous handle", &testpilotspb.NexusHandlerReply{HandleSlotId: "capability", Reply: &testpilotspb.NexusHandlerReply_Response{Response: &nexuspb.StartOperationResponse{Variant: &nexuspb.StartOperationResponse_AsyncSuccess{AsyncSuccess: &nexuspb.StartOperationResponse_Async{}}}}},
			func(t *testing.T, result nexus.HandlerStartOperationResult[any], err error, bridge *testBridge) {
				require.NoError(t, err)
				require.Equal(t, "request-id", result.(*nexus.HandlerStartOperationResultAsync).OperationToken)
				require.True(t, bridge.published)
				require.Equal(t, "capability", bridge.slot)
			}},
		{"handler error", &testpilotspb.NexusHandlerReply{Reply: &testpilotspb.NexusHandlerReply_Error{Error: &nexuspb.HandlerError{ErrorType: "BAD_REQUEST", Failure: &nexuspb.Failure{Message: "malformed request"}, RetryBehavior: enumspb.NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_NON_RETRYABLE}}},
			func(t *testing.T, result nexus.HandlerStartOperationResult[any], err error, bridge *testBridge) {
				var handlerErr *nexus.HandlerError
				require.ErrorAs(t, err, &handlerErr)
				require.Equal(t, nexus.HandlerErrorTypeBadRequest, handlerErr.Type)
				require.Equal(t, "malformed request", handlerErr.Message)
				require.Equal(t, nexus.HandlerErrorRetryBehaviorNonRetryable, handlerErr.RetryBehavior)
				require.Nil(t, result)
				require.False(t, bridge.published)
			}},
		{"failed start", &testpilotspb.NexusHandlerReply{Reply: &testpilotspb.NexusHandlerReply_Response{Response: &nexuspb.StartOperationResponse{Variant: &nexuspb.StartOperationResponse_Failure{Failure: &failurepb.Failure{Message: "canceled by the handler", FailureInfo: &failurepb.Failure_CanceledFailureInfo{CanceledFailureInfo: &failurepb.CanceledFailureInfo{}}}}}}},
			func(t *testing.T, result nexus.HandlerStartOperationResult[any], err error, bridge *testBridge) {
				var operationErr *nexus.OperationError
				require.ErrorAs(t, err, &operationErr)
				require.Equal(t, nexus.OperationStateCanceled, operationErr.State)
				require.Equal(t, "canceled by the handler", operationErr.Message)
				require.Nil(t, result)
			}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prepared := preparedRuntimeFixtureWithProfile(t, testpilotspb.NEXUS_RESPONSE_KIND_SYNCHRONOUS, typedProfile, func(program *testpilotspb.Program) {
				program.Slots = []*testpilotspb.Slot{{SlotId: "capability", Content: &testpilotspb.Slot_OpaqueHandle{OpaqueHandle: &testpilotspb.OpaqueHandleType{}}}}
				program.Entrypoints[2].Instructions[0].Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_NexusHandlerReply{NexusHandlerReply: tc.reply}}
			})
			host, definition := runtimeTestDriver(t, prepared)
			bridge := newTestBridge()
			options := SessionOptions{Bridge: bridge, NewCapability: func(context.Context, testpilot.Coordinate, testpilot.CapabilityEffect) (testpilot.OpaqueCapability, error) {
				return &struct{}{}, nil
			}}
			session, _, request := runtimeTestSessionWithBinding(t, host, definition, prepared, "run", "temporal-run", WorkflowBinding{Namespace: "default-test-namespace", WorkflowID: "workflow", WorkflowType: "workflow-type", TaskQueue: "task-queue"}, options)
			workflowRoute, err := host.admitWorkflow(workflowDelivery(request, "temporal-run"))
			require.NoError(t, err)
			header, err := session.preparedNexusHeader(workflowRoute.activation, "start", nil)
			require.NoError(t, err)
			nexusRoute, err := host.admitNexus(t.Context(), "task-queue", delivery.NexusDelivery{Header: header, RequestID: "request-id"}, func() {})
			require.NoError(t, err)
			result, err := session.executeNexus(t.Context(), nexusRoute.activation, nil, nexus.StartOperationOptions{CallbackURL: "https://callback.invalid/private", RequestID: "request-id"})
			tc.check(t, result, err, bridge)
		})
	}
}

// A carried completion reaches the callback as its body: a payload verbatim, and a failure as the
// operation error it denotes, canceled or failed by its failure info.
func TestCompletionTransportDeliversCarriedPayloadAndFailure(t *testing.T) {
	type received struct {
		data        []byte
		contentType string
		state       string
	}
	requests := make(chan received, 1)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read completion body: %v", err)
		}
		requests <- received{data, r.Header.Get("Content-Type"), r.Header.Get("Nexus-Operation-State")}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()
	transport, err := newCompletionTransport(nil, "", callbackLimits())
	require.NoError(t, err)
	effect, err := transport.newEffect(completionInfo{URL: target.URL, OperationToken: "token"})
	require.NoError(t, err)

	payload := jsonPayload(t, map[string]string{"answer": "carried"})
	result := effect.Invoke(t.Context(), payload, 4096)
	require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, result.Outcome.Status)
	req := <-requests
	require.Equal(t, "succeeded", req.state)
	require.Equal(t, string(payload.GetData()), string(req.data))

	for _, tc := range []struct {
		name    string
		failure *failurepb.Failure
		state   string
	}{
		{"failed", &failurepb.Failure{Message: "handler gave up", FailureInfo: &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{Type: "GaveUp", NonRetryable: true}}}, "failed"},
		{"canceled", &failurepb.Failure{Message: "handler canceled", FailureInfo: &failurepb.Failure_CanceledFailureInfo{CanceledFailureInfo: &failurepb.CanceledFailureInfo{}}}, "canceled"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := effect.Invoke(t.Context(), tc.failure, 4096)
			require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, result.Outcome.Status)
			req := <-requests
			require.Equal(t, tc.state, req.state)
			require.Contains(t, string(req.data), tc.failure.GetMessage())
		})
	}
}

func TestCompletionEffectAcceptsTypedCompletions(t *testing.T) {
	transport, err := newCompletionTransport(nil, "", callbackLimits())
	require.NoError(t, err)
	effect, err := transport.newEffect(completionInfo{URL: "http://localhost", OperationToken: "token"})
	require.NoError(t, err)
	payload := &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_NexusOperationCompletion{NexusOperationCompletion: &testpilotspb.NexusOperationCompletion{Result: &testpilotspb.NexusOperationCompletion_Payload{Payload: &commonpb.Payload{}}}}}
	failure := &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_NexusOperationCompletion{NexusOperationCompletion: &testpilotspb.NexusOperationCompletion{Result: &testpilotspb.NexusOperationCompletion_Failure{Failure: &failurepb.Failure{}}}}}
	untyped := &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_CompleteNexusOperation{CompleteNexusOperation: &testpilotspb.CompleteNexusOperation{}}}
	require.True(t, effect.Accepts(t.Context(), payload, &commonpb.Payload{}))
	require.True(t, effect.Accepts(t.Context(), failure, &failurepb.Failure{}))
	require.False(t, effect.Accepts(t.Context(), payload, &failurepb.Failure{}))
	require.False(t, effect.Accepts(t.Context(), failure, &commonpb.Payload{}))
	require.False(t, effect.Accepts(t.Context(), untyped, &commonpb.Payload{}))
	require.False(t, effect.Accepts(t.Context(), payload, callbackValue()))
	require.Equal(t, "invalid_argument", effect.Invoke(t.Context(), &testpilotspb.InstructionOutcome{}, 4096).Outcome.ProtocolCode)
}

func TestOperationErrorRequiresAFailure(t *testing.T) {
	_, err := operationError(nil)
	require.ErrorIs(t, err, ErrInvalid)
}
