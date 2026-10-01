// Package caseproducer lowers one checked Go Model Query into a Testpilot Case through a named
// realization, the Go counterpart of model/lean/Umpire/Case/Producer.lean. One checked Model, one
// selected witness and one realization become one Case: the Program is the realization's
// scaffolding with the path's actions placed where the realization binds them, and the Contract is
// the correlated capability the Property's clauses, placed by the Scenario, lower to.
//
// Orders, identities and spellings follow the Lean producer exactly, because the Case bytes are
// compared with the checked-in fixtures Lean renders.
package caseproducer

import (
	"fmt"
	"slices"
	"strings"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/model/go/umpire"
)

// Error is a production failure: the construct that could not be realized and the definition it
// belongs to, as `Umpire.Case.Compiler.Error` names them.
type Error struct {
	Definition string
	Construct  string
}

func (e *Error) Error() string { return e.Definition + ": " + e.Construct }

func reject(definition, construct string) error { return &Error{definition, construct} }

// Identity names one Case. Every other identity derives from the fixture name.
type Identity struct {
	CaseID, Fixture, ProgramID, ContractID, RunScope string
}

// IdentityOf is the identity a fixture name derives under a Case ID root.
func IdentityOf(root, fixture string) Identity {
	id := root + "." + fixture
	return Identity{CaseID: id, Fixture: fixture, ProgramID: id + ".program", ContractID: id + ".contract",
		RunScope: fixture}
}

// IdentityFor is the identity the Lean `case` command gives a set's Query: the Case ID
// `<root>.<set>.<query>`, and the fixture `<set>-<query>` every other identity derives from.
func IdentityFor(root, set, query string) Identity {
	id := root + "." + set + "." + query
	fixture := set + "-" + query
	return Identity{CaseID: id, Fixture: fixture, ProgramID: id + ".program", ContractID: id + ".contract",
		RunScope: fixture}
}

// Placement is where a node lands: the Case, the instance that performs it, and how many there are.
type Placement struct {
	Identity      Identity
	Number, Count int
}

// Suffix is the suffix an instance's ids carry: none on a Case over one instance.
func (p Placement) Suffix() string {
	if p.Count <= 1 {
		return ""
	}
	return fmt.Sprintf("-%d", p.Number)
}

// Recorded is where one evidence kind is read from: one arm of the history event's attributes, or
// the elements of a repeated field a unary RPC returns.
type Recorded struct {
	HistoryAttributes string
	Method, Path      string
}

// EvidenceSource is what a realization knows about one admitted evidence kind.
type EvidenceSource struct {
	EventKind        string
	Recorded         Recorded
	OperationKeyPath string
	KindID, SourceID string
}

func (s *EvidenceSource) readsHistory() bool { return s.Recorded.HistoryAttributes != "" }

// EvidenceRule is one resolved (action, source) pair: the action a recorded event confirms, and how
// to read it.
type EvidenceRule struct {
	Action umpire.Atom
	Source *EvidenceSource
}

// ReadsHistory reports whether the rule reads a history event, the only kind a history read lifts.
func (r EvidenceRule) ReadsHistory() bool { return r.Source.readsHistory() }

// step is one taken step as model values: its action and result.
type step struct {
	Action  umpire.Atom
	State   umpire.Atom
	Outcome umpire.Atom
	Facts   []umpire.Atom
}

func (s step) same(o step) bool {
	return s.Action == o.Action && s.State == o.State && s.Outcome == o.Outcome && slices.Equal(s.Facts, o.Facts)
}

// resolvedRule is an evidence rule and the steps its evidence confirms: the silent steps before its
// own, then its own.
type resolvedRule struct {
	Rule  EvidenceRule
	Steps []step
}

// Item is one item of an entrypoint's instruction sequence.
type Item interface{ isItem() }

// Fixed is a node every Case carries once.
type Fixed struct {
	Node func(Placement, []EvidenceRule) *testpilotspb.InstructionNode
}

