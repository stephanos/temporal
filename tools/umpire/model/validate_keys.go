package model

import (
	"errors"
	"fmt"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
)

// schedules evaluates every Scenario's start, which must be a state of its machine or composition,
// and checks a composition's keys against its classes. It runs once the rest of the Model admits, so
// what it evaluates is well formed.
func (v *validator) schedules(m *umpirespb.Model, composed map[string]classKeys) {
	for _, s := range m.GetScenarios() {
		at, owner := s.GetPosition(), s.GetMachine()+"."+s.GetName()
		state := v.machines[s.GetMachine()].GetStateType()
		c := v.compositions[s.GetMachine()]
		if c != nil {
			state = c.GetStateType()
		}
		if s.GetStart() != nil {
			start, err := v.in.Eval(s.GetStart())
			switch {
			case err != nil:
				v.report(at, "%s: its start: %v", owner, err)
			case !v.in.conforms(start, named(state)):
				v.report(at, "%s starts at %s, which is no %s", owner, start.Key(), state)
			default:
			}
		}
		if c == nil || len(s.GetKeys()) == 0 {
			continue
		}
		keys, ok := v.readable(at, owner, composed[c.GetName()])
		if !ok {
			continue
		}
		for _, k := range s.GetKeys() {
			if _, ok := keys[k]; !ok {
				v.report(at, "%s: %s has no class %s", owner, c.GetName(), k)
			}
		}
	}
}

// readable is a composition's class keys for a claim that names some of them, and whether they are
// known. Keys past a ceiling were not listed, which is the claim's error at its own position; two
// classes that share a key were reported where the composition is declared.
func (v *validator) readable(at *umpirespb.Position, owner string, keys classKeys) (map[string]string, bool) {
	var limit *LimitError
	if errors.As(keys.err, &limit) {
		v.errs = append(v.errs, fmt.Errorf("%s: %s: its classes: %w", where(at), owner, keys.err))
	}
	return keys.owners, keys.err == nil
}

// selectors checks the steps a composition's Property is about against the composition's classes:
// one class by its key, and an action by the classes its name begins, a sync's name or
// `<field>_<action>` for a member's own. A member's action a sync takes has no class of its own, so
// neither names it. It runs once the rest of the Model admits, as schedules does.
func (v *validator) selectors(m *umpirespb.Model, composed map[string]classKeys) {
	for _, p := range m.GetProperties() {
		c := v.compositions[p.GetMachine()]
		if c == nil || p.GetWhen() == nil {
			continue
		}
		at, owner := p.GetPosition(), p.GetMachine()+"."+p.GetName()
		keys, ok := v.readable(at, owner, composed[c.GetName()])
		if !ok {
			continue
		}
		switch w := p.GetWhen().(type) {
		case *umpirespb.Property_WhenClass:
			if key := classKey(v.in, v.actions, w.WhenClass); !hasKey(keys, key) {
				v.report(at, "%s: %s has no class %s", owner, c.GetName(), key)
			}
		case *umpirespb.Property_WhenAction:
			if !hasAction(keys, w.WhenAction) {
				v.report(at, "%s: %s has no class of the action %s", owner, c.GetName(), w.WhenAction)
			}
		default:
		}
	}
}

func hasKey(keys map[string]string, key string) bool {
	_, ok := keys[key]
	return ok
}

// hasAction is whether some class of these keys is of the action of this name.
func hasAction(keys map[string]string, action string) bool {
	for key := range keys {
		if actionOf(key) == action {
			return true
		}
	}
	return false
}

// identities lists each machine's state, outcome and fact catalogs and its classes, and refuses two
// values, two classes, or two state and class pairs that share a key. A catalog past the default
// ceilings is left to Build, which refuses it within its own.
func (v *validator) identities(m *umpirespb.Model) {
	report := func(err error) {
		var limit *LimitError
		if err != nil && !errors.As(err, &limit) {
			v.errs = append(v.errs, err)
		}
	}
	for _, mm := range m.GetMachines() {
		// The same counts Build refuses a machine by bound what is listed here.
		if err := v.in.preflight(mm, v.actions); err != nil {
			report(err)
			continue
		}
		var states []Value
		for i, t := range []string{mm.GetStateType(), mm.GetOutcomeType(), mm.GetFactType()} {
			if t == "" {
				continue
			}
			values, err := v.in.Members(named(t))
			report(err)
			if i == 0 {
				states = values
			}
		}
		classes, err := v.in.classes(mm, v.actions)
		report(err)
		if states != nil && err == nil {
			report(rowKeys(mm, states, classes))
		}
	}
}

// classKeys is a composition's class keys, each with the class it names, or why they are not known.
type classKeys struct {
	owners map[string]string
	err    error
}

// composedClasses keys every composition's classes, and reports a composition two of whose classes
// share a key; one past the default ceilings is left to the Scenarios that read its keys.
func (v *validator) composedClasses(m *umpirespb.Model) map[string]classKeys {
	out := map[string]classKeys{}
	for _, c := range m.GetCompositions() {
		owners, err := v.composedKeys(c)
		var limit *LimitError
		if err != nil && !errors.As(err, &limit) {
			v.errs = append(v.errs, err)
		}
		if limit != nil {
			limit.Machine = c.GetName()
		}
		out[c.GetName()] = classKeys{owners: owners, err: err}
	}
	return out
}

