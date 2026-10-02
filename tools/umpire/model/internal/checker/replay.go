package checker

import (
	"fmt"
	"slices"
)

// edge is one step of a path through a table: the row taken and the result it produced.
type edge struct {
	row    string
	result Result
}

// trace spells a path through the table as a witness, each value with its Definition ID.
func (t *Table) trace(initial string, path []edge) *Trace {
	owner := t.owner()
	atom := func(kind, value string) Atom { return Atom{ID: t.Family.ID(kind, owner, value), Value: value} }
	var steps []TraceStep
	for _, e := range path {
		facts := []Atom{}
		for _, f := range e.result.Facts {
			facts = append(facts, atom("fact", f))
		}
		row := t.Rows[t.rowIndex(e.row)]
		steps = append(steps, TraceStep{Action: atom("action", row.Action), Outcome: atom("outcome", e.result.Outcome),
			State: atom("state", e.result.State), Facts: facts})
	}
	return &Trace{Initial: atom("state", initial), Steps: steps}
}

// pathTo is the shortest path from a start to a state: breadth-first from the starts in order, rows
// in table order and results in result order, keeping the first-discovered parent.
func (t *Table) pathTo(target string) (string, []edge, bool) {
	type visit struct {
		start  bool
		parent string
		via    edge
	}
	seen := map[string]visit{}
	var frontier []string
	for _, s := range t.Starts {
		if _, ok := seen[s]; !ok {
			seen[s] = visit{start: true}
			frontier = append(frontier, s)
		}
	}
	for _, found := seen[target]; !found && len(frontier) > 0; _, found = seen[target] {
		var next []string
		for _, s := range frontier {
			for _, row := range t.RowsFrom(s) {
				for _, res := range row.Results {
					if _, ok := seen[res.State]; !ok {
						seen[res.State] = visit{parent: s, via: edge{row.Key, res}}
						next = append(next, res.State)
					}
				}
			}
		}
		frontier = next
	}
	v, ok := seen[target]
	if !ok {
		return "", nil, false
	}
	var path []edge
	s := target
	for ; !v.start; v = seen[s] {
		path = append([]edge{v.via}, path...)
		s = v.parent
	}
	return s, path, true
}

// Replay checks that a witness is a path of this table: it starts in one of its states and each
// step is a result of the row its action takes from the state before it.
// Every value's Definition ID must be the one this table gives it.
func (t *Table) Replay(w *Trace) error {
	_, err := t.replay(w)
	return err
}

// replay reads a witness back as the path it takes.
func (t *Table) replay(w *Trace) ([]edge, error) {
	if w == nil {
		return nil, errorf(t.Machine, "there is no witness to replay")
	}
	state := w.Initial.Value
	if _, ok := t.stateValue[state]; !ok {
		return nil, errorf(t.Machine, "the witness starts at '%s', which is not a state", state)
	}
	if err := t.binds("the start", "state", w.Initial); err != nil {
		return nil, err
	}
	var path []edge
	for i, step := range w.Steps {
		if err := t.bindsStep(i+1, step); err != nil {
			return nil, err
		}
		enabled, row, _ := t.pair(state, step.Action.Value)
		if enabled != pairEnabled {
			return nil, errorf(t.Machine, "step %d takes %s, which is not enabled at '%s'", i+1, step.Action.Value, state)
		}
		k := slices.IndexFunc(row.Results, func(r Result) bool { return step.matches(r) })
		if k < 0 {
			return nil, errorf(t.Machine, "step %d takes %s from '%s' to '%s', which is no result of that row",
				i+1, step.Action.Value, state, step.State.Value)
		}
		path = append(path, edge{row.Key, row.Results[k]})
		state = step.State.Value
	}
	return path, nil
}

// binds checks that a witness value carries the Definition ID this table gives it, so a witness of
// another family, owner or kind of definition is not read as one of this table's.
func (t *Table) binds(what, kind string, a Atom) error {
	if want := t.Family.ID(kind, t.owner(), a.Value); a.ID != want {
		return errorf(t.Machine, "%s '%s' is identified as %s, which is not the Definition ID %s", what, a.Value, a.ID, want)
	}
	return nil
}

func (t *Table) bindsStep(i int, step TraceStep) error {
	if err := t.binds(fmt.Sprintf("step %d's action", i), "action", step.Action); err != nil {
		return err
	}
	if err := t.binds(fmt.Sprintf("step %d's outcome", i), "outcome", step.Outcome); err != nil {
		return err
	}
	if err := t.binds(fmt.Sprintf("step %d's state", i), "state", step.State); err != nil {
		return err
	}
	for _, f := range step.Facts {
		if err := t.binds(fmt.Sprintf("step %d's fact", i), "fact", f); err != nil {
			return err
		}
	}
	return nil
}

func (s TraceStep) matches(r Result) bool {
	if s.Outcome.Value != r.Outcome || s.State.Value != r.State || len(s.Facts) != len(r.Facts) {
		return false
	}
	for i, f := range s.Facts {
		if f.Value != r.Facts[i] {
			return false
		}
	}
	return true
}

// Replay checks an answer's witness against the same table, Property and Monitors the search read:
// the witness is a path the Scenario admits from its start, and it ends where the answer says, a
// completed trace on which a find's claim held or a step on which a verify's claim or the named
// Monitor failed.
func (q *Query) Replay(a Answer) error {
	s, err := q.searcher()
	if err != nil {
		return err
	}
	path, err := s.t.replay(a.Witness)
	if err != nil {
		return wrapError(q.decl(), err)
	}
	if a.Witness.Initial.Value != q.Scenario.Start {
		return errorf(q.decl(), "the witness starts at '%s', and %s starts at '%s'",
			a.Witness.Initial.Value, q.Scenario.Name, q.Scenario.Start)
	}
	s.nodes = []node{s.initial()}
	for i, e := range path {
		row := s.t.Rows[s.t.rowIndex(e.row)]
		if s.nodes[i].pos >= s.depth() || !s.scheduled(s.nodes[i], row) {
			return errorf(q.decl(), "step %d takes %s, which %s does not schedule there", i+1, row.Action, q.Scenario.Name)
		}
		n, err := s.step(i, row, e.result)
		// A witness may end on a step some Monitor could not be read on, when what was read of the
		// step already answers the Query.
		ends := i == len(path)-1 && s.q.Unknown != nil
		if err != nil && (!ends || !s.q.Unknown(err) || !s.answers(n)) {
			return err
		}
		s.nodes = append(s.nodes, n)
	}
	last := s.nodes[len(s.nodes)-1]
	switch a.Outcome {
	case Found:
		if !s.realizes(last) {
			return errorf(q.decl(), "the witness does not complete a trace of %s on which %s holds",
				q.Scenario.Name, q.Property.Name)
		}
	case CounterexampleFound:
		if !s.fails(last) || a.Monitor != "" && !slices.ContainsFunc(s.verdicts(last), func(v MonitorVerdict) bool {
			return v.Name == a.Monitor && v.Verdict == MonitorViolated
		}) {
			return errorf(q.decl(), "the witness ends in a state where no claim fails")
		}
	default:
		return errorf(q.decl(), "a %s answer has no witness to replay", a.Outcome)
	}
	return nil
}
