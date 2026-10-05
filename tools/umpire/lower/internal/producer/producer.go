// Package producer lowers one checked Go Model Query into a Testpilot Case through a named
// realization. One checked Model, one selected witness and one realization become one Case: the
// Program is the realization's scaffolding with the path's actions placed where the realization
// binds them, and the Contract is the correlated capability the Property's clauses, placed by the
// Scenario, lower to.
//
// Orders, identities and spellings are fixed, because the Case bytes are compared with the
// checked-in fixtures.
//
// A realization may also declare the fields a kind keeps, a kind every Case carries because its
// source is exhaustive, the Run's own record and a read of one message as sources, and the steps of a
// path a kind confirms, which is how a class a path takes more than once is confirmed step by step. A
// realization that declares none of it produces the bytes it did before these declarations existed.
package producer

import (
	"fmt"
	"slices"
	"strings"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
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

// IdentityFor is the identity of a set's Query: the Case ID
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

// suffix is the suffix an instance's ids carry: none on a Case over one instance.
func (p Placement) suffix() string {
	if p.Count <= 1 {
		return ""
	}
	return fmt.Sprintf("-%d", p.Number)
}

// Recorded is where one evidence kind is read from: one arm of the history event's attributes, the
// elements of a repeated field a unary RPC returns, the one message at a path of what it returns, or
// the Run's own record.
type Recorded struct {
	HistoryAttributes string
	Method, Path      string
	// Single reads the one message at Path, where a read otherwise takes each element of a repeated
	// field.
	Single bool
	// RunEvent is the Run's own record, where the kind is read from no response.
	RunEvent *RunEvent
}

// RunEvent is the Run Events of one kind that one instruction of an entrypoint records, where Guard
// holds of the event's payload.
type RunEvent struct {
	Kind                        testpilotspb.RunEventKind
	EntrypointID, InstructionID string
	// RunKeyed keys the evidence by the Run's own ID, and the kind then names no operation key path.
	RunKeyed bool
	Guard    *testpilotspb.Expression
}

// EvidenceField is one field evidence of a kind retains: where it is read in the recorded data and
// the scalar it is. Role is the identity the realization says the field names. No part of a Case
// states it, so it reaches a Case in the fingerprint of its projection alone.
type EvidenceField struct {
	ID, Path string
	Type     testpilotspb.ScalarKind
	Role     string
}

// Taking is one step of a path: the Occurrence-th step, counted from one, of the class Key names.
type Taking struct {
	Key        string
	Occurrence int
}

// EvidenceSource is what a realization knows about one admitted evidence kind.
type EvidenceSource struct {
	EventKind        string
	Recorded         Recorded
	OperationKeyPath string
	KindID, SourceID string
	// Fields is the fields evidence of the kind retains, in declaration order.
	Fields []EvidenceField
	// Exhaustive says the kind's source reports every occurrence of what it records, so every Case
	// carries the kind, on its path or off it: only a kind a Case carries can be seen to be absent.
	Exhaustive bool
	// Confirms names the steps of a path one piece of evidence of this kind confirms. A kind that
	// names steps confirms those and no other, all of them or none; a kind that names none confirms
	// the one step of a path that records what the kind records.
	Confirms []Taking
}

func (s *EvidenceSource) readsHistory() bool { return s.Recorded.HistoryAttributes != "" }

// EvidenceRule is one resolved (action, source) pair: the action a recorded event confirms, and how
// to read it.
type EvidenceRule struct {
	Action umpiremodel.Atom
	Source *EvidenceSource
}

// ReadsHistory reports whether the rule reads a history event, the only kind a history read lifts.
func (r EvidenceRule) ReadsHistory() bool { return r.Source.readsHistory() }

// step is one taken step as model values: its action and result.
type step struct {
	Action  umpiremodel.Atom
	State   umpiremodel.Atom
	Outcome umpiremodel.Atom
	Facts   []umpiremodel.Atom
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

func (Fixed) isItem()      {}
func (WhenOnPath) isItem() {}
func (Actions) isItem()    {}

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
	// RequiredSettings are the dynamic-configuration settings the realized system must run under, in
	// the order the realization declares them.
	RequiredSettings []*testpilotspb.RequiredSetting
	// InstructionDefaults are the limits an instruction that writes none takes, and RunOrderIsCausal
	// whether the Run's record order orders one operation's evidence across sources, as the
	// realization's API behavior declares them.
	InstructionDefaults *testpilotspb.InstructionLimits
	RunOrderIsCausal    bool
}

// ActionBinding is what one action class is realized as. Key is the class key a Scenario spells;
// Action is the Definition ID the realization states, read where the key names no class.
type ActionBinding struct {
	Action        string
	Key           string
	InstructionID string
	Node          func(Placement, string) *testpilotspb.InstructionNode
}

func (b ActionBinding) resolve(t *umpiremodel.Table) string {
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
	// Target is the Behavior Fingerprint of a table, when the caller holds it already.
	Target Fingerprinted
}

// Fingerprinted is a table with its Behavior Fingerprint. A caller that lowers many Queries of one
// bound table computes it once; a production of a Query on any other table computes its own.
type Fingerprinted struct {
	Table       *umpiremodel.Table
	Fingerprint string
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
func Produce(q *umpiremodel.Query, identity Identity, r *Realization, source Source) (*testpilotspb.Case, error) {
	p, err := newProduction(q, identity, r, source)
	if err != nil {
		return nil, err
	}
	return p.produce()
}

// production is one Case being produced.
type production struct {
	q        *umpiremodel.Query
	t        *umpiremodel.Table
	identity Identity
	r        *Realization
	source   Source
	answer   umpiremodel.Answer
	steps    []step
	initial  umpiremodel.Atom
	opening  umpiremodel.Atom
	schedule []string // the pinned schedule's action ids, in trace order
	// fingerprint is the table's Behavior Fingerprint, once it has been read.
	fingerprint string
}

// targetFingerprint is the table's Behavior Fingerprint: the caller's, when it is of this table, and
// otherwise computed once for the production, which names it three times.
func (p *production) targetFingerprint() string {
	switch {
	case p.fingerprint != "":
	case p.r.Target.Table == p.t && p.r.Target.Fingerprint != "":
		p.fingerprint = p.r.Target.Fingerprint
	default:
		p.fingerprint = p.t.TargetFingerprint()
	}
	return p.fingerprint
}

func newProduction(q *umpiremodel.Query, identity Identity, r *Realization, source Source) (*production, error) {
	t, err := q.Scenario.Machine.Table()
	if err != nil {
		return nil, err
	}
	a, err := q.Answer()
	if err != nil {
		return nil, err
	}
	if a.Outcome != umpiremodel.Outcome(umpiremodel.Found) || a.Witness == nil {
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
func Preflight(q *umpiremodel.Query, identity Identity, r *Realization) error {
	p, err := newProduction(q, identity, r, Source{})
	if err != nil {
		return err
	}
	_, err = p.decide()
	return err
}

// Confirmation is the steps of a Query's path one kind of evidence confirms, each by its place on the
// path, counted from zero, in path order: the steps that record nothing among them.
type Confirmation struct {
	Source *EvidenceSource
	Steps  []int
}

// Confirmations is which kind of evidence confirms each step of a Query's path, as Produce decides
// it and in path order. It refuses what Produce refuses of the path's evidence.
func Confirmations(q *umpiremodel.Query, identity Identity, r *Realization) ([]Confirmation, error) {
	p, err := newProduction(q, identity, r, Source{})
	if err != nil {
		return nil, err
	}
	resolved, _, err := p.resolveEvidence(p.derivedEvidence())
	if err != nil {
		return nil, err
	}
	var out []Confirmation
	at := 0
	for _, rule := range resolved {
		confirmed := Confirmation{Source: rule.Rule.Source}
		for range rule.Steps {
			confirmed.Steps = append(confirmed.Steps, at)
			at++
		}
		out = append(out, confirmed)
	}
	return out, nil
}

// decided is what a production decides before it writes a Case.
type decided struct {
	silentGaps          []*testpilotspb.KnownGap
	groups              []umpiremodel.Group
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
	propertyFingerprint := umpiremodel.Fingerprint(p.correlatedPropertySemantic(propertyID, clauses))
	offPath := p.offPath(evidenceRules)
	plan := p.projection(evidenceRules, offPath)
	contract := p.correlatedContract(plan, clauses)
	var rules []EvidenceRule
	for _, r := range evidenceRules {
		rules = append(rules, r.Rule)
	}
	// A kind carried off the path is read like the path's own, by a rule that confirms no action.
	for _, s := range offPath {
		rules = append(rules, EvidenceRule{Source: s})
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
	queryFingerprint := umpiremodel.Fingerprint(p.q.QueryCanonicalOf(p.t, umpiremodel.Fingerprint(propertySemantic), p.targetFingerprint()))
	provenance := &testpilotspb.CaseProvenance{
		ProducerId:      p.r.ProducerID,
		ProducerVersion: p.r.ProducerVersion,
		Definitions: []*testpilotspb.DefinitionBinding{
			{DefinitionId: p.t.IDs().Target, BehaviorFingerprint: p.targetFingerprint(), Kind: testpilotspb.DEFINITION_KIND_TARGET},
			{DefinitionId: p.q.Scenario.ScenarioID(p.t), BehaviorFingerprint: umpiremodel.Fingerprint(scenarioSemantic),
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

// sourceOf is the kind that confirms the one step of a path that records a fact of this name: the
// kind that records it and names no step of its own.
func (p *production) sourceOf(kind string) (*EvidenceSource, bool) {
	for _, s := range p.r.Sources {
		if s.EventKind == kind && len(s.Confirms) == 0 {
			return s, true
		}
	}
	return nil, false
}

// recorded reports whether some kind of the realization records a fact of this name.
func (p *production) recorded(kind string) bool {
	return slices.ContainsFunc(p.r.Sources, func(s *EvidenceSource) bool { return s.EventKind == kind })
}

// confirming is the one decision of which kind of evidence confirms a step of a path, in its one
// order. A kind that names the step confirms it, and two that name it are refused, as is one that
// records none of the step's facts. Any other step is confirmed by the kind that records the first
// fact its class is named by, among the kinds that name no step. A step whose class records facts
// that only kinds naming other steps record has no evidence of its own, and is refused: nothing is
// confirmed by a kind that says it confirms something else. A step whose class records nothing
// evidence names is confirmed by no kind, which is no error.
//
// names is the evidence names the step's class records on the path, in the order the path meets them,
// and facts the evidence names the step itself records.
func confirming(sources []*EvidenceSource, taking Taking, action string, names, facts []string) (*EvidenceSource, error) {
	var named *EvidenceSource
	for _, s := range sources {
		if !slices.Contains(s.Confirms, taking) {
			continue
		}
		if named != nil {
			return nil, reject(s.KindID, "evidence.taking-ambiguous")
		}
		named = s
	}
	if named != nil {
		if !slices.Contains(facts, named.EventKind) {
			return nil, reject(named.KindID, "evidence.taking-crossed")
		}
		return named, nil
	}
	for _, name := range names {
		for _, s := range sources {
			if s.EventKind == name && len(s.Confirms) == 0 {
				return s, nil
			}
		}
	}
	if len(names) > 0 {
		return nil, reject(action, "evidence.taking-unrecorded")
	}
	return nil, nil
}

// evidenceNames is the evidence names the facts of a step are recorded under, in fact order.
func (p *production) evidenceNames(facts []umpiremodel.Atom) []string {
	var out []string
	for _, f := range facts {
		for _, line := range p.t.Evidence {
			if f.Value == line[0] || strings.HasPrefix(f.Value, line[0]+"-") {
				out = append(out, line[1])
				break
			}
		}
	}
	return out
}

// resolveEvidence resolves each mapping against the realization's admitted kinds and walks the
// witness: an observed step's rule confirms the silent steps before it together with its own, and
// each silent step becomes a Known Gap. A step is confirmed by the kind confirming gives it. A kind
// that names several steps confirms them by one piece of evidence, so its rule takes each of them as
// the path reaches it, with the silent steps between, and no other kind's evidence lies between them;
// and it confirms every step it names or none, since its evidence is of all of them.
func (p *production) resolveEvidence(mappings [][2]string) ([]resolvedRule, []*testpilotspb.KnownGap, error) {
	names := map[string][]string{}
	for _, m := range mappings {
		if !p.recorded(m[1]) {
			return nil, nil, reject(m[1], "evidence.kind-unknown")
		}
		if !slices.Contains(p.schedule, m[0]) {
			return nil, nil, reject(m[0], "evidence.action-unselected")
		}
		names[m[0]] = append(names[m[0]], m[1])
	}
	var resolved []resolvedRule
	var silent []step
	var gaps []*testpilotspb.KnownGap
	taken := map[string]int{}
	for _, s := range p.steps {
		taken[s.Action.Value]++
		source, err := confirming(p.r.Sources, Taking{Key: s.Action.Value, Occurrence: taken[s.Action.Value]}, s.Action.ID,
			names[s.Action.ID], p.evidenceNames(s.Facts))
		if err != nil {
			return nil, nil, err
		}
		if source == nil {
			silent = append(silent, s)
			if !slices.ContainsFunc(gaps, func(g *testpilotspb.KnownGap) bool { return g.GetSubject() == s.Action.ID }) {
				gaps = append(gaps, silentGap(s.Action))
			}
			continue
		}
		if resolved, err = claim(resolved, source, silent, s); err != nil {
			return nil, nil, err
		}
		silent = nil
	}
	if len(silent) > 0 {
		return nil, nil, reject(silent[0].Action.ID, "evidence.action-unmapped")
	}
	for _, r := range resolved {
		for _, named := range r.Rule.Source.Confirms {
			if taken[named.Key] < named.Occurrence {
				return nil, nil, reject(r.Rule.Source.KindID, "evidence.confirmation-partial")
			}
		}
	}
	return resolved, gaps, nil
}

// claim gives a step, with the silent steps before it, to the rule of the kind that confirms it. A
// kind no rule has yet opens one. A kind that names no step confirms one step of a path, so a second
// step of it is refused: as a class taken again where the first was of the same class, and as a kind
// two steps would share where it was not. A kind that names its steps takes the step into its rule,
// which must be the last one: its one piece of evidence confirms its steps with nothing between.
func claim(resolved []resolvedRule, source *EvidenceSource, silent []step, s step) ([]resolvedRule, error) {
	claimed := slices.IndexFunc(resolved, func(r resolvedRule) bool { return r.Rule.Source == source })
	switch {
	case claimed < 0:
		return append(resolved, resolvedRule{EvidenceRule{s.Action, source}, append(silent, s)}), nil
	case len(source.Confirms) == 0 && resolved[claimed].Rule.Action.ID == s.Action.ID:
		return nil, reject(s.Action.ID, "evidence.action-repeated")
	case len(source.Confirms) == 0:
		return nil, reject(s.Action.ID, "evidence.kind-repeated")
	case claimed != len(resolved)-1:
		return nil, reject(source.KindID, "evidence.kind-interrupted")
	default:
		resolved[claimed].Steps = append(append(resolved[claimed].Steps, silent...), s)
		return resolved, nil
	}
}

// offPath is the exhaustive kinds no rule of the path reads, in declaration order: a Case carries
// them with the path's own, and gives them no meaning.
func (p *production) offPath(rules []resolvedRule) []*EvidenceSource {
	var out []*EvidenceSource
	for _, s := range p.r.Sources {
		if s.Exhaustive && !slices.ContainsFunc(rules, func(r resolvedRule) bool { return r.Rule.Source.KindID == s.KindID }) {
			out = append(out, s)
		}
	}
	return out
}

// silentGap is the Known Gap a silent step records: the Contract infers it from the evidence of the
// step after it.
func silentGap(action umpiremodel.Atom) *testpilotspb.KnownGap {
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
// kind its facts record, so a Run that took it reads as a violation rather than as nothing. The
// resolved rules hold the path's steps in path order, each rule its own, so the steps before one in
// its rule are the ones another result of its row is confirmed with.
func (p *production) alternativeRules(resolved []resolvedRule) ([]resolvedRule, error) {
	rules := slices.Clone(resolved)
	prior := p.initial
	for _, r := range resolved {
		for i, s := range r.Steps {
			added, err := p.alternativesOf(rules, prior, s, r.Steps[:i])
			if err != nil {
				return nil, err
			}
			rules = append(rules, added...)
			prior = s.State
		}
	}
	return rules, nil
}

// alternativesOf is one rule per other result of the witnessed row taken from prior, each confirming
// the silent steps before it together with its own.
func (p *production) alternativesOf(rules []resolvedRule, prior umpiremodel.Atom, taken step, before []step) ([]resolvedRule, error) {
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

func (p *production) firstKind(facts []umpiremodel.Atom) (string, bool) {
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

func (p *production) stepOf(action string, res umpiremodel.Result) step {
	s := step{Action: p.t.ActionAtom(action), State: p.t.StateAtom(res.State), Outcome: p.t.OutcomeAtom(res.Outcome),
		Facts: []umpiremodel.Atom{}}
	for _, f := range res.Facts {
		s.Facts = append(s.Facts, p.t.FactAtom(f))
	}
	return s
}
