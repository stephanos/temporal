package temporal_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport/facadetest"
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
// realizes: a Nexus or activity schedule command is admitted, and a command type the Driver cannot realize is left
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

	activity := commandCase(&commandpb.Command{
		CommandType: enumspb.COMMAND_TYPE_SCHEDULE_ACTIVITY_TASK,
		Attributes: &commandpb.Command_ScheduleActivityTaskCommandAttributes{ScheduleActivityTaskCommandAttributes: &commandpb.ScheduleActivityTaskCommandAttributes{
			ActivityType: &commonpb.ActivityType{Name: "activity-type"}, TaskQueue: &taskqueuepb.TaskQueue{Name: "queue"}, StartToCloseTimeout: durationpb.New(1000000000),
		}},
	})
	profile, err = temporal.DeriveProfile(activity, catalog, environment)
	require.NoError(t, err)
	require.Equal(t, []enumspb.CommandType{enumspb.COMMAND_TYPE_SCHEDULE_ACTIVITY_TASK}, profile.CommandTypes)
	_, err = testpilot.Prepare(activity, profile)
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

const (
	startWorkflowExecution = "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"
	startActivityExecution = "/temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution"
)

func environmentReference(id string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_EnvironmentBindingId{EnvironmentBindingId: id}}}}
}

func textLiteral(value string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: value}}}}
}

// startCall is a controller call of method on the workflow service that names the bound namespace
// and task queue, as a carried start must.
func startCall(id, method string, assignments ...*testpilotspb.RequestAssignment) *testpilotspb.InstructionNode {
	return &testpilotspb.InstructionNode{
		InstructionId: id,
		Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRpc{EndpointRoleId: "workflow-service", Method: method, RequestAssignments: append([]*testpilotspb.RequestAssignment{
			{Target: "namespace", Value: environmentReference("namespace")},
			{Target: "task_queue.name", Value: environmentReference("task-queue")},
		}, assignments...)}}},
	}
}

// activityCase is commandCase's workflow beside a standalone activity: a controller starts the
// workflow, then the activity, whose entrypoint is the one Finish an activity script lowers to.
func activityCase() *testpilotspb.Case {
	source := commandCase(&commandpb.Command{
		CommandType: enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION,
		Attributes: &commandpb.Command_ScheduleNexusOperationCommandAttributes{ScheduleNexusOperationCommandAttributes: &commandpb.ScheduleNexusOperationCommandAttributes{
			Endpoint: "nexus-endpoint", Service: "service", Operation: "operation", ScheduleToCloseTimeout: durationpb.New(2000000000),
		}},
	})
	source.Program.Roles = append(source.Program.Roles, &testpilotspb.Role{RoleId: "workflow-service", Kind: testpilotspb.ROLE_KIND_ENDPOINT})
	handler := &testpilotspb.Entrypoint{
		EntrypointId: "handler",
		Activation:   &testpilotspb.Entrypoint_NexusHandler{NexusHandler: &testpilotspb.NexusHandlerActivation{Service: "service", Operation: "operation", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}},
	}
	activity := &testpilotspb.Entrypoint{
		EntrypointId: "activity",
		Activation:   &testpilotspb.Entrypoint_Activity{Activity: &testpilotspb.ActivityActivation{ActivityType: "activity-type", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}},
		Instructions: []*testpilotspb.InstructionNode{{InstructionId: "run-attempt", Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: textLiteral("done")}}}}},
	}
	controller := &testpilotspb.Entrypoint{
		EntrypointId: "controller", Activation: &testpilotspb.Entrypoint_Controller{Controller: &testpilotspb.ControllerActivation{}},
		Instructions: []*testpilotspb.InstructionNode{
			startCall("start-workflow", startWorkflowExecution, &testpilotspb.RequestAssignment{Target: "workflow_type.name", Value: textLiteral("workflow-type")}, &testpilotspb.RequestAssignment{Target: "workflow_id", Value: textLiteral("workflow-id")}),
			startCall("start-activity", startActivityExecution, &testpilotspb.RequestAssignment{Target: "activity_type.name", Value: textLiteral("activity-type")}, &testpilotspb.RequestAssignment{Target: "activity_id", Value: textLiteral("activity-id")}),
		},
	}
	source.Program.Entrypoints = []*testpilotspb.Entrypoint{controller, source.Program.Entrypoints[0], handler, activity}
	return source
}

