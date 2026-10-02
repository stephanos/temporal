package checker

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strconv"
	"strings"
)

// Canonical encodings and Behavior Fingerprints, byte-compatible with the Lean ones in
// model/lean/Umpire/Model/Canonical.lean, Scenario/Check.lean, Property/Check.lean and Query/Check.lean.
// A fingerprint is "sha256:" and the hex SHA-256 of a domain line and the canonical content
// (`Umpire.Fingerprint.derive`).

// Fingerprint is the Behavior Fingerprint of already-canonical content.
func Fingerprint(canonical string) string {
	sum := sha256.Sum256([]byte("umpire.behavior-fingerprint/v1\n" + canonical))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// quote is Lean.Json.compress of a string: JSON escaping without HTML escaping.
func quote(s string) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	return strings.TrimSuffix(b.String(), "\n")
}

func jsonArray(items []string) string { return "[" + strings.Join(items, ",") + "]" }

func quoted(items []string) []string {
	out := make([]string, len(items))
	for i, s := range items {
		out[i] = quote(s)
	}
	return out
}

// sortedUnique is `Canonical.canonicalStrings`: sorted, duplicates removed.
func sortedUnique(items []string) []string {
	out := slices.Clone(items)
	slices.Sort(out)
	return slices.Compact(out)
}

// Identities a declared machine owns besides its catalog members (`Umpire.Command.Authoring`).
func (t *Table) capabilityID() string { return t.Family.ID("capability", t.owner(), "transitions") }
func (t *Table) providerID() string   { return t.Family.ID("provider", t.owner(), "finite-table") }
func (t *Table) lawID() string        { return t.Family.ID("law", t.owner(), "canonical-table") }
func (t *Table) kernelID() string     { return t.Family.ID("kernel", t.owner(), "planner") }

// RoleID is the operation role a Scenario's setup binds, named after the machine's entity.
func (t *Table) RoleID() string { return t.Family.ID("role", t.owner(), t.Entity) }

type declaration struct{ id, kind string }

// meanings are the catalog members a machine's provider gives meaning to: states and state fields,
// actions, outcomes and facts.
func (t *Table) meanings() []declaration {
	ids := t.IDs()
	var out []declaration
	for _, s := range ids.States {
		out = append(out, declaration{s, "state"})
	}
	for _, f := range ids.StateFields {
		out = append(out, declaration{f[1], "state"})
	}
	for _, a := range ids.Actions {
		out = append(out, declaration{a, "action"})
	}
	for _, o := range ids.Outcomes {
		out = append(out, declaration{o, "outcome"})
	}
	for _, f := range ids.Facts {
		out = append(out, declaration{f, "fact"})
	}
	return sortDeclarations(out)
}

func sortDeclarations(ds []declaration) []declaration {
	slices.SortStableFunc(ds, func(a, b declaration) int {
		if c := strings.Compare(a.id, b.id); c != 0 {
			return c
		}
		return strings.Compare(a.kind, b.kind)
	})
	return ds
}

func meaningsJSON(ms []declaration) string {
	items := make([]string, len(ms))
	for i, m := range ms {
		items[i] = `{"id":` + quote(m.id) + `,"kind":` + quote(m.kind) + `,"behaviorVersion":` + quote(m.id) + `}`
	}
	return jsonArray(items)
}

