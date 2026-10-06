package lint

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/model"
)

// tallies is one kind's tallies as they are counted, by owner. Each thing a kind looks at is added
// once, satisfied or not, so a tally's population less its findings is what satisfies the kind.
type tallies struct {
	kind  Kind
	owner map[string]*Tally
}

func tally(k Kind) tallies { return tallies{kind: k, owner: map[string]*Tally{}} }

// add counts one thing of an owner, and a finding of it where it does not satisfy the kind.
func (t tallies) add(owner string, satisfied bool, subject string, at *umpirespb.Position, format string, args ...any) {
	t.addAt(owner, satisfied, subject, where(at), format, args...)
}

// addAt is add at a position already spelled `file:line`, as the law sidecar records one.
func (t tallies) addAt(owner string, satisfied bool, subject, position string, format string, args ...any) {
	x, ok := t.owner[owner]
	if !ok {
		x = &Tally{Kind: t.kind, Owner: owner}
		t.owner[owner] = x
	}
	x.Population++
	if !satisfied {
		x.Findings = append(x.Findings, Finding{Kind: t.kind, Owner: owner, Subject: subject,
			Message: fmt.Sprintf(format, args...), Position: position})
	}
}

// list is the tallies by owner.
func (t tallies) list() []Tally {
	owners := make([]string, 0, len(t.owner))
	for o := range t.owner {
		owners = append(owners, o)
	}
	slices.Sort(owners)
	out := make([]Tally, len(owners))
	for i, o := range owners {
		out[i] = *t.owner[o]
	}
	return out
}

// unaskedProperties is each Property, by the machine or composition it is declared on, that no
// Query names.
func unaskedProperties(m *Model) ([]Tally, error) {
	asked := map[[2]string]bool{}
	for _, q := range m.IR.GetQueries() {
		asked[[2]string{q.GetProperty().GetMachine(), q.GetProperty().GetName()}] = true
	}
	t := tally(UnaskedProperty)
	for _, p := range m.IR.GetProperties() {
		t.add(p.GetMachine(), asked[[2]string{p.GetMachine(), p.GetName()}], p.GetName(), p.GetPosition(),
			"%s is named by no Query", p.GetName())
	}
	return t.list(), nil
}

// unfiredVerifies is each verify Query, by its Property's owner, whose receipt shows the Property
// read on no step the search explored: a verified result that holds of nothing. A counterexample
// fired it whatever the receipt says it exercised.
func unfiredVerifies(m *Model) ([]Tally, error) {
	t := tally(UnfiredVerify)
	for _, q := range m.IR.GetQueries() {
		if q.GetForm() != umpirespb.Query_FORM_VERIFY {
			continue
		}
		// A Query's receipt is keyed by its Scenario's machine, which the search runs on.
		i := slices.IndexFunc(m.Verified.Receipts, func(r model.Receipt) bool {
			return r.Subject == model.QuerySubject && r.Key.Name == q.GetName() && r.Key.Owner == q.GetScenario().GetMachine()
		})
		if i < 0 {
			return nil, fmt.Errorf("no receipt answers the Query %s", q.GetName())
		}
		r := m.Verified.Receipts[i]
		t.add(q.GetProperty().GetMachine(), r.Exercised || r.Kind == model.Counterexample, q.GetName(), q.GetPosition(),
			"%s: its Property fired on no step the search explored (%s)", q.GetName(), r.Kind)
	}
	return t.list(), nil
}

// system is whether an action is the system's own, which no realization performs: a timer, an
// internal step, a channel's delivery or loss, or an action of the actor `system`.
func system(a *umpirespb.Action) bool {
	return a.GetActor() == "system" || a.GetTimer() || a.GetInternal() || a.GetDelivers() != "" || a.GetLoses() != ""
}

