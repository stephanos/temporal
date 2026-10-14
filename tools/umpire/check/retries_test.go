package check

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/interp"
	"google.golang.org/protobuf/proto"
)

func TestRetriesFixtureQueriesVerifyWithinTheirLimits(t *testing.T) {
	m := lifted(t, "retryCapabilities")
	want := map[string]bool{}
	for _, binding := range []string{"waitingFailures.eligible", "waitingFailures.network", "controlledFailures.eligible", "controlledFailures.fatal"} {
		for _, property := range []string{"attemptCountIsWithinPolicy", "failureEndsFailed", "failureReturnsToWaiting"} {
			want[binding+"."+property] = true
		}
	}
	for _, binding := range []string{"controlledFailures.eligible", "controlledFailures.fatal"} {
		for _, property := range []string{"failurePauses", "failureCancels"} {
			want[binding+"."+property] = true
		}
	}
	want["waitingFailures.terminalStatesAreFinal"] = true
	want["waitingFailures.closedIsRejectedUniformly"] = true
	require.Len(t, m.GetQueries(), 18)
	r := checked(t, m)
	require.Len(t, r.Receipts, 18)
	for _, receipt := range r.Receipts {
		require.True(t, want[receipt.Key.Name], receipt.Key.Name)
		delete(want, receipt.Key.Name)
		require.Equal(t, Verified, receipt.Kind, receipt.Explanation)
		require.True(t, receipt.Exercised, receipt.Key.Name)
		q := admQuery(m, receipt.Key.Name)
		require.Equal(t, umpirespb.Query_FORM_VERIFY, q.GetForm())
		require.True(t, admScenario(m, q.GetScenario().GetMachine(), q.GetScenario().GetName()).GetFree())
		require.Empty(t, receipt.Holes)
	}
	require.Empty(t, want)

	reader, err := NewRealizer(m, DefaultScope)
	require.NoError(t, err)
	for _, tc := range []struct {
		query, row, after string
	}{
		{"waitingFailures.eligible.failureReturnsToWaiting", "waiting-0-0-fail", "waiting-1-0"},
		{"waitingFailures.eligible.failureEndsFailed", "waiting-1-0-fail", "failed-1-0"},
		{"waitingFailures.network.failureReturnsToWaiting", "waiting-0-2-fault", "waiting-0-2"},
		{"controlledFailures.eligible.failureCancels", "waiting-1-false-true-fail", "canceled-1-false-true"},
	} {
		q := admQuery(m, tc.query)
		bound, err := reader.Bound(ClaimKey{Family: "fixture.retrycapabilities", Owner: q.GetScenario().GetMachine(), Name: tc.query})
		require.NoError(t, err)
		i := slices.IndexFunc(bound.Table.Rows, func(row interp.Row) bool { return row.Key == tc.row })
		require.GreaterOrEqual(t, i, 0, tc.row)
		row := bound.Table.Rows[i]
		require.Len(t, row.Results, 1)
		require.Equal(t, tc.after, row.Results[0].State)
		require.True(t, bound.Property.About(row.Action))
		held, err := bound.Property.Holds(row.Source, row.Results[0])
		require.NoError(t, err)
		require.True(t, held, tc.query)
	}
}

func retryLanding(m *umpirespb.Model, name, phaseType, phase string, guard *umpirespb.Expr, count *int64) {
	f := functionNamed(m, "fixture.retrycapabilities."+name)
	s := expr(f.GetParams()[0].GetName())
	landing := proto.Clone(f.GetBody().GetIf().GetThen()).(*umpirespb.Expr)
	for _, update := range landing.GetList().GetItems()[0].GetConstruct().GetArgs()[1].GetCopy().GetUpdates() {
		switch update.GetName() {
		case "phase":
			update.Value = expr(caseOf("fixture.retrycapabilities."+phaseType, phase))
		case "attempts":
			if count != nil {
				update.Value = expr(admIntValue(*count))
			}
		}
	}
	waiting := binary(umpirespb.Binary_OP_EQ, field(s, "phase"), expr(caseOf("fixture.retrycapabilities."+phaseType, "waiting")))
	f.Body = expr(&umpirespb.If{Condition: binary(umpirespb.Binary_OP_AND, waiting, guard), Then: landing, Else: f.GetBody()})
}

