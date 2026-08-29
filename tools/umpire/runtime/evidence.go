package runtime

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

const (
	EvidenceSourceCleanup           = "umpire.evidence.source.cleanup"
	EvidenceSourceControlReceipt    = artifactv2.ControlReceiptSourceDefinitionID
	EvidenceSourceHistory           = "umpire.evidence.source.history"
	EvidenceSourceParticipantOutput = "umpire.evidence.source.participant-output"
)

var evidenceSourceOrder = [...]string{
	EvidenceSourceCleanup,
	EvidenceSourceControlReceipt,
	EvidenceSourceHistory,
	EvidenceSourceParticipantOutput,
}

const (
	EvidenceFieldCancellationCallbackCount = "umpire.evidence.field.cancellation-callback-count"
	EvidenceFieldCommandKind               = "umpire.evidence.field.command-kind"
	EvidenceFieldEndpointIdentity          = "umpire.evidence.field.endpoint-identity"
	EvidenceFieldErrorCode                 = "umpire.evidence.field.error-code"
	EvidenceFieldEventID                   = "umpire.evidence.field.event-id"
	EvidenceFieldEventType                 = "umpire.evidence.field.event-type"
	EvidenceFieldNamespaceIdentity         = "umpire.evidence.field.namespace-identity"
	EvidenceFieldOpenHandleCount           = "umpire.evidence.field.open-handle-count"
	EvidenceFieldOperationCorrelationID    = "umpire.evidence.field.operation-correlation-id"
	EvidenceFieldRunCorrelationID          = "umpire.evidence.field.run-correlation-id"
	EvidenceFieldStatus                    = "umpire.evidence.field.status"
	EvidenceFieldTaskQueueIdentity         = "umpire.evidence.field.task-queue-identity"
	EvidenceFieldWorkflowCorrelationID     = "umpire.evidence.field.workflow-correlation-id"
)

var evidenceFieldAllowlist = map[string]struct{}{
	EvidenceFieldCancellationCallbackCount:               {},
	EvidenceFieldCommandKind:                             {},
	EvidenceFieldEndpointIdentity:                        {},
	EvidenceFieldErrorCode:                               {},
	EvidenceFieldEventID:                                 {},
	EvidenceFieldEventType:                               {},
	EvidenceFieldNamespaceIdentity:                       {},
	EvidenceFieldOpenHandleCount:                         {},
	EvidenceFieldOperationCorrelationID:                  {},
	EvidenceFieldRunCorrelationID:                        {},
	EvidenceFieldStatus:                                  {},
	EvidenceFieldTaskQueueIdentity:                       {},
	EvidenceFieldWorkflowCorrelationID:                   {},
	artifactv2.ControlReceiptActionFieldDefinitionID:     {},
	artifactv2.ControlReceiptAttemptFieldDefinitionID:    {},
	artifactv2.ControlReceiptOccurrenceFieldDefinitionID: {},
}

var numericEvidenceFields = map[string]struct{}{
	EvidenceFieldCancellationCallbackCount:            {},
	EvidenceFieldOpenHandleCount:                      {},
	artifactv2.ControlReceiptAttemptFieldDefinitionID: {},
}

var digestEvidenceFields = map[string]struct{}{
	EvidenceFieldEndpointIdentity:  {},
	EvidenceFieldNamespaceIdentity: {},
	EvidenceFieldTaskQueueIdentity: {},
}

type appendOutcome uint8

const (
	appendRejected appendOutcome = iota
	appendRetained
	appendCapacity
)

type retainedEvidenceFact struct {
	fact    Fact
	ordinal uint64
	bytes   uint64
}

type evidenceSourceState struct {
	definitionID string
	status       string
	closed       bool
	capacity     bool
	byteCount    uint64
	facts        []retainedEvidenceFact
}

type evidenceAccumulator struct {
	limits       map[Phase]PhaseLimit
	phaseRecords map[Phase]uint64
	phaseBytes   map[Phase]uint64
	sources      map[string]*evidenceSourceState
	facts        map[string]retainedEvidenceFact
	knownGaps    []artifactv2.KnownGap
	totalRecords uint64
	totalBytes   uint64
}