// realized is each machine some realization runs, by name, and its realizations.
func (m *Model) realized() map[string][]*umpirespb.Realization {
	out := map[string][]*umpirespb.Realization{}
	for _, r := range m.IR.GetRealizations() {
		out[r.GetMachine()] = append(out[r.GetMachine()], r)
	}
	return out
}

// machine is the declaration of the machine of a name, or none.
func (m *Model) machine(name string) *umpirespb.Machine {
	i := slices.IndexFunc(m.IR.GetMachines(), func(d *umpirespb.Machine) bool { return d.GetName() == name })
	if i < 0 {
		return nil
	}
	return m.IR.GetMachines()[i]
}

// declared is the declared type of a name, or none.
func (m *Model) declared(name string) *umpirespb.Type {
	i := slices.IndexFunc(m.IR.GetTypes(), func(t *umpirespb.Type) bool { return t.GetName() == name })
	if i < 0 {
		return nil
	}
	return m.IR.GetTypes()[i]
}

// unperformedActions is each action a realized machine's step bindings name whose party is not the
// system's, which no performance of a realization of it binds and no activity script starts with.
func unperformedActions(m *Model) ([]Tally, error) {
	t := tally(UnperformedAction)
	for name, rs := range m.realized() {
		performed := map[string]bool{}
		for _, r := range rs {
			for _, s := range r.GetScripts() {
				for _, c := range s.GetActivity().GetStarts() {
					performed[c.GetAction()] = true
				}
				for _, item := range s.GetItems() {
					for _, p := range item.GetPerforms() {
						performed[p.GetStep().GetAction()] = true
					}
				}
			}
		}
		seen := map[string]bool{}
		for _, b := range m.machine(name).GetSteps() {
			a := m.actions[b.GetAction()]
			if a == nil || system(a) || seen[a.GetId()] {
				continue
			}
			seen[a.GetId()] = true
			t.add(name, performed[a.GetId()], a.GetName(), a.GetPosition(),
				"%s, an action of %s, is performed by no realization", a.GetName(), a.GetActor())
		}
	}
	return t.list(), nil
}

// unevidencedFacts is each fact constructor of a realized machine whose evidence name no evidence kind
// of a realization of it records. A constructor the machine's evidence function names nothing for
// has no name, and nothing records it.
func unevidencedFacts(m *Model) ([]Tally, error) {
	t := tally(UnevidencedFact)
	for name, rs := range m.realized() {
		decl, machine := m.machine(name), m.Machines[name]
		if decl.GetFactType() == "" || machine == nil {
			continue
		}
		recorded := map[string]bool{}
		for _, r := range rs {
			for _, e := range r.GetEvidence() {
				recorded[e.GetRecords()] = true
			}
		}
		evidence := map[string]string{}
		for _, line := range machine.Table.Evidence {
			evidence[line[0]] = line[1]
		}
		facts := m.declared(decl.GetFactType())
		for _, c := range facts.GetEnum().GetCases() {
			records, named := evidence[c.GetName()]
			if !named {
				t.add(name, false, c.GetName(), facts.GetPosition(), "fact %s has no evidence name", c.GetName())
				continue
			}
			t.add(name, recorded[records], c.GetName(), facts.GetPosition(), "fact %s: no evidence kind records %s", c.GetName(), records)
		}
	}
	return t.list(), nil
}

// untakenChoices is each named choice of the functions a machine's step bindings call, directly or
// through the functions they call, that no result of a row of a reachable state takes.
func untakenChoices(m *Model) ([]Tally, error) {
	functions := map[string]*umpirespb.Function{}
	for _, f := range m.IR.GetFunctions() {
		functions[f.GetName()] = f
	}
	t := tally(UntakenChoice)
	for _, decl := range m.IR.GetMachines() {
		machine := m.Machines[decl.GetName()]
		if machine == nil {
			continue
		}
		taken := map[string]bool{}
		for _, row := range reachableRows(machine.Table) {
			for _, r := range row.Results {
				taken[r.Choice] = true
			}
		}
		c := choices{functions: functions, called: map[string]bool{}, at: map[string]*umpirespb.Position{}}
		for _, b := range decl.GetSteps() {
			c.call(b.GetFunction())
		}
		for _, name := range c.names {
			t.add(decl.GetName(), taken[name], name, c.at[name], "choice %s is taken by no reachable state", name)
		}
	}
	return t.list(), nil
}

