package portableevaluation

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	"google.golang.org/protobuf/proto"
)

type normalizedField struct {
	reference   *umpirespb.EvidenceFieldReference
	declaration *umpirespb.EvidenceFieldDeclaration
	value       *umpirespb.Value
	digestToken string
}

type normalizedRecord struct {
	fact   artifactv2.RawEvidenceFact
	fields []*normalizedField
}

type emission struct {
	emit   *umpirespb.Emit
	record *normalizedRecord
	value  *umpirespb.ModelValue
	link   *umpirespb.EvidenceLink
}

func (i *interpreter) evaluateObservation() (*umpirespb.ObservationEvaluationResult, *evaluationFailure) {
	records, failure := i.normalizeEvidence()
	if failure != nil {
		return nil, failure
	}
	emissions := make([]*emission, 0, len(i.contract.GetObservation().GetEmits()))
	for _, emit := range i.contract.GetObservation().GetEmits() {
		candidate, failure := i.evaluateEmit(emit, records)
		if failure != nil {
			return nil, failure
		}
		emissions = append(emissions, candidate)
	}
	if failure := validateUniqueCoordinates(emissions); failure != nil {
		return nil, failure
	}
	sortEmissions(emissions)
	if failure := i.validateEmitOrdering(emissions); failure != nil {
		return nil, failure
	}
	trace, failure := buildTrace(emissions)
	if failure != nil {
		return nil, failure
	}
	trace.TraceId = traceIdentity(trace)
	links := make([]*umpirespb.EvidenceLink, len(emissions))
	for index, candidate := range emissions {
		links[index] = candidate.link
	}
	return &umpirespb.ObservationEvaluationResult{
		Status:        umpirespb.OBSERVATION_STATUS_ACCEPTED,
		Trace:         trace,
		EvidenceLinks: links,
	}, nil
}

func (i *interpreter) evaluateEmit(
	emit *umpirespb.Emit,
	records []*normalizedRecord,
) (*emission, *evaluationFailure) {
	candidates := make([]*emission, 0, 1)
	for _, record := range records {
		if record.fact.KindDefinitionID != emit.GetSourceKindDefinitionId() {
			continue
		}
		candidate, failure := i.evaluateEmitRecord(emit, record)
		if failure != nil {
			return nil, failure
		}
		if candidate != nil {
			candidates = append(candidates, candidate)
		}
	}
	return selectEmission(emit, candidates)
}

func (i *interpreter) evaluateEmitRecord(
	emit *umpirespb.Emit,
	record *normalizedRecord,
) (*emission, *evaluationFailure) {
	if failure := i.work.charge(i.ctx.Err(), umpirespb.WORK_UNIT_KIND_RULE_RECORD_CANDIDATE, 1); failure != nil {
		return nil, failure
	}
	condition, failure := i.evaluateExpression(emit.GetCondition(), record)
	if failure != nil {
		return nil, failure
	}
	conditionValue, ok := condition.GetValue().(*umpirespb.Value_BoolValue)
	if !ok {
		return nil, typeFailure(emit.GetDefinitionId(), emit.GetCoordinate(), "emit condition is not Boolean")
	}
	if !conditionValue.BoolValue {
		return nil, nil
	}
	value, failure := i.evaluateExpression(emit.GetValue(), record)
	if failure != nil {
		return nil, failure
	}
	if failure = validateCoordinateKind(emit); failure != nil {
		return nil, failure
	}
	if failure = i.work.charge(i.ctx.Err(), umpirespb.WORK_UNIT_KIND_EMITTED_COORDINATE, 1); failure != nil {
		return nil, failure
	}
	candidate := &emission{
		emit:   emit,
		record: record,
		value: &umpirespb.ModelValue{
			Definition: proto.CloneOf(emit.GetOutputDefinition()),
			Kind:       emit.GetOutputKind(),
			Value:      proto.CloneOf(value),
		},
	}
	candidate.link, failure = i.evidenceLink(candidate)
	if failure != nil {
		return nil, failure
	}
	return candidate, nil
}

func selectEmission(emit *umpirespb.Emit, candidates []*emission) (*emission, *evaluationFailure) {
	switch len(candidates) {
	case 0:
		return nil, &evaluationFailure{
			class:   umpirespb.DIAGNOSTIC_CLASS_UNKNOWN,
			code:    umpirespb.DIAGNOSTIC_CODE_MISSING_COORDINATE,
			related: []string{emit.GetDefinitionId()}, coord: proto.CloneOf(emit.GetCoordinate()),
			detail: "emit did not establish its required Model coordinate",
		}
	case 1:
		return candidates[0], nil
	default:
		code := umpirespb.DIAGNOSTIC_CODE_DUPLICATE_COORDINATE
		for _, candidate := range candidates[1:] {
			if !proto.Equal(candidate.value, candidates[0].value) {
				code = umpirespb.DIAGNOSTIC_CODE_CONTRADICTORY_COORDINATE
				break
			}
		}
		return nil, &evaluationFailure{
			class: umpirespb.DIAGNOSTIC_CLASS_CONFLICT, code: code,
			related: []string{emit.GetDefinitionId()}, coord: proto.CloneOf(emit.GetCoordinate()),
			observed: int64(len(candidates)), detail: "multiple Evidence records established one Model coordinate",
		}
	}
}

