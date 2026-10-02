package checker

import "fmt"

// Answer searches the Query. The search is breadth-first over the product of the machine state,
// the Scenario's progress through its pinned schedule, and the Property monitor, with a visited
// set, as `Umpire.Search.Product` describes. Successors come in the table's row order and, within a
// row, in result order, and the first-discovered parent is kept, so the witness is the shortest
// and ties go to the lower index: the order Veil's checker and the reference search agree on.
func (q *Query) Answer() (Answer, error) {
	s, err := q.searcher()
	if err != nil {
		return Answer{}, err
	}
	a, err := s.run()
	if err != nil {
		return Answer{}, err
	}
	a.Unknown, a.Expanded = s.unknown, s.expanded
	return a, nil
}

// searcher checks the Query and prepares one search of it.
func (q *Query) searcher() (*searcher, error) {
	if err := q.check(); err != nil {
		return nil, err
	}
	t, err := q.Scenario.Machine.Table()
	if err != nil {
		return nil, err
	}
	if err := q.checkScenario(t); err != nil {
		return nil, err
	}
	var ref *Refinement
	if q.refinement != nil {
		if ref, err = q.refinement(); err != nil {
			return nil, err
		}
		if err := q.checkRefined(t, ref); err != nil {
			return nil, err
		}
	}
	ends := map[string]bool{}
	for _, e := range t.Ends {
		ends[e] = true
	}
	return &searcher{q: q, t: t, ref: ref, ends: ends, monRead: make([]bool, len(q.monitors))}, nil
}

func (q *Query) checkScenario(t *Table) error {
	if _, ok := t.StateValue(q.Scenario.Start); !ok {
		return errorf(q.decl(), "%s starts at %s, which is not a state of %s",
			q.Scenario.Name, q.Scenario.Start, t.Machine)
	}
	if q.Scenario.free {
		return nil
	}
	for _, a := range q.Scenario.Actions {
		if _, ok := indexOf(t.Actions, a); !ok {
			return errorf(q.decl(), "%s names %s, which is not an action class of %s",
				q.Scenario.Name, a, t.Machine)
		}
	}
	return nil
}

func (q *Query) checkRefined(t *Table, ref *Refinement) error {
	p := q.Property
	if ref.Product != p.Machine.Name() {
		return errorf(q.decl(), "%s is declared on %s, and the refinement reads %s as %s",
			p.Name, p.Machine.Name(), ref.Machine, ref.Product)
	}
	if err := q.checkKeyRefined(t, ref); err != nil {
		return err
	}
	if p.when == nil {
		return nil
	}
	for _, a := range t.Actions {
		if p.when(a) {
			return nil
		}
	}
	return errorf(q.decl(), "the Property names the action '%s' of '%s', and '%s' has no "+
		"action of that name; a Property on the refined machine is read on the refining one "+
		"through the values of the same name, and a state through its map",
		p.whenLabel, ref.Product, ref.Machine)
}

// monitor is the Property's part of a product state: whether its clause has fired, and whether
// every firing held.
type monitor struct {
	fired, held bool
}

type node struct {
	state  string
	pos    int
	mon    monitor
	mons   []monitorState
	parent int
	row    string
	result Result
	// terminal marks a node the search does not continue past: a step some Monitor could not be read
	// on, kept because what was read of it answers the Query.
	terminal bool
}

type productKey struct {
	state string
	pos   int
	mon   monitor
	mons  string
}

func (s *searcher) keyOf(n node) productKey {
	return productKey{n.state, s.progress(n.pos), n.mon, identity(n.mons)}
}

// searcher is one breadth-first search over a Query's product state space.
type searcher struct {
	q              *Query
	t              *Table
	ref            *Refinement
	ends           map[string]bool
	nodes          []node
	visited        map[productKey]bool
	counterexample int
	found          int
	exercised      bool
	monRead        []bool
	// exceeded is set when a new state would take the search past its limit.
	exceeded bool
	// unknown is the unknowns the search explored, noted once each; expanded counts the nodes whose
	// successors it read.
	unknown  []UnknownReach
	noted    map[[2]string]bool
	expanded int
}

