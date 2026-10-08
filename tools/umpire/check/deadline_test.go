package check

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/interp"
	"google.golang.org/protobuf/proto"
)

const deadlineFixture = "fixture.deadlinecapabilities."

func TestDeadlineFixtureQueriesVerifyWithinTheirLimits(t *testing.T) {
	m := lifted(t, "deadlineCapabilities")
	want := map[string]bool{}
	for _, binding := range []string{"schedule", "start"} {
		for _, property := range []string{"firesInWindow", "deadlineTimesOut"} {
			want["deadlineTimers."+binding+"."+property] = true
		}
	}
	want["deadlineTimers.start.deadlineReturnsToWaiting"] = true
	want["deadlineTimers.start.deadlinePauses"] = true
	for _, property := range []string{"firesInWindow", "deadlineTimesOut", "deadlineReturnsToWaiting"} {
		want["simpleDeadlines.retry."+property] = true
	}
	require.Len(t, m.GetQueries(), len(want))
	r := checked(t, m)
	require.Len(t, r.Receipts, len(want))
	for _, receipt := range r.Receipts {
		require.True(t, want[receipt.Key.Name], receipt.Key.Name)
		delete(want, receipt.Key.Name)
		require.Equal(t, Verified, receipt.Kind, receipt.Explanation)
		require.True(t, receipt.Exercised, receipt.Key.Name)
		require.Empty(t, receipt.Holes)
		q := admQuery(m, receipt.Key.Name)
		require.Equal(t, umpirespb.Query_FORM_VERIFY, q.GetForm())
		require.True(t, admScenario(m, q.GetScenario().GetMachine(), q.GetScenario().GetName()).GetFree())
	}
	require.Empty(t, want)

	reader, err := NewRealizer(m, DefaultScope)
	require.NoError(t, err)
	for _, tc := range []struct {
		binding, action, before, after string
	}{
		{"schedule", "scheduleExpired-true", "waiting-true-true-true-false-false", "timedOut-true-true-true-false-false"},
		{"start", "startExpired", "held-true-true-true-false-false", "waiting-true-true-false-false-false"},
		{"start", "startExpired", "held-true-false-true-false-false", "waiting-true-false-false-false-false"},
		{"start", "startExpired", "held-true-true-true-true-false", "suspended-true-true-false-true-false"},
	} {
		name := "deadlineTimers." + tc.binding + ".firesInWindow"
		q := admQuery(m, name)
		bound, err := reader.Bound(ClaimKey{Family: admMachine(m, "deadlineTimers").GetFamily(), Owner: q.GetScenario().GetMachine(), Name: name})
		require.NoError(t, err)
		i := slices.IndexFunc(bound.Table.Rows, func(row interp.Row) bool { return row.Key == tc.before+"-"+tc.action })
		require.GreaterOrEqual(t, i, 0, tc.before+"-"+tc.action)
		row := bound.Table.Rows[i]
		require.Len(t, row.Results, 1)
		require.Equal(t, tc.after, row.Results[0].State)
		require.True(t, bound.Property.About(row.Action))
		require.False(t, bound.Property.About("keep"))
		require.Equal(t, tc.binding == "start", bound.Property.About("startExpired"))
		require.Equal(t, tc.binding == "schedule", bound.Property.About("scheduleExpired-true"))
		require.False(t, bound.Property.About("scheduleExpired-false"))
		held, err := bound.Property.Holds(row.Source, row.Results[0])
		require.NoError(t, err)
		require.True(t, held)
	}
}

func deadlineState(phase string, flags ...bool) *umpirespb.Expr {
	args := []*umpirespb.Expr{expr(caseOf(deadlineFixture+"Phase", phase))}
	for _, flag := range flags {
		args = append(args, expr(boolValue(flag)))
	}
	return expr(&umpirespb.Construct{Type: deadlineFixture + "DeadlineState", Args: args})
}

func deadlineLanding(m *umpirespb.Model, function, before, phase string, fact string) {
	f := functionNamed(m, deadlineFixture+"Steps$."+function)
	s := expr(f.GetParams()[0].GetName())
	facts := &umpirespb.ListOf{}
	if fact != "" {
		facts.Items = []*umpirespb.Expr{expr(&umpirespb.Construct{Type: deadlineFixture + "Fact", Case: "timeout", Args: []*umpirespb.Expr{expr(caseOf(deadlineFixture+"TimeoutType", fact))}})}
	}
	result := expr(&umpirespb.ListOf{Items: []*umpirespb.Expr{expr(&umpirespb.Construct{
		Type: interp.StepType,
		Args: []*umpirespb.Expr{
			expr(caseOf(deadlineFixture+"Answer", "ok")),
			expr(&umpirespb.Copy{Base: s, Updates: []*umpirespb.NamedExpr{{Name: "phase", Value: expr(caseOf(deadlineFixture+"Phase", phase))}}}),
			expr(facts),
			expr(&umpirespb.Value{Kind: &umpirespb.Value_Text{Text: ""}}),
		},
	})}})
	guard := binary(umpirespb.Binary_OP_EQ, s, deadlineStateKey(before))
	f.Body = expr(&umpirespb.If{Condition: guard, Then: result, Else: f.GetBody()})
}

