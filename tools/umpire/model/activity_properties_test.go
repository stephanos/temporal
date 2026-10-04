package model

// Every Property of the standalone activity Model, evaluated on every row of its machine on both
// sides. A Query over one of the baseline's paths reads a Property on the few steps of that path; a
// predicate that differs on any other row would answer every such Query alike. Here each row is a
// Scenario of its own, one step from the row's state by the row's class, and each Property is
// verified over it through Check. The expected answers and witnesses come from the immutable
// row and refined-property snapshots, so an edited predicate cannot alter its own expectation.
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
	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpire "go.temporal.io/server/tools/umpire/model/internal/checker"
	"google.golang.org/protobuf/proto"
)

// oneStep bounds a row's Scenario: its one step, and room for every result of the row.
var oneStep = Limits{Name: "oneStep", Steps: 1, Actions: 1, Search: 64}

// rowSide is a Property's answer on one row.
type rowSide struct {
	Outcome Outcome
	// About is whether the Property is about the row's step.
	About   bool
	Witness *Trace
}

func rowKeyOf(property, machine, row string) string {
	return fmt.Sprintf("%s on %s at %s", property, machine, row)
}

// The frozen baseline predates the product's capabilities: it names the product's two laws after the
// vals that declared their calls, where the capability declaration now names each
// `activityProduct.<law>`. The predicates are the laws', so every row answers alike under either name.
var renamedActivityProperties = map[string]string{
	"terminalIsFinal":       "activityProduct.terminalStatesAreFinal",
	"pausedIsNotDispatched": "activityProduct.pausedIsNotDispatched",
}

// The laws the capabilities bring that the baseline never declared, so Go has no answer to compare
// with theirs: a closed activity's uniform rejection, and the protocol's functional laws.
var activityLawsAfterBaseline = []string{
	"activityProduct.closedIsRejectedUniformly",
	"activityProtocol.terminateSettles",
	"activityProtocol.cancelIsRequested",
}

// The Queries the frozen baseline answers that the capabilities retired, each to the generated
// Query that verifies the same Property now. A retired Query verified its law through the protocol
// over one of the baseline's paths; its twin verifies it over a free search of the product, so only
// the verdict carries over, not the counts, rows or witness.
var retiredActivityQueries = map[string]string{
	"query activityProtocol terminalHolds": "query activityProduct activityProduct.terminalStatesAreFinal",
	"query activityProtocol pauseHolds":    "query activityProduct activityProduct.pausedIsNotDispatched",
}

// renamedProperty is a frozen Property's name in the lifted Model.
func renamedProperty(name string) string {
	if renamed, ok := renamedActivityProperties[name]; ok {
		return renamed
	}
	return name
}

// comparedRows is the row answers of the Properties the baseline declares, without those of the laws
// it predates, and how many of those it dropped.
func comparedRows(rows map[string]rowSide) (map[string]rowSide, int) {
	out := map[string]rowSide{}
	for key, side := range rows {
		property, _, _ := strings.Cut(key, " on ")
		if !slices.Contains(activityLawsAfterBaseline, property) {
			out[key] = side
		}
	}
	return out, len(rows) - len(out)
}

func frozenPropertyRows(t *testing.T) map[string]rowSide {
	t.Helper()
	meaning := frozenReaderMeaning(t, "activity")
	tables := map[string]*migrationTable{}
	for _, subject := range meaning.Subjects {
		tables[subject.Name] = subject.Table
	}
	out := map[string]rowSide{}
	for _, property := range meaning.Properties {
		require.Empty(t, property.Error)
		table := tables[property.Owner].Table
		rows := map[string]Row{}
		for _, row := range table.Rows {
			rows[row.Key] = row
		}
		atom := func(keys, ids []string, key string) Atom {
			index := slices.Index(keys, key)
			require.GreaterOrEqual(t, index, 0, key)
			return Atom{ID: ids[index], Value: key}
		}
		for _, answer := range property.Rows {
			require.Empty(t, answer.Error)
			row := rows[answer.Row]
			require.Len(t, row.Results, 1, row.Key)
			side := rowSide{Outcome: umpire.VerifiedWithinLimits, About: answer.About}
			if answer.About && !answer.Holds {
				result := row.Results[answer.Result]
				step := TraceStep{Action: atom(table.Actions, table.IDs.Actions, row.Action),
					Outcome: atom(table.Outcomes, table.IDs.Outcomes, result.Outcome), State: atom(table.States, table.IDs.States, result.State)}
				for _, fact := range result.Facts {
					step.Facts = append(step.Facts, atom(table.Facts, table.IDs.Facts, fact))
				}
				side.Outcome = umpire.CounterexampleFound
				side.Witness = &Trace{Initial: atom(table.States, table.IDs.States, row.Source), Steps: []TraceStep{step}}
			}
			out[rowKeyOf(renamedProperty(property.Name), property.Owner, row.Key)] = side
		}
	}
	var refined []migrationRefinedProperty
	frozenReaderJSON(t, "refined-properties/ir/activity.json", &refined)
	for _, row := range refined {
		require.Empty(t, row.Error)
		out[rowKeyOf(renamedProperty(row.Name), row.Machine, row.Row)] = rowSide{Outcome: row.Outcome, About: row.About, Witness: row.Witness}
	}
	return out
}