func newEvidenceAccumulator(limits []PhaseLimit) *evidenceAccumulator {
	accumulator := &evidenceAccumulator{
		limits:       make(map[Phase]PhaseLimit, len(limits)),
		phaseRecords: make(map[Phase]uint64, len(limits)),
		phaseBytes:   make(map[Phase]uint64, len(limits)),
		sources:      make(map[string]*evidenceSourceState, len(evidenceSourceOrder)),
		facts:        make(map[string]retainedEvidenceFact),
		knownGaps:    []artifactv2.KnownGap{},
	}
	for _, limit := range limits {
		accumulator.limits[limit.phase] = limit
	}
	for _, definitionID := range evidenceSourceOrder {
		accumulator.sources[definitionID] = &evidenceSourceState{
			definitionID: definitionID,
			status:       "partial",
			facts:        []retainedEvidenceFact{},
		}
	}
	return accumulator
}

func (a *evidenceAccumulator) append(phase Phase, fact Fact) (appendOutcome, error) {
	return a.appendFact(phase, fact, false)
}

func (a *evidenceAccumulator) appendControlReceipt(phase Phase, fact Fact) (appendOutcome, error) {
	return a.appendFact(phase, fact, true)
}

func (a *evidenceAccumulator) appendFact(
	phase Phase,
	fact Fact,
	controlReceipt bool,
) (appendOutcome, error) {
	limit, ok := a.limits[phase]
	if !ok {
		return appendRejected, fmt.Errorf("evidence phase %q is not configured", phase)
	}
	source, ok := a.sources[fact.sourceDefinitionID]
	if !ok {
		return appendRejected, fmt.Errorf("evidence fact %q has unknown source", fact.definitionID)
	}
	if source.closed {
		return appendRejected, fmt.Errorf("evidence source %q is already closed", source.definitionID)
	}
	if (fact.sourceDefinitionID == EvidenceSourceControlReceipt) != controlReceipt {
		return appendRejected, fmt.Errorf("control receipt source is engine-owned")
	}
	if _, duplicate := a.facts[fact.definitionID]; duplicate {
		return appendRejected, fmt.Errorf("evidence fact %q is duplicated", fact.definitionID)
	}
	if err := validateEvidenceFields(fact.fields); err != nil {
		return appendRejected, err
	}
	ordinal := uint64(len(source.facts))
	if err := a.validateCauses(source.definitionID, ordinal, fact); err != nil {
		return appendRejected, err
	}
	rawFact, err := rawEvidenceFact(fact, ordinal)
	if err != nil {
		return appendRejected, err
	}
	encoded, err := artifact.CanonicalPretty(rawFact)
	if err != nil {
		return appendRejected, err
	}
	recordBytes := uint64(len(encoded))
	payloadBytes := evidencePayloadBytes(fact.fields)
	if payloadBytes > artifact.MaximumEvidenceFactPayloadBytes {
		return appendRejected, fmt.Errorf("evidence fact %q exceeds payload limit", fact.definitionID)
	}
	if a.phaseRecords[phase]+1 > limit.maxRecords ||
		a.phaseBytes[phase]+recordBytes > limit.maxBytes ||
		a.totalRecords+1 > artifact.MaximumEvidenceFacts ||
		a.totalBytes+recordBytes > artifact.MaximumRawEvidencePayloadBytes {
		a.markCapacity(source)
		return appendCapacity, nil
	}
	retained := retainedEvidenceFact{fact: fact, ordinal: ordinal, bytes: recordBytes}
	source.facts = append(source.facts, retained)
	source.byteCount += recordBytes
	a.facts[fact.definitionID] = retained
	a.phaseRecords[phase]++
	a.phaseBytes[phase] += recordBytes
	a.totalRecords++
	a.totalBytes += recordBytes
	return appendRetained, nil
}

func (a *evidenceAccumulator) validateCauses(
	sourceDefinitionID string,
	ordinal uint64,
	fact Fact,
) error {
	for _, causalID := range fact.causalDefinitionIDs {
		cause, ok := a.facts[causalID]
		if !ok {
			return fmt.Errorf("evidence fact %q has unknown cause %q", fact.definitionID, causalID)
		}
		comparison := strings.Compare(cause.fact.sourceDefinitionID, sourceDefinitionID)
		if comparison > 0 || comparison == 0 && cause.ordinal >= ordinal {
			return fmt.Errorf("evidence fact %q has non-preceding cause %q", fact.definitionID, causalID)
		}
	}
	return nil
}

func (a *evidenceAccumulator) markCapacity(source *evidenceSourceState) {
	if source.capacity {
		return
	}
	source.capacity = true
	subject := source.definitionID
	a.knownGaps = append(a.knownGaps, artifactv2.KnownGap{
		Kind: "input", Code: "umpire.evidence.gap.capacity", Subject: &subject,
	})
}

