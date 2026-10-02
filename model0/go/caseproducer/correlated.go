package caseproducer

import (
	"slices"
	"strconv"
	"strings"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/model/go/umpire"
)

// clause is one lowered requirement as an operation-correlated clause: from the operation's first
// selected action, the required value is due within as many transitions as the Scenario places
// between them.
type clause struct {
	id       string
	response pattern
	bound    int
}

// pattern is a clause pattern: the trace field, the value it references, and the value it equals.
type pattern struct {
	field     string // selected-action, resulting-state, outcome or observation
	reference string
	value     string
}

func (c pattern) json() string {
	return `{"field":` + quote(c.field) + `,"reference":` + quote(c.reference) +
		`,"constraint":{"kind":"equals","value":` + quote(c.value) + `}}`
}

// holds reports whether one taken step already carries the pattern's value.
func (c pattern) holds(s step) bool {
	carries := func(a umpire.Atom) bool { return a.ID == c.reference && a.Value == c.value }
	switch c.field {
	case "selected-action":
		return carries(s.Action)
	case "resulting-state":
		return carries(s.State)
	case "outcome":
		return carries(s.Outcome)
	case "observation":
		return slices.ContainsFunc(s.Facts, carries)
	default:
		return false
	}
}

// stepField is the correlated step field a pattern's trace field reads.
func (c pattern) stepField() testpilotspb.CorrelatedStepField {
	switch c.field {
	case "selected-action":
		return testpilotspb.CORRELATED_STEP_FIELD_ACTION
	case "outcome":
		return testpilotspb.CORRELATED_STEP_FIELD_OUTCOME
	case "resulting-state":
		return testpilotspb.CORRELATED_STEP_FIELD_STATE
	default:
		return testpilotspb.CORRELATED_STEP_FIELD_FACT
	}
}

// condition is the step condition a pattern lowers to: its step reference compared equal with the
// text it requires.
func (c pattern) condition() *testpilotspb.Expression {
	return equal(correlatedStep(c.stepField(), c.reference), literal(text(c.value)))
}

// scopedClauses places every lowered clause by the Scenario: its trigger is the operation's first
// action, its bound the position of the clause's own action in the pinned schedule. A clause whose
// value the trace already carries before its action would answer without the action being observed,
// so it rejects. Clauses come in the checked Property's order, which is by clause id.
func (p *production) scopedClauses(groups []umpire.Group) ([]clause, error) {
	propertyID := p.q.Property.PropertyID(p.t)
	var out []clause
	for _, g := range groups {
		bound := slices.Index(p.schedule, p.t.ActionAtom(g.Trigger).ID)
		if bound < 0 {
			return nil, reject(propertyID, "property.clause-occurrence")
		}
		for _, r := range g.Requirements {
			id := propertyID + "." + r.Label
			var resp pattern
			switch r.Kind {
			case umpire.FactRequirement:
				resp = pattern{"observation", p.t.FactAtom(r.Value).ID, r.Value}
			case umpire.OutcomeRequirement:
				resp = pattern{"outcome", p.t.OutcomeAtom(r.Value).ID, r.Value}
			case umpire.StateRequirement:
				resp = pattern{"resulting-state", p.t.StateAtom(r.Value).ID, r.Value}
			default:
				return nil, reject(id, "property.clause-shape")
			}
			for _, s := range p.steps[:bound] {
				if resp.holds(s) {
					return nil, reject(id, "property.clause-early-response")
				}
			}
			out = append(out, clause{id: id, response: resp, bound: bound})
		}
	}
	if len(out) == 0 {
		return nil, reject(propertyID, "property.clauses.absent")
	}
	slices.SortStableFunc(out, func(a, b clause) int { return strings.Compare(a.id, b.id) })
	return out, nil
}

func (p *production) trigger() pattern {
	return pattern{"selected-action", p.opening.ID, p.opening.Value}
}

// correlatedPropertySemantic is the semantic string of the Property the Case carries: no same-step
// clauses, and one correlated rule per placed clause (`correlatedRuleJson`).
func (p *production) correlatedPropertySemantic(propertyID string, clauses []clause) string {
	var rules []string
	for _, c := range clauses {
		rules = append(rules, `{"id":`+quote(c.id)+`,"kind":"correlated-eventually-within/v1","trigger":`+
			p.trigger().json()+`,"response":`+c.response.json()+
			`,"scope":[`+quote(p.r.ScopeField)+`],"key":`+quote(p.r.OperationKey)+
			`,"clock":"operation-transitions","bound":`+strconv.Itoa(c.bound)+`,"ending":"partial"}`)
	}
	return p.t.PropertyHeaderJSON(propertyID) + `,"logicalTimeSource":null,"clauses":[],"correlatedRules":[` +
		strings.Join(rules, ",") + `]}`
}