func (s *searcher) free() bool { return s.q.Scenario.free }

// depth is the step bound: the Query's limit, and a pinned schedule's own length.
func (s *searcher) depth() int {
	if s.free() {
		return s.q.Limits.Steps
	}
	return min(s.q.Limits.Steps, len(s.q.Scenario.Actions))
}

// progress is the Scenario's part of a product state. A free Scenario's progress records nothing,
// so two paths reaching one model state with one monitor state are one product state whatever
// their depth: breadth-first search reaches each first at its minimal depth, which is what makes
// the dedup sound under the step bound.
func (s *searcher) progress(pos int) int {
	if s.free() {
		return 0
	}
	return pos
}

func (s *searcher) run() (Answer, error) {
	s.nodes = []node{s.initial()}
	s.visited = map[productKey]bool{s.keyOf(s.nodes[0]): true}
	s.counterexample, s.found = -1, -1
	frontier := []int{0}
	for len(frontier) > 0 || s.exceeded {
		if len(s.visited) > s.q.Limits.Search || s.exceeded {
			if s.counterexample >= 0 {
				// A violation found within the limit stands: `Umpire.Search` gives a verify's
				// counterexample precedence over the traversal's termination.
				return s.conclude(), nil
			}
			return Answer{Outcome: LimitReached, Explored: len(s.visited),
				Explanation: fmt.Sprintf("the limits %s allow %d product states", s.q.Limits.Name, s.q.Limits.Search)}, nil
		}
		var next []int
		for _, i := range frontier {
			added, err := s.expand(i)
			if err != nil {
				return Answer{}, err
			}
			if s.found >= 0 {
				return s.answer(Found, s.found, ""), nil
			}
			if s.exceeded {
				break
			}
			next = append(next, added...)
		}
		frontier = next
	}
	return s.conclude(), nil
}

// expand adds every unvisited successor of one node, in row order then result order.
func (s *searcher) expand(i int) ([]int, error) {
	n := s.nodes[i]
	if n.pos >= s.depth() {
		return nil, nil
	}
	s.expanded++
	s.noteUnknownRows(i)
	var added []int
	for _, row := range s.t.RowsFrom(n.state) {
		if !s.scheduled(n, row) {
			continue
		}
		for _, res := range row.Results {
			next, kept, err := s.successor(i, row, res)
			if err != nil {
				return nil, err
			}
			if !kept {
				continue
			}
			key := s.keyOf(next)
			if s.visited[key] {
				continue
			}
			if len(s.visited) >= s.q.Limits.Search {
				// The limit bounds the states the search visits, so a state past it is neither
				// visited nor read: no answer rests on work the limit does not allow.
				s.exceeded = true
				return added, nil
			}
			s.visited[key] = true
			s.nodes = append(s.nodes, next)
			j := len(s.nodes) - 1
			s.record(j)
			if s.found >= 0 {
				return added, nil
			}
			if !next.terminal {
				added = append(added, j)
			}
		}
	}
	return added, nil
}

func (s *searcher) initial() node {
	mons := make([]monitorState, len(s.q.monitors))
	for k, m := range s.q.monitors {
		mons[k] = monitorState{key: m.Initial}
	}
	return node{state: s.q.Scenario.Start, mon: monitor{held: true}, mons: mons, parent: -1}
}

// scheduled reports whether the Scenario admits a row as the next step after n.
func (s *searcher) scheduled(n node, row Row) bool { return s.admits(n, row.Action) }

// admits reports whether the Scenario admits an action class as the next step after n.
func (s *searcher) admits(n node, action string) bool {
	return s.free() || action == s.q.Scenario.Actions[n.pos]
}

// successor is the node one result of a row leads to from node i, and whether the search keeps it.
// An error of the Property's or a Monitor's function that the Query does not class as unknown fails
// the search. One it does is reported, and the search does not continue through the step, since the
// claim's or a Monitor's state past it is unknown; the step is kept all the same, as a terminal
// node, when what was read of it already answers the Query.
func (s *searcher) successor(i int, row Row, res Result) (node, bool, error) {
	next, err := s.step(i, row, res)
	if err == nil {
		return next, true, nil
	}
	if s.q.Unknown == nil || !s.q.Unknown(err) {
		return node{}, false, err
	}
	s.noteUnknown(UnknownClaim, i, row.Key, row.Action, err)
	next.terminal = true
	return next, s.answers(next), nil
}

