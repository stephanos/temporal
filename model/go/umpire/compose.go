package umpire

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

type compositionMember struct {
	field string
	model Model
}

type compositionSync struct {
	name string
	refs [2][2]string // member, action
}

// Compose starts a composition declaration.
func Compose[S any](family Family, name string) *Composition[S] {
	return &Composition[S]{family: family, name: name}
}

// Name is the composition's declared name.
func (c *Composition[S]) Name() string { return c.name }

// Member adds a member machine under the state field tagged field.
func (c *Composition[S]) Member(field string, m Model) *Composition[S] {
	c.members = append(c.members, compositionMember{field, m})
	return c
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

// Own is the composed key of a member's own action class.
func (c *Composition[S]) Own(member string, class Class) string { return member + "_" + class.Key() }

// Synced is the composed key of a synchronized step whose first member takes class.
func (c *Composition[S]) Synced(name string, class Class) string {
	_, suffix, _ := strings.Cut(class.Key(), class.Decl.Name)
	return name + suffix
}

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

type composedAction struct {
	key   string
	moves []memberMove
}

type memberMove struct {
	member int
	action string
}

// composer holds what building one composition needs: the member tables, the composed actions and
// the member state keys each composed key stands for.
type composer[S any] struct {
	c          *Composition[S]
	owner      string
	tables     []*Table
	fieldIndex []int
	actions    []composedAction
	split      map[string][]string
}

func (c *Composition[S]) build() (*Table, error) {
	if c.err != nil {
		return nil, c.err
	}
	b := &composer[S]{c: c, owner: "compose-" + c.name, split: map[string][]string{}}
	if err := b.resolveMembers(); err != nil {
		return nil, err
	}
	if err := b.collectActions(); err != nil {
		return nil, err
	}
	start := make([]string, len(b.tables))
	for i, t := range b.tables {
		start[i] = t.Starts[0]
	}
	reached := b.explore(start)
	return b.assemble(start, reached)
}

func (b *composer[S]) resolveMembers() error {
	st := reflect.TypeFor[S]()
	if st.Kind() != reflect.Struct {
		return errorf(b.owner, "the composed state %s is not a structure", st)
	}
	for _, m := range b.c.members {
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

func (b *composer[S]) memberIndex(field string) int {
	return slices.IndexFunc(b.c.members, func(m compositionMember) bool { return m.field == field })
}

// collectActions lists every synchronized pair of classes, then every member class no Sync names,
// sorted by composed key.
func (b *composer[S]) collectActions() error {
	synced := map[[2]string]bool{}
	for _, s := range b.c.syncs {
		first, second := b.memberIndex(s.refs[0][0]), b.memberIndex(s.refs[1][0])
		if first < 0 || second < 0 {
			return errorf(b.owner, "sync %s names a member the composition does not have", s.name)
		}
		synced[s.refs[0]], synced[s.refs[1]] = true, true
		for _, x := range classesOf(b.tables[first], s.refs[0][1]) {
			for _, y := range classesOf(b.tables[second], s.refs[1][1]) {
				key := s.name + strings.TrimPrefix(x, s.refs[0][1]) + strings.TrimPrefix(y, s.refs[1][1])
				b.actions = append(b.actions, composedAction{key, []memberMove{{first, x}, {second, y}}})
			}
		}
	}
	for i, t := range b.tables {
		field := b.c.members[i].field
		for _, a := range t.Actions {
			if !synced[[2]string{field, actionName(a)}] {
				b.actions = append(b.actions, composedAction{field + "_" + a, []memberMove{{i, a}}})
			}
		}
	}
	slices.SortFunc(b.actions, func(x, y composedAction) int { return compareStrings(x.key, y.key) })
	return nil
}

func classesOf(t *Table, action string) []string {
	var out []string
	for _, a := range t.Actions {
		if actionName(a) == action {
			out = append(out, a)
		}
	}
	return out
}

// partialStep is a composed result under construction, one member move at a time.
type partialStep struct {
	parts   []string
	outcome string
	facts   []string
}

// stepFrom is the composed results of one action from one composed state: every member moves by
// one of its rows, and a synchronized step takes the product of its members' results. The
// outcome is the first member's; the facts are every member's, in member order.
func (b *composer[S]) stepFrom(parts []string, a composedAction) []Result {
	partial := []partialStep{{parts: slices.Clone(parts), facts: []string{}}}
	for k, mv := range a.moves {
		row, ok := rowOf(b.tables[mv.member], parts[mv.member], mv.action)
		if !ok {
			return nil
		}
		field := b.c.members[mv.member].field
		var next []partialStep
		for _, p := range partial {
			for _, res := range row.Results {
				n := partialStep{parts: slices.Clone(p.parts), outcome: p.outcome, facts: slices.Clone(p.facts)}
				n.parts[mv.member] = res.State
				if k == 0 {
					n.outcome = field + "_" + res.Outcome
				}
				for _, f := range res.Facts {
					n.facts = append(n.facts, field+"_"+f)
				}
				next = append(next, n)
			}
		}
		partial = next
	}
	out := make([]Result, len(partial))
	for i, p := range partial {
		key := strings.Join(p.parts, "_")
		b.split[key] = p.parts
		out[i] = Result{Outcome: p.outcome, State: key, Facts: p.facts}
	}
	return out
}

func rowOf(t *Table, state, action string) (Row, bool) {
	for _, r := range t.RowsFrom(state) {
		if r.Action == action {
			return r, true
		}
	}
	return Row{}, false
}

// explore collects every composed state reachable from the start.
func (b *composer[S]) explore(start []string) map[string]bool {
	key := strings.Join(start, "_")
	b.split[key] = start
	seen := map[string]bool{key: true}
	queue := [][]string{start}
	for len(queue) > 0 {
		parts := queue[0]
		queue = queue[1:]
		for _, a := range b.actions {
			for _, r := range b.stepFrom(parts, a) {
				if !seen[r.State] {
					seen[r.State] = true
					queue = append(queue, b.split[r.State])
				}
			}
		}
	}
	return seen
}

// assemble lays the reachable composition out as a table: states sorted by key, rows states-major
// over the sorted actions.
func (b *composer[S]) assemble(start []string, reached map[string]bool) (*Table, error) {
	t := &Table{Machine: b.c.name, Owner: b.owner, Family: b.c.family, stateValue: map[string]any{}}
	for k := range reached {
		t.States = append(t.States, k)
	}
	slices.SortFunc(t.States, compareStrings)
	for _, a := range b.actions {
		t.Actions = append(t.Actions, a.key)
	}
	t.Facts = []string{}
	for i, m := range b.tables {
		field := b.c.members[i].field
		for _, o := range m.Outcomes {
			t.Outcomes = append(t.Outcomes, field+"_"+o)
		}
	}
	for i, m := range b.tables {
		field := b.c.members[i].field
		for _, f := range m.Facts {
			t.Facts = append(t.Facts, field+"_"+f)
		}
		t.StateFields = append(t.StateFields, composedFields(field, m)...)
	}
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
	t.Starts = []string{strings.Join(start, "_")}
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
	var rows []Row
	for _, a := range b.actions {
		results := b.stepFrom(b.split[s], a)
		if len(results) == 0 {
			continue
		}
		for i := range results {
			v, ok := t.stateValue[results[i].State].(S)
			if !ok {
				return nil, errorf(b.owner, "composed state %s has no typed value", results[i].State)
			}
			results[i].Step = Step[S, string, string]{Outcome: results[i].Outcome, State: v, Facts: results[i].Facts}
		}
		rows = append(rows, Row{Key: rowKey(s, a.key), Source: s, Action: a.key, Results: results})
	}
	return rows, nil
}

// composedFields names a member's state fields in the composition: the member's name for a
// one-field member, and "<member>_<field>" otherwise. A refining member's field for the machine it
// refines is a reading of its state, not a field of it, so the composition does not carry it.
func composedFields(field string, m *Table) []string {
	if len(m.StateFields) == 1 {
		return []string{field}
	}
	var out []string
	for _, f := range m.StateFields {
		if f != m.refinedField {
			out = append(out, field+"_"+f)
		}
	}
	return out
}

// value builds the typed composed state from its members' typed states.
func (b *composer[S]) value(parts []string) (S, error) {
	var zero S
	v := reflect.New(reflect.TypeFor[S]()).Elem()
	for i, p := range parts {
		member, ok := b.tables[i].StateValue(p)
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

// actionName is the action a class key belongs to: the key before its first "-".
func actionName(key string) string {
	name, _, _ := strings.Cut(key, "-")
	return name
}
