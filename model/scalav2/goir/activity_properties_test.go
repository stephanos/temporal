package goir

// Every Property of the standalone activity Model, evaluated on every row of its machine on both
// sides. A Query over one of the baseline's paths reads a Property on the few steps of that path; a
// predicate that differs on any other row would answer every such Query alike. Here each row is a
// Scenario of its own, one step from the row's state by the row's class, and each Property is
// verified over it: Go answers through umpire's search and the IR through Check, so neither side's
// predicate is read by anything but the checker that owns it.
//
// What one answer says of a row: whether the Property is about its step (the claim fired), and
// whether it holds there (verified, or a counterexample that is the step). A product machine's
// Properties are also read on every row of the machine that refines it, through the refinement, and
// the composition's Property on every row of the composition, by the row's composed class key.

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	"go.temporal.io/server/model/go/standaloneactivity"
	"go.temporal.io/server/model/go/umpire"
	"google.golang.org/protobuf/proto"
)

// oneStep bounds a row's Scenario: its one step, and room for every result of the row.
var oneStep = umpire.Limits{Name: "oneStep", Steps: 1, Actions: 1, Search: 64}

// rowSide is a Property's answer on one row.
type rowSide struct {
	Outcome umpire.Outcome
	// About is whether the Property is about the row's step.
	About   bool
	Witness *umpire.Trace
}

func rowKeyOf(property, machine, row string) string {
	return fmt.Sprintf("%s on %s at %s", property, machine, row)
}

// goRowSides answers, for every row of a Go machine, the Queries `over` declares on the row's
// one-step Scenario. Every row of the baseline has one result, so an answer over a row is an answer
// about that result; a row with several would need a Scenario per result, and fails here.
func goRowSides[S, O, F any](t *testing.T, m *umpire.Machine[S, O, F], over func(*umpire.Scenario[S]) []*umpire.Query) map[string]rowSide {
	t.Helper()
	table := goTable(t, m)
	return goTableRowSides(t, table, over, func(row umpire.Row, state S) *umpire.Scenario[S] {
		class, ok := table.Class(row.Action)
		require.True(t, ok, row.Key)
		return m.Scenario("row." + row.Key).Starts(state).Actions(class)
	})
}

// goComposedRowSides is goRowSides for a composition, whose row is scheduled by its composed class key.
func goComposedRowSides[S any](t *testing.T, c *umpire.Composition[S], over func(*umpire.Scenario[S]) []*umpire.Query) map[string]rowSide {
	t.Helper()
	return goTableRowSides(t, goTable(t, c), over, func(row umpire.Row, state S) *umpire.Scenario[S] {
		return c.Scenario("row." + row.Key).Starts(state).ActionKeys(row.Action)
	})
}

func goTableRowSides[S any](t *testing.T, table *umpire.Table, over func(*umpire.Scenario[S]) []*umpire.Query,
	oneRow func(row umpire.Row, state S) *umpire.Scenario[S]) map[string]rowSide {
	t.Helper()
	out := map[string]rowSide{}
	for _, row := range table.Rows {
		require.Len(t, row.Results, 1, row.Key)
		state, ok := table.StateValue(row.Source)
		require.True(t, ok, row.Key)
		typed, ok := state.(S)
		require.True(t, ok, row.Key)
		scenario := oneRow(row, typed)
		for _, q := range over(scenario) {
			answer, err := q.Answer()
			require.NoError(t, err, "%s at %s", q.Property.Name, row.Key)
			out[rowKeyOf(q.Property.Name, table.Machine, row.Key)] = rowSide{Outcome: answer.Outcome, About: answer.Exercised,
				Witness: answer.Witness}
		}
	}
	return out
}

