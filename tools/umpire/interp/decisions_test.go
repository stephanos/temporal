package interp

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

func whyIn(t *testing.T, m *umpirespb.Model, machine, state, class string) *Why {
	t.Helper()
	machines, err := Build(m)
	require.NoError(t, err)
	w, err := NewInterpreter(m).Why(machines[machine], state, class)
	require.NoError(t, err)
	return w
}

func TestWhyNamesTheDecisionThatDisabledAPair(t *testing.T) {
	m := readIR(t, activityIR)

	// A pause of a paused activity fires no rule: the control's rules are matched on the input first,
	// then each rule of the class is tried on the state, and the last one tried disabled the pair.
	w := whyIn(t, m, "activitySystem", "paused-1-unset-unset-unset", "control-pause")
	require.Empty(t, w.Steps)
	require.Nil(t, w.Hole)
	last, ok := w.Last()
	require.True(t, ok)
	require.False(t, last.Match())
	require.False(t, last.Then)
	require.True(t, last.State)
	require.Contains(t, last.Position, "model/temporal/features/standaloneactivity/system/System.scala:")
	// The match on the input decided first, on the input alone.
	input := w.Decisions[0]
	require.True(t, input.Match())
	require.Equal(t, "control", input.Expr.GetMatch().GetScrutinee().GetVar())
	require.False(t, input.State)

	// An unstarted activity has nothing to control: no rule's phase holds.
	w = whyIn(t, m, "activitySystem", "unstarted-0-unset-unset-unset", "control-pause")
	last, ok = w.Last()
	require.True(t, ok)
	require.False(t, last.Match())
	require.False(t, last.Then)
	require.True(t, last.State)

	// A terminal phase's notFound row decides through the named predicate its rule's guard calls.
	w = whyIn(t, m, "activitySystem", "completed-1-unset-unset-unset", "control-pause")
	require.Len(t, w.Steps, 1)
	guard := w.Decisions[1]
	require.Equal(t, []string{"temporal.features.standaloneactivity.system.ActivitySystem$.states$.terminal"}, guard.Calls)
	require.True(t, guard.Then)
	require.True(t, guard.State)
	require.False(t, guard.Nested)
}

func TestWhyTellsAnInputDecisionFromAStateDecision(t *testing.T) {
	m := readIR(t, activityIR)
	// A non-retryable failure decides on the input, then on the phase its rule names (`held`).
	w := whyIn(t, m, "activitySystem", "started-1-unset-unset-unset", "respond-failed-false")
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
	// Rewrite the control's rules for a pause as a default arm that fires none, as `case _ => Nil`
	// lifts.
	var rewritten bool
	for _, f := range m.GetFunctions() {
		if f.GetName() != "activitySystem.rules.control" {
			continue
		}
		for _, c := range f.GetBody().GetMatch().GetCases() {
			if c.GetPattern().GetLiteral().GetEnum().GetCase() == "pause" {
				c.Pattern = &umpirespb.Pattern{Kind: &umpirespb.Pattern_Wildcard{Wildcard: &umpirespb.Empty{}}}
				c.Body = &umpirespb.Expr{Kind: &umpirespb.Expr_List{List: &umpirespb.ListOf{}}}
				rewritten = true
			}
		}
	}
	require.True(t, rewritten)
	w := whyIn(t, m, "activitySystem", "cancelRequested-1-unset-unset-unset", "control-pause")
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
	_, err = in.Why(machines["activitySystem"], "nowhere", "control-pause")
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
	law := "activityProduct.property.activityProduct.pausedIsNotDispatched"
	step := func(state string) Value {
		for _, tr := range product.Transitions {
			if tr.Source.Key() == state {
				return tr.Steps[0]
			}
		}
		t.Fatalf("no row from %s", state)
		return Value{}
	}
	// Away from paused the law is decided by the state before alone.
	scheduled, _ := product.State("scheduled")
	v, read, err := in.Reads(law, []Value{scheduled, step("scheduled")}, nil)
	require.NoError(t, err)
	require.True(t, v.Bool)
	require.Equal(t, []bool{true, false}, read)
	// From paused it reads where the step goes.
	paused, _ := product.State("paused")
	_, read, err = in.Reads(law, []Value{paused, step("paused")}, nil)
	require.NoError(t, err)
	require.Equal(t, []bool{true, true}, read)
}
