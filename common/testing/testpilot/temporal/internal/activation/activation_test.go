package activation

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
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

func preparedRuntimeFixtureWithProfile(t *testing.T, responseKind testpilotspb.NexusResponseKind, modifyProfile func(*testpilot.ProfileSpec), modify ...func(*testpilotspb.Program)) testpilot.PreparedProgram {
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
			{ID: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, Methods: []string{method}, ReservationCarriers: []testpilot.ReservationCarrierPolicy{{Method: method, Shapes: []testpilot.ReservationCarrierShape{{Context: testpilotspb.ENTRYPOINT_KIND_WORKFLOW, MaximumCount: 8}, {Context: testpilotspb.ENTRYPOINT_KIND_NEXUS_HANDLER, MaximumCount: 8}}}}},
			{ID: "worker", Kind: testpilotspb.ROLE_KIND_WORKER},
			{ID: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE},
			{ID: "nexus-endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT},
		},
		Capabilities: []testpilot.Capability{testpilot.InvokeRPC, testpilot.StartNexusOperation, testpilot.Await, testpilot.Finish, testpilot.RespondNexus},
		EnvironmentBindings: []testpilot.EnvironmentBinding{
			{ID: "namespace", Value: "namespace"}, {ID: "task-queue", Value: "task-queue"}, {ID: "nexus-endpoint", Value: "endpoint"},
		},
		ProgramLimits: proto.CloneOf(limits), ContractLimits: contractLimits,
	}
	status := runtimeStatusSchema()
	controller := &testpilotspb.InstructionDefinition{
		InstructionId: "call", Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRPC{EndpointRoleId: "endpoint", Method: method, RequestAssignments: []*testpilotspb.RequestAssignment{
			{Target: runtimeField("namespace"), Value: runtimeEnvironment("namespace")},
			{Target: runtimeField("task_queue", "name"), Value: runtimeEnvironment("task-queue")},
		}}}},
		Outcome: proto.CloneOf(status), Limits: runtimeBounds(), ActivationReservations: []*testpilotspb.ActivationReservationDefinition{{EntrypointId: "workflow", Count: 1}, {EntrypointId: "handler", Count: 1}},
	}
	start := &testpilotspb.InstructionDefinition{
		InstructionId: "start", Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_StartNexusOperation{StartNexusOperation: &testpilotspb.StartNexusOperation{EndpointRoleId: "nexus-endpoint", Service: "service", Operation: "operation", Input: runtimeText("request")}}},
		Outcome: proto.CloneOf(status), Limits: runtimeBounds(),
	}
	await := &testpilotspb.InstructionDefinition{
		InstructionId: "await", Dependencies: []*testpilotspb.InstructionRef{{EntrypointId: "workflow", InstructionId: "start"}},
		Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitOutcome{AwaitOutcome: &testpilotspb.AwaitInstruction{Instruction: &testpilotspb.InstructionRef{EntrypointId: "workflow", InstructionId: "start"}}}},
		Outcome:     runtimeValueOutcomeSchema(), Limits: runtimeBounds(),
	}
	finish := &testpilotspb.InstructionDefinition{
		InstructionId: "finish", Dependencies: []*testpilotspb.InstructionRef{{EntrypointId: "workflow", InstructionId: "await"}},
		Guard:       runtimeSucceeded("workflow", "await"),
		Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Outcome{Outcome: &testpilotspb.InstructionOutcomeRef{Instruction: &testpilotspb.InstructionRef{EntrypointId: "workflow", InstructionId: "await"}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE}}}}}},
		Outcome:     proto.CloneOf(status), Limits: runtimeBounds(),
	}
	respond := &testpilotspb.InstructionDefinition{
		InstructionId: "respond", Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_RespondNexus{RespondNexus: &testpilotspb.RespondNexus{Kind: responseKind, Result: runtimeText("accepted")}}},
		Outcome: proto.CloneOf(status), Limits: runtimeBounds(),
	}
	program := &testpilotspb.Program{
		ProgramId:   "program",
		Environment: []*testpilotspb.EnvironmentDefinition{{BindingId: "namespace"}, {BindingId: "task-queue"}, {BindingId: "nexus-endpoint"}},
		Roles: []*testpilotspb.RoleDefinition{
			{RoleId: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT},
			{RoleId: "worker", Kind: testpilotspb.ROLE_KIND_WORKER, NamespaceBindingId: "namespace"},
			{RoleId: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingId: "namespace", ResourceBindingId: "task-queue"},
			{RoleId: "nexus-endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, ResourceBindingId: "nexus-endpoint"},
		},
		Entrypoints: []*testpilotspb.EntrypointDefinition{
			{EntrypointId: "controller", Activation: &testpilotspb.EntrypointDefinition_Controller{Controller: &testpilotspb.ControllerActivation{}}, Instructions: []*testpilotspb.InstructionDefinition{controller}},
			{EntrypointId: "workflow", Activation: &testpilotspb.EntrypointDefinition_Workflow{Workflow: &testpilotspb.WorkflowActivation{WorkflowType: "workflow-type", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}, Instructions: []*testpilotspb.InstructionDefinition{start, await, finish}},
			{EntrypointId: "handler", Activation: &testpilotspb.EntrypointDefinition_NexusHandler{NexusHandler: &testpilotspb.NexusHandlerActivation{Service: "service", Operation: "operation", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}, Instructions: []*testpilotspb.InstructionDefinition{respond}},
		},
		Cleanup: &testpilotspb.CleanupDefinition{EntrypointId: "cleanup"}, Limits: limits,
	}
	if responseKind == testpilotspb.NEXUS_RESPONSE_KIND_ASYNCHRONOUS {
		program.Slots = []*testpilotspb.SlotDefinition{{SlotId: "capability", Content: &testpilotspb.SlotDefinition_OpaqueCapability{OpaqueCapability: &testpilotspb.OpaqueCapabilityType{}}}}
		respond.Instruction.GetRespondNexus().CapabilitySlotId = "capability"
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
		Rules: []*testpilotspb.ContractRuleDefinition{{
			RuleId:         "complete",
			Kind:           testpilotspb.CONTRACT_RULE_KIND_SAFETY,
			InitialStateId: "open",
			States: []*testpilotspb.ContractStateDefinition{
				{StateId: "open", Status: testpilotspb.CONTRACT_STATE_STATUS_NONTERMINAL},
				{StateId: "closed", Status: testpilotspb.CONTRACT_STATE_STATUS_SATISFIED},
			},
			Transitions: []*testpilotspb.ContractTransitionDefinition{{
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
	return &testpilotspb.InstructionLimits{TimeoutMilliseconds: 1000, MaxAttempts: 1, MaxEmittedEvents: 4, MaxResponseBytes: 64 << 10}
}

func runtimeText(value string) *testpilotspb.ProgramExpression {
	return &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_Text{Text: value}}}}
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
		Left:  &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Outcome{Outcome: &testpilotspb.InstructionOutcomeRef{Instruction: &testpilotspb.InstructionRef{EntrypointId: entrypoint, InstructionId: instruction}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS}}},
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

func TestConstruction(t *testing.T) {
	state, err := New(testpilot.EntrypointPlan{})
	require.Error(t, err)
	require.Nil(t, state)
	program := preparedRuntimeFixture(t, testpilotspb.NEXUS_RESPONSE_KIND_SYNCHRONOUS)
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
	plan, ok := findEntrypoint(preparedRuntimeFixture(t, testpilotspb.NEXUS_RESPONSE_KIND_SYNCHRONOUS, modify...), "workflow")
	require.True(t, ok)
	return plan
}

func success(value string) *testpilotspb.InstructionOutcome {
	return &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, Value: &testpilotspb.Value{Value: &testpilotspb.Value_Text{Text: value}}}
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
				require.Equal(t, "result", state.values[testpilot.ValueReference{Kind: testpilot.OutcomeReference, Entrypoint: "workflow", ID: "await", Field: int32(testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE)}].GetText(), "only successful outcomes have a result")
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

	state = newState(t, plan)
	_, enabled, err := state.Evaluate(t.Context(), 2)
	require.Error(t, err)
	require.False(t, enabled)
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
		"protocol":       {Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS},
	} {
		t.Run(name, func(t *testing.T) {
			state := newState(t, plan)
			evaluateEnabled(t, state, 1)
			require.Error(t, state.Admit(t.Context(), 1, outcome))
			require.Empty(t, state.values)
			require.Error(t, state.Admit(t.Context(), 1, success("retry")))
			_, _, err := state.Evaluate(t.Context(), 2)
			require.Error(t, err)
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
	for _, allowance := range []int64{-1, 0, 18, 19, math.MaxInt64} {
		t.Run(fmt.Sprint(allowance), func(t *testing.T) {
			state := newState(t, plan)
			state.remaining = allowance
			input, enabled, err := state.Evaluate(t.Context(), 0)
			if allowance == 19 {
				require.NoError(t, err)
				require.True(t, enabled)
				require.Equal(t, "request", input.GetText())
				require.Zero(t, state.remaining)
				_, _, err = state.Evaluate(t.Context(), 1)
				require.Error(t, err)
				require.Error(t, state.Admit(t.Context(), 0, &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}))
			} else {
				require.Error(t, err)
				require.Nil(t, input)
				require.False(t, enabled)
				if allowance == 18 {
					require.EqualValues(t, 8, state.remaining)
				}
			}
		})
	}
	state := newState(t, plan)
	initial := state.remaining
	evaluateEnabled(t, state, 0)
	require.Equal(t, initial-19, state.remaining)
	evaluateEnabled(t, state, 1)
	require.NoError(t, state.Admit(t.Context(), 1, success("result")))
	_, work, err := plan.Instructions()[1].ValidateOutcome(t.Context(), success("result"), initial)
	require.NoError(t, err)
	require.Equal(t, initial-19-work, state.remaining)
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
		require.Equal(t, "request", evaluateEnabled(t, state, 0).GetText())
		evaluateEnabled(t, state, 1)
		require.NoError(t, state.Admit(t.Context(), 1, success(value)))
		require.Equal(t, value, evaluateEnabled(t, state, 2).GetText())
	}
	exercise(t, "first")
	exercise(t, "second")
	for i := range 10 {
		t.Run(fmt.Sprint(i), func(t *testing.T) { t.Parallel(); exercise(t, fmt.Sprint(i)) })
	}
	snapshot := plan.Instructions()[0].Source()
	snapshot.Instruction.GetStartNexusOperation().Input = runtimeText("mutated")
	exercise(t, "after snapshot mutation")
	require.True(t, proto.Equal(before, plan.Activation()))
	state := newState(t, plan)
	_, _, err := state.Evaluate(t.Context(), 2)
	require.Error(t, err)
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
					finish.Guard = &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Present{Present: &testpilotspb.ProgramPresentExpression{Operand: proto.CloneOf(finish.Instruction.GetFinish().Result)}}}
				case "false all":
					finish.Guard = &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_All{All: &testpilotspb.ProgramAllExpression{Operands: []*testpilotspb.ProgramExpression{boolean(false), finish.Guard}}}}
				case "true":
					finish.Guard = boolean(true)
					finish.Instruction.GetFinish().Result.GetOutcome().Field = testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS
				default:
					t.Fatalf("unknown guard mode %q", mode)
				}
			})
			state := newState(t, plan)
			input, enabled, err := state.Evaluate(t.Context(), 2)
			if mode == "true" {
				require.Error(t, err)
				require.Equal(t, plan.RuntimeWorkLimit()-6, state.remaining)
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
				require.Equal(t, "present", evaluateEnabled(t, state, 2).GetText())
			}
		})
	}
}

func boolean(value bool) *testpilotspb.ProgramExpression {
	return &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: value}}}}
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
	require.Equal(t, "result", first.GetText())
	first.Value = success("mutated input").Value.Value
	require.Equal(t, "result", evaluateEnabled(t, state, 3).GetText())
}