// WhenOnPath is a node a Case carries once for each instance whose path performs a class Keys names.
type WhenOnPath struct {
	Keys []string
	Node func(Placement, []EvidenceRule) *testpilotspb.InstructionNode
}

// Actions is where the path's actions of these classes land, in path order.
type Actions struct{ Classes []string }

// PerInstance is these items once per instance.
type PerInstance struct{ Items []Item }

func (Fixed) isItem()       {}
func (WhenOnPath) isItem()  {}
func (Actions) isItem()     {}
func (PerInstance) isItem() {}

// EntrypointPlan is one entrypoint of a realization's Program: how it activates, and its items.
type EntrypointPlan struct {
	Activate    func(Placement, []*testpilotspb.InstructionNode) *testpilotspb.Entrypoint
	Items       []Item
	PerInstance bool
}

// ProgramPlan is everything a realization's Program carries besides its actions.
type ProgramPlan struct {
	Roles         []*testpilotspb.Role
	Slots         []*testpilotspb.Slot
	InstanceSlots func(Placement) []*testpilotspb.Slot
	Observations  []*testpilotspb.Observation
	Entrypoints   []EntrypointPlan
	Cleanup       *testpilotspb.Cleanup
}

// ActionBinding is what one action class is realized as. Key is the class key a Scenario spells;
// Action is the Definition ID the realization states, read where the key names no class.
type ActionBinding struct {
	Action        string
	Key           string
	InstructionID string
	Node          func(Placement, string) *testpilotspb.InstructionNode
}

func (b ActionBinding) resolve(t *umpire.Table) string {
	if b.Key == "" {
		return b.Action
	}
	if slices.Contains(t.Actions, b.Key) {
		return t.ActionAtom(b.Key).ID
	}
	return b.Action
}

// ProjectionLimits are the semantic window of the projection.
type ProjectionLimits struct{ Events, Buffered, Keys, Support, Work, EventSize int64 }

// Realization is the platform-owned binding of a Model to the runtime.
type Realization struct {
	Plan                  ProgramPlan
	Actions               []ActionBinding
	ProducerID            string
	ProducerVersion       string
	ProjectionID          string
	ScopeField            string
	OperationKey          string
	HistoryObservation    string
	CorrelatedObservation string
	Sources               []*EvidenceSource
	ProjectionLimits      ProjectionLimits
}

// Source is where a Model is declared, as Case provenance names it.
type Source struct {
	Path       string
	Provenance string
}

func (s Source) pb() *testpilotspb.SourceLocation {
	return &testpilotspb.SourceLocation{Path: s.Path, Line: 1, Column: 1, Provenance: s.Provenance}
}

// Produce lowers one checked find Query into a Case.
func Produce(q *umpire.Query, identity Identity, r *Realization, source Source) (*testpilotspb.Case, error) {
	p, err := newProduction(q, identity, r, source)
	if err != nil {
		return nil, err
	}
	return p.produce()
}

// production is one Case being produced.
type production struct {
	q        *umpire.Query
	t        *umpire.Table
	identity Identity
	r        *Realization
	source   Source
	answer   umpire.Answer
	steps    []step
	initial  umpire.Atom
	opening  umpire.Atom
	schedule []string // the pinned schedule's action ids, in trace order
}

func newProduction(q *umpire.Query, identity Identity, r *Realization, source Source) (*production, error) {
	t, err := q.Scenario.Machine.Table()
	if err != nil {
		return nil, err
	}
	a, err := q.Answer()
	if err != nil {
		return nil, err
	}
	if a.Outcome != umpire.Found || a.Witness == nil {
		return nil, reject(string(t.Family)+".query."+q.Name, "witness.absent")
	}
	p := &production{q: q, t: t, identity: identity, r: r, source: source, answer: a,
		initial: a.Witness.Initial}
	for _, s := range a.Witness.Steps {
		p.steps = append(p.steps, step{s.Action, s.State, s.Outcome, s.Facts})
	}
	for _, key := range q.Scenario.Actions {
		p.schedule = append(p.schedule, t.ActionAtom(key).ID)
	}
	if len(p.schedule) == 0 {
		return nil, reject(q.Scenario.ScenarioID(t), "behavior.sequence.absent")
	}
	p.opening = t.ActionAtom(q.Scenario.Actions[0])
	return p, nil
}