// choices is the named choices of the functions a walk calls, in the order it first meets them, and
// where it first met each.
type choices struct {
	functions map[string]*umpirespb.Function
	called    map[string]bool
	names     []string
	at        map[string]*umpirespb.Position
}

func (c *choices) call(function string) {
	if c.called[function] {
		return
	}
	c.called[function] = true
	c.expr(c.functions[function].GetBody())
}

func (c *choices) expr(e *umpirespb.Expr) {
	if e == nil {
		return
	}
	switch k := e.GetKind().(type) {
	case *umpirespb.Expr_Field:
		c.expr(k.Field.GetBase())
	case *umpirespb.Expr_Call:
		c.exprs(k.Call.GetArgs())
		c.call(k.Call.GetFunction())
	case *umpirespb.Expr_Construct:
		if name := k.Construct.GetChoice(); name != "" && c.at[name] == nil {
			c.names = append(c.names, name)
			c.at[name] = e.GetPosition()
		}
		c.exprs(k.Construct.GetArgs())
	case *umpirespb.Expr_Copy:
		c.expr(k.Copy.GetBase())
		for _, u := range k.Copy.GetUpdates() {
			c.expr(u.GetValue())
		}
	case *umpirespb.Expr_Unary:
		c.expr(k.Unary.GetOperand())
	case *umpirespb.Expr_Binary:
		c.exprs([]*umpirespb.Expr{k.Binary.GetLeft(), k.Binary.GetRight()})
	case *umpirespb.Expr_If:
		c.exprs([]*umpirespb.Expr{k.If.GetCondition(), k.If.GetThen(), k.If.GetElse()})
	case *umpirespb.Expr_Match:
		c.expr(k.Match.GetScrutinee())
		for _, mc := range k.Match.GetCases() {
			c.exprs([]*umpirespb.Expr{mc.GetGuard(), mc.GetBody()})
		}
	case *umpirespb.Expr_Let:
		c.exprs([]*umpirespb.Expr{k.Let.GetValue(), k.Let.GetBody()})
	case *umpirespb.Expr_List:
		c.exprs(k.List.GetItems())
	case *umpirespb.Expr_Lambda:
		c.expr(k.Lambda.GetBody())
	case *umpirespb.Expr_Inbox:
		c.exprs([]*umpirespb.Expr{k.Inbox.GetContents(), k.Inbox.GetMessage()})
	default:
		// A literal, a variable and a hole hold no expression.
	}
}

func (c *choices) exprs(es []*umpirespb.Expr) {
	for _, e := range es {
		c.expr(e)
	}
}

// reachableRows is the rows of a table whose state it reaches.
func reachableRows(t *model.Table) []model.Row {
	reachable := map[string]bool{}
	for _, s := range t.Reachable {
		reachable[s] = true
	}
	var out []model.Row
	for _, row := range t.Rows {
		if reachable[row.Source] {
			out = append(out, row)
		}
	}
	return out
}

// unreadRefinements is each refining machine no Query reads a Property through its refinement on.
func unreadRefinements(m *Model) ([]Tally, error) {
	read := map[string]bool{}
	for _, q := range m.IR.GetQueries() {
		if !q.GetThrough() {
			continue
		}
		s, err := m.scenario(q)
		if err != nil {
			return nil, err
		}
		read[s.GetMachine()] = true
	}
	t := tally(UnreadRefinement)
	for _, decl := range m.IR.GetMachines() {
		r := decl.GetRefines()
		if r == nil {
			continue
		}
		t.add(decl.GetName(), read[decl.GetName()], decl.GetName()+" refines "+r.GetProduct(), decl.GetPosition(),
			"no Query reads through %s's refinement of %s", decl.GetName(), r.GetProduct())
	}
	return t.list(), nil
}