func (i *interpreter) normalizeEvidence() ([]*normalizedRecord, *evaluationFailure) {
	profile := i.contract.GetObservation().GetProfile()
	sources, failure := i.validateClosedSources(profile)
	if failure != nil {
		return nil, failure
	}
	kinds := make(map[string]*umpirespb.EvidenceKindDeclaration, len(profile.GetKinds()))
	for _, kind := range profile.GetKinds() {
		kinds[kind.GetKindDefinitionId()] = kind
	}
	records, countsByKind, countsBySource, failure := i.normalizeRecords(kinds, len(sources))
	if failure != nil {
		return nil, failure
	}
	if failure = validateSourceCounts(i.request.RawEvidence.Sources, countsBySource); failure != nil {
		return nil, failure
	}
	if failure = validateCardinalities(profile.GetCardinalities(), countsByKind); failure != nil {
		return nil, failure
	}
	if failure = validateCausalOrder(records); failure != nil {
		return nil, failure
	}
	if failure = validateCorrelations(profile.GetCorrelationSlots(), records); failure != nil {
		return nil, failure
	}
	return records, nil
}

func (i *interpreter) validateClosedSources(
	profile *umpirespb.EvidenceProfile,
) (map[string]*umpirespb.EvidenceSourceDeclaration, *evaluationFailure) {
	sources := make(map[string]*umpirespb.EvidenceSourceDeclaration, len(profile.GetSources()))
	for _, source := range profile.GetSources() {
		sources[source.GetSourceDefinitionId()] = source
	}
	actualSources := make(map[string]artifactv2.RawEvidenceSource, len(i.request.RawEvidence.Sources))
	for _, source := range i.request.RawEvidence.Sources {
		if _, declared := sources[source.SourceDefinitionID]; !declared {
			return nil, conflictFailure(umpirespb.DIAGNOSTIC_CODE_SOURCE_IDENTITY,
				[]string{source.SourceDefinitionID}, "Raw Evidence contains an undeclared source")
		}
		if _, duplicate := actualSources[source.SourceDefinitionID]; duplicate {
			return nil, conflictFailure(umpirespb.DIAGNOSTIC_CODE_SOURCE_IDENTITY,
				[]string{source.SourceDefinitionID}, "Raw Evidence repeats a source")
		}
		actualSources[source.SourceDefinitionID] = source
		if source.Status != "closed" {
			return nil, missingClosureFailure(source.SourceDefinitionID)
		}
	}
	for _, source := range profile.GetSources() {
		if _, found := actualSources[source.GetSourceDefinitionId()]; !found {
			return nil, missingClosureFailure(source.GetSourceDefinitionId())
		}
	}
	if i.request.RawEvidence.CaptureStatus != "closed" {
		return nil, missingClosureFailure(profile.GetDefinition().GetDefinitionId())
	}
	return sources, nil
}

func (i *interpreter) normalizeRecords(
	kinds map[string]*umpirespb.EvidenceKindDeclaration,
	sourceCount int,
) (
	records []*normalizedRecord,
	countsByKind map[string]int64,
	countsBySource map[string]int64,
	failure *evaluationFailure,
) {
	records = make([]*normalizedRecord, 0, len(i.request.RawEvidence.Facts))
	countsByKind = make(map[string]int64, len(kinds))
	countsBySource = make(map[string]int64, sourceCount)
	expectedOrdinal := make(map[string]uint64, sourceCount)
	seenFacts := make(map[string]struct{}, len(i.request.RawEvidence.Facts))
	for _, fact := range i.request.RawEvidence.Facts {
		if _, duplicate := seenFacts[fact.FactDefinitionID]; duplicate {
			return nil, nil, nil, conflictFailure(umpirespb.DIAGNOSTIC_CODE_DUPLICATE_BINDING,
				[]string{fact.FactDefinitionID}, "Raw Evidence repeats an identity")
		}
		seenFacts[fact.FactDefinitionID] = struct{}{}
		kind := kinds[fact.KindDefinitionID]
		if kind == nil {
			return nil, nil, nil, unsupportedFailure(umpirespb.DIAGNOSTIC_CODE_UNDECLARED_KIND,
				[]string{fact.FactDefinitionID, fact.KindDefinitionID}, "Raw Evidence kind is not declared")
		}
		if kind.GetSourceDefinitionId() != fact.SourceDefinitionID {
			return nil, nil, nil, conflictFailure(umpirespb.DIAGNOSTIC_CODE_SOURCE_IDENTITY,
				[]string{fact.FactDefinitionID, fact.SourceDefinitionID}, "Evidence kind is crossed with another source")
		}
		ordinal, err := strconv.ParseUint(string(fact.Ordinal), 10, 64)
		if err != nil || ordinal != expectedOrdinal[fact.SourceDefinitionID] {
			return nil, nil, nil, conflictFailure(umpirespb.DIAGNOSTIC_CODE_SOURCE_ORDER,
				[]string{fact.FactDefinitionID, fact.SourceDefinitionID}, "source-local ordinal is not contiguous")
		}
		expectedOrdinal[fact.SourceDefinitionID]++
		countsBySource[fact.SourceDefinitionID]++
		countsByKind[fact.KindDefinitionID]++
		record, failure := i.normalizeRecord(fact, kind)
		if failure != nil {
			return nil, nil, nil, failure
		}
		records = append(records, record)
	}
	return records, countsByKind, countsBySource, nil
}

