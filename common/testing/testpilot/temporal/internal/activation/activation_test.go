package activation

import (
	"context"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport/facadetest"
	"google.golang.org/protobuf/proto"
)

func preparedRuntimeFixture(t *testing.T, modify ...func(*testpilotspb.Program)) testpilot.PreparedProgram {
	t.Helper()
	return facadetest.Capture(t, facadetest.RuntimeCase(t, facadetest.SyncReply, []enumspb.CommandType{enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION}, nil, modify...))
}

func TestConstruction(t *testing.T) {
	state, err := New(testpilot.EntrypointPlan{})
	require.Error(t, err)
	require.Nil(t, state)
	program := preparedRuntimeFixture(t)
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
	plan, ok := findEntrypoint(preparedRuntimeFixture(t, modify...), "workflow")
	require.True(t, ok)
	return plan
}

func success(value string) *testpilotspb.InstructionOutcome {
	return &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, Value: facadetest.CarriedValue(value)}
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
			require.Nil(t, state.lookup(testpilot.ValueReference{Kind: testpilot.SlotReference, ID: "private-handle"}))
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
				require.Equal(t, "result", facadetest.CarriedText(t, state.values[testpilot.ValueReference{Kind: testpilot.OutcomeReference, Entrypoint: "workflow", ID: "await", Field: int32(testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE)}]), "only successful outcomes have a result")
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
				require.Equal(t, "result", facadetest.CarriedText(t, input))
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
		require.Equal(t, value, facadetest.CarriedText(t, evaluateEnabled(t, state, 2)))
	}
	exercise(t, "first")
	exercise(t, "second")
	for i := range 10 {
		t.Run(fmt.Sprint(i), func(t *testing.T) { t.Parallel(); exercise(t, fmt.Sprint(i)) })
	}
	snapshot := plan.Instructions()[0].Source()
	snapshot.Instruction.GetWorkflowCommand().GetCommand().GetScheduleNexusOperationCommandAttributes().Input = facadetest.Payload("mutated")
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
				require.Equal(t, "present", facadetest.CarriedText(t, evaluateEnabled(t, state, 2)))
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
	require.Equal(t, "result", facadetest.CarriedText(t, first))
	first.Value = success("mutated input").Value.Value
	require.Equal(t, "result", facadetest.CarriedText(t, evaluateEnabled(t, state, 3)))
}
