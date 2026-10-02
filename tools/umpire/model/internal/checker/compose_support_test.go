package checker

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"
)

// Composition builds one Model from machines of different entities, for a claim no one of them
// can state: the Go form of the Lean `compose` command. Its state S is a structure with one field
// per member, tagged with the member's name. A `Sync` pairs member actions into one step; a member
// action no Sync names steps its member alone.
//
// The table follows `Umpire.Command.Compose`: only the rows reachable from the starts, a composed
// state keyed by its members' state keys joined by "_" in field order, a member's own action keyed
// "<member>_<class>", a synchronized action keyed by its Sync name, outcomes and facts keyed
// "<member>_<key>", and Definition IDs owned by "compose-<name>".
type Composition[S any] struct {
	family  Family
	name    string
	members []compositionMember
	syncs   []compositionSync
	ends    func(S) bool
	err     error
	names   claimNames

	once  sync.Once
	table *Table
	terr  error
}

// Compose starts a composition declaration.
func Compose[S any](family Family, name string) *Composition[S] {
	return &Composition[S]{family: family, name: name}
}

// Name is the composition's declared name.
func (c *Composition[S]) Name() string { return c.name }

// Member adds a member machine under the state field tagged field.
func (c *Composition[S]) Member(field string, m Model) *Composition[S] {
	c.members = append(c.members, compositionMember{field: field, model: m})
	return c
}

// Replaces says the member at field stands in for replaced within this composition: a detailed
// provider in place of an opaque one. The member's machine must declare a refinement of replaced
// that holds and reads every start of it, and a check of the composition relies on none of
// replaced's assumptions.
func (c *Composition[S]) Replaces(field string, replaced Model) *Composition[S] {
	i := slices.IndexFunc(c.members, func(m compositionMember) bool { return m.field == field })
	if i < 0 {
		if c.err == nil {
			c.err = errorf("compose-"+c.name, "the composition replaces %s at %s, and has no member %s",
				replaced.Name(), field, field)
		}
		return c
	}
	c.members[i].replaces = replaced
	return c
}

// replacing is a member machine that may stand in for the machine it refines.
type replacing interface {
	refines() Model
	checkRefinement(coverStarts bool) (*Refinement, error)
}

// Sync pairs two members' actions into one step named name. Each ref is "<member>.<action>"; a
// malformed ref is reported when the table is built.
func (c *Composition[S]) Sync(name, first, second string) *Composition[S] {
	var refs [2][2]string
	for i, r := range []string{first, second} {
		member, action, ok := strings.Cut(r, ".")
		if !ok && c.err == nil {
			c.err = errorf("compose-"+c.name, "sync reference %q is not <member>.<action>", r)
		}
		refs[i] = [2]string{member, action}
	}
	c.syncs = append(c.syncs, compositionSync{name, refs})
	return c
}

// Ends says which composed states the composition may end in.
func (c *Composition[S]) Ends(end func(S) bool) *Composition[S] { c.ends = end; return c }

// stateKey is a composed state's key: its members' state keys in member order, joined by "_".
func (c *Composition[S]) stateKey(s S) string {
	v := reflect.ValueOf(s)
	var parts []string
	for _, m := range c.members {
		for j := range v.NumField() {
			if f := v.Type().Field(j); f.Tag.Get("umpire") == m.field {
				parts = append(parts, keyOf(v.Field(j), f.Type))
			}
		}
	}
	return strings.Join(parts, "_")
}

// Table enumerates the reachable composition.
func (c *Composition[S]) Table() (*Table, error) {
	c.once.Do(func() { c.table, c.terr = c.build() })
	return c.table, c.terr
}

// composer holds what building one composition needs: the member tables, the composed actions and
// the member state keys each composed key stands for.
type composer[S any] struct {
	c *Composition[S]
	*composing
	fieldIndex []int
}