// Preflight decides everything Produce decides for a Query and writes no Case: that the Query has a
// witness, which evidence confirms each step, which clauses the Property lowers to, and where the
// path's actions land in the Program. It reports what Produce would refuse, by the same steps.
func Preflight(q *umpire.Query, identity Identity, r *Realization) error {
	p, err := newProduction(q, identity, r, Source{})
	if err != nil {
		return err
	}
	_, err = p.decide()
	return err
}

// decided is what a production decides before it writes a Case.
type decided struct {
	silentGaps          []*testpilotspb.KnownGap
	groups              []umpire.Group
	clauses             []clause
	propertyID          string
	propertyFingerprint string
	plan                projectionPlan
	contract            *testpilotspb.CorrelatedContract
	program             *testpilotspb.Program
}

// decide resolves the witness's evidence, lowers the Property and assembles the Program: every step of
// a production that can refuse the Query.
func (p *production) decide() (*decided, error) {
	witnessRules, silentGaps, err := p.resolveEvidence(p.derivedEvidence())
	if err != nil {
		return nil, err
	}
	evidenceRules, err := p.alternativeRules(witnessRules)
	if err != nil {
		return nil, err
	}
	groups, err := p.q.Property.Lower()
	if err != nil {
		return nil, err
	}
	clauses, err := p.scopedClauses(groups)
	if err != nil {
		return nil, err
	}
	propertyID := p.q.Property.PropertyID(p.t)
	propertyFingerprint := umpire.Fingerprint(p.correlatedPropertySemantic(propertyID, clauses))
	plan := p.projection(evidenceRules)
	contract := p.correlatedContract(plan, clauses)
	var rules []EvidenceRule
	for _, r := range evidenceRules {
		rules = append(rules, r.Rule)
	}
	program, err := p.assembleProgram(rules)
	if err != nil {
		return nil, err
	}
	return &decided{silentGaps: silentGaps, groups: groups, clauses: clauses, propertyID: propertyID,
		propertyFingerprint: propertyFingerprint, plan: plan, contract: contract, program: program}, nil
}

