// Package checker evaluates claims over finite tables: the search, refinement, composition, monitor
// and progress checks the model reader runs over a table's keys.
package checker

import (
	"fmt"
	"maps"
	"slices"
)

// Family is the root a model's Definition IDs hang off, such as "temporal.nexus.caller". Lean
// derives it from the namespace below Temporal.Feature; Go packages name it explicitly.
type Family string

// ID is `<family>.<kind>.<owner>.<member>`, the shape `Umpire.Command.Origin.ownedId` builds.
func (f Family) ID(kind, owner, member string) string {
	return string(f) + "." + kind + "." + owner + "." + member
}

// Target is the machine's own Definition ID.
func (f Family) Target(machine string) string { return string(f) + ".target." + machine }

// Result is one outcome of a row: the outcome, the next state, and the recorded facts, all as keys.
// Step holds what the table's builder carries with the result, such as a ComposedStep. Because and
// Choice are what the step record explains and names it: inert, so no key, ID or fingerprint reads them.
type Result struct {
	Outcome string   `json:"outcome"`
	State   string   `json:"state"`
	Facts   []string `json:"facts"`
	Step    any      `json:"-"`
	Because string   `json:"-"`
	Choice  string   `json:"-"`
}

// Row is one enabled state and action class. An absent pair is disabled.
type Row struct {
	Key     string   `json:"key"`
	Source  string   `json:"source"`
	Action  string   `json:"action"`
	Results []Result `json:"results"`
}

// Table is a machine's finite table in Lean's catalog and row order. States, actions, outcomes
// and facts are keys.
type Table struct {
	Machine   string   `json:"machine"`
	States    []string `json:"states"`
	Actions   []string `json:"actions"`
	Outcomes  []string `json:"outcomes"`
	Facts     []string `json:"facts"`
	Starts    []string `json:"starts"`
	Ends      []string `json:"ends"`
	Reachable []string `json:"reachable"`
	Rows      []Row    `json:"transitions"`

	// Unknown lists the pairs whose steps are unknown: neither a row nor a disabled pair. A search
	// reports the ones it explores and takes none; no identity, reachability or fingerprint reads them.
	Unknown []UnknownPair `json:"-"`

	// Stuck is the first reachable state that is not an end and has no row, or "".
	Stuck string `json:"-"`

	// Owner is the name Definition IDs hang off: the machine's name, or "compose-<name>".
	Owner        string   `json:"-"`
	Family       Family   `json:"-"`
	StateFields  []string `json:"-"`
	refinedField string
	// Entity is the name of the entity the machine keeps state for.
	Entity string
	// Evidence is the machine's evidence lines in declaration order: a fact constructor and the
	// recorded kind that confirms it.
	Evidence    [][2]string
	Assumptions []Assumption
	fieldValues map[string][]Atom
	// keyClaims is the Abstraction Claims a table built from keys was given.
	keyClaims   []Claim
	rowsFrom    map[string][]int
	unknownFrom map[string][]int
	parts       map[string][]string
	model       *tableModel
	err         error
}

// UnknownPair is a state and action class whose steps are unknown, such as a hole of the IR: Row is
// the key the pair would have as a row, and Cause what left it unknown.
type UnknownPair struct {
	Row    string
	Source string
	Action string
	Cause  error
}

// unknownsFrom lists the unknown pairs at this state, in the order the table lists them.
func (t *Table) unknownsFrom(state string) []UnknownPair {
	idx := t.unknownFrom[state]
	out := make([]UnknownPair, len(idx))
	for i, j := range idx {
		out[i] = t.Unknown[j]
	}
	return out
}

// pairStatus is what a table says of a state and an action class. The order is the rule every check
// reads pairs by: a step that needs several moves has the least of their statuses, so a disabled
// move disables it and an unknown one leaves it unknown; a state that needs only one of its pairs
// has the greatest of theirs.
type pairStatus int

const (
	// pairDisabled is a pair with no step: no row, or a row with no result.
	pairDisabled pairStatus = iota
	// pairUnknown is an unknown pair: it may have a step.
	pairUnknown
	// pairEnabled is a row with a result.
	pairEnabled
)

