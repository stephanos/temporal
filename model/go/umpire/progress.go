package umpire

import (
	"fmt"
	"slices"
	"strings"
)

// Assumption is a name every result of a check that relies on it carries, as the IR's `Assumption`.
// Its Fair entries name action classes, or actions for all of their classes, that are weakly fair
// under it: a path a progress check considers does not keep one of those classes enabled at every
// state of a cycle without taking it there.
type Assumption struct {
	Name string
	Fair []string
}

// Assumes lists the assumptions every check of the machine relies on.
func (m *Machine[S, O, F]) Assumes(assumptions ...Assumption) *Machine[S, O, F] {
	m.assumptions = append(m.assumptions, assumptions...)
	return m
}

// Progress is a bounded progress claim of a table, as the IR's `Progress`: from every reachable
// state From accepts, a state To accepts follows within Within steps, on every path the table's and
// the claim's assumptions admit. A To state at the From state itself discharges it, as TLA+'s `~>`
// reads.
type Progress struct {
	Name        string
	Within      int
	Assumptions []Assumption
	from, to    func(t *Table, state string) (bool, error)
}

// NewProgress declares a progress claim over a machine whose state type is S.
func NewProgress[S any](name string, from, to func(S) bool, within int, assumptions ...Assumption) *Progress {
	typed := func(f func(S) bool) func(*Table, string) (bool, error) {
		return func(t *Table, key string) (bool, error) {
			v, _ := t.StateValue(key)
			s, ok := v.(S)
			if !ok {
				return false, errorf("progress "+name, "the state %s of %s is not a %T", key, t.Machine, s)
			}
			return f(s), nil
		}
	}
	return &Progress{Name: name, Within: within, Assumptions: assumptions, from: typed(from), to: typed(to)}
}

// KeyProgress declares a progress claim over state keys, for a table with no typed values.
func KeyProgress(name string, from, to func(state string) bool, within int, assumptions ...Assumption) *Progress {
	keyed := func(f func(string) bool) func(*Table, string) (bool, error) {
		return func(_ *Table, key string) (bool, error) { return f(key), nil }
	}
	return &Progress{Name: name, Within: within, Assumptions: assumptions, from: keyed(from), to: keyed(to)}
}

// ProgressKind is one of the ways a progress claim is violated, which are reported apart.
type ProgressKind string

const (
	// DeadlockKind is a path from a From state that reaches a state with no row before a To state.
	DeadlockKind ProgressKind = "deadlock"
	// CycleKind is a fair non-progress cycle: a cycle reachable from a From state through no To
	// state, on which no state is a To state and every fair class enabled throughout it is taken.
	CycleKind ProgressKind = "fair-cycle"
	// DeadlineKind is a path of Within steps from a From state with no To state on it.
	DeadlineKind ProgressKind = "deadline"
)

// Unresolved is a check whose Steps limit left reachable states unexplored: absence of a violation
// is unproven, and a finite prefix that ends open proves nothing either.
const Unresolved Outcome = "unresolved"

// ProgressVerdict is one kind of violation's answer: CounterexampleFound with its witness, which
// starts at a start of the table; VerifiedWithinLimits once every reachable state was explored;
// Unresolved when the Steps limit cut the exploration; LimitReached when the Search limit cut the
// work the verdict needs.
type ProgressVerdict struct {
	Outcome Outcome
	Witness *Trace
	// Loop is the index of the state a lasso witness returns to at its end, or -1.
	Loop        int
	Explanation string
}

// ProgressAnswer is a progress claim's answer: a verdict per kind, the assumptions it relies on,
// how many reachable states From accepts, and how much work the check spent.
type ProgressAnswer struct {
	Claim       string
	Assumptions []string
	Deadlock    ProgressVerdict
	Cycle       ProgressVerdict
	Deadline    ProgressVerdict
	From        int
	Explored    int
}

// CheckProgress checks a progress claim of a table within limits. Steps bounds the depth explored
// from the starts. Search bounds the whole check's work, which Explored reports: a unit for each
// state explored, for each region state the cycle and deadline checks read, and for each step of a
// cycle or deadline witness they build; no work or memory grows with Within. A verdict whose work
// the remaining units cannot pay for is LimitReached, and a violation found before stands.
func CheckProgress(t *Table, p *Progress, limits Limits) (ProgressAnswer, error) {
	c, err := newProgressChecker(t, p, limits)
	if err != nil {
		return ProgressAnswer{}, err
	}
	if err := c.explore(); err != nil {
		return ProgressAnswer{}, err
	}
	a := ProgressAnswer{Claim: p.Name, Assumptions: c.assumptionNames(), From: len(c.sources)}
	a.Deadlock = c.verdict(DeadlockKind, c.dead, false)
	parts := c.parts()
	cycle, cut := c.cycle(parts)
	a.Cycle = c.verdict(CycleKind, cycle, cut)
	deadline, cut := c.deadline(parts)
	a.Deadline = c.verdict(DeadlineKind, deadline, cut)
	a.Explored = c.budget.used
	return a, nil
}