func deadlineStateKey(key string) *umpirespb.Expr {
	parts := strings.Split(key, "-")
	flags := make([]bool, len(parts)-1)
	for i, part := range parts[1:] {
		flags[i] = part == "true"
	}
	return deadlineState(parts[0], flags...)
}

func TestDeadlineFixtureMutantsHaveFreshReplayableCounterexamples(t *testing.T) {
	for _, tc := range []struct {
		name, function, before, phase, fact, binding, property string
	}{
		{"unarmed schedule without terminal fact", "schedule", "waiting-false-true-true-false-false", "waiting", "", "schedule", "firesInWindow"},
		{"schedule outside covered role without terminal fact", "schedule", "held-true-true-true-false-false", "waiting", "", "schedule", "firesInWindow"},
		{"disallowed dispatch without terminal fact", "schedule", "waiting-true-false-true-false-false", "waiting", "", "schedule", "firesInWindow"},
		{"unarmed retry without terminal fact", "start", "held-false-true-true-false-false", "waiting", "", "start", "firesInWindow"},
		{"retry outside covered role without terminal fact", "start", "waiting-true-true-true-false-false", "waiting", "", "start", "firesInWindow"},
		{"wrong ordinary retry", "start", "held-true-true-true-false-false", "suspended", "", "start", "deadlineReturnsToWaiting"},
		{"wrong pause retry", "start", "held-true-true-true-true-false", "waiting", "", "start", "deadlinePauses"},
		{"wrong nonretryable landing", "schedule", "waiting-true-true-true-false-false", "waiting", "schedule", "schedule", "deadlineTimesOut"},
		{"extra retry after exhaustion", "start", "held-true-true-false-false-false", "waiting", "", "start", "deadlineTimesOut"},
		{"pause at exhaustion", "start", "held-true-true-false-true-false", "suspended", "", "start", "deadlineTimesOut"},
		{"cancel as ordinary retry", "start", "held-true-true-true-false-true", "waiting", "", "start", "deadlineTimesOut"},
		{"cancel yields to pause", "start", "held-true-true-true-true-true", "suspended", "", "start", "deadlineTimesOut"},
		{"cancel at exhaustion retries", "start", "held-true-true-false-false-true", "waiting", "", "start", "deadlineTimesOut"},
		{"schedule missing typed terminal fact", "schedule", "waiting-true-true-true-false-false", "timedOut", "", "schedule", "deadlineTimesOut"},
		{"schedule wrong typed terminal fact", "schedule", "waiting-true-true-true-false-false", "timedOut", "start", "schedule", "deadlineTimesOut"},
		{"retry exhausted missing terminal fact", "start", "held-true-true-false-false-false", "timedOut", "", "start", "deadlineTimesOut"},
		{"retry exhausted wrong typed terminal fact", "start", "held-true-true-false-false-false", "timedOut", "schedule", "start", "deadlineTimesOut"},
		{"ordinary retry emits bound timeout", "start", "held-true-true-true-false-false", "waiting", "start", "start", "deadlineReturnsToWaiting"},
		{"ordinary retry emits another timeout", "start", "held-true-true-true-false-false", "waiting", "schedule", "start", "deadlineReturnsToWaiting"},
		{"pause retry emits another timeout", "start", "held-true-true-true-true-false", "suspended", "schedule", "start", "deadlinePauses"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := proto.Clone(lifted(t, "deadlineCapabilities")).(*umpirespb.Model)
			deadlineLanding(m, tc.function, tc.before, tc.phase, tc.fact)
			name := "deadlineTimers." + tc.binding + "." + tc.property
			r := receiptOf(t, checked(t, m), "query deadlineTimers "+name)
			require.Equal(t, Counterexample, r.Kind, r.Explanation)
			require.True(t, r.Exercised)
			require.NotNil(t, r.Witness)
			require.NotEmpty(t, r.Witness.Steps)
			require.Contains(t, r.Rows[len(r.Rows)-1], tc.before+"-")
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

func TestDeadlineUnrelatedActionsSkipPredicateAndExercise(t *testing.T) {
	m := proto.Clone(lifted(t, "deadlineCapabilities")).(*umpirespb.Model)
	name := "deadlineTimers.start.firesInWindow"
	p := admProperty(m, "deadlineTimers", name)
	functionNamed(m, p.GetHolds()).Requires = expr(boolValue(false))
	q := admQuery(m, name)
	sc := admScenario(m, "deadlineTimers", q.GetScenario().GetName())
	sc.Free, sc.Actions = false, []*umpirespb.ActionClass{{Action: deadlineFixture + "observer.keep"}}
	m = recounted(t, m)
	r := receiptOf(t, checked(t, m), "query deadlineTimers "+name)
	require.Equal(t, Verified, r.Kind, r.Explanation)
	require.False(t, r.Exercised)
	require.Empty(t, r.Holes)
}