func (p *production) produce() (*testpilotspb.Case, error) {
	d, err := p.decide()
	if err != nil {
		return nil, err
	}
	silentGaps, groups, clauses, propertyID := d.silentGaps, d.groups, d.clauses, d.propertyID
	propertyFingerprint, plan, contract, program := d.propertyFingerprint, d.plan, d.contract, d.program
	scenarioSemantic := p.q.Scenario.ScenarioSemantic(p.t)
	propertySemantic := p.t.PropertySemantic(propertyID, groups)
	queryFingerprint := umpire.Fingerprint(p.q.QueryCanonical(p.t, umpire.Fingerprint(propertySemantic)))
	provenance := &testpilotspb.CaseProvenance{
		ProducerId:      p.r.ProducerID,
		ProducerVersion: p.r.ProducerVersion,
		Definitions: []*testpilotspb.DefinitionBinding{
			{DefinitionId: p.t.IDs().Target, BehaviorFingerprint: p.t.TargetFingerprint(), Kind: testpilotspb.DEFINITION_KIND_TARGET},
			{DefinitionId: p.q.Scenario.ScenarioID(p.t), BehaviorFingerprint: umpire.Fingerprint(scenarioSemantic),
				Kind: testpilotspb.DEFINITION_KIND_SCENARIO},
			{DefinitionId: string(p.t.Family) + ".query." + p.q.Name, BehaviorFingerprint: queryFingerprint,
				Kind: testpilotspb.DEFINITION_KIND_QUERY},
			{DefinitionId: propertyID, BehaviorFingerprint: propertyFingerprint, Kind: testpilotspb.DEFINITION_KIND_PROPERTY},
		},
		Sources:   []*testpilotspb.SourceLocation{p.source.pb(), p.source.pb(), p.source.pb(), p.source.pb()},
		KnownGaps: silentGaps,
	}
	for _, c := range clauses {
		provenance.CorrelatedRules = append(provenance.CorrelatedRules, &testpilotspb.CorrelatedRuleBinding{
			RuleId: c.id, PropertyId: propertyID, PropertyFingerprint: propertyFingerprint,
			ProjectionId: p.r.ProjectionID, ProjectionFingerprint: plan.fingerprint, Source: p.source.pb()})
	}
	for _, claim := range p.t.Claims() {
		if slices.Contains(p.schedule, claim.Member) {
			provenance.AbstractionClaims = append(provenance.AbstractionClaims, &testpilotspb.AbstractionClaim{
				Action: claim.Action, Field: claim.Field, ClassName: claim.ClassName, Example: claim.Example})
		}
	}
	c := &testpilotspb.Case{
		CaseId:     p.identity.CaseID,
		Version:    &testpilotspb.FormatVersion{Major: 1},
		Provenance: provenance,
		Program:    program,
		Contract:   &testpilotspb.Contract{ContractId: p.identity.ContractID, Correlated: contract},
	}
	if err := localize(c); err != nil {
		return nil, err
	}
	return c, nil
}

// derivedEvidence is the evidence mappings a witness implies under the machine's own evidence
// lines: one per fact a step records that some line covers, naming the step's action.
func (p *production) derivedEvidence() [][2]string {
	var out [][2]string
	for _, s := range p.steps {
		for _, f := range s.Facts {
			for _, line := range p.t.Evidence {
				if f.Value == line[0] || strings.HasPrefix(f.Value, line[0]+"-") {
					m := [2]string{s.Action.ID, line[1]}
					if !slices.Contains(out, m) {
						out = append(out, m)
					}
					break
				}
			}
		}
	}
	return out
}

func (p *production) sourceOf(kind string) (*EvidenceSource, bool) {
	for _, s := range p.r.Sources {
		if s.EventKind == kind {
			return s, true
		}
	}
	return nil, false
}

// resolveEvidence resolves each mapping against the realization's admitted kinds and walks the
// witness: an observed step's rule confirms the silent steps before it together with its own, and
// each silent step becomes a Known Gap.
func (p *production) resolveEvidence(mappings [][2]string) ([]resolvedRule, []*testpilotspb.KnownGap, error) {
	type admitted struct {
		action string
		source *EvidenceSource
	}
	var admittedRules []admitted
	for _, m := range mappings {
		s, ok := p.sourceOf(m[1])
		if !ok {
			return nil, nil, reject(m[1], "evidence.kind-unknown")
		}
		if !slices.Contains(p.schedule, m[0]) {
			return nil, nil, reject(m[0], "evidence.action-unselected")
		}
		admittedRules = append(admittedRules, admitted{m[0], s})
	}
	var resolved []resolvedRule
	var silent []step
	var gaps []*testpilotspb.KnownGap
	for _, s := range p.steps {
		i := slices.IndexFunc(admittedRules, func(a admitted) bool { return a.action == s.Action.ID })
		if i < 0 {
			silent = append(silent, s)
			if !slices.ContainsFunc(gaps, func(g *testpilotspb.KnownGap) bool { return g.GetSubject() == s.Action.ID }) {
				gaps = append(gaps, silentGap(s.Action))
			}
			continue
		}
		if slices.ContainsFunc(resolved, func(r resolvedRule) bool { return r.Rule.Action.ID == s.Action.ID }) {
			return nil, nil, reject(s.Action.ID, "evidence.action-repeated")
		}
		resolved = append(resolved, resolvedRule{EvidenceRule{s.Action, admittedRules[i].source}, append(silent, s)})
		silent = nil
	}
	if len(silent) > 0 {
		return nil, nil, reject(silent[0].Action.ID, "evidence.action-unmapped")
	}
	return resolved, gaps, nil
}