// TargetSemantic is `Canonical.targetSemanticJson` for a declared machine.
func (t *Table) TargetSemantic() string {
	decls := append(t.meanings(),
		declaration{t.Family.Target(t.owner()), "target"},
		declaration{t.kernelID(), "machine"},
		declaration{t.capabilityID(), "capability"},
		declaration{t.lawID(), "law"},
		declaration{t.providerID(), "provider"})
	for _, r := range t.Rows {
		decls = append(decls, declaration{t.Family.ID("relation", t.owner(), r.Key), "relation"})
	}
	decls = sortDeclarations(decls)
	declJSON := make([]string, len(decls))
	for i, d := range decls {
		declJSON[i] = `{"id":` + quote(d.id) + `,"kind":` + quote(d.kind) + `,"version":1,"behaviorVersion":` +
			quote(d.id) + `}`
	}
	provider := `{"id":` + quote(t.providerID()) + `,"capabilityId":` + quote(t.capabilityID()) +
		`,"capabilityVersion":1,"behaviorVersion":` + quote(t.capabilityID()) +
		`,"meanings":` + meaningsJSON(t.meanings()) +
		`,"laws":[{"id":` + quote(t.lawID()) + `,"body":` + quote(t.lawID()) + `}]}`
	return `{"id":` + quote(t.Family.Target(t.owner())) +
		`,"declarations":` + jsonArray(declJSON) +
		`,"requiredCapabilities":` + jsonArray([]string{quote(t.capabilityID())}) +
		`,"providers":` + jsonArray([]string{provider}) +
		`,"connectors":[]` +
		`,"kernel":{"id":` + quote(t.kernelID()) + `,"version":1}` +
		`,"behavior":` + t.behaviorJSON() + `}`
}

// TargetFingerprint is the machine's Behavior Fingerprint.
func (t *Table) TargetFingerprint() string { return Fingerprint(t.TargetSemantic()) }

type behaviorRow struct {
	prior, action, outcome, state string
	facts                         []string
}

func compareBehaviorRows(a, b behaviorRow) int {
	for _, c := range []int{strings.Compare(a.prior, b.prior), strings.Compare(a.action, b.action),
		strings.Compare(a.outcome, b.outcome), strings.Compare(a.state, b.state)} {
		if c != 0 {
			return c
		}
	}
	return slices.Compare(a.facts, b.facts)
}

// behaviorJSON is `targetBehaviorDescriptionJson`: sorted domains, the initial states under the
// machine's one setup, and every row result sorted as the derived `Ord` on the row orders it.
func (t *Table) behaviorJSON() string {
	var rows []behaviorRow
	for _, r := range t.Rows {
		for _, res := range r.Results {
			rows = append(rows, behaviorRow{r.Source, r.Action, res.Outcome, res.State, res.Facts})
		}
	}
	slices.SortStableFunc(rows, compareBehaviorRows)
	rows = slices.CompactFunc(rows, func(a, b behaviorRow) bool { return compareBehaviorRows(a, b) == 0 })
	transitions := make([]string, len(rows))
	for i, r := range rows {
		transitions[i] = `{"priorState":` + quote(r.prior) + `,"action":` + quote(r.action) +
			`,"outcome":` + quote(r.outcome) + `,"state":` + quote(r.state) +
			`,"facts":` + jsonArray(quoted(r.facts)) + `}`
	}
	setup := t.SetupKey()
	initial := make([]string, 0, len(t.Starts))
	for _, s := range sortedUnique(t.Starts) {
		initial = append(initial, `{"setup":`+quote(setup)+`,"state":`+quote(s)+`}`)
	}
	return `{"domains":{"setups":` + jsonArray([]string{quote(setup)}) +
		`,"states":` + jsonArray(quoted(sortedUnique(t.States))) +
		`,"actions":` + jsonArray(quoted(sortedUnique(t.Actions))) +
		`,"outcomes":` + jsonArray(quoted(sortedUnique(t.Outcomes))) +
		`,"observations":` + jsonArray(quoted(sortedUnique(t.Facts))) + `}` +
		`,"initialStates":` + jsonArray(initial) +
		`,"transitions":` + jsonArray(transitions) + `}`
}

// SetupKey is the machine's one setup, spelled as the Lean `starts:` line names the start: the
// start state's first field.
func (t *Table) SetupKey() string {
	start := t.Starts[0]
	first, _, _ := strings.Cut(start, "-")
	return first
}