func endpointPolicy(t *testing.T, profile testpilot.ProfileSpec) testpilot.RolePolicy {
	t.Helper()
	for _, role := range profile.Roles {
		if role.ID == "workflow-service" {
			return role
		}
	}
	require.FailNow(t, "the derived Profile has no workflow-service role")
	return testpilot.RolePolicy{}
}

// DeriveProfile names each start the carrier of what the Temporal Driver delivers through it: a
// workflow start carries the workflow and the Nexus handlers it reaches, exactly as it did before a
// Program could run an activity, and an activity start carries the activity alone.
func TestDeriveProfileNamesEachStartTheCarrierOfWhatItActivates(t *testing.T) {
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	environment := temporal.Environment{Identity: "activity", Namespace: "namespace", TaskQueue: "task-queue", NexusEndpoint: "endpoint"}
	workflowCarrier := testpilot.ReservationCarrierPolicy{Method: startWorkflowExecution, Shapes: []testpilot.ReservationCarrierShape{{Kind: testpilot.WorkflowEntrypoint, MaximumCount: 1}, {Kind: testpilot.NexusHandlerEntrypoint, MaximumCount: 1}}}
	activityCarrier := testpilot.ReservationCarrierPolicy{Method: startActivityExecution, Shapes: []testpilot.ReservationCarrierShape{{Kind: testpilot.ActivityEntrypoint, MaximumCount: 1}}}
	// An activity is activated once per attempt its script declares, so its carrier admits that many.
	retriedCarrier := testpilot.ReservationCarrierPolicy{Method: startActivityExecution, Shapes: []testpilot.ReservationCarrierShape{{Kind: testpilot.ActivityEntrypoint, MaximumCount: 2}}}
	retried := func(program *testpilotspb.Program) {
		script := program.Entrypoints[3]
		script.Instructions = append([]*testpilotspb.InstructionNode{attemptFailure("first-attempt", "transient", false)}, script.Instructions...)
	}

	for name, test := range map[string]struct {
		mutate func(*testpilotspb.Program)
		want   testpilot.RolePolicy
		plans  map[string]testpilot.ReservationCarrierPlan
	}{
		"a workflow and a standalone activity": {
			mutate: func(*testpilotspb.Program) {},
			want: testpilot.RolePolicy{ID: "workflow-service", Kind: testpilotspb.ROLE_KIND_ENDPOINT,
				Methods:             []string{startWorkflowExecution, startActivityExecution},
				ReservationCarriers: []testpilot.ReservationCarrierPolicy{workflowCarrier, activityCarrier}},
			plans: map[string]testpilot.ReservationCarrierPlan{
				"start-workflow": {EndpointRoleID: "workflow-service", Method: startWorkflowExecution,
					Reservations: []testpilot.ReservationTopology{{EntrypointID: "workflow", Kind: testpilot.WorkflowEntrypoint, Count: 1}, {EntrypointID: "handler", Kind: testpilot.NexusHandlerEntrypoint, Count: 1}},
					Routes:       []testpilot.ReservationRoute{{WorkflowEntrypointID: "workflow", SourceInstructionID: "command", HandlerEntrypointID: "handler"}}},
				"start-activity": {EndpointRoleID: "workflow-service", Method: startActivityExecution,
					Reservations: []testpilot.ReservationTopology{{EntrypointID: "activity", Kind: testpilot.ActivityEntrypoint, Count: 1}}},
			},
		},
		"an activity whose script declares a retry": {
			mutate: retried,
			want: testpilot.RolePolicy{ID: "workflow-service", Kind: testpilotspb.ROLE_KIND_ENDPOINT,
				Methods:             []string{startWorkflowExecution, startActivityExecution},
				ReservationCarriers: []testpilot.ReservationCarrierPolicy{workflowCarrier, retriedCarrier}},
			plans: map[string]testpilot.ReservationCarrierPlan{
				"start-workflow": {EndpointRoleID: "workflow-service", Method: startWorkflowExecution,
					Reservations: []testpilot.ReservationTopology{{EntrypointID: "workflow", Kind: testpilot.WorkflowEntrypoint, Count: 1}, {EntrypointID: "handler", Kind: testpilot.NexusHandlerEntrypoint, Count: 1}},
					Routes:       []testpilot.ReservationRoute{{WorkflowEntrypointID: "workflow", SourceInstructionID: "command", HandlerEntrypointID: "handler"}}},
				"start-activity": {EndpointRoleID: "workflow-service", Method: startActivityExecution,
					Reservations: []testpilot.ReservationTopology{{EntrypointID: "activity", Kind: testpilot.ActivityEntrypoint, Count: 2}}},
			},
		},
		"the workflow alone, as before": {
			mutate: func(program *testpilotspb.Program) {
				program.Entrypoints[0].Instructions = program.Entrypoints[0].Instructions[:1]
				program.Entrypoints = program.Entrypoints[:3]
			},
			want: testpilot.RolePolicy{ID: "workflow-service", Kind: testpilotspb.ROLE_KIND_ENDPOINT,
				Methods:             []string{startWorkflowExecution},
				ReservationCarriers: []testpilot.ReservationCarrierPolicy{workflowCarrier}},
			plans: map[string]testpilot.ReservationCarrierPlan{
				"start-workflow": {EndpointRoleID: "workflow-service", Method: startWorkflowExecution,
					Reservations: []testpilot.ReservationTopology{{EntrypointID: "workflow", Kind: testpilot.WorkflowEntrypoint, Count: 1}, {EntrypointID: "handler", Kind: testpilot.NexusHandlerEntrypoint, Count: 1}},
					Routes:       []testpilot.ReservationRoute{{WorkflowEntrypointID: "workflow", SourceInstructionID: "command", HandlerEntrypointID: "handler"}}},
			},
		},
		// The Case starts an activity it runs no script for, so nothing is reserved and the start
		// stays an ordinary call.
		"an activity the Case only observes": {
			mutate: func(program *testpilotspb.Program) {
				program.Entrypoints[0].Instructions = program.Entrypoints[0].Instructions[1:]
				program.Entrypoints = program.Entrypoints[:1]
				program.Roles = program.Roles[3:]
			},
			want:  testpilot.RolePolicy{ID: "workflow-service", Kind: testpilotspb.ROLE_KIND_ENDPOINT, Methods: []string{startActivityExecution}},
			plans: map[string]testpilot.ReservationCarrierPlan{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			source := activityCase()
			test.mutate(source.Program)
			if name == "an activity the Case only observes" {
				for _, assignment := range source.Program.Entrypoints[0].Instructions[0].GetInstruction().GetInvokeRpc().RequestAssignments[:2] {
					assignment.Value = textLiteral("literal")
				}
			}
			profile, err := temporal.DeriveProfile(source, catalog, environment)
			require.NoError(t, err)
			require.Equal(t, test.want, endpointPolicy(t, profile))
			// Failing an attempt is its own instruction, authorized only where the Case uses it.
			require.Equal(t, name == "an activity whose script declares a retry", slices.Contains(profile.Opcodes, testpilot.ActivityAttemptFailure))
			prepared, err := testpilot.Prepare(source, profile)
			require.NoError(t, err)
			program := facadetest.Capture(t, prepared)
			plans := map[string]testpilot.ReservationCarrierPlan{}
			for _, instruction := range source.Program.Entrypoints[0].Instructions {
				if plan, carried := program.ReservationCarrier("controller", instruction.InstructionId); carried {
					plans[instruction.InstructionId] = plan
				}
			}
			require.Equal(t, test.plans, plans)
		})
	}
}

// A Profile that authorizes the activity start as an ordinary call but names it no carrier, as a
// consumer without an SDK activity worker would, cannot activate the Case's activity script, and
// preparation says so before any Driver exists.
func TestPrepareRejectsAnActivityScriptUnderAProfileWithoutItsCarrier(t *testing.T) {
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	source := activityCase()
	profile, err := temporal.DeriveProfile(source, catalog, temporal.Environment{Identity: "activity", Namespace: "namespace", TaskQueue: "task-queue", NexusEndpoint: "endpoint"})
	require.NoError(t, err)
	for index, role := range profile.Roles {
		if role.ID == "workflow-service" {
			profile.Roles[index].ReservationCarriers = role.ReservationCarriers[:1]
		}
	}
	_, err = testpilot.Prepare(source, profile)
	var diagnostic *testpilot.PreparationError
	require.ErrorAs(t, err, &diagnostic)
	require.Equal(t, testpilot.PreparationUnavailable, diagnostic.Category)
	require.Equal(t, "activity", diagnostic.Path)
}
