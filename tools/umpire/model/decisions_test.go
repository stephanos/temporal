package model

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
	m := activityModel(t)

	// A pause of a paused activity is disabled by the arm of its phase.
	w := whyIn(t, m, "activityProtocol", "paused-1-unset-unset-unset", "control-pause")
	require.Empty(t, w.Steps)
	require.Nil(t, w.Hole)
	last, ok := w.Last()
	require.True(t, ok)
	require.True(t, last.Match())
	require.False(t, last.Wildcard)
	require.True(t, last.State)
	require.Contains(t, last.Position, "model/temporal/standaloneactivity/Model.scala:")
	// The control's own match decided first, on the input alone.
	var input Decision
	for _, d := range w.Decisions {
		if d.Match() && d.Expr.GetMatch().GetScrutinee().GetVar() == "c" {
			input = d
		}
	}
	require.NotNil(t, input.Expr)
	require.False(t, input.State)

	// An unstarted activity has nothing to control: an `if` over the phase.
	w = whyIn(t, m, "activityProtocol", "unstarted-0-unset-unset-unset", "control-pause")
	last, ok = w.Last()
	require.True(t, ok)
	require.False(t, last.Match())
	require.True(t, last.Then)
	require.True(t, last.State)

	// A terminal phase's notFound row decides through the named predicate it calls.
	w = whyIn(t, m, "activityProtocol", "completed-1-unset-unset-unset", "control-pause")
	require.Len(t, w.Steps, 1)
	first := w.Decisions[0]
	require.Equal(t, []string{"temporal.standaloneactivity.Protocol$.terminal"}, first.Calls)
	require.True(t, first.State)
	require.False(t, first.Nested)
}

func TestWhyTellsAnInputDecisionFromAStateDecision(t *testing.T) {
	m := activityModel(t)
	// A non-retryable failure decides on the input, after `held` decided on the phase.
	w := whyIn(t, m, "activityProtocol", "started-1-unset-unset-unset", "attemptResult-failed-false")
	require.Len(t, w.Steps, 1)
	var sawInput bool
	for _, d := range w.Decisions {
		if !d.Match() && !d.State {
			sawInput = true
		}
	}
	require.True(t, sawInput)
}

func TestWhyMarksAWildcardArm(t *testing.T) {
	m := proto.Clone(activityModel(t)).(*umpirespb.Model)
	// Rewrite the pause arm for cancelRequested as a default arm, as `case _ => Nil` lifts.
	var rewritten bool
	var visit func(x *umpirespb.Expr)
	visit = func(x *umpirespb.Expr) {
		if x == nil {
			return
		}
		switch k := x.GetKind().(type) {
		case *umpirespb.Expr_If:
			visit(k.If.GetCondition())
			visit(k.If.GetThen())
			visit(k.If.GetElse())
		case *umpirespb.Expr_Match:
			for _, c := range k.Match.GetCases() {
				if lit := c.GetPattern().GetLiteral().GetEnum(); lit.GetCase() == "cancelRequested" && !rewritten &&
					len(c.GetBody().GetList().GetItems()) == 0 && c.GetBody().GetList() != nil {
					c.Pattern = &umpirespb.Pattern{Kind: &umpirespb.Pattern_Wildcard{Wildcard: &umpirespb.Empty{}}}
					rewritten = true
				}
				visit(c.GetBody())
			}
		default:
		}
	}
	for _, f := range m.GetFunctions() {
		if f.GetName() == "temporal.standaloneactivity.Protocol$.control" {
			visit(f.GetBody())
		}
	}
	require.True(t, rewritten)
	w := whyIn(t, m, "activityProtocol", "cancelRequested-1-unset-unset-unset", "control-pause")
	last, ok := w.Last()
	require.True(t, ok)
	require.True(t, last.Wildcard)
	require.Empty(t, w.Steps)
}

func TestWhyRefusesAnUnknownStateOrClass(t *testing.T) {
	m := activityModel(t)
	machines, err := Build(m)
	require.NoError(t, err)
	in := NewInterpreter(m)
	_, err = in.Why(machines["activityProtocol"], "nowhere", "control-pause")
	require.ErrorContains(t, err, "no state nowhere")
	_, err = in.Why(machines["activityProtocol"], "paused-1-unset-unset-unset", "control-nothing")
	require.ErrorContains(t, err, "no class control-nothing")
}

func TestReadsSaysWhetherAClaimReadsItsStep(t *testing.T) {
	m := activityModel(t)
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

func TestRealizerGivesTheRefinementCheckReads(t *testing.T) {
	r, err := NewRealizer(activityModel(t), DefaultScope)
	require.NoError(t, err)
	rows, err := r.Refinement("activityProtocol")
	require.NoError(t, err)
	carried := map[string]string{}
	for _, row := range rows {
		if row.Product != nil {
			carried[row.Key] = *row.Product
		}
	}
	require.Equal(t, "attemptStart", carried["scheduled-0-unset-unset-unset-attemptStart"])
	_, err = r.Refinement("activityProduct")
	require.ErrorContains(t, err, "refines no machine")
}
