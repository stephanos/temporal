package lint

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
)

// holesOf reads the hole kinds and tables of the activity IR after the mutations.
func holesOf(t *testing.T, options Options, mutations ...func(*umpirespb.Model)) ([]*Table, map[Kind]map[string][]Finding) {
	t.Helper()
	m := read(t, activityIR, mutations...)
	m.options = options
	tables, tallies, err := m.holes()
	require.NoError(t, err)
	out := map[Kind]map[string][]Finding{}
	for _, x := range tallies {
		require.LessOrEqual(t, len(x.Findings), x.Population)
		if out[x.Kind] == nil {
			out[x.Kind] = map[string][]Finding{}
		}
		out[x.Kind][x.Owner] = append(out[x.Kind][x.Owner], x.Findings...)
	}
	return tables, out
}

func subjectsOf(fs []Finding) []string {
	out := make([]string, len(fs))
	for i, f := range fs {
		out[i] = f.Subject
	}
	return out
}

// rewrite applies a change to every match case of a function's body, depth first.
func rewrite(function string, change func(c *umpirespb.MatchCase) bool, ifs func(x *umpirespb.If) bool) func(*umpirespb.Model) {
	return func(ir *umpirespb.Model) {
		var visit func(x *umpirespb.Expr) bool
		visit = func(x *umpirespb.Expr) bool {
			switch k := x.GetKind().(type) {
			case *umpirespb.Expr_If:
				if ifs != nil && ifs(k.If) {
					return true
				}
				return visit(k.If.GetThen()) || visit(k.If.GetElse())
			case *umpirespb.Expr_Match:
				for _, c := range k.Match.GetCases() {
					if change != nil && change(c) || visit(c.GetBody()) {
						return true
					}
				}
			default:
			}
			return false
		}
		for _, f := range ir.GetFunctions() {
			if f.GetName() == function {
				visit(f.GetBody())
			}
		}
	}
}

// pauseOfCancelRequestedByDefault decides a pause of a cancel-requested activity by a default arm,
// as `s.phase match { case _ => Nil }` lifts, before the pause's rules are tried.
var pauseOfCancelRequestedByDefault = rewrite("activityProtocol.rules.control", func(c *umpirespb.MatchCase) bool {
	if c.GetPattern().GetLiteral().GetEnum().GetCase() != "pause" {
		return false
	}
	phase := func() *umpirespb.Expr {
		return &umpirespb.Expr{Kind: &umpirespb.Expr_Field{Field: &umpirespb.FieldAccess{
			Base: &umpirespb.Expr{Kind: &umpirespb.Expr_Var{Var: "s"}}, Field: "phase"}}}
	}
	cancelRequested := &umpirespb.Expr{Kind: &umpirespb.Expr_Literal{Literal: &umpirespb.Value{Kind: &umpirespb.Value_Enum{
		Enum: &umpirespb.EnumValue{Type: "temporal.standaloneactivity.Phase", Case: "cancelRequested"}}}}}
	none := &umpirespb.Expr{Kind: &umpirespb.Expr_List{List: &umpirespb.ListOf{}}}
	byDefault := &umpirespb.Expr{Position: c.GetBody().GetPosition(), Kind: &umpirespb.Expr_Match{Match: &umpirespb.Match{
		Scrutinee: phase(),
		Cases: []*umpirespb.MatchCase{{
			Pattern: &umpirespb.Pattern{Kind: &umpirespb.Pattern_Wildcard{Wildcard: &umpirespb.Empty{}}},
			Body:    none,
		}},
	}}}
	c.Body = &umpirespb.Expr{Position: c.GetBody().GetPosition(), Kind: &umpirespb.Expr_If{If: &umpirespb.If{
		Condition: &umpirespb.Expr{Kind: &umpirespb.Expr_Binary{Binary: &umpirespb.Binary{
			Op:    umpirespb.Binary_OP_CONTAINS,
			Left:  phase(),
			Right: &umpirespb.Expr{Kind: &umpirespb.Expr_List{List: &umpirespb.ListOf{Items: []*umpirespb.Expr{cancelRequested}}}},
		}}},
		Then: byDefault,
		Else: c.GetBody(),
	}}}
	return true
}, nil)

func TestDisabledByDefault(t *testing.T) {
	_, clean := holesOf(t, Options{})
	require.Empty(t, clean[DisabledByDefault]["activityProtocol"])

	tables, holes := holesOf(t, Options{}, pauseOfCancelRequestedByDefault)
	found := holes[DisabledByDefault]["activityProtocol"]
	require.Equal(t, []string{"control-pause in cancelRequested"}, subjectsOf(found))
	require.Contains(t, found[0].Position, "model/temporal/features/standaloneactivity/StandaloneActivity.scala:")
	require.Contains(t, found[0].Message, "s.phase is _")

	// The default arm shows as `?` in the table, with its hole.
	protocol := tables[slices.IndexFunc(tables, func(t *Table) bool { return t.Machine == "activityProtocol" })]
	i := slices.IndexFunc(protocol.Rules, func(r Rule) bool { return r.Class == "control-pause" && r.Label == "cancelRequested" })
	require.GreaterOrEqual(t, i, 0)
	require.Equal(t, Silent, protocol.Rules[i].Modality)
	require.Contains(t, protocol.Rules[i].Holes, DisabledByDefault)

	// An `if` whose condition names no field of the state decides by default too.
	_, holes = holesOf(t, Options{}, rewrite("activityProtocol.rules.attemptResult", nil, func(x *umpirespb.If) bool {
		phases := x.GetCondition().GetBinary().GetRight().GetList().GetItems()
		if len(phases) != 1 || phases[0].GetLiteral().GetEnum().GetCase() != "cancelRequested" || x.GetElse().GetList() == nil {
			return false
		}
		x.Condition = &umpirespb.Expr{Kind: &umpirespb.Expr_Literal{Literal: &umpirespb.Value{Kind: &umpirespb.Value_Bool{Bool: false}}}}
		return true
	}))
	// The canceled answer's one rule now names nothing of the state, so every phase it is disabled in
	// is decided by default.
	require.Equal(t, []string{"attemptResult-canceled in unstarted, scheduled, backingOff, started, paused, pauseRequested, cancelRequested, completed, failed, canceled, terminated, timedOut"}, subjectsOf(holes[DisabledByDefault]["activityProtocol"]))
}

