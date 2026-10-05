package worker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	historypb "go.temporal.io/api/history/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	sdkworker "go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport/facadetest"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
)

const activityTaskQueue = "activity-task-queue"

// scheduleActivityProgram turns the runtime fixture's workflow into one that schedules an activity
// on its own task-queue role, awaits it and finishes with its result, and drops the Nexus handler
// nothing schedules any more.
func scheduleActivityProgram(attributes *commandpb.ScheduleActivityTaskCommandAttributes) func(*testpilotspb.Program) {
	return func(program *testpilotspb.Program) {
		program.Roles = append(program.Roles, &testpilotspb.Role{RoleId: "activity-queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingId: "namespace", ResourceBindingId: activityTaskQueue})
		program.Entrypoints[1].Instructions[0].Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_WorkflowCommand{WorkflowCommand: &testpilotspb.WorkflowCommand{Command: &commandpb.Command{
			CommandType: enumspb.COMMAND_TYPE_SCHEDULE_ACTIVITY_TASK,
			Attributes:  &commandpb.Command_ScheduleActivityTaskCommandAttributes{ScheduleActivityTaskCommandAttributes: attributes},
		}}}}
		// The await outlasts the retry backoff an attempt that fails retryably waits out.
		program.Entrypoints[1].Instructions[1].Limits.Timeout = &testpilotspb.InstructionLimits_TimeoutMilliseconds{TimeoutMilliseconds: 10000}
		program.Entrypoints = program.Entrypoints[:2]
	}
}

// finishWithAwaitStatus makes the workflow finish, whatever the Await's outcome, with its status.
func finishWithAwaitStatus(program *testpilotspb.Program) {
	finish := program.Entrypoints[1].Instructions[2]
	finish.Guard = &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: true}}}}
	finish.Instruction.GetFinish().Result.GetReference().GetOutcome().Field = testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS
}

func authorizeActivityQueue(profile *testpilot.ProfileSpec) {
	authorizeActivityQueueIn("default-test-namespace")(profile)
}

