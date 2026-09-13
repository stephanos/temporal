package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func preparedRuntimeFixture(t *testing.T, responseKind testpilotspb.NexusResponseKind, modify ...func(*testpilotspb.Program)) testpilot.PreparedProgram {
	return preparedRuntimeFixtureWithProfile(t, responseKind, nil, modify...)
}

func preparedRuntimeFixtureForNamespace(t *testing.T, namespace string, responseKind testpilotspb.NexusResponseKind, modify ...func(*testpilotspb.Program)) testpilot.PreparedProgram {
	return preparedRuntimeFixtureWithProfile(t, responseKind, func(profile *testpilot.ProfileSpec) {
		for index := range profile.EnvironmentBindings {
			if profile.EnvironmentBindings[index].ID == "namespace" {
				profile.EnvironmentBindings[index].Value = namespace
			}
		}
	}, modify...)
}

func preparedRuntimeFixtureWithProfile(t *testing.T, responseKind testpilotspb.NexusResponseKind, modifyProfile func(*testpilot.ProfileSpec), modify ...func(*testpilotspb.Program)) testpilot.PreparedProgram {
	t.Helper()
	return capturePreparedProgram(t, preparedRuntimeCase(t, responseKind, modifyProfile, modify...))
}

func preparedRuntimeCase(t *testing.T, responseKind testpilotspb.NexusResponseKind, modifyProfile func(*testpilot.ProfileSpec), modify ...func(*testpilotspb.Program)) *testpilot.PreparedCase {
	t.Helper()
	file := workflowservice.File_temporal_api_workflowservice_v1_service_proto
	catalog, err := testpilot.NewCatalog(descriptorClosure(file))
	require.NoError(t, err)
	limits := &testpilotspb.ProgramLimits{MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 16, MaxAttempts: 16, MaxRunEvents: 16, MaxExpressionDepth: 16, MaxPathFanout: 32, MaxRequestBytes: 64 << 10, MaxResponseBytes: 64 << 10, MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 5000}
	method := "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"
	contractLimits := &testpilotspb.ContractLimits{MaxRules: 8, MaxStates: 16, MaxTransitions: 16, MaxExpressionDepth: 16, MaxWorkPerEvent: 100000, MaxTotalWork: 1000000000, MaxCaptures: 8, MaxCaptureBytes: 65536}
	profile := testpilot.ProfileSpec{
		Identity: "profile", Catalog: catalog,
		Roles: []testpilot.RolePolicy{
			{ID: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, Methods: []string{method}, ReservationCarriers: []testpilot.ReservationCarrierPolicy{{Method: method, Shapes: []testpilot.ReservationCarrierShape{{Kind: testpilotspb.ENTRYPOINT_KIND_WORKFLOW, MaximumCount: 8}, {Kind: testpilotspb.ENTRYPOINT_KIND_NEXUS_HANDLER, MaximumCount: 8}}}}},
			{ID: "worker", Kind: testpilotspb.ROLE_KIND_WORKER},
			{ID: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE},
			{ID: "nexus-endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT},
		},
		Opcodes: []testpilot.Opcode{testpilot.InvokeRPC, testpilot.StartNexusOperation, testpilot.Await, testpilot.Finish, testpilot.RespondNexus},
		EnvironmentBindings: []testpilot.EnvironmentBinding{
			{ID: "namespace", Value: "namespace"}, {ID: "task-queue", Value: "task-queue"}, {ID: "nexus-endpoint", Value: "endpoint"},
		},
		ProgramLimits: proto.CloneOf(limits), ContractLimits: contractLimits,
	}
	status := runtimeStatusSchema()
	controller := &testpilotspb.InstructionNode{
		InstructionId: "call", Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRpc{EndpointRoleId: "endpoint", Method: method, RequestAssignments: []*testpilotspb.RequestAssignment{
			{Target: runtimeField("namespace"), Value: runtimeEnvironment("namespace")},
			{Target: runtimeField("task_queue", "name"), Value: runtimeEnvironment("task-queue")},
		}}}},
		Outcome: proto.CloneOf(status), Limits: runtimeBounds(), ActivationReservations: []*testpilotspb.ActivationReservationDefinition{{EntrypointId: "workflow", Count: 1}, {EntrypointId: "handler", Count: 1}},
	}
	start := &testpilotspb.InstructionNode{
		InstructionId: "start", Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_StartNexusOperation{StartNexusOperation: &testpilotspb.StartNexusOperation{EndpointRoleId: "nexus-endpoint", Service: "service", Operation: "operation", Input: runtimeText("request")}}},
		Outcome: proto.CloneOf(status), Limits: runtimeBounds(),
	}
	await := &testpilotspb.InstructionNode{
		InstructionId: "await", Dependencies: []*testpilotspb.InstructionReference{{EntrypointId: "workflow", InstructionId: "start"}},
		Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitInstruction{AwaitInstruction: &testpilotspb.AwaitInstruction{Instruction: &testpilotspb.InstructionReference{EntrypointId: "workflow", InstructionId: "start"}}}},
		Outcome:     runtimeValueOutcomeSchema(), Limits: runtimeBounds(),
	}
	finish := &testpilotspb.InstructionNode{
		InstructionId: "finish", Dependencies: []*testpilotspb.InstructionReference{{EntrypointId: "workflow", InstructionId: "await"}},
		Guard:       runtimeSucceeded("workflow", "await"),
		Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Outcome{Outcome: &testpilotspb.InstructionOutcomeRef{Instruction: &testpilotspb.InstructionReference{EntrypointId: "workflow", InstructionId: "await"}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE}}}}}},
		Outcome:     proto.CloneOf(status), Limits: runtimeBounds(),
	}
	respond := &testpilotspb.InstructionNode{
		InstructionId: "respond", Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_RespondNexus{RespondNexus: &testpilotspb.RespondNexus{Kind: responseKind, Result: runtimeText("accepted")}}},
		Outcome: proto.CloneOf(status), Limits: runtimeBounds(),
	}
	program := &testpilotspb.Program{
		ProgramId:   "program",
		Environment: []*testpilotspb.EnvironmentDefinition{{BindingId: "namespace"}, {BindingId: "task-queue"}, {BindingId: "nexus-endpoint"}},
		Roles: []*testpilotspb.Role{
			{RoleId: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT},
			{RoleId: "worker", Kind: testpilotspb.ROLE_KIND_WORKER, NamespaceBindingId: "namespace"},
			{RoleId: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingId: "namespace", ResourceBindingId: "task-queue"},
			{RoleId: "nexus-endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, ResourceBindingId: "nexus-endpoint"},
		},
		Entrypoints: []*testpilotspb.Entrypoint{
			{EntrypointId: "controller", Activation: &testpilotspb.Entrypoint_Controller{Controller: &testpilotspb.ControllerActivation{}}, Instructions: []*testpilotspb.InstructionNode{controller}},
			{EntrypointId: "workflow", Activation: &testpilotspb.Entrypoint_Workflow{Workflow: &testpilotspb.WorkflowActivation{WorkflowType: "workflow-type", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}, Instructions: []*testpilotspb.InstructionNode{start, await, finish}},
			{EntrypointId: "handler", Activation: &testpilotspb.Entrypoint_NexusHandler{NexusHandler: &testpilotspb.NexusHandlerActivation{Service: "service", Operation: "operation", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}, Instructions: []*testpilotspb.InstructionNode{respond}},
		},
		Cleanup: &testpilotspb.Cleanup{EntrypointId: "cleanup"}, Limits: limits,
	}
	if responseKind == testpilotspb.NEXUS_RESPONSE_KIND_ASYNCHRONOUS {
		program.Slots = []*testpilotspb.Slot{{SlotId: "capability", Content: &testpilotspb.Slot_OpaqueHandle{OpaqueHandle: &testpilotspb.OpaqueHandleType{}}}}
		respond.Instruction.GetRespondNexus().HandleSlotId = "capability"
	}
	for _, apply := range modify {
		apply(program)
	}
	if modifyProfile != nil {
		modifyProfile(&profile)
	}
	contract := &testpilotspb.Contract{
		ContractId: "contract",
		Limits:     contractLimits,
		Rules: []*testpilotspb.ContractRule{{
			RuleId:         "complete",
			Kind:           testpilotspb.CONTRACT_RULE_KIND_SAFETY,
			InitialStateId: "open",
			States: []*testpilotspb.ContractState{
				{StateId: "open", Status: testpilotspb.CONTRACT_STATE_STATUS_PENDING},
				{StateId: "closed", Status: testpilotspb.CONTRACT_STATE_STATUS_SATISFIED},
			},
			Transitions: []*testpilotspb.ContractTransition{{
				TransitionId:  "close",
				SourceStateId: "open",
				TargetStateId: "closed",
				EventFilter:   &testpilotspb.RunEventFilter{Kinds: []testpilotspb.RunEventKind{testpilotspb.RUN_EVENT_KIND_RUN_CLOSED}},
				Predicate:     &testpilotspb.ContractExpression{Expression: &testpilotspb.ContractExpression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: true}}}},
				SupportKind:   testpilotspb.CONTRACT_SUPPORT_KIND_MATCHING_EVENT,
			}},
		}},
	}
	prepared, err := testpilot.Prepare(&testpilotspb.Case{Version: &testpilotspb.FormatVersion{Major: 1}, CaseId: "case", Program: program, Contract: contract}, profile)
	require.NoError(t, err)
	return prepared
}

