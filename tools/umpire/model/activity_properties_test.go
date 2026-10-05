package model

// Every Property of the standalone activity Model, evaluated on every row of its machine. A Query over
// one of the Scenarios' paths reads a Property on the few steps of that path; a predicate that differs
// on any other row would answer every such Query alike. Here each row is a Scenario of its own, one
// step from the row's state by the row's class, and each Property is verified over it through Check.
// The expected answers are pinned as counts and failing rows, and a counterexample must be the row's
// own step, so an edited predicate cannot alter its own expectation.
//
// What one answer says of a row: whether the Property is about its step (the claim fired), and
// whether it holds there (verified, or a counterexample that is the step). A product machine's
// Properties are also read on every row of the machine that refines it, through the refinement, and
// the composition's Property on every row of the composition, by the row's composed class key.

import (
	"fmt"
	"slices"
	"strings"
	"sync"
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

// rowStep is the one-step witness of a row: its source, and its class taken to its one result.
func rowStep(t *testing.T, table *Table, row Row) *Trace {
	t.Helper()
	require.Len(t, row.Results, 1, row.Key)
	atom := func(keys, ids []string, key string) Atom {
		index := slices.Index(keys, key)
		require.GreaterOrEqual(t, index, 0, key)
		return Atom{ID: ids[index], Value: key}
	}
	ids, result := table.IDs(), row.Results[0]
	step := TraceStep{Action: atom(table.Actions, ids.Actions, row.Action), Outcome: atom(table.Outcomes, ids.Outcomes, result.Outcome),
		State: atom(table.States, ids.States, result.State)}
	for _, fact := range result.Facts {
		step.Facts = append(step.Facts, atom(table.Facts, ids.Facts, fact))
	}
	return &Trace{Initial: atom(table.States, ids.States, row.Source), Steps: []TraceStep{step}}
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
			// Any other kind is no answer about the row, and equals no other answer.
			side.Outcome = Outcome(r.Kind)
		}
		out[key] = side
	}
	require.Len(t, out, len(asked))
	return out
}

