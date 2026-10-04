package model

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpire "go.temporal.io/server/tools/umpire/model/internal/checker"
)

// Scope is the finite scope a check of a Model runs within. A receipt carries the bounds it ran
// under, and a result past one of them says so rather than read as exhaustive.
type Scope struct {
	// Ceilings bound interpreting the Model's machines.
	Ceilings Ceilings
	// Compose bounds building each composition.
	Compose ComposeCeiling
	// Progress bounds each progress claim's check: the IR gives a claim only its `within`.
	Progress Limits
	// QuerySearch, when above 0, is the most product states a Query's search may visit where its own
	// Limits allow more.
	QuerySearch int
}

// DefaultScope is the scope Check runs within when a caller names none of its own.
var DefaultScope = Scope{
	Ceilings: defaultCeilings,
	Compose:  ComposeCeiling{States: 1 << 16, Evaluations: 1 << 20, Results: 1 << 20},
	Progress: Limits{Name: "default", Steps: 1 << 10, Search: 1 << 20},
}

// ReceiptKind is what a receipt says of the declaration it is about. The kinds are kept apart so that
// what was not checked, what was cut short and what was left unknown never read as a result.
type ReceiptKind string

const (
	// AdmissionError is a Model a reader rejects before any check (SEMANTICS.md, Admission).
	AdmissionError ReceiptKind = "admission-error"
	// DeclarationError is a declaration that cannot be read as the IR says it is: a claim function
	// that returns no Boolean, a monitor state outside its domain, a Query whose parts do not fit.
	DeclarationError ReceiptKind = "declaration-error"
	// ResourceLimit is work a ceiling of the scope refused before any of it was done.
	ResourceLimit ReceiptKind = "resource-limit"
	// LimitReached is a search its Search limit cut: it proves nothing.
	LimitReached ReceiptKind = "limit-reached"
	// Unresolved is a progress check its Steps limit left open: a prefix is no counterexample.
	Unresolved ReceiptKind = "unresolved"
	// RefinementRejected is a refinement that does not hold.
	RefinementRejected ReceiptKind = "refinement-rejected"
	// Counterexample is a violation with a witness that replays.
	Counterexample ReceiptKind = "counterexample"
	// Verified is a claim that holds on everything the scope reaches, with nothing left unknown.
	Verified ReceiptKind = "verified-within-limits"
	// Found is a find's witness, which replays.
	Found ReceiptKind = "found"
	// NotFound is a find with no witness within its Limits and nothing left unknown.
	NotFound ReceiptKind = "not-found"
	// Incomplete is a result that found no violation or witness and read a hole on the way.
	Incomplete ReceiptKind = "incomplete"
	// Unsupported is a declaration this reader does not check.
	Unsupported ReceiptKind = "unsupported"
	// ReplayFailed is a result whose witness did not replay through a fresh interpretation, or whose
	// two readings disagree: an error, whatever the result said.
	ReplayFailed ReceiptKind = "replay-failed"
)

// Subject is the kind of declaration a receipt is about.
type Subject string

const (
	// ModelSubject is the Model as a whole, which admission rejects.
	ModelSubject Subject = "model"
	// MachineSubject is a machine that could not be interpreted, whose claims and compositions could
	// not be checked for it.
	MachineSubject     Subject = "machine"
	RefinementSubject  Subject = "refinement"
	CompositionSubject Subject = "composition"
	QuerySubject       Subject = "query"
	ProgressSubject    Subject = "progress"
)

// ClaimKey names a declaration by the family and the machine or composition it belongs to and its
// own name there. Two machines of one family may each declare a claim of one name, which share a
// Definition ID (`PropertyDecl.PropertyID`) and stay two keys.
type ClaimKey struct {
	Family string
	Owner  string
	Name   string
}

// HoleEdge is where a check met a hole.
type HoleEdge string