// ScenarioSemantic is `behaviorSemanticJson` for a pinned Scenario.
func (s *ScenarioDecl) ScenarioSemantic(t *Table) string {
	actionID := func(key string) string { return t.Family.ID("action", t.owner(), key) }
	occurrence := func(n int) string { return t.Family.ID("occurrence", s.Name, strconv.Itoa(n)) }
	var allowed []string
	for _, a := range s.Actions {
		allowed = append(allowed, actionID(a))
	}
	allowed = sortedUnique(allowed)
	var occurrences, ordering, exactly []string
	counts := map[string]int{}
	for i, a := range s.Actions {
		occurrences = append(occurrences, `{"id":`+quote(occurrence(i+1))+`,"action":`+quote(actionID(a))+`}`)
		exactly = append(exactly, quote(actionID(a)))
		counts[actionID(a)]++
		if i > 0 {
			ordering = append(ordering, `{"before":`+quote(occurrence(i))+`,"after":`+quote(occurrence(i+1))+`}`)
		}
	}
	var bounds []string
	for _, a := range allowed {
		n := strconv.Itoa(counts[a])
		bounds = append(bounds, `{"action":`+quote(a)+`,"minimum":`+n+`,"maximum":`+n+`}`)
	}
	start := s.Start
	return `{"id":` + quote(s.ScenarioID(t)) +
		`,"version":1,"requires":` + jsonArray([]string{quote(t.capabilityID())}) +
		`,"roles":[{"id":` + quote(t.RoleID()) + `,"valueKind":"state"}]` +
		`,"setup":[{"id":` + quote(t.Family.ID("setup", s.Name, t.Entity)) + `,"relation":"equal","left":{"role":` +
		quote(t.RoleID()) + `},"right":{"value":{"identity":` + quote(t.Family.ID("state", t.owner(), start)) +
		`,"value":` + quote(start) + `}}}]` +
		`,"allowedActions":` + jsonArray(quoted(allowed)) +
		`,"requiredOccurrences":` + jsonArray(occurrences) +
		`,"forbiddenActions":[]` +
		`,"occurrenceBounds":` + jsonArray(bounds) +
		`,"ordering":` + jsonArray(ordering) +
		`,"sequences":[],"adjacencies":[]` +
		`,"actionsExactly":` + jsonArray(exactly) +
		`,"traceExactly":null,"spaceStatus":"unclassified"}`
}

// ScenarioID is the Scenario's Definition ID.
func (s *ScenarioDecl) ScenarioID(t *Table) string { return string(t.Family) + ".behavior." + s.Name }

// PropertyID is the Property's Definition ID.
func (p *PropertyDecl) PropertyID(t *Table) string { return string(t.Family) + ".property." + p.Name }

// patternJSON is one clause pattern: a trace field, the value it references and the constraint.
func patternJSON(field, reference, value string) string {
	return `{"field":` + quote(field) + `,"reference":` + quote(reference) +
		`,"constraint":{"kind":"equals","value":` + quote(value) + `}}`
}

// clauseJSON encodes one lowered same-step clause: a fact is an input-output clause, a state or an
// outcome a transition-contract clause.
func (t *Table) clauseJSON(propertyID, action string, r Requirement) string {
	id := propertyID + "." + r.Label
	trigger := patternJSON("selected-action", t.Family.ID("action", t.owner(), action), action)
	switch r.Kind {
	case FactRequirement:
		return `{"id":` + quote(id) + `,"kind":"input-output","input":` + trigger +
			`,"output":` + patternJSON("observation", t.Family.ID("fact", t.owner(), r.Value), r.Value) + `}`
	case OutcomeRequirement:
		return `{"id":` + quote(id) + `,"kind":"transition-contract","precondition":` + trigger +
			`,"postcondition":` + patternJSON("outcome", t.Family.ID("outcome", t.owner(), r.Value), r.Value) + `}`
	default:
		return `{"id":` + quote(id) + `,"kind":"transition-contract","precondition":` + trigger +
			`,"postcondition":` + patternJSON("resulting-state", t.Family.ID("state", t.owner(), r.Value), r.Value) + `}`
	}
}

