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

// DeriveProfile authorizes a read declaration's method on the role its poll names, and the
// ReadEvidence Opcode, so a Case that polls a read observation prepares under its derived Profile.
func TestDeriveProfileAuthorizesTheMethodAReadDeclarationPolls(t *testing.T) {
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	environment := temporal.Environment{Identity: "read", Namespace: "namespace", TaskQueue: "task-queue", NexusEndpoint: "endpoint"}
	const describe = "/temporal.api.workflowservice.v1.WorkflowService/DescribeWorkflowExecution"
	source := commandCase(&commandpb.Command{
		CommandType: enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION,
		Attributes: &commandpb.Command_ScheduleNexusOperationCommandAttributes{ScheduleNexusOperationCommandAttributes: &commandpb.ScheduleNexusOperationCommandAttributes{
			Endpoint: "nexus-endpoint", Service: "service", Operation: "operation", ScheduleToCloseTimeout: durationpb.New(2000000000),
		}},
	})
	source.Program.Roles = append(source.Program.Roles, &testpilotspb.Role{RoleId: "workflow-service", Kind: testpilotspb.ROLE_KIND_ENDPOINT})
	source.Program.Observations = []*testpilotspb.Observation{{ObservationId: "evidence", Type: &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: "temporal.server.api.testpilot.v1.CorrelatedEvidence"}}}}}}}
	source.Program.Evidence = []*testpilotspb.EvidenceDeclaration{{
		EvidenceId: "pendingAttempts", EvidenceSource: "describe", Operation: "scheduled_event_id",
		Source: &testpilotspb.EvidenceDeclaration_Read{Read: &testpilotspb.ReadSource{Method: describe, Path: "pending_nexus_operations"}},
		Fields: []*testpilotspb.EvidenceFieldDeclaration{{FieldId: "attempts", Path: "attempt"}},
	}}
	source.Program.Entrypoints = append(source.Program.Entrypoints, &testpilotspb.Entrypoint{
		EntrypointId: "controller", Activation: &testpilotspb.Entrypoint_Controller{Controller: &testpilotspb.ControllerActivation{}},
		Instructions: []*testpilotspb.InstructionNode{{InstructionId: "pending-attempts", Limits: &testpilotspb.InstructionLimits{Timeout: &testpilotspb.InstructionLimits_TimeoutMilliseconds{TimeoutMilliseconds: 1000}, Attempts: &testpilotspb.InstructionLimits_MaxAttempts{MaxAttempts: 1}}, Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ReadEvidence{ReadEvidence: &testpilotspb.ReadEvidence{
			EvidenceId: "pendingAttempts", EndpointRoleId: "workflow-service", PollIntervalMilliseconds: 100,
			Until: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Present{Present: &testpilotspb.PresentExpression{Operand: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Path{Path: &testpilotspb.PathExpression{Operand: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_ProjectedValue{ProjectedValue: &testpilotspb.ProjectedValueReference{}}}}}, Path: "attempt"}}}}}},
		}}}}},
	})
	profile, err := temporal.DeriveProfile(source, catalog, environment)
	require.NoError(t, err)
	require.Contains(t, profile.Opcodes, testpilot.ReadEvidence)
	var methods []string
	for _, role := range profile.Roles {
		if role.ID == "workflow-service" {
			methods = role.Methods
		}
	}
	require.Equal(t, []string{describe}, methods)
	_, err = testpilot.Prepare(source, profile)
	require.NoError(t, err)

	// A poll of a kind the Program does not declare as a read has no method to authorize.
	source.Program.Evidence[0].Source = &testpilotspb.EvidenceDeclaration_HistoryEvent{HistoryEvent: &testpilotspb.HistoryEventSource{AttributesField: "nexus_operation_started_event_attributes"}}
	_, err = temporal.DeriveProfile(source, catalog, environment)
	require.ErrorIs(t, err, temporal.ErrInvalid)
}
