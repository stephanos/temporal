// Package backends exports a finite slice of the Umpire IR to other checkers and holds each of them
// to the Go reading of the same Model. An export is accepted only where the backend's own evaluation
// agrees with Go's: Quint on every reachable state, transition, result and disabled pair of each
// exported machine, on its monitors and on its Properties, and P on one monitor over bounded event
// traces. What a backend is not given, and what it could not be run on, is listed as such and is never
// an agreement.
//
// The Go side is tools/umpire/model, through its public API alone.
package export

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
)

// Claim is what a receipt is about. The claims are kept apart: that a backend's transitions are
// Go's says nothing of what a checker of that backend explored, and neither is a refinement.
type Claim string

const (
	// TransitionAgreement is a machine's reachable transition relation, compared pair by pair.
	TransitionAgreement Claim = "transition-agreement"
	// PropertyAgreement is a machine's Properties, compared on every step from every reachable state:
	// whether each is about the step, and whether it holds of it.
	PropertyAgreement Claim = "property-agreement"
	// MonitorAgreement is monitors compared with Go's: by Quint over every reachable step of the product
	// of a machine and its monitors, and by P over one monitor's verdicts on bounded event traces.
	MonitorAgreement Claim = "monitor-agreement"
	// CheckerCoverage is what a backend's checker explored, and how.
	CheckerCoverage Claim = "checker-coverage"
	// ModuleRefinement is a refinement between machines, which no backend here checks.
	ModuleRefinement Claim = "module-refinement"
	// QueryAgreement is the Model's Queries and ProgressAgreement its progress claims, which no backend
	// here answers.
	QueryAgreement    Claim = "query-agreement"
	ProgressAgreement Claim = "progress-agreement"
)

// Kind is what a receipt says of its claim.
type Kind string

const (
	// Agreed is a comparison that ran over everything its receipt counts and found no difference.
	Agreed Kind = "agreed"
	// Disagreed is a comparison that found a difference.
	Disagreed Kind = "disagreed"
	// Covered is a checker's run, with what it explored. It is no agreement.
	Covered Kind = "covered"
	// Unsupported is a declaration the backend is not given. It is listed, and is no check.
	Unsupported Kind = "unsupported"
	// ResourceLimit is a composition past a ceiling of the scope: Go builds no table of it, and nothing
	// of it is exported.
	ResourceLimit Kind = "resource-limit"
	// NotRun is a check the backend's tool could not be run on. It is no agreement.
	NotRun Kind = "not-run"
	// WitnessRejected is a backend's counterexample that did not replay through Go: an error, whatever
	// else the comparison found.
	WitnessRejected Kind = "witness-rejected"
)

// Witness is a backend's counterexample of one monitor, as a path of the machine.
type Witness struct {
	Monitor string
	Trace   *umpiremodel.Trace
}

// Receipt is what one comparison of a backend with Go established, and over what.
type Receipt struct {
	Backend string
	// Model names the slice the comparison is of, as its Slice is named.
	Model       string
	Claim       Claim
	Subject     string
	Kind        Kind
	Explanation string

	// States, Pairs, Enabled, Disabled and Results count the reachable states and the state and class
	// pairs a transition agreement compared: every pair is either enabled, with its results, or
	// disabled.
	States, Pairs, Enabled, Disabled, Results int
	// ProductStates and Steps count the states of the product of a machine and its monitors and the
	// steps from them a monitor agreement compared.
	ProductStates, Steps int
	// Properties, Reads and About count the Properties a property agreement compared, the steps each
	// was read on, and the readings on which a Property was about its step.
	Properties, Reads, About int
	// Traces, Accepted and Rejected count the event traces a monitor agreement compared.
	Traces, Accepted, Rejected int
	// Violated names the monitors the backend found violated, and Witnesses its counterexamples, each
	// replayed through Go.
	Violated  []string
	Witnesses []Witness
	// Differences lists what differs, the first few.
	Differences []string
}