func TestSilentRejection(t *testing.T) {
	tables, holes := holesOf(t, Options{})
	silent := subjectsOf(holes[SilentRejection]["activityProtocol"])
	// A caller's pause of a paused activity is an RPC the server answers, and the Model says nothing.
	require.Contains(t, silent, "control-pause in unstarted, paused, pauseRequested, cancelRequested")
	// A timer is the system's: disabled, it is prohibited, not silent.
	for _, s := range silent {
		require.False(t, strings.HasPrefix(s, "backoff"), s)
		_, phases, _ := strings.Cut(s, " in ")
		require.NotContains(t, strings.Split(phases, ", "), "completed", "an end state is no silence")
	}
	protocol := tables[slices.IndexFunc(tables, func(t *Table) bool { return t.Machine == "activityProtocol" })]
	i := slices.IndexFunc(protocol.Rules, func(r Rule) bool { return r.Class == "backoff" && r.Modality == MustNot })
	require.GreaterOrEqual(t, i, 0)
	require.Equal(t, "s.phase != backingOff", protocol.Rules[i].Text)
	require.Contains(t, protocol.Rules[i].Position, "model/temporal/features/standaloneactivity/StandaloneActivity.scala:")
}

func TestUnconstrainedResult(t *testing.T) {
	_, holes := holesOf(t, Options{})
	unconstrained := subjectsOf(holes[UnconstrainedResult]["activityProtocol"])
	// The backoff timer's rows land somewhere no claim reads.
	require.Contains(t, unconstrained, "backoff")
	// A terminate is held to `terminated`; a pause from paused is read by the product's law.
	require.NotContains(t, unconstrained, "control-terminate")
	require.NotContains(t, unconstrained, "control-pause")
	// Through the refinement, a transition law of the product reads the protocol's carried rows.
	require.NotContains(t, unconstrained, "control-unpause")
}

func TestWitnessOnly(t *testing.T) {
	_, holes := holesOf(t, Options{})
	require.Contains(t, subjectsOf(holes[WitnessOnly]["activityProtocol"]), "completes")
	// A verify of a composition's Property holds the table to it.
	require.Empty(t, holes[WitnessOnly]["standaloneActivity"])

	_, holes = holesOf(t, Options{}, func(ir *umpirespb.Model) {
		for _, q := range ir.GetQueries() {
			if q.GetProperty().GetName() == "completes" {
				q.Form = umpirespb.Query_FORM_VERIFY
			}
		}
	})
	require.NotContains(t, subjectsOf(holes[WitnessOnly]["activityProtocol"]), "completes")
}

func TestMustNotPinnedIsOffByDefault(t *testing.T) {
	_, holes := holesOf(t, Options{})
	require.NotContains(t, holes, MustNotPinned)

	tables, holes := holesOf(t, Options{MustNotPinned: true})
	pinned := holes[MustNotPinned]["activityProtocol"]
	// An end state's disabled pairs are no hole, in the table as in the findings.
	protocol := tables[slices.IndexFunc(tables, func(t *Table) bool { return t.Machine == "activityProtocol" })]
	for _, c := range protocol.Cells {
		phase, _, _ := strings.Cut(c.State, "-")
		if slices.Contains([]string{"completed", "failed", "canceled", "terminated", "timedOut"}, phase) {
			require.NotContains(t, c.Holes, MustNotPinned, "%s %s", c.State, c.Class)
		}
	}
	i := slices.IndexFunc(pinned, func(f Finding) bool { return strings.HasPrefix(f.Subject, "backoff in ") })
	require.GreaterOrEqual(t, i, 0)
	// From paused, the product's law reads any step, so it pins the disabled backoff there.
	require.NotContains(t, pinned[i].Subject, "paused")
	require.Contains(t, pinned[i].Subject, "scheduled")
}

func TestTablesAreWrittenByMachineAndClass(t *testing.T) {
	m := read(t, activityIR, pauseOfCancelRequestedByDefault)
	tables, _, err := m.holes()
	require.NoError(t, err)
	var out strings.Builder
	require.NoError(t, WriteTables(&out, &Result{File: "activity.json", Tables: tables}))
	text := out.String()
	require.Contains(t, text, "rules activity.json activityProtocol by phase\n")
	require.Contains(t, text, "\n  control-pause\n")
	require.Regexp(t, `\n    scheduled, backingOff +MAY +accepted -> paused \[statusPaused\]`, text)
	require.Regexp(t, `\n    cancelRequested +\? +s\.phase is _ +model/temporal/features/standaloneactivity/StandaloneActivity\.scala:\d+ +disabled-by-default +silent-rejection`, text)
}
