package checker

import (
	"fmt"
	"slices"
)

type refinementDecl[S any] struct {
	product Model
	mapKey  func(S) string
	mapAny  func(S) any
	stepOf  func(state any, outcome string, facts []string) (any, error)
}

// Refines declares that m refines product through the state map f. The generic method ties the
// map's result type to the product's state type, so a map into another machine's state is a
// compile error.
func (m *Machine[S, O, F]) Refines[PS, PO, PF any](product *Machine[PS, PO, PF], f func(S) PS) *Machine[S, O, F] {
	m.refinement = &refinementDecl[S]{
		product: product,
		mapKey:  func(s S) string { return KeyOf(f(s)) },
		mapAny:  func(s S) any { return f(s) },
		stepOf: func(state any, outcome string, facts []string) (any, error) {
			outcomes, err := DomainOf[PO]()
			if err != nil {
				return nil, err
			}
			productFacts, err := DomainOf[PF]()
			if err != nil {
				return nil, err
			}
			step := Step[PS, PO, PF]{State: state.(PS)}
			found := false
			for _, o := range outcomes {
				if KeyOf(o) == outcome {
					step.Outcome, found = o, true
				}
			}
			if !found {
				return nil, fmt.Errorf("outcome %s has no product outcome of that name", outcome)
			}
			keys := make([]string, len(productFacts))
			for i, pf := range productFacts {
				keys[i] = KeyOf(pf)
			}
			for _, fact := range facts {
				if k, ok := sameNamedKey(keys, fact); ok {
					i := slices.Index(keys, k)
					step.Facts = append(step.Facts, productFacts[i])
				}
			}
			return step, nil
		},
	}
	return m
}

// Refinement checks the declared refinement under the rule `Umpire.Command.deriveRefinement`
// applies: every outcome reads as a product outcome of the same name; every start reads as a
// product start; and every row result is carried by a product row from the mapped source that
// reaches the mapped target with the same outcome and whose facts all appear among the result's
// facts, preferring the product action of the row's own name, or else the mapped states are equal
// and the result is a stutter.
func (m *Machine[S, O, F]) Refinement() (*Refinement, error) {
	return m.checkRefinement(m.coverStarts)
}

// checkRefinement checks the declared refinement, requiring every product start to be read by a
// start when coverStarts is set.
func (m *Machine[S, O, F]) checkRefinement(coverStarts bool) (*Refinement, error) {
	if m.refinement == nil {
		return nil, errorf(m.name, "the machine declares no refinement")
	}
	src, err := m.Table()
	if err != nil {
		return nil, err
	}
	dst, err := m.refinement.product.Table()
	if err != nil {
		return nil, err
	}
	spec := RefinementSpec{CoverStarts: coverStarts,
		MapState: func(state string) (string, error) {
			v, ok := src.stateValue[state].(S)
			if !ok {
				return "", fmt.Errorf("unknown state %s", state)
			}
			return m.refinement.mapKey(v), nil
		}}
	if m.visibleFact != nil {
		if spec.SeesFact, err = seesKey(m.visibleFact); err != nil {
			return nil, errorf(m.name, "fact type: %v", err)
		}
	}
	if m.visibleOutcome != nil {
		if spec.SeesOutcome, err = seesKey(m.visibleOutcome); err != nil {
			return nil, errorf(m.name, "outcome type: %v", err)
		}
	}
	ref, err := RefineTables(src, dst, spec)
	if err != nil {
		return nil, err
	}
	ref.stepOfFn = m.refinement.stepOf
	ref.mapValue = func(state string) (any, error) {
		v, ok := src.stateValue[state].(S)
		if !ok {
			return nil, fmt.Errorf("unknown state %s", state)
		}
		return m.refinement.mapAny(v), nil
	}
	return ref, nil
}

func (m *Machine[S, O, F]) refines() Model {
	if m.refinement == nil {
		return nil
	}
	return m.refinement.product
}

// seesKey reads a predicate over a finite type as one over its keys.
func seesKey[T any](sees func(T) bool) (func(string) bool, error) {
	members, err := DomainOf[T]()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, v := range members {
		seen[KeyOf(v)] = sees(v)
	}
	return func(key string) bool { return seen[key] }, nil
}

// Visible names the facts of this machine the refined machine sees: a carrying product step
// records each one the result records, and a stutter records none (the IR's `visible`).
func (m *Machine[S, O, F]) Visible(sees func(F) bool) *Machine[S, O, F] {
	m.visibleFact = sees
	return m
}

// VisibleOutcomes names the outcomes of this machine the refined machine sees: a stutter's outcome
// is none of them (the IR's `visible_outcomes`).
func (m *Machine[S, O, F]) VisibleOutcomes(sees func(O) bool) *Machine[S, O, F] {
	m.visibleOutcome = sees
	return m
}

// CoversStarts requires every start of the refined machine to be read by a start of this one, so
// the refinement accounts for every initial state the refined machine admits.
func (m *Machine[S, O, F]) CoversStarts() *Machine[S, O, F] {
	m.coverStarts = true
	return m
}