// protoValue is a value as the IR writes one, for a Scenario that starts in it or takes it as input.
func protoValue(v Value) *umpirespb.Value {
	fields := func(vs []Value) []*umpirespb.Value {
		var out []*umpirespb.Value
		for _, f := range vs {
			out = append(out, protoValue(f))
		}
		return out
	}
	switch v.Kind {
	case BoolValue:
		return &umpirespb.Value{Kind: &umpirespb.Value_Bool{Bool: v.Bool}}
	case IntValue:
		return &umpirespb.Value{Kind: &umpirespb.Value_Int{Int: v.Int}}
	case TextValue:
		return &umpirespb.Value{Kind: &umpirespb.Value_Text{Text: v.Text}}
	case EnumValue:
		return &umpirespb.Value{Kind: &umpirespb.Value_Enum{Enum: &umpirespb.EnumValue{Type: v.Type, Case: v.Case, Fields: fields(v.Fields)}}}
	case RecordValue:
		return &umpirespb.Value{Kind: &umpirespb.Value_Record{Record: &umpirespb.RecordValue{Type: v.Type, Fields: fields(v.Fields)}}}
	case ListValue:
		return &umpirespb.Value{Kind: &umpirespb.Value_List{List: &umpirespb.ListValue{Items: fields(v.Items)}}}
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
func irPropertyRows(t *testing.T, base *umpirespb.Model) map[string]rowSide {
	t.Helper()
	m := proto.Clone(base).(*umpirespb.Model)
	properties := slices.Clone(m.GetProperties())
	asked := map[string]string{}
	// ask adds a row's Scenario of the machine or composition `name`, and the verifies over it.
	ask := func(name, refined, row string, scenario *umpirespb.Scenario) {
		declared := false
		for _, p := range properties {
			own := p.GetMachine() == name
			if !own && p.GetMachine() != refined {
				continue
			}
			declared = true
			key := rowKeyOf(p.GetName(), name, row)
			asked["query "+name+" "+key] = key
			m.Queries = append(m.Queries, &umpirespb.Query{Name: key, Position: scenario.GetPosition(), Form: umpirespb.Query_FORM_VERIFY,
				Property: &umpirespb.ClaimRef{Machine: p.GetMachine(), Name: p.GetName()},
				Scenario: &umpirespb.ClaimRef{Machine: name, Name: scenario.GetName()}, Through: !own,
				Limits: &umpirespb.Limits{Name: oneStep.Name, Steps: int32(oneStep.Steps), Actions: int32(oneStep.Actions),
					Search: int32(oneStep.Search)}})
		}
		if declared {
			m.Scenarios = append(m.Scenarios, scenario)
		}
	}
	from := func(name string, at *umpirespb.Position, row Row, state Value) *umpirespb.Scenario {
		require.Len(t, row.Results, 1, row.Key)
		return &umpirespb.Scenario{Machine: name, Name: "row." + row.Key, Position: at,
			Start: &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Literal{Literal: protoValue(state)}}}
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
			step := &umpirespb.ActionClass{Action: class.Action.GetId()}
			for _, in := range class.Inputs {
				step.Inputs = append(step.Inputs, protoValue(in))
			}
			scenario := from(name, mm.Decl.GetPosition(), row, state)
			scenario.Actions = []*umpirespb.ActionClass{step}
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
			side.Outcome = Outcome(r.Kind)
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

func sameTrace(a, b *Trace) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Initial == b.Initial && slices.EqualFunc(a.Steps, b.Steps, func(x, y TraceStep) bool {
		return x.Action == y.Action && x.Outcome == y.Outcome && x.State == y.State && slices.Equal(x.Facts, y.Facts)
	})
}

// Every Property the comparison covers says the same of every row on both sides: whether it is about
// the row's step, and whether it holds there. The Properties asked are exactly the compared ones and
// the laws the baseline predates, so one added to the baseline is asked here or fails the inventory.
func TestActivityPropertiesAgreeOnEveryRow(t *testing.T) {
	want := frozenPropertyRows(t)
	all := irPropertyRows(t, activityModel(t))
	got, after := comparedRows(all)
	require.Empty(t, rowDisagreements(want, got))

	machines := built(t, activityModel(t))
	protocol, product := machines["activityProtocol"].Table, machines["activityProduct"].Table
	composed := composedTable(t, activityModel(t), "standaloneActivity")
	require.Len(t, want, 10*len(protocol.Rows)+2*len(product.Rows)+len(composed.Rows))
	// The product's closed rejection on both sides and the protocol's two functional laws.
	require.Equal(t, 3*len(protocol.Rows)+len(product.Rows), after)

	// The rows give every Property both answers to disagree about: steps it is not about, steps it
	// holds on and, for the Properties about one class, steps it fails on. A Property is tallied over
	// the rows of the table its compared Query runs on: the product's laws over the product's, where
	// their generated Queries search, and the laws the baseline predates by the IR's answers, which
	// are the baseline's on every other row.
	over := map[string]*Table{}
	for _, q := range activityModel(t).GetQueries() {
		owner := q.GetScenario().GetMachine()
		table := composed
		if mm := machines[owner]; mm != nil {
			table = mm.Table
		}
		over[q.GetProperty().GetName()] = table
	}
	tally := map[string][3]int{}
	for property, table := range over {
		for _, row := range table.Rows {
			side, ok := all[rowKeyOf(property, table.Machine, row.Key)]
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
	require.Len(t, tally, 11+len(activityLawsAfterBaseline))
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

// pathDisagreements is every compared path Query whose answer through a lifted Model differs from
// its frozen receipt, as a list. A retired Query is no path Query any more; twinDisagreements
// compares its verdict.
func pathDisagreements(t *testing.T, m *umpirespb.Model) []string {
	t.Helper()
	report := checked(t, m)
	var out []string
	for _, expected := range frozenReaderMeaning(t, "activity").Receipts {
		if expected.Subject != QuerySubject {
			continue
		}
		if _, retired := retiredActivityQueries[receiptKey(expected.Receipt)]; retired {
			continue
		}
		got := receiptOf(t, report, receiptKey(expected.Receipt))
		if got.Kind != expected.Kind || got.Explored != expected.Explored || got.Expanded != expected.Expanded ||
			got.Exercised != expected.Exercised || !slices.Equal(got.Rows, expected.Rows) || !sameTrace(got.Witness, expected.Witness) {
			out = append(out, expected.Key.Name)
		}
	}
	return out
}

// twinDisagreements is every retired Query whose generated twin answers a lifted Model with another
// verdict than the frozen receipt, as a list.
func twinDisagreements(t *testing.T, m *umpirespb.Model) []string {
	t.Helper()
	report := checked(t, m)
	var out []string
	for _, expected := range frozenReaderMeaning(t, "activity").Receipts {
		twin, retired := retiredActivityQueries[receiptKey(expected.Receipt)]
		if expected.Subject != QuerySubject || !retired {
			continue
		}
		if receiptOf(t, report, twin).Kind != expected.Kind {
			out = append(out, expected.Key.Name)
		}
	}
	slices.Sort(out)
	return out
}

func activityFunction(t *testing.T, m *umpirespb.Model, name string) *umpirespb.Function {
	t.Helper()
	f := functionNamed(m, name)
	require.NotNil(t, f, name)
	return f
}

// narrowed is a predicate that also asks `extra`.
func narrowed(f *umpirespb.Function, extra *umpirespb.Expr) {
	f.Body = &umpirespb.Expr{Position: f.GetBody().GetPosition(),
		Kind: &umpirespb.Expr_Binary{Binary: &umpirespb.Binary{Op: umpirespb.Binary_OP_AND, Left: f.GetBody(), Right: extra}}}
}

// stateField reads a field of the state of the step a predicate's parameter names.
func stateField(f *umpirespb.Function, param int, name string) *umpirespb.Expr {
	return field(field(expr(f.GetParams()[param].GetName()), "state"), name)
}

// A lifted predicate that differs from Go's away from the baseline's paths answers every path Query
// as Go does, and is told apart by the rows: one mutant for each way a Property is declared, a
// same-step predicate, the class a Property is about, a transition predicate read on its own
// machine and through the refinement, and the composition's Property, by its predicate and by the
// composed action it is about. A retired Query's generated twin searches its machine freely, so it
// is no path Query and may catch a mutant the paths miss; `twins` names the retired Queries whose
// twin does.
func TestActivityPropertyRowsCatchWhatThePathsMiss(t *testing.T) {
	want := frozenPropertyRows(t)
	for name, mutant := range map[string]struct {
		mutate func(t *testing.T, m *umpirespb.Model)
		rows   []string
		twins  []string
	}{
		// The review's example: the cancel request of the baseline's path is taken at attempt 1.
		"cancelRequestedWhileStarted also wants the first attempt": {
			mutate: func(t *testing.T, m *umpirespb.Model) {
				f := activityFunction(t, m, "activityProtocol.property.cancelRequestedWhileStarted")
				narrowed(f, binary(umpirespb.Binary_OP_EQ, stateField(f, 0, "attempts"), expr(admIntValue(1))))
			},
			rows: []string{"cancelRequestedWhileStarted on activityProtocol at scheduled-0-unset-unset-unset-control-requestCancel"},
		},
		// The one attempt result on the baseline's path to a canceled activity is the canceled answer.
		"canceledByWorker is about every attempt result": {
			mutate: func(_ *testing.T, m *umpirespb.Model) {
				admProperty(m, "activityProtocol", "canceledByWorker").When = &umpirespb.Property_WhenAction{WhenAction: "attemptResult"}
			},
			rows: []string{"canceledByWorker on activityProtocol at started-1-unset-unset-unset-attemptResult-completed"},
		},
		// No path of the baseline that reads this Property terminates the activity. The free search of
		// the product that retired pauseHolds does: it terminates a scheduled activity.
		"pausedIsNotDispatched also forbids a terminate": {
			mutate: func(t *testing.T, m *umpirespb.Model) {
				f := activityFunction(t, m, "activityProduct.property.activityProduct.pausedIsNotDispatched")
				narrowed(f, binary(umpirespb.Binary_OP_NE, stateField(f, 1, "phase"),
					expr(admEnum("temporal.standaloneactivity.ProductPhase", "terminated"))))
			},
			rows: []string{
				"activityProduct.pausedIsNotDispatched on activityProduct at scheduled-control-terminate",
				"activityProduct.pausedIsNotDispatched on activityProtocol at scheduled-0-unset-unset-unset-control-terminate",
			},
			twins: []string{"pauseHolds"},
		},
		// The one attempt start on the cross-entity path is the first attempt's.
		"startedByPollingWorker also wants the first attempt": {
			mutate: func(t *testing.T, m *umpirespb.Model) {
				f := activityFunction(t, m, "standaloneActivity.property.startedByPollingWorker")
				narrowed(f, binary(umpirespb.Binary_OP_EQ, field(stateField(f, 0, "activity"), "attempts"), expr(admIntValue(1))))
			},
			rows: []string{"startedByPollingWorker on standaloneActivity at scheduled-1-unset-unset-unset_polling-attemptStart"},
		},
		// The backoff on the cross-entity path is taken while the worker polls, so a claim about the
		// activity's own backoff holds there too, and is read on a step of the path.
		"startedByPollingWorker is about the backoff": {
			mutate: func(_ *testing.T, m *umpirespb.Model) {
				admProperty(m, "standaloneActivity", "startedByPollingWorker").When = &umpirespb.Property_WhenAction{WhenAction: "activity_backoff"}
			},
			rows: []string{
				"startedByPollingWorker on standaloneActivity at scheduled-0-unset-unset-unset_polling-attemptStart",
				"startedByPollingWorker on standaloneActivity at backingOff-1-unset-unset-unset_stopped-activity_backoff",
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			m := proto.Clone(activityModel(t)).(*umpirespb.Model)
			mutant.mutate(t, m)
			require.NoError(t, Validate(m))
			require.Empty(t, pathDisagreements(t, m), "the path Queries tell the mutant apart on their own")
			require.Equal(t, mutant.twins, twinDisagreements(t, m))
			rows, _ := comparedRows(irPropertyRows(t, m))
			differing := rowDisagreements(want, rows)
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
	m := proto.Clone(activityModel(t)).(*umpirespb.Model)
	var attemptStart string
	for _, a := range m.GetActions() {
		if a.GetName() == "attemptStart" {
			attemptStart = a.GetId()
		}
	}
	require.NotEmpty(t, attemptStart)
	admProperty(m, "standaloneActivity", "startedByPollingWorker").When = &umpirespb.Property_WhenClass{
		WhenClass: &umpirespb.ActionClass{Action: attemptStart}}
	require.NoError(t, Validate(m))
	require.Empty(t, pathDisagreements(t, m))
	require.Empty(t, twinDisagreements(t, m))
	rows, _ := comparedRows(irPropertyRows(t, m))
	require.Empty(t, rowDisagreements(frozenPropertyRows(t), rows))
}