// pair reads one state and action class: its status, the row of an enabled pair, and the unknown
// pair of an unknown one. It is the one place that tells the three apart.
func (t *Table) pair(state, action string) (pairStatus, Row, UnknownPair) {
	for _, j := range t.rowsFrom[state] {
		if r := t.Rows[j]; r.Action == action {
			if len(r.Results) == 0 {
				return pairDisabled, Row{}, UnknownPair{}
			}
			return pairEnabled, r, UnknownPair{}
		}
	}
	for _, j := range t.unknownFrom[state] {
		if u := t.Unknown[j]; u.Action == action {
			return pairUnknown, Row{}, u
		}
	}
	return pairDisabled, Row{}, UnknownPair{}
}

// steps is whether a state takes a step: enabled when some pair at it is, unknown when none is and
// some pair is unknown, and disabled otherwise.
func (t *Table) steps(state string) pairStatus {
	status := pairDisabled
	if len(t.unknownFrom[state]) > 0 {
		status = pairUnknown
	}
	for _, j := range t.rowsFrom[state] {
		if len(t.Rows[j].Results) > 0 {
			return pairEnabled
		}
	}
	return status
}

// Err is the declaration error of a table built from a spec that does not fit together, or nil.
// NewTable has no error to return, so every check of such a table reports this instead of reading it.
func (t *Table) Err() error { return t.err }

// Parts is the member state keys a composed state's key stands for, in member order.
func (t *Table) Parts(state string) ([]string, bool) {
	parts, ok := t.parts[state]
	return slices.Clone(parts), ok
}

// RowsFrom lists the rows whose source is this state, in table order.
func (t *Table) RowsFrom(state string) []Row {
	idx := t.rowsFrom[state]
	out := make([]Row, len(idx))
	for i, j := range idx {
		out[i] = t.Rows[j]
	}
	return out
}

// FieldValues is a state's fields as atoms: each field's Definition ID and the spelling the state
// holds it at.
func (t *Table) FieldValues(state string) []Atom { return t.fieldValues[state] }

// StateAtom is a state as the Model Value a trace and a Contract carry: its Definition ID and its
// key. ActionAtom, OutcomeAtom and FactAtom are the same for the other members.
func (t *Table) StateAtom(key string) Atom {
	return Atom{ID: t.Family.ID("state", t.owner(), key), Value: key}
}
func (t *Table) ActionAtom(key string) Atom {
	return Atom{ID: t.Family.ID("action", t.owner(), key), Value: key}
}
func (t *Table) OutcomeAtom(key string) Atom {
	return Atom{ID: t.Family.ID("outcome", t.owner(), key), Value: key}
}
func (t *Table) FactAtom(key string) Atom {
	return Atom{ID: t.Family.ID("fact", t.owner(), key), Value: key}
}

// OwnerName is the name the table's Definition IDs hang off.
func (t *Table) OwnerName() string { return t.owner() }

// Claim is one Abstraction Claim the machine's actions make: the class member that realizes it, the
// action declaration, the input field, the class spelled as Lean spells it, and the example.
type Claim struct {
	Member    string
	Action    string
	Field     string
	ClassName string
	Example   string
}

// Claims lists the machine's Abstraction Claims in claim order.
func (t *Table) Claims() []Claim {
	return slices.Clone(t.keyClaims)
}

// IDs is the table's Definition IDs, in catalog order.
type IDs struct {
	Target      string      `json:"target"`
	States      []string    `json:"states"`
	StateFields [][2]string `json:"stateFields"`
	Actions     []string    `json:"actions"`
	Outcomes    []string    `json:"outcomes"`
	Facts       []string    `json:"facts"`
}

// IDs derives every Definition ID the machine owns.
func (t *Table) IDs() IDs {
	owner := t.owner()
	ids := IDs{Target: t.Family.Target(owner)}
	for _, s := range t.States {
		ids.States = append(ids.States, t.Family.ID("state", owner, s))
	}
	ids.StateFields = [][2]string{}
	for _, f := range t.StateFields {
		ids.StateFields = append(ids.StateFields, [2]string{f, t.Family.ID("state-field", owner, f)})
	}
	for _, a := range t.Actions {
		ids.Actions = append(ids.Actions, t.Family.ID("action", owner, a))
	}
	for _, o := range t.Outcomes {
		ids.Outcomes = append(ids.Outcomes, t.Family.ID("outcome", owner, o))
	}
	ids.Facts = []string{}
	for _, f := range t.Facts {
		ids.Facts = append(ids.Facts, t.Family.ID("fact", owner, f))
	}
	return ids
}

