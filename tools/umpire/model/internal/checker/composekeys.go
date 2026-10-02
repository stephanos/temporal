package checker

import (
	"fmt"
	"math"
	"slices"
)

// ComposeMember is one member of a composition of tables: the table, under the name its composed
// keys carry.
type ComposeMember struct {
	Field string
	Table *Table
	// Replaces, when set, is the table this member stands in for within the composition: a detailed
	// provider in place of an opaque one. Refinement reads Table as it, and must hold and read every
	// start of it; the composition then relies on none of its assumptions.
	Replaces   *Table
	Refinement RefinementSpec
}

// ComposeSync pairs two members' actions into one step named Name.
type ComposeSync struct {
	Name         string
	FirstMember  string
	FirstAction  string
	SecondMember string
	SecondAction string
}

// ComposeCeiling bounds the work of composing: the composed states kept, the evaluations of one
// composed state and one composed action, and the results those evaluations produce. The composed
// actions count against Evaluations, since every one is evaluated at a start. A product, of the
// members' starts, of two members' synchronized classes or of their results, is counted by its size
// before any of it is built.
type ComposeCeiling struct {
	States      int64
	Evaluations int64
	Results     int64
}

// ComposeSpec declares a composition of tables computed outside this package, as Compose declares
// one of typed machines.
type ComposeSpec struct {
	Family  Family
	Name    string
	Members []ComposeMember
	Syncs   []ComposeSync
	// Ends says which composed states the composition may end in, reading a state by its key and its
	// member state keys in member order. Nil ends in none.
	Ends func(key string, parts []string) (bool, error)
	// Ceiling is required: each of its bounds is above 0.
	Ceiling ComposeCeiling
}

// ComposedStep is what a result of a table ComposeTables builds carries as its Step: the member
// state keys of the state it reaches, and the member rows and results it takes, in move order.
type ComposedStep struct {
	Parts []string
	Moves []MemberMove
}

// MemberMove is one member's part of a composed step: the member's index, the key of the row it
// takes, and the index of the result within that row.
type MemberMove struct {
	Member int
	Row    string
	Result int
}

// ComposeLimitError is a composition that does not fit its ceiling: the resource, "states",
// "evaluations" or "results", the ceiling, and a count the composition needs at least. Overflow reports that the
// count is past what Needed can hold. No table is built: one cut at the ceiling would read as the
// whole composition.
type ComposeLimitError struct {
	Composition string
	Resource    string
	Ceiling     int64
	Needed      int64
	Overflow    bool
}

func (e *ComposeLimitError) Error() string {
	needed := fmt.Sprintf("at least %d", e.Needed)
	if e.Overflow {
		needed = fmt.Sprintf("more than %d", e.Needed)
	}
	return fmt.Sprintf("compose-%s: the ceiling allows %d %s, and the composition needs %s",
		e.Composition, e.Ceiling, e.Resource, needed)
}

// ComposeTables builds the reachable composition of member tables as Compose builds one of typed
// machines: the same starts, actions, results, keys, layout, assumptions and replacement rule. A
// state and an action a member leaves unknown are an unknown pair of the composition, unless
// another move of the step is disabled, which disables it. The error is a *ComposeLimitError when the
// composition does not fit the ceiling, a *RefinementError when a replacing member does not refine
// what it replaces, and an *Error otherwise.
func ComposeTables(spec ComposeSpec) (*Table, error) {
	b := &composing{family: spec.Family, name: spec.Name, owner: "compose-" + spec.Name,
		split: map[string][]string{}, ceiling: &spec.Ceiling}
	if spec.Ceiling.States <= 0 || spec.Ceiling.Evaluations <= 0 || spec.Ceiling.Results <= 0 {
		return nil, errorf(b.owner, "the ceiling allows %d states, %d evaluations and %d results, and a composition "+
			"is bounded by one above 0 of each", spec.Ceiling.States, spec.Ceiling.Evaluations, spec.Ceiling.Results)
	}
	if err := b.resolveTables(spec.Members); err != nil {
		return nil, err
	}
	for _, m := range spec.Members {
		if m.Replaces == nil {
			continue
		}
		refinement := m.Refinement
		refinement.CoverStarts = true
		if _, err := RefineTables(m.Table, m.Replaces, refinement); err != nil {
			return nil, err
		}
	}
	for _, s := range spec.Syncs {
		b.syncs = append(b.syncs, compositionSync{s.Name,
			[2][2]string{{s.FirstMember, s.FirstAction}, {s.SecondMember, s.SecondAction}}})
	}
	if err := b.collectActions(); err != nil {
		return nil, err
	}
	if err := b.checkStartProduct(); err != nil {
		return nil, err
	}
	starts := b.starts()
	reached := b.explore(starts)
	if b.collision != nil {
		return nil, b.collision
	}
	if b.limit != nil {
		return nil, b.limit
	}
	return b.assembleKeys(starts, reached, spec.Ends)
}