// answers reports whether what was read of a step some Monitor could not be read on answers the
// Query: a Monitor is passive, so one that is unknown takes away neither a violation the Property or
// an earlier Monitor established on the step nor a find's witness. A step the Property could not be
// read on is no node, and answers nothing.
func (s *searcher) answers(n node) bool {
	if n.row == "" {
		return false
	}
	if s.q.Form == VerifyForm {
		return s.fails(n)
	}
	return s.realizes(n)
}

// step is the node one result of a row leads to from node i: the Property monitor and every
// watching Monitor advanced over it.
// With an error of a Monitor's function, the node is the step as far as it was read: the Property
// and the Monitors before that one advanced, that one marked unknown in its state before the step,
// and the ones after it as they were. With an error of the Property's, there is no node.
func (s *searcher) step(i int, row Row, res Result) (node, error) {
	n := s.nodes[i]
	mon, err := s.observe(n, row, res)
	if err != nil {
		return node{}, err
	}
	before, _ := s.t.StateValue(n.state)
	mons := make([]monitorState, len(n.mons))
	for k, m := range s.q.monitors {
		ms, err := s.watch(k, m, n, before, res)
		if err != nil {
			copy(mons[k:], n.mons[k:])
			mons[k].unknown = true
			return node{state: res.State, pos: n.pos + 1, mon: mon, mons: mons, parent: i, row: row.Key, result: res},
				wrapError(s.q.decl(), err)
		}
		mons[k] = ms
	}
	return node{state: res.State, pos: n.pos + 1, mon: mon, mons: mons, parent: i, row: row.Key, result: res}, nil
}

// watch advances one watching Monitor over a step from n, reading its verdict where its evaluation
// point says.
func (s *searcher) watch(k int, m *Monitor, n node, before any, res Result) (monitorState, error) {
	ms := n.mons[k]
	var err error
	if ms.key, err = m.next(ms.key, n.state, before, res); err != nil {
		return ms, err
	}
	reads, err := m.At.reads(res, s.ends)
	if err != nil || !reads {
		return ms, err
	}
	violated, err := m.violated(ms.key)
	if err != nil {
		return ms, err
	}
	ms.read, s.monRead[k] = true, true
	ms.violated = ms.violated || violated
	return ms, nil
}

// observe advances the Property monitor over one step.
func (s *searcher) observe(n node, row Row, res Result) (monitor, error) {
	p, mon := s.q.Property, n.mon
	switch {
	case p.keyLevel():
		return s.observeKeys(n, row, res)
	case p.IsTransition():
		before, err := s.readState(n.state)
		if err != nil {
			return mon, err
		}
		after, err := s.readStep(res)
		if err != nil {
			return mon, err
		}
		mon.fired, mon.held = true, mon.held && p.holds2(before, after)
		s.exercised = true
	case p.Triggers(row.Action):
		step, err := s.readStep(res)
		if err != nil {
			return mon, err
		}
		mon.fired, mon.held = true, mon.held && p.holds(step)
		s.exercised = true
	default:
	}
	return mon, nil
}

// record notes whether a new node answers the Query: a completed trace on which a find's claim
// fired and held, or the first step on which a verify's claim failed.
func (s *searcher) record(j int) {
	n := s.nodes[j]
	switch s.q.Form {
	case FindForm:
		if s.realizes(n) {
			s.found = j
		}
	case VerifyForm:
		if s.fails(n) && s.counterexample < 0 {
			s.counterexample = j
		}
	default:
	}
}

// realizes reports whether a node completes a trace on which a find's claim fired and held.
func (s *searcher) realizes(n node) bool {
	complete := s.free() || n.pos == len(s.q.Scenario.Actions)
	return complete && n.mon.fired && n.mon.held
}

// fails reports whether a verify's claim, or a watching Monitor, fails on the path to a node.
func (s *searcher) fails(n node) bool { return !n.mon.held || s.violated(n) != "" }

