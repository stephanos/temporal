package engine

import (
	"fmt"
	"math"
	"slices"
	"strings"
)

type compositionMember struct {
	field    string
	model    Model
	replaces Model
}

type compositionSync struct {
	name string
	refs [2][2]string // member, action
}

type composedAction struct {
	key   string
	moves []memberMove
}

type memberMove struct {
	member int
	action string
}

// composing is the part of building a composition that reads keys only.
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
		b.remember(key, p.parts)
		out[i] = Result{Outcome: p.outcome, State: key, Facts: p.facts, Step: ComposedStep{Parts: p.parts}}
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
	t := &Table{Machine: b.name, Owner: b.owner, Family: b.family, parts: map[string][]string{}}
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

// actionName is the action a class key belongs to: the key before its first "-".
func actionName(key string) string {
	name, _, _ := strings.Cut(key, "-")
	return name
}