func TestRetriesFixtureMutantsHaveFreshReplayableCounterexamples(t *testing.T) {
	count := func(n int64) *umpirespb.Expr {
		return binary(umpirespb.Binary_OP_EQ, field(expr("s"), "attempts"), expr(admIntValue(n)))
	}
	pause, cancel := field(expr("s"), "pendingPause"), field(expr("s"), "pendingCancel")
	and := func(a, b *umpirespb.Expr) *umpirespb.Expr { return binary(umpirespb.Binary_OP_AND, a, b) }
	absent := func(x *umpirespb.Expr) *umpirespb.Expr {
		return binary(umpirespb.Binary_OP_EQ, x, expr(boolValue(false)))
	}
	ordinary := and(count(0), and(absent(pause), absent(cancel)))
	two := int64(2)
	for _, tc := range []struct {
		name, function, phaseType, phase, query string
		guard                                   *umpirespb.Expr
		count                                   *int64
	}{
		{"wrong ordinary", "ControlSteps$.fail", "ControlPhase", "canceled", "controlledFailures.eligible.failureReturnsToWaiting", ordinary, nil},
		{"premature failure", "ControlSteps$.fail", "ControlPhase", "failed", "controlledFailures.eligible.failureReturnsToWaiting", ordinary, nil},
		{"wrong pause", "ControlSteps$.fail", "ControlPhase", "waiting", "controlledFailures.eligible.failurePauses", and(count(0), and(pause, absent(cancel))), nil},
		{"wrong cancellation priority", "ControlSteps$.fail", "ControlPhase", "suspended", "controlledFailures.eligible.failureCancels", and(count(0), and(pause, cancel)), nil},
		{"fatal with controls", "ControlSteps$.fatal", "ControlPhase", "canceled", "controlledFailures.fatal.failureEndsFailed", and(pause, cancel), nil},
		{"extra retry after exhaustion", "ControlSteps$.fail", "ControlPhase", "waiting", "controlledFailures.eligible.failureEndsFailed", and(count(1), and(absent(pause), absent(cancel))), nil},
		{"pause at exhaustion", "ControlSteps$.fail", "ControlPhase", "suspended", "controlledFailures.eligible.failureEndsFailed", and(count(1), and(pause, absent(cancel))), nil},
		{"exhausted cancellation", "ControlSteps$.fail", "ControlPhase", "failed", "controlledFailures.eligible.failureCancels", and(count(1), cancel), nil},
		{"policy one count two", "WaitingSteps$.fail", "SimplePhase", "failed", "waitingFailures.eligible.attemptCountIsWithinPolicy", count(1), &two},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := proto.Clone(lifted(t, "retryCapabilities")).(*umpirespb.Model)
			retryLanding(m, tc.function, tc.phaseType, tc.phase, tc.guard, tc.count)
			q := admQuery(m, tc.query)
			r := receiptOf(t, checked(t, m), "query "+q.GetScenario().GetMachine()+" "+tc.query)
			require.Equal(t, Counterexample, r.Kind, r.Explanation)
			require.True(t, r.Exercised)
			require.NotNil(t, r.Witness)
			require.NotEmpty(t, r.Witness.Steps)
			if tc.count != nil {
				require.Equal(t, []string{"waiting-0-0-fail", "waiting-1-0-fail"}, r.Rows)
				require.Equal(t, "failed-2-0", r.Witness.Steps[len(r.Witness.Steps)-1].State.Value)
			}
			fresh, err := NewRealizer(m, DefaultScope)
			require.NoError(t, err)
			query, err := fresh.Find(r.Key)
			require.NoError(t, err)
			a, err := query.Answer()
			require.NoError(t, err)
			require.Equal(t, r.Witness, a.Witness)
			require.NoError(t, query.Replay(a))
		})
	}
}

func TestRetriesFixtureFiniteTwoAllowsReachableCountTwo(t *testing.T) {
	m := proto.Clone(lifted(t, "retryCapabilities")).(*umpirespb.Model)
	two := int64(2)
	countOne := binary(umpirespb.Binary_OP_EQ, field(expr("s"), "attempts"), expr(admIntValue(1)))
	retryLanding(m, "WaitingSteps$.fail", "SimplePhase", "failed", countOne, &two)
	maximum := functionNamed(m, "fixture.retrycapabilities.WaitingSteps$.maxOne")
	maximum.Body.GetConstruct().Args[0] = expr(admIntValue(2))
	name := "waitingFailures.eligible.attemptCountIsWithinPolicy"
	r := receiptOf(t, checked(t, m), "query waitingFailures "+name)
	require.Equal(t, Verified, r.Kind, r.Explanation)
	require.True(t, r.Exercised)
	fresh, err := NewRealizer(m, DefaultScope)
	require.NoError(t, err)
	bound, err := fresh.Bound(r.Key)
	require.NoError(t, err)
	i := slices.IndexFunc(bound.Table.Rows, func(row interp.Row) bool { return row.Key == "waiting-1-0-fail" })
	require.GreaterOrEqual(t, i, 0)
	row := bound.Table.Rows[i]
	require.Len(t, row.Results, 1)
	require.Equal(t, "failed-2-0", row.Results[0].State)
	query, err := fresh.Find(r.Key)
	require.NoError(t, err)
	a, err := query.Answer()
	require.NoError(t, err)
	require.Equal(t, VerifiedWithinLimits, a.Outcome)
	require.Nil(t, a.Witness)
	require.True(t, a.Exercised)
}