// violated names the first watching Monitor violated on the path to a node, or "".
func (s *searcher) violated(n node) string {
	for k, m := range n.mons {
		if m.violated {
			return s.q.monitors[k].Name
		}
	}
	return ""
}

func (s *searcher) conclude() Answer {
	if s.q.Form == VerifyForm {
		if s.counterexample >= 0 {
			n := s.nodes[s.counterexample]
			if !n.mon.held {
				return s.answer(CounterexampleFound, s.counterexample, fmt.Sprintf("%s fails at %s",
					s.q.Property.Name, describeState(n.state)))
			}
			a := s.answer(CounterexampleFound, s.counterexample, fmt.Sprintf("the monitor %s is violated at %s",
				s.violated(n), describeState(n.state)))
			a.Monitor = s.violated(n)
			return a
		}
		return Answer{Outcome: VerifiedWithinLimits, Explored: len(s.visited), Exercised: s.exercised,
			Monitors: s.explored()}
	}
	return Answer{Outcome: NotFound, Explored: len(s.visited), Explanation: fmt.Sprintf(
		"no trace of %s within %s reaches %s", s.q.Scenario.Name, s.q.Limits.Name, s.q.Property.Name)}
}

func (s *searcher) answer(outcome Outcome, j int, explanation string) Answer {
	return Answer{Outcome: outcome, Witness: s.witness(j), Rows: s.rowsOf(j), Explored: len(s.visited),
		Explanation: explanation, Exercised: s.exercised, Monitors: s.verdicts(s.nodes[j])}
}

// verdicts is every watching Monitor's verdict on the path to a node.
func (s *searcher) verdicts(n node) []MonitorVerdict {
	var out []MonitorVerdict
	for k, m := range n.mons {
		out = append(out, MonitorVerdict{Name: s.q.monitors[k].Name, State: m.key, Verdict: m.verdict()})
	}
	return out
}

// explored is every watching Monitor's verdict over a search with no witness: held when some
// explored step read it, unread otherwise.
func (s *searcher) explored() []MonitorVerdict {
	var out []MonitorVerdict
	for k, m := range s.q.monitors {
		v := MonitorUnread
		if s.monRead[k] {
			v = MonitorHeld
		}
		out = append(out, MonitorVerdict{Name: m.Name, Verdict: v})
	}
	return out
}

// readState turns a state into what the Property reads: the typed state, or the refined machine's
// typed state through the refinement.
func (s *searcher) readState(key string) (any, error) {
	if s.ref != nil {
		return s.ref.MapValue(key)
	}
	v, _ := s.t.StateValue(key)
	return v, nil
}

// readStep turns a step into what the Property reads: the typed step, or the refined machine's
// typed step through the refinement.
func (s *searcher) readStep(res Result) (any, error) {
	if s.ref == nil {
		return res.Step, nil
	}
	return s.ref.productStep(res)
}

func (s *searcher) rowsOf(j int) []string {
	var rows []string
	for ; s.nodes[j].parent >= 0; j = s.nodes[j].parent {
		rows = append([]string{s.nodes[j].row}, rows...)
	}
	return rows
}

func (s *searcher) witness(j int) *Trace {
	var path []edge
	for ; s.nodes[j].parent >= 0; j = s.nodes[j].parent {
		n := s.nodes[j]
		path = append([]edge{{n.row, n.result}}, path...)
	}
	return s.t.trace(s.nodes[0].state, path)
}

func (t *Table) rowIndex(key string) int {
	for i, r := range t.Rows {
		if r.Key == key {
			return i
		}
	}
	return -1
}

func indexOf(xs []string, x string) (int, bool) {
	for i, y := range xs {
		if y == x {
			return i, true
		}
	}
	return -1, false
}

// productStep reads a refining result as the refined machine's typed step: its state through the
// map, and its outcome and facts by name, which is all a refined Property reads.
func (r *Refinement) productStep(res Result) (any, error) {
	v, err := r.MapValue(res.State)
	if err != nil {
		return nil, err
	}
	return r.stepOf(v, res.Outcome, res.Facts)
}