func (i *interpreter) normalizeRecord(
	fact artifactv2.RawEvidenceFact,
	kind *umpirespb.EvidenceKindDeclaration,
) (*normalizedRecord, *evaluationFailure) {
	record := &normalizedRecord{fact: fact}
	for _, rawField := range fact.Fields {
		declaration := findFieldDeclaration(kind, rawField.FieldDefinitionID)
		if declaration == nil {
			return nil, unsupportedFailure(umpirespb.DIAGNOSTIC_CODE_UNDECLARED_FIELD,
				[]string{fact.FactDefinitionID, rawField.FieldDefinitionID}, "Raw Evidence field is not declared")
		}
		for _, existing := range record.fields {
			if existing.reference.GetFieldDefinitionId() == rawField.FieldDefinitionID {
				return nil, conflictFailure(umpirespb.DIAGNOSTIC_CODE_DUPLICATE_FIELD,
					[]string{fact.FactDefinitionID, rawField.FieldDefinitionID}, "Raw Evidence repeats a field")
			}
		}
		normalized, failure := i.normalizeField(fact, rawField, declaration)
		if failure != nil {
			return nil, failure
		}
		record.fields = append(record.fields, normalized)
	}
	return record, nil
}

func validateSourceCounts(
	sources []artifactv2.RawEvidenceSource,
	countsBySource map[string]int64,
) *evaluationFailure {
	for _, source := range sources {
		declared, err := strconv.ParseInt(string(source.FactCount), 10, 64)
		if err != nil || declared != countsBySource[source.SourceDefinitionID] {
			return missingClosureFailure(source.SourceDefinitionID)
		}
	}
	return nil
}

func validateCardinalities(
	cardinalities []*umpirespb.EvidenceCardinality,
	countsByKind map[string]int64,
) *evaluationFailure {
	for _, cardinality := range cardinalities {
		count := countsByKind[cardinality.GetKindDefinitionId()]
		if count < cardinality.GetMinimum() {
			return &evaluationFailure{
				class: umpirespb.DIAGNOSTIC_CLASS_UNKNOWN, code: umpirespb.DIAGNOSTIC_CODE_MISSING_BINDING,
				related: []string{cardinality.GetKindDefinitionId()}, observed: count,
				detail: "Evidence kind does not meet its minimum cardinality",
			}
		}
		if count > cardinality.GetMaximum() {
			return conflictFailure(umpirespb.DIAGNOSTIC_CODE_DUPLICATE_BINDING,
				[]string{cardinality.GetKindDefinitionId()}, "Evidence kind exceeds its maximum cardinality")
		}
	}
	return nil
}

func (i *interpreter) normalizeField(
	fact artifactv2.RawEvidenceFact,
	raw artifactv2.RawEvidenceField,
	declaration *umpirespb.EvidenceFieldDeclaration,
) (*normalizedField, *evaluationFailure) {
	field := &normalizedField{
		reference: &umpirespb.EvidenceFieldReference{
			KindDefinitionId: fact.KindDefinitionID, FieldDefinitionId: raw.FieldDefinitionID,
		},
		declaration: declaration,
	}
	related := []string{fact.FactDefinitionID, raw.FieldDefinitionID}
	switch declaration.GetDisposition() {
	case umpirespb.FIELD_DISPOSITION_KIND_RETAIN:
		if raw.Disposition != "plain" {
			return nil, unsupportedFailure(umpirespb.DIAGNOSTIC_CODE_TYPE_MISMATCH, related,
				"retained field does not contain plain Evidence")
		}
		value, failure := i.normalizeValue(declaration.GetValueKind(), raw.Value, related)
		if failure != nil {
			return nil, failure
		}
		field.value = value
	case umpirespb.FIELD_DISPOSITION_KIND_REDACT:
		if raw.Disposition != "redacted" || raw.Value != nil {
			return nil, unsupportedFailure(umpirespb.DIAGNOSTIC_CODE_TYPE_MISMATCH, related,
				"redacted field exposes raw material")
		}
	case umpirespb.FIELD_DISPOSITION_KIND_HASH:
		token, ok := raw.Value.(string)
		if raw.Disposition != "sha256" || !ok || !artifactv2.ValidDigest(token) {
			return nil, unsupportedFailure(umpirespb.DIAGNOSTIC_CODE_TYPE_MISMATCH, related,
				"hashed field does not contain its admitted digest token")
		}
		field.digestToken = token
	case umpirespb.FIELD_DISPOSITION_KIND_REJECT:
		return nil, unsupportedFailure(umpirespb.DIAGNOSTIC_CODE_UNDECLARED_FIELD, related,
			"rejected field is present in Raw Evidence")
	default:
		return nil, unsupportedFailure(umpirespb.DIAGNOSTIC_CODE_UNSUPPORTED_OPERATOR, related,
			"field disposition is unsupported")
	}
	return field, nil
}