// scenario is the Scenario a Query names.
func (m *Model) scenario(q *umpirespb.Query) (*umpirespb.Scenario, error) {
	ref := q.GetScenario()
	i := slices.IndexFunc(m.IR.GetScenarios(), func(s *umpirespb.Scenario) bool {
		return s.GetMachine() == ref.GetMachine() && s.GetName() == ref.GetName()
	})
	if i < 0 {
		return nil, fmt.Errorf("the Query %s names no Scenario %s of %s", q.GetName(), ref.GetName(), ref.GetMachine())
	}
	return m.IR.GetScenarios()[i], nil
}

// commands is every command of a realization's scripts: each item's own, and each performance's.
func commands(r *umpirespb.Realization) []*umpirespb.Command {
	var out []*umpirespb.Command
	for _, s := range r.GetScripts() {
		for _, item := range s.GetItems() {
			if item.GetCommand() != nil {
				out = append(out, item.GetCommand())
			}
			for _, p := range item.GetPerforms() {
				out = append(out, p.GetCommand())
			}
		}
	}
	return out
}

// unreadObservations is each observation of a realization that neither its correlation names nor a
// read of one of its commands observes a value into or lifts evidence into.
func unreadObservations(m *Model) ([]Tally, error) {
	t := tally(UnreadObservation)
	for _, r := range m.IR.GetRealizations() {
		read := map[string]bool{r.GetCorrelation().GetObservation(): true}
		for _, c := range commands(r) {
			for _, rd := range c.GetRpc().GetReads() {
				for _, tg := range rd.GetTargets() {
					read[tg.GetObserve()], read[tg.GetLift()] = true, true
				}
			}
		}
		for _, o := range r.GetObservations() {
			t.add(r.GetMachine(), o.GetId() != "" && read[o.GetId()], o.GetId(), o.GetPosition(),
				"observation %s of %s is filled by no read and named by no correlation", o.GetId(), r.GetName())
		}
	}
	return t.list(), nil
}

// explicitWaits is each poll that writes its own interval in a realization that declares an API
// behavior, where the lowering would derive the poll's wait from the hints and leaves a written one as
// it is. A realization that declares none has nothing to derive a wait from, so its polls are not
// counted.
func explicitWaits(m *Model) ([]Tally, error) {
	t := tally(ExplicitWait)
	for _, r := range m.IR.GetRealizations() {
		if r.GetBehavior() == nil {
			continue
		}
		for _, s := range r.GetScripts() {
			for _, item := range s.GetItems() {
				cs := []*umpirespb.Command{item.GetCommand()}
				for _, p := range item.GetPerforms() {
					cs = append(cs, p.GetCommand())
				}
				for _, c := range cs {
					if poll := c.GetPoll(); poll != nil {
						t.add(r.GetMachine(), poll.GetIntervalMs() == 0, s.GetId()+"/"+c.GetId(), c.GetPosition(),
							"command %s/%s of %s polls every %d milliseconds, though its realization declares the API behavior a read's wait is derived from",
							s.GetId(), c.GetId(), r.GetName(), poll.GetIntervalMs())
					}
				}
			}
		}
	}
	return t.list(), nil
}