// budget is the work a check may still spend.
type budget struct {
	limit, used int
}

// spend takes n units if they are left.
func (b *budget) spend(n int) bool {
	if n > b.limit-b.used {
		return false
	}
	b.used += n
	return true
}

func (b *budget) left() int { return b.limit - b.used }

// progressChecker explores one table for one progress claim.
type progressChecker struct {
	t      *Table
	p      *Progress
	limits Limits
	budget budget
	fair   []fairClass
	// order lists the explored states in breadth-first order from the starts, parent their tree.
	order    []string
	parent   map[string]edgeFrom
	isTo     map[string]bool
	sources  []string
	open     bool
	exceeded bool
	// region is every explored state reachable from a source through no To state, in the order it
	// joined; regionParent is its tree, and dead the first deadlock it found.
	region       []string
	inRegion     map[string]bool
	regionParent map[string]edgeFrom
	dead         *violation
}

type edgeFrom struct {
	source string
	edge
}

type fairClass struct {
	class      string
	assumption string
}

func newProgressChecker(t *Table, p *Progress, limits Limits) (*progressChecker, error) {
	decl := "progress " + p.Name
	if p.Within < 1 {
		return nil, errorf(decl, "within %d steps is fewer than one", p.Within)
	}
	if limits.Steps < 0 || limits.Search < 0 {
		return nil, errorf(decl, "the limits %s are below 0", limits.Name)
	}
	c := &progressChecker{t: t, p: p, limits: limits, budget: budget{limit: limits.Search},
		parent: map[string]edgeFrom{}, isTo: map[string]bool{}, inRegion: map[string]bool{},
		regionParent: map[string]edgeFrom{}}
	for _, a := range c.assumptions() {
		for _, f := range a.Fair {
			found := false
			for _, class := range t.Actions {
				if class == f || actionName(class) == f {
					c.fair = append(c.fair, fairClass{class, a.Name})
					found = true
				}
			}
			if !found {
				return nil, errorf(decl, "the assumption %s makes %s fair, which is no action of %s", a.Name, f, t.Machine)
			}
		}
	}
	return c, nil
}

// assumptions is the table's assumptions followed by the claim's, each name once, fair for the
// classes every declaration of that name makes fair.
func (c *progressChecker) assumptions() []Assumption {
	var out []Assumption
	for _, a := range append(slices.Clone(c.t.Assumptions), c.p.Assumptions...) {
		out = mergeAssumption(out, a)
	}
	return out
}

func (c *progressChecker) assumptionNames() []string {
	var out []string
	for _, a := range c.assumptions() {
		out = append(out, a.Name)
	}
	return out
}

// explore walks the table breadth-first from its starts within the limits, growing the region and
// noting the first deadlock as it goes, so a deadlock found before the ceiling stands.
func (c *progressChecker) explore() error {
	var frontier []string
	for _, s := range c.t.Starts {
		if _, ok := c.parent[s]; ok {
			continue
		}
		if !c.budget.spend(1) {
			c.exceeded = true
			return nil
		}
		if err := c.discover(s, edgeFrom{}); err != nil {
			return err
		}
		frontier = append(frontier, s)
	}
	for depth := 0; len(frontier) > 0 && !c.exceeded; depth++ {
		if depth == c.limits.Steps {
			c.open = slices.ContainsFunc(frontier, c.leavesExplored)
			return nil
		}
		next, err := c.expand(frontier)
		if err != nil {
			return err
		}
		frontier = next
	}
	return nil
}

// expand explores the unexplored successors of a frontier, in row and result order, until the
// budget runs out.
func (c *progressChecker) expand(frontier []string) ([]string, error) {
	var next []string
	for _, s := range frontier {
		for _, row := range c.t.RowsFrom(s) {
			for _, res := range row.Results {
				via := edgeFrom{s, edge{row.Key, res}}
				if _, ok := c.parent[res.State]; ok {
					c.enter(res.State, via)
					continue
				}
				if !c.budget.spend(1) {
					c.exceeded = true
					return next, nil
				}
				if err := c.discover(res.State, via); err != nil {
					return nil, err
				}
				next = append(next, res.State)
			}
		}
	}
	return next, nil
}

