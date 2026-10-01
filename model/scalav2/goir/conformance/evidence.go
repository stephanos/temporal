package conformance

import (
	"fmt"
	"slices"
	"strings"

	modelirspb "go.temporal.io/server/api/modelir/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/proto"
)

// This file is the one place a Run's recorded evidence is read. Everything after it sees an
// observation: which instance of the machine it is about, which fact it reports, which attempt and
// delivery the fact belongs to, and what it is ordered after.
//
// The instance is the operation key under the Run's one scope, and the order is the evidence's
// parents and source ordinals; both are text of the correlated evidence today. The attempt and the
// delivery are typed roles of an observation that nothing can fill yet: a realization declares the
// field that scopes a Run and the field that names an operation, and no field for an attempt or a
// delivery, so a Case whose evidence retains any other field is refused when the factory is prepared
// rather than read with that field ignored. When the Run protocol's typed attempt identity is in
// reach, `observed` is where it is read into `attempt` and `delivery`, and that refusal is where a
// declared role lifts it. Nothing else of the package changes.

// kind is one kind of evidence the Case carries, as the realization declares it.
type kind struct {
	// local is the name the Case spells the kind and its source by.
	local, source string
	records       string
	// fields is the fields the Case declares for the kind. None of them is retained.
	fields []*testpilotspb.CorrelatedFieldPolicy
}

// role is one typed correlation role of an observation: known when the evidence names it.
type role struct {
	known bool
	id    string
}

// agrees reports whether two observations may be of one step as far as this role goes: they do unless
// both name it and name it differently.
func (r role) agrees(other role) bool { return !r.known || !other.known || r.id == other.id }

// observation is one recorded fact of one instance of the machine.
type observation struct {
	// identity names the evidence among all of the Run's.
	identity string
	// scope is the Run scope the evidence is recorded under, and instance the machine instance: the
	// operation key under that scope.
	scope, instance string
	sequence        int64
	kind            *kind
	source          string
	ordinal         int64
	// attempt and delivery name the execution attempt and the delivery the fact belongs to, where the
	// evidence says.
	attempt, delivery role
	// after is the identities the evidence names as its causal parents.
	after []string
	// recorded is the evidence as the Run carries it, which a republished identity must equal.
	recorded *testpilotspb.CorrelatedEvidence
}

// sameStep reports whether two observations may be facts of one step.
func (o *observation) sameStep(other *observation) bool {
	return o.attempt.agrees(other.attempt) && o.delivery.agrees(other.delivery)
}

// reader reads the evidence observation of one Case through one realization.
type reader struct {
	observation string
	// scope is the Case's scope fields, in order, and sources the evidence sources it names.
	scope   []string
	sources []string
	kinds   map[string]*kind
}

