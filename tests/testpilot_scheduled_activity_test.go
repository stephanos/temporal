//go:build test_dep && integration

package tests

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
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

const (
	scheduledActivityNamespaceBinding = "temporal.worker.namespace"
	scheduledActivityQueueBinding     = "temporal.task-queue.resource"
	scheduledActivityStartMethod      = "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"
	scheduledActivityHistoryMethod    = "/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory"
)

func scheduledActivityReference(reference *testpilotpb.Reference) *testpilotpb.Expression {
	return &testpilotpb.Expression{Expression: &testpilotpb.Expression_Reference{Reference: reference}}
}

func scheduledActivityLiteral(value *testpilotpb.Value) *testpilotpb.Expression {
	return &testpilotpb.Expression{Expression: &testpilotpb.Expression_Literal{Literal: value}}
}

// scheduledActivityCase is a hand-built Case with no Model behind it: a controller starts a
// workflow whose Driver entrypoint schedules an activity, awaits it and finishes with the awaited
// outcome field, and the activity's attempts are the Driver's activity entrypoint script. The
// controller then waits for the workflow to close. The Contract asks only that the Run closes.
func scheduledActivityCase(attributes *commandpb.ScheduleActivityTaskCommandAttributes, finishWith testpilotpb.InstructionOutcomeField, script ...*testpilotpb.Instruction) *testpilotpb.Case {
	namespace := scheduledActivityReference(&testpilotpb.Reference{Reference: &testpilotpb.Reference_EnvironmentBindingId{EnvironmentBindingId: scheduledActivityNamespaceBinding}})
	queue := scheduledActivityReference(&testpilotpb.Reference{Reference: &testpilotpb.Reference_EnvironmentBindingId{EnvironmentBindingId: scheduledActivityQueueBinding}})
	runID := scheduledActivityReference(&testpilotpb.Reference{Reference: &testpilotpb.Reference_Run{Run: &testpilotpb.RunReference{}}})
	alwaysRuns := scheduledActivityLiteral(&testpilotpb.Value{Value: &testpilotpb.Value_BoolValue{BoolValue: true}})
	limits := &testpilotpb.InstructionLimits{Timeout: &testpilotpb.InstructionLimits_TimeoutMilliseconds{TimeoutMilliseconds: 20000}}
	// The awaited value is present only once the Await succeeded; its status always is.
	finishGuard := alwaysRuns
	if finishWith == testpilotpb.INSTRUCTION_OUTCOME_FIELD_VALUE {
		finishGuard = &testpilotpb.Expression{Expression: &testpilotpb.Expression_Compare{Compare: &testpilotpb.CompareExpression{
			Operator: testpilotpb.COMPARISON_OPERATOR_EQUAL,
			Left: scheduledActivityReference(&testpilotpb.Reference{Reference: &testpilotpb.Reference_Outcome{Outcome: &testpilotpb.InstructionOutcomeReference{
				Instruction: &testpilotpb.InstructionReference{EntrypointId: "workflow", InstructionId: "await"}, Field: testpilotpb.INSTRUCTION_OUTCOME_FIELD_STATUS,
			}}}),
			Right: scheduledActivityLiteral(&testpilotpb.Value{Value: &testpilotpb.Value_EnumValue{EnumValue: &testpilotpb.EnumValue{Name: "INSTRUCTION_OUTCOME_STATUS_SUCCEEDED"}}}),
		}}}
	}
	attributes.ActivityType = &commonpb.ActivityType{Name: "scheduled-activity-type"}
	attributes.TaskQueue = &taskqueuepb.TaskQueue{Name: "temporal.task-queue"}
	activity := &testpilotpb.Entrypoint{
		EntrypointId: "activity",
		Activation:   &testpilotpb.Entrypoint_Activity{Activity: &testpilotpb.ActivityActivation{ActivityType: "scheduled-activity-type", WorkerRoleId: "temporal.worker", TaskQueueRoleId: "temporal.task-queue"}},
	}
	for index, instruction := range script {
		activity.Instructions = append(activity.Instructions, &testpilotpb.InstructionNode{InstructionId: fmt.Sprintf("attempt-%d", index+1), Instruction: instruction, Limits: limits})
	}
	return &testpilotpb.Case{
		Version: &testpilotpb.FormatVersion{Major: 1},
		CaseId:  "scheduled-activity",
		Program: &testpilotpb.Program{
			ProgramId: "scheduled-activity",
			Roles: []*testpilotpb.Role{
				{RoleId: "temporal.workflow-service", Kind: testpilotpb.ROLE_KIND_ENDPOINT},
				{RoleId: "temporal.worker", Kind: testpilotpb.ROLE_KIND_WORKER, NamespaceBindingId: scheduledActivityNamespaceBinding},
				{RoleId: "temporal.task-queue", Kind: testpilotpb.ROLE_KIND_TASK_QUEUE, NamespaceBindingId: scheduledActivityNamespaceBinding, ResourceBindingId: scheduledActivityQueueBinding},
			},
			Entrypoints: []*testpilotpb.Entrypoint{
				{
					EntrypointId: "controller", Activation: &testpilotpb.Entrypoint_Controller{Controller: &testpilotpb.ControllerActivation{}},
					Instructions: []*testpilotpb.InstructionNode{
						{InstructionId: "start-workflow", Instruction: &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_InvokeRpc{InvokeRpc: &testpilotpb.InvokeRpc{
							EndpointRoleId: "temporal.workflow-service", Method: scheduledActivityStartMethod,
							RequestAssignments: []*testpilotpb.RequestAssignment{
								{Target: "namespace", Value: namespace},
								{Target: "workflow_id", Value: runID},
								{Target: "workflow_type.name", Value: scheduledActivityLiteral(&testpilotpb.Value{Value: &testpilotpb.Value_TextValue{TextValue: "scheduled-activity-workflow"}})},
								{Target: "task_queue.name", Value: queue},
							},
						}}}},
						{InstructionId: "await-close", Limits: limits, Instruction: &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_InvokeRpc{InvokeRpc: &testpilotpb.InvokeRpc{
							EndpointRoleId: "temporal.workflow-service", Method: scheduledActivityHistoryMethod,
							RequestAssignments: []*testpilotpb.RequestAssignment{
								{Target: "namespace", Value: namespace},
								{Target: "execution.workflow_id", Value: runID},
								{Target: "wait_new_event", Value: scheduledActivityLiteral(&testpilotpb.Value{Value: &testpilotpb.Value_BoolValue{BoolValue: true}})},
								{Target: "history_event_filter_type", Value: scheduledActivityLiteral(&testpilotpb.Value{Value: &testpilotpb.Value_EnumValue{EnumValue: &testpilotpb.EnumValue{Name: "HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT"}}})},
							},
						}}}},
					},
				},
				{
					EntrypointId: "workflow",
					Activation:   &testpilotpb.Entrypoint_Workflow{Workflow: &testpilotpb.WorkflowActivation{WorkflowType: "scheduled-activity-workflow", WorkerRoleId: "temporal.worker", TaskQueueRoleId: "temporal.task-queue"}},
					Instructions: []*testpilotpb.InstructionNode{
						{InstructionId: "schedule", Limits: limits, Instruction: &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_WorkflowCommand{WorkflowCommand: &testpilotpb.WorkflowCommand{Command: &commandpb.Command{
							CommandType: enumspb.COMMAND_TYPE_SCHEDULE_ACTIVITY_TASK,
							Attributes:  &commandpb.Command_ScheduleActivityTaskCommandAttributes{ScheduleActivityTaskCommandAttributes: attributes},
						}}}}},
						{InstructionId: "await", Limits: limits, Guard: alwaysRuns, Instruction: &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_AwaitInstruction{AwaitInstruction: &testpilotpb.AwaitInstruction{
							Instruction: &testpilotpb.InstructionReference{EntrypointId: "workflow", InstructionId: "schedule"},
						}}}},
						{InstructionId: "finish", Limits: limits, Guard: finishGuard, Instruction: &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_Finish{Finish: &testpilotpb.Finish{
							Result: scheduledActivityReference(&testpilotpb.Reference{Reference: &testpilotpb.Reference_Outcome{Outcome: &testpilotpb.InstructionOutcomeReference{
								Instruction: &testpilotpb.InstructionReference{EntrypointId: "workflow", InstructionId: "await"}, Field: finishWith,
							}}}),
						}}}},
					},
				},
				activity,
			},
			Cleanup: &testpilotpb.Cleanup{EntrypointId: "cleanup"},
		},
		Contract: &testpilotpb.Contract{
			ContractId: "scheduled-activity",
			Rules: []*testpilotpb.ContractRule{{
				RuleId: "closed", Kind: testpilotpb.CONTRACT_RULE_KIND_SAFETY, InitialStateId: "open",
				States: []*testpilotpb.ContractState{
					{StateId: "open", Status: testpilotpb.CONTRACT_STATE_STATUS_PENDING},
					{StateId: "closed", Status: testpilotpb.CONTRACT_STATE_STATUS_SATISFIED},
				},
				Transitions: []*testpilotpb.ContractTransition{{
					TransitionId: "close", SourceStateId: "open", TargetStateId: "closed",
					EventFilter: &testpilotpb.RunEventFilter{Kinds: []testpilotpb.RunEventKind{testpilotpb.RUN_EVENT_KIND_RUN_CLOSED}},
					Predicate:   alwaysRuns,
					SupportKind: testpilotpb.CONTRACT_SUPPORT_KIND_MATCHING_EVENT,
				}},
			}},
		},
	}
}