const (
	// RowHole is a hole row: a state and class whose steps are unknown.
	RowHole HoleEdge = "row"
	// ClaimHole is a step a Property's or a Monitor's function could not be read on, or a state a
	// progress claim's could not.
	ClaimHole HoleEdge = "claim"
	// DeclarationHole is a hole reached outside any step: in a start, an `ends`, a refinement's map or
	// what it names visible.
	DeclarationHole HoleEdge = "declaration"
)

// HoleReach is one hole a check read: the declared hole, or none for a value no case matches, the
// row it was met at, and for a search the shortest path it took there.
type HoleReach struct {
	Edge     HoleEdge
	ID       string
	Name     string
	Row      string
	Position string
	Depth    int
	Prefix   *Trace
}

// ResourceBound is a ceiling some work needed more than.
type ResourceBound struct {
	Resource string
	Ceiling  int64
	Needed   int64
	Overflow bool
}

// Receipt is what checking one declaration of a Model established, and within what.
type Receipt struct {
	Subject Subject
	Key     ClaimKey
	// Part is the kind of violation a progress claim's receipt is about, which are reported apart.
	Part        ProgressKind
	Kind        ReceiptKind
	Position    string
	Explanation string

	// Target and Fingerprint identify the table the check read: its Definition ID and its Behavior
	// Fingerprint, which no source position enters.
	Target      string
	Fingerprint string
	// Property and Scenario are a Query's claims.
	Property ClaimKey
	Scenario ClaimKey

	// Limits are the limits the check ran within, and Bound the ceiling that refused it.
	Limits Limits
	Bound  *ResourceBound
	// Assumptions names what the result relies on.
	Assumptions []string

	// Explored counts the check's work: the product states a search visited, the units a progress
	// check spent, the rows a refinement read, the states a composition reached. Expanded is the
	// product states a search read the successors of, and Exercised reports that the claim was read on
	// some step or state, so a verified result is not vacuous.
	Explored  int
	Expanded  int
	Exercised bool
	// TableRows is how many rows the table the check read holds, whatever part of them it read.
	TableRows int
	// Holes are the holes the check read. They leave a result with no witness incomplete and take
	// nothing from one with a witness.
	Holes []HoleReach

	Witness *Trace
	Rows    []string
	// Loop is the index of the state a lasso witness returns to, or -1.
	Loop           int
	ProductWitness *Trace
	Failure        RefinementFailure
	Monitor        string
	Monitors       []MonitorVerdict
	// Cause is the error a receipt that is no result reports.
	Cause error
	// Also holds the results folded into this one whose witnesses it does not carry itself.
	Also []Receipt
}

// precedence orders the kinds for a receipt made of several results, the highest first: an error is
// an error whatever else was found, a violation stands before a limit, a limit before a hole, and
// what was not checked before what held. A find's witness is realized, so it stands before a result a
// hole leaves incomplete, whatever that hole hides.
var precedence = [][]ReceiptKind{
	{ReplayFailed, AdmissionError, DeclarationError},
	{RefinementRejected, Counterexample},
	{ResourceLimit, LimitReached, Unresolved},
	{Found},
	{Incomplete},
	{Unsupported},
	{Verified, NotFound},
}

func (k ReceiptKind) rank() int {
	return slices.IndexFunc(precedence, func(level []ReceiptKind) bool { return slices.Contains(level, k) })
}

// fold makes one receipt of the results of several checks of one declaration: the members of a
// composition, a search and the refinement it reads through, a result and the replay of its
// witnesses. It is the one place a result is weighed against another, so that none hides another.
//
// The receipt is the result of the highest precedence, the first of several, whole. It lists every
// hole of every result, in their order, and Also holds every other result whose witness it does not
// carry. A claim found to hold, or found nowhere, is not established once any of them read a hole: it
// is incomplete, by all of the holes. A found witness is not so undone: it stays found beside a result
// a hole leaves incomplete, with that result's holes listed and its witness kept in Also.
func fold(parts ...Receipt) Receipt {
	winner := 0
	for i, p := range parts {
		if p.Kind.rank() < parts[winner].Kind.rank() {
			winner = i
		}
	}
	out := parts[winner]
	out.Holes = nil
	for i, p := range parts {
		for _, h := range p.Holes {
			if !slices.Contains(out.Holes, h) {
				out.Holes = append(out.Holes, h)
			}
		}
		unkept := p.Witness != nil && p.Witness != out.Witness || p.ProductWitness != nil && p.ProductWitness != out.ProductWitness
		if i != winner && unkept {
			out.Also = append(out.Also, p)
		}
	}
	if (out.Kind == Verified || out.Kind == NotFound) && len(out.Holes) > 0 {
		out.Kind = Incomplete
	}
	if out.Kind == Incomplete {
		out.Explanation = unknownBecause(out.Holes)
	}
	return out
}

