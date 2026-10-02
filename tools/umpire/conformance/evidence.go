package conformance

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

// This file is the one place a Run's recorded evidence is read. Everything after it sees an
// observation: which instance of the machine it is about, which fact it reports, which attempt and
// delivery the fact belongs to, and what it is ordered after.
//
// The instance is the operation key under the Run's one scope, and the order is the evidence's
// parents and source ordinals; both are text of the correlated evidence. Evidence of a kind that is
// the Run's own record is read only from a Run Event the kind's source takes (admits). The attempt and the delivery
// are typed roles of an observation, filled from two places: a field the Case retains and the
// realization gives that role, and the activity attempt a Run Event records beside the evidence it
// carries, which is typed data of the Run protocol. Where both name an identity they must name the
// same one. A field the Case retains and the realization does not declare is refused when the factory
// is prepared: nothing says what it names, so it could be neither matched nor safely ignored.

// kind is one kind of evidence the Case carries, as the realization declares it.
type kind struct {
	// local is the name the Case spells the kind and its source by.
	local, source string
	records       string
	// fields is the fields the Case declares for the kind, and roles the identity each retained one
	// names, by the Case's name for the field.
	fields []*testpilotspb.CorrelatedFieldPolicy
	roles  map[string]umpirespb.EvidenceField_Role
	// closing is the instruction whose read closes the kind's source, for a kind declared exhaustive.
	closing *coordinate
	// record is the Run Event source of a kind that is the Run's own record: the events its evidence
	// is read from, and from no other.
	record *umpirespb.RunEventSource
}

// coordinate names one instruction of a Case's Program.
type coordinate struct{ entrypoint, instruction string }

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
	// evidence says, and run the activity run the Run Event that carried it records.
	attempt, delivery, run role
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
func newReader(r *umpirespb.Realization, source *testpilotspb.Case) (*reader, error) {
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
	declared := map[string]*umpirespb.Evidence{}
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
		k := &kind{local: rule.GetKind(), source: spelled(e.GetSource()), records: e.GetRecords(), fields: rule.GetFields(), record: e.GetRunEvent()}
		if !slices.Contains(out.sources, k.source) {
			return nil, located(e.GetPosition(), "evidence %s is recorded from %s, which is not a source of the Case %s", e.GetId(), k.source, source.GetCaseId())
		}
		var err error
		if k.roles, err = rolesOf(r, source, e, rule, spelled); err != nil {
			return nil, err
		}
		if e.GetExhaustive() {
			if k.closing = closingOf(r, source, e.GetId()); k.closing == nil {
				at := closingRead(r, e.GetId())
				return nil, located(e.GetPosition(), "case %s carries the exhaustive evidence %s and no instruction %s/%s, the read realization %s closes it by",
					source.GetCaseId(), e.GetId(), at.entrypoint, at.instruction, r.GetName())
			}
		}
		out.kinds[rule.GetKind()] = k
	}
	return out, nil
}

// rolesOf is the identity each field a Case retains of one kind names, by the Case's name for the
// field. A retained field may be what tells two attempts or two deliveries of one operation apart, so
// it is read only as what the realization says it is: one the realization does not declare for the
// kind, or redacts, is refused.
func rolesOf(r *umpirespb.Realization, source *testpilotspb.Case, e *umpirespb.Evidence, rule *testpilotspb.CorrelatedProjectionRule,
	spelled func(string) string) (map[string]umpirespb.EvidenceField_Role, error) {
	fields := map[string]*umpirespb.EvidenceField{}
	for _, f := range e.GetFields() {
		fields[spelled(f.GetId())] = f
	}
	roles := map[string]umpirespb.EvidenceField_Role{}
	for _, field := range rule.GetFields() {
		f, declared := fields[field.GetFieldId()]
		switch disposition := field.GetDisposition(); {
		case disposition == testpilotspb.CORRELATED_FIELD_DISPOSITION_REDACT || disposition == testpilotspb.CORRELATED_FIELD_DISPOSITION_REJECT:
		case disposition != testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN:
			return nil, located(e.GetPosition(), "case %s: evidence of kind %s: field %s has no disposition", source.GetCaseId(), rule.GetKind(), field.GetFieldId())
		case !declared:
			return nil, located(e.GetPosition(), "case %s: evidence of kind %s retains field %s, which realization %s does not declare for it",
				source.GetCaseId(), rule.GetKind(), field.GetFieldId(), r.GetName())
		case f.GetRedacted():
			return nil, located(e.GetPosition(), "case %s: evidence of kind %s retains field %s, which realization %s redacts",
				source.GetCaseId(), rule.GetKind(), field.GetFieldId(), r.GetName())
		default:
			roles[field.GetFieldId()] = f.GetRole()
		}
	}
	return roles, nil
}