func (t *Table) owner() string {
	if t.Owner != "" {
		return t.Owner
	}
	return t.Machine
}

// finish indexes the rows and computes reachability and the stuck state.
func (t *Table) finish() {
	t.rowsFrom = map[string][]int{}
	for i, r := range t.Rows {
		t.rowsFrom[r.Source] = append(t.rowsFrom[r.Source], i)
	}
	t.unknownFrom = map[string][]int{}
	if t.err == nil {
		for i, u := range t.Unknown {
			t.unknownFrom[u.Source] = append(t.unknownFrom[u.Source], i)
		}
	}
	t.model = &tableModel{table: t}
	t.Reachable = reachable(t.Starts, t.Rows)
	ends := map[string]bool{}
	for _, e := range t.Ends {
		ends[e] = true
	}
	t.Stuck = ""
	for _, s := range t.Reachable {
		if !ends[s] && t.steps(s) != pairEnabled {
			t.Stuck = s
			break
		}
	}
}

// reachable is `Umpire.Command.reachableFrom`: sweep the rows in table order, appending each newly
// reached result state, until a sweep adds nothing.
func reachable(starts []string, rows []Row) []string {
	seen := slices.Clone(starts)
	in := map[string]bool{}
	for _, s := range seen {
		in[s] = true
	}
	for {
		grown := false
		for _, r := range rows {
			if !in[r.Source] {
				continue
			}
			for _, res := range r.Results {
				if !in[res.State] {
					in[res.State] = true
					seen = append(seen, res.State)
					grown = true
				}
			}
		}
		if !grown {
			return seen
		}
	}
}

// Error is a model-checking failure, reported against the declaration it belongs to.
type Error struct {
	Declaration string
	Message     string
	cause       error
}

func (e *Error) Error() string { return e.Declaration + ": " + e.Message }

// Unwrap is the error a callback returned, when the failure reports one, so its type survives.
func (e *Error) Unwrap() error { return e.cause }

// wrapError reports another error against a declaration, spelled as it spells itself.
func wrapError(decl string, err error) error {
	return &Error{Declaration: decl, Message: err.Error(), cause: err}
}

func errorf(decl, format string, args ...any) error {
	return &Error{Declaration: decl, Message: fmt.Sprintf(format, args...)}
}

func rowKey(state, action string) string { return state + "-" + action }

// TableSpec is a machine's table computed outside this package, such as by an interpreter of the
// Umpire IR (tools/umpire/model). It carries only what a table's keys say. It serves identities,
// reachability and fingerprints, and claims declared over its keys (KeyProperty, KeyScenario,
// KeyFind): such a Property's predicate reads a result's keys, is searched, and lowers to clauses.
// The state fields and Abstraction Claims are given with the spec.
type TableSpec struct {
	Machine     string
	Owner       string
	Family      Family
	States      []string
	Actions     []string
	Outcomes    []string
	Facts       []string
	Starts      []string
	Ends        []string
	Rows        []Row
	StateFields []string
	Entity      string
	Evidence    [][2]string
	Assumptions []Assumption
	// Unknown lists the pairs whose steps are unknown. A pair is a state and an action class of the
	// table, is keyed as its row would be, is no row of it, and is listed once.
	Unknown []UnknownPair
	// RefinedField names the state field that carries the refined machine's state, for a refining
	// machine: a reading of its state a composition does not carry.
	RefinedField string
	// FieldValues is each state's fields as atoms, by the state's key, for a table whose states are
	// structured. Every key is a state.
	FieldValues map[string][]Atom
	// Claims lists the Abstraction Claims of the actions the table's classes are of, in claim order.
	// Every Member is an action class's Definition ID.
	Claims []Claim
}