// String is the receipt on one line: who compared what, what came of it, and over how much.
func (r Receipt) String() string {
	out := fmt.Sprintf("%s %s %s %s: %s", r.Backend, r.Model, r.Claim, r.Subject, r.Kind)
	if r.Explanation != "" {
		out += "; " + r.Explanation
	}
	if len(r.Violated) > 0 {
		out += "; violated: " + strings.Join(r.Violated, ", ")
	}
	if len(r.Differences) > 0 {
		out += "; differences: " + strings.Join(r.Differences, " | ")
	}
	return out
}

// UnsupportedError is a construct an exporter does not translate. Nothing is exported around it.
type UnsupportedError struct {
	Backend   string
	Construct string
	Position  string
}

func (e *UnsupportedError) Error() string {
	at := ""
	if e.Position != "" {
		at = e.Position + ": "
	}
	return fmt.Sprintf("%s%s export: unsupported: %s", at, e.Backend, e.Construct)
}

// Slice is an admitted Model with Go's interpretation of it: what a backend is exported from and
// compared with.
type Slice struct {
	// Name is what the slice's receipts call it: the caller's name for the Model, such as its file's.
	Name     string
	Model    *umpirespb.Model
	machines map[string]*umpiremodel.Machine
	in       *umpiremodel.Interpreter
	// bound reads the Model's compositions as the reader's checker builds them.
	bound   *umpiremodel.Realizer
	types   map[string]*umpirespb.Type
	actions map[string]*umpirespb.Action
}

// Open admits a Model and interprets its machines, within the reader's default scope.
func Open(m *umpirespb.Model) (*Slice, error) { return OpenWithin(m, umpiremodel.DefaultScope) }

// OpenWithin is Open within a scope, whose ceilings bound the Model's compositions.
func OpenWithin(m *umpirespb.Model, scope umpiremodel.Scope) (*Slice, error) {
	bound, err := umpiremodel.NewRealizer(m, scope)
	if err != nil {
		return nil, err
	}
	machines, err := umpiremodel.Build(m)
	var hole *umpiremodel.Hole
	if errors.As(err, &hole) {
		return nil, &UnsupportedError{Backend: "backend", Construct: "a machine left without a table by " + describeHole(m, hole.ID),
			Position: hole.Position}
	}
	if err != nil {
		return nil, err
	}
	s := &Slice{Model: m, machines: machines, in: umpiremodel.NewInterpreter(m), bound: bound, types: map[string]*umpirespb.Type{},
		actions: map[string]*umpirespb.Action{}}
	for _, t := range m.GetTypes() {
		s.types[t.GetName()] = t
	}
	for _, a := range m.GetActions() {
		s.actions[a.GetId()] = a
	}
	return s, nil
}

func describeHole(m *umpirespb.Model, id string) string {
	for _, h := range m.GetHoles() {
		if h.GetId() == id {
			return "the hole " + h.GetName()
		}
	}
	return "an undeclared hole"
}

// result is one result of a row, by keys.
type result struct {
	Outcome string
	State   string
	Facts   []string
	Because string
}

// machineView is a machine as a backend and Go are compared on it: its starts, the states they
// reach, the ends among them, its classes, and for every reachable state and class the results, none
// for a disabled pair. Closed is whether no step leaves the reachable states.
type machineView struct {
	Starts  []string
	Reach   []string
	Closed  bool
	Ends    []string
	Classes []string
	Rows    map[string]map[string][]result
	Product *productView
	// Properties names the machine's Properties, and Claims holds, for every reachable state and class
	// and each result of the row, what each Property says of that step.
	Properties []string
	Claims     map[string]map[string][][]claimRead
}

// claimRead is one Property read on one step: whether it is about the step, and whether it holds of
// it. A Property that is not about a step holds of it.
type claimRead struct {
	About bool
	Holds bool
}

// productState is a state of the product of a machine and its monitors.
type productState struct {
	State string
	Mu    []string
}

func (p productState) key() string { return p.State + " | " + strings.Join(p.Mu, " | ") }

// productStep is what one result of a row does to the monitors: each monitor's state after it,
// whether its verdict is read there, and whether that state violates it.
type productStep struct {
	Mu   []string
	Read []bool
	Viol []bool
}

