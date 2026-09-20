package activation

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	nexuspb "go.temporal.io/api/nexus/v1"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
)

// nexusReplyKind picks the typed handler reply the fixture's handler entrypoint answers with.
type nexusReplyKind uint8

const (
	synchronousReply nexusReplyKind = iota
	asynchronousReply
)

func preparedRuntimeFixture(t *testing.T, responseKind nexusReplyKind, modify ...func(*testpilotspb.Program)) testpilot.PreparedProgram {
	return preparedRuntimeFixtureWithProfile(t, responseKind, nil, modify...)
}

func preparedRuntimeFixtureWithProfile(t *testing.T, responseKind nexusReplyKind, modifyProfile func(*testpilot.ProfileSpec), modify ...func(*testpilotspb.Program)) testpilot.PreparedProgram {
	t.Helper()
	file := workflowservice.File_temporal_api_workflowservice_v1_service_proto
	catalog, err := testpilot.NewCatalog(descriptorClosure(file))
	require.NoError(t, err)
	limits := &testpilotspb.ProgramLimits{MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 16, MaxAttempts: 16, MaxRunEvents: 16, MaxExpressionDepth: 16, MaxPathFanout: 32, MaxRequestBytes: 64 << 10, MaxResponseBytes: 64 << 10, MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 5000, MaxInstructionEmittedEvents: 4, MaxInstructionResponseBytes: 64 << 10}
	method := "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"
	contractLimits := &testpilotspb.ContractLimits{MaxRules: 8, MaxStates: 16, MaxTransitions: 16, MaxExpressionDepth: 16, MaxWorkPerEvent: 100000, MaxTotalWork: 1000000000, MaxCaptures: 8, MaxCaptureBytes: 65536}
	profile := testpilot.ProfileSpec{
		Identity: "profile", Catalog: catalog,
		Roles: []testpilot.RolePolicy{
			{ID: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, Methods: []string{method}, ReservationCarriers: []testpilot.ReservationCarrierPolicy{{Method: method, Shapes: []testpilot.ReservationCarrierShape{{Kind: testpilot.WorkflowEntrypoint, MaximumCount: 8}, {Kind: testpilot.NexusHandlerEntrypoint, MaximumCount: 8}}}}},
			{ID: "worker", Kind: testpilotspb.ROLE_KIND_WORKER},
			{ID: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE},
			{ID: "nexus-endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT},
		},
		Opcodes:      []testpilot.Opcode{testpilot.InvokeRPC, testpilot.WorkflowCommand, testpilot.Await, testpilot.Finish, testpilot.NexusHandlerReply},
		CommandTypes: []enumspb.CommandType{enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION},
		EnvironmentBindings: []testpilot.EnvironmentBinding{
			{ID: "namespace", Value: "namespace"}, {ID: "task-queue", Value: "task-queue"}, {ID: "nexus-endpoint", Value: "endpoint"},
		},
		ProgramLimits: proto.CloneOf(limits), ContractLimits: contractLimits,
	}
	controller := &testpilotspb.InstructionNode{
		InstructionId: "call", Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRpc{EndpointRoleId: "endpoint", Method: method, RequestAssignments: []*testpilotspb.RequestAssignment{
			{Target: runtimeField("namespace"), Value: runtimeEnvironment("namespace")},
			{Target: runtimeField("task_queue", "name"), Value: runtimeEnvironment("task-queue")},
		}}}},
		Limits: runtimeBounds(),
	}
	start := &testpilotspb.InstructionNode{
		InstructionId: "start", Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_WorkflowCommand{WorkflowCommand: &testpilotspb.WorkflowCommand{Command: &commandpb.Command{
			CommandType: enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION,
			Attributes: &commandpb.Command_ScheduleNexusOperationCommandAttributes{ScheduleNexusOperationCommandAttributes: &commandpb.ScheduleNexusOperationCommandAttributes{
				Endpoint: "nexus-endpoint", Service: "service", Operation: "operation", Input: runtimePayload("request"),
			}},
		}}}},
		Limits: runtimeBounds(),
	}
	await := &testpilotspb.InstructionNode{
		InstructionId: "await", Guard: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: true}}}},
		Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitInstruction{AwaitInstruction: &testpilotspb.AwaitInstruction{Instruction: &testpilotspb.InstructionReference{EntrypointId: "workflow", InstructionId: "start"}}}},
		Limits:      runtimeBounds(),
	}
	finish := &testpilotspb.InstructionNode{
		InstructionId: "finish",
		Guard:         runtimeSucceeded("workflow", "await"),
		Instruction:   &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_Outcome{Outcome: &testpilotspb.InstructionOutcomeReference{Instruction: &testpilotspb.InstructionReference{EntrypointId: "workflow", InstructionId: "await"}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE}}}}}}}},
		Limits:        runtimeBounds(),
	}
	reply := &testpilotspb.NexusHandlerReply{Reply: &testpilotspb.NexusHandlerReply_Response{Response: &nexuspb.StartOperationResponse{
		Variant: &nexuspb.StartOperationResponse_SyncSuccess{SyncSuccess: &nexuspb.StartOperationResponse_Sync{Payload: runtimePayload("accepted")}},
	}}}
	if responseKind == asynchronousReply {
		reply = &testpilotspb.NexusHandlerReply{HandleSlotId: "capability", Reply: &testpilotspb.NexusHandlerReply_Response{Response: &nexuspb.StartOperationResponse{
			Variant: &nexuspb.StartOperationResponse_AsyncSuccess{AsyncSuccess: &nexuspb.StartOperationResponse_Async{}},
		}}}
	}
	respond := &testpilotspb.InstructionNode{
		InstructionId: "respond", Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_NexusHandlerReply{NexusHandlerReply: reply}},
		Limits: runtimeBounds(),
	}
	program := &testpilotspb.Program{
		ProgramId: "program",
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
		Cleanup: &testpilotspb.Cleanup{EntrypointId: "cleanup"},
	}
	if responseKind == asynchronousReply {
		program.Slots = []*testpilotspb.Slot{{SlotId: "capability", Content: &testpilotspb.Slot_OpaqueHandle{OpaqueHandle: &testpilotspb.OpaqueHandleType{}}}}
	}
	for _, apply := range modify {
		apply(program)
	}
	if modifyProfile != nil {
		modifyProfile(&profile)
	}
	contract := &testpilotspb.Contract{
		ContractId: "contract",
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
				Predicate:     &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: true}}}},
				SupportKind:   testpilotspb.CONTRACT_SUPPORT_KIND_MATCHING_EVENT,
			}},
		}},
	}
	prepared, err := testpilot.Prepare(&testpilotspb.Case{Version: &testpilotspb.FormatVersion{Major: 1}, CaseId: "case", Program: program, Contract: contract}, profile)
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
	return &testpilotspb.InstructionLimits{Timeout: &testpilotspb.InstructionLimits_TimeoutMilliseconds{TimeoutMilliseconds: 1000}, Attempts: &testpilotspb.InstructionLimits_MaxAttempts{MaxAttempts: 1}}
}