func (c *Composition[S]) build() (*Table, error) {
	if c.err != nil {
		return nil, c.err
	}
	b := &composer[S]{c: c, composing: &composing{family: c.family, name: c.name, owner: "compose-" + c.name,
		members: c.members, syncs: c.syncs, split: map[string][]string{}}}
	if err := b.resolveMembers(); err != nil {
		return nil, err
	}
	if err := b.checkReplacements(); err != nil {
		return nil, err
	}
	if err := b.collectActions(); err != nil {
		return nil, err
	}
	starts := b.starts()
	reached := b.explore(starts)
	if b.collision != nil {
		return nil, b.collision
	}
	return b.assemble(starts, reached)
}

// checkReplacements checks each replacing member's refinement of what it replaces.
func (b *composer[S]) checkReplacements() error {
	for _, m := range b.members {
		if m.replaces == nil {
			continue
		}
		r, ok := m.model.(replacing)
		if !ok || r.refines() != m.replaces {
			return errorf(b.owner, "the member %s replaces %s, and %s does not refine it",
				m.field, m.replaces.Name(), m.model.Name())
		}
		if _, err := r.checkRefinement(true); err != nil {
			return err
		}
	}
	return nil
}

func (b *composer[S]) resolveMembers() error {
	st := reflect.TypeFor[S]()
	if st.Kind() != reflect.Struct {
		return errorf(b.owner, "the composed state %s is not a structure", st)
	}
	for _, m := range b.members {
		t, err := m.model.Table()
		if err != nil {
			return err
		}
		index := -1
		for j := range st.NumField() {
			if st.Field(j).Tag.Get("umpire") == m.field {
				index = j
			}
		}
		if index < 0 {
			return errorf(b.owner, "no field of %s is tagged umpire:%q", st, m.field)
		}
		b.tables = append(b.tables, t)
		b.fieldIndex = append(b.fieldIndex, index)
	}
	return nil
}

// assemble lays the reachable composition out as a table: states sorted by key, rows states-major
// over the sorted actions.
func (b *composer[S]) assemble(starts [][]string, reached map[string]bool) (*Table, error) {
	t := b.catalogs(reached)
	for _, k := range t.States {
		v, err := b.value(b.split[k])
		if err != nil {
			return nil, err
		}
		t.stateValue[k] = v
	}
	for _, s := range t.States {
		rows, err := b.rowsFrom(t, s)
		if err != nil {
			return nil, err
		}
		t.Rows = append(t.Rows, rows...)
	}
	if err := b.startsAndAssumptions(t, starts); err != nil {
		return nil, err
	}
	for _, s := range t.States {
		v, ok := t.stateValue[s].(S)
		if !ok {
			return nil, errorf(b.owner, "composed state %s has no typed value", s)
		}
		if b.c.ends != nil && b.c.ends(v) {
			t.Ends = append(t.Ends, s)
		}
	}
	t.finish()
	return t, nil
}

// rowsFrom is every composed row from one state, in action order, with typed steps a Property
// reads.
func (b *composer[S]) rowsFrom(t *Table, s string) ([]Row, error) {
	rows := b.keyRowsFrom(t, s)
	for _, row := range rows {
		results := row.Results
		for i := range results {
			v, ok := t.stateValue[results[i].State].(S)
			if !ok {
				return nil, errorf(b.owner, "composed state %s has no typed value", results[i].State)
			}
			results[i].Step = Step[S, string, string]{Outcome: results[i].Outcome, State: v, Facts: results[i].Facts}
		}
	}
	return rows, nil
}

// value builds the typed composed state from its members' typed states.
func (b *composer[S]) value(parts []string) (S, error) {
	var zero S
	v := reflect.New(reflect.TypeFor[S]()).Elem()
	for i, p := range parts {
		member, ok := b.tables[i].stateValue[p]
		if !ok {
			return zero, fmt.Errorf("%s: member state %s is unknown", b.owner, p)
		}
		v.Field(b.fieldIndex[i]).Set(reflect.ValueOf(member))
	}
	s, ok := v.Interface().(S)
	if !ok {
		return zero, fmt.Errorf("%s: composed state is not a %T", b.owner, zero)
	}
	return s, nil
}