// discover explores a state: it evaluates the claim there and joins the region as a source, the
// states From accepts and To does not, or through the step that reached it.
func (c *progressChecker) discover(s string, via edgeFrom) error {
	c.parent[s] = via
	c.order = append(c.order, s)
	from, err := c.p.from(c.t, s)
	if err != nil {
		return err
	}
	to, err := c.p.to(c.t, s)
	if err != nil {
		return err
	}
	c.isTo[s] = to
	if from && !to {
		c.sources = append(c.sources, s)
		c.join(s, edgeFrom{})
		return nil
	}
	c.enter(s, via)
	return nil
}

// enter joins an explored state to the region through a step from a region state.
func (c *progressChecker) enter(s string, via edgeFrom) {
	if via.source != "" && c.inRegion[via.source] && !c.isTo[s] && !c.inRegion[s] {
		c.join(s, via)
	}
}

// join adds a state to the region, and every explored state it reaches through no To state.
func (c *progressChecker) join(s string, via edgeFrom) {
	pending := []edgeFrom{via}
	targets := []string{s}
	for len(pending) > 0 {
		u, v := targets[0], pending[0]
		targets, pending = targets[1:], pending[1:]
		if c.inRegion[u] {
			continue
		}
		c.inRegion[u] = true
		c.regionParent[u] = v
		c.region = append(c.region, u)
		if c.dead == nil && len(c.t.RowsFrom(u)) == 0 {
			start, path := c.regionPath(u)
			c.dead = &violation{start: start, path: path, loop: -1,
				why: fmt.Sprintf("'%s' has no step and %s does not hold there", u, c.p.Name)}
		}
		for _, x := range c.inside(u) {
			if !c.inRegion[x.result.State] {
				targets = append(targets, x.result.State)
				pending = append(pending, edgeFrom{u, x})
			}
		}
	}
}

// leavesExplored reports whether a state has a step to a state not yet explored.
func (c *progressChecker) leavesExplored(s string) bool {
	for _, row := range c.t.RowsFrom(s) {
		for _, res := range row.Results {
			if _, ok := c.parent[res.State]; !ok {
				return true
			}
		}
	}
	return false
}

// inside lists the steps from an explored state to an explored state no To accepts.
func (c *progressChecker) inside(s string) []edge {
	var out []edge
	for _, row := range c.t.RowsFrom(s) {
		for _, res := range row.Results {
			if _, explored := c.parent[res.State]; explored && !c.isTo[res.State] {
				out = append(out, edge{row.Key, res})
			}
		}
	}
	return out
}

// regionPath is the witness path to a region state: from a start to its source, then to it.
func (c *progressChecker) regionPath(s string) (string, []edge) {
	var tail []edge
	for ; c.regionParent[s].source != ""; s = c.regionParent[s].source {
		tail = append([]edge{c.regionParent[s].edge}, tail...)
	}
	var head []edge
	for ; c.parent[s].source != ""; s = c.parent[s].source {
		head = append([]edge{c.parent[s].edge}, head...)
	}
	return s, append(head, tail...)
}

// violation is a found witness, or nil.
type violation struct {
	start string
	path  []edge
	loop  int
	why   string
}

func (c *progressChecker) verdict(kind ProgressKind, v *violation, cut bool) ProgressVerdict {
	switch {
	case v != nil:
		return ProgressVerdict{Outcome: CounterexampleFound, Witness: c.t.trace(v.start, v.path), Loop: v.loop,
			Explanation: fmt.Sprintf("%s: %s", kind, v.why)}
	case c.exceeded || cut:
		return ProgressVerdict{Outcome: LimitReached, Loop: -1, Explanation: fmt.Sprintf(
			"the limits %s allow %d units of work, and the check spent %d without finding a %s or ruling one out",
			c.limits.Name, c.limits.Search, c.budget.used, kind)}
	case c.open:
		return ProgressVerdict{Outcome: Unresolved, Loop: -1, Explanation: fmt.Sprintf(
			"the limits %s explore %d steps, and states beyond them are unexplored; no %s was found, "+
				"and a finite prefix that ends open is not a counterexample", c.limits.Name, c.limits.Steps, kind)}
	default:
		return ProgressVerdict{Outcome: VerifiedWithinLimits, Loop: -1}
	}
}

