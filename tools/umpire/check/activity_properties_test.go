package check

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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpire "go.temporal.io/server/tools/umpire/internal/engine"
	"go.temporal.io/server/tools/umpire/interp"
	"go.temporal.io/server/tools/umpire/ir"
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
func rowStep(t *testing.T, table *interp.Table, row interp.Row) *Trace {
	t.Helper()
	require.Len(t, row.Results, 1, row.Key)
	atom := func(keys, ids []string, key string) interp.Atom {
		index := slices.Index(keys, key)
		require.GreaterOrEqual(t, index, 0, key)
		return interp.Atom{ID: ids[index], Value: key}
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
func protoValue(v interp.Value) *umpirespb.Value {
	fields := func(vs []interp.Value) []*umpirespb.Value {
		var out []*umpirespb.Value
		for _, f := range vs {
			out = append(out, protoValue(f))
		}
		return out
	}
	switch v.Kind {
	case interp.BoolValue:
		return &umpirespb.Value{Kind: &umpirespb.Value_Bool{Bool: v.Bool}}
	case interp.IntValue:
		return &umpirespb.Value{Kind: &umpirespb.Value_Int{Int: v.Int}}
	case interp.TextValue:
		return &umpirespb.Value{Kind: &umpirespb.Value_Text{Text: v.Text}}
	case interp.EnumValue:
		return &umpirespb.Value{Kind: &umpirespb.Value_Enum{Enum: &umpirespb.EnumValue{Type: v.Type, Case: v.Case, Fields: fields(v.Fields)}}}
	case interp.RecordValue:
		return &umpirespb.Value{Kind: &umpirespb.Value_Record{Record: &umpirespb.RecordValue{Type: v.Type, Fields: fields(v.Fields)}}}
	case interp.ListValue:
		return &umpirespb.Value{Kind: &umpirespb.Value_List{List: &umpirespb.ListValue{Items: fields(v.Items)}}}
	case interp.LambdaValue:
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
	from := func(name string, at *umpirespb.Position, row interp.Row, state interp.Value) *umpirespb.Scenario {
		require.Len(t, row.Results, 1, row.Key)
		return &umpirespb.Scenario{Machine: name, Name: "row." + row.Key, Position: at,
			Start: &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Literal{Literal: protoValue(state)}}}
	}
	for name, mm := range built(t, m) {
		classes := map[string]interp.Class{}
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
	protocol, product := machines["activitySystem"].Table, machines["activityProduct"].Table
	composed := composedTable(t, m, "standaloneActivity")
	// Ten Properties of the protocol and three of the product, read on the protocol through the
	// refinement too, and the composition's one.
	require.Len(t, all, 13*len(protocol.Rows)+3*len(product.Rows)+len(composed.Rows))

	tables := map[string]*interp.Table{"activitySystem": protocol, "activityProduct": product, "standaloneActivity": composed}
	rows := map[string]map[string]interp.Row{}
	for name, table := range tables {
		rows[name] = map[string]interp.Row{}
		for _, row := range table.Rows {
			rows[name][row.Key] = row
		}
	}
	tally := map[string]propertyTally{}
	failing := map[string]int{}
	answers := map[string][]string{}
	for key, side := range all {
		claim, at, _ := strings.Cut(key, " at ")
		_, machine, _ := strings.Cut(claim, " on ")
		row, ok := rows[machine][at]
		require.True(t, ok, key)
		answers[claim] = append(answers[claim], fmt.Sprintf("%s: %t %s", at, side.About, side.Outcome))
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
		"activityProduct.closedIsRejectedUniformly on activityProduct": {0, 43, 0},
		"activityProduct.closedIsRejectedUniformly on activitySystem":  {0, 1668, 120},
		"activityProduct.pausedIsNotDispatched on activityProduct":     {0, 43, 0},
		"activityProduct.pausedIsNotDispatched on activitySystem":      {0, 1788, 0},
		"activityProduct.terminalStatesAreFinal on activityProduct":    {0, 43, 0},
		"activityProduct.terminalStatesAreFinal on activitySystem":     {0, 1788, 0},
		"activitySystem.cancelIsRequested on activitySystem":           {1524, 144, 120},
		"activitySystem.terminateSettles on activitySystem":            {1524, 144, 120},
		"cancelRequestedWhileStarted on activitySystem":                {1524, 144, 120},
		"canceledByWorker on activitySystem":                           {1764, 24, 0},
		"completes on activitySystem":                                  {1716, 72, 0},
		"nonRetryableFails on activitySystem":                          {1716, 72, 0},
		"retryCompletes on activitySystem":                             {1716, 3, 69},
		"scheduleToStartFires on activitySystem":                       {1764, 24, 0},
		"startToCloseFires on activitySystem":                          {1752, 36, 0},
		"startedByPollingWorker on standaloneActivity":                 {2494, 24, 0},
		"terminated on activitySystem":                                 {1524, 144, 120},
	}, tally)
	// A phase holds 24 states, three attempt counts by eight deadline settings. In each of the five
	// closed phases the product's uniform rejection fails on every worker stop and the Properties about
	// a cancel request or a terminate on every such request; retryCompletes fails on all but one
	// completion of each phase that holds an attempt.
	failures := map[string]int{}
	for _, phase := range []string{"canceled", "completed", "failed", "terminated", "timedOut"} {
		failures["activityProduct.closedIsRejectedUniformly on activitySystem: "+phase+" stop"] = 24
		failures["activitySystem.cancelIsRequested on activitySystem: "+phase+" requestCancel"] = 24
		failures["cancelRequestedWhileStarted on activitySystem: "+phase+" requestCancel"] = 24
		failures["activitySystem.terminateSettles on activitySystem: "+phase+" terminate"] = 24
		failures["terminated on activitySystem: "+phase+" terminate"] = 24
	}
	for _, phase := range []string{"started", "cancelRequested", "pauseRequested"} {
		failures["retryCompletes on activitySystem: "+phase+" respondCompleted"] = 23
	}
	require.Equal(t, failures, failing)

	// Which rows each Property is about and holds on, as a digest of its sorted row answers, so two
	// Properties with the same counts cannot trade rows. Properties that answer every row alike share
	// a digest: the product's three laws on the product and the two that hold everywhere on the
	// protocol, and the protocol's two Properties about a cancel request and its two about a terminate.
	digests := map[string]string{}
	for claim, lines := range answers {
		slices.Sort(lines)
		sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
		digests[claim] = hex.EncodeToString(sum[:8])
	}
	require.Equal(t, map[string]string{
		"activityProduct.closedIsRejectedUniformly on activityProduct": "dcccb38657cbb272",
		"activityProduct.closedIsRejectedUniformly on activitySystem":  "aca3674367133567",
		"activityProduct.pausedIsNotDispatched on activityProduct":     "dcccb38657cbb272",
		"activityProduct.pausedIsNotDispatched on activitySystem":      "b2b01fb82419845f",
		"activityProduct.terminalStatesAreFinal on activityProduct":    "dcccb38657cbb272",
		"activityProduct.terminalStatesAreFinal on activitySystem":     "b2b01fb82419845f",
		"activitySystem.cancelIsRequested on activitySystem":           "d7be46713ca32d2d",
		"activitySystem.terminateSettles on activitySystem":            "9c62747efd31fa10",
		"cancelRequestedWhileStarted on activitySystem":                "d7be46713ca32d2d",
		"canceledByWorker on activitySystem":                           "ac4fc0e790c9c725",
		"completes on activitySystem":                                  "9b2400cb53e0908a",
		"nonRetryableFails on activitySystem":                          "ef6c558943b85ee3",
		"retryCompletes on activitySystem":                             "d212627794c856ab",
		"scheduleToStartFires on activitySystem":                       "b122a9d419983e01",
		"startToCloseFires on activitySystem":                          "50698e45d993235d",
		"startedByPollingWorker on standaloneActivity":                 "c1eeda2f52651784",
		"terminated on activitySystem":                                 "9c62747efd31fa10",
	}, digests)
}

// pathAnswer is what a Query says: its verdict, how much its search explored and expanded, whether
// its claim was read on some step, and each step of its witness: the class, the answer, the state it
// reaches and the facts it records.
type pathAnswer struct {
	Kind               ReceiptKind
	Explored, Expanded int
	Exercised          bool
	Witness            []string
}

func witnessStep(class, outcome, state string, facts []string) string {
	return fmt.Sprintf("%s: %s, %s %v", class, outcome, state, facts)
}

func pathAnswerOf(r Receipt) pathAnswer {
	answer := pathAnswer{Kind: r.Kind, Explored: r.Explored, Expanded: r.Expanded, Exercised: r.Exercised}
	if r.Witness != nil {
		for _, step := range r.Witness.Steps {
			var facts []string
			for _, fact := range step.Facts {
				facts = append(facts, fact.Value)
			}
			answer.Witness = append(answer.Witness, witnessStep(step.Action.Value, step.Outcome.Value, step.State.Value, facts))
		}
	}
	return answer
}

// Every Query of the Model answers as pinned: a found one by the path its Scenario places, a verified
// one over its whole search, and each reads its claim on some step.
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
		return pathAnswer{Kind: Found, Explored: explored, Expanded: explored - 1, Exercised: true, Witness: witness}
	}
	verified := func(explored, expanded int) pathAnswer {
		return pathAnswer{Kind: Verified, Explored: explored, Expanded: expanded, Exercised: true}
	}
	// accepted is a step every actor's request on these paths is answered by.
	accepted := func(class, state string, facts ...string) string { return witnessStep(class, "accepted", state, facts) }
	var (
		scheduled       = accepted("start-unset-unset-unset", "scheduled-0-unset-unset-unset", "statusScheduled")
		stopped         = accepted("stop", "scheduled-0-unset-unset-unset")
		started         = accepted("poll", "started-1-unset-unset-unset", "statusStarted", "attemptCount")
		cancelRequested = accepted("requestCancel", "cancelRequested-1-unset-unset-unset", "statusCancelRequested")
		canceled        = accepted("respondCanceled", "canceled-1-unset-unset-unset", "statusCanceled")
		terminated      = accepted("terminate", "terminated-0-unset-unset-unset", "statusTerminated")
	)
	require.Equal(t, map[string]pathAnswer{
		"activityProduct.closedIsRejectedUniformly": verified(10, 10),
		"activityProduct.pausedIsNotDispatched":     verified(10, 10),
		"activityProduct.terminalStatesAreFinal":    verified(10, 10),
		"activitySystem.cancelIsRequested": found(4, scheduled, stopped,
			accepted("requestCancel", "cancelRequested-0-unset-unset-unset", "statusCancelRequested")),
		"activitySystem.terminateSettles": found(4, scheduled, stopped, terminated),
		"cancel":                          found(5, scheduled, started, cancelRequested, canceled),
		"cancelRequest":                   found(5, scheduled, started, cancelRequested, canceled),
		"completion": found(4, scheduled, started,
			accepted("respondCompleted", "completed-1-unset-unset-unset", "statusCompleted")),
		"nonRetryableFailure": found(4, scheduled, started,
			accepted("respondFailed-fatal", "failed-1-unset-unset-unset", "statusFailed")),
		"pauseResume": found(6, scheduled,
			accepted("pause", "paused-0-unset-unset-unset", "statusPaused"),
			accepted("unpause", "scheduled-0-unset-unset-unset", "statusScheduled"), started,
			accepted("respondCompleted", "completed-1-unset-unset-unset", "statusCompleted")),
		"retry": found(7, scheduled, started,
			accepted("respondFailed-retryable", "backingOff-1-unset-unset-unset", "statusScheduled", "attemptCount"),
			accepted("backoff", "scheduled-1-unset-unset-unset"),
			accepted("poll", "started-2-unset-unset-unset", "statusStarted", "attemptCount"),
			accepted("respondCompleted", "completed-2-unset-unset-unset", "statusCompleted")),
		"scheduleToStartTimeout": found(4,
			accepted("start-unset-expires-unset", "scheduled-0-unset-expires-unset", "statusScheduled"),
			accepted("stop", "scheduled-0-unset-expires-unset"),
			accepted("scheduleToStart", "timedOut-0-unset-expires-unset", "statusTimedOut-scheduleToStart")),
		"startToCloseTimeout": found(4,
			accepted("start-unset-unset-expires", "scheduled-0-unset-unset-expires", "statusScheduled"),
			accepted("poll", "started-1-unset-unset-expires", "statusStarted", "attemptCount"),
			accepted("startToClose", "timedOut-1-unset-unset-expires", "statusTimedOut-startToClose")),
		"terminate":                  found(4, scheduled, stopped, terminated),
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
				f := activityFunction(t, m, "activitySystem.property.cancelRequestedWhileStarted")
				narrowed(f, binary(umpirespb.Binary_OP_EQ, stateField(f, 0, "attempts"), expr(admIntValue(1))))
			},
			rows: []string{"cancelRequestedWhileStarted on activitySystem at scheduled-0-unset-unset-unset-requestCancel"},
		},
		// The one worker answer on the path to a canceled activity is the canceled answer.
		"canceledByWorker is about completion": {
			mutate: func(_ *testing.T, m *umpirespb.Model) {
				admProperty(m, "activitySystem", "canceledByWorker").When = &umpirespb.Property_WhenAction{WhenAction: "respondCompleted"}
			},
			rows: []string{"canceledByWorker on activitySystem at started-1-unset-unset-unset-respondCompleted"},
		},
		// No path that reads this Property terminates the activity. The free search of the product
		// that verifies it does: it terminates a scheduled activity.
		"pausedIsNotDispatched also forbids a terminate": {
			mutate: func(t *testing.T, m *umpirespb.Model) {
				f := activityFunction(t, m, "activityProduct.property.activityProduct.pausedIsNotDispatched")
				narrowed(f, binary(umpirespb.Binary_OP_NE, stateField(f, 1, "phase"),
					expr(admEnum("temporal.features.activity.standalone.product.Phase", "terminated"))))
			},
			rows: []string{
				"activityProduct.pausedIsNotDispatched on activityProduct at scheduled-terminate",
				"activityProduct.pausedIsNotDispatched on activitySystem at scheduled-0-unset-unset-unset-terminate",
			},
			generated: []string{"activityProduct.pausedIsNotDispatched"},
		},
		// The one attempt start on the cross-entity path is the first attempt's.
		"startedByPollingWorker also wants the first attempt": {
			mutate: func(t *testing.T, m *umpirespb.Model) {
				f := activityFunction(t, m, "standaloneActivity.property.startedByPollingWorker")
				narrowed(f, binary(umpirespb.Binary_OP_EQ, field(stateField(f, 0, "activity"), "attempts"), expr(admIntValue(1))))
			},
			rows: []string{"startedByPollingWorker on standaloneActivity at scheduled-1-unset-unset-unset_polling-poll"},
		},
		// The backoff on the cross-entity path is taken while the worker polls, so a claim about the
		// activity's own backoff holds there too, and is read on a step of the path.
		"startedByPollingWorker is about the backoff": {
			mutate: func(_ *testing.T, m *umpirespb.Model) {
				admProperty(m, "standaloneActivity", "startedByPollingWorker").When = &umpirespb.Property_WhenAction{WhenAction: "activity_backoff"}
			},
			rows: []string{
				"startedByPollingWorker on standaloneActivity at scheduled-0-unset-unset-unset_polling-poll",
				"startedByPollingWorker on standaloneActivity at backingOff-1-unset-unset-unset_stopped-activity_backoff",
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			m := proto.Clone(activityModel(t)).(*umpirespb.Model)
			mutant.mutate(t, m)
			require.NoError(t, ir.Validate(m))
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

// The sync poll is named as the activity's action it takes, so that action's one class is
// keyed as the composed step is. The cross-entity Property declared about that class answers every
// row and every path as the one declared about the action does.
func TestActivityCrossEntityClaimByClassAnswersAlike(t *testing.T) {
	m := proto.Clone(activityModel(t)).(*umpirespb.Model)
	var poll string
	for _, a := range m.GetActions() {
		if a.GetName() == "poll" {
			poll = a.GetId()
		}
	}
	require.NotEmpty(t, poll)
	admProperty(m, "standaloneActivity", "startedByPollingWorker").When = &umpirespb.Property_WhenClass{
		WhenClass: &umpirespb.ActionClass{Action: poll}}
	require.NoError(t, ir.Validate(m))
	require.Empty(t, pathDisagreements(t, m))
	require.Empty(t, generatedDisagreements(t, m))
	require.Empty(t, rowDisagreements(irPropertyRows(t, activityModel(t)), irPropertyRows(t, m)))
}