// unreachableValues is each value of each field of a machine's state that no reachable state holds.
// A state of an enum type is one field, named after its type. Each field is read on its own: a
// combination of values no reachable state holds is no finding.
func unreachableValues(m *Model) ([]Tally, error) {
	t := tally(UnreachableValue)
	for _, decl := range m.IR.GetMachines() {
		machine := m.Machines[decl.GetName()]
		state := m.declared(decl.GetStateType())
		if machine == nil || state == nil {
			continue
		}
		var reached []model.Value
		for _, key := range machine.Table.Reachable {
			v, ok := machine.State(key)
			if !ok {
				return nil, fmt.Errorf("%s: no state %s", decl.GetName(), key)
			}
			reached = append(reached, v)
		}
		l := stateLeaves{m: m, t: t, machine: decl.GetName(), at: state.GetPosition(), reached: reached}
		if state.GetRecord() != nil {
			for i, f := range state.GetRecord().GetFields() {
				if err := l.visit(f.GetName(), f.GetType(), func(v model.Value) model.Value { return v.Fields[i] }); err != nil {
					return nil, err
				}
			}
			continue
		}
		root := &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Named{Named: decl.GetStateType()}}
		if err := l.visit(decl.GetStateType()[strings.LastIndex(decl.GetStateType(), ".")+1:], root, func(v model.Value) model.Value { return v }); err != nil {
			return nil, err
		}
	}
	return t.list(), nil
}

// stateLeaves reads one machine's state field by field, down to its leaves: the cases of an enum,
// apart from what they carry, and the values of a Boolean or an integer range. A record is its fields,
// and a channel's contents, like a record, are a combination, which is never reported.
type stateLeaves struct {
	m       *Model
	t       tallies
	machine string
	at      *umpirespb.Position
	reached []model.Value
}

func (l stateLeaves) visit(path string, ref *umpirespb.TypeRef, of func(model.Value) model.Value) error {
	if record := l.m.declared(ref.GetNamed()).GetRecord(); ref.GetNamed() != "" && record != nil {
		for i, f := range record.GetFields() {
			if err := l.visit(path+"."+f.GetName(), f.GetType(), func(v model.Value) model.Value { return of(v).Fields[i] }); err != nil {
				return err
			}
		}
		return nil
	}
	values, key, err := l.values(path, ref)
	if err != nil || key == nil {
		return err
	}
	held := map[string]bool{}
	for _, v := range l.reached {
		held[key(of(v))] = true
	}
	for _, v := range values {
		l.t.add(l.machine, held[v], path+"="+v, l.at, "%s %s is held by no reachable state", path, v)
	}
	return nil
}

// values is a leaf's values and how a value of the field is keyed among them, or none for a field
// that is a combination.
func (l stateLeaves) values(path string, ref *umpirespb.TypeRef) ([]string, func(model.Value) string, error) {
	var values []string
	switch {
	case ref.GetNamed() != "":
		for _, c := range l.m.declared(ref.GetNamed()).GetEnum().GetCases() {
			values = append(values, c.GetName())
		}
		return values, func(v model.Value) string { return v.Case }, nil
	case ref.GetBool() != nil || ref.GetIntRange() != nil:
		catalog, err := l.m.In.Members(ref)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: field %s: %w", l.machine, path, err)
		}
		for _, v := range catalog {
			values = append(values, v.Key())
		}
		return values, model.Value.Key, nil
	default:
		return nil, nil, nil
	}
}

// neverEnabled is each action class of a machine that no row of a reachable state enables.
func neverEnabled(m *Model) ([]Tally, error) {
	t := tally(NeverEnabled)
	for _, decl := range m.IR.GetMachines() {
		machine := m.Machines[decl.GetName()]
		if machine == nil {
			continue
		}
		enabled := map[string]bool{}
		for _, row := range reachableRows(machine.Table) {
			enabled[row.Action] = true
		}
		for _, c := range machine.Classes {
			at := c.Action.GetPosition()
			if i := slices.IndexFunc(decl.GetSteps(), func(b *umpirespb.StepBinding) bool {
				return b.GetAction() == c.Action.GetId()
			}); i >= 0 {
				at = decl.GetSteps()[i].GetPosition()
			}
			t.add(decl.GetName(), enabled[c.Key], c.Key, at, "%s is enabled in no reachable state", c.Key)
		}
	}
	return t.list(), nil
}