// authorizeActivityQueueIn binds the activity's task-queue role, under the namespace the SDK test
// environment or replayer names.
func authorizeActivityQueueIn(namespace string) func(*testpilot.ProfileSpec) {
	return func(profile *testpilot.ProfileSpec) {
		profile.Roles = append(profile.Roles, testpilot.RolePolicy{ID: "activity-queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE})
		profile.EnvironmentBindings = append(profile.EnvironmentBindings, testpilot.EnvironmentBinding{ID: activityTaskQueue, Value: activityTaskQueue})
		for index := range profile.EnvironmentBindings {
			if profile.EnvironmentBindings[index].ID == "namespace" {
				profile.EnvironmentBindings[index].Value = namespace
			}
		}
	}
}

func scheduledActivity() *commandpb.ScheduleActivityTaskCommandAttributes {
	return &commandpb.ScheduleActivityTaskCommandAttributes{
		ActivityId: "activity", ActivityType: &commonpb.ActivityType{Name: "activity-type"}, TaskQueue: &taskqueuepb.TaskQueue{Name: "activity-queue"},
		Input:                  &commonpb.Payloads{Payloads: []*commonpb.Payload{facadetest.Payload("request"), facadetest.Payload("second")}},
		ScheduleToCloseTimeout: durationpb.New(5 * time.Second),
		StartToCloseTimeout:    durationpb.New(2 * time.Second),
		RetryPolicy:            &commonpb.RetryPolicy{InitialInterval: durationpb.New(time.Second), MaximumAttempts: 2, NonRetryableErrorTypes: []string{"fatal"}},
	}
}

// driverWorkflowsOnly is the Driver's interceptor for workflows alone: the activity the workflow
// schedules runs on an ordinary worker the Driver does not intercept.
type driverWorkflowsOnly struct {
	interceptor.WorkerInterceptorBase
	driver *sdkWorkerInterceptor
}

func (i *driverWorkflowsOnly) InterceptWorkflow(ctx workflow.Context, next interceptor.WorkflowInboundInterceptor) interceptor.WorkflowInboundInterceptor {
	return i.driver.InterceptWorkflow(ctx, next)
}

type activityCall struct {
	info          activity.Info
	first, second string
}

// runScheduledActivity runs the fixture's workflow through the SDK with an ordinary activity
// registered under the scheduled type, which answers as answer does.
func runScheduledActivity(t *testing.T, attributes *commandpb.ScheduleActivityTaskCommandAttributes, answer func(activityCall) (string, error), modify ...func(*testpilotspb.Program)) (*testsuite.TestWorkflowEnvironment, *Session, []activityCall) {
	t.Helper()
	prepared := preparedRuntimeFixtureWithProfile(t, replySynchronous, authorizeActivityQueue, append([]func(*testpilotspb.Program){scheduleActivityProgram(attributes)}, modify...)...)
	host, definition := runtimeTestDriver(t, prepared)
	require.Equal(t, map[string]string{"activity-queue": activityTaskQueue}, definition.queues)
	host.options.client = &recordingClient{}
	binding := delivery.WorkflowBinding{Namespace: "default-test-namespace", WorkflowID: "default-test-workflow-id", WorkflowType: "workflow-type", TaskQueue: "task-queue"}
	session, _, request := runtimeTestSessionWithBinding(t, host, definition, prepared, "run", "default-test-run-id", binding, SessionOptions{Bridge: newTestBridge()})

	var suite testsuite.WorkflowTestSuite
	environment := suite.NewTestWorkflowEnvironment()
	environment.SetWorkerOptions(sdkworker.Options{Interceptors: []interceptor.WorkerInterceptor{&driverWorkflowsOnly{driver: &sdkWorkerInterceptor{host: host, queue: "task-queue", registration: definition.registrations[0]}}}})
	environment.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: binding.WorkflowID, TaskQueue: binding.TaskQueue})
	environment.SetHeader(request.GetHeader())
	var calls []activityCall
	environment.RegisterActivityWithOptions(func(ctx context.Context, first, second string) (string, error) {
		call := activityCall{info: activity.GetInfo(ctx), first: first, second: second}
		calls = append(calls, call)
		return answer(call)
	}, activity.RegisterOptions{Name: "activity-type"})
	environment.RegisterDynamicWorkflow(host.dynamicWorkflow, workflow.DynamicRegisterOptions{})
	environment.ExecuteWorkflow("workflow-type", "untouched")
	return environment, session, calls
}

// The workflow schedules the activity the command carries, on the queue its task-queue role binds,
// with the carried input unconverted and the carried timeout and retry policy; it awaits the
// activity's result and completes with that payload, whole.
func TestSDKWorkflowSchedulesAnActivityAwaitsItAndFinishesWithItsResult(t *testing.T) {
	environment, session, calls := runScheduledActivity(t, scheduledActivity(), func(call activityCall) (string, error) {
		if call.info.Attempt == 1 {
			return "", temporal.NewApplicationError("first attempt", "retry")
		}
		return call.first + " " + call.second + " done", nil
	})
	require.NoError(t, environment.GetWorkflowError())
	var result testpilotspb.Value
	require.NoError(t, environment.GetWorkflowResult(&result))
	require.Equal(t, "request second done", facadetest.CarriedText(t, &result))
	require.Len(t, calls, 2)
	scheduled := calls[1]
	require.Equal(t, "request", scheduled.first)
	require.Equal(t, "second", scheduled.second)
	require.Equal(t, "activity", scheduled.info.ActivityID)
	require.Equal(t, "activity-type", scheduled.info.ActivityType.Name)
	require.Equal(t, activityTaskQueue, scheduled.info.TaskQueue)
	require.Equal(t, 5*time.Second, scheduled.info.ScheduleToCloseTimeout)
	require.Equal(t, 2*time.Second, scheduled.info.StartToCloseTimeout)
	require.Equal(t, int32(2), scheduled.info.Attempt)
	workflowResult, err := reservationForEntrypoint(t, session, "workflow").Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, workflowResult.Outcome.GetStatus())
}