// Report is the receipts of one Model under one scope, ordered by subject and key, so the order the
// IR lists its declarations in does not show.
type Report struct {
	Scope    Scope
	Receipts []Receipt
}

// Check admits a Model, interprets it within the scope, and checks every refinement, composition,
// Query and progress claim it declares with its private checker (internal/checker), whose algorithms it adds
// nothing to. Every witness is replayed against a second interpretation before its receipt is given.
func Check(m *umpirespb.Model, scope Scope) *Report {
	return check(m, scope, m)
}

// check is Check with the Model its witnesses are replayed against, which Check gives the Model
// itself, interpreted afresh.
func check(m *umpirespb.Model, scope Scope, replay *umpirespb.Model) *Report {
	return checkWithBinding(m, scope, replay, nil)
}

func checkWithBinding(m *umpirespb.Model, scope Scope, replay *umpirespb.Model, first *binding) *Report {
	r := &Report{Scope: scope}
	if err := Validate(m); err != nil {
		for _, problem := range problems(err) {
			x := receipt(ModelSubject, ClaimKey{}, nil)
			x.Kind, x.Cause, x.Explanation = AdmissionError, problem, problem.Error()
			var located *Error
			if errors.As(problem, &located) {
				x.Position = located.Position
			}
			r.Receipts = append(r.Receipts, x)
		}
		return r
	}
	if first == nil {
		first = bind(m, scope)
	}
	c := newCheckerWithBinding(first, replay)
	for _, mm := range m.GetMachines() {
		if err := c.first.failed[mm.GetName()]; err != nil {
			key := ClaimKey{Family: mm.GetFamily(), Owner: mm.GetName()}
			r.Receipts = append(r.Receipts, c.failed(receipt(MachineSubject, key, mm.GetPosition()), err))
		}
		if mm.GetRefines() != nil {
			r.Receipts = append(r.Receipts, c.refinement(mm))
		}
	}
	for _, composition := range m.GetCompositions() {
		if x, checked := c.composition(composition); checked {
			r.Receipts = append(r.Receipts, x)
		}
	}
	for _, q := range m.GetQueries() {
		r.Receipts = append(r.Receipts, c.query(q))
	}
	for _, p := range m.GetProgress() {
		r.Receipts = append(r.Receipts, c.progress(p)...)
	}
	subjects := []Subject{ModelSubject, MachineSubject, RefinementSubject, CompositionSubject, QuerySubject, ProgressSubject}
	slices.SortStableFunc(r.Receipts, func(x, y Receipt) int {
		return cmp.Or(cmp.Compare(slices.Index(subjects, x.Subject), slices.Index(subjects, y.Subject)),
			strings.Compare(x.Key.Family, y.Key.Family), strings.Compare(x.Key.Owner, y.Key.Owner),
			strings.Compare(x.Key.Name, y.Key.Name))
	})
	return r
}

// problems is every problem admission reported, each on its own.
func problems(err error) []error {
	var joined interface{ Unwrap() []error }
	if errors.As(err, &joined) {
		return joined.Unwrap()
	}
	return []error{err}
}

func receipt(subject Subject, key ClaimKey, at *umpirespb.Position) Receipt {
	return Receipt{Subject: subject, Key: key, Position: where(at), Loop: -1}
}

// checker gives one Model's receipts: first is the interpretation its checks read, and again a
// second one of the replay Model, made when the first witness needs it.
type checker struct {
	scope  Scope
	first  *binding
	replay *umpirespb.Model
	fresh  *binding
	// prints holds each table's Behavior Fingerprint, and holes each declared hole's name by id.
	prints map[*Table]string
	holes  map[string]string
}