func (a *evidenceAccumulator) closeSource(sourceDefinitionID string, status string) error {
	source, ok := a.sources[sourceDefinitionID]
	if !ok {
		return fmt.Errorf("evidence source %q is unknown", sourceDefinitionID)
	}
	if source.closed {
		return fmt.Errorf("evidence source %q closed more than once", sourceDefinitionID)
	}
	switch status {
	case "closed", "partial", "failed":
	default:
		return fmt.Errorf("evidence source status %q is invalid", status)
	}
	if source.capacity && status == "closed" {
		status = "partial"
	}
	source.status = status
	source.closed = true
	return nil
}

func (a *evidenceAccumulator) retainedCount() uint64 {
	return a.totalRecords
}

func (a *evidenceAccumulator) materialize() (
	[]artifactv2.RawEvidenceSource,
	[]artifactv2.RawEvidenceFact,
	[]artifactv2.KnownGap,
) {
	sources := make([]artifactv2.RawEvidenceSource, 0, len(evidenceSourceOrder))
	facts := make([]artifactv2.RawEvidenceFact, 0, a.totalRecords)
	for _, definitionID := range evidenceSourceOrder {
		source := a.sources[definitionID]
		status := source.status
		if !source.closed {
			status = "partial"
		}
		sources = append(sources, artifactv2.RawEvidenceSource{
			SourceDefinitionID: definitionID,
			Status:             status,
			FactCount:          artifactv2.NaturalFromUint64(uint64(len(source.facts))),
			ByteCount:          artifactv2.NaturalFromUint64(source.byteCount),
		})
		for _, retained := range source.facts {
			raw, _ := rawEvidenceFact(retained.fact, retained.ordinal)
			facts = append(facts, raw)
		}
	}
	gaps := slices.Clone(a.knownGaps)
	slices.SortFunc(gaps, func(left, right artifactv2.KnownGap) int {
		if comparison := strings.Compare(left.Kind, right.Kind); comparison != 0 {
			return comparison
		}
		if comparison := strings.Compare(left.Code, right.Code); comparison != 0 {
			return comparison
		}
		return strings.Compare(stringPointerValue(left.Subject), stringPointerValue(right.Subject))
	})
	return sources, facts, gaps
}

func rawEvidenceFact(fact Fact, ordinal uint64) (artifactv2.RawEvidenceFact, error) {
	fields := make([]artifactv2.RawEvidenceField, len(fact.fields))
	for index, field := range fact.fields {
		disposition := "plain"
		value := any(field.value)
		if _, numeric := numericEvidenceFields[field.definitionID]; numeric {
			if _, err := strconv.ParseUint(field.value, 10, 64); err != nil {
				return artifactv2.RawEvidenceFact{}, fmt.Errorf("numeric evidence field %q is invalid", field.definitionID)
			}
			value = json.Number(field.value)
		}
		if _, digest := digestEvidenceFields[field.definitionID]; digest {
			if !artifactv2.ValidDigest(field.value) {
				return artifactv2.RawEvidenceFact{}, fmt.Errorf("digest evidence field %q is invalid", field.definitionID)
			}
			disposition = "sha256"
		}
		fields[index] = artifactv2.RawEvidenceField{
			FieldDefinitionID: field.definitionID,
			Disposition:       disposition,
			Value:             value,
		}
	}
	return artifactv2.RawEvidenceFact{
		FactDefinitionID: fact.definitionID, SourceDefinitionID: fact.sourceDefinitionID,
		Ordinal: artifactv2.NaturalFromUint64(ordinal), KindDefinitionID: fact.kindDefinitionID,
		CausalFactDefinitionIDs: slices.Clone(fact.causalDefinitionIDs), Fields: fields,
	}, nil
}

func validateEvidenceFields(fields []FactField) error {
	for _, field := range fields {
		if _, ok := evidenceFieldAllowlist[field.definitionID]; !ok {
			return fmt.Errorf("evidence field %q is not allowlisted", field.definitionID)
		}
		if _, numeric := numericEvidenceFields[field.definitionID]; numeric {
			if _, err := strconv.ParseUint(field.value, 10, 64); err != nil {
				return fmt.Errorf("numeric evidence field %q is invalid", field.definitionID)
			}
		}
		if _, digest := digestEvidenceFields[field.definitionID]; digest && !artifactv2.ValidDigest(field.value) {
			return fmt.Errorf("digest evidence field %q is invalid", field.definitionID)
		}
	}
	return nil
}

func evidencePayloadBytes(fields []FactField) int {
	total := 0
	for _, field := range fields {
		total += len(field.value)
	}
	return total
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
