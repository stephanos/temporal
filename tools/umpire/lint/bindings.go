package lint

import (
	"slices"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/interp"
)

// A realization's coverage of its machine's classes (fn-133 R6). A class pattern a binding writes is
// exact: `start(scheduleToStart := expires)` is the one class whose omitted inputs are at their
// domain's first value, and it binds no other class of the action.
//
// A realizable path is one a Case can drive: every step of it is of a class some command or activity
// start of the realization performs, or of a system action, which no command performs. A binding of
// a class no realizable path reaches a state enabling is unreachable: no Case ever carries it. A
// class of an action the realization performs that a realizable path can take, but no binding
// performs, is uncovered: a Query whose path takes it has no Case.

// realizable is the classes the realization performs, by key, and the states of its machine a
// realizable path reaches.
func (m *Model) realizable(r *umpirespb.Realization, machine *interp.Machine) (bound map[string]bool, reached map[string]bool, err error) {
	bound = map[string]bool{}
	for _, s := range r.GetScripts() {
		for _, c := range s.GetActivity().GetStarts() {
			key, err := m.classKey(c)
			if err != nil {
				return nil, nil, err
			}
			bound[key] = true
		}
		for _, item := range s.GetItems() {
			for _, p := range item.GetPerforms() {
				key, err := m.classKey(p.GetStep())
				if err != nil {
					return nil, nil, err
				}
				bound[key] = true
			}
		}
	}
	systemic := map[string]bool{}
	for _, c := range machine.Classes {
		systemic[c.Key] = system(c.Action)
	}
	reached = map[string]bool{}
	frontier := slices.Clone(machine.Table.Starts)
	for len(frontier) > 0 {
		s := frontier[0]
		frontier = frontier[1:]
		if reached[s] {
			continue
		}
		reached[s] = true
		for _, row := range machine.Table.Rows {
			if row.Source != s || !(bound[row.Action] || systemic[row.Action]) {
				continue
			}
			for _, result := range row.Results {
				frontier = append(frontier, result.State)
			}
		}
	}
	return bound, reached, nil
}

// unreachableBindings is each class a `perform` or an `onPath` of a realization binds that no state a
// realizable path reaches enables.
func unreachableBindings(m *Model) ([]Tally, error) {
	t := tally(UnreachableBinding)
	for _, r := range m.IR.GetRealizations() {
		machine := m.Machines[r.GetMachine()]
		if machine == nil {
			continue
		}
		_, reached, err := m.realizable(r, machine)
		if err != nil {
			return nil, err
		}
		enabled := map[string]bool{}
		for _, row := range machine.Table.Rows {
			if reached[row.Source] {
				enabled[row.Action] = true
			}
		}
		binding := func(c *umpirespb.ActionClass, at *umpirespb.Position) error {
			key, err := m.classKey(c)
			if err != nil {
				return err
			}
			t.add(r.GetName(), enabled[key], key, at,
				"realization %s binds %s, which no state a realizable path of %s reaches enables: no Case carries the binding",
				r.GetName(), key, r.GetMachine())
			return nil
		}
		for _, s := range r.GetScripts() {
			for _, item := range s.GetItems() {
				for _, p := range item.GetPerforms() {
					if err := binding(p.GetStep(), p.GetPosition()); err != nil {
						return nil, err
					}
				}
				for _, c := range item.GetWhen() {
					if err := binding(c, item.GetPosition()); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	return t.list(), nil
}

// uncoveredClasses is each class of an action a realization performs, not a system action's, that a
// state a realizable path reaches enables and no binding of the realization performs.
func uncoveredClasses(m *Model) ([]Tally, error) {
	t := tally(UncoveredClass)
	for _, r := range m.IR.GetRealizations() {
		machine := m.Machines[r.GetMachine()]
		if machine == nil {
			continue
		}
		bound, reached, err := m.realizable(r, machine)
		if err != nil {
			return nil, err
		}
		performed := map[string]bool{}
		for _, c := range machine.Classes {
			if bound[c.Key] {
				performed[c.Action.GetId()] = true
			}
		}
		enabled := map[string]bool{}
		for _, row := range machine.Table.Rows {
			if reached[row.Source] {
				enabled[row.Action] = true
			}
		}
		for _, c := range machine.Classes {
			if !performed[c.Action.GetId()] || system(c.Action) || !enabled[c.Key] {
				continue
			}
			t.add(r.GetName(), bound[c.Key], c.Key, r.GetPosition(),
				"realization %s performs %s, and no binding performs its class %s, which a realizable path can take: "+
					"bind it, or accept it with the reason it is unrealizable", r.GetName(), c.Action.GetName(), c.Key)
		}
	}
	return t.list(), nil
}