// rowDisagreements is every row answer of got that is not want's, in key order.
func rowDisagreements(want, got map[string]rowSide) []string {
	var out []string
	for key, w := range want {
		g, ok := got[key]
		switch {
		case !ok:
			out = append(out, key+": no answer")
		case w.Outcome != g.Outcome || w.About != g.About:
			out = append(out, fmt.Sprintf("%s: %s (about the step: %t), now %s (about the step: %t)", key, w.Outcome, w.About,
				g.Outcome, g.About))
		case !sameTrace(w.Witness, g.Witness):
			out = append(out, key+": the witnesses differ")
		default:
		}
	}
	for key := range got {
		if _, ok := want[key]; !ok {
			out = append(out, key+": no answer before")
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

// propertyTally is how a Property answers the rows of one machine or composition: on how many the
// step is not what it is about, on how many it holds and on how many it fails.
type propertyTally struct{ NotAbout, Holds, Fails int }

// Every Property of the Model is asked on every row of its machine, of the machine that refines it and
// of the composition, and says of each row what is pinned here: how many steps it is about and holds
// on, and on which rows it fails, by the phase of the row's state and the row's class. A row a
// Property fails on is its own counterexample: the witness is the row's one step.
func TestActivityPropertiesOnEveryRow(t *testing.T) {
	m := activityModel(t)
	all := irPropertyRows(t, m)
	machines := built(t, m)
	protocol, product := machines["activityProtocol"].Table, machines["activityProduct"].Table
	composed := composedTable(t, m, "standaloneActivity")
	// Ten Properties of the protocol and three of the product, read on the protocol through the
	// refinement too, and the composition's one.
	require.Len(t, all, 13*len(protocol.Rows)+3*len(product.Rows)+len(composed.Rows))

	tables := map[string]*Table{"activityProtocol": protocol, "activityProduct": product, "standaloneActivity": composed}
	rows := map[string]map[string]Row{}
	for name, table := range tables {
		rows[name] = map[string]Row{}
		for _, row := range table.Rows {
			rows[name][row.Key] = row
		}
	}
	tally := map[string]propertyTally{}
	failing := map[string]int{}
	for key, side := range all {
		claim, at, _ := strings.Cut(key, " at ")
		_, machine, _ := strings.Cut(claim, " on ")
		row, ok := rows[machine][at]
		require.True(t, ok, key)
		counts := tally[claim]
		switch {
		case !side.About:
			counts.NotAbout++
		case side.Outcome == umpire.VerifiedWithinLimits:
			counts.Holds++
		default:
			counts.Fails++
			require.Equal(t, umpire.CounterexampleFound, side.Outcome, key)
			require.True(t, sameTrace(rowStep(t, tables[machine], row), side.Witness), "%s: the counterexample is not the row's step", key)
			phase, _, _ := strings.Cut(row.Source, "-")
			failing[claim+": "+phase+" "+row.Action]++
		}
		tally[claim] = counts
	}
	require.Equal(t, map[string]propertyTally{
		"activityProduct.closedIsRejectedUniformly on activityProduct":  {0, 43, 0},
		"activityProduct.closedIsRejectedUniformly on activityProtocol": {0, 1668, 120},
		"activityProduct.pausedIsNotDispatched on activityProduct":      {0, 43, 0},
		"activityProduct.pausedIsNotDispatched on activityProtocol":     {0, 1788, 0},
		"activityProduct.terminalStatesAreFinal on activityProduct":     {0, 43, 0},
		"activityProduct.terminalStatesAreFinal on activityProtocol":    {0, 1788, 0},
		"activityProtocol.cancelIsRequested on activityProtocol":        {1524, 144, 120},
		"activityProtocol.terminateSettles on activityProtocol":         {1524, 144, 120},
		"cancelRequestedWhileStarted on activityProtocol":               {1524, 144, 120},
		"canceledByWorker on activityProtocol":                          {1764, 24, 0},
		"completes on activityProtocol":                                 {1716, 72, 0},
		"nonRetryableFails on activityProtocol":                         {1716, 72, 0},
		"retryCompletes on activityProtocol":                            {1716, 3, 69},
		"scheduleToStartFires on activityProtocol":                      {1764, 24, 0},
		"startToCloseFires on activityProtocol":                         {1752, 36, 0},
		"startedByPollingWorker on standaloneActivity":                  {2494, 24, 0},
		"terminated on activityProtocol":                                {1524, 144, 120},
	}, tally)
	// A phase holds 24 states, three attempt counts by eight deadline settings. In each of the five
	// closed phases the product's uniform rejection fails on every worker stop and the Properties about
	// a cancel request or a terminate on every such request; retryCompletes fails on all but one
	// completion of each phase that holds an attempt.
	failures := map[string]int{}
	for _, phase := range []string{"canceled", "completed", "failed", "terminated", "timedOut"} {
		failures["activityProduct.closedIsRejectedUniformly on activityProtocol: "+phase+" workerStop"] = 24
		failures["activityProtocol.cancelIsRequested on activityProtocol: "+phase+" control-requestCancel"] = 24
		failures["cancelRequestedWhileStarted on activityProtocol: "+phase+" control-requestCancel"] = 24
		failures["activityProtocol.terminateSettles on activityProtocol: "+phase+" control-terminate"] = 24
		failures["terminated on activityProtocol: "+phase+" control-terminate"] = 24
	}
	for _, phase := range []string{"started", "cancelRequested", "pauseRequested"} {
		failures["retryCompletes on activityProtocol: "+phase+" attemptResult-completed"] = 23
	}
	require.Equal(t, failures, failing)
}

// pathAnswer is what a Query says: its verdict, how much its search explored and expanded, and the
// classes of its witness.
type pathAnswer struct {
	Kind               ReceiptKind
	Explored, Expanded int
	Witness            []string
}

func pathAnswerOf(r Receipt) pathAnswer {
	answer := pathAnswer{Kind: r.Kind, Explored: r.Explored, Expanded: r.Expanded}
	if r.Witness != nil {
		for _, step := range r.Witness.Steps {
			answer.Witness = append(answer.Witness, step.Action.Value)
		}
	}
	return answer
}

// Every Query of the Model answers as pinned: a found one by the path its Scenario places, a verified
// one over its whole search.
func TestActivityQueriesAnswerAlongTheirPaths(t *testing.T) {
	got := map[string]pathAnswer{}
	for _, r := range activityChecked(t).Receipts {
		if r.Subject == QuerySubject {
			if r.Kind == Found {
				require.Len(t, r.Rows, len(r.Witness.Steps), "%s: a row for each step of the witness", receiptKey(r))
			}
			got[r.Key.Name] = pathAnswerOf(r)
		}
	}
	found := func(explored int, witness ...string) pathAnswer {
		return pathAnswer{Kind: Found, Explored: explored, Expanded: explored - 1, Witness: witness}
	}
	verified := func(explored, expanded int) pathAnswer {
		return pathAnswer{Kind: Verified, Explored: explored, Expanded: expanded}
	}
	const start, deadline, scheduleToStart = "start-unset-unset-unset", "start-unset-unset-expires", "start-unset-expires-unset"
	require.Equal(t, map[string]pathAnswer{
		"activityProduct.closedIsRejectedUniformly": verified(10, 10),
		"activityProduct.pausedIsNotDispatched":     verified(10, 10),
		"activityProduct.terminalStatesAreFinal":    verified(10, 10),
		"activityProtocol.cancelIsRequested":        found(4, start, "workerStop", "control-requestCancel"),
		"activityProtocol.terminateSettles":         found(4, start, "workerStop", "control-terminate"),
		"cancel":                                    found(5, start, "attemptStart", "control-requestCancel", "attemptResult-canceled"),
		"cancelRequest":                             found(5, start, "attemptStart", "control-requestCancel", "attemptResult-canceled"),
		"completion":                                found(4, start, "attemptStart", "attemptResult-completed"),
		"nonRetryableFailure":                       found(4, start, "attemptStart", "attemptResult-failed-false"),
		"pauseResume":                               found(6, start, "control-pause", "control-unpause", "attemptStart", "attemptResult-completed"),
		"retry": found(7, start, "attemptStart", "attemptResult-failed-true", "backoff", "attemptStart",
			"attemptResult-completed"),
		"scheduleToStartTimeout":     found(4, scheduleToStart, "workerStop", "scheduleToStart"),
		"startToCloseTimeout":        found(4, deadline, "attemptStart", "startToClose"),
		"terminate":                  found(4, start, "workerStop", "control-terminate"),
		"stoppedWorkerStartsNothing": verified(7, 6),
	}, got)
}

// A Query the activity's capabilities generate is named `<machine>.<law>`; every other is a Scenario's
// path.
func generatedQuery(r Receipt) bool { return strings.Contains(r.Key.Name, ".") }

// activityReport is the Model as lifted, checked once: what a mutant's answers are compared with.
var activityReport = sync.OnceValues(func() (*Report, error) {
	m, err := activityBaseline()
	if err != nil {
		return nil, err
	}
	return Check(m, DefaultScope), nil
})

func activityChecked(t *testing.T) *Report {
	t.Helper()
	r, err := activityReport()
	require.NoError(t, err)
	return r
}

// pathDisagreements is every path Query whose answer through m differs from its answer through the
// Model as lifted, as a list.
func pathDisagreements(t *testing.T, m *umpirespb.Model) []string {
	t.Helper()
	report := checked(t, m)
	var out []string
	for _, expected := range activityChecked(t).Receipts {
		if expected.Subject != QuerySubject || generatedQuery(expected) {
			continue
		}
		got := receiptOf(t, report, receiptKey(expected))
		if got.Kind != expected.Kind || got.Explored != expected.Explored || got.Expanded != expected.Expanded ||
			got.Exercised != expected.Exercised || !slices.Equal(got.Rows, expected.Rows) || !sameTrace(got.Witness, expected.Witness) {
			out = append(out, expected.Key.Name)
		}
	}
	return out
}

// generatedDisagreements is every generated Query that answers m with another verdict than the Model as
// lifted, as a list.
func generatedDisagreements(t *testing.T, m *umpirespb.Model) []string {
	t.Helper()
	report := checked(t, m)
	var out []string
	for _, expected := range activityChecked(t).Receipts {
		if expected.Subject != QuerySubject || !generatedQuery(expected) {
			continue
		}
		if receiptOf(t, report, receiptKey(expected)).Kind != expected.Kind {
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

// A predicate that differs from the lifted one away from the Scenarios' paths answers every path
// Query as the lifted one does, and is told apart by the rows: one mutant for each way a Property is
// declared, a same-step predicate, the class a Property is about, a transition predicate read on its
// own machine and through the refinement, and the composition's Property, by its predicate and by the
// composed action it is about. A generated Query searches its machine freely, so it is no path Query
// and may catch a mutant the paths miss; `generated` names those that do.
func TestActivityPropertyRowsCatchWhatThePathsMiss(t *testing.T) {
	want := irPropertyRows(t, activityModel(t))
	for name, mutant := range map[string]struct {
		mutate    func(t *testing.T, m *umpirespb.Model)
		rows      []string
		generated []string
	}{
		// The cancel request of the Scenario's path is taken at attempt 1.
		"cancelRequestedWhileStarted also wants the first attempt": {
			mutate: func(t *testing.T, m *umpirespb.Model) {
				f := activityFunction(t, m, "activityProtocol.property.cancelRequestedWhileStarted")
				narrowed(f, binary(umpirespb.Binary_OP_EQ, stateField(f, 0, "attempts"), expr(admIntValue(1))))
			},
			rows: []string{"cancelRequestedWhileStarted on activityProtocol at scheduled-0-unset-unset-unset-control-requestCancel"},
		},
		// The one attempt result on the path to a canceled activity is the canceled answer.
		"canceledByWorker is about every attempt result": {
			mutate: func(_ *testing.T, m *umpirespb.Model) {
				admProperty(m, "activityProtocol", "canceledByWorker").When = &umpirespb.Property_WhenAction{WhenAction: "attemptResult"}
			},
			rows: []string{"canceledByWorker on activityProtocol at started-1-unset-unset-unset-attemptResult-completed"},
		},
		// No path that reads this Property terminates the activity. The free search of the product
		// that verifies it does: it terminates a scheduled activity.
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
			generated: []string{"activityProduct.pausedIsNotDispatched"},
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
			require.Equal(t, mutant.generated, generatedDisagreements(t, m))
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
	require.Empty(t, generatedDisagreements(t, m))
	require.Empty(t, rowDisagreements(irPropertyRows(t, activityModel(t)), irPropertyRows(t, m)))
}