// goPropertyRows is every compared Property of the Go baseline on every row: the protocol machine's
// own, the product machine's own, the product machine's read through the protocol's refinement, and
// the composition's own.
func goPropertyRows(t *testing.T) map[string]rowSide {
	t.Helper()
	via := standaloneactivity.ActivityProtocol.Via(standaloneactivity.ActivityProduct)
	protocol := []*umpire.Property[standaloneactivity.ProtocolState]{standaloneactivity.Completes,
		standaloneactivity.NonRetryableFails, standaloneactivity.RetryCompletes, standaloneactivity.CancelRequestedWhileStarted,
		standaloneactivity.CanceledByWorker, standaloneactivity.TerminatedClaim, standaloneactivity.ScheduleToStartFires,
		standaloneactivity.StartToCloseFires}
	product := []*umpire.Property[standaloneactivity.ProductState]{standaloneactivity.TerminalIsFinal,
		standaloneactivity.PausedIsNotDispatched}
	out := goRowSides(t, standaloneactivity.ActivityProtocol, func(s *umpire.Scenario[standaloneactivity.ProtocolState]) []*umpire.Query {
		var queries []*umpire.Query
		for _, p := range protocol {
			queries = append(queries, s.Verify("row", p, oneStep))
		}
		for _, p := range product {
			queries = append(queries, s.VerifyRefined("row", p, via, oneStep))
		}
		return queries
	})
	for key, side := range goRowSides(t, standaloneactivity.ActivityProduct, func(s *umpire.Scenario[standaloneactivity.ProductState]) []*umpire.Query {
		var queries []*umpire.Query
		for _, p := range product {
			queries = append(queries, s.Verify("row", p, oneStep))
		}
		return queries
	}) {
		out[key] = side
	}
	for key, side := range goComposedRowSides(t, standaloneactivity.StandaloneActivity, func(s *umpire.Scenario[standaloneactivity.StandaloneActivityState]) []*umpire.Query {
		return []*umpire.Query{s.Verify("row", standaloneactivity.StartedByPollingWorker, oneStep)}
	}) {
		out[key] = side
	}
	return out
}

// protoValue is a value as the IR writes one, for a Scenario that starts in it or takes it as input.
func protoValue(v Value) *modelirspb.Value {
	fields := func(vs []Value) []*modelirspb.Value {
		var out []*modelirspb.Value
		for _, f := range vs {
			out = append(out, protoValue(f))
		}
		return out
	}
	switch v.Kind {
	case BoolValue:
		return &modelirspb.Value{Kind: &modelirspb.Value_Bool{Bool: v.Bool}}
	case IntValue:
		return &modelirspb.Value{Kind: &modelirspb.Value_Int{Int: v.Int}}
	case TextValue:
		return &modelirspb.Value{Kind: &modelirspb.Value_Text{Text: v.Text}}
	case EnumValue:
		return &modelirspb.Value{Kind: &modelirspb.Value_Enum{Enum: &modelirspb.EnumValue{Type: v.Type, Case: v.Case, Fields: fields(v.Fields)}}}
	case RecordValue:
		return &modelirspb.Value{Kind: &modelirspb.Value_Record{Record: &modelirspb.RecordValue{Type: v.Type, Fields: fields(v.Fields)}}}
	case ListValue:
		return &modelirspb.Value{Kind: &modelirspb.Value_List{List: &modelirspb.ListValue{Items: fields(v.Items)}}}
	case LambdaValue:
		panic("a function is no state and no input")
	default:
		panic("no such kind of value")
	}
}