// productView is the product of a machine and its monitors: for every reachable product state and
// class, one productStep per result of the row, in the row's result order.
type productView struct {
	Monitors []string
	Starts   []productState
	Closed   bool
	States   map[string]productState
	Steps    map[string]map[string][]productStep
}

// view is Go's reading of a machine: its table over the reachable states, and the product with its
// monitors as the IR declares them.
func (s *Slice) view(mm *umpiremodel.Machine) (*machineView, error) {
	t := mm.Table
	v := &machineView{Starts: slices.Clone(t.Starts), Reach: sorted(t.Reachable), Closed: true, Classes: sorted(t.Actions),
		Rows: map[string]map[string][]result{}}
	for _, e := range t.Ends {
		if slices.Contains(t.Reachable, e) {
			v.Ends = append(v.Ends, e)
		}
	}
	slices.Sort(v.Ends)
	for _, state := range t.Reachable {
		by := map[string][]result{}
		for _, class := range t.Actions {
			by[class] = nil
		}
		for _, row := range t.RowsFrom(state) {
			for _, r := range row.Results {
				by[row.Action] = append(by[row.Action], result{Outcome: r.Outcome, State: r.State, Facts: slices.Clone(r.Facts),
					Because: r.Because})
			}
		}
		v.Rows[state] = by
	}
	if len(mm.Monitors) > 0 {
		p, err := s.product(mm)
		if err != nil {
			return nil, err
		}
		v.Product = &p.productView
	}
	return v, s.claims(mm, v)
}

// composedView is Go's reading of a composition: the table the reader's checker builds for it, and what
// each of its Properties, as the checker binds them, says of every step. The table holds the states
// the members' starts reach and no others.
func composedView(c *umpiremodel.Composed) (*machineView, error) {
	t := c.Table
	v := &machineView{Starts: slices.Clone(t.Starts), Reach: sorted(t.States), Closed: true, Ends: sorted(t.Ends), Classes: sorted(t.Actions),
		Rows: map[string]map[string][]result{}}
	for _, p := range c.Properties {
		v.Properties = append(v.Properties, p.Name)
	}
	if len(c.Properties) > 0 {
		v.Claims = map[string]map[string][][]claimRead{}
	}
	for _, state := range t.States {
		by, claims := map[string][]result{}, map[string][][]claimRead{}
		for _, class := range t.Actions {
			by[class], claims[class] = nil, nil
		}
		for _, row := range t.RowsFrom(state) {
			for _, r := range row.Results {
				by[row.Action] = append(by[row.Action], result{Outcome: r.Outcome, State: r.State, Facts: slices.Clone(r.Facts), Because: r.Because})
				reads, err := composedReads(c.Properties, state, row.Action, r)
				if err != nil {
					return nil, err
				}
				claims[row.Action] = append(claims[row.Action], reads)
			}
		}
		v.Rows[state] = by
		if v.Claims != nil {
			v.Claims[state] = claims
		}
	}
	return v, nil
}

// composedReads reads a composition's Properties on one step, as the reader's checker binds them. A
// Property that is not about the step holds of it, and its function is not called there.
func composedReads(properties []umpiremodel.BoundProperty, state, class string, step umpiremodel.Result) ([]claimRead, error) {
	reads := make([]claimRead, len(properties))
	for i, p := range properties {
		reads[i] = claimRead{Holds: true}
		if !p.About(class) {
			continue
		}
		held, err := p.Holds(state, step)
		if err != nil {
			return nil, err
		}
		reads[i] = claimRead{About: true, Holds: held}
	}
	return reads, nil
}

// properties is the Properties a Model declares on a machine, in the Model's order.
func (s *Slice) properties(machine string) []*umpirespb.Property {
	var out []*umpirespb.Property
	for _, p := range s.Model.GetProperties() {
		if p.GetMachine() == machine {
			out = append(out, p)
		}
	}
	return out
}