// TestTestpilotScheduledActivityAttempts drives the Driver's generic workflow-scheduled activity
// primitives against the in-process server with hand-built Cases: the Driver's workflow schedules
// an activity, awaits it and finishes, and each attempt the server issues is answered by the
// activity entrypoint's script, completing, failing retryably then completing, or withholding its
// answer until the server times the attempt out.
func TestTestpilotScheduledActivityAttempts(t *testing.T) {
	env := newTestpilotTestEnvironment(t)
	completes := &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_Finish{Finish: &testpilotpb.Finish{Result: scheduledActivityLiteral(&testpilotpb.Value{Value: &testpilotpb.Value_TextValue{TextValue: "done"}})}}}
	fails := &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_ActivityAttemptFailure{ActivityAttemptFailure: &testpilotpb.ActivityAttemptFailure{
		Failure: &failurepb.Failure{Message: "not yet", FailureInfo: &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{Type: "transient"}}},
	}}}
	withholds := &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_ActivityAttemptWithholding{ActivityAttemptWithholding: &testpilotpb.ActivityAttemptWithholding{}}}
	retried := func(maximumAttempts int32, startToClose time.Duration) *commandpb.ScheduleActivityTaskCommandAttributes {
		return &commandpb.ScheduleActivityTaskCommandAttributes{
			ScheduleToCloseTimeout: durationpb.New(15 * time.Second), StartToCloseTimeout: durationpb.New(startToClose),
			RetryPolicy: &commonpb.RetryPolicy{InitialInterval: durationpb.New(time.Second), BackoffCoefficient: 1, MaximumAttempts: maximumAttempts},
		}
	}
	for _, test := range []struct {
		name       string
		source     *testpilotpb.Case
		responses  []testpilotpb.ActivityAttemptResponse
		closedWith enumspb.EventType
	}{
		{
			name:       "complete",
			source:     scheduledActivityCase(retried(2, 5*time.Second), testpilotpb.INSTRUCTION_OUTCOME_FIELD_VALUE, completes, fails),
			responses:  []testpilotpb.ActivityAttemptResponse{testpilotpb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED, testpilotpb.ACTIVITY_ATTEMPT_RESPONSE_NOT_NEEDED},
			closedWith: enumspb.EVENT_TYPE_ACTIVITY_TASK_COMPLETED,
		},
		{
			name:       "retry then complete",
			source:     scheduledActivityCase(retried(2, 5*time.Second), testpilotpb.INSTRUCTION_OUTCOME_FIELD_VALUE, fails, completes),
			responses:  []testpilotpb.ActivityAttemptResponse{testpilotpb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_RETRYABLE, testpilotpb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED},
			closedWith: enumspb.EVENT_TYPE_ACTIVITY_TASK_COMPLETED,
		},
		{
			name:       "withheld until timed out",
			source:     scheduledActivityCase(retried(1, 2*time.Second), testpilotpb.INSTRUCTION_OUTCOME_FIELD_STATUS, withholds),
			responses:  []testpilotpb.ActivityAttemptResponse{testpilotpb.ACTIVITY_ATTEMPT_RESPONSE_WITHHELD},
			closedWith: enumspb.EVENT_TYPE_ACTIVITY_TASK_TIMED_OUT,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			name := "umpire-scheduled-activity-" + map[string]string{"complete": "complete", "retry then complete": "retry", "withheld until timed out": "withheld"}[test.name]
			live := bindCase(t, env, test.source, CaseBinding{Identity: name, Namespace: name, TaskQueue: name})
			ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
			defer cancel()
			run, verdict, err := live.prepared.Run(ctx, live.driver)
			require.NoError(t, err)
			require.Equal(t, testpilotpb.RUN_DISPOSITION_COMPLETED, run.GetDisposition(), run.GetDiagnostics())
			require.Equal(t, testpilotpb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())

			var responses []testpilotpb.ActivityAttemptResponse
			for _, event := range run.GetEvents() {
				if attempt := event.GetOutcome().GetActivityAttempt(); attempt != nil {
					require.NotEmpty(t, attempt.GetActivityRunId())
					responses = append(responses, attempt.GetResponse())
				}
			}
			require.Equal(t, test.responses, responses)

			// The history is the server's own record of what the Driver's workflow did.
			var closed []enumspb.EventType
			iterator := live.client.GetWorkflowHistory(ctx, run.GetRunId(), "", false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
			var completion *commonpb.Payloads
			for iterator.HasNext() {
				event, err := iterator.Next()
				require.NoError(t, err)
				switch event.GetEventType() {
				case enumspb.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED:
					require.Equal(t, "scheduled-activity-type", event.GetActivityTaskScheduledEventAttributes().GetActivityType().GetName())
					require.Equal(t, name, event.GetActivityTaskScheduledEventAttributes().GetTaskQueue().GetName())
				case enumspb.EVENT_TYPE_ACTIVITY_TASK_COMPLETED, enumspb.EVENT_TYPE_ACTIVITY_TASK_TIMED_OUT, enumspb.EVENT_TYPE_ACTIVITY_TASK_FAILED:
					closed = append(closed, event.GetEventType())
				case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED:
					completion = event.GetWorkflowExecutionCompletedEventAttributes().GetResult()
				default:
				}
			}
			require.Equal(t, []enumspb.EventType{test.closedWith}, closed)
			require.NotNil(t, completion, "the Driver's workflow completed")
		})
	}
}
