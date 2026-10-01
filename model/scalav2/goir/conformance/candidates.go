package conformance

import (
	"encoding/binary"
	"fmt"
)

// ordered is the observations of one instance of the machine with what each is ordered after: bit i
// of before[j] says observation i happened no later than observation j. An observation is ordered
// after its causal parents and after the earlier ordinals of its own source, and after whatever
// those are ordered after. Nothing else orders two observations: not the Run Events that carried
// them, and not a clock.
type ordered struct {
	evidence []*observation
	before   []uint64
}

// order relates one instance's observations. named finds any observation of the Run by its identity,
// so that a parent recorded for another instance is told apart from one not recorded at all.
func order(evidence []*observation, named map[string]*observation) (*ordered, error) {
	out := &ordered{evidence: evidence, before: make([]uint64, len(evidence))}
	index := make(map[string]int, len(evidence))
	for i, e := range evidence {
		index[e.identity] = i
	}
	for j, e := range evidence {
		for _, parent := range e.after {
			if i, here := index[parent]; here {
				out.before[j] |= 1 << i
			} else if other, recorded := named[parent]; recorded {
				return nil, &EvidenceError{Event: max(e.sequence, other.sequence), Message: fmt.Sprintf("evidence %s names %s, evidence of another operation, as its causal parent",
					e.identity, parent)}
			}
		}
		for i, earlier := range evidence {
			if earlier.source == e.source && earlier.ordinal < e.ordinal {
				out.before[j] |= 1 << i
			}
		}
	}
	out.close()
	for j, e := range evidence {
		if out.before[j]>>j&1 == 1 {
			return nil, &EvidenceError{Event: e.sequence, Message: fmt.Sprintf("evidence %s is ordered before itself", e.identity)}
		}
	}
	return out, nil
}

// close makes the order transitive: an observation is after whatever its predecessors are after.
func (o *ordered) close() {
	for changed := true; changed; {
		changed = false
		for j := range o.before {
			closure := o.before[j]
			for i := range o.before {
				if o.before[j]>>i&1 == 1 {
					closure |= o.before[i]
				}
			}
			if closure != o.before[j] {
				o.before[j], changed = closure, true
			}
		}
	}
}

// cell is one claim along one execution: what it says so far, and a monitor's state.
type cell struct {
	status status
	state  int32
}

// candidate is the executions that share an end: the observations they have explained, the state
// they leave the machine in, and what they say of each claim. Monitor states are part of it, so two
// histories that leave a monitor in different states stay two.
type candidate struct {
	explained uint64
	state     int32
	cells     []cell
}

func (c *candidate) key() string {
	key := make([]byte, 0, 12+5*len(c.cells))
	key = binary.LittleEndian.AppendUint64(key, c.explained)
	key = binary.LittleEndian.AppendUint32(key, uint32(c.state))
	for _, cell := range c.cells {
		key = append(key, byte(cell.status))
		key = binary.LittleEndian.AppendUint32(key, uint32(cell.state))
	}
	return string(key)
}

// survey is what one reading of one instance's evidence found: how many candidates explain all of
// it, what they say of each claim, and whether a hole row was in reach of any candidate tried.
type survey struct {
	candidates int
	holes      bool
	tallies    []tally
}

// regime says where an exploration reads the Run from.
type regime struct {
	// ended says the Run is over, so a monitor read at the end of a path is read.
	ended bool
	// event is the Run Event the reading was made at, for an error to name.
	event int64
}

// explore keeps every execution of the machine from its start that explains the observations in an
// order they admit, and counts what the ones that explain all of them say. It takes each step of
// each candidate once, charging spent, and stops with a LimitError at a ceiling rather than answer
// from a part.
func (p *plan) explore(o *ordered, how regime, spent *int) (*survey, error) {
	out := &survey{tallies: make([]tally, len(p.claims))}
	first := &candidate{state: p.start, cells: make([]cell, len(p.claims))}
	seen := map[string]struct{}{first.key(): {}}
	queue := []*candidate{first}
	all := uint64(1)<<len(o.evidence) - 1
	for len(queue) > 0 {
		at := queue[0]
		queue = queue[1:]
		if err := p.count(out, at, all, how); err != nil {
			return nil, err
		}
		reached, err := p.expand(o, at, how, spent, seen)
		if err != nil {
			return nil, err
		}
		queue = append(queue, reached...)
	}
	return out, nil
}

// expand takes every step of one candidate, each with every set of observations it can explain, and
// returns the candidates not seen before. Nothing is listed ahead of a ceiling: a set is built one
// observation at a time, each addition charged before it is made, and each step taken is charged
// before its candidate is.
func (p *plan) expand(o *ordered, at *candidate, how regime, spent *int, seen map[string]struct{}) ([]*candidate, error) {
	var reached []*candidate
	charge := func() error {
		if *spent >= p.limits.MaxWork {
			return &LimitError{Resource: "work", Ceiling: p.limits.MaxWork, Event: how.event}
		}
		*spent++
		return nil
	}
	for _, s := range p.steps[at.state] {
		// tried is the sets this step was taken with: two facts of one name reach a set two ways.
		tried := map[uint64]struct{}{}
		take := func(explains uint64) error {
			if _, again := tried[explains]; again || !o.admits(at.explained, explains) {
				return nil
			}
			if err := charge(); err != nil {
				return err
			}
			tried[explains] = struct{}{}
			next, err := p.taking(at, s, explains)
			if err != nil {
				return err
			}
			if _, known := seen[next.key()]; known {
				return nil
			}
			if len(seen) >= p.limits.MaxCandidates {
				return &LimitError{Resource: "candidates", Ceiling: p.limits.MaxCandidates, Event: how.event}
			}
			seen[next.key()] = struct{}{}
			reached = append(reached, next)
			return nil
		}
		if err := p.explaining(o, at.explained, s.facts, 0, charge, take); err != nil {
			return nil, err
		}
	}
	return reached, nil
}