// An activity that fails past its retry policy fails the Await rather than the workflow: the Await
// records the SDK failure, which a Finish may read.
func TestSDKWorkflowAwaitRecordsTheActivityFailure(t *testing.T) {
	environment, session, calls := runScheduledActivity(t, scheduledActivity(), func(activityCall) (string, error) {
		return "", temporal.NewApplicationError("fatal failure", "fatal")
	}, finishWithAwaitStatus)
	require.NoError(t, environment.GetWorkflowError())
	var result testpilotspb.Value
	require.NoError(t, environment.GetWorkflowResult(&result))
	require.Equal(t, "INSTRUCTION_OUTCOME_STATUS_SDK_FAILURE", result.GetEnumValue().GetName())
	require.Len(t, calls, 1, "a non-retryable error type of the carried retry policy ends the activity")
	workflowResult, err := reservationForEntrypoint(t, session, "workflow").Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, workflowResult.Outcome.GetStatus())
}

// routedActivityProgram is the runtime fixture's workflow scheduling an activity on its own task
// queue whose attempts the Driver answers from the activity entrypoint's script.
func routedActivityProgram(attributes *commandpb.ScheduleActivityTaskCommandAttributes, script ...*testpilotspb.Instruction) func(*testpilotspb.Program) {
	return func(program *testpilotspb.Program) {
		attributes.TaskQueue = &taskqueuepb.TaskQueue{Name: "queue"}
		scheduleActivityProgram(attributes)(program)
		activity := &testpilotspb.Entrypoint{
			EntrypointId: "activity",
			Activation:   &testpilotspb.Entrypoint_Activity{Activity: &testpilotspb.ActivityActivation{ActivityType: "activity-type", WorkerRoleId: "worker", TaskQueueRoleId: "queue", AttemptNumbering: &testpilotspb.AttemptNumbering{First: 1, OneRun: true}}},
		}
		for index, instruction := range script {
			activity.Instructions = append(activity.Instructions, &testpilotspb.InstructionNode{InstructionId: fmt.Sprintf("attempt-%d", index+1), Instruction: instruction, Limits: facadetest.Bounds()})
		}
		program.Entrypoints = append(program.Entrypoints, activity)
	}
}

func authorizeRoutedActivity(attempts int64) func(*testpilot.ProfileSpec) {
	return authorizeRoutedActivityIn("default-test-namespace", attempts)
}

func authorizeRoutedActivityIn(namespace string, attempts int64) func(*testpilot.ProfileSpec) {
	return func(profile *testpilot.ProfileSpec) {
		authorizeActivityQueueIn(namespace)(profile)
		profile.Roles[0].ReservationCarriers[0].Shapes = []testpilot.ReservationCarrierShape{{Kind: testpilot.WorkflowEntrypoint, MaximumCount: 1}, {Kind: testpilot.ActivityEntrypoint, MaximumCount: attempts}}
		profile.Opcodes = append(profile.Opcodes, testpilot.ActivityAttemptFailure, testpilot.ActivityAttemptWithholding)
	}
}

var (
	completeAttempt = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: facadetest.Text("done")}}}
	failAttempt     = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptFailure{ActivityAttemptFailure: &testpilotspb.ActivityAttemptFailure{
		Failure: &failurepb.Failure{Message: "not yet", FailureInfo: &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{Type: "transient"}}},
	}}}
	withholdAttempt = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptWithholding{ActivityAttemptWithholding: &testpilotspb.ActivityAttemptWithholding{}}}
)