// about is whether a Property is about the steps of a class: every step, the steps of one class, or
// the steps of every class of one action.
func (s *Slice) about(p *umpirespb.Property, class umpiremodel.Class) bool {
	switch w := p.GetWhen().(type) {
	case *umpirespb.Property_WhenClass:
		parts := []string{s.actions[w.WhenClass.GetAction()].GetName()}
		for _, input := range w.WhenClass.GetInputs() {
			parts = append(parts, s.literal(input).Key())
		}
		return class.Key == strings.Join(parts, "-")
	case *umpirespb.Property_WhenAction:
		return class.Action.GetName() == w.WhenAction
	default:
		return true
	}
}

// claims reads every Property of a machine on every step from every reachable state, with the reader's
// interpreter: a same-step Property of the step record, a transition Property of the state before
// the step and the step record.
func (s *Slice) claims(mm *umpiremodel.Machine, v *machineView) error {
	properties := s.properties(mm.Decl.GetName())
	if len(properties) == 0 {
		return nil
	}
	v.Claims = map[string]map[string][][]claimRead{}
	for _, p := range properties {
		v.Properties = append(v.Properties, p.GetName())
	}
	for _, state := range v.Reach {
		v.Claims[state] = map[string][][]claimRead{}
	}
	for _, tr := range mm.Transitions {
		by, reachable := v.Claims[tr.Source.Key()]
		if !reachable {
			continue
		}
		for _, step := range tr.Steps {
			reads := make([]claimRead, len(properties))
			for i, p := range properties {
				var err error
				if reads[i], err = s.read(p, tr, step); err != nil {
					return err
				}
			}
			by[tr.Class.Key] = append(by[tr.Class.Key], reads)
		}
	}
	return nil
}

// read reads one Property on one step of a row. A Property that is not about the step holds of it,
// and its function is not called there.
func (s *Slice) read(p *umpirespb.Property, tr umpiremodel.Transition, step umpiremodel.Value) (claimRead, error) {
	if !s.about(p, tr.Class) {
		return claimRead{Holds: true}, nil
	}
	args := []umpiremodel.Value{step}
	if p.GetTransition() {
		args = []umpiremodel.Value{tr.Source, step}
	}
	held, err := s.in.Call(p.GetHolds(), args, p.GetPosition())
	if err != nil {
		return claimRead{}, err
	}
	if held.Kind != umpiremodel.BoolValue {
		return claimRead{}, fmt.Errorf("%s.%s: %s is %s, not a Boolean", p.GetMachine(), p.GetName(), p.GetHolds(), held.Key())
	}
	return claimRead{About: true, Holds: held.Bool}, nil
}

func sorted(xs []string) []string {
	out := slices.Clone(xs)
	slices.Sort(out)
	return out
}

// goProduct is Go's product of a machine and its monitors, with how far each violation is.
type goProduct struct {
	productView
	// successors is the product states one step from each.
	successors map[string][]string
	// depth is how many steps the farthest product state is from the starts, and violatedAt, by
	// monitor, the length of a shortest path on whose last step it is read and violated.
	depth      int
	violatedAt map[string]int
}

// watching is a machine's monitors as the product steps them: each declaration, and the monitor
// states by key.
type watching struct {
	decls  []*umpirespb.Monitor
	ends   map[string]bool
	states []map[string]umpiremodel.Value
}

func (s *Slice) watching(mm *umpiremodel.Machine) (*watching, []string, error) {
	w := &watching{decls: mm.Monitors, ends: map[string]bool{}}
	for _, e := range mm.Table.Ends {
		w.ends[e] = true
	}
	var initial []string
	for _, mo := range mm.Monitors {
		members, err := s.in.Members(mo.GetState())
		if err != nil {
			return nil, nil, err
		}
		byKey := map[string]umpiremodel.Value{}
		for _, v := range members {
			byKey[v.Key()] = v
		}
		w.states = append(w.states, byKey)
		v, err := s.in.Eval(mo.GetInitial())
		if err != nil {
			return nil, nil, err
		}
		if _, ok := byKey[v.Key()]; !ok {
			return nil, nil, fmt.Errorf("monitor %s: its initial state %s is outside its states", mo.GetName(), v.Key())
		}
		initial = append(initial, v.Key())
	}
	return w, initial, nil
}