// runtimePayload is the carried payload a typed instruction of the fixture holds, encoded as the
// SDK's default data converter encodes a string.
func runtimePayload(value string) *commonpb.Payload {
	return &commonpb.Payload{Metadata: map[string][]byte{"encoding": []byte("json/plain")}, Data: []byte(strconv.Quote(value))}
}

func runtimeText(value string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: value}}}}
}

func runtimeEnvironment(id string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_EnvironmentBindingId{EnvironmentBindingId: id}}}}
}

func runtimeField(fields ...string) string {
	return strings.Join(fields, ".")
}

func runtimeSucceeded(entrypoint, instruction string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{Operator: testpilotspb.COMPARISON_OPERATOR_EQUAL,
		Left:  &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_Outcome{Outcome: &testpilotspb.InstructionOutcomeReference{Instruction: &testpilotspb.InstructionReference{EntrypointId: entrypoint, InstructionId: instruction}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS}}}}},
		Right: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_EnumValue{EnumValue: &testpilotspb.EnumValue{Name: "INSTRUCTION_OUTCOME_STATUS_SUCCEEDED"}}}}},
	}}}
}

func runtimeStatusType() *testpilotspb.ValueType {
	return &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Enumeration{Enumeration: &testpilotspb.NamedType{ProtobufType: "temporal.server.api.testpilot.v1.InstructionOutcomeStatus"}}}}}
}