// irPropertyRows is every Property of a lifted Model on every row, answered by Check. To a copy of
// the Model it adds, for every row of every machine and of every composition, the row's one-step
// Scenario, and a verify of each Property of the machine or composition and of each Property of the
// machine it refines, read through that refinement. It names no machine and no Property: whatever the
// Model declares is what is asked.
func irPropertyRows(t *testing.T, base *modelirspb.Model) map[string]rowSide {
	t.Helper()
	m := proto.Clone(base).(*modelirspb.Model)
	properties := slices.Clone(m.GetProperties())
	asked := map[string]string{}
	// ask adds a row's Scenario of the machine or composition `name`, and the verifies over it.
	ask := func(name, refined, row string, scenario *modelirspb.Scenario) {
		declared := false
		for _, p := range properties {
			own := p.GetMachine() == name
			if !own && p.GetMachine() != refined {
				continue
			}
			declared = true
			key := rowKeyOf(p.GetName(), name, row)
			asked["query "+name+" "+key] = key
			m.Queries = append(m.Queries, &modelirspb.Query{Name: key, Position: scenario.GetPosition(), Form: modelirspb.Query_FORM_VERIFY,
				Property: &modelirspb.ClaimRef{Machine: p.GetMachine(), Name: p.GetName()},
				Scenario: &modelirspb.ClaimRef{Machine: name, Name: scenario.GetName()}, Through: !own,
				Limits: &modelirspb.Limits{Name: oneStep.Name, Steps: int32(oneStep.Steps), Actions: int32(oneStep.Actions),
					Search: int32(oneStep.Search)}})
		}
		if declared {
			m.Scenarios = append(m.Scenarios, scenario)
		}
	}
	from := func(name string, at *modelirspb.Position, row umpire.Row, state Value) *modelirspb.Scenario {
		require.Len(t, row.Results, 1, row.Key)
		return &modelirspb.Scenario{Machine: name, Name: "row." + row.Key, Position: at,
			Start: &modelirspb.Expr{Position: at, Kind: &modelirspb.Expr_Literal{Literal: protoValue(state)}}}
	}
	for name, mm := range built(t, m) {
		classes := map[string]Class{}
		for _, c := range mm.Classes {
			classes[c.Key] = c
		}
		for _, row := range mm.Table.Rows {
			state, ok := mm.State(row.Source)
			require.True(t, ok, row.Key)
			class := classes[row.Action]
			step := &modelirspb.ActionClass{Action: class.Action.GetId()}
			for _, in := range class.Inputs {
				step.Inputs = append(step.Inputs, protoValue(in))
			}
			scenario := from(name, mm.Decl.GetPosition(), row, state)
			scenario.Actions = []*modelirspb.ActionClass{step}
			ask(name, mm.Decl.GetRefines().GetProduct(), row.Key, scenario)
		}
	}
	composed := bind(base, DefaultScope)
	for _, c := range m.GetCompositions() {
		s := composed.subject(c.GetName())
		require.NoError(t, s.err)
		for _, row := range s.table.Rows {
			state, err := s.state(row.Source)
			require.NoError(t, err, row.Key)
			scenario := from(c.GetName(), c.GetPosition(), row, state)
			scenario.Keys = []string{row.Action}
			ask(c.GetName(), "", row.Key, scenario)
		}
	}
	out := map[string]rowSide{}
	for _, r := range Check(m, DefaultScope).Receipts {
		key, ok := asked[receiptKey(r)]
		if !ok {
			continue
		}
		side := rowSide{About: r.Exercised, Witness: r.Witness}
		switch r.Kind {
		case Verified:
			side.Outcome = umpire.VerifiedWithinLimits
		case Counterexample:
			side.Outcome = umpire.CounterexampleFound
		default:
			// Any other kind is no answer about the row, and equals no answer of Go's.
			side.Outcome = umpire.Outcome(r.Kind)
		}
		out[key] = side
	}
	require.Len(t, out, len(asked))
	return out
}

// rowDisagreements is every row answer the two sides do not share, in key order.
func rowDisagreements(want, got map[string]rowSide) []string {
	var out []string
	for key, w := range want {
		g, ok := got[key]
		switch {
		case !ok:
			out = append(out, key+": the IR has no answer")
		case w.Outcome != g.Outcome || w.About != g.About:
			out = append(out, fmt.Sprintf("%s: Go %s (about the step: %t), the IR %s (about the step: %t)", key, w.Outcome, w.About,
				g.Outcome, g.About))
		case !sameTrace(w.Witness, g.Witness):
			out = append(out, key+": the witnesses differ")
		default:
		}
	}
	for key := range got {
		if _, ok := want[key]; !ok {
			out = append(out, key+": Go has no answer")
		}
	}
	slices.Sort(out)
	return out
}

func sameTrace(a, b *umpire.Trace) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Initial == b.Initial && slices.EqualFunc(a.Steps, b.Steps, func(x, y umpire.TraceStep) bool {
		return x.Action == y.Action && x.Outcome == y.Outcome && x.State == y.State && slices.Equal(x.Facts, y.Facts)
	})
}