// regionParts is the region's strongly connected parts: each state's part, each part's members in
// region order, and whether a step stays inside it. Tarjan's algorithm completes a part after every
// part it reaches, so the parts are numbered sinks first.
type regionParts struct {
	of      map[string]int
	members [][]string
	cyclic  []bool
}

// parts divides the region into its strongly connected parts, or is nil when the budget cannot pay
// a unit per region state.
func (c *progressChecker) parts() *regionParts {
	if c.exceeded || !c.budget.spend(len(c.region)) {
		return nil
	}
	of := c.components()
	count := 0
	for _, id := range of {
		count = max(count, id+1)
	}
	p := &regionParts{of: of, members: make([][]string, count), cyclic: make([]bool, count)}
	for _, s := range c.region {
		id := of[s]
		p.members[id] = append(p.members[id], s)
		p.cyclic[id] = p.cyclic[id] || slices.ContainsFunc(c.inside(s), func(x edge) bool { return of[x.result.State] == id })
	}
	return p
}

// steps is the steps inside one part, by source.
func (c *progressChecker) steps(parts *regionParts, id int) map[string][]edge {
	steps := map[string][]edge{}
	for _, s := range parts.members[id] {
		for _, x := range c.inside(s) {
			if parts.of[x.result.State] == id {
				steps[s] = append(steps[s], x)
			}
		}
	}
	return steps
}

// cycle is a fair non-progress cycle: the first strongly connected part of the region, in region
// order, with a step inside it and in which every fair class enabled at all of its states is taken
// by a step inside it. A cycle through all of its states taking every step inside it is the
// fairest, so the part has a fair cycle exactly when this holds, and the witness tours every state
// and one step of each class fairness requires. Reading a part costs a unit per member; cut reports
// that the budget ran out first.
func (c *progressChecker) cycle(parts *regionParts) (*violation, bool) {
	if parts == nil {
		return nil, true
	}
	checked := map[int]bool{}
	for _, e := range c.region {
		id := parts.of[e]
		if checked[id] || !parts.cyclic[id] {
			continue
		}
		checked[id] = true
		members := parts.members[id]
		if !c.budget.spend(len(members)) {
			return nil, true
		}
		steps := c.steps(parts, id)
		required, fair := c.required(members, steps)
		if !fair {
			continue
		}
		tour, ok := c.tour(e, members, steps, required)
		if !ok {
			return nil, true
		}
		start, path := c.regionPath(e)
		return &violation{start: start, path: append(path, tour...), loop: len(path),
			why: fmt.Sprintf("the cycle through '%s' never reaches a state where %s holds, and it takes every "+
				"fair class enabled throughout it", e, c.p.Name)}, false
	}
	return nil, false
}

// required is one step inside a part of each fair class enabled at all of its members, and whether
// every such class has one.
func (c *progressChecker) required(members []string, steps map[string][]edge) ([]edge, bool) {
	var out []edge
	for _, f := range c.fair {
		if slices.ContainsFunc(members, func(s string) bool { return !c.enabled(s, f.class) }) {
			continue
		}
		x, ok := c.takes(members, steps, f.class)
		if !ok {
			return nil, false
		}
		out = append(out, x)
	}
	return out, true
}

func (c *progressChecker) enabled(s, class string) bool {
	return slices.ContainsFunc(c.t.RowsFrom(s), func(r Row) bool { return r.Action == class })
}

// takes is the first step inside a component of one class.
func (c *progressChecker) takes(members []string, steps map[string][]edge, class string) (edge, bool) {
	for _, s := range members {
		for _, x := range steps[s] {
			if c.t.Rows[c.t.rowIndex(x.row)].Action == class {
				return x, true
			}
		}
	}
	return edge{}, false
}

// tour is a cycle from e through every member and every required step, back to e, or false when
// the budget cannot pay a unit per state its searches visit and per step it takes.
func (c *progressChecker) tour(e string, members []string, steps map[string][]edge, required []edge) ([]edge, bool) {
	var path []edge
	at := e
	visited := map[string]bool{e: true}
	walk := func(xs []edge, ok bool) bool {
		if !ok || !c.budget.spend(len(xs)) {
			return false
		}
		for _, x := range xs {
			path = append(path, x)
			at = x.result.State
			visited[at] = true
		}
		return true
	}
	if !walk(steps[e][:1], true) {
		return nil, false
	}
	for _, s := range members {
		if !visited[s] && !walk(c.shortestInside(at, s, steps)) {
			return nil, false
		}
	}
	for _, x := range required {
		source := c.t.Rows[c.t.rowIndex(x.row)].Source
		if !walk(c.shortestInside(at, source, steps)) || !walk([]edge{x}, true) {
			return nil, false
		}
	}
	if !walk(c.shortestInside(at, e, steps)) {
		return nil, false
	}
	return path, true
}