func (i *interpreter) normalizeValue(
	kind umpirespb.ValueKind,
	raw interface{},
	related []string,
) (*umpirespb.Value, *evaluationFailure) {
	switch kind {
	case umpirespb.VALUE_KIND_TEXT:
		value, ok := raw.(string)
		if ok {
			return &umpirespb.Value{Value: &umpirespb.Value_Text{Text: value}}, nil
		}
	case umpirespb.VALUE_KIND_NATURAL:
		var value string
		switch typed := raw.(type) {
		case json.Number:
			value = string(typed)
		case string:
			value = typed
		default:
		}
		if validNatural(value) {
			if compareNatural(value, i.contract.GetLimits().GetMaxNatural()) > 0 {
				return nil, unsupportedFailure(umpirespb.DIAGNOSTIC_CODE_NATURAL_OUT_OF_RANGE, related,
					"natural exceeds the contract bound")
			}
			return &umpirespb.Value{Value: &umpirespb.Value_Natural{Natural: value}}, nil
		}
		if value != "" {
			return nil, unsupportedFailure(umpirespb.DIAGNOSTIC_CODE_NONCANONICAL_NATURAL, related,
				"natural is not canonical unsigned base-10")
		}
	case umpirespb.VALUE_KIND_BOOLEAN:
		value, ok := raw.(bool)
		if ok {
			return &umpirespb.Value{Value: &umpirespb.Value_BoolValue{BoolValue: value}}, nil
		}
	default:
		return nil, unsupportedFailure(umpirespb.DIAGNOSTIC_CODE_UNSUPPORTED_OPERATOR, related,
			"Evidence field tagged type is unsupported")
	}
	return nil, unsupportedFailure(umpirespb.DIAGNOSTIC_CODE_TYPE_MISMATCH, related,
		"Raw Evidence value does not match the declared tagged type")
}

func (i *interpreter) evaluateExpression(
	expression *umpirespb.ObservationExpression,
	record *normalizedRecord,
) (*umpirespb.Value, *evaluationFailure) {
	if failure := i.work.charge(i.ctx.Err(), umpirespb.WORK_UNIT_KIND_EXPRESSION_VISIT, 1); failure != nil {
		return nil, failure
	}
	switch operator := expression.GetOperator().(type) {
	case *umpirespb.ObservationExpression_LiteralText:
		return &umpirespb.Value{Value: &umpirespb.Value_Text{Text: operator.LiteralText.GetValue()}}, nil
	case *umpirespb.ObservationExpression_LiteralNatural:
		return &umpirespb.Value{Value: &umpirespb.Value_Natural{Natural: operator.LiteralNatural.GetValue()}}, nil
	case *umpirespb.ObservationExpression_Field:
		matches := recordFields(record, operator.Field)
		switch len(matches) {
		case 0:
			return nil, &evaluationFailure{
				class: umpirespb.DIAGNOSTIC_CLASS_UNKNOWN, code: umpirespb.DIAGNOSTIC_CODE_MISSING_FIELD,
				related: []string{record.fact.FactDefinitionID, operator.Field.GetFieldDefinitionId()},
				detail:  "Evidence field is absent",
			}
		case 1:
			if matches[0].value == nil {
				return nil, unsupportedFailure(umpirespb.DIAGNOSTIC_CODE_TYPE_MISMATCH,
					[]string{record.fact.FactDefinitionID, operator.Field.GetFieldDefinitionId()},
					"field disposition does not expose a typed value")
			}
			return proto.CloneOf(matches[0].value), nil
		default:
			return nil, conflictFailure(umpirespb.DIAGNOSTIC_CODE_DUPLICATE_FIELD,
				[]string{record.fact.FactDefinitionID, operator.Field.GetFieldDefinitionId()},
				"Evidence field is ambiguous")
		}
	case *umpirespb.ObservationExpression_NaturalRenderV1:
		value, failure := i.evaluateExpression(operator.NaturalRenderV1.GetOperand(), record)
		if failure != nil {
			return nil, failure
		}
		natural, ok := value.GetValue().(*umpirespb.Value_Natural)
		if !ok {
			return nil, typeFailure(record.fact.FactDefinitionID, nil, "natural_render_v1 operand is not natural")
		}
		return &umpirespb.Value{Value: &umpirespb.Value_Text{Text: natural.Natural}}, nil
	case *umpirespb.ObservationExpression_Present:
		value, failure := i.evaluateExpression(operator.Present.GetOperand(), record)
		if failure != nil {
			if failure.code == umpirespb.DIAGNOSTIC_CODE_MISSING_FIELD ||
				failure.code == umpirespb.DIAGNOSTIC_CODE_MISSING_BINDING {
				return &umpirespb.Value{Value: &umpirespb.Value_BoolValue{}}, nil
			}
			return nil, failure
		}
		return &umpirespb.Value{Value: &umpirespb.Value_BoolValue{BoolValue: value != nil}}, nil
	case *umpirespb.ObservationExpression_Equals:
		left, failure := i.evaluateExpression(operator.Equals.GetLeft(), record)
		if failure != nil {
			return nil, failure
		}
		right, failure := i.evaluateExpression(operator.Equals.GetRight(), record)
		if failure != nil {
			return nil, failure
		}
		if valueKind(left) != valueKind(right) {
			return nil, typeFailure(record.fact.FactDefinitionID, nil, "equals operands have different tagged types")
		}
		return &umpirespb.Value{Value: &umpirespb.Value_BoolValue{BoolValue: proto.Equal(left, right)}}, nil
	case *umpirespb.ObservationExpression_All:
		return i.evaluateBooleanOperands(record, operator.All.GetOperands(), true)
	case *umpirespb.ObservationExpression_Any:
		return i.evaluateBooleanOperands(record, operator.Any.GetOperands(), false)
	default:
		return nil, unsupportedFailure(umpirespb.DIAGNOSTIC_CODE_UNSUPPORTED_OPERATOR,
			[]string{record.fact.FactDefinitionID}, "Observation operator is unsupported")
	}
}

