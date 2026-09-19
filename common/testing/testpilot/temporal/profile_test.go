package temporal_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	commandpb "go.temporal.io/api/command/v1"
	enumspb "go.temporal.io/api/enums/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/common/testing/testpilot/temporal/worker"
	"google.golang.org/protobuf/types/known/durationpb"
)

func commandCase(command *commandpb.Command) *testpilotspb.Case {
	limits := &testpilotspb.InstructionLimits{Timeout: &testpilotspb.InstructionLimits_TimeoutMilliseconds{TimeoutMilliseconds: 1000}, Attempts: &testpilotspb.InstructionLimits_MaxAttempts{MaxAttempts: 1}}
	return &testpilotspb.Case{
		Version: &testpilotspb.FormatVersion{Major: 1}, CaseId: "temporal.case.command",
		Program: &testpilotspb.Program{
			ProgramId: "program",
			Roles: []*testpilotspb.Role{
				{RoleId: "worker", Kind: testpilotspb.ROLE_KIND_WORKER, NamespaceBindingId: "namespace"},
				{RoleId: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingId: "namespace", ResourceBindingId: "task-queue"},
				{RoleId: "nexus-endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, ResourceBindingId: "nexus-endpoint"},
			},
			Entrypoints: []*testpilotspb.Entrypoint{{
				EntrypointId: "workflow",
				Activation:   &testpilotspb.Entrypoint_Workflow{Workflow: &testpilotspb.WorkflowActivation{WorkflowType: "workflow-type", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}},
				Instructions: []*testpilotspb.InstructionNode{
					{InstructionId: "command", Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_WorkflowCommand{WorkflowCommand: &testpilotspb.WorkflowCommand{Command: command}}}, Limits: limits},
					{InstructionId: "finish", Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: "done"}}}}}}}, Limits: limits},
				},
			}},
			Cleanup: &testpilotspb.Cleanup{EntrypointId: "cleanup"},
		},
		Contract: &testpilotspb.Contract{
			ContractId: "contract",
			Rules: []*testpilotspb.ContractRule{{
				RuleId: "complete", Kind: testpilotspb.CONTRACT_RULE_KIND_SAFETY, InitialStateId: "open",
				States: []*testpilotspb.ContractState{
					{StateId: "open", Status: testpilotspb.CONTRACT_STATE_STATUS_PENDING},
					{StateId: "closed", Status: testpilotspb.CONTRACT_STATE_STATUS_SATISFIED},
				},
				Transitions: []*testpilotspb.ContractTransition{{
					TransitionId: "close", SourceStateId: "open", TargetStateId: "closed",
					EventFilter: &testpilotspb.RunEventFilter{Kinds: []testpilotspb.RunEventKind{testpilotspb.RUN_EVENT_KIND_RUN_CLOSED}},
					Predicate:   &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: true}}}},
					SupportKind: testpilotspb.CONTRACT_SUPPORT_KIND_MATCHING_EVENT,
				}},
			}},
		},
	}
}

// DeriveProfile admits exactly the command types the Case carries that the worker Driver
// realizes: a schedule command is admitted, and a command type the Driver cannot realize is left
// out, so the Case rejects at preparation as one the Profile does not admit.
func TestDeriveProfileAdmitsOnlyRealizableCommandTypes(t *testing.T) {
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	environment := temporal.Environment{Identity: "command", Namespace: "namespace", TaskQueue: "task-queue", NexusEndpoint: "endpoint"}

	schedule := commandCase(&commandpb.Command{
		CommandType: enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION,
		Attributes: &commandpb.Command_ScheduleNexusOperationCommandAttributes{ScheduleNexusOperationCommandAttributes: &commandpb.ScheduleNexusOperationCommandAttributes{
			Endpoint: "nexus-endpoint", Service: "service", Operation: "operation", ScheduleToCloseTimeout: durationpb.New(2000000000),
		}},
	})
	profile, err := temporal.DeriveProfile(schedule, catalog, environment)
	require.NoError(t, err)
	require.Equal(t, []enumspb.CommandType{enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION}, profile.CommandTypes)
	require.Contains(t, profile.Opcodes, testpilot.WorkflowCommand)
	_, err = testpilot.Prepare(schedule, profile)
	require.NoError(t, err)

	timer := commandCase(&commandpb.Command{
		CommandType: enumspb.COMMAND_TYPE_START_TIMER,
		Attributes:  &commandpb.Command_StartTimerCommandAttributes{StartTimerCommandAttributes: &commandpb.StartTimerCommandAttributes{TimerId: "timer", StartToFireTimeout: durationpb.New(1000000000)}},
	})
	profile, err = temporal.DeriveProfile(timer, catalog, environment)
	require.NoError(t, err)
	require.Empty(t, profile.CommandTypes)
	_, err = testpilot.Prepare(timer, profile)
	var diagnostic *testpilot.PreparationError
	require.ErrorAs(t, err, &diagnostic)
	require.Equal(t, testpilot.PreparationErrorCategory("unsupported"), diagnostic.Category)
	require.Equal(t, "program.entrypoints[workflow].instructions[command].instruction.workflow_command.command.command_type", diagnostic.Path)

	// Every command type the worker realizes is a declared one.
	for _, commandType := range worker.CommandTypes() {
		_, declared := enumspb.CommandType_name[int32(commandType)]
		require.True(t, declared)
		require.NotEqual(t, enumspb.COMMAND_TYPE_UNSPECIFIED, commandType)
	}
}
