package umpire

import (
	"errors"
	"fmt"
	"slices"
)

// RequirementKind is what one lowered clause fixes.
type RequirementKind int

const (
	StateRequirement RequirementKind = iota
	OutcomeRequirement
	FactRequirement
)

// Requirement is one clause a predicate fixes: its label, what it fixes, and the key it fixes.
type Requirement struct {
	Label string
	Kind  RequirementKind
	Value string
}

// Group is the clauses one trigger carries: an action key for a same-step claim.
type Group struct {
	Trigger      string
	Requirements []Requirement
}

// Lower enumerates a same-step Property's predicate into the clauses that say the same thing over
// the machine's own table, as `Umpire.Command.enumerateSameStep` does: over the steps the trigger
// admits, the predicate fixes a value when every accepted step carries it, the domain has another
// value, and the predicate rejects every accepted step with it changed. The fixed values must carry
// the predicate exactly. A predicate the clause language cannot carry is refused with the reason.
//
// Only the whole-state, outcome and fact readings are ported: the one-field reading a composed
// claim needs is not, because no realized Query in this repository reads a composition.
//
// A Property over a table's keys lowers the same way, its predicate asked about results by their
// keys. Such a predicate may fail to answer; a step it cannot be read on is not a step it rejects,
// so the Property is then not lowered and the error is the predicate's.
func (p *PropertyDecl) Lower() ([]Group, error) {
	if p.IsTransition() {
		return nil, errorf("property "+p.Name, "a transition claim is searched and verified, never realized")
	}
	t, err := p.Machine.Table()
	if err != nil {
		return nil, err
	}
	if t.alter.state == nil {
		return nil, errorf("property "+p.Name, "a claim of a composition is searched and verified, never realized")
	}
	ask := &asking{p: p}
	var groups []Group
	for _, action := range t.Actions {
		if !p.Triggers(action) {
			continue
		}
		var results []Result
		for _, r := range t.Rows {
			if r.Action == action {
				results = append(results, r.Results...)
			}
		}
		trigger := "`" + action + "`"
		if len(results) == 0 {
			return nil, errorf("property "+p.Name, "no step of this machine is admitted at %s, so the "+
				"predicate has nothing to hold on", trigger)
		}
		reqs, err := ask.fixedRequirements(t, results)
		if ask.unread != nil {
			return nil, wrapError("property "+p.Name, ask.unread)
		}
		if err != nil {
			return nil, errorf("property "+p.Name, "%s at %s", err, trigger)
		}
		if len(reqs) == 0 {
			return nil, errorf("property "+p.Name, "the predicate holds on every step of this machine at %s "+
				"and fixes no state, outcome or fact, so it claims nothing", trigger)
		}
		groups = append(groups, Group{Trigger: action, Requirements: reqs})
	}
	return groups, nil
}

// asking asks a Property about steps while it is lowered. A typed predicate always answers. One over
// a table's keys may fail: the first failure is kept as unread, and nothing asked after it counts.
type asking struct {
	p      *PropertyDecl
	unread error
}

func (p *asking) accepts(step Result) bool {
	if p.p.keyHolds == nil {
		return p.p.holds(step.Step)
	}
	if p.unread != nil {
		return false
	}
	held, err := p.p.keyHolds(step)
	if err != nil {
		p.unread = err
		return false
	}
	return held
}

// fixedRequirements is `fixedRequirements` without the field reading.
func (p *asking) fixedRequirements(t *Table, results []Result) ([]Requirement, error) {
	var accepted []Result
	for _, r := range results {
		if p.accepts(r) {
			accepted = append(accepted, r)
		}
	}
	if len(accepted) == 0 {
		return nil, errors.New("the predicate holds on no step of this machine")
	}
	first := accepted[0]
	alter := t.alterer()
	state := first.State
	stateFixed := slices.ContainsFunc(t.States, func(s string) bool { return s != state })
	outcome := first.Outcome
	outcomeFixed := slices.ContainsFunc(t.Outcomes, func(o string) bool { return o != outcome })
	for _, step := range accepted {
		stateFixed = stateFixed && step.State == state && !slices.ContainsFunc(t.States, func(other string) bool {
			return other != state && p.accepts(alter.state(step, other))
		})
		outcomeFixed = outcomeFixed && step.Outcome == outcome && !slices.ContainsFunc(t.Outcomes, func(other string) bool {
			return other != outcome && p.accepts(alter.outcome(step, other))
		})
	}
	var facts []string
	for _, f := range slices.Compact(slices.Clone(first.Facts)) {
		fixed := true
		for _, step := range accepted {
			fixed = fixed && slices.Contains(step.Facts, f) && !p.accepts(alter.without(step, f))
		}
		if fixed {
			facts = append(facts, f)
		}
	}
	carried := func(step Result) bool {
		return (!stateFixed || step.State == state) && (!outcomeFixed || step.Outcome == outcome) &&
			allIn(facts, step.Facts)
	}
	for _, step := range results {
		if carried(step) != p.accepts(step) {
			return nil, fmt.Errorf("the predicate is not a conjunction of one state, one outcome and facts: "+
				"the clauses it fixes cannot tell the step to %s with outcome %s and facts %v apart from "+
				"the steps it accepts", step.State, step.Outcome, step.Facts)
		}
	}
	var out []Requirement
	if stateFixed {
		out = append(out, Requirement{Label: "state-" + state, Kind: StateRequirement, Value: state})
	}
	if outcomeFixed {
		out = append(out, Requirement{Label: "outcome-" + outcome, Kind: OutcomeRequirement, Value: outcome})
	}
	for _, f := range facts {
		out = append(out, Requirement{Label: "fact-" + f, Kind: FactRequirement, Value: f})
	}
	return out, nil
}

// alterer rebuilds a result with its state, outcome or one fact changed, typed, so the predicate
// can be asked about it. The table supplies the typed values behind each key.
type alterer struct {
	state   func(Result, string) Result
	outcome func(Result, string) Result
	without func(Result, string) Result
}

func (t *Table) alterer() alterer {
	return t.alter
}

// keyAlterer rebuilds a result of a table over keys with its state, outcome or one fact changed. The
// result is its keys alone: it carries no step, since no step was taken to it.
func keyAlterer() alterer {
	keyed := func(r Result) Result {
		return Result{Outcome: r.Outcome, State: r.State, Facts: slices.Clone(r.Facts), Because: r.Because}
	}
	return alterer{
		state: func(r Result, key string) Result {
			out := keyed(r)
			out.State = key
			return out
		},
		outcome: func(r Result, key string) Result {
			out := keyed(r)
			out.Outcome = key
			return out
		},
		without: func(r Result, key string) Result {
			out := keyed(r)
			out.Facts = slices.DeleteFunc(out.Facts, func(f string) bool { return f == key })
			return out
		},
	}
}