func (i *interpreter) evaluateBooleanOperands(
	record *normalizedRecord,
	operands []*umpirespb.ObservationExpression,
	all bool,
) (*umpirespb.Value, *evaluationFailure) {
	result := all
	var firstFailure *evaluationFailure
	for _, operand := range operands {
		value, failure := i.evaluateExpression(operand, record)
		if failure != nil {
			if firstFailure == nil {
				firstFailure = failure
			}
			continue
		}
		boolean, ok := value.GetValue().(*umpirespb.Value_BoolValue)
		if !ok {
			if firstFailure == nil {
				firstFailure = typeFailure(record.fact.FactDefinitionID, nil, "all/any operand is not Boolean")
			}
			continue
		}
		if all {
			result = result && boolean.BoolValue
		} else {
			result = result || boolean.BoolValue
		}
	}
	if firstFailure != nil {
		return nil, firstFailure
	}
	return &umpirespb.Value{Value: &umpirespb.Value_BoolValue{BoolValue: result}}, nil
}

func (i *interpreter) evidenceLink(candidate *emission) (*umpirespb.EvidenceLink, *evaluationFailure) {
	references := expressionFieldReferences(candidate.emit.GetCondition())
	references = append(references, expressionFieldReferences(candidate.emit.GetValue())...)
	references = uniqueFieldReferences(references)
	applied := make([]*umpirespb.AppliedDisposition, 0, len(references))
	for _, reference := range references {
		matches := recordFields(candidate.record, reference)
		if len(matches) != 1 {
			return nil, conflictFailure(umpirespb.DIAGNOSTIC_CODE_DUPLICATE_FIELD,
				[]string{candidate.record.fact.FactDefinitionID, reference.GetFieldDefinitionId()},
				"emitted fact lacks one exact disposition input")
		}
		field := matches[0]
		disposition := &umpirespb.AppliedDisposition{
			Field:                    proto.CloneOf(field.reference),
			Disposition:              field.declaration.GetDisposition(),
			DigestPolicyDefinitionId: field.declaration.GetDigestPolicyDefinitionId(),
			DigestToken:              field.digestToken,
		}
		if field.value != nil {
			disposition.NormalizedValue = proto.CloneOf(field.value)
		}
		applied = append(applied, disposition)
	}
	link := &umpirespb.EvidenceLink{
		Coordinate:            proto.CloneOf(candidate.emit.GetCoordinate()),
		Mapping:               proto.CloneOf(i.contract.GetObservation().GetMapping()),
		RuleDefinitionId:      candidate.emit.GetDefinitionId(),
		EvidenceDefinitionIds: []string{candidate.record.fact.FactDefinitionID},
		AppliedDispositions:   applied,
	}
	for _, fact := range i.request.RawEvidence.Facts {
		ordinal, _ := strconv.ParseInt(string(fact.Ordinal), 10, 64)
		link.OrderingSupport = append(link.OrderingSupport, &umpirespb.OrderingFact{
			EvidenceDefinitionId:        fact.FactDefinitionID,
			SourceDefinitionId:          fact.SourceDefinitionID,
			Ordinal:                     ordinal,
			CausalEvidenceDefinitionIds: append([]string(nil), fact.CausalFactDefinitionIDs...),
		})
	}
	for _, source := range i.request.RawEvidence.Sources {
		recordCount, _ := strconv.ParseInt(string(source.FactCount), 10, 64)
		byteCount, _ := strconv.ParseInt(string(source.ByteCount), 10, 64)
		link.ClosureSupport = append(link.ClosureSupport, &umpirespb.ClosureFact{
			SourceDefinitionId: source.SourceDefinitionID,
			RecordCount:        recordCount,
			ByteCount:          byteCount,
		})
	}
	return link, nil
}