// NewTable builds a table from keys, indexing its rows and computing reachability and the stuck
// state.
// A spec that does not fit together (see checkSpec) still builds a table, whose Err every check of
// it reports.
func NewTable(spec TableSpec) *Table {
	t := &Table{Machine: spec.Machine, Owner: spec.Owner, Family: spec.Family, States: spec.States,
		Actions: spec.Actions, Outcomes: spec.Outcomes, Facts: spec.Facts, Starts: spec.Starts, Ends: spec.Ends,
		Rows: spec.Rows, StateFields: spec.StateFields, Entity: spec.Entity, Evidence: spec.Evidence}
	t.Assumptions = spec.Assumptions
	t.Unknown, t.refinedField = spec.Unknown, spec.RefinedField
	t.fieldValues, t.keyClaims = spec.FieldValues, spec.Claims
	t.err = t.checkSpec()
	if t.Facts == nil {
		t.Facts = []string{}
	}
	t.finish()
	return t
}

// checkSpec rejects unknown pairs and a refined field that do not fit the table's catalogs and rows.
// It rejects field values of what is no state and a claim of what is no action class the same way,
// and then a table with no start and rows that do not fit its catalogs.
func (t *Table) checkSpec() error {
	if t.refinedField != "" && !slices.Contains(t.StateFields, t.refinedField) {
		return errorf(t.Machine, "the refined field %s is not a state field", t.refinedField)
	}
	for _, state := range slices.Sorted(maps.Keys(t.fieldValues)) {
		if !slices.Contains(t.States, state) {
			return errorf(t.Machine, "the fields of '%s' are given, and it is not a state", state)
		}
	}
	for _, c := range t.keyClaims {
		if !slices.ContainsFunc(t.Actions, func(a string) bool { return t.ActionAtom(a).ID == c.Member }) {
			return errorf(t.Machine, "the claim on %s is of no action class", c.Member)
		}
	}
	type pair struct{ source, action string }
	rows, rowPairs := map[string]bool{}, map[pair]bool{}
	for _, r := range t.Rows {
		rows[r.Key], rowPairs[pair{r.Source, r.Action}] = true, true
	}
	seen, seenPairs := map[string]bool{}, map[pair]bool{}
	for _, u := range t.Unknown {
		at := pair{u.Source, u.Action}
		switch {
		case !slices.Contains(t.States, u.Source):
			return errorf(t.Machine, "the unknown pair '%s' is at '%s', which is not a state", u.Row, u.Source)
		case !slices.Contains(t.Actions, u.Action):
			return errorf(t.Machine, "the unknown pair '%s' takes %s, which is not an action class", u.Row, u.Action)
		case u.Row != rowKey(u.Source, u.Action):
			return errorf(t.Machine, "the unknown pair '%s' is at '%s' and takes %s, so its key is '%s'",
				u.Row, u.Source, u.Action, rowKey(u.Source, u.Action))
		case rows[u.Row] || rowPairs[at]:
			return errorf(t.Machine, "the pair '%s' is unknown and is a row; a pair is one or the other", u.Row)
		case seen[u.Row] || seenPairs[at]:
			return errorf(t.Machine, "the pair '%s' is unknown twice", u.Row)
		default:
			seen[u.Row], seenPairs[at] = true, true
		}
	}
	return t.checkRows()
}

// checkRows rejects a table with no start, and a row that is not at a state, takes no action class,
// shares its key with another, or leads to what is no state. The IR's interpreter builds no such
// table; a declared machine's are rejected before they are listed.
func (t *Table) checkRows() error {
	if len(t.Starts) == 0 {
		return errorf(t.Machine, "the table has no start")
	}
	states, actions, keys := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, s := range t.States {
		states[s] = true
	}
	for _, a := range t.Actions {
		actions[a] = true
	}
	for _, r := range t.Rows {
		switch {
		case !states[r.Source]:
			return errorf(t.Machine, "the row '%s' is at '%s', which is not a state", r.Key, r.Source)
		case !actions[r.Action]:
			return errorf(t.Machine, "the row '%s' takes %s, which is not an action class", r.Key, r.Action)
		case keys[r.Key]:
			return errorf(t.Machine, "the row '%s' is listed twice", r.Key)
		}
		keys[r.Key] = true
		for _, res := range r.Results {
			if !states[res.State] {
				return errorf(t.Machine, "the row '%s' leads to '%s', which is not a state", r.Key, res.State)
			}
		}
	}
	return nil
}
