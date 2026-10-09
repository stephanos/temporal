package interp

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

func whyIn(t *testing.T, m *umpirespb.Model, machine, state, class string) *Why {
	t.Helper()
	machines, err := Build(m)
	require.NoError(t, err)
	w, err := NewInterpreter(m).Why(machines[machine], state, class)
	require.NoError(t, err)
	return w
}

func TestWhyExplainsTheDecisionThatDisabledAPair(t *testing.T) {
	m := readIR(t, activityIR)

	// A pause of a paused activity fires no rule: each rule of the input-free action is tried on the
	// state, and the last one tried disabled the pair.
	w := whyIn(t, m, "activitySystem", "paused-1-unset-unset-unset", "pause")
	require.Empty(t, w.Steps)
	require.Nil(t, w.Hole)
	last, ok := w.Last()
	require.True(t, ok)
	require.False(t, last.Match())
	require.False(t, last.Then)
	require.True(t, last.State)
	require.Contains(t, last.Position, "model/temporal/features/activity/standalone/system/System.scala:")
	// Splitting control into one action per RPC removes the old input Match from this step function.
	for _, decision := range w.Decisions {
		require.False(t, decision.Match())
		require.True(t, decision.State)
	}

	// An unstarted activity has nothing to control: no rule's phase holds.
	w = whyIn(t, m, "activitySystem", "unstarted-0-unset-unset-unset", "pause")
	last, ok = w.Last()
	require.True(t, ok)
	require.False(t, last.Match())
	require.False(t, last.Then)
	require.True(t, last.State)

	// A terminal phase's notFound row decides through its role-expanded phase guard. Role expansion
	// leaves no helper call: the decision itself still records the source guard and its result.
	w = whyIn(t, m, "activitySystem", "completed-1-unset-unset-unset", "pause")
	require.Len(t, w.Steps, 1)
	guard := w.Decisions[0]
	require.Empty(t, guard.Calls)
	require.True(t, guard.Then)
	require.True(t, guard.State)
	require.False(t, guard.Nested)
}

func TestWhyTellsAnInputDecisionFromAStateDecision(t *testing.T) {
	m := readIR(t, activityIR)
	// A non-retryable failure decides on the input, then on the phase its rule names (`held`).
	w := whyIn(t, m, "activitySystem", "started-1-unset-unset-unset", "respondFailed-fatal")
	require.Len(t, w.Steps, 1)
	var sawInput, sawState bool
	for _, d := range w.Decisions {
		sawInput = sawInput || d.Match() && !d.State
		sawState = sawState || !d.Match() && d.State
	}
	require.True(t, sawInput)
	require.True(t, sawState)
}

func TestWhyMarksAWildcardArm(t *testing.T) {
	m := proto.Clone(readIR(t, activityIR)).(*umpirespb.Model)
	// Rewrite the pause's input-free step function as a default arm that fires none, as
	// `case _ => Nil` lifts.
	var rewritten bool
	for _, f := range m.GetFunctions() {
		if f.GetName() != "activitySystem.rules.pause" {
			continue
		}
		f.Body = &umpirespb.Expr{Position: f.GetBody().GetPosition(), Kind: &umpirespb.Expr_Match{Match: &umpirespb.Match{
			Scrutinee: &umpirespb.Expr{Kind: &umpirespb.Expr_Field{Field: &umpirespb.FieldAccess{
				Base: &umpirespb.Expr{Kind: &umpirespb.Expr_Var{Var: "s"}}, Field: "phase"}}},
			Cases: []*umpirespb.MatchCase{{
				Pattern: &umpirespb.Pattern{Kind: &umpirespb.Pattern_Wildcard{Wildcard: &emptypb.Empty{}}},
				Body:    &umpirespb.Expr{Kind: &umpirespb.Expr_List{List: &umpirespb.ListOf{}}},
			}},
		}}}
		rewritten = true
	}
	require.True(t, rewritten)
	w := whyIn(t, m, "activitySystem", "cancelRequested-1-unset-unset-unset", "pause")
	last, ok := w.Last()
	require.True(t, ok)
	require.True(t, last.Wildcard)
	require.Empty(t, w.Steps)
}

func TestWhyRefusesAnUnknownStateOrClass(t *testing.T) {
	m := readIR(t, activityIR)
	machines, err := Build(m)
	require.NoError(t, err)
	in := NewInterpreter(m)
	_, err = in.Why(machines["activitySystem"], "nowhere", "pause")
	require.ErrorContains(t, err, "no state nowhere")
	_, err = in.Why(machines["activitySystem"], "paused-1-unset-unset-unset", "control-nothing")
	require.ErrorContains(t, err, "no class control-nothing")
}

func TestReadsSaysWhetherAClaimReadsItsStep(t *testing.T) {
	m := readIR(t, activityIR)
	machines, err := Build(m)
	require.NoError(t, err)
	product := machines["activityProduct"]
	in := NewInterpreter(m)
	property := "activityProduct.property.activityProduct.pausedIsNotDispatched"
	step := func(state string) Value {
		for _, tr := range product.Transitions {
			if tr.Source.Key() == state {
				return tr.Steps[0]
			}
		}
		t.Fatalf("no row from %s", state)
		return Value{}
	}
	// Away from paused the Property is decided by the state before alone.
	scheduled, _ := product.State("scheduled")
	v, read, err := in.Reads(property, []Value{scheduled, step("scheduled")}, nil)
	require.NoError(t, err)
	require.True(t, v.Bool)
	require.Equal(t, []bool{true, false}, read)
	// From paused it reads where the step goes.
	paused, _ := product.State("paused")
	_, read, err = in.Reads(property, []Value{paused, step("paused")}, nil)
	require.NoError(t, err)
	require.Equal(t, []bool{true, true}, read)
}