func validateCoordinateKind(emit *umpirespb.Emit) *evaluationFailure {
	want := umpirespb.DEFINITION_KIND_STATE
	switch emit.GetCoordinate().GetField() {
	case umpirespb.TRACE_FIELD_SELECTED_ACTION:
		want = umpirespb.DEFINITION_KIND_ACTION
	case umpirespb.TRACE_FIELD_MODEL_OUTCOME:
		want = umpirespb.DEFINITION_KIND_OUTCOME
	case umpirespb.TRACE_FIELD_OBSERVATION:
		want = umpirespb.DEFINITION_KIND_OBSERVATION
	case umpirespb.TRACE_FIELD_INITIAL_STATE, umpirespb.TRACE_FIELD_PRIOR_STATE,
		umpirespb.TRACE_FIELD_RESULTING_STATE:
	default:
		return unsupportedFailure(umpirespb.DIAGNOSTIC_CODE_UNSUPPORTED_OPERATOR,
			[]string{emit.GetDefinitionId()}, "emit coordinate is unsupported")
	}
	if emit.GetOutputKind() != want {
		return typeFailure(emit.GetDefinitionId(), emit.GetCoordinate(), "emit output kind does not match its coordinate")
	}
	return nil
}

func validateUniqueCoordinates(emissions []*emission) *evaluationFailure {
	seen := make(map[string]*emission, len(emissions))
	for _, candidate := range emissions {
		key := coordinateKey(candidate.emit.GetCoordinate())
		if prior := seen[key]; prior != nil {
			code := umpirespb.DIAGNOSTIC_CODE_DUPLICATE_COORDINATE
			if !proto.Equal(prior.value, candidate.value) {
				code = umpirespb.DIAGNOSTIC_CODE_CONTRADICTORY_COORDINATE
			}
			return &evaluationFailure{
				class: umpirespb.DIAGNOSTIC_CLASS_CONFLICT, code: code,
				related: []string{prior.emit.GetDefinitionId(), candidate.emit.GetDefinitionId()},
				coord:   proto.CloneOf(candidate.emit.GetCoordinate()), detail: "multiple emits target one coordinate",
			}
		}
		seen[key] = candidate
	}
	return nil
}

func (i *interpreter) validateEmitOrdering(emissions []*emission) *evaluationFailure {
	byID := make(map[string]*emission, len(emissions))
	for _, candidate := range emissions {
		byID[candidate.emit.GetDefinitionId()] = candidate
	}
	for _, ordering := range i.contract.GetObservation().GetOrdering() {
		predecessor := byID[ordering.GetPredecessorEmitDefinitionId()]
		successor := byID[ordering.GetSuccessorEmitDefinitionId()]
		if predecessor == nil || successor == nil || compareCoordinates(
			predecessor.emit.GetCoordinate(), successor.emit.GetCoordinate()) >= 0 {
			return conflictFailure(umpirespb.DIAGNOSTIC_CODE_CAUSAL_ORDER,
				[]string{ordering.GetPredecessorEmitDefinitionId(), ordering.GetSuccessorEmitDefinitionId()},
				"emit ordering contradicts Model coordinates")
		}
	}
	return nil
}

func buildTrace(emissions []*emission) (*umpirespb.ModelTrace, *evaluationFailure) {
	trace := &umpirespb.ModelTrace{}
	byCoordinate := make(map[string]*emission, len(emissions))
	var maximumStep int64
	for _, candidate := range emissions {
		byCoordinate[coordinateKey(candidate.emit.GetCoordinate())] = candidate
		maximumStep = max(maximumStep, candidate.emit.GetCoordinate().GetStep())
	}
	initial := byCoordinate[coordinateKey(&umpirespb.ModelCoordinate{Field: umpirespb.TRACE_FIELD_INITIAL_STATE})]
	if initial == nil {
		return nil, missingTraceCoordinate(umpirespb.TRACE_FIELD_INITIAL_STATE, 0, 0)
	}
	trace.InitialState = proto.CloneOf(initial.value)
	prior := trace.InitialState
	for step := int64(1); step <= maximumStep; step++ {
		action := byCoordinate[coordinateKey(&umpirespb.ModelCoordinate{Field: umpirespb.TRACE_FIELD_SELECTED_ACTION, Step: step})]
		outcome := byCoordinate[coordinateKey(&umpirespb.ModelCoordinate{Field: umpirespb.TRACE_FIELD_MODEL_OUTCOME, Step: step})]
		state := byCoordinate[coordinateKey(&umpirespb.ModelCoordinate{Field: umpirespb.TRACE_FIELD_RESULTING_STATE, Step: step})]
		if action == nil {
			return nil, missingTraceCoordinate(umpirespb.TRACE_FIELD_SELECTED_ACTION, step, 0)
		}
		if outcome == nil {
			return nil, missingTraceCoordinate(umpirespb.TRACE_FIELD_MODEL_OUTCOME, step, 0)
		}
		if state == nil {
			return nil, missingTraceCoordinate(umpirespb.TRACE_FIELD_RESULTING_STATE, step, 0)
		}
		traceStep := &umpirespb.ModelTraceStep{
			Position: step, PriorState: proto.CloneOf(prior),
			SelectedAction: proto.CloneOf(action.value), ModelOutcome: proto.CloneOf(outcome.value),
			ResultingState: proto.CloneOf(state.value),
		}
		for position := int64(1); ; position++ {
			candidate := byCoordinate[coordinateKey(&umpirespb.ModelCoordinate{
				Field: umpirespb.TRACE_FIELD_OBSERVATION, Step: step, Position: position,
			})]
			if candidate == nil {
				break
			}
			traceStep.Observations = append(traceStep.Observations, proto.CloneOf(candidate.value))
		}
		for _, candidate := range emissions {
			coordinate := candidate.emit.GetCoordinate()
			if coordinate.GetField() == umpirespb.TRACE_FIELD_OBSERVATION && coordinate.GetStep() == step &&
				coordinate.GetPosition() > int64(len(traceStep.Observations)) {
				return nil, &evaluationFailure{
					class: umpirespb.DIAGNOSTIC_CLASS_CONFLICT, code: umpirespb.DIAGNOSTIC_CODE_EXTRA_COORDINATE,
					coord: proto.CloneOf(coordinate), detail: "observation coordinates are not contiguous",
				}
			}
		}
		if explicit := byCoordinate[coordinateKey(&umpirespb.ModelCoordinate{
			Field: umpirespb.TRACE_FIELD_PRIOR_STATE, Step: step,
		})]; explicit != nil && !proto.Equal(explicit.value, prior) {
			return nil, &evaluationFailure{
				class: umpirespb.DIAGNOSTIC_CLASS_CONFLICT,
				code:  umpirespb.DIAGNOSTIC_CODE_CONTRADICTORY_COORDINATE,
				coord: proto.CloneOf(explicit.emit.GetCoordinate()), detail: "explicit prior state contradicts trace order",
			}
		}
		trace.Steps = append(trace.Steps, traceStep)
		prior = state.value
	}
	return trace, nil
}

