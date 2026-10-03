package worker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	sdkworker "go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport/facadetest"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
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
	profile.Roles = append(profile.Roles, testpilot.RolePolicy{ID: "activity-queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE})
	profile.EnvironmentBindings = append(profile.EnvironmentBindings, testpilot.EnvironmentBinding{ID: activityTaskQueue, Value: activityTaskQueue})
	for index := range profile.EnvironmentBindings {
		if profile.EnvironmentBindings[index].ID == "namespace" {
			profile.EnvironmentBindings[index].Value = "default-test-namespace"
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