// projectionPlan is the checked projection: its rules sorted by kind, its canonical fingerprint, and
// the rows an operation can take under the actions its rules confirm.
type projectionPlan struct {
	rules       []resolvedRule
	sources     []string
	fingerprint string
	transitions []projectedRow
}

type projectedRow struct {
	prior  umpire.Atom
	result step
}

// projection is `Case.Projection.check` over the declaration the evidence rules make. A kind carried
// off the path is a rule that confirms nothing.
func (p *production) projection(rules []resolvedRule, offPath []*EvidenceSource) projectionPlan {
	sorted := slices.Clone(rules)
	for _, s := range offPath {
		sorted = append(sorted, resolvedRule{Rule: EvidenceRule{Source: s}})
	}
	var sources []string
	for _, r := range sorted {
		if !slices.Contains(sources, r.Rule.Source.SourceID) {
			sources = append(sources, r.Rule.Source.SourceID)
		}
	}
	slices.Sort(sources)
	slices.SortStableFunc(sorted, func(a, b resolvedRule) int {
		return strings.Compare(a.Rule.Source.KindID, b.Rule.Source.KindID)
	})
	return projectionPlan{rules: sorted, sources: sources,
		fingerprint: umpire.Fingerprint(p.projectionCanonical(sorted, sources)),
		transitions: p.projectedRows(sorted)}
}

// projectionCanonical is the array the projection's fingerprint hashes. Each rule is its kind, the
// fields the kind retains, each with its path, its type and the identity it names, and what its
// evidence means: the steps it confirms, or nothing.
func (p *production) projectionCanonical(sorted []resolvedRule, sources []string) string {
	var ruleJSON []string
	for _, r := range sorted {
		var confirmed []string
		for _, s := range r.Steps {
			confirmed = append(confirmed, "["+quote(s.Action.Value)+","+quote(s.State.Value)+","+quote(s.Outcome.Value)+
				","+jsonArray(atomValues(s.Facts))+"]")
		}
		var fields []string
		for _, f := range r.Rule.Source.Fields {
			// The type is spelled by its name in the protocol, which no rendering of the Go value changes.
			fields = append(fields, jsonArray([]string{quote(f.ID), quote(f.Path), quote(testpilotspb.ScalarKind_name[int32(f.Type)]), quote("retain"), quote(f.Role)}))
		}
		meaning := `["confirmed",null,` + jsonArray(confirmed) + "]"
		if len(r.Steps) == 0 {
			meaning = `["irrelevant"]`
		}
		ruleJSON = append(ruleJSON, "["+quote(r.Rule.Source.KindID)+","+jsonArray(fields)+","+meaning+"]")
	}
	l := p.r.ProjectionLimits
	// The limits are numbers written into the array as they are, not strings.
	limits := []string{strconv.FormatInt(l.Events, 10), strconv.FormatInt(l.Buffered, 10),
		strconv.FormatInt(l.Keys, 10), strconv.FormatInt(l.Support, 10),
		strconv.FormatInt(l.Work, 10), strconv.FormatInt(l.EventSize, 10)}
	return jsonArray([]string{quote("checked-projection/v2"), quote(p.r.ProjectionID),
		quote(p.t.TargetFingerprint()), quote(p.t.SetupKey()), quote(p.initial.Value),
		jsonArray([]string{quote(p.r.ScopeField)}), quote(p.r.OperationKey),
		jsonArray(quotedAll(sources)), jsonArray(ruleJSON), jsonArray(limits)})
}

// projectedRows is every row an operation can take: from the start, under the actions the rules
// confirm, and the states those reach.
func (p *production) projectedRows(sorted []resolvedRule) []projectedRow {
	var relevant []string
	for _, r := range sorted {
		for _, s := range r.Steps {
			if !slices.Contains(relevant, s.Action.Value) {
				relevant = append(relevant, s.Action.Value)
			}
		}
	}
	var rows []projectedRow
	for _, prior := range p.reachableUnder(relevant) {
		for _, action := range relevant {
			for _, res := range p.resultsOf(prior, action) {
				rows = append(rows, projectedRow{p.t.StateAtom(prior), res})
			}
		}
	}
	return rows
}