func runtimeTextType() *testpilotspb.ValueType {
	return &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Scalar{Scalar: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_TEXT}}}}}
}

func TestConstruction(t *testing.T) {
	state, err := New(testpilot.EntrypointPlan{})
	require.Error(t, err)
	require.Nil(t, state)
	program := preparedRuntimeFixture(t, synchronousReply)
	for _, id := range []string{"controller", "workflow", "handler"} {
		t.Run(id, func(t *testing.T) {
			plan, ok := findEntrypoint(program, id)
			require.True(t, ok)
			state, err := New(plan)
			if id == "controller" {
				require.Error(t, err)
				require.Nil(t, state)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, state)
			require.Equal(t, plan.RuntimeWorkLimit(), state.remaining)
		})
	}
}

func newState(t *testing.T, plan testpilot.EntrypointPlan) *State {
	t.Helper()
	state, err := New(plan)
	require.NoError(t, err)
	require.NotNil(t, state)
	return state
}

func workflowPlan(t *testing.T, modify ...func(*testpilotspb.Program)) testpilot.EntrypointPlan {
	t.Helper()
	plan, ok := findEntrypoint(preparedRuntimeFixture(t, synchronousReply, modify...), "workflow")
	require.True(t, ok)
	return plan
}

func success(value string) *testpilotspb.InstructionOutcome {
	return &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, Value: carriedValue(value)}
}

// carriedValue is what an awaited Nexus operation answers with: the handler's payload, carried
// whole rather than converted, so the Await's VALUE is that message.
func carriedValue(value string) *testpilotspb.Value {
	carried, err := anypb.New(runtimePayload(value))
	if err != nil {
		panic(err)
	}
	return &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: carried}}
}

// carriedText reads the text back out of the payload carriedValue wrapped.
func carriedText(t *testing.T, value *testpilotspb.Value) string {
	t.Helper()
	var payload commonpb.Payload
	require.NoError(t, value.GetMessageValue().UnmarshalTo(&payload))
	text, err := strconv.Unquote(string(payload.GetData()))
	require.NoError(t, err)
	return text
}

func evaluateEnabled(t *testing.T, state *State, index int) *testpilotspb.Value {
	t.Helper()
	input, enabled, err := state.Evaluate(t.Context(), index)
	require.NoError(t, err)
	require.True(t, enabled)
	return input
}

func TestGuardAndOwnership(t *testing.T) {
	plan := workflowPlan(t)
	for _, succeeded := range []bool{false, true} {
		t.Run(fmt.Sprint(succeeded), func(t *testing.T) {
			state := newState(t, plan)
			require.Nil(t, evaluateEnabled(t, state, 1))
			outcome := success("result")
			if !succeeded {
				outcome = &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SDK_FAILURE}
			}
			require.NoError(t, state.Admit(t.Context(), 1, outcome))
			require.Nil(t, state.lookup(testpilot.ValueReference{Kind: testpilot.SlotReference, ID: "private-capability"}))
			outcome.Value = success("mutated").Value
			input, enabled, err := state.Evaluate(t.Context(), 2)
			require.NoError(t, err)
			require.Equal(t, succeeded, enabled)
			if succeeded {
				require.True(t, proto.Equal(success("result").Value, input))
				input.Value = success("changed input").Value.Value
			} else {
				require.Nil(t, input)
				require.Error(t, state.Admit(t.Context(), 2, outcome))
			}
			_, _, err = state.Evaluate(t.Context(), 2)
			require.Error(t, err)
			require.Error(t, state.Admit(t.Context(), 1, success("replacement")))
			if succeeded {
				require.Equal(t, "result", carriedText(t, state.values[testpilot.ValueReference{Kind: testpilot.OutcomeReference, Entrypoint: "workflow", ID: "await", Field: int32(testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE)}]), "only successful outcomes have a result")
			}
		})
	}
}

