package checker

import (
	"fmt"
	"strings"
)

// PropertyDecl is a Property, which Queries and the search read.
type PropertyDecl struct {
	Name      string
	Machine   Model
	when      func(action string) bool
	whenLabel string
	// keyHolds and keyHolds2 are the claim of a Property declared over a table's keys.
	keyHolds  func(step Result) (bool, error)
	keyHolds2 func(before string, step Result) (bool, error)
}

// isTransition reports whether the Property relates a step to the state before it.
func (p *PropertyDecl) isTransition() bool { return p.keyHolds2 != nil }

// triggers reports whether a same-step Property is about the step of this action class.
func (p *PropertyDecl) triggers(action string) bool { return p.when == nil || p.when(action) }

// ScenarioDecl is a Scenario.
type ScenarioDecl struct {
	Name    string
	Machine Model
	Start   string
	Actions []string
	free    bool
}

// Limits bounds a Query: steps is the depth bound, actions the schedule length, and search the
// number of product states the search may visit before it reports limit-reached.
type Limits struct {
	Name    string
	Steps   int
	Actions int
	Search  int
}

// QueryForm is what a Query asks.
type QueryForm int

const (
	// FindForm asks for one trace of the Scenario on which the Property holds.
	FindForm QueryForm = iota
	// VerifyForm asks whether the Property holds on every trace of the Scenario.
	VerifyForm
)

// Query is a bounded question about a machine: a Property, a Scenario, and Limits.
type Query struct {
	Name       string
	Form       QueryForm
	Property   *PropertyDecl
	Scenario   *ScenarioDecl
	Limits     Limits
	refinement func() (*Refinement, error)
	monitors   []*Monitor
	// Unknown says which errors of a Property's or a Monitor's functions leave a step unknown rather
	// than fail the search: the step is reported, the search does not continue through it, and an
	// answer with no witness is incomplete. Nil lets every such error fail the search.
	Unknown func(error) bool
}

// Outcome is a Query's answer, spelled as Lean spells `PlanningOutcome.name`.
type Outcome string

const (
	Found                Outcome = "found"
	NotFound             Outcome = "not-found"
	VerifiedWithinLimits Outcome = "verified-within-limits"
	CounterexampleFound  Outcome = "counterexample-found"
	LimitReached         Outcome = "limit-reached"
)

// Atom is a value on a trace with the Definition ID it belongs to, as Lean's `ModelValue`.
type Atom struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// TraceStep is one step of a witness.
type TraceStep struct {
	Action  Atom   `json:"action"`
	Outcome Atom   `json:"outcome"`
	State   Atom   `json:"state"`
	Facts   []Atom `json:"facts"`
}

// Trace is a witness: the start and the steps taken.
type Trace struct {
	Initial Atom        `json:"initial"`
	Steps   []TraceStep `json:"steps"`
}

// Answer is a Query's outcome, its witness for a found Query or its counterexample, and how much
// the search explored.
type Answer struct {
	Outcome     Outcome
	Witness     *Trace
	Explored    int
	Explanation string
	// Rows are the row keys the witness takes, in order.
	Rows []string
	// Exercised reports that the Property's clause fired on some explored step, so a verified
	// answer was earned rather than vacuous.
	Exercised bool
	// Monitor names the Monitor whose violation is the counterexample, or "".
	Monitor string
	// Monitors are the watching Monitors' verdicts, in the order the Query names them.
	Monitors []MonitorVerdict
	// Unknown lists the unknown pairs and steps the search explored, in the order it met them. A pair
	// past the step bound, one the Scenario does not schedule, and one at a state the search never
	// expanded are not explored.
	Unknown []UnknownReach
	// Expanded is how many product states the search read the successors of.
	Expanded int
}

func (q *Query) decl() string { return "query " + q.Name }

// check rejects a Query whose parts do not fit together.
func (q *Query) check() error {
	if q.Property == nil || q.Scenario == nil {
		return errorf(q.decl(), "a Query names a Property and a Scenario")
	}
	if err := q.checkKeyClaims(); err != nil {
		return err
	}
	if q.Form == FindForm && q.Property.isTransition() {
		return errorf(q.decl(), "find names %s, a transition claim; a find realizes a same-step claim",
			q.Property.Name)
	}
	if q.refinement == nil && q.Property.Machine != q.Scenario.Machine {
		return errorf(q.decl(), "%s is declared on %s, but %s runs on %s, which does not refine it",
			q.Property.Name, q.Property.Machine.Name(), q.Scenario.Name, q.Scenario.Machine.Name())
	}
	if len(q.Scenario.Actions) > q.Limits.Actions {
		return errorf(q.decl(), "%s pins %d actions and the limits %s allow %d",
			q.Scenario.Name, len(q.Scenario.Actions), q.Limits.Name, q.Limits.Actions)
	}
	return q.checkMonitors()
}

func describeState(key string) string { return "{" + strings.ReplaceAll(key, "-", ", ") + "}" }

func (a Answer) String() string {
	return fmt.Sprintf("%s after %d product states%s", a.Outcome, a.Explored,
		map[bool]string{true: ": " + a.Explanation, false: ""}[a.Explanation != ""])
}
