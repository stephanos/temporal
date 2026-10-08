package check

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/interp"
)

func selectedCounter(action bool) *umpirespb.Model {
	m := counter{k: 2, skip: true}.model()
	p := admProperty(m, "counter", "always")
	p.Transition = true
	if action {
		p.When = &umpirespb.Property_WhenAction{WhenAction: "tick"}
	} else {
		p.When = &umpirespb.Property_WhenClass{WhenClass: &umpirespb.ActionClass{Action: "generic.tick"}}
	}
	f := functionNamed(m, "generic.always")
	f.Params = append([]*umpirespb.Param{{Name: "before", Type: interp.Named("Counter")}}, f.GetParams()...)
	f.Requires = binary(umpirespb.Binary_OP_EQ, field(expr("after"), "outcome"), expr(caseOf("O", "ok")))
	f.Body = binary(umpirespb.Binary_OP_EQ, field(field(expr("after"), "state"), "n"),
		binary(umpirespb.Binary_OP_ADD, field(expr("before"), "n"), expr(admIntValue(1))))
	return m
}

func TestSelectedTransitionsReadBeforeStateAndSkipUnrelatedErrors(t *testing.T) {
	for _, action := range []bool{false, true} {
		name := "class"
		if action {
			name = "action"
		}
		t.Run(name, func(t *testing.T) {
			m := selectedCounter(action)
			r := receiptOf(t, checked(t, m), "query counter counter.all")
			require.Equal(t, Verified, r.Kind, r.Explanation)
			require.True(t, r.Exercised)
			reader, err := NewRealizer(m, DefaultScope)
			require.NoError(t, err)
			bound, err := reader.Bound(r.Key)
			require.NoError(t, err)
			require.True(t, bound.Property.About("tick"))
			require.False(t, bound.Property.About("skip"))
		})
	}
}

func TestSelectedTransitionsKeepExerciseErrorsLimitsAndRefusal(t *testing.T) {
	for name, tc := range map[string]struct {
		mutate    func(*umpirespb.Model)
		kind      ReceiptKind
		exercised bool
	}{
		"never selected": {func(m *umpirespb.Model) {
			s := admScenario(m, "counter", "all")
			s.Free, s.Actions = false, []*umpirespb.ActionClass{{Action: "generic.skip"}}
			admQuery(m, "counter.all").GetLimits().Actions = 1
		}, Verified, false},
		"selected error": {func(m *umpirespb.Model) {
			functionNamed(m, "generic.always").Requires = expr(boolValue(false))
		}, DeclarationError, false},
		"search limit": {func(m *umpirespb.Model) {
			admQuery(m, "counter.all").GetLimits().Search = 1
		}, LimitReached, false},
		"transition find": {func(m *umpirespb.Model) {
			admQuery(m, "counter.all").Form = umpirespb.Query_FORM_FIND
		}, DeclarationError, false},
	} {
		t.Run(name, func(t *testing.T) {
			m := selectedCounter(true)
			tc.mutate(m)
			r := receiptOf(t, checked(t, m), "query counter counter.all")
			require.Equal(t, tc.kind, r.Kind, r.Explanation)
			require.Equal(t, tc.exercised, r.Exercised)
			if tc.kind == DeclarationError {
				require.NotNil(t, r.Cause)
				require.NotEmpty(t, r.Position)
			}
		})
	}
}

func TestSelectedTransitionCounterexampleReplaysFresh(t *testing.T) {
	m := selectedCounter(false)
	functionNamed(m, "generic.always").Body = binary(umpirespb.Binary_OP_LT, field(expr("before"), "n"), expr(admIntValue(1)))
	r := receiptOf(t, checked(t, m), "query counter counter.all")
	require.Equal(t, Counterexample, r.Kind, r.Explanation)
	require.True(t, r.Exercised)
	require.Equal(t, []string{"0-tick", "1-tick"}, r.Rows)
	require.NotEmpty(t, r.Witness.Steps)
	fresh, err := NewRealizer(m, DefaultScope)
	require.NoError(t, err)
	q, err := fresh.Find(r.Key)
	require.NoError(t, err)
	a, err := q.Answer()
	require.NoError(t, err)
	require.Equal(t, r.Witness, a.Witness)
	require.NoError(t, q.Replay(a))
}

func TestSelectedTransitionHolesRemainUnknown(t *testing.T) {
	for _, selected := range []bool{false, true} {
		name := "skipped"
		selector := "crash"
		want := Verified
		if selected {
			name, selector, want = "selected", "put", Incomplete
		}
		t.Run(name, func(t *testing.T) {
			m := mutated(t, "declarations", noCrash, func(m *umpirespb.Model) {
				admProperty(m, "disk", "durableStays").When = &umpirespb.Property_WhenAction{WhenAction: selector}
				f := functionNamed(m, "disk.property.durableStays")
				f.Body = &umpirespb.Expr{Position: f.GetBody().GetPosition(), Kind: &umpirespb.Expr_Hole{Hole: crashHole}}
				if !selected {
					// The selector must stay bound even when every row of its action is disabled.
					admMachine(m, "disk").Steps = append(admMachine(m, "disk").Steps,
						&umpirespb.StepBinding{Action: admDeclaredID + "crash", Function: admDeclaredPkg + "crashStep", Position: at(1)})
					function(m, "crashStep").Body = expr(&umpirespb.ListOf{})
				}
			})
			r := receiptOf(t, checked(t, m), "query disk durableStays")
			require.Equal(t, want, r.Kind, r.Explanation)
			if selected {
				require.Equal(t, []HoleReach{{Edge: ClaimHole, ID: crashHole, Name: "crashUnmodeled", Row: "empty-put"}}, holes(r))
			} else {
				require.Empty(t, r.Holes)
				require.False(t, r.Exercised)
			}
		})
	}
}

func TestSelectedTransitionKeepsUnselectedRowHoles(t *testing.T) {
	m := mutated(t, "declarations", func(m *umpirespb.Model) {
		admProperty(m, "disk", "durableStays").When = &umpirespb.Property_WhenAction{WhenAction: "put"}
	})
	r := receiptOf(t, checked(t, m), "query disk durableStays")
	require.Equal(t, Incomplete, r.Kind, r.Explanation)
	require.True(t, r.Exercised)
	require.Equal(t, []HoleReach{{Edge: RowHole, ID: crashHole, Name: "crashUnmodeled", Row: "staged-crash", Depth: 1}}, holes(r))
	require.Equal(t, []string{"put"}, taken(r.Holes[0].Prefix))
}