// resolveTables checks the members and lists their tables: each under its own field, and a
// replacing one with the map that reads it as what it replaces.
func (b *composing) resolveTables(members []ComposeMember) error {
	for _, m := range members {
		switch {
		case m.Field == "":
			return errorf(b.owner, "a member has no field")
		case b.memberIndex(m.Field) >= 0:
			return errorf(b.owner, "two members are at %s", m.Field)
		case m.Table == nil:
			return errorf(b.owner, "the member %s has no table", m.Field)
		case m.Replaces == nil && m.Refinement.MapState != nil:
			return errorf(b.owner, "the member %s names a refinement, and replaces nothing", m.Field)
		case m.Replaces != nil && m.Refinement.MapState == nil:
			return errorf(b.owner, "the member %s replaces %s, and names no map that reads %s as it",
				m.Field, m.Replaces.Machine, m.Table.Machine)
		default:
		}
		member := compositionMember{field: m.Field, model: m.Table.Model()}
		if m.Replaces != nil {
			member.replaces = m.Replaces.Model()
		}
		t, err := member.model.Table()
		if err != nil {
			return err
		}
		if len(t.Starts) == 0 {
			return errorf(b.owner, "the member %s has no start", m.Field)
		}
		b.members = append(b.members, member)
		b.tables = append(b.tables, t)
	}
	return nil
}

// checkStartProduct refuses a product of the members' starts above the state ceiling before the
// starts are listed, so a product too large to hold is never built.
func (b *composing) checkStartProduct() error {
	product := int64(1)
	for _, t := range b.tables {
		n := int64(len(t.Starts))
		if product > math.MaxInt64/n {
			return &ComposeLimitError{Composition: b.name, Resource: "states", Ceiling: b.ceiling.States,
				Needed: math.MaxInt64, Overflow: true}
		}
		product *= n
		if product > b.ceiling.States {
			return &ComposeLimitError{Composition: b.name, Resource: "states", Ceiling: b.ceiling.States, Needed: product}
		}
	}
	return nil
}

// assembleKeys lays the reachable composition out as a table of keys: states sorted by key, rows
// states-major over the sorted actions.
func (b *composing) assembleKeys(starts [][]string, reached map[string]bool,
	ends func(key string, parts []string) (bool, error)) (*Table, error) {
	t := b.catalogs(reached)
	t.classes, t.decls = map[string]Class{}, map[string]*ActionDecl{}
	for _, s := range t.States {
		t.stateValue[s] = s
		t.Rows = append(t.Rows, b.keyRowsFrom(t, s)...)
	}
	if err := b.startsAndAssumptions(t, starts); err != nil {
		return nil, err
	}
	for _, s := range t.States {
		if ends == nil {
			break
		}
		end, err := ends(s, slices.Clone(t.parts[s]))
		if err != nil {
			return nil, wrapError(b.owner, err)
		}
		if end {
			t.Ends = append(t.Ends, s)
		}
	}
	t.finish()
	return t, nil
}