// Every Property the comparison covers says the same of every row on both sides: whether it is about
// the row's step, and whether it holds there. The Properties asked are exactly the compared ones, so
// one added to the baseline is asked here or fails the inventory.
func TestActivityPropertiesAgreeOnEveryRow(t *testing.T) {
	want := goPropertyRows(t)
	got := irPropertyRows(t, activityModel(t))
	require.Empty(t, rowDisagreements(want, got))

	protocol, product := goTable(t, standaloneactivity.ActivityProtocol), goTable(t, standaloneactivity.ActivityProduct)
	composed := goTable(t, standaloneactivity.StandaloneActivity)
	require.Len(t, want, 10*len(protocol.Rows)+2*len(product.Rows)+len(composed.Rows))

	// The rows give every Property both answers to disagree about: steps it is not about, steps it
	// holds on and, for the Properties about one class, steps it fails on. A Property is tallied over
	// the rows of the table its compared Query runs on.
	over := map[string]*umpire.Table{}
	for _, q := range comparedQueries() {
		over[q.Property.Name] = goTable(t, q.Scenario.Machine)
	}
	tally := map[string][3]int{}
	for property, table := range over {
		for _, row := range table.Rows {
			side, ok := want[rowKeyOf(property, table.Machine, row.Key)]
			require.True(t, ok, "%s at %s", property, row.Key)
			counts := tally[property]
			switch {
			case !side.About:
				counts[0]++
			case side.Outcome == umpire.VerifiedWithinLimits:
				counts[1]++
			default:
				counts[2]++
			}
			tally[property] = counts
		}
	}
	require.Len(t, tally, 11)
	failing := 0
	for property, counts := range tally {
		table := over[property]
		require.Equal(t, len(table.Rows), counts[0]+counts[1]+counts[2], property)
		require.Positive(t, counts[1], "%s holds on no row", property)
		if counts[2] > 0 {
			failing++
		}
		t.Logf("%s on %s: not about %d rows, holds on %d, fails on %d", property, table.Machine, counts[0], counts[1], counts[2])
	}
	require.Positive(t, failing)
	require.Equal(t, composed, over["startedByPollingWorker"])
}

// pathDisagreements is every compared Query whose answer through a lifted Model is not Go's: what
// TestActivityClaimsEqualTheGoModel compares, as a list.
func pathDisagreements(t *testing.T, m *modelirspb.Model) []string {
	t.Helper()
	report := checked(t, m)
	var out []string
	for _, q := range comparedQueries() {
		answer, err := q.Answer()
		require.NoError(t, err)
		got := receiptOf(t, report, queryReceipt(q))
		if string(got.Kind) != string(answer.Outcome) || got.Explored != answer.Explored || got.Expanded != answer.Expanded ||
			got.Exercised != answer.Exercised || !slices.Equal(got.Rows, answer.Rows) || !sameTrace(got.Witness, answer.Witness) {
			out = append(out, q.Name)
		}
	}
	return out
}

func activityFunction(t *testing.T, m *modelirspb.Model, name string) *modelirspb.Function {
	t.Helper()
	f := functionNamed(m, name)
	require.NotNil(t, f, name)
	return f
}

// narrowed is a predicate that also asks `extra`.
func narrowed(f *modelirspb.Function, extra *modelirspb.Expr) {
	f.Body = &modelirspb.Expr{Position: f.GetBody().GetPosition(),
		Kind: &modelirspb.Expr_Binary{Binary: &modelirspb.Binary{Op: modelirspb.Binary_OP_AND, Left: f.GetBody(), Right: extra}}}
}

// stateField reads a field of the state of the step a predicate's parameter names.
func stateField(f *modelirspb.Function, param int, name string) *modelirspb.Expr {
	return field(field(expr(f.GetParams()[param].GetName()), "state"), name)
}