// step advances every monitor over one step: the IR's `next`, its evaluation point and its
// `violated`, evaluated by the reader's interpreter.
func (s *Slice) step(w *watching, mu []string, source umpiremodel.Value, step umpiremodel.Value, target string) (productStep, error) {
	out := productStep{Mu: make([]string, len(mu)), Read: make([]bool, len(mu)), Viol: make([]bool, len(mu))}
	for k, mo := range w.decls {
		next, err := s.in.Call(mo.GetNext(), []umpiremodel.Value{w.states[k][mu[k]], source, step}, mo.GetPosition())
		if err != nil {
			return out, err
		}
		if _, ok := w.states[k][next.Key()]; !ok {
			return out, fmt.Errorf("monitor %s: %s is outside its states", mo.GetName(), next.Key())
		}
		out.Mu[k] = next.Key()
		switch e := mo.GetEvaluate().(type) {
		case *umpirespb.Monitor_EveryStep:
			out.Read[k] = true
		case *umpirespb.Monitor_AtEnds:
			out.Read[k] = w.ends[target]
		case *umpirespb.Monitor_After:
			if out.Read[k], err = s.decide(e.After, step, mo); err != nil {
				return out, err
			}
		default:
			return out, fmt.Errorf("monitor %s has no evaluation point", mo.GetName())
		}
		if out.Viol[k], err = s.decide(mo.GetViolated(), next, mo); err != nil {
			return out, err
		}
	}
	return out, nil
}

func (s *Slice) decide(function string, arg umpiremodel.Value, mo *umpirespb.Monitor) (bool, error) {
	v, err := s.in.Call(function, []umpiremodel.Value{arg}, mo.GetPosition())
	if err != nil {
		return false, err
	}
	if v.Kind != umpiremodel.BoolValue {
		return false, fmt.Errorf("monitor %s: %s is %s, not a Boolean", mo.GetName(), function, v.Key())
	}
	return v.Bool, nil
}

// product explores the product of a machine and its monitors from the machine's starts, to the end.
func (s *Slice) product(mm *umpiremodel.Machine) (*goProduct, error) {
	w, initial, err := s.watching(mm)
	if err != nil {
		return nil, err
	}
	p := &goProduct{productView: productView{Closed: true, States: map[string]productState{}, Steps: map[string]map[string][]productStep{}},
		successors: map[string][]string{}, violatedAt: map[string]int{}}
	for _, mo := range mm.Monitors {
		p.Monitors = append(p.Monitors, mo.GetName())
	}
	from := map[string][]umpiremodel.Transition{}
	for _, tr := range mm.Transitions {
		key := tr.Source.Key()
		from[key] = append(from[key], tr)
	}
	// The states are met in breadth-first order, so the first path to one is a shortest: steps holds
	// its length.
	steps := map[string]int{}
	var queue []productState
	meet := func(ps productState, after int) {
		if _, seen := p.States[ps.key()]; !seen {
			p.States[ps.key()] = ps
			queue = append(queue, ps)
			steps[ps.key()], p.depth = after, max(p.depth, after)
		}
	}
	for _, start := range mm.Table.Starts {
		ps := productState{State: start, Mu: slices.Clone(initial)}
		p.Starts = append(p.Starts, ps)
		meet(ps, 0)
	}
	for len(queue) > 0 {
		ps := queue[0]
		queue = queue[1:]
		key := ps.key()
		p.Steps[key] = map[string][]productStep{}
		for _, class := range mm.Table.Actions {
			p.Steps[key][class] = nil
		}
		for _, tr := range from[ps.State] {
			for _, step := range tr.Steps {
				target := step.Fields[1].Key()
				taken, err := s.step(w, ps.Mu, tr.Source, step, target)
				if err != nil {
					return nil, err
				}
				p.Steps[key][tr.Class.Key] = append(p.Steps[key][tr.Class.Key], taken)
				next := productState{State: target, Mu: taken.Mu}
				p.successors[key] = append(p.successors[key], next.key())
				meet(next, steps[key]+1)
				p.violate(taken, steps[key]+1)
			}
		}
	}
	return p, nil
}