// newChecker interprets an admitted Model for checking.
func newChecker(m *umpirespb.Model, scope Scope, replay *umpirespb.Model) *checker {
	return newCheckerWithBinding(bind(m, scope), replay)
}

func newCheckerWithBinding(first *binding, replay *umpirespb.Model) *checker {
	c := &checker{scope: first.scope, replay: replay, first: first, prints: map[*Table]string{}, holes: map[string]string{}}
	for _, h := range first.model.GetHoles() {
		c.holes[h.GetId()] = h.GetName()
	}
	return c
}

func (c *checker) again() *binding {
	if c.fresh == nil {
		c.fresh = bind(c.replay, c.scope)
	}
	return c.fresh
}

// reads names the table a receipt's check read, and the assumptions the table carries.
func (c *checker) reads(r Receipt, t *Table) Receipt {
	if _, ok := c.prints[t]; !ok {
		c.prints[t] = t.TargetFingerprint()
	}
	r.Target, r.Fingerprint, r.TableRows = t.Family.Target(t.OwnerName()), c.prints[t], len(t.Rows)
	r.Assumptions = nil
	for _, a := range t.Assumptions {
		r.Assumptions = append(r.Assumptions, a.Name)
	}
	return r
}

// failed is the receipt of a check that gave no answer: what it could not do is told apart, and none
// of it reads as a result. A hole it stopped at was reached outside any step: one inside a claim's
// function is unknown evidence of a search or a progress check, which goes on past it.
func (c *checker) failed(r Receipt, err error) Receipt {
	r.Cause, r.Explanation = err, err.Error()
	var (
		unsupported *unsupportedError
		composition *ComposeLimitError
		limit       *LimitError
		refinement  *unrefined
		accepted    *unseen
		members     *unbuiltMembers
		hole        *Hole
	)
	switch {
	case errors.As(err, &members):
		// Each member that failed is a result of its own, and the composition's is the fold of them.
		parts := make([]Receipt, len(members.errs))
		for i, reason := range members.errs {
			parts[i] = c.failed(r, reason)
		}
		return fold(parts...)
	case errors.As(err, &unsupported):
		r.Kind = Unsupported
	case errors.As(err, &composition):
		r.Kind = ResourceLimit
		r.Bound = &ResourceBound{Resource: composition.Resource, Ceiling: composition.Ceiling, Needed: composition.Needed,
			Overflow: composition.Overflow}
	case errors.As(err, &limit):
		r.Kind = ResourceLimit
		r.Bound = &ResourceBound{Resource: limit.Resource, Ceiling: limit.Ceiling, Needed: limit.Needed, Overflow: limit.Overflow}
	case errors.As(err, &refinement):
		return c.unrefined(r, refinement)
	case errors.As(err, &accepted):
		// The generic check accepted every row of the refining table before the holes left it unknown.
		r = c.reads(r, accepted.source.table)
		r.Kind, r.Explored, r.Holes = Incomplete, len(accepted.source.table.Rows), c.declared(accepted.holes)
	case errors.As(err, &hole):
		r.Kind, r.Holes = Incomplete, c.declared(holesIn(err))
	default:
		r.Kind = DeclarationError
	}
	return fold(r)
}

