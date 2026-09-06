package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func preparedRuntimeFixture(t *testing.T, responseKind testpilotpb.NexusResponseKind, modify ...func(*testpilotpb.Program)) testpilot.PreparedProgram {
	t.Helper()
	file := workflowservice.File_temporal_api_workflowservice_v1_service_proto
	catalog, err := testpilot.NewCatalog(descriptorClosure(file))
	require.NoError(t, err)
	limits := &testpilotpb.ProgramLimits{MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 16, MaxAttempts: 16, MaxRunEvents: 16, MaxExpressionDepth: 16, MaxPathFanout: 32, MaxRequestBytes: 64 << 10, MaxResponseBytes: 64 << 10, MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 5000}
	method := "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"
	contractLimits := &testpilotpb.ContractLimits{MaxRules: 8, MaxStates: 16, MaxTransitions: 16, MaxExpressionDepth: 16, MaxWorkPerEvent: 100000, MaxTotalWork: 1000000000, MaxCaptures: 8, MaxCaptureBytes: 65536}
	profile := testpilot.ProfileSpec{
		Identity: "profile", Catalog: catalog,
		Roles: []testpilot.RolePolicy{
			{ID: "endpoint", Kind: testpilotpb.ROLE_KIND_ENDPOINT, Methods: []string{method}, ReservationCarriers: []testpilot.ReservationCarrierPolicy{{Method: method, Shapes: []testpilot.ReservationCarrierShape{{Context: testpilotpb.ENTRYPOINT_KIND_WORKFLOW, MaximumCount: 8}, {Context: testpilotpb.ENTRYPOINT_KIND_NEXUS_HANDLER, MaximumCount: 8}}}}},
			{ID: "worker", Kind: testpilotpb.ROLE_KIND_WORKER},
			{ID: "queue", Kind: testpilotpb.ROLE_KIND_TASK_QUEUE},
		},
		Capabilities:  []testpilot.Capability{testpilot.InvokeRPC, testpilot.StartNexusOperation, testpilot.Await, testpilot.Finish, testpilot.RespondNexus},
		ProgramLimits: proto.CloneOf(limits), ContractLimits: contractLimits,
	}
	status := runtimeStatusSchema()
	controller := &testpilotpb.InstructionDefinition{
		InstructionId: "call", Instruction: &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_InvokeRpc{InvokeRpc: &testpilotpb.InvokeRPC{EndpointRoleId: "endpoint", Method: method}}},
		Outcome: proto.CloneOf(status), Limits: runtimeBounds(), ActivationReservations: []*testpilotpb.ActivationReservationDefinition{{EntrypointId: "workflow", Count: 1}, {EntrypointId: "handler", Count: 1}},
	}
	start := &testpilotpb.InstructionDefinition{
		InstructionId: "start", Instruction: &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_StartNexusOperation{StartNexusOperation: &testpilotpb.StartNexusOperation{EndpointRoleId: "endpoint", Service: "service", Operation: "operation", Input: runtimeText("request")}}},
		Outcome: proto.CloneOf(status), Limits: runtimeBounds(),
	}
	await := &testpilotpb.InstructionDefinition{
		InstructionId: "await", Dependencies: []*testpilotpb.InstructionRef{{EntrypointId: "workflow", InstructionId: "start"}},
		Instruction: &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_AwaitOutcome{AwaitOutcome: &testpilotpb.AwaitInstruction{Instruction: &testpilotpb.InstructionRef{EntrypointId: "workflow", InstructionId: "start"}}}},
		Outcome:     runtimeValueOutcomeSchema(), Limits: runtimeBounds(),
	}
	finish := &testpilotpb.InstructionDefinition{
		InstructionId: "finish", Dependencies: []*testpilotpb.InstructionRef{{EntrypointId: "workflow", InstructionId: "await"}},
		Guard:       runtimeSucceeded("workflow", "await"),
		Instruction: &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_Finish{Finish: &testpilotpb.Finish{Result: &testpilotpb.ProgramExpression{Expression: &testpilotpb.ProgramExpression_Outcome{Outcome: &testpilotpb.InstructionOutcomeRef{Instruction: &testpilotpb.InstructionRef{EntrypointId: "workflow", InstructionId: "await"}, Field: testpilotpb.INSTRUCTION_OUTCOME_FIELD_VALUE}}}}}},
		Outcome:     proto.CloneOf(status), Limits: runtimeBounds(),
	}
	respond := &testpilotpb.InstructionDefinition{
		InstructionId: "respond", Instruction: &testpilotpb.Instruction{Instruction: &testpilotpb.Instruction_RespondNexus{RespondNexus: &testpilotpb.RespondNexus{Kind: responseKind, Result: runtimeText("accepted")}}},
		Outcome: proto.CloneOf(status), Limits: runtimeBounds(),
	}
	program := &testpilotpb.Program{
		ProgramId: "program", Roles: []*testpilotpb.RoleDefinition{{RoleId: "endpoint", Kind: testpilotpb.ROLE_KIND_ENDPOINT}, {RoleId: "worker", Kind: testpilotpb.ROLE_KIND_WORKER}, {RoleId: "queue", Kind: testpilotpb.ROLE_KIND_TASK_QUEUE}},
		Entrypoints: []*testpilotpb.EntrypointDefinition{
			{EntrypointId: "controller", Activation: &testpilotpb.EntrypointDefinition_Controller{Controller: &testpilotpb.ControllerActivation{}}, Instructions: []*testpilotpb.InstructionDefinition{controller}},
			{EntrypointId: "workflow", Activation: &testpilotpb.EntrypointDefinition_Workflow{Workflow: &testpilotpb.WorkflowActivation{WorkflowType: "workflow-type", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}, Instructions: []*testpilotpb.InstructionDefinition{start, await, finish}},
			{EntrypointId: "handler", Activation: &testpilotpb.EntrypointDefinition_NexusHandler{NexusHandler: &testpilotpb.NexusHandlerActivation{Service: "service", Operation: "operation", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}, Instructions: []*testpilotpb.InstructionDefinition{respond}},
		},
		Cleanup: &testpilotpb.CleanupDefinition{EntrypointId: "cleanup"}, Limits: limits,
	}
	if responseKind == testpilotpb.NEXUS_RESPONSE_KIND_ASYNCHRONOUS {
		program.Slots = []*testpilotpb.SlotDefinition{{SlotId: "capability", Content: &testpilotpb.SlotDefinition_OpaqueCapability{OpaqueCapability: &testpilotpb.OpaqueCapabilityType{}}}}
		respond.Instruction.GetRespondNexus().CapabilitySlotId = "capability"
	}
	for _, apply := range modify {
		apply(program)
	}
	contract := &testpilotpb.Contract{
		ContractId: "contract",
		Limits:     contractLimits,
		Rules: []*testpilotpb.ContractRuleDefinition{{
			RuleId:         "complete",
			Kind:           testpilotpb.CONTRACT_RULE_KIND_SAFETY,
			InitialStateId: "open",
			States: []*testpilotpb.ContractStateDefinition{
				{StateId: "open", Status: testpilotpb.CONTRACT_STATE_STATUS_NONTERMINAL},
				{StateId: "closed", Status: testpilotpb.CONTRACT_STATE_STATUS_SATISFIED},
			},
			Transitions: []*testpilotpb.ContractTransitionDefinition{{
				TransitionId:  "close",
				SourceStateId: "open",
				TargetStateId: "closed",
				EventFilter:   &testpilotpb.RunEventFilter{Kinds: []testpilotpb.RunEventKind{testpilotpb.RUN_EVENT_KIND_RUN_CLOSED}},
				Predicate:     &testpilotpb.ContractExpression{Expression: &testpilotpb.ContractExpression_Literal{Literal: &testpilotpb.Value{Value: &testpilotpb.Value_BoolValue{BoolValue: true}}}},
				SupportKind:   testpilotpb.CONTRACT_SUPPORT_KIND_MATCHING_EVENT,
			}},
		}},
	}
	prepared, err := testpilot.Prepare(&testpilotpb.Case{Version: &testpilotpb.FormatVersion{Major: 1}, CaseId: "case", Program: program, Contract: contract}, profile)
	require.NoError(t, err)
	driver := &programCaptureDriver{identity: prepared.Identity()}
	_, _, err = prepared.Run(t.Context(), driver)
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

func runtimeBounds() *testpilotpb.InstructionLimits {
	return &testpilotpb.InstructionLimits{TimeoutMilliseconds: 1000, MaxAttempts: 1, MaxEmittedEvents: 4, MaxResponseBytes: 64 << 10}
}

func runtimeText(value string) *testpilotpb.ProgramExpression {
	return &testpilotpb.ProgramExpression{Expression: &testpilotpb.ProgramExpression_Literal{Literal: &testpilotpb.Value{Value: &testpilotpb.Value_Text{Text: value}}}}
}

func runtimeSucceeded(entrypoint, instruction string) *testpilotpb.ProgramExpression {
	return &testpilotpb.ProgramExpression{Expression: &testpilotpb.ProgramExpression_Equals{Equals: &testpilotpb.ProgramEqualsExpression{
		Left:  &testpilotpb.ProgramExpression{Expression: &testpilotpb.ProgramExpression_Outcome{Outcome: &testpilotpb.InstructionOutcomeRef{Instruction: &testpilotpb.InstructionRef{EntrypointId: entrypoint, InstructionId: instruction}, Field: testpilotpb.INSTRUCTION_OUTCOME_FIELD_STATUS}}},
		Right: &testpilotpb.ProgramExpression{Expression: &testpilotpb.ProgramExpression_Literal{Literal: &testpilotpb.Value{Value: &testpilotpb.Value_EnumValue{EnumValue: &testpilotpb.EnumValue{Number: int32(testpilotpb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED)}}}}},
	}}}
}

func runtimeStatusSchema() *testpilotpb.InstructionOutcomeDefinition {
	return &testpilotpb.InstructionOutcomeDefinition{Fields: []*testpilotpb.OutcomeFieldDefinition{{Field: testpilotpb.INSTRUCTION_OUTCOME_FIELD_STATUS, Type: runtimeStatusType()}}}
}

func runtimeValueOutcomeSchema() *testpilotpb.InstructionOutcomeDefinition {
	return &testpilotpb.InstructionOutcomeDefinition{Fields: []*testpilotpb.OutcomeFieldDefinition{{Field: testpilotpb.INSTRUCTION_OUTCOME_FIELD_STATUS, Type: runtimeStatusType()}, {Field: testpilotpb.INSTRUCTION_OUTCOME_FIELD_VALUE, Type: runtimeTextType()}}}
}

func runtimeStatusType() *testpilotpb.ValueType {
	return &testpilotpb.ValueType{Shape: &testpilotpb.ValueType_Singular{Singular: &testpilotpb.SingularType{Type: &testpilotpb.SingularType_Enumeration{Enumeration: &testpilotpb.NamedType{ProtobufType: "temporal.server.api.testpilot.v1.InstructionOutcomeStatus"}}}}}
}

func runtimeTextType() *testpilotpb.ValueType {
	return &testpilotpb.ValueType{Shape: &testpilotpb.ValueType_Singular{Singular: &testpilotpb.SingularType{Type: &testpilotpb.SingularType_Scalar{Scalar: &testpilotpb.ScalarType{Kind: testpilotpb.SCALAR_KIND_TEXT}}}}}
}