func TestEvaluationLifecycle(t *testing.T) {
	plan := workflowPlan(t)
	for _, index := range []int{-1, 3} {
		state := newState(t, plan)
		_, _, err := state.Evaluate(t.Context(), index)
		require.Error(t, err)
		require.Error(t, state.Admit(t.Context(), index, nil))
	}
	state := newState(t, plan)
	require.Error(t, state.Admit(t.Context(), 1, success("early")))
	require.Nil(t, evaluateEnabled(t, state, 1))
	remaining := state.remaining
	_, _, err := state.Evaluate(t.Context(), 1)
	require.Error(t, err)
	require.Equal(t, remaining, state.remaining)
	require.NoError(t, state.Admit(t.Context(), 1, success("result")))
	_, _, err = state.Evaluate(t.Context(), 1)
	require.Error(t, err)

	// finish's guard compares await's status, which is absent until await is admitted, so finish is
	// skipped rather than enabled.
	state = newState(t, plan)
	input, enabled, err := state.Evaluate(t.Context(), 2)
	require.NoError(t, err)
	require.False(t, enabled)
	require.Nil(t, input)
	remaining = state.remaining
	_, _, err = state.Evaluate(t.Context(), 2)
	require.Error(t, err)
	require.Equal(t, remaining, state.remaining)
	require.Error(t, state.Admit(t.Context(), 2, nil))
}

func TestRejectedOutcomesAreAtomic(t *testing.T) {
	plan := workflowPlan(t)
	for name, outcome := range map[string]*testpilotspb.InstructionOutcome{
		"nil":            nil,
		"unspecified":    {},
		"unknown status": {Status: 999},
		"missing value":  {Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED},
		"wrong type":     {Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, Value: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: true}}},
		"oversized":      success(strings.Repeat("x", 65537)),
		"protocol":       {Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE},
	} {
		t.Run(name, func(t *testing.T) {
			state := newState(t, plan)
			evaluateEnabled(t, state, 1)
			require.Error(t, state.Admit(t.Context(), 1, outcome))
			require.Empty(t, state.values)
			require.Error(t, state.Admit(t.Context(), 1, success("retry")))
			// No status was recorded, so the dependent's success guard is false.
			input, enabled, err := state.Evaluate(t.Context(), 2)
			require.NoError(t, err)
			require.False(t, enabled)
			require.Nil(t, input)
		})
	}
	state := newState(t, plan)
	evaluateEnabled(t, state, 0)
	require.Error(t, state.Admit(t.Context(), 0, success("undeclared future")))
	require.Empty(t, state.values)
}

func TestCanceledAndNilContexts(t *testing.T) {
	plan := workflowPlan(t)
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	for _, ctx := range []context.Context{nil, canceled} {
		state := newState(t, plan)
		_, _, err := state.Evaluate(ctx, 1)
		require.Error(t, err)
		if ctx != nil {
			require.ErrorIs(t, err, context.Canceled)
		}
		_, _, err = state.Evaluate(t.Context(), 1)
		require.Error(t, err)
		state = newState(t, plan)
		evaluateEnabled(t, state, 1)
		err = state.Admit(ctx, 1, success("result"))
		require.Error(t, err)
		if ctx != nil {
			require.ErrorIs(t, err, context.Canceled)
		}
		require.Empty(t, state.values)
		require.Error(t, state.Admit(t.Context(), 1, success("retry")))
	}
}