// unrefined is the receipt of a refinement the generic check does not accept: rejected, with the
// rule it breaks, or left unknown by the reachable holes of the refining machine. Its witnesses are
// replayed against the refining and the refined machine's tables.
// Whichever declaration met the refinement, its own receipt, a Query that reads through it or a
// composition it lets down, the check that failed read the refining machine's table: the receipt
// names that table and its assumptions.
// The generic check reaches a machine's reachable holes only after every row refines, so a refinement
// they leave unknown read every row. It does not say how many rows it read before a start or a row it
// rejects, so a rejection counts none.
func (c *checker) unrefined(r Receipt, e *unrefined) Receipt {
	rejected, machine := e.rejected, e.source.machine
	r = c.reads(r, e.source.table)
	r.Explored = 0
	r.Witness, r.ProductWitness = rejected.Witness, rejected.ProductWitness
	r.Holes = c.declared(e.unread)
	if rejected.Kind == umpire.RefinementIncomplete {
		r.Kind, r.Explored = Incomplete, len(e.source.table.Rows)
		r.Holes = append(c.reachableHoles(machine), r.Holes...)
	} else {
		r.Kind, r.Failure = RefinementRejected, rejected.Kind
	}
	var err error
	if r.Witness != nil {
		err = c.replays(machine.Decl.GetName(), r.Witness)
	}
	if r.ProductWitness != nil {
		err = errors.Join(err, c.replays(machine.Decl.GetRefines().GetProduct(), r.ProductWitness))
	}
	return c.replayed(r, err)
}

// replays replays a witness against a subject's table of the second interpretation.
func (c *checker) replays(name string, witness *Trace) error {
	s := c.again().subject(name)
	if s.err != nil {
		return s.err
	}
	return s.table.Replay(witness)
}

// replayed is a result with the replay of its witnesses: an error, whatever it said, when any of them
// did not replay.
func (c *checker) replayed(r Receipt, errs ...error) Receipt {
	return fold(append([]Receipt{r}, notReplayed(fold(r), errors.Join(errs...))...)...)
}

// notReplayed is the result a failed replay of a receipt's witnesses is, to fold with it, and none
// when they replayed. It is the receipt itself, turned an error.
func notReplayed(r Receipt, err error) []Receipt {
	if err == nil {
		return nil
	}
	what := "witness"
	switch r.Kind {
	case Counterexample:
		what = "counterexample"
	case RefinementRejected:
		what = "rejection's witness"
	case Incomplete:
		what = "path to a hole"
	default:
	}
	r.Cause = fmt.Errorf("the %s did not replay: %w", what, err)
	r.Kind, r.Explanation = ReplayFailed, r.Cause.Error()
	return []Receipt{r}
}

// holesIn is every hole an error holds, in the order it lists them.
func holesIn(err error) []*Hole {
	var hole *Hole
	var joined interface{ Unwrap() []error }
	var out []*Hole
	switch {
	case errors.As(err, &joined):
		for _, e := range joined.Unwrap() {
			out = append(out, holesIn(e)...)
		}
	case errors.As(err, &hole):
		out = append(out, hole)
	default:
	}
	return out
}

// declared lists holes reached outside any step.
func (c *checker) declared(holes []*Hole) []HoleReach {
	var out []HoleReach
	for _, h := range holes {
		out = append(out, c.hole(DeclarationHole, h))
	}
	return out
}

// hole is a hole an error reached, named as the Model declares it.
func (c *checker) hole(edge HoleEdge, cause error) HoleReach {
	out := HoleReach{Edge: edge}
	var hole *Hole
	if errors.As(cause, &hole) {
		out.ID, out.Name, out.Position = hole.ID, c.holes[hole.ID], hole.Position
	}
	return out
}

// reachableHoles is a machine's hole rows that its starts reach: what leaves its refinement unknown.
func (c *checker) reachableHoles(mm *Machine) []HoleReach {
	var out []HoleReach
	for _, h := range mm.reachableHoles() {
		reach := c.hole(RowHole, h.Hole)
		reach.Row = h.Row
		out = append(out, reach)
	}
	return out
}

// explored is the holes a search or a progress check read, with the path it took to each.
func (c *checker) explored(unknown []UnknownReach) []HoleReach {
	var out []HoleReach
	for _, u := range unknown {
		edge := RowHole
		if u.Kind == umpire.UnknownClaim {
			edge = ClaimHole
		}
		reach := c.hole(edge, u.Cause)
		reach.Row, reach.Depth, reach.Prefix = u.Row, u.Depth, u.Prefix
		out = append(out, reach)
	}
	return out
}