// composedKeys is every class key of a composition: `<field>_<class>` for a member's own class, and a
// sync's name followed by the inputs of each of its classes. A member's action a sync names steps
// only with its pair, so it has no class of its own.
// A key two classes spell alike, as `_` in a field or an action name and a sync named like a
// member's class allow, is refused where the composition is declared.
func (v *validator) composedKeys(c *umpirespb.Composition) (map[string]string, error) {
	if err := v.composedCount(c); err != nil {
		return nil, err
	}
	keys := &owned{composition: c, owners: map[string]string{}}
	synced := syncedActions(c)
	members := map[string]*umpirespb.Machine{}
	for _, mb := range c.GetMembers() {
		members[mb.GetField()] = v.machines[mb.GetMachine()]
		if err := v.memberKeys(keys, mb, members[mb.GetField()], synced); err != nil {
			return nil, err
		}
	}
	for _, s := range c.GetSyncs() {
		if err := v.syncKeys(keys, s, members); err != nil {
			return nil, err
		}
	}
	return keys.owners, nil
}

// syncedActions is the member actions a composition's syncs name, each as its member's field and the
// action's name.
func syncedActions(c *umpirespb.Composition) map[[2]string]bool {
	synced := map[[2]string]bool{}
	for _, s := range c.GetSyncs() {
		for _, move := range []*umpirespb.SyncMove{s.GetFirst(), s.GetSecond()} {
			synced[[2]string{move.GetMember(), move.GetAction()}] = true
		}
	}
	return synced
}

// composedCount counts a composition's class keys, its members' classes no sync names and each
// sync's pairs of them, refusing them past the Members ceiling before any is made.
func (v *validator) composedCount(c *umpirespb.Composition) error {
	var n count
	synced := syncedActions(c)
	members := map[string]*umpirespb.Machine{}
	for _, mb := range c.GetMembers() {
		members[mb.GetField()] = v.machines[mb.GetMachine()]
		for _, b := range members[mb.GetField()].GetSteps() {
			a, ok := v.actions[b.GetAction()]
			if !ok {
				return errorAt(b.GetPosition(), "no action %s", b.GetAction())
			}
			if synced[[2]string{mb.GetField(), a.GetName()}] {
				continue
			}
			k, err := v.in.sizeOfProduct(inputFields(a))
			if err != nil {
				return err
			}
			n = n.plus(k)
		}
	}
	for _, s := range c.GetSyncs() {
		first, err := v.actionCount(members[s.GetFirst().GetMember()], s.GetFirst().GetAction())
		if err != nil {
			return err
		}
		second, err := v.actionCount(members[s.GetSecond().GetMember()], s.GetSecond().GetAction())
		if err != nil {
			return err
		}
		n = n.plus(first.times(second))
	}
	return v.in.within("classes", v.in.ceilings.Members, n)
}

// actionCount counts the classes of the action of this name a member binds.
func (v *validator) actionCount(mm *umpirespb.Machine, action string) (count, error) {
	for _, b := range mm.GetSteps() {
		if a := v.actions[b.GetAction()]; a.GetName() == action {
			return v.in.sizeOfProduct(inputFields(a))
		}
	}
	return count{}, nil
}

// owned is a composition's class keys as they are made, each with the class it names.
type owned struct {
	composition *umpirespb.Composition
	owners      map[string]string
}

func (o *owned) own(key, owner string) error {
	if earlier, ok := o.owners[key]; ok && earlier != owner {
		c := o.composition
		return errorAt(c.GetPosition(), "composition %s: %s and %s share the key %q", c.GetName(), earlier, owner, key)
	}
	o.owners[key] = owner
	return nil
}

func (v *validator) memberKeys(keys *owned, mb *umpirespb.Member, mm *umpirespb.Machine, synced map[[2]string]bool) error {
	for _, b := range mm.GetSteps() {
		if synced[[2]string{mb.GetField(), v.actions[b.GetAction()].GetName()}] {
			continue
		}
		classes, err := v.inputKeys(v.actions[b.GetAction()])
		if err != nil {
			return err
		}
		for _, k := range classes {
			class := strings.Join(append([]string{v.actions[b.GetAction()].GetName()}, k...), "-")
			if err := keys.own(mb.GetField()+"_"+class, "member "+mb.GetField()+"'s class "+class); err != nil {
				return err
			}
		}
	}
	return nil
}

func (v *validator) syncKeys(keys *owned, s *umpirespb.Sync, members map[string]*umpirespb.Machine) error {
	first, err := v.syncInputs(members[s.GetFirst().GetMember()], s.GetFirst().GetAction())
	if err != nil {
		return err
	}
	second, err := v.syncInputs(members[s.GetSecond().GetMember()], s.GetSecond().GetAction())
	if err != nil {
		return err
	}
	for _, x := range first {
		for _, y := range second {
			owner := "sync " + s.GetName() + " of " + s.GetFirst().GetMember() + "." + s.GetFirst().GetAction() + " and " +
				s.GetSecond().GetMember() + "." + s.GetSecond().GetAction()
			if len(x)+len(y) > 0 {
				owner += "(" + strings.Join(x, ", ") + "; " + strings.Join(y, ", ") + ")"
			}
			if err := keys.own(strings.Join(append(append([]string{s.GetName()}, x...), y...), "-"), owner); err != nil {
				return err
			}
		}
	}
	return nil
}

// syncInputs is the input keys of every class of the action of this name a member binds.
func (v *validator) syncInputs(mm *umpirespb.Machine, action string) ([][]string, error) {
	for _, b := range mm.GetSteps() {
		if a := v.actions[b.GetAction()]; a.GetName() == action {
			return v.inputKeys(a)
		}
	}
	return nil, nil
}

// inputKeys is, for every class of an action, the keys of its inputs.
func (v *validator) inputKeys(a *umpirespb.Action) ([][]string, error) {
	assignments, err := v.in.product(inputFields(a))
	if err != nil {
		return nil, err
	}
	out := make([][]string, len(assignments))
	for i, inputs := range assignments {
		for _, x := range inputs {
			out[i] = append(out[i], x.Key())
		}
	}
	return out, nil
}