// PropertySemantic is `propertySemanticJson` for a same-step Property lowered to its clauses.
func (t *Table) PropertySemantic(propertyID string, groups []Group) string {
	var clauses []string
	var ids []string
	byID := map[string]string{}
	for _, g := range groups {
		for _, r := range g.Requirements {
			id := propertyID + "." + r.Label
			ids = append(ids, id)
			byID[id] = t.clauseJSON(propertyID, g.Trigger, r)
		}
	}
	slices.Sort(ids)
	for _, id := range ids {
		clauses = append(clauses, byID[id])
	}
	return t.propertyHeader(propertyID) + `,"logicalTimeSource":null,"clauses":` + jsonArray(clauses) + `}`
}

// Quote is Lean.Json.compress of a string.
func Quote(s string) string { return quote(s) }

// PropertyHeaderJSON is a Property semantic string up to its meanings: the part every Property of
// the machine shares.
func (t *Table) PropertyHeaderJSON(propertyID string) string { return t.propertyHeader(propertyID) }

func (t *Table) propertyHeader(propertyID string) string {
	return `{"id":` + quote(propertyID) + `,"version":1,"requires":` + jsonArray([]string{quote(t.capabilityID())}) +
		`,"capabilities":[{"id":` + quote(t.capabilityID()) + `,"version":1,"behaviorVersion":` +
		quote(t.capabilityID()) + `}]` +
		`,"meanings":` + meaningsJSON(t.meanings())
}

// QueryCanonical is the Query's canonical form (`Umpire.Query.Check`), which its fingerprint hashes
// whole.
func (q *Query) QueryCanonical(t *Table, propertyFingerprint string) string {
	form := "find"
	if q.Form == VerifyForm {
		form = "verify"
	}
	start := q.Scenario.Start
	role := `[[{"role":` + quote(t.RoleID()) + `,"value":{"definitionId":` +
		quote(t.Family.ID("state", t.owner(), start)) + `,"value":` + quote(start) + `}}]]`
	var actions []string
	for _, a := range t.Actions {
		actions = append(actions, `{"definitionId":`+quote(t.Family.ID("action", t.owner(), a))+`,"value":`+quote(a)+`}`)
	}
	roleFP := Fingerprint("query-role-domain/v1\n" + role)
	actionFP := Fingerprint("query-action-domain/v1\n" + jsonArray(actions))
	limit := func(v int, unit string) string {
		return `{"value":` + strconv.Itoa(v) + `,"unit":` + quote(unit) + `}`
	}
	return `{"id":` + quote(string(t.Family)+".query."+q.Name) + `,"version":1,"form":` + quote(form) +
		`,"properties":[{"id":` + quote(q.Property.PropertyID(t)) + `,"behaviorFingerprint":` + quote(propertyFingerprint) + `}]` +
		`,"behavior":{"id":` + quote(q.Scenario.ScenarioID(t)) + `,"behaviorFingerprint":` +
		quote(Fingerprint(q.Scenario.ScenarioSemantic(t))) + `}` +
		`,"limits":{"steps":` + limit(q.Limits.Steps, "steps") + `,"actions":` + limit(q.Limits.Actions, "actions") +
		`,"search":` + limit(q.Limits.Search, "search") + `}` +
		`,"policy":{"strategy":"shortest","seed":17}` +
		`,"target":{"id":` + quote(t.Family.Target(t.owner())) + `,"behaviorFingerprint":` + quote(t.TargetFingerprint()) +
		`,"composition":[` + quote(t.capabilityID()) + `,` + quote(t.providerID()) + `],"kernel":{"id":` +
		quote(t.kernelID()) + `}}` +
		`,"finiteCompleteness":{"roleDomainFingerprint":` + quote(roleFP) + `,"actionDomainFingerprint":` +
		quote(actionFP) + `}}`
}
