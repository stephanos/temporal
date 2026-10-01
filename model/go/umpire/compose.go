package umpire

import (
	"fmt"
	"math"
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
	field    string
	model    Model
	replaces Model
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
	c *Composition[S]
	*composing
	fieldIndex []int
}

// composing is the part of building a composition that reads keys only, which a typed composition
// and one of key-only tables (ComposeTables) share, so the two cannot drift apart.
type composing struct {
	family    Family
	name      string
	owner     string
	members   []compositionMember
	syncs     []compositionSync
	tables    []*Table
	actions   []composedAction
	split     map[string][]string
	collision error
	// ceiling bounds the composed states and the evaluations of a state and an action; nil bounds
	// neither. limit is the first ceiling the build went past.
	ceiling     *ComposeCeiling
	evaluations int64
	limit       *ComposeLimitError
	// classes and results count the composed actions listed and the results evaluations produced.
	classes, results int64
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

// starts is every composed start: the product of the members' starts in member order, the last
// member varying fastest.
func (b *composing) starts() [][]string {
	out := [][]string{{}}
	for _, t := range b.tables {
		var next [][]string
		for _, prefix := range out {
			for _, s := range t.Starts {
				next = append(next, append(slices.Clone(prefix), s))
			}
		}
		out = next
	}
	return out
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

// assumptions is every member's assumptions in member order, each fair class read as the composed
// classes that step it; two members' assumptions of one name are one, fair for both members' classes.
// A replacing member's are its own, also one the machine it replaces names: that machine is no member,
// so none of its assumptions is read here.
func (b *composing) assumptions() ([]Assumption, error) {
	var out []Assumption
	for i := range b.members {
		for _, a := range b.tables[i].Assumptions {
			fair, err := b.composedFair(i, a)
			if err != nil {
				return nil, err
			}
			out = mergeAssumption(out, Assumption{Name: a.Name, Fair: fair})
		}
	}
	return out, nil
}

// mergeAssumption adds an assumption, or its fair classes to the one of its name.
func mergeAssumption(out []Assumption, a Assumption) []Assumption {
	k := slices.IndexFunc(out, func(x Assumption) bool { return x.Name == a.Name })
	if k < 0 {
		return append(out, Assumption{Name: a.Name, Fair: slices.Clone(a.Fair)})
	}
	for _, f := range a.Fair {
		if !slices.Contains(out[k].Fair, f) {
			out[k].Fair = append(out[k].Fair, f)
		}
	}
	return out
}

// composedFair is the composed classes a member's fair classes, or actions, step: its own classes
// keyed "<member>_<class>" and the synchronized steps it takes part in.
func (b *composing) composedFair(member int, a Assumption) ([]string, error) {
	var out []string
	for _, f := range a.Fair {
		found := false
		for _, ca := range b.actions {
			for _, mv := range ca.moves {
				if mv.member == member && (mv.action == f || actionName(mv.action) == f) {
					found = true
					if !slices.Contains(out, ca.key) {
						out = append(out, ca.key)
					}
				}
			}
		}
		if !found {
			machine := b.tables[member].Machine
			return nil, errorf(b.owner, "the assumption %s of %s makes %s fair, which is no action of %s",
				a.Name, machine, f, machine)
		}
	}
	return out, nil
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

func (b *composing) memberIndex(field string) int {
	return slices.IndexFunc(b.members, func(m compositionMember) bool { return m.field == field })
}

// collectActions lists every synchronized pair of classes, then every member class no Sync names,
// sorted by composed key.
func (b *composing) collectActions() error {
	synced := map[[2]string]bool{}
	for _, s := range b.syncs {
		first, second := b.memberIndex(s.refs[0][0]), b.memberIndex(s.refs[1][0])
		if first < 0 || second < 0 {
			return errorf(b.owner, "sync %s names a member the composition does not have", s.name)
		}
		synced[s.refs[0]], synced[s.refs[1]] = true, true
		for i, member := range []int{first, second} {
			if len(classesOf(b.tables[member], s.refs[i][1])) == 0 {
				return errorf(b.owner, "sync %s names %s.%s, and %s has no action %s",
					s.name, s.refs[i][0], s.refs[i][1], b.tables[member].Machine, s.refs[i][1])
			}
		}
		firsts, seconds := classesOf(b.tables[first], s.refs[0][1]), classesOf(b.tables[second], s.refs[1][1])
		if !b.admitsClasses(int64(len(firsts)), int64(len(seconds))) {
			return b.limit
		}
		for _, x := range firsts {
			for _, y := range seconds {
				key := s.name + strings.TrimPrefix(x, s.refs[0][1]) + strings.TrimPrefix(y, s.refs[1][1])
				b.actions = append(b.actions, composedAction{key, []memberMove{{first, x}, {second, y}}})
			}
		}
	}
	if err := b.collectOwnActions(synced); err != nil {
		return err
	}
	slices.SortFunc(b.actions, func(x, y composedAction) int { return compareStrings(x.key, y.key) })
	return nil
}

// collectOwnActions lists every member class no Sync names, each counted before it is listed.
func (b *composing) collectOwnActions(synced map[[2]string]bool) error {
	for i, t := range b.tables {
		field := b.members[i].field
		for _, a := range t.Actions {
			if synced[[2]string{field, actionName(a)}] {
				continue
			}
			if !b.admitsClasses(1, 1) {
				return b.limit
			}
			b.actions = append(b.actions, composedAction{field + "_" + a, []memberMove{{i, a}}})
		}
	}
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
	moves   []MemberMove
}

// memberUnknown is the member's unknown pair that leaves a composed pair unknown.
type memberUnknown struct {
	member int
	pair   UnknownPair
}

// stepFrom is the composed results of one action from one composed state: every member moves by
// one of its rows, and a synchronized step takes the product of its members' results. The
// outcome is the first member's; the facts are every member's, in member order.
func (b *composing) stepFrom(parts []string, a composedAction) []Result {
	status, rows, _ := b.movesFrom(parts, a)
	if status != pairEnabled {
		return nil
	}
	partial := []partialStep{{parts: slices.Clone(parts), facts: []string{}}}
	for k, mv := range a.moves {
		row := rows[k]
		field := b.members[mv.member].field
		var next []partialStep
		for _, p := range partial {
			for j, res := range row.Results {
				n := partialStep{parts: slices.Clone(p.parts), outcome: p.outcome, facts: slices.Clone(p.facts),
					moves: append(slices.Clone(p.moves), MemberMove{Member: mv.member, Row: row.Key, Result: j})}
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
		b.remember(key, p.parts)
		out[i] = Result{Outcome: p.outcome, State: key, Facts: p.facts, Step: ComposedStep{Parts: p.parts, Moves: p.moves}}
	}
	return out
}

// movesFrom reads every move of a composed action from a composed state, before any result is
// expanded, so a step that does not happen builds no part of a product. A step needs every move:
// one disabled move disables it whatever else is unknown, and an unknown move leaves it unknown
// whatever else is enabled. It returns the rows of an enabled step, and the first unknown move of an
// unknown one.
func (b *composing) movesFrom(parts []string, a composedAction) (pairStatus, []Row, *memberUnknown) {
	status, rows := pairEnabled, make([]Row, len(a.moves))
	var unknown *memberUnknown
	for k, mv := range a.moves {
		move, row, pair := b.tables[mv.member].pair(parts[mv.member], mv.action)
		switch move {
		case pairDisabled:
			return pairDisabled, nil, nil
		case pairUnknown:
			if unknown == nil {
				unknown = &memberUnknown{mv.member, pair}
			}
		case pairEnabled:
			rows[k] = row
		default:
		}
		status = min(status, move)
	}
	return status, rows, unknown
}

// remember notes the member states a composed key stands for. The key joins member keys that may
// hold "_" themselves, so two different member states can share it; the first such pair is the
// collision build reports, since exploring one would silently stand for both.
func (b *composing) remember(key string, parts []string) {
	prev, ok := b.split[key]
	if !ok {
		b.split[key] = parts
		return
	}
	if !slices.Equal(prev, parts) && b.collision == nil {
		b.collision = errorf(b.owner, "the member states %v and %v are both keyed '%s', "+
			"so the composed key does not tell them apart", prev, parts, key)
	}
}

// explore collects every composed state reachable from the start.
// Under a ceiling, a state is counted before it is kept, an evaluation before it is made and its
// results before they are built, and the first one past the ceiling stops the exploration with its
// limit set: what was collected by then is not the composition.
func (b *composing) explore(starts [][]string) map[string]bool {
	seen := map[string]bool{}
	var queue [][]string
	for _, start := range starts {
		key := strings.Join(start, "_")
		b.remember(key, start)
		if !seen[key] {
			if !b.admitsState(len(seen)) {
				return seen
			}
			seen[key] = true
			queue = append(queue, start)
		}
	}
	for len(queue) > 0 {
		parts := queue[0]
		queue = queue[1:]
		for _, a := range b.actions {
			if !b.admitsEvaluation() || !b.admitsResults(parts, a) {
				return seen
			}
			for _, r := range b.stepFrom(parts, a) {
				if !seen[r.State] {
					if !b.admitsState(len(seen)) {
						return seen
					}
					seen[r.State] = true
					queue = append(queue, b.split[r.State])
				}
			}
		}
	}
	return seen
}

// catalogs starts the composed table: its states sorted by key with the member states each stands
// for, its actions, and its outcomes, facts and state fields prefixed per member.
func (b *composing) catalogs(reached map[string]bool) *Table {
	t := &Table{Machine: b.name, Owner: b.owner, Family: b.family, stateValue: map[string]any{},
		parts: map[string][]string{}}
	for k := range reached {
		t.States = append(t.States, k)
		t.parts[k] = b.split[k]
	}
	slices.SortFunc(t.States, compareStrings)
	for _, a := range b.actions {
		t.Actions = append(t.Actions, a.key)
	}
	t.Facts = []string{}
	for i, m := range b.tables {
		field := b.members[i].field
		for _, o := range m.Outcomes {
			t.Outcomes = append(t.Outcomes, field+"_"+o)
		}
	}
	for i, m := range b.tables {
		field := b.members[i].field
		for _, f := range m.Facts {
			t.Facts = append(t.Facts, field+"_"+f)
		}
		t.StateFields = append(t.StateFields, composedFields(field, m)...)
	}
	return t
}

// keyRowsFrom is every composed row from one state, in action order, each result with the member
// rows it takes; a pair a member leaves unknown joins the table's unknown pairs instead.
func (b *composing) keyRowsFrom(t *Table, s string) []Row {
	var rows []Row
	for _, a := range b.actions {
		if status, _, u := b.movesFrom(b.split[s], a); status == pairUnknown {
			t.Unknown = append(t.Unknown, b.composedUnknown(s, a, u))
			continue
		}
		results := b.stepFrom(b.split[s], a)
		if len(results) == 0 {
			continue
		}
		rows = append(rows, Row{Key: rowKey(s, a.key), Source: s, Action: a.key, Results: results})
	}
	return rows
}

// composedUnknown is the composed pair a member's unknown pair leaves unknown, with the member's
// cause kept.
func (b *composing) composedUnknown(s string, a composedAction, u *memberUnknown) UnknownPair {
	cause := &Error{Declaration: b.owner, Message: fmt.Sprintf("the pair '%s' of the member %s is unknown",
		u.pair.Row, b.members[u.member].field), cause: u.pair.Cause}
	if u.pair.Cause != nil {
		cause.Message += ": " + u.pair.Cause.Error()
	}
	return UnknownPair{Row: rowKey(s, a.key), Source: s, Action: a.key, Cause: cause}
}

// startsAndAssumptions lists the composed starts, each once in start order, and the assumptions.
func (b *composing) startsAndAssumptions(t *Table, starts [][]string) error {
	for _, start := range starts {
		if key := strings.Join(start, "_"); !slices.Contains(t.Starts, key) {
			t.Starts = append(t.Starts, key)
		}
	}
	assumptions, err := b.assumptions()
	if err != nil {
		return err
	}
	t.Assumptions = assumptions
	return nil
}

// admitsState reports whether one more composed state fits under the ceiling, of which kept are
// already kept.
func (b *composing) admitsState(kept int) bool {
	if b.ceiling == nil || int64(kept) < b.ceiling.States {
		return true
	}
	b.limit = &ComposeLimitError{Composition: b.name, Resource: "states", Ceiling: b.ceiling.States,
		Needed: int64(kept) + 1}
	return false
}

// admitsEvaluation counts one more evaluation of a state and an action, if the ceiling has room.
func (b *composing) admitsEvaluation() bool {
	if b.ceiling == nil {
		return true
	}
	if b.evaluations >= b.ceiling.Evaluations {
		b.limit = &ComposeLimitError{Composition: b.name, Resource: "evaluations", Ceiling: b.ceiling.Evaluations,
			Needed: b.evaluations + 1}
		return false
	}
	b.evaluations++
	return true
}

// admitsClasses counts the composed actions a product of two members' classes lists, by its size
// and before any is listed, if the ceiling has room. Every composed action is evaluated at a start,
// so more of them than the evaluations allowed cannot fit.
func (b *composing) admitsClasses(firsts, seconds int64) bool {
	if b.ceiling == nil {
		return true
	}
	var overflow bool
	b.classes, overflow = addProduct(b.classes, firsts, seconds)
	if !overflow && b.classes <= b.ceiling.Evaluations {
		return true
	}
	b.limit = &ComposeLimitError{Composition: b.name, Resource: "evaluations", Ceiling: b.ceiling.Evaluations,
		Needed: b.classes, Overflow: overflow}
	return false
}

// admitsResults counts the results one composed action produces from one composed state, the
// product of its moves' results, by its size and before any is built, if the ceiling has room.
func (b *composing) admitsResults(parts []string, a composedAction) bool {
	if b.ceiling == nil {
		return true
	}
	status, rows, _ := b.movesFrom(parts, a)
	if status != pairEnabled {
		return true
	}
	size, overflow := int64(1), false
	for _, row := range rows {
		var over bool
		size, over = addProduct(0, size, int64(len(row.Results)))
		overflow = overflow || over
	}
	var over bool
	b.results, over = addProduct(b.results, size, 1)
	if overflow = overflow || over; !overflow && b.results <= b.ceiling.Results {
		return true
	}
	b.limit = &ComposeLimitError{Composition: b.name, Resource: "results", Ceiling: b.ceiling.Results,
		Needed: b.results, Overflow: overflow}
	return false
}

// addProduct is sum + x*y for counts, none below 0, and whether it is past what a count holds, in
// which case it is the largest count.
func addProduct(sum, x, y int64) (int64, bool) {
	if x != 0 && y > (math.MaxInt64-sum)/x {
		return math.MaxInt64, true
	}
	return sum + x*y, false
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