// stuckStates is each reachable state of a machine that is no end and in which no row has a result:
// no action class, a timer's and an internal step's included, can happen there. A state with a hole
// row is not stuck: a hole declares that unmodeled behavior may happen there, not a forgotten rule,
// as the progress check reads it (model/SEMANTICS.md, Progress). The finding is at the machine and
// carries the shortest path from a start to the state.
func stuckStates(m *Model) ([]Tally, error) {
	t := tally(StuckState)
	for _, decl := range m.IR.GetMachines() {
		machine := m.Machines[decl.GetName()]
		if machine == nil {
			continue
		}
		table := machine.Table
		steps := map[string]bool{}
		for _, row := range table.Rows {
			if len(row.Results) > 0 {
				steps[row.Source] = true
			}
		}
		for _, h := range machine.Holes {
			steps[h.Source] = true
		}
		for _, s := range table.Reachable {
			t.add(decl.GetName(), steps[s] || slices.Contains(table.Ends, s), s, decl.GetPosition(),
				"%s is reachable, is no end and enables no action class: no action can happen in it, so a timer or an "+
					"internal step may be missing a rule; if it is meant to be final, declare it in the machine's ends; "+
					"reached by %s", s, spellPath(table.PathTo(s)))
		}
	}
	return t.list(), nil
}

// spellPath spells a witness as its start and each step's action and outcome to the state it reaches.
func spellPath(w *model.Trace) string {
	if w == nil {
		return "no path"
	}
	var b strings.Builder
	b.WriteString(w.Initial.Value)
	for _, s := range w.Steps {
		fmt.Fprintf(&b, " -%s/%s-> %s", s.Action.Value, s.Outcome.Value, s.State.Value)
	}
	return b.String()
}

// unproduced is each outcome and each fact of a machine that no result of a row of a reachable state
// produces.
func unproduced(m *Model) ([]Tally, error) {
	t := tally(Unproduced)
	for _, decl := range m.IR.GetMachines() {
		machine := m.Machines[decl.GetName()]
		if machine == nil {
			continue
		}
		outcomes, facts := map[string]bool{}, map[string]bool{}
		for _, row := range reachableRows(machine.Table) {
			for _, r := range row.Results {
				outcomes[r.Outcome] = true
				for _, f := range r.Facts {
					facts[f] = true
				}
			}
		}
		for _, o := range machine.Table.Outcomes {
			t.add(decl.GetName(), outcomes[o], "outcome "+o, m.declared(decl.GetOutcomeType()).GetPosition(),
				"outcome %s is produced by no reachable state", o)
		}
		for _, f := range machine.Table.Facts {
			t.add(decl.GetName(), facts[f], "fact "+f, m.declared(decl.GetFactType()).GetPosition(),
				"fact %s is recorded by no reachable state", f)
		}
	}
	return t.list(), nil
}

// unrealizedFinds is each find Query, by its Scenario's machine, whose standing lowering gives as
// `no-realization`: no realization runs its machine.
func unrealizedFinds(m *Model) ([]Tally, error) {
	if m.lowering.Unrealized == nil {
		return nil, errors.New("lint needs lowering's standing of a find Query, and was given none")
	}
	t := tally(UnrealizedFind)
	for _, q := range m.IR.GetQueries() {
		if q.GetForm() != umpirespb.Query_FORM_FIND {
			continue
		}
		s, err := m.scenario(q)
		if err != nil {
			return nil, err
		}
		unrealized, err := m.lowering.Unrealized(q, s, m.IR.GetRealizations())
		if err != nil {
			return nil, fmt.Errorf("the Query %s: %w", q.GetName(), err)
		}
		t.add(s.GetMachine(), !unrealized, q.GetName(), q.GetPosition(), "%s: no realization runs %s", q.GetName(), s.GetMachine())
	}
	return t.list(), nil
}
