package umpire

import (
	"fmt"
	"slices"
	"strings"
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
// Step holds the typed step a Property reads.
type Result struct {
	Outcome string   `json:"outcome"`
	State   string   `json:"state"`
	Facts   []string `json:"facts"`
	Step    any      `json:"-"`
	Because string   `json:"-"`
}

// Row is one enabled state and action class. An absent pair is disabled.
type Row struct {
	Key     string   `json:"key"`
	Source  string   `json:"source"`
	Action  string   `json:"action"`
	Results []Result `json:"results"`
}

// Table is a machine's finite table in Lean's catalog and row order. States, actions, outcomes
// and facts are keys; the typed values they stand for are kept for Properties and compositions.
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
	alter       alterer
	fieldValues map[string][]Atom
	stateValue  map[string]any   // state key to typed state
	classes     map[string]Class // action key to class, for declared machines
	decls       map[string]*ActionDecl
	rowsFrom    map[string][]int
}

// StateValue is the typed state a key stands for.
func (t *Table) StateValue(key string) (any, bool) {
	v, ok := t.stateValue[key]
	return v, ok
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

// CapabilityID is the machine's capability; ProviderID and KernelID are its other own identities.
func (t *Table) CapabilityID() string { return t.capabilityID() }
func (t *Table) ProviderID() string   { return t.providerID() }
func (t *Table) KernelID() string     { return t.kernelID() }

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
	var out []Claim
	for _, c := range t.claims() {
		out = append(out, Claim{Member: t.Family.ID("action", t.owner(), c.classKey),
			Action: string(t.Family) + ".action." + c.decl.Name, Field: c.decl.Inputs[0],
			ClassName: c.spelling, Example: c.example})
	}
	return out
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
	t.Reachable = reachable(t.Starts, t.Rows)
	ends := map[string]bool{}
	for _, e := range t.Ends {
		ends[e] = true
	}
	t.Stuck = ""
	for _, s := range t.Reachable {
		if !ends[s] && len(t.rowsFrom[s]) == 0 {
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
}

func (e *Error) Error() string { return e.Declaration + ": " + e.Message }

func errorf(decl, format string, args ...any) error {
	return &Error{Declaration: decl, Message: fmt.Sprintf(format, args...)}
}

func rowKey(state, action string) string { return state + "-" + action }

func joinKeys(parts []string, sep string) string { return strings.Join(parts, sep) }

// TableSpec is a machine's table computed outside this package, such as by an interpreter of the
// Umpire IR (model/scalav2/goir). It carries only what a table's keys say; a table built from it has
// no typed values, so it serves identities, reachability and fingerprints, not Properties.
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
}

// NewTable builds a table from keys, indexing its rows and computing reachability and the stuck
// state as a declared machine's table does.
func NewTable(spec TableSpec) *Table {
	t := &Table{Machine: spec.Machine, Owner: spec.Owner, Family: spec.Family, States: spec.States,
		Actions: spec.Actions, Outcomes: spec.Outcomes, Facts: spec.Facts, Starts: spec.Starts, Ends: spec.Ends,
		Rows: spec.Rows, StateFields: spec.StateFields, Entity: spec.Entity, Evidence: spec.Evidence,
		stateValue: map[string]any{}, classes: map[string]Class{}, decls: map[string]*ActionDecl{}}
	if t.Facts == nil {
		t.Facts = []string{}
	}
	for _, s := range t.States {
		t.stateValue[s] = s
	}
	t.finish()
	return t
}