// reachableUnder is the states an operation reaches from the start by these actions, breadth-first,
// each frontier in the order the states were first reached (`reachableStates`).
func (p *production) reachableUnder(actions []string) []string {
	visited := []string{p.initial.Value}
	for frontier := visited; len(frontier) > 0; {
		var next []string
		for _, state := range frontier {
			for _, action := range actions {
				for _, res := range p.resultsOf(state, action) {
					if !slices.Contains(next, res.State.Value) && !slices.Contains(visited, res.State.Value) {
						next = append(next, res.State.Value)
					}
				}
			}
		}
		visited = append(visited, next...)
		frontier = next
	}
	return visited
}

// correlatedContract is `Umpire.Case.Correlated.lower`'s wire form.
func (p *production) correlatedContract(plan projectionPlan, clauses []clause) *testpilotspb.CorrelatedContract {
	c := &testpilotspb.CorrelatedContract{
		ProjectionId:          p.r.ProjectionID,
		ProjectionFingerprint: plan.fingerprint,
		EvidenceObservationId: p.r.CorrelatedObservation,
		ScopeFields:           []string{p.r.ScopeField},
		OperationField:        p.r.OperationKey,
		Sources:               plan.sources,
		InitialState:          modelValue(p.initial),
		InitialStateFields:    p.fields(p.initial.Value),
	}
	for _, row := range plan.transitions {
		t := p.output(row.result)
		t.PriorState = modelValue(row.prior)
		t.PriorFields = p.fields(row.prior.Value)
		c.Transitions = append(c.Transitions, t)
	}
	for _, r := range plan.rules {
		rule := &testpilotspb.CorrelatedProjectionRule{Kind: r.Rule.Source.KindID, Meaning: testpilotspb.CORRELATED_EVIDENCE_MEANING_CONFIRMED}
		if len(r.Steps) == 0 {
			rule.Meaning = testpilotspb.CORRELATED_EVIDENCE_MEANING_IRRELEVANT
		}
		for _, s := range r.Steps {
			rule.Outputs = append(rule.Outputs, p.output(s))
		}
		for _, f := range r.Rule.Source.Fields {
			rule.Fields = append(rule.Fields, &testpilotspb.CorrelatedFieldPolicy{FieldId: f.ID, Type: &testpilotspb.ScalarType{Kind: f.Type},
				Disposition: testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN})
		}
		c.ProjectionRules = append(c.ProjectionRules, rule)
	}
	for _, cl := range clauses {
		c.Rules = append(c.Rules, &testpilotspb.CorrelatedRule{
			RuleId: cl.id, Clock: testpilotspb.CORRELATED_CLOCK_OPERATION_TRANSITIONS, Bound: int64(cl.bound),
			Ending: testpilotspb.TRACE_ENDING_PARTIAL, Trigger: p.trigger().condition(), Response: cl.response.condition()})
	}
	return c
}

func (p *production) output(s step) *testpilotspb.CorrelatedTransition {
	t := &testpilotspb.CorrelatedTransition{Action: modelValue(s.Action), State: modelValue(s.State),
		Outcome: modelValue(s.Outcome), StateFields: p.fields(s.State.Value)}
	for _, f := range s.Facts {
		t.Facts = append(t.Facts, modelValue(f))
	}
	return t
}

func (p *production) fields(state string) []*testpilotspb.ModelValue {
	var out []*testpilotspb.ModelValue
	for _, a := range p.t.FieldValues(state) {
		out = append(out, modelValue(a))
	}
	return out
}

func modelValue(a umpire.Atom) *testpilotspb.ModelValue {
	return &testpilotspb.ModelValue{DefinitionId: a.ID, Value: a.Value}
}

func atomValues(atoms []umpire.Atom) []string {
	out := make([]string, len(atoms))
	for i, a := range atoms {
		out[i] = quote(a.Value)
	}
	return out
}

func quotedAll(items []string) []string {
	out := make([]string, len(items))
	for i, s := range items {
		out[i] = quote(s)
	}
	return out
}

func jsonArray(items []string) string { return "[" + strings.Join(items, ",") + "]" }

func quote(s string) string { return umpire.Quote(s) }