// A lifted predicate that differs from Go's away from the baseline's paths answers every path Query
// as Go does, and is told apart by the rows: one mutant for each way a Property is declared, a
// same-step predicate, the class a Property is about, a transition predicate read on its own
// machine and through the refinement, and the composition's Property, by its predicate and by the
// composed action it is about.
func TestActivityPropertyRowsCatchWhatThePathsMiss(t *testing.T) {
	want := goPropertyRows(t)
	for name, mutant := range map[string]struct {
		mutate func(t *testing.T, m *modelirspb.Model)
		rows   []string
	}{
		// The review's example: the cancel request of the baseline's path is taken at attempt 1.
		"cancelRequestedWhileStarted also wants the first attempt": {
			mutate: func(t *testing.T, m *modelirspb.Model) {
				f := activityFunction(t, m, "activityProtocol.property.cancelRequestedWhileStarted")
				narrowed(f, binary(modelirspb.Binary_OP_EQ, stateField(f, 0, "attempts"), expr(admIntValue(1))))
			},
			rows: []string{"cancelRequestedWhileStarted on activityProtocol at scheduled-0-unset-unset-unset-control-requestCancel"},
		},
		// The one attempt result on the baseline's path to a canceled activity is the canceled answer.
		"canceledByWorker is about every attempt result": {
			mutate: func(_ *testing.T, m *modelirspb.Model) {
				admProperty(m, "activityProtocol", "canceledByWorker").When = &modelirspb.Property_WhenAction{WhenAction: "attemptResult"}
			},
			rows: []string{"canceledByWorker on activityProtocol at started-1-unset-unset-unset-attemptResult-completed"},
		},
		// No path of the baseline that reads this Property terminates the activity.
		"pausedIsNotDispatched also forbids a terminate": {
			mutate: func(t *testing.T, m *modelirspb.Model) {
				f := activityFunction(t, m, "activityProduct.property.pausedIsNotDispatched")
				narrowed(f, binary(modelirspb.Binary_OP_NE, stateField(f, 1, "phase"),
					expr(admEnum("temporal.standaloneactivity.ProductPhase", "terminated"))))
			},
			rows: []string{
				"pausedIsNotDispatched on activityProduct at scheduled-control-terminate",
				"pausedIsNotDispatched on activityProtocol at scheduled-0-unset-unset-unset-control-terminate",
			},
		},
		// The one attempt start on the cross-entity path is the first attempt's.
		"startedByPollingWorker also wants the first attempt": {
			mutate: func(t *testing.T, m *modelirspb.Model) {
				f := activityFunction(t, m, "standaloneActivity.property.startedByPollingWorker")
				narrowed(f, binary(modelirspb.Binary_OP_EQ, field(stateField(f, 0, "activity"), "attempts"), expr(admIntValue(1))))
			},
			rows: []string{"startedByPollingWorker on standaloneActivity at scheduled-1-unset-unset-unset_polling-attemptStart"},
		},
		// The backoff on the cross-entity path is taken while the worker polls, so a claim about the
		// activity's own backoff holds there too, and is read on a step of the path.
		"startedByPollingWorker is about the backoff": {
			mutate: func(_ *testing.T, m *modelirspb.Model) {
				admProperty(m, "standaloneActivity", "startedByPollingWorker").When = &modelirspb.Property_WhenAction{WhenAction: "activity_backoff"}
			},
			rows: []string{
				"startedByPollingWorker on standaloneActivity at scheduled-0-unset-unset-unset_polling-attemptStart",
				"startedByPollingWorker on standaloneActivity at backingOff-1-unset-unset-unset_stopped-activity_backoff",
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			m := proto.Clone(activityModel(t)).(*modelirspb.Model)
			mutant.mutate(t, m)
			require.NoError(t, Validate(m))
			require.Empty(t, pathDisagreements(t, m), "the path Queries tell the mutant apart on their own")
			differing := rowDisagreements(want, irPropertyRows(t, m))
			require.NotEmpty(t, differing)
			for _, row := range mutant.rows {
				require.True(t, slices.ContainsFunc(differing, func(d string) bool { return strings.HasPrefix(d, row+":") }),
					"%s is not among the %d rows that differ", row, len(differing))
			}
		})
	}
}

// The sync attemptStart is named as the activity's action it takes, so that action's one class is
// keyed as the composed step is. The cross-entity Property declared about that class answers every
// row and every path as the one declared about the action does.
func TestActivityCrossEntityClaimByClassAnswersAlike(t *testing.T) {
	m := proto.Clone(activityModel(t)).(*modelirspb.Model)
	var attemptStart string
	for _, a := range m.GetActions() {
		if a.GetName() == "attemptStart" {
			attemptStart = a.GetId()
		}
	}
	require.NotEmpty(t, attemptStart)
	admProperty(m, "standaloneActivity", "startedByPollingWorker").When = &modelirspb.Property_WhenClass{
		WhenClass: &modelirspb.ActionClass{Action: attemptStart}}
	require.NoError(t, Validate(m))
	require.Empty(t, pathDisagreements(t, m))
	require.Empty(t, rowDisagreements(goPropertyRows(t), irPropertyRows(t, m)))
}