// explaining visits the sets of unexplained observations a step's facts can be explained by: for each
// fact, no observation, which is the fact going unobserved, or one of that fact's name that is of the
// same attempt and delivery as the ones already in the set. The empty set is the step taken
// unobserved, which every step may be: no kind of evidence is declared to report every occurrence of
// its fact.
func (p *plan) explaining(o *ordered, explained uint64, facts []string, set uint64, charge func() error, visit func(uint64) error) error {
	if len(facts) == 0 {
		return visit(set)
	}
	if err := p.explaining(o, explained, facts[1:], set, charge, visit); err != nil {
		return err
	}
	for j, e := range o.evidence {
		if e.kind.records != facts[0] || (explained|set)>>j&1 == 1 || !o.together(set, e) {
			continue
		}
		if err := charge(); err != nil {
			return err
		}
		if err := p.explaining(o, explained, facts[1:], set|1<<j, charge, visit); err != nil {
			return err
		}
	}
	return nil
}

// together reports whether an observation may be of the same step as the ones in a set.
func (o *ordered) together(set uint64, e *observation) bool {
	for i, other := range o.evidence {
		if set>>i&1 == 1 && !e.sameStep(other) {
			return false
		}
	}
	return true
}

// admits reports whether a step may explain a set once these are explained: each observation of the
// set is ordered after nothing but what is explained already or explained with it.
func (o *ordered) admits(explained, set uint64) bool {
	for j := range o.evidence {
		if set>>j&1 == 1 && o.before[j]&^(explained|set) != 0 {
			return false
		}
	}
	return true
}

// count adds one candidate to a survey: the hole rows at its state, which taint every claim it has
// not violated, and, when it explains all the evidence, what it says of each claim.
func (p *plan) count(out *survey, at *candidate, all uint64, how regime) error {
	if len(p.holes[at.state]) > 0 {
		out.holes = true
		for i, cell := range at.cells {
			out.tallies[i].tainted = out.tallies[i].tainted || cell.status != violated
		}
	}
	if at.explained != all {
		return nil
	}
	out.candidates++
	for i, cell := range at.cells {
		final, err := p.claims[i].atEnd(cell, how.ended && p.ends[at.state])
		if err != nil {
			return err
		}
		out.tallies[i].count[final]++
	}
	return nil
}

// taking is the candidate one step of another leads to, explaining these observations.
func (p *plan) taking(at *candidate, s step, explains uint64) (*candidate, error) {
	next := &candidate{explained: at.explained | explains, state: s.target, cells: make([]cell, len(at.cells))}
	for i, cell := range at.cells {
		var err error
		if next.cells[i], err = p.claims[i].after(cell, s); err != nil {
			return nil, err
		}
	}
	return next, nil
}

func raise(old, read status) status { return max(old, read) }

// after is a claim's cell after one more step.
func (c *claim) after(at cell, s step) (cell, error) {
	if !c.monitor {
		switch c.property[s.index] {
		case holds:
			at.status = raise(at.status, held)
		case fails:
			at.status = violated
		case unreadable:
			at.status = raise(at.status, unknown)
		case malformed:
			return at, c.errs[[2]int{s.index, 0}]
		default:
		}
		return at, nil
	}
	if at.state == lost {
		return at, nil
	}
	if err, bad := c.errs[[2]int{s.index, int(at.state) + 1}]; bad {
		return at, err
	}
	if at.state = c.next[at.state][s.index]; at.state == lost {
		at.status = raise(at.status, unknown)
		return at, nil
	}
	switch c.read[s.index] {
	case holds:
		return c.verdict(at)
	case unreadable:
		at.status = raise(at.status, unknown)
	case malformed:
		return at, c.errs[[2]int{s.index, 0}]
	default:
	}
	return at, nil
}

// verdict reads a monitor where its evaluation point is.
func (c *claim) verdict(at cell) (cell, error) {
	switch c.violated[at.state] {
	case holds:
		at.status = violated
	case fails:
		at.status = raise(at.status, held)
	case unreadable:
		at.status = raise(at.status, unknown)
	default:
		return at, c.errs[[2]int{-1, int(at.state)}]
	}
	return at, nil
}

// atEnd is what an execution says of a claim where it stops: what it said after its last step, and,
// for a monitor read at the end of a path that ends where the machine may end, its verdict there.
func (c *claim) atEnd(at cell, ended bool) (status, error) {
	if !c.atEnds || !ended {
		return at.status, nil
	}
	if at.state == lost {
		return raise(at.status, unknown), nil
	}
	at, err := c.verdict(at)
	return at.status, err
}
