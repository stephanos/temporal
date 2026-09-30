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
func (p *PropertyDecl) Lower() ([]Group, error) {
	if p.IsTransition() {
		return nil, errorf("property "+p.Name, "a transition claim is searched and verified, never realized")
	}
	t, err := p.Machine.Table()
	if err != nil {
		return nil, err
	}
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
		reqs, err := p.fixedRequirements(t, results)
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

func (p *PropertyDecl) accepts(step Result) bool { return p.holds(step.Step) }

// fixedRequirements is `fixedRequirements` without the field reading.
func (p *PropertyDecl) fixedRequirements(t *Table, results []Result) ([]Requirement, error) {
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