// unknownBecause says which holes leave a result incomplete, by name and row and with no position,
// so that moving a Model's sources changes no receipt but in where it points.
func unknownBecause(holes []HoleReach) string {
	parts := make([]string, len(holes))
	for i, h := range holes {
		name := "an undeclared hole"
		if h.Name != "" {
			name = "the hole " + h.Name
		}
		switch h.Edge {
		case RowHole:
			parts[i] = fmt.Sprintf("%s at the row '%s'", name, h.Row)
		case ClaimHole:
			parts[i] = fmt.Sprintf("%s in a claim's function", name)
			switch {
			case h.Row != "":
				parts[i] += fmt.Sprintf(" on the row '%s'", h.Row)
			case h.Prefix != nil:
				// A progress claim is read at a state, the one the path to the hole ends in.
				at := h.Prefix.Initial
				if n := len(h.Prefix.Steps); n > 0 {
					at = h.Prefix.Steps[n-1].State
				}
				parts[i] += fmt.Sprintf(" at the state '%s'", at.Value)
			default:
			}
		default:
			parts[i] = name + " outside any step"
		}
	}
	return "the result is incomplete: the check read " + strings.Join(parts, ", and ")
}

// refinement is the receipt of a refining machine's declared refinement, checked by RefineTables.
func (c *checker) refinement(decl *umpirespb.Machine) Receipt {
	s := c.first.subject(decl.GetName())
	r := receipt(RefinementSubject, ClaimKey{Family: s.family, Owner: s.name, Name: decl.GetRefines().GetProduct()}, s.at)
	if s.err != nil {
		return c.failed(r, s.err)
	}
	r = c.reads(r, s.table)
	if checked := c.first.refinement(s); checked.err != nil {
		return c.failed(r, checked.err)
	}
	r.Kind, r.Explored = Verified, len(s.table.Rows)
	return r
}

// composition is the receipt of a composition: why it could not be built, or, for one a member of
// which replaces a machine, that every replacement holds. A composition that builds and replaces
// nothing checks nothing, and has no receipt.
func (c *checker) composition(decl *umpirespb.Composition) (Receipt, bool) {
	s := c.first.subject(decl.GetName())
	r := receipt(CompositionSubject, ClaimKey{Family: s.family, Owner: s.name}, s.at)
	if s.err != nil {
		return c.failed(r, s.err), true
	}
	replaces := slices.ContainsFunc(decl.GetMembers(), func(m *umpirespb.Member) bool { return m.GetReplaces() != "" })
	r = c.reads(r, s.table)
	r.Kind, r.Explored = Verified, len(s.table.States)
	return r, replaces
}

// query is a Query's receipt: the generic search's answer, with the holes it explored.
func (c *checker) query(q *umpirespb.Query) Receipt {
	on := c.first.subject(q.GetScenario().GetMachine())
	r := receipt(QuerySubject, ClaimKey{Family: on.family, Owner: q.GetScenario().GetMachine(), Name: q.GetName()}, q.GetPosition())
	r.Property, r.Scenario, r.Limits = c.claim(q.GetProperty()), c.claim(q.GetScenario()), c.first.limits(q)
	bound, err := c.first.query(q)
	if err != nil {
		return c.failed(r, err)
	}
	r = c.reads(r, bound.table)
	a, err := bound.q.Answer()
	if err != nil {
		return c.failed(r, err)
	}
	r.Explored, r.Expanded, r.Exercised = a.Explored, a.Expanded, a.Exercised
	r.Witness, r.Rows, r.Monitor, r.Monitors, r.Explanation = a.Witness, a.Rows, a.Monitor, a.Monitors, a.Explanation
	r.Holes = c.explored(a.Unknown)
	switch a.Outcome {
	case umpire.Found:
		r.Kind = Found
	case umpire.CounterexampleFound:
		r.Kind = Counterexample
	case umpire.LimitReached:
		r.Kind = LimitReached
	case umpire.NotFound:
		r.Kind = NotFound
	default:
		r.Kind = Verified
	}
	// The refinement the Property is read through may itself be left unknown, by the machine's
	// reachable holes or by a hole in what it names visible: the search's result, incomplete by them.
	var parts []Receipt
	if bound.through != nil {
		unknown := r
		unknown.Kind, unknown.Holes = Incomplete, c.declared(bound.through.unread)
		if bound.through.incomplete != nil {
			unknown.Holes = append(c.reachableHoles(on.machine), unknown.Holes...)
		}
		if len(unknown.Holes) > 0 {
			parts = append(parts, unknown)
		}
	}
	parts = append(parts, r)
	if a.Witness != nil || len(a.Unknown) > 0 {
		parts = append(parts, notReplayed(fold(r), c.replayQuery(q, a))...)
	}
	return fold(parts...)
}