// shortestInside is the shortest path from one part member to another through its steps, or false
// when the budget cannot pay a unit per state the search visits.
func (c *progressChecker) shortestInside(from, to string, steps map[string][]edge) ([]edge, bool) {
	parent := map[string]edgeFrom{from: {}}
	queue := []string{from}
	for i := 0; i < len(queue) && queue[i] != to; i++ {
		if !c.budget.spend(1) {
			return nil, false
		}
		for _, x := range steps[queue[i]] {
			if _, ok := parent[x.result.State]; !ok {
				parent[x.result.State] = edgeFrom{queue[i], x}
				queue = append(queue, x.result.State)
			}
		}
	}
	var path []edge
	for s := to; s != from; s = parent[s].source {
		path = append([]edge{parent[s].edge}, path...)
	}
	return path, true
}

// deadline is a path through the region from the first source, in exploration order, that takes
// Within steps: one exists from a state that reaches a cyclic part, whatever Within is, and from
// another exactly when its longest path is that long. Reading the parts costs a unit per region
// state and the witness a unit per step; a witness of Within steps the budget cannot pay for is a
// lasso, which unrolls to one. cut reports that the budget ran out first.
func (c *progressChecker) deadline(parts *regionParts) (*violation, bool) {
	if parts == nil || !c.budget.spend(len(c.region)) {
		return nil, true
	}
	endless := map[string]bool{}
	longest := map[string]int{}
	for id := range parts.members {
		for _, s := range parts.members[id] {
			endless[s] = parts.cyclic[id]
			for _, x := range c.inside(s) {
				u := x.result.State
				if parts.of[u] != id {
					endless[s] = endless[s] || endless[u]
					longest[s] = max(longest[s], longest[u]+1)
				}
			}
		}
	}
	for _, f := range c.sources {
		if !endless[f] && longest[f] < c.p.Within {
			continue
		}
		start, path := c.regionPath(f)
		var tail []edge
		loop := -1
		switch {
		case c.p.Within <= c.budget.left():
			tail = c.unroll(f, endless, longest)
		case endless[f]:
			var ok bool
			if tail, loop, ok = c.lasso(f, parts, endless); !ok {
				return nil, true
			}
			if loop >= 0 {
				loop += len(path)
			}
		default:
			return nil, true
		}
		return &violation{start: start, path: append(path, tail...), loop: loop,
			why: fmt.Sprintf("no state where %s holds follows '%s' within %d steps", c.p.Name, f, c.p.Within)}, false
	}
	return nil, false
}

// unroll takes Within steps from s through the region, each to a state a long enough path leaves.
func (c *progressChecker) unroll(s string, endless map[string]bool, longest map[string]int) []edge {
	var path []edge
	for k := c.p.Within; k > 0; k-- {
		c.budget.spend(1)
		steps := c.inside(s)
		x := steps[slices.IndexFunc(steps, func(x edge) bool {
			return endless[x.result.State] || longest[x.result.State] >= k-1
		})]
		path = append(path, x)
		s = x.result.State
	}
	return path
}

// lasso is a path from s to a cyclic part and once around a cycle of it, with the index of the
// state it returns to, or false when the budget cannot pay a unit per step and per state searched.
func (c *progressChecker) lasso(s string, parts *regionParts, endless map[string]bool) ([]edge, int, bool) {
	var path []edge
	for !parts.cyclic[parts.of[s]] {
		if !c.budget.spend(1) {
			return nil, 0, false
		}
		steps := c.inside(s)
		x := steps[slices.IndexFunc(steps, func(x edge) bool { return endless[x.result.State] })]
		path = append(path, x)
		s = x.result.State
	}
	loop := len(path)
	steps := c.steps(parts, parts.of[s])
	if !c.budget.spend(1) {
		return nil, 0, false
	}
	first := steps[s][0]
	back, ok := c.shortestInside(first.result.State, s, steps)
	if !ok || !c.budget.spend(len(back)) {
		return nil, 0, false
	}
	return append(append(path, first), back...), loop, true
}