func capturePreparedProgram(t *testing.T, prepared *testpilot.PreparedCase) testpilot.PreparedProgram {
	t.Helper()
	driver := &programCaptureDriver{identity: prepared.Identity()}
	_, _, err := prepared.Run(t.Context(), driver)
	require.ErrorIs(t, err, errProgramCaptured)
	return driver.program
}

var errProgramCaptured = errors.New("prepared Program captured")

type programCaptureDriver struct {
	identity testpilot.DriverIdentity
	program  testpilot.PreparedProgram
}

func (d *programCaptureDriver) Identity(context.Context) (testpilot.DriverIdentity, error) {
	return d.identity, nil
}

func (d *programCaptureDriver) Validate(context.Context, testpilot.PreparedProgram) error { return nil }

func (d *programCaptureDriver) Open(_ context.Context, _ string, program testpilot.PreparedProgram) (testpilot.Session, error) {
	d.program = program
	return nil, errProgramCaptured
}

func descriptorClosure(root protoreflect.FileDescriptor) *descriptorpb.FileDescriptorSet {
	seen := make(map[string]struct{})
	result := &descriptorpb.FileDescriptorSet{}
	var add func(protoreflect.FileDescriptor)
	add = func(file protoreflect.FileDescriptor) {
		if _, exists := seen[file.Path()]; exists {
			return
		}
		seen[file.Path()] = struct{}{}
		imports := file.Imports()
		for i := 0; i < imports.Len(); i++ {
			add(imports.Get(i))
		}
		result.File = append(result.File, protodesc.ToFileDescriptorProto(file))
	}
	add(root)
	return result
}