// replayQuery replays an answer's witness, and the path to each hole it explored, against the same
// Query declared over a second interpretation.
func (c *checker) replayQuery(q *umpirespb.Query, a Answer) error {
	bound, err := c.again().query(q)
	if err != nil {
		return err
	}
	if a.Witness != nil {
		err = errors.Join(bound.q.Replay(a), bound.table.Replay(a.Witness))
	}
	for _, u := range a.Unknown {
		err = errors.Join(err, bound.table.Replay(u.Prefix))
	}
	return err
}

// claim keys a Property or a Scenario by the family and name of the machine or composition it is
// declared on, and its name there.
func (c *checker) claim(ref *umpirespb.ClaimRef) ClaimKey {
	return ClaimKey{Family: c.first.subject(ref.GetMachine()).family, Owner: ref.GetMachine(), Name: ref.GetName()}
}

// progress is a progress claim's receipts: one for each kind of violation, which are reported apart,
// or one for a claim that could not be checked.
func (c *checker) progress(p *umpirespb.Progress) []Receipt {
	s := c.first.subject(p.GetMachine())
	r := receipt(ProgressSubject, ClaimKey{Family: s.family, Owner: p.GetMachine(), Name: p.GetName()}, p.GetPosition())
	r.Limits = c.scope.Progress
	claim, _, err := c.first.progress(p)
	if err != nil {
		return []Receipt{c.failed(r, err)}
	}
	r = c.reads(r, s.table)
	a, err := umpire.CheckProgress(s.table, claim, c.scope.Progress)
	if err != nil {
		return []Receipt{c.failed(r, err)}
	}
	r.Assumptions, r.Explored, r.Exercised, r.Holes = a.Assumptions, a.Explored, a.From > 0, c.explored(a.Unknown)
	paths := c.replayPaths(p, a.Unknown)
	var out []Receipt
	for _, part := range []struct {
		kind    ProgressKind
		verdict ProgressVerdict
	}{{umpire.DeadlockKind, a.Deadlock}, {umpire.CycleKind, a.Cycle}, {umpire.DeadlineKind, a.Deadline}} {
		x := r
		x.Part, x.Explanation, x.Loop = part.kind, part.verdict.Explanation, part.verdict.Loop
		var witness error
		switch part.verdict.Outcome {
		case umpire.CounterexampleFound:
			x.Kind, x.Witness = Counterexample, part.verdict.Witness
			witness = c.replayProgress(p, part.kind, part.verdict)
		case umpire.Unresolved:
			x.Kind = Unresolved
		case umpire.LimitReached:
			x.Kind = LimitReached
		default:
			x.Kind = Verified
		}
		out = append(out, c.replayed(x, witness, paths))
	}
	return out
}

// replayPaths replays the path a progress check took to each hole it read against the machine's table
// of a second interpretation.
func (c *checker) replayPaths(p *umpirespb.Progress, unknown []UnknownReach) error {
	var err error
	for _, u := range unknown {
		err = errors.Join(err, c.replays(p.GetMachine(), u.Prefix))
	}
	return err
}

// replayProgress replays a violation's witness against the claim declared over a second
// interpretation.
func (c *checker) replayProgress(p *umpirespb.Progress, kind ProgressKind, verdict ProgressVerdict) error {
	claim, s, err := c.again().progress(p)
	if err != nil {
		return err
	}
	return claim.Replay(s.table, kind, verdict)
}