func TestWorkAccounting(t *testing.T) {
	plan := workflowPlan(t)
	// A schedule command and a true guard evaluate to nothing, so finish's guard and input are what
	// an allowance has to cover; measuring the cost keeps the table off a literal the fixture owns.
	measured := newState(t, plan)
	evaluateEnabled(t, measured, 1)
	require.NoError(t, measured.Admit(t.Context(), 1, success("result")))
	before := measured.remaining
	evaluateEnabled(t, measured, 2)
	evaluation := before - measured.remaining
	require.Positive(t, evaluation)
	for _, allowance := range []int64{-1, 0, evaluation - 1, evaluation, math.MaxInt64} {
		t.Run(fmt.Sprint(allowance), func(t *testing.T) {
			state := newState(t, plan)
			evaluateEnabled(t, state, 1)
			require.NoError(t, state.Admit(t.Context(), 1, success("result")))
			state.remaining = allowance
			input, enabled, err := state.Evaluate(t.Context(), 2)
			if allowance == evaluation {
				require.NoError(t, err)
				require.True(t, enabled)
				require.Equal(t, "result", carriedText(t, input))
				require.Zero(t, state.remaining)
				require.Error(t, state.Admit(t.Context(), 2, &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}))
			} else {
				require.Error(t, err)
				require.Nil(t, input)
				require.False(t, enabled)
				require.GreaterOrEqual(t, state.remaining, int64(0))
			}
		})
	}
	state := newState(t, plan)
	initial := state.remaining
	evaluateEnabled(t, state, 0)
	require.Equal(t, initial, state.remaining, "a schedule command evaluates no expression")
	evaluateEnabled(t, state, 1)
	require.NoError(t, state.Admit(t.Context(), 1, success("result")))
	_, work, err := plan.Instructions()[1].ValidateOutcome(t.Context(), success("result"), initial)
	require.NoError(t, err)
	require.Equal(t, initial-work, state.remaining)
	for _, allowance := range []int64{work - 1, work} {
		tight := newState(t, plan)
		evaluateEnabled(t, tight, 1)
		tight.remaining = allowance
		err := tight.Admit(t.Context(), 1, success("result"))
		if allowance == work {
			require.NoError(t, err)
			require.Zero(t, tight.remaining)
		} else {
			require.Error(t, err)
			require.Empty(t, tight.values)
			require.Less(t, tight.remaining, allowance)
			require.GreaterOrEqual(t, tight.remaining, int64(0))
		}
	}
	state = newState(t, plan)
	evaluateEnabled(t, state, 0)
	initial = state.remaining
	_, charged, err := plan.Instructions()[0].ValidateOutcome(t.Context(), success("undeclared"), initial)
	require.Error(t, err)
	require.Positive(t, charged)
	require.Error(t, state.Admit(t.Context(), 0, success("undeclared")))
	require.Equal(t, initial-charged, state.remaining)
}

func TestIndependentActivations(t *testing.T) {
	plan := workflowPlan(t)
	before := plan.Activation()
	t.Cleanup(func() { require.True(t, proto.Equal(before, plan.Activation())) })
	exercise := func(t *testing.T, value string) {
		t.Helper()
		state := newState(t, plan)
		// A schedule command carries its own message, so it evaluates to no input.
		require.Nil(t, evaluateEnabled(t, state, 0))
		evaluateEnabled(t, state, 1)
		require.NoError(t, state.Admit(t.Context(), 1, success(value)))
		require.Equal(t, value, carriedText(t, evaluateEnabled(t, state, 2)))
	}
	exercise(t, "first")
	exercise(t, "second")
	for i := range 10 {
		t.Run(fmt.Sprint(i), func(t *testing.T) { t.Parallel(); exercise(t, fmt.Sprint(i)) })
	}
	snapshot := plan.Instructions()[0].Source()
	snapshot.Instruction.GetWorkflowCommand().GetCommand().GetScheduleNexusOperationCommandAttributes().Input = runtimePayload("mutated")
	exercise(t, "after snapshot mutation")
	require.True(t, proto.Equal(before, plan.Activation()))
	// A fresh activation sees no other activation's await outcome, so finish's success guard is false.
	state := newState(t, plan)
	input, enabled, err := state.Evaluate(t.Context(), 2)
	require.NoError(t, err)
	require.False(t, enabled)
	require.Nil(t, input)
}