// closingRead is the command a realization closes an exhaustive kind of evidence by: its script and
// its id, which are the entrypoint and the instruction of every Case that carries it. Admission lets
// through no exhaustive kind without one. The command that closes its own record may perform a step,
// and is then the instruction of the first step of its class a Case's path takes.
func closingRead(r *umpirespb.Realization, evidence string) coordinate {
	for _, s := range r.GetScripts() {
		for _, item := range s.GetItems() {
			if slices.Contains(item.GetCommand().GetCloses(), evidence) {
				return coordinate{entrypoint: s.GetId(), instruction: item.GetCommand().GetId()}
			}
			for _, performance := range item.GetPerforms() {
				if slices.Contains(performance.GetCommand().GetCloses(), evidence) {
					return coordinate{entrypoint: s.GetId(), instruction: performance.GetCommand().GetId()}
				}
			}
		}
	}
	return coordinate{}
}

// closingOf is the instruction of the Case that is the closing read of an exhaustive kind, or nil
// where the Case has none.
func closingOf(r *umpirespb.Realization, source *testpilotspb.Case, evidence string) *coordinate {
	at := closingRead(r, evidence)
	for _, entrypoint := range source.GetProgram().GetEntrypoints() {
		if entrypoint.GetEntrypointId() != at.entrypoint {
			continue
		}
		for _, node := range entrypoint.GetInstructions() {
			if node.GetInstructionId() == at.instruction && at.instruction != "" {
				return &at
			}
		}
	}
	return nil
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
		if found, err = r.observed(sequence, evidence, event.GetOutcome().GetActivityAttempt()); err != nil {
			return nil, err
		}
		if err := occurrence(found.kind, event); err != nil {
			return nil, err
		}
	}
	return found, nil
}