// runRoutedActivity runs the workflow through the SDK with the Driver intercepting both the
// workflow and the activity it schedules, so each attempt is answered by the activity's script.
func runRoutedActivity(t *testing.T, attributes *commandpb.ScheduleActivityTaskCommandAttributes, script []*testpilotspb.Instruction, modify ...func(*testpilotspb.Program)) (*testsuite.TestWorkflowEnvironment, *Session) {
	t.Helper()
	modifiers := append([]func(*testpilotspb.Program){routedActivityProgram(attributes, script...)}, modify...)
	prepared := preparedRuntimeFixtureWithProfile(t, replySynchronous, authorizeRoutedActivity(int64(len(script))), modifiers...)
	host, definition := runtimeTestDriver(t, prepared)
	host.options.client = &recordingClient{}
	binding := delivery.WorkflowBinding{Namespace: "default-test-namespace", WorkflowID: "default-test-workflow-id", WorkflowType: "workflow-type", TaskQueue: "task-queue"}
	session, _, request := runtimeTestSessionWithBinding(t, host, definition, prepared, "run", "default-test-run-id", binding, SessionOptions{Bridge: newTestBridge()})

	var suite testsuite.WorkflowTestSuite
	environment := suite.NewTestWorkflowEnvironment()
	environment.SetWorkerOptions(sdkworker.Options{Interceptors: []interceptor.WorkerInterceptor{&sdkWorkerInterceptor{host: host, queue: "task-queue", registration: definition.registrations[0]}}})
	environment.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: binding.WorkflowID, TaskQueue: binding.TaskQueue})
	environment.SetHeader(request.GetHeader())
	environment.RegisterDynamicWorkflow(host.dynamicWorkflow, workflow.DynamicRegisterOptions{})
	environment.RegisterDynamicActivity(host.dynamicActivity, activity.DynamicRegisterOptions{})
	environment.ExecuteWorkflow("workflow-type", "untouched")
	return environment, session
}

// attemptOutcome is the outcome the activity's attempt at ordinal settled with.
func attemptOutcome(t *testing.T, session *Session, ordinal int64) *testpilotspb.InstructionOutcome {
	t.Helper()
	for _, raw := range session.reservations {
		if raw.Identity().EntrypointID == "activity" && raw.Identity().Ordinal == ordinal {
			result, err := raw.Wait(t.Context())
			require.NoError(t, err)
			return result.Outcome
		}
	}
	require.FailNow(t, "missing attempt reservation", ordinal)
	return nil
}

func requireAttempt(t *testing.T, outcome *testpilotspb.InstructionOutcome, status testpilotspb.InstructionOutcomeStatus, response testpilotspb.ActivityAttemptResponse, sdkAttempt int32) {
	t.Helper()
	require.Equal(t, status, outcome.GetStatus(), outcome.GetDetail())
	require.Equal(t, response, outcome.GetActivityAttempt().GetResponse())
	require.Equal(t, sdkAttempt, outcome.GetActivityAttempt().GetSdkAttempt())
	require.Equal(t, "default-test-run-id", outcome.GetActivityAttempt().GetActivityRunId(), "a scheduled activity's attempts name their workflow's run")
}

// The attempts of an activity the Driver's workflow schedules reach the Driver's activity
// entrypoint by the route the schedule command carries, and attempt N runs the script's Nth
// instruction: one completes the activity, a retryable failure is followed by the next attempt, and
// a withheld answer leaves the attempt to its deadline, which times the activity out. Attempts the
// closed workflow's activity never needed are settled as not needed.
func TestSDKWorkflowRoutesItsActivityAttemptsToTheActivityScript(t *testing.T) {
	retried := func() *commandpb.ScheduleActivityTaskCommandAttributes {
		return &commandpb.ScheduleActivityTaskCommandAttributes{
			ActivityType: &commonpb.ActivityType{Name: "activity-type"}, ScheduleToCloseTimeout: durationpb.New(5 * time.Second), StartToCloseTimeout: durationpb.New(2 * time.Second),
			RetryPolicy: &commonpb.RetryPolicy{InitialInterval: durationpb.New(time.Second), MaximumAttempts: 2},
		}
	}
	t.Run("complete", func(t *testing.T) {
		environment, session := runRoutedActivity(t, retried(), []*testpilotspb.Instruction{completeAttempt, failAttempt})
		require.NoError(t, environment.GetWorkflowError())
		var result testpilotspb.Value
		require.NoError(t, environment.GetWorkflowResult(&result))
		var payload commonpb.Payload
		require.NoError(t, result.GetMessageValue().UnmarshalTo(&payload))
		var answered testpilotspb.Value
		require.NoError(t, converter.GetDefaultDataConverter().FromPayload(&payload, &answered))
		require.Equal(t, "done", answered.GetTextValue())
		requireAttempt(t, attemptOutcome(t, session, 0), testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED, 1)
		requireAttempt(t, attemptOutcome(t, session, 1), testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED, testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_NOT_NEEDED, 0)
	})
	t.Run("retry then complete", func(t *testing.T) {
		environment, session := runRoutedActivity(t, retried(), []*testpilotspb.Instruction{failAttempt, completeAttempt})
		require.NoError(t, environment.GetWorkflowError())
		requireAttempt(t, attemptOutcome(t, session, 0), testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_RETRYABLE, 1)
		requireAttempt(t, attemptOutcome(t, session, 1), testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED, 2)
	})
	t.Run("withheld until timed out", func(t *testing.T) {
		attributes := retried()
		attributes.StartToCloseTimeout = durationpb.New(time.Second)
		attributes.RetryPolicy.MaximumAttempts = 1
		environment, session := runRoutedActivity(t, attributes, []*testpilotspb.Instruction{withholdAttempt}, finishWithAwaitStatus)
		require.NoError(t, environment.GetWorkflowError())
		var result testpilotspb.Value
		require.NoError(t, environment.GetWorkflowResult(&result))
		require.Equal(t, "INSTRUCTION_OUTCOME_STATUS_TIMED_OUT", result.GetEnumValue().GetName())
		requireAttempt(t, attemptOutcome(t, session, 0), testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_WITHHELD, 1)
	})
}