func findFieldDeclaration(kind *umpirespb.EvidenceKindDeclaration, fieldID string) *umpirespb.EvidenceFieldDeclaration {
	for _, field := range kind.GetFields() {
		if field.GetFieldDefinitionId() == fieldID {
			return field
		}
	}
	return nil
}

func recordFields(record *normalizedRecord, reference *umpirespb.EvidenceFieldReference) []*normalizedField {
	matches := make([]*normalizedField, 0, 1)
	if record.fact.KindDefinitionID != reference.GetKindDefinitionId() {
		return matches
	}
	for _, field := range record.fields {
		if field.reference.GetFieldDefinitionId() == reference.GetFieldDefinitionId() {
			matches = append(matches, field)
		}
	}
	return matches
}

func validateCausalOrder(records []*normalizedRecord) *evaluationFailure {
	byID := make(map[string]*normalizedRecord, len(records))
	for _, record := range records {
		byID[record.fact.FactDefinitionID] = record
	}
	for _, record := range records {
		for _, parentID := range record.fact.CausalFactDefinitionIDs {
			parent := byID[parentID]
			if parent == nil || parent == record {
				return conflictFailure(umpirespb.DIAGNOSTIC_CODE_CAUSAL_ORDER,
					[]string{record.fact.FactDefinitionID, parentID}, "causal parent is missing or cyclic")
			}
			if parent.fact.SourceDefinitionID == record.fact.SourceDefinitionID &&
				compareArtifactNaturals(parent.fact.Ordinal, record.fact.Ordinal) >= 0 {
				return conflictFailure(umpirespb.DIAGNOSTIC_CODE_CAUSAL_ORDER,
					[]string{record.fact.FactDefinitionID, parentID}, "causal parent does not precede its child")
			}
		}
	}
	visiting := make(map[string]bool, len(records))
	visited := make(map[string]bool, len(records))
	var visit func(*normalizedRecord) bool
	visit = func(record *normalizedRecord) bool {
		id := record.fact.FactDefinitionID
		if visiting[id] {
			return false
		}
		if visited[id] {
			return true
		}
		visiting[id] = true
		for _, parentID := range record.fact.CausalFactDefinitionIDs {
			if !visit(byID[parentID]) {
				return false
			}
		}
		visiting[id] = false
		visited[id] = true
		return true
	}
	for _, record := range records {
		if !visit(record) {
			return conflictFailure(umpirespb.DIAGNOSTIC_CODE_CAUSAL_ORDER,
				[]string{record.fact.FactDefinitionID}, "causal Evidence contains a cycle")
		}
	}
	return nil
}

func validateCorrelations(slots []*umpirespb.CorrelationSlot, records []*normalizedRecord) *evaluationFailure {
	for _, slot := range slots {
		var expected string
		hasExpected := false
		for _, reference := range slot.GetFields() {
			found := false
			for _, record := range records {
				for _, field := range recordFields(record, reference) {
					value, ok := comparableFieldValue(field)
					if !ok {
						return unsupportedFailure(umpirespb.DIAGNOSTIC_CODE_TYPE_MISMATCH,
							[]string{slot.GetDefinitionId(), reference.GetFieldDefinitionId()},
							"correlation field has no comparable admitted value")
					}
					found = true
					if !hasExpected {
						expected = value
						hasExpected = true
					} else if expected != value {
						return conflictFailure(umpirespb.DIAGNOSTIC_CODE_CORRELATION,
							[]string{slot.GetDefinitionId(), reference.GetFieldDefinitionId()},
							"correlation slot contains contradictory values")
					}
				}
			}
			if !found {
				return &evaluationFailure{
					class: umpirespb.DIAGNOSTIC_CLASS_UNKNOWN, code: umpirespb.DIAGNOSTIC_CODE_MISSING_BINDING,
					related: []string{slot.GetDefinitionId(), reference.GetFieldDefinitionId()},
					detail:  "correlation slot is missing a required field",
				}
			}
		}
	}
	return nil
}