// newReader binds what the Case says of its evidence to the realization's declarations, by the
// Definition ID each of the Case's local names stands for: the field that scopes a Run and the field
// that names an operation, and each kind the Case carries, which are the ones its correlated Contract
// gives a meaning, with its source and its fields. A role the Case's evidence would need and neither
// declares is a refusal here.
func newReader(r *modelirspb.Realization, source *testpilotspb.Case) (*reader, error) {
	at := r.GetPosition()
	contract := source.GetContract().GetCorrelated()
	if contract == nil {
		return nil, located(at, "case %s has no correlated Contract: nothing says which kinds of evidence its Runs carry", source.GetCaseId())
	}
	if observed := contract.GetEvidenceObservationId(); observed == "" || observed != r.GetCorrelation().GetObservation() {
		return nil, located(at, "case %s reads its evidence from observation %q, and realization %s lifts it into %q", source.GetCaseId(),
			observed, r.GetName(), r.GetCorrelation().GetObservation())
	}
	local := map[string]string{}
	definition := map[string]string{}
	for _, name := range source.GetProvenance().GetLocalNames() {
		local[name.GetDefinitionId()], definition[name.GetLocalName()] = name.GetLocalName(), name.GetDefinitionId()
	}
	spelled := func(id string) string {
		if name, ok := local[id]; ok {
			return name
		}
		return id
	}
	if run := spelled(r.GetCorrelation().GetRun()); !slices.Equal(contract.GetScopeFields(), []string{run}) {
		return nil, located(r.GetCorrelation().GetPosition(), "case %s scopes its evidence by %v, and realization %s scopes a Run by %s", source.GetCaseId(),
			contract.GetScopeFields(), r.GetName(), run)
	}
	if operation := spelled(r.GetCorrelation().GetOperation()); contract.GetOperationField() != operation {
		return nil, located(r.GetCorrelation().GetPosition(), "case %s keys operations by %s, and realization %s keys them by %s", source.GetCaseId(),
			contract.GetOperationField(), r.GetName(), operation)
	}
	declared := map[string]*modelirspb.Evidence{}
	for _, e := range r.GetEvidence() {
		declared[e.GetId()] = e
	}
	out := &reader{observation: contract.GetEvidenceObservationId(), scope: contract.GetScopeFields(), sources: contract.GetSources(), kinds: map[string]*kind{}}
	for _, rule := range contract.GetProjectionRules() {
		id, ok := definition[rule.GetKind()]
		if !ok {
			id = rule.GetKind()
		}
		e, ok := declared[id]
		if !ok {
			return nil, located(at, "case %s carries evidence of kind %s, which realization %s does not declare", source.GetCaseId(), rule.GetKind(), r.GetName())
		}
		k := &kind{local: rule.GetKind(), source: spelled(e.GetSource()), records: e.GetRecords(), fields: rule.GetFields()}
		if !slices.Contains(out.sources, k.source) {
			return nil, located(e.GetPosition(), "evidence %s is recorded from %s, which is not a source of the Case %s", e.GetId(), k.source, source.GetCaseId())
		}
		for _, field := range k.fields {
			switch field.GetDisposition() {
			case testpilotspb.CORRELATED_FIELD_DISPOSITION_REDACT, testpilotspb.CORRELATED_FIELD_DISPOSITION_REJECT:
			case testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN:
				// A retained field may be what tells two attempts or two deliveries of one operation apart.
				return nil, located(e.GetPosition(), "case %s is not supported: evidence of kind %s retains field %s, and neither the realization nor the Case "+
					"declares the correlation role it plays, so it could be neither matched nor safely ignored", source.GetCaseId(), rule.GetKind(), field.GetFieldId())
			default:
				return nil, located(e.GetPosition(), "case %s: evidence of kind %s: field %s has no disposition", source.GetCaseId(), rule.GetKind(), field.GetFieldId())
			}
		}
		out.kinds[rule.GetKind()] = k
	}
	return out, nil
}

// identityOf spells an evidence identity, and scopeOf the scope it is recorded under.
func identityOf(id *testpilotspb.CorrelatedIdentity) string {
	return fmt.Sprintf("%s%s#%d", scopeOf(id), id.GetEvidenceSource(), id.GetOrdinal())
}

func scopeOf(id *testpilotspb.CorrelatedIdentity) string {
	var b strings.Builder
	for _, scope := range id.GetScope() {
		fmt.Fprintf(&b, "%s=%q;", scope.GetFieldId(), scope.GetValue().GetTextValue())
	}
	return b.String()
}

// read is the evidence one Run Event carries, or nothing for an event that carries none. Evidence
// that is not what the Case and the realization declare is an error at that event: it is never
// dropped, and never read as something else.
func (r *reader) read(event *testpilotspb.RunEvent) (*observation, error) {
	sequence := event.GetSequence()
	var found *observation
	for _, result := range event.GetObservations() {
		if result.GetObservationId() != r.observation {
			continue
		}
		if found != nil {
			return nil, &EvidenceError{Event: sequence, Message: "the evidence observation is recorded twice"}
		}
		packed := result.GetValue().GetMessageValue()
		if packed == nil {
			return nil, &EvidenceError{Event: sequence, Message: "the evidence observation carries no message"}
		}
		evidence := &testpilotspb.CorrelatedEvidence{}
		if err := packed.UnmarshalTo(evidence); err != nil {
			return nil, &EvidenceError{Event: sequence, Message: "the evidence observation is no correlated evidence: " + err.Error()}
		}
		var err error
		if found, err = r.observed(sequence, evidence); err != nil {
			return nil, err
		}
	}
	return found, nil
}