// components numbers the region's strongly connected parts (Tarjan's algorithm over its steps).
func (c *progressChecker) components() map[string]int {
	index, low := map[string]int{}, map[string]int{}
	onStack := map[string]bool{}
	var stack []string
	component := map[string]int{}
	next, count := 0, 0
	var connect func(s string)
	connect = func(s string) {
		index[s], low[s] = next, next
		next++
		stack = append(stack, s)
		onStack[s] = true
		for _, x := range c.inside(s) {
			u := x.result.State
			if _, seen := index[u]; !seen {
				connect(u)
				low[s] = min(low[s], low[u])
			} else if onStack[u] {
				low[s] = min(low[s], index[u])
			}
		}
		if low[s] == index[s] {
			for {
				u := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[u] = false
				component[u] = count
				if u == s {
					break
				}
			}
			count++
		}
	}
	for _, s := range c.region {
		if _, seen := index[s]; !seen {
			connect(s)
		}
	}
	return component
}

// Replay checks a verdict's witness against the table and the claim: it is a path from a start
// that reaches a From state and after it no To state, and it ends as its kind says, at a state
// with no row, after Within steps or back at its Loop state, or back at its Loop state on a cycle
// that takes every fair class enabled throughout it.
func (p *Progress) Replay(t *Table, kind ProgressKind, v ProgressVerdict) error {
	decl := "progress " + p.Name
	c, err := newProgressChecker(t, p, Limits{})
	if err != nil {
		return err
	}
	path, err := t.replay(v.Witness)
	if err != nil {
		return errorf(decl, "%v", err)
	}
	if !slices.Contains(t.Starts, v.Witness.Initial.Value) {
		return errorf(decl, "the witness starts at '%s', which is not a start of %s", v.Witness.Initial.Value, t.Machine)
	}
	states := []string{v.Witness.Initial.Value}
	for _, e := range path {
		states = append(states, e.result.State)
	}
	source := -1
	for k := len(states) - 1; k >= 0; k-- {
		to, err := p.to(t, states[k])
		if err != nil {
			return err
		}
		if to {
			break
		}
		from, err := p.from(t, states[k])
		if err != nil {
			return err
		}
		if from {
			source = k
		}
	}
	if source < 0 {
		return errorf(decl, "the witness reaches no state where the claim starts that is followed by no state where it holds")
	}
	last := states[len(states)-1]
	switch kind {
	case DeadlockKind:
		if len(t.RowsFrom(last)) > 0 {
			return errorf(decl, "the witness ends at '%s', which has a step", last)
		}
	case DeadlineKind:
		lasso := v.Loop >= source && v.Loop < len(path) && states[v.Loop] == last
		if len(path)-source < p.Within && !lasso {
			return errorf(decl, "the witness takes %d steps after '%s', fewer than %d, and does not end in a cycle",
				len(path)-source, states[source], p.Within)
		}
	case CycleKind:
		if v.Loop < source || v.Loop >= len(path) || states[v.Loop] != last {
			return errorf(decl, "the witness does not return to its state %d", v.Loop)
		}
		return c.fairOn(states[v.Loop:len(states)-1], path[v.Loop:])
	default:
		return errorf(decl, "%s is no kind of progress violation", kind)
	}
	return nil
}

// fairOn checks that a cycle takes every fair class enabled at all of its states.
func (c *progressChecker) fairOn(states []string, cycle []edge) error {
	for _, f := range c.fair {
		if slices.ContainsFunc(states, func(s string) bool { return !c.enabled(s, f.class) }) {
			continue
		}
		if !slices.ContainsFunc(cycle, func(x edge) bool { return c.t.Rows[c.t.rowIndex(x.row)].Action == f.class }) {
			return errorf("progress "+c.p.Name, "%s stays enabled on the cycle and is never taken, which %s forbids",
				f.class, f.assumption)
		}
	}
	return nil
}

func (a ProgressAnswer) String() string {
	var parts []string
	for _, v := range []struct {
		kind ProgressKind
		v    ProgressVerdict
	}{{DeadlockKind, a.Deadlock}, {CycleKind, a.Cycle}, {DeadlineKind, a.Deadline}} {
		parts = append(parts, fmt.Sprintf("%s %s", v.kind, v.v.Outcome))
	}
	return fmt.Sprintf("%s under [%s]: %s", a.Claim, strings.Join(a.Assumptions, ", "), strings.Join(parts, ", "))
}