func comparableFieldValue(field *normalizedField) (string, bool) {
	if field.value != nil {
		return valueKey(field.value), true
	}
	if field.digestToken != "" {
		return "digest\x00" + field.digestToken, true
	}
	return "", false
}

func expressionFieldReferences(expression *umpirespb.ObservationExpression) []*umpirespb.EvidenceFieldReference {
	if expression == nil {
		return nil
	}
	switch operator := expression.GetOperator().(type) {
	case *umpirespb.ObservationExpression_Field:
		return []*umpirespb.EvidenceFieldReference{operator.Field}
	case *umpirespb.ObservationExpression_NaturalRenderV1:
		return expressionFieldReferences(operator.NaturalRenderV1.GetOperand())
	case *umpirespb.ObservationExpression_Present:
		return expressionFieldReferences(operator.Present.GetOperand())
	case *umpirespb.ObservationExpression_Equals:
		return append(expressionFieldReferences(operator.Equals.GetLeft()),
			expressionFieldReferences(operator.Equals.GetRight())...)
	case *umpirespb.ObservationExpression_All:
		return operandFieldReferences(operator.All.GetOperands())
	case *umpirespb.ObservationExpression_Any:
		return operandFieldReferences(operator.Any.GetOperands())
	default:
		return nil
	}
}

func operandFieldReferences(operands []*umpirespb.ObservationExpression) []*umpirespb.EvidenceFieldReference {
	var references []*umpirespb.EvidenceFieldReference
	for _, operand := range operands {
		references = append(references, expressionFieldReferences(operand)...)
	}
	return references
}

func uniqueFieldReferences(references []*umpirespb.EvidenceFieldReference) []*umpirespb.EvidenceFieldReference {
	result := make([]*umpirespb.EvidenceFieldReference, 0, len(references))
	seen := make(map[string]struct{}, len(references))
	for _, reference := range references {
		key := reference.GetKindDefinitionId() + "\x00" + reference.GetFieldDefinitionId()
		if _, found := seen[key]; found {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, reference)
	}
	return result
}

func conflictFailure(code umpirespb.DiagnosticCode, related []string, detail string) *evaluationFailure {
	return &evaluationFailure{class: umpirespb.DIAGNOSTIC_CLASS_CONFLICT, code: code, related: related, detail: detail}
}

func unsupportedFailure(code umpirespb.DiagnosticCode, related []string, detail string) *evaluationFailure {
	return &evaluationFailure{class: umpirespb.DIAGNOSTIC_CLASS_UNSUPPORTED, code: code, related: related, detail: detail}
}

func missingClosureFailure(sourceID string) *evaluationFailure {
	return &evaluationFailure{
		class: umpirespb.DIAGNOSTIC_CLASS_UNKNOWN, code: umpirespb.DIAGNOSTIC_CODE_MISSING_CLOSURE,
		related: []string{sourceID}, detail: "contract-required Evidence source is not closed",
	}
}

func typeFailure(related string, coordinate *umpirespb.ModelCoordinate, detail string) *evaluationFailure {
	return &evaluationFailure{
		class: umpirespb.DIAGNOSTIC_CLASS_UNSUPPORTED, code: umpirespb.DIAGNOSTIC_CODE_TYPE_MISMATCH,
		related: []string{related}, coord: proto.CloneOf(coordinate), detail: detail,
	}
}

func missingTraceCoordinate(field umpirespb.TraceField, step, position int64) *evaluationFailure {
	coordinate := &umpirespb.ModelCoordinate{Field: field, Step: step, Position: position}
	return &evaluationFailure{
		class: umpirespb.DIAGNOSTIC_CLASS_UNKNOWN, code: umpirespb.DIAGNOSTIC_CODE_MISSING_COORDINATE,
		coord: coordinate, detail: "trace is missing a required coordinate",
	}
}

func validNatural(value string) bool {
	if value == "" || len(value) > 1 && value[0] == '0' {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func compareNatural(left, right string) int {
	if len(left) != len(right) {
		return len(left) - len(right)
	}
	return strings.Compare(left, right)
}

func compareArtifactNaturals(left, right artifactv2.Natural) int {
	return compareNatural(string(left), string(right))
}

func valueKind(value *umpirespb.Value) string {
	switch value.GetValue().(type) {
	case *umpirespb.Value_Text:
		return "text"
	case *umpirespb.Value_Natural:
		return "natural"
	case *umpirespb.Value_BoolValue:
		return "boolean"
	default:
		return ""
	}
}

func valueKey(value *umpirespb.Value) string {
	switch tagged := value.GetValue().(type) {
	case *umpirespb.Value_Text:
		return "text\x00" + tagged.Text
	case *umpirespb.Value_Natural:
		return "natural\x00" + tagged.Natural
	case *umpirespb.Value_BoolValue:
		return fmt.Sprintf("boolean\x00%t", tagged.BoolValue)
	default:
		return ""
	}
}