// scoped checks that an identity is recorded under the Case's scope fields, in their order, each with
// a value, and from a source the Case names.
func (r *reader) scoped(id *testpilotspb.CorrelatedIdentity) string {
	if id.GetEvidenceSource() == "" || id.GetOrdinal() < 0 {
		return "no source identity"
	}
	if !slices.Contains(r.sources, id.GetEvidenceSource()) {
		return fmt.Sprintf("source %q, which is not a source of the Case", id.GetEvidenceSource())
	}
	if !slices.EqualFunc(id.GetScope(), r.scope, func(field *testpilotspb.NamedValue, declared string) bool {
		return field.GetFieldId() == declared && field.GetValue().GetTextValue() != ""
	}) {
		return fmt.Sprintf("scope %s, which is not the Case's scope %v with a value for each field", scopeOf(id), r.scope)
	}
	return ""
}

// carried checks that evidence carries its kind's fields as the Case declares them: each declared
// field once, a redacted one without a value and a rejected one not at all, and no other.
func carried(declared *kind, evidence *testpilotspb.CorrelatedEvidence) string {
	seen := map[string]bool{}
	for _, field := range evidence.GetFields() {
		at := slices.IndexFunc(declared.fields, func(policy *testpilotspb.CorrelatedFieldPolicy) bool {
			return policy.GetFieldId() == field.GetFieldId()
		})
		switch {
		case at < 0:
			return fmt.Sprintf("carries field %s, which its kind does not declare", field.GetFieldId())
		case seen[field.GetFieldId()]:
			return fmt.Sprintf("carries field %s twice", field.GetFieldId())
		case declared.fields[at].GetDisposition() == testpilotspb.CORRELATED_FIELD_DISPOSITION_REJECT:
			return fmt.Sprintf("carries field %s, which the Case rejects", field.GetFieldId())
		case field.GetValue() != nil:
			return fmt.Sprintf("carries a value for field %s, which the Case redacts", field.GetFieldId())
		default:
			seen[field.GetFieldId()] = true
		}
	}
	for _, policy := range declared.fields {
		if policy.GetDisposition() == testpilotspb.CORRELATED_FIELD_DISPOSITION_REDACT && !seen[policy.GetFieldId()] {
			return "carries no field " + policy.GetFieldId()
		}
	}
	return ""
}

// observed is one piece of evidence as an observation, checked against what the Case declares for its
// kind.
func (r *reader) observed(sequence int64, evidence *testpilotspb.CorrelatedEvidence) (*observation, error) {
	refused := func(format string, args ...any) (*observation, error) {
		return nil, &EvidenceError{Event: sequence, Message: fmt.Sprintf(format, args...)}
	}
	declared, ok := r.kinds[evidence.GetKind()]
	if !ok {
		return refused("evidence of kind %q, which the Case does not carry", evidence.GetKind())
	}
	id := evidence.GetIdentity()
	if problem := r.scoped(id); problem != "" {
		return refused("evidence with %s", problem)
	}
	if id.GetEvidenceSource() != declared.source {
		return refused("evidence of kind %q from source %q, which is declared for source %q", declared.local, id.GetEvidenceSource(), declared.source)
	}
	if evidence.GetOperation() == "" {
		return refused("evidence that names no operation")
	}
	if problem := carried(declared, evidence); problem != "" {
		return refused("evidence of kind %q %s", declared.local, problem)
	}
	scope := scopeOf(id)
	found := &observation{identity: identityOf(id), scope: scope, instance: scope + evidence.GetOperation(), sequence: sequence, kind: declared,
		source: id.GetEvidenceSource(), ordinal: id.GetOrdinal(), recorded: proto.CloneOf(evidence)}
	for _, parent := range evidence.GetParents() {
		if parent.GetEvidenceSource() == "" {
			return refused("evidence with an empty causal parent")
		}
		if problem := r.scoped(parent); problem != "" {
			return refused("evidence with a causal parent of %s", problem)
		}
		if scopeOf(parent) != scope {
			return refused("evidence with a causal parent under another scope")
		}
		found.after = append(found.after, identityOf(parent))
	}
	return found, nil
}