// silentGap is the Known Gap a silent step records: the Contract infers it from the evidence of the
// step after it.
func silentGap(action umpire.Atom) *testpilotspb.KnownGap {
	return &testpilotspb.KnownGap{
		Kind:            testpilotspb.KNOWN_GAP_KIND_CAPABILITY,
		Code:            action.ID + ".unobserved",
		SubjectPresence: &testpilotspb.KnownGap_Subject{Subject: action.ID},
		DetailPresence: &testpilotspb.KnownGap_Detail{Detail: "the step '" + action.Value + "' records nothing an " +
			"evidence line names, so the Contract infers it from the evidence of the step after it rather " +
			"than observing it"},
	}
}

// alternativeRules adds a rule for every other result of a witnessed row, declared by the first
// kind its facts record, so a Run that took it reads as a violation rather than as nothing.
func (p *production) alternativeRules(resolved []resolvedRule) ([]resolvedRule, error) {
	rules := slices.Clone(resolved)
	prior := p.initial
	var silent []step
	for _, s := range p.steps {
		witness := slices.IndexFunc(rules, func(r resolvedRule) bool { return r.Steps[len(r.Steps)-1].same(s) })
		before := silent
		if witness >= 0 {
			before = rules[witness].Steps[:len(rules[witness].Steps)-1]
		}
		added, err := p.alternativesOf(rules, prior, s, before)
		if err != nil {
			return nil, err
		}
		rules = append(rules, added...)
		if witness >= 0 {
			silent = nil
		} else {
			silent = append(silent, s)
		}
		prior = s.State
	}
	return rules, nil
}

// alternativesOf is one rule per other result of the witnessed row taken from prior, each confirming
// the silent steps before it together with its own.
func (p *production) alternativesOf(rules []resolvedRule, prior umpire.Atom, taken step, before []step) ([]resolvedRule, error) {
	var added []resolvedRule
	for _, res := range p.resultsOf(prior.Value, taken.Action.Value) {
		if res.same(taken) {
			continue
		}
		kind, ok := p.firstKind(res.Facts)
		if !ok {
			continue
		}
		declared := func(r resolvedRule) bool { return r.Rule.Source.EventKind == kind }
		if slices.ContainsFunc(rules, declared) || slices.ContainsFunc(added, declared) {
			return nil, reject(kind, "evidence.kind-ambiguous")
		}
		src, ok := p.sourceOf(kind)
		if !ok {
			return nil, reject(kind, "evidence.kind-unknown")
		}
		added = append(added, resolvedRule{EvidenceRule{taken.Action, src}, append(slices.Clone(before), res)})
	}
	return added, nil
}

func (p *production) firstKind(facts []umpire.Atom) (string, bool) {
	for _, f := range facts {
		for _, line := range p.t.Evidence {
			if f.Value == line[0] || strings.HasPrefix(f.Value, line[0]+"-") {
				return line[1], true
			}
		}
	}
	return "", false
}

// resultsOf is every result the machine gives an action from a state, as taken steps.
func (p *production) resultsOf(state, action string) []step {
	var out []step
	for _, r := range p.t.RowsFrom(state) {
		if r.Action != action {
			continue
		}
		for _, res := range r.Results {
			out = append(out, p.stepOf(action, res))
		}
	}
	return out
}

func (p *production) stepOf(action string, res umpire.Result) step {
	s := step{Action: p.t.ActionAtom(action), State: p.t.StateAtom(res.State), Outcome: p.t.OutcomeAtom(res.Outcome),
		Facts: []umpire.Atom{}}
	for _, f := range res.Facts {
		s.Facts = append(s.Facts, p.t.FactAtom(f))
	}
	return s
}
