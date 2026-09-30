package umpire

import (
	"fmt"
	"slices"
	"strings"
)

type refinementDecl[S any] struct {
	product Model
	mapKey  func(S) string
	mapAny  func(S) any
	stepOf  func(state any, outcome string, facts []string) (any, error)
}

// Refinement is a checked `refines:` between a machine and the machine it refines: for every row
// result, the product action whose step carries it, or "" for a stutter.
type Refinement struct {
	Machine string
	Product string
	Rows    []RefinementRow
	// MapState reads a refining state key as the product state it stands for.
	MapState func(string) (string, error)
	// MapValue reads a refining state key as the typed product state.
	MapValue func(string) (any, error)
	stepOfFn func(state any, outcome string, facts []string) (any, error)
}

// RefinementRow pairs one row result with its product action, "" for a stutter.
type RefinementRow struct {
	Key     string  `json:"key"`
	Product *string `json:"product"`
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
	mapKey := func(state string) (string, error) {
		v, ok := src.stateValue[state].(S)
		if !ok {
			return "", fmt.Errorf("unknown state %s", state)
		}
		return m.refinement.mapKey(v), nil
	}
	ref := &Refinement{Machine: m.name, Product: dst.Machine, MapState: mapKey, stepOfFn: m.refinement.stepOf,
		MapValue: func(state string) (any, error) {
			v, ok := src.stateValue[state].(S)
			if !ok {
				return nil, fmt.Errorf("unknown state %s", state)
			}
			return m.refinement.mapAny(v), nil
		}}
	decl := m.name + " refines " + dst.Machine
	if err := checkRefinementCatalogs(decl, src, dst, mapKey); err != nil {
		return nil, err
	}
	for _, row := range src.Rows {
		from, _ := mapKey(row.Source)
		for _, res := range row.Results {
			to, _ := mapKey(res.State)
			carrier, ok := carrierOf(dst, row, res, from, to)
			switch {
			case ok:
				ref.Rows = append(ref.Rows, RefinementRow{Key: row.Key, Product: &carrier})
			case from == to:
				ref.Rows = append(ref.Rows, RefinementRow{Key: row.Key})
			default:
				return nil, errorf(decl, "the row '%s' steps from '%s' to '%s', which read as '%s' and '%s' "+
					"in %s; %s has no step from '%s' reaching '%s' with outcome '%s' and the facts [%s], "+
					"and the two are not equal, so the row is neither a step of %s nor a stutter",
					row.Key, row.Source, res.State, from, to, dst.Machine, dst.Machine, from, to,
					res.Outcome, strings.Join(res.Facts, ", "), dst.Machine)
			}
		}
	}
	return ref, nil
}

// checkRefinementCatalogs checks that every outcome reads as a product outcome of the same name
// and every start as a product start.
func checkRefinementCatalogs(decl string, src, dst *Table, mapKey func(string) (string, error)) error {
	for _, o := range src.Outcomes {
		if !slices.Contains(dst.Outcomes, o) {
			return errorf(decl, "'%s' is an outcome of %s and no outcome of %s has that name",
				o, src.Machine, dst.Machine)
		}
	}
	for _, s := range src.Starts {
		mapped, _ := mapKey(s)
		if !slices.Contains(dst.Starts, mapped) {
			return errorf(decl, "%s starts at '%s', which reads as '%s', and %s does not start there",
				src.Machine, s, mapped, dst.Machine)
		}
	}
	return nil
}

// carrierOf is the product action whose step carries a row result: a product row from the mapped
// source reaching the mapped target with the same outcome and whose facts all appear among the
// result's facts, preferring the product action of the row's own name.
func carrierOf(dst *Table, row Row, res Result, from, to string) (string, bool) {
	facts := mapFacts(res.Facts, dst.Facts)
	var carriers []string
	for _, c := range dst.RowsFrom(from) {
		if slices.ContainsFunc(c.Results, func(cr Result) bool {
			return cr.State == to && cr.Outcome == res.Outcome && allIn(cr.Facts, facts)
		}) {
			carriers = append(carriers, c.Action)
		}
	}
	if preferred, ok := sameNamedKey(dst.Actions, row.Action); ok && slices.Contains(carriers, preferred) {
		return preferred, true
	}
	if len(carriers) > 0 {
		return carriers[0], true
	}
	return "", false
}

func (r *Refinement) stepOf(state any, outcome string, facts []string) (any, error) {
	return r.stepOfFn(state, outcome, facts)
}

// sameNamedKey is the product key a key names by default: the same key, or the constructor it
// applies (`Umpire.Command.sameNamedKey`).
func sameNamedKey(product []string, key string) (string, bool) {
	if slices.Contains(product, key) {
		return key, true
	}
	constructor, _, _ := strings.Cut(key, "-")
	if slices.Contains(product, constructor) {
		return constructor, true
	}
	return "", false
}

// mapFacts reads refining facts as the product facts of the same name; a fact the product does
// not name is one it does not see.
func mapFacts(facts, product []string) []string {
	var out []string
	for _, f := range facts {
		if k, ok := sameNamedKey(product, f); ok {
			out = append(out, k)
		}
	}
	return out
}

func allIn(xs, ys []string) bool {
	for _, x := range xs {
		if !slices.Contains(ys, x) {
			return false
		}
	}
	return true
}