func runtimeBounds() *testpilotspb.InstructionLimits {
	return &testpilotspb.InstructionLimits{TimeoutMilliseconds: 1000, MaxAttempts: 1, MaxEmittedEvents: 4, MaxResponseBytes: 64 << 10}
}

func runtimeText(value string) *testpilotspb.ProgramExpression {
	return &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: value}}}}
}

func runtimeEnvironment(id string) *testpilotspb.ProgramExpression {
	return &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Environment{Environment: &testpilotspb.EnvironmentRef{BindingId: id}}}
}

func runtimeField(fields ...string) *testpilotspb.FieldPath {
	path := &testpilotspb.FieldPath{Segments: make([]*testpilotspb.FieldPathSegment, len(fields))}
	for index, field := range fields {
		path.Segments[index] = &testpilotspb.FieldPathSegment{Field: field}
	}
	return path
}

func runtimeSucceeded(entrypoint, instruction string) *testpilotspb.ProgramExpression {
	return &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Equals{Equals: &testpilotspb.ProgramEqualsExpression{
		Left:  &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Outcome{Outcome: &testpilotspb.InstructionOutcomeRef{Instruction: &testpilotspb.InstructionReference{EntrypointId: entrypoint, InstructionId: instruction}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS}}},
		Right: &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_EnumValue{EnumValue: &testpilotspb.EnumValue{Number: int32(testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED)}}}}},
	}}}
}

func runtimeStatusSchema() *testpilotspb.InstructionOutcomeDefinition {
	return &testpilotspb.InstructionOutcomeDefinition{Fields: []*testpilotspb.OutcomeFieldDefinition{{Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS, Type: runtimeStatusType()}}}
}

func runtimeValueOutcomeSchema() *testpilotspb.InstructionOutcomeDefinition {
	return &testpilotspb.InstructionOutcomeDefinition{Fields: []*testpilotspb.OutcomeFieldDefinition{{Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS, Type: runtimeStatusType()}, {Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE, Type: runtimeTextType()}}}
}

func runtimeStatusType() *testpilotspb.ValueType {
	return &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Enumeration{Enumeration: &testpilotspb.NamedType{ProtobufType: "temporal.server.api.testpilot.v1.InstructionOutcomeStatus"}}}}}
}

func runtimeTextType() *testpilotspb.ValueType {
	return &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Scalar{Scalar: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_TEXT}}}}}
}
