package check

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
)

func TestClaimReportsKeepSafetyAndWitnessMeaning(t *testing.T) {
	for _, tc := range []struct {
		name  string
		model func() *umpirespb.Model
		form  umpirespb.Query_Form
		kind  ReceiptKind
	}{
		{"postcondition", func() *umpirespb.Model { return counter{k: 2}.model() }, umpirespb.Query_FORM_VERIFY, Verified},
		{"transition", func() *umpirespb.Model { return selectedCounter(true) }, umpirespb.Query_FORM_VERIFY, Verified},
		{"monitor", func() *umpirespb.Model { return counter{k: 2, monitor: true}.model() }, umpirespb.Query_FORM_VERIFY, Verified},
		{"witness", func() *umpirespb.Model { return counter{k: 2}.model() }, umpirespb.Query_FORM_FIND, Found},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := tc.model()
			m.Queries[0].Form = tc.form
			x := receiptOf(t, Check(m, DefaultScope), "query counter counter.all")
			require.Equal(t, tc.kind, x.Kind, x.Explanation)
			require.Equal(t, Safety, x.Classification())
			require.Equal(t, QuerySubject, x.Subject)
			require.Empty(t, x.Part)
			require.Zero(t, x.Within)
			require.Equal(t, "generic:70", x.Position)
			require.Equal(t, ClaimKey{Family: "generic", Owner: "counter", Name: "always"}, x.Property)
			require.Equal(t, Limits{Name: "wide", Steps: 8, Search: 1 << 20}, x.Limits)
			require.True(t, x.Exercised)
			if tc.kind == Found {
				require.NotNil(t, x.Witness)
				require.NotEmpty(t, x.Witness.Steps)
			}
		})
	}
}

func TestProgressReceiptKeepsTheDeclaredBoundSeparateFromSearch(t *testing.T) {
	m := counter{k: 2}.model()
	m.Assumptions = []*umpirespb.Assumption{
		{Id: "generic.tableReady", Name: "tableReady", Position: at(81)},
		{Id: "generic.ready", Name: "ready", Position: at(82)},
		{Id: "generic.recovered", Name: "recovered", Position: at(83)},
	}
	m.Machines[0].Assumes = []string{"generic.tableReady"}
	m.Progress[0].Assumptions = []string{"generic.recovered", "generic.ready"}
	scope := DefaultScope
	scope.Progress = Limits{Name: "report search", Steps: 9, Search: 128}
	r := Check(m, scope)
	for _, part := range []string{"deadlock", "fair-cycle", "deadline"} {
		x := receiptOf(t, r, "progress counter reachesTop "+part)
		require.Equal(t, Verified, x.Kind, x.Explanation)
		require.True(t, x.Exercised)
		require.Equal(t, scope.Progress, x.Limits)
		require.Equal(t, int32(2), x.Within)
		require.NotEqual(t, int(x.Within), x.Limits.Steps)
		require.Equal(t, BoundedLiveness, x.Classification())
		require.Equal(t, "generic:80", x.Position)
		require.Equal(t, []string{"tableReady", "recovered", "ready"}, x.Assumptions)
	}
}

func TestProgressReportingKeepsIndependentSafetyViolations(t *testing.T) {
	for _, tc := range []struct {
		name        string
		prepare     func(*umpirespb.Model, *Scope) *umpirespb.Progress
		kind        ReceiptKind
		why         string
		assumptions []string
	}{
		{"predicate hole", func(m *umpirespb.Model, _ *Scope) *umpirespb.Progress {
			m.Holes = []*umpirespb.Hole{{Id: "generic.progressUnknown", Name: "progressUnknown", Position: at(84)}}
			functionNamed(m, "generic.atZero").Body = &umpirespb.Expr{Position: at(85), Kind: &umpirespb.Expr_Hole{Hole: "generic.progressUnknown"}}
			return m.Progress[0]
		}, Incomplete, "progressUnknown", []string{"tableReady", "recovered", "ready"}},
		{"predicate prerequisite", func(m *umpirespb.Model, _ *Scope) *umpirespb.Progress {
			functionNamed(m, "generic.atZero").Requires = expr(boolValue(false))
			return m.Progress[0]
		}, DeclarationError, "outside its precondition", []string{"tableReady", "recovered", "ready"}},
		{"unsupported composition", func(m *umpirespb.Model, _ *Scope) *umpirespb.Progress {
			watched := counter{k: 1, monitor: true}.model()
			m.Functions = append(m.Functions, watched.Functions[len(watched.Functions)-2:]...)
			m.Monitors, m.Machines[0].Monitors = watched.Monitors, watched.Machines[0].Monitors
			return m.Progress[1]
		}, Unsupported, "names monitors", []string{"recovered", "ready"}},
		{"search exhausted", func(m *umpirespb.Model, scope *Scope) *umpirespb.Progress {
			scope.Progress.Search = 0
			return m.Progress[0]
		}, LimitReached, "", []string{"tableReady", "recovered", "ready"}},
		{"depth exhausted", func(m *umpirespb.Model, scope *Scope) *umpirespb.Progress {
			scope.Progress.Steps = 0
			return m.Progress[0]
		}, Unresolved, "", []string{"tableReady", "recovered", "ready"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, scope := composedProgressModel(), DefaultScope
			m.Assumptions = []*umpirespb.Assumption{
				{Id: "generic.tableReady", Name: "tableReady", Position: at(81)},
				{Id: "generic.ready", Name: "ready", Position: at(82)},
				{Id: "generic.recovered", Name: "recovered", Position: at(83)},
			}
			m.Machines[0].Assumes = []string{"generic.tableReady"}
			functionNamed(m, "generic.always").Body = expr(boolValue(false))
			p := tc.prepare(m, &scope)
			p.Within, p.Assumptions = 3, []string{"generic.recovered", "generic.ready"}
			r := Check(m, scope)
			safety := receiptOf(t, r, "query counter counter.all")
			require.Equal(t, Counterexample, safety.Kind, safety.Explanation)
			require.Equal(t, Safety, safety.Classification())
			require.Equal(t, "generic:70", safety.Position)
			require.Equal(t, []string{"tableReady"}, safety.Assumptions)
			require.Equal(t, []string{"tick"}, taken(safety.Witness))
			require.Equal(t, []string{"0-tick"}, safety.Rows)
			key := "progress " + p.Machine + " " + p.Name
			parts := []string{"deadlock", "fair-cycle", "deadline"}
			if tc.kind == Unsupported || tc.kind == DeclarationError {
				parts = []string{""}
			}
			for _, part := range parts {
				atKey := key
				if part != "" {
					atKey += " " + part
				}
				x := receiptOf(t, r, atKey)
				require.Equal(t, tc.kind, x.Kind, x.Explanation)
				require.Equal(t, BoundedLiveness, x.Classification())
				require.Equal(t, "generic:"+map[string]string{"counter": "80", "two": "83"}[p.Machine], x.Position)
				require.Equal(t, int32(3), x.Within)
				require.Equal(t, scope.Progress, x.Limits)
				require.Equal(t, tc.assumptions, x.Assumptions)
				require.Contains(t, x.Explanation, tc.why)
				require.Nil(t, x.Witness)
				if tc.kind == Incomplete {
					require.NotEmpty(t, x.Holes)
				}
				if tc.kind == Unsupported {
					require.Contains(t, r.Unsupported(), x)
					require.NotContains(t, r.Checks(), x)
				}
			}
		})
	}
}