// occurrence checks that evidence of a kind that is the Run's own record is carried by a Run Event the
// kind's source takes: the declaration says which events are occurrences of the evidence, and evidence
// on any other event is none of them. A guard that cannot be evaluated on the event is its error.
func occurrence(declared *kind, event *testpilotspb.RunEvent) error {
	if declared.record == nil {
		return nil
	}
	admitted, err := admits(declared.record, event)
	if err != nil {
		return err
	}
	if !admitted {
		return &EvidenceError{Event: event.GetSequence(), Message: fmt.Sprintf("evidence of kind %q on a Run Event its source does not take: the Run's record of %s/%s under its guard",
			declared.local, declared.record.GetScript(), declared.record.GetCommand())}
	}
	return nil
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
// field once, a retained one with a value, a redacted one without and a rejected one not at all, and
// no other.
func carried(declared *kind, evidence *testpilotspb.CorrelatedEvidence) string {
	seen := map[string]bool{}
	for _, field := range evidence.GetFields() {
		at := slices.IndexFunc(declared.fields, func(policy *testpilotspb.CorrelatedFieldPolicy) bool {
			return policy.GetFieldId() == field.GetFieldId()
		})
		if at < 0 {
			return fmt.Sprintf("carries field %s, which its kind does not declare", field.GetFieldId())
		}
		if seen[field.GetFieldId()] {
			return fmt.Sprintf("carries field %s twice", field.GetFieldId())
		}
		_, valued := scalar(field.GetValue())
		switch disposition := declared.fields[at].GetDisposition(); {
		case disposition == testpilotspb.CORRELATED_FIELD_DISPOSITION_REJECT:
			return fmt.Sprintf("carries field %s, which the Case rejects", field.GetFieldId())
		case disposition == testpilotspb.CORRELATED_FIELD_DISPOSITION_REDACT && field.GetValue() != nil:
			return fmt.Sprintf("carries a value for field %s, which the Case redacts", field.GetFieldId())
		case disposition == testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN && field.GetValue() == nil:
			return fmt.Sprintf("carries no value for field %s, which the Case retains", field.GetFieldId())
		case disposition == testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN && !valued:
			return fmt.Sprintf("carries a value for field %s that is no text, number or flag", field.GetFieldId())
		default:
			seen[field.GetFieldId()] = true
		}
	}
	for _, policy := range declared.fields {
		if policy.GetDisposition() != testpilotspb.CORRELATED_FIELD_DISPOSITION_REJECT && !seen[policy.GetFieldId()] {
			return "carries no field " + policy.GetFieldId()
		}
	}
	return ""
}

// scalar spells a value of the portable evidence domain, a text, an unsigned integer or a flag, and
// is whether the value is one.
func scalar(v *testpilotspb.Value) (string, bool) {
	switch value := v.GetValue().(type) {
	case *testpilotspb.Value_TextValue:
		return value.TextValue, true
	case *testpilotspb.Value_UnsignedIntegerValue:
		return value.UnsignedIntegerValue, true
	case *testpilotspb.Value_BoolValue:
		return strconv.FormatBool(value.BoolValue), true
	default:
		return "", false
	}
}

// named is the one identity two sources give a role: the field the evidence retains, and the typed
// record of the Run Event. Either may be silent; where both speak they agree, or the evidence is
// crossed.
func named(what, field, fromField, fromEvent string) (role, string) {
	switch {
	case fromField != "" && fromEvent != "" && fromField != fromEvent:
		return role{}, fmt.Sprintf("evidence whose field %s names %s %q on a Run Event of %s %q", field, what, fromField, what, fromEvent)
	case fromField != "":
		return role{known: true, id: fromField}, ""
	case fromEvent != "":
		return role{known: true, id: fromEvent}, ""
	default:
		return role{}, ""
	}
}

// observed is one piece of evidence as an observation, checked against what the Case declares for its
// kind. attempt is the activity attempt the Run Event that carried the evidence records, or nil.
func (r *reader) observed(sequence int64, evidence *testpilotspb.CorrelatedEvidence, attempt *testpilotspb.ActivityAttempt) (*observation, error) {
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
	// The identities the evidence's own fields name, and the ones the Run Event records as typed data.
	fields, names := map[umpirespb.EvidenceField_Role]string{}, map[umpirespb.EvidenceField_Role]string{}
	for _, field := range evidence.GetFields() {
		if role := declared.roles[field.GetFieldId()]; role != umpirespb.EvidenceField_ROLE_UNSPECIFIED {
			fields[role], _ = scalar(field.GetValue())
			names[role] = field.GetFieldId()
		}
	}
	if name, keyed := names[umpirespb.EvidenceField_ROLE_OPERATION]; keyed && fields[umpirespb.EvidenceField_ROLE_OPERATION] != evidence.GetOperation() {
		return refused("evidence of kind %q for operation %q, whose field %s names operation %q", declared.local, evidence.GetOperation(), name,
			fields[umpirespb.EvidenceField_ROLE_OPERATION])
	}
	delivered := ""
	if attempt.GetSdkAttempt() > 0 {
		delivered = strconv.FormatInt(int64(attempt.GetSdkAttempt()), 10)
	}
	var crossed string
	if found.attempt, crossed = named("attempt", names[umpirespb.EvidenceField_ROLE_ATTEMPT], fields[umpirespb.EvidenceField_ROLE_ATTEMPT], delivered); crossed != "" {
		return refused("%s", crossed)
	}
	if found.delivery, crossed = named("delivery", names[umpirespb.EvidenceField_ROLE_DELIVERY], fields[umpirespb.EvidenceField_ROLE_DELIVERY],
		attempt.GetDeliveryId()); crossed != "" {
		return refused("%s", crossed)
	}
	found.run, _ = named("activity run", "", "", attempt.GetActivityRunId())
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

// closingOutcome is what became of the read that closes an exhaustive source, as the Run records it.
type closingOutcome uint8

const (
	// notRun is a read the Run records no completion of.
	notRun closingOutcome = iota
	readFailed
	readSucceeded
)

// sourceClosed is the one rule by which an exhaustive source says what did not happen: the Run closed
// complete, the last completion of its closing read is a success, and its ordinals are unbroken, so no
// evidence of it is missing from the record. Short of that it proves what it reports and no more.
func sourceClosed(positive bool, read closingOutcome, unbroken bool) bool {
	return positive && read == readSucceeded && unbroken
}

// ordinals is the ordinals a Run recorded for one source.
type ordinals struct {
	count int
	// highest is the greatest ordinal recorded.
	highest int64
}

func (o *ordinals) record(ordinal int64) {
	o.count++
	o.highest = max(o.highest, ordinal)
}

// unbroken is whether the ordinals are zero up to their count. Each is recorded once, since an
// identity recorded again is not counted again, so they are when the greatest is the count less one.
func (o *ordinals) unbroken() bool { return o == nil || o.count == 0 || o.highest == int64(o.count)-1 }