func findEntrypoint(program testpilot.PreparedProgram, id string) (testpilot.EntrypointPlan, bool) {
	for _, plan := range program.Entrypoints() {
		if plan.ID() == id {
			return plan, true
		}
	}
	return testpilot.EntrypointPlan{}, false
}

func TestPresenceAndMissingRequiredInput(t *testing.T) {
	for _, mode := range []string{"present", "false all", "true"} {
		t.Run(mode, func(t *testing.T) {
			plan := workflowPlan(t, func(program *testpilotspb.Program) {
				finish := program.Entrypoints[1].Instructions[2]
				switch mode {
				case "present":
					finish.Guard = &testpilotspb.Expression{Expression: &testpilotspb.Expression_Present{Present: &testpilotspb.PresentExpression{Operand: proto.CloneOf(finish.Instruction.GetFinish().Result)}}}
				case "false all":
					finish.Guard = &testpilotspb.Expression{Expression: &testpilotspb.Expression_All{All: &testpilotspb.AllExpression{Operands: []*testpilotspb.Expression{boolean(false), finish.Guard}}}}
				case "true":
					finish.Guard = boolean(true)
					finish.Instruction.GetFinish().Result.GetReference().GetOutcome().Field = testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS
				default:
					t.Fatalf("unknown guard mode %q", mode)
				}
			})
			state := newState(t, plan)
			input, enabled, err := state.Evaluate(t.Context(), 2)
			if mode == "true" {
				require.Error(t, err)
				// A true guard binds as no guard, so only the input's evaluation is charged.
				require.Equal(t, plan.RuntimeWorkLimit()-1, state.remaining)
			} else {
				require.NoError(t, err)
			}
			require.Nil(t, input)
			require.False(t, enabled)
			require.Error(t, state.Admit(t.Context(), 2, nil))
			if mode == "present" {
				state = newState(t, plan)
				evaluateEnabled(t, state, 1)
				require.NoError(t, state.Admit(t.Context(), 1, success("present")))
				require.Equal(t, "present", carriedText(t, evaluateEnabled(t, state, 2)))
			}
		})
	}
}

func boolean(value bool) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: value}}}}
}

func TestRepeatedReadsOwnTheirValues(t *testing.T) {
	plan := workflowPlan(t, func(program *testpilotspb.Program) {
		entry := program.Entrypoints[1]
		second := proto.CloneOf(entry.Instructions[2])
		second.InstructionId = "second-finish"
		entry.Instructions = append(entry.Instructions, second)
	})
	state := newState(t, plan)
	evaluateEnabled(t, state, 1)
	outcome := success("result")
	require.NoError(t, state.Admit(t.Context(), 1, outcome))
	outcome.Value.Value = success("mutated raw").Value.Value
	snapshot, _, err := plan.Instructions()[1].ValidateOutcome(t.Context(), success("foreign"), plan.RuntimeWorkLimit())
	require.NoError(t, err)
	snapshot.Fields[testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE].Value = success("mutated snapshot").Value.Value
	first := evaluateEnabled(t, state, 2)
	require.Equal(t, "result", carriedText(t, first))
	first.Value = success("mutated input").Value.Value
	require.Equal(t, "result", carriedText(t, evaluateEnabled(t, state, 3)))
}
