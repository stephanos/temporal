package umpire

import (
	"cmp"
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
	ref.MapValue = func(state string) (any, error) {
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

// RefinementSpec is how one table reads as another: the state map, and optionally what the refined
// table sees and whether the refinement must cover every one of its starts. A nil SeesFact or
// SeesOutcome reads facts and outcomes as `Umpire.Command.deriveRefinement` does.
type RefinementSpec struct {
	// Declaration names the refinement in diagnostics; "<src> refines <dst>" when empty.
	Declaration string
	MapState    func(string) (string, error)
	SeesFact    func(fact string) bool
	SeesOutcome func(outcome string) bool
	CoverStarts bool
}

// RefinementFailure is the rule a refinement breaks.
type RefinementFailure string

const (
	// RefinementCatalog is an outcome the refined machine does not name.
	RefinementCatalog RefinementFailure = "catalog"
	// RefinementInitial is a start that reads as no refined start, or a refined start no start
	// reads as.
	RefinementInitial RefinementFailure = "initial"
	// RefinementUnmatched is a result no refined step carries and that is not a stutter.
	RefinementUnmatched RefinementFailure = "unmatched"
	// RefinementVisibleStutter is a stutter that records a fact or an outcome the refined machine
	// sees.
	RefinementVisibleStutter RefinementFailure = "visible-stutter"
)

// RefinementError is a refinement that does not hold. Witness is the shortest path of the refining
// table from one of its starts that ends in the failing result, or at the failing start; it is nil
// for a row no start reaches and for a failure of the refined table's own, whose ProductWitness is
// the refined start no start reads as.
type RefinementError struct {
	Declaration    string
	Message        string
	Kind           RefinementFailure
	Witness        *Trace
	ProductWitness *Trace
}

func (e *RefinementError) Error() string { return e.Declaration + ": " + e.Message }

// RefineTables checks that src refines dst under the rule `Umpire.Command.deriveRefinement` applies,
// narrowed by the spec's projection (SEMANTICS.md, Machines 6): every outcome reads as a product
// outcome of the same name; every start reads as a product start; and every row result is carried
// by a product row from the mapped source that reaches the mapped target with the same outcome and
// whose facts all appear among the result's facts, preferring the product action of the row's own
// name, or else the mapped states are equal and the result is a stutter.
func RefineTables(src, dst *Table, spec RefinementSpec) (*Refinement, error) {
	decl := cmp.Or(spec.Declaration, src.Machine+" refines "+dst.Machine)
	fail := func(kind RefinementFailure, witness *Trace, format string, args ...any) error {
		return &RefinementError{Declaration: decl, Message: fmt.Sprintf(format, args...), Kind: kind, Witness: witness}
	}
	ref := &Refinement{Machine: src.Machine, Product: dst.Machine, MapState: spec.MapState}
	if err := checkRefinementCatalogs(decl, src, dst, spec); err != nil {
		return nil, err
	}
	for _, row := range src.Rows {
		from, err := spec.MapState(row.Source)
		if err != nil {
			return nil, errorf(decl, "%v", err)
		}
		for _, res := range row.Results {
			to, err := spec.MapState(res.State)
			if err != nil {
				return nil, errorf(decl, "%v", err)
			}
			seen := spec.seen(res)
			carrier, ok := carrierOf(dst, row, res, from, to, seen)
			switch {
			case ok:
				ref.Rows = append(ref.Rows, RefinementRow{Key: row.Key, Product: &carrier})
			case from == to && len(seen.facts) == 0 && !seen.outcome:
				ref.Rows = append(ref.Rows, RefinementRow{Key: row.Key})
			case from == to:
				return nil, fail(RefinementVisibleStutter, src.witnessOf(row, res),
					"the row '%s' steps from '%s' to '%s', which both read as '%s' in %s, and %s; a stutter "+
						"changes nothing %s sees, and %s has no step that records it",
					row.Key, row.Source, res.State, from, dst.Machine, seen.describe(res, dst.Machine),
					dst.Machine, dst.Machine)
			default:
				return nil, fail(RefinementUnmatched, src.witnessOf(row, res),
					"the row '%s' steps from '%s' to '%s', which read as '%s' and '%s' "+
						"in %s; %s has no step from '%s' reaching '%s' with outcome '%s' and the facts [%s], "+
						"and the two are not equal, so the row is neither a step of %s nor a stutter",
					row.Key, row.Source, res.State, from, to, dst.Machine, dst.Machine, from, to,
					res.Outcome, strings.Join(res.Facts, ", "), dst.Machine)
			}
		}
	}
	return ref, nil
}

// seen is what the refined machine sees of one result under a projection.
type seen struct {
	facts   []string
	outcome bool
}

func (spec RefinementSpec) seen(res Result) seen {
	var out seen
	if spec.SeesFact != nil {
		for _, f := range res.Facts {
			if spec.SeesFact(f) {
				out.facts = append(out.facts, f)
			}
		}
	}
	out.outcome = spec.SeesOutcome != nil && spec.SeesOutcome(res.Outcome)
	return out
}

func (s seen) describe(res Result, product string) string {
	var parts []string
	if len(s.facts) > 0 {
		parts = append(parts, fmt.Sprintf("it records %s, which %s sees", strings.Join(s.facts, ", "), product))
	}
	if s.outcome {
		parts = append(parts, fmt.Sprintf("its outcome %s is one %s sees", res.Outcome, product))
	}
	return strings.Join(parts, ", and ")
}

// witnessOf is the shortest path from a start that ends in one result of a row, or nil when no
// start reaches the row.
func (t *Table) witnessOf(row Row, res Result) *Trace {
	start, path, ok := t.pathTo(row.Source)
	if !ok {
		return nil
	}
	return t.trace(start, append(path, edge{row.Key, res}))
}

// checkRefinementCatalogs checks that every outcome reads as a product outcome of the same name
// and every start as a product start.
func checkRefinementCatalogs(decl string, src, dst *Table, spec RefinementSpec) error {
	for _, o := range src.Outcomes {
		if !slices.Contains(dst.Outcomes, o) {
			return &RefinementError{Declaration: decl, Kind: RefinementCatalog, Message: fmt.Sprintf(
				"'%s' is an outcome of %s and no outcome of %s has that name", o, src.Machine, dst.Machine)}
		}
	}
	read := map[string]bool{}
	for _, s := range src.Starts {
		mapped, err := spec.MapState(s)
		if err != nil {
			return errorf(decl, "%v", err)
		}
		if !slices.Contains(dst.Starts, mapped) {
			return &RefinementError{Declaration: decl, Kind: RefinementInitial, Witness: src.trace(s, nil),
				Message: fmt.Sprintf("%s starts at '%s', which reads as '%s', and %s does not start there",
					src.Machine, s, mapped, dst.Machine)}
		}
		read[mapped] = true
	}
	if !spec.CoverStarts {
		return nil
	}
	for _, p := range dst.Starts {
		if !read[p] {
			return &RefinementError{Declaration: decl, Kind: RefinementInitial, ProductWitness: dst.trace(p, nil),
				Message: fmt.Sprintf("%s starts at '%s', which no start of %s reads as", dst.Machine, p, src.Machine)}
		}
	}
	return nil
}

// carrierOf is the product action whose step carries a row result: a product row from the mapped
// source reaching the mapped target with the same outcome and whose facts all appear among the
// result's facts, preferring the product action of the row's own name.
// Under a projection, the product result also records every fact of the result it sees.
func carrierOf(dst *Table, row Row, res Result, from, to string, seen seen) (string, bool) {
	facts := mapFacts(res.Facts, dst.Facts)
	visible := mapFacts(seen.facts, dst.Facts)
	if len(visible) < len(seen.facts) {
		return "", false
	}
	var carriers []string
	for _, c := range dst.RowsFrom(from) {
		if slices.ContainsFunc(c.Results, func(cr Result) bool {
			return cr.State == to && cr.Outcome == res.Outcome && allIn(cr.Facts, facts) && allIn(visible, cr.Facts)
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