// The workflow replays over the history of a scheduled activity: over the partial history up to
// the schedule its admission stays open, with the routing header the schedule command carried, and
// over the whole history it completes, once.
func TestSDKWorkflowReplayerCompletesAWorkflowAwaitingItsActivity(t *testing.T) {
	attributes := &commandpb.ScheduleActivityTaskCommandAttributes{ActivityId: "activity", ActivityType: &commonpb.ActivityType{Name: "activity-type"}, StartToCloseTimeout: durationpb.New(2 * time.Second)}
	prepared := preparedRuntimeFixtureWithProfile(t, replySynchronous, authorizeRoutedActivityIn("ReplayNamespace", 1), routedActivityProgram(attributes, completeAttempt))
	host, definition := runtimeTestDriver(t, prepared)
	host.options.client = &recordingClient{}
	binding := delivery.WorkflowBinding{Namespace: "ReplayNamespace", WorkflowID: "replay-workflow", WorkflowType: "workflow-type", TaskQueue: "task-queue"}
	session, _, request := runtimeTestSessionWithBinding(t, host, definition, prepared, "run", "replay-run", binding, SessionOptions{Bridge: newTestBridge()})
	routed, err := host.admitWorkflow(workflowDelivery(request, "replay-run"))
	require.NoError(t, err)
	name, routeHeader, carriesRoute := session.preparedActivityHeader(routed.activation, "start")
	require.True(t, carriesRoute)

	dataConverter := converter.GetDefaultDataConverter()
	arguments, err := dataConverter.ToPayloads("untouched")
	require.NoError(t, err)
	answered, err := dataConverter.ToPayloads(&testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: "done"}})
	require.NoError(t, err)
	carried, err := anypb.New(answered.GetPayloads()[0])
	require.NoError(t, err)
	workflowResult, err := dataConverter.ToPayloads(&testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: carried}})
	require.NoError(t, err)
	task := func(id int64) []*historypb.HistoryEvent {
		return []*historypb.HistoryEvent{
			{EventId: id, EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED, Attributes: &historypb.HistoryEvent_WorkflowTaskScheduledEventAttributes{WorkflowTaskScheduledEventAttributes: &historypb.WorkflowTaskScheduledEventAttributes{}}},
			{EventId: id + 1, EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED, Attributes: &historypb.HistoryEvent_WorkflowTaskStartedEventAttributes{WorkflowTaskStartedEventAttributes: &historypb.WorkflowTaskStartedEventAttributes{ScheduledEventId: id}}},
			{EventId: id + 2, EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED, Attributes: &historypb.HistoryEvent_WorkflowTaskCompletedEventAttributes{WorkflowTaskCompletedEventAttributes: &historypb.WorkflowTaskCompletedEventAttributes{ScheduledEventId: id, StartedEventId: id + 1}}},
		}
	}
	history := []*historypb.HistoryEvent{{
		EventId: 1, EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
		Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{
			WorkflowType: &commonpb.WorkflowType{Name: binding.WorkflowType}, TaskQueue: &taskqueuepb.TaskQueue{Name: binding.TaskQueue},
			Input: arguments, Header: request.GetHeader(), OriginalExecutionRunId: "replay-run", WorkflowId: binding.WorkflowID,
		}},
	}}
	history = append(history, task(2)...)
	history = append(history,
		&historypb.HistoryEvent{EventId: 5, EventType: enumspb.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED, Attributes: &historypb.HistoryEvent_ActivityTaskScheduledEventAttributes{ActivityTaskScheduledEventAttributes: &historypb.ActivityTaskScheduledEventAttributes{
			ActivityId: "activity", ActivityType: &commonpb.ActivityType{Name: "activity-type"}, TaskQueue: &taskqueuepb.TaskQueue{Name: "task-queue"},
			Header: &commonpb.Header{Fields: map[string]*commonpb.Payload{name: routeHeader}}, WorkflowTaskCompletedEventId: 4,
			ScheduleToCloseTimeout: durationpb.New(time.Second), StartToCloseTimeout: durationpb.New(2 * time.Second),
		}}},
		// The Await bounds its wait by its own timeout.
		&historypb.HistoryEvent{EventId: 6, EventType: enumspb.EVENT_TYPE_TIMER_STARTED, Attributes: &historypb.HistoryEvent_TimerStartedEventAttributes{TimerStartedEventAttributes: &historypb.TimerStartedEventAttributes{TimerId: "6", StartToFireTimeout: durationpb.New(10 * time.Second), WorkflowTaskCompletedEventId: 4}}},
		&historypb.HistoryEvent{EventId: 7, EventType: enumspb.EVENT_TYPE_ACTIVITY_TASK_STARTED, Attributes: &historypb.HistoryEvent_ActivityTaskStartedEventAttributes{ActivityTaskStartedEventAttributes: &historypb.ActivityTaskStartedEventAttributes{ScheduledEventId: 5, Attempt: 1}}},
		&historypb.HistoryEvent{EventId: 8, EventType: enumspb.EVENT_TYPE_ACTIVITY_TASK_COMPLETED, Attributes: &historypb.HistoryEvent_ActivityTaskCompletedEventAttributes{ActivityTaskCompletedEventAttributes: &historypb.ActivityTaskCompletedEventAttributes{ScheduledEventId: 5, StartedEventId: 7, Result: answered}}},
	)
	history = append(history, task(9)...)
	// A history that records no SDK flags leaves the Await's timer running when its condition holds.
	history = append(history, &historypb.HistoryEvent{EventId: 12, EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED, Attributes: &historypb.HistoryEvent_WorkflowExecutionCompletedEventAttributes{WorkflowExecutionCompletedEventAttributes: &historypb.WorkflowExecutionCompletedEventAttributes{Result: workflowResult, WorkflowTaskCompletedEventId: 11}}})

	replayer, err := sdkworker.NewWorkflowReplayerWithOptions(sdkworker.WorkflowReplayerOptions{Interceptors: []interceptor.WorkerInterceptor{
		&sdkWorkerInterceptor{host: host, queue: binding.TaskQueue, registration: definition.registrations[0]},
	}})
	require.NoError(t, err)
	replayer.RegisterDynamicWorkflow(host.dynamicWorkflow, workflow.DynamicRegisterOptions{})
	options := sdkworker.ReplayWorkflowHistoryOptions{OriginalExecution: workflow.Execution{ID: binding.WorkflowID, RunID: "replay-run"}}
	require.NoError(t, replayer.ReplayWorkflowHistoryWithOptions(nil, &historypb.History{Events: history[:6]}, options))
	require.False(t, routed.admission.terminal)
	require.NoError(t, replayer.ReplayWorkflowHistoryWithOptions(nil, &historypb.History{Events: history}, options))
	require.True(t, routed.admission.terminal)
	workflowOutcome, err := reservationForEntrypoint(t, session, "workflow").Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, workflowOutcome.Outcome.GetStatus())
	require.Len(t, session.workflowAdmissions, 1)
}