// violate notes, for each monitor a step is read and violated on, the first such step's distance
// from the starts.
func (p *goProduct) violate(taken productStep, after int) {
	for k, monitor := range p.Monitors {
		if _, found := p.violatedAt[monitor]; !found && taken.Read[k] && taken.Viol[k] {
			p.violatedAt[monitor] = after
		}
	}
}

// depth is how many steps the farthest state of a graph is from its starts.
func depth(starts []string, successors func(string) []string) int {
	seen := map[string]bool{}
	frontier := slices.Clone(starts)
	for _, s := range starts {
		seen[s] = true
	}
	d := 0
	for {
		var next []string
		for _, s := range frontier {
			for _, t := range successors(s) {
				if !seen[t] {
					seen[t] = true
					next = append(next, t)
				}
			}
		}
		if len(next) == 0 {
			return d
		}
		frontier, d = next, d+1
	}
}

// Replay replays a backend's counterexample of a monitor through a fresh interpretation of the
// Model: the trace must be a path of the machine's table, by its Definition IDs, from one of its
// starts, and the monitor must be read and violated on its last step. Anything else is an error.
func (s *Slice) Replay(machine, monitor string, trace *umpiremodel.Trace) error {
	fresh, err := Open(s.Model)
	if err != nil {
		return err
	}
	return fresh.replay(machine, monitor, trace)
}

func (s *Slice) replay(machine, monitor string, trace *umpiremodel.Trace) error {
	mm := s.machines[machine]
	if mm == nil {
		return fmt.Errorf("the Model has no machine %s", machine)
	}
	if err := mm.Table.Replay(trace); err != nil {
		return err
	}
	if !slices.Contains(mm.Table.Starts, trace.Initial.Value) {
		return fmt.Errorf("%s: the witness starts at '%s', which is no start", machine, trace.Initial.Value)
	}
	if len(trace.Steps) == 0 {
		return fmt.Errorf("%s: the witness takes no step, and a monitor is read on a step", machine)
	}
	w, mu, err := s.watching(mm)
	if err != nil {
		return err
	}
	k := slices.IndexFunc(mm.Monitors, func(mo *umpirespb.Monitor) bool { return mo.GetName() == monitor })
	if k < 0 {
		return fmt.Errorf("%s names no monitor %s", machine, monitor)
	}
	state := trace.Initial.Value
	var last productStep
	for i, step := range trace.Steps {
		j := slices.IndexFunc(mm.Transitions, func(tr umpiremodel.Transition) bool {
			return tr.Source.Key() == state && tr.Class.Key == step.Action.Value
		})
		if j < 0 {
			return fmt.Errorf("%s: step %d takes %s, which is not enabled at '%s'", machine, i+1, step.Action.Value, state)
		}
		// Build lists a machine's transitions row for row with its table.
		tr := mm.Transitions[j]
		n := slices.IndexFunc(mm.Table.Rows[j].Results, func(r umpiremodel.Result) bool {
			return r.Outcome == step.Outcome.Value && r.State == step.State.Value && slices.Equal(r.Facts, atomValues(step.Facts))
		})
		if n < 0 {
			return fmt.Errorf("%s: step %d is no result of the row '%s'", machine, i+1, tr.Row)
		}
		if last, err = s.step(w, mu, tr.Source, tr.Steps[n], step.State.Value); err != nil {
			return err
		}
		mu, state = last.Mu, step.State.Value
	}
	if !last.Read[k] || !last.Viol[k] {
		return fmt.Errorf("%s: the monitor %s is not violated on the last step of the witness, where it is in the state %s",
			machine, monitor, last.Mu[k])
	}
	return nil
}

func atomValues(atoms []umpiremodel.Atom) []string {
	out := make([]string, len(atoms))
	for i, a := range atoms {
		out[i] = a.Value
	}
	return out
}
