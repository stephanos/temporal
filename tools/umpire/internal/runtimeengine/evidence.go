package runtimeengine

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

const (
	EvidenceSourceCleanup           = umpireruntime.EvidenceSourceCleanup
	EvidenceSourceControlReceipt    = umpireruntime.EvidenceSourceControlReceipt
	EvidenceSourceHistory           = umpireruntime.EvidenceSourceHistory
	EvidenceSourceParticipantOutput = umpireruntime.EvidenceSourceParticipantOutput
)

var evidenceSourceOrder = [...]string{
	EvidenceSourceCleanup,
	EvidenceSourceControlReceipt,
	EvidenceSourceHistory,
	EvidenceSourceParticipantOutput,
}

const (
	EvidenceFieldCancellationCallbackCount = umpireruntime.EvidenceFieldCancellationCallbackCount
	EvidenceFieldCommandKind               = umpireruntime.EvidenceFieldCommandKind
	EvidenceFieldEndpointIdentity          = umpireruntime.EvidenceFieldEndpointIdentity
	EvidenceFieldErrorCode                 = umpireruntime.EvidenceFieldErrorCode
	EvidenceFieldEventID                   = umpireruntime.EvidenceFieldEventID
	EvidenceFieldEventType                 = umpireruntime.EvidenceFieldEventType
	EvidenceFieldNamespaceIdentity         = umpireruntime.EvidenceFieldNamespaceIdentity
	EvidenceFieldOpenHandleCount           = umpireruntime.EvidenceFieldOpenHandleCount
	EvidenceFieldOperationCorrelationID    = umpireruntime.EvidenceFieldOperationCorrelationID
	EvidenceFieldRunCorrelationID          = umpireruntime.EvidenceFieldRunCorrelationID
	EvidenceFieldStatus                    = umpireruntime.EvidenceFieldStatus
	EvidenceFieldTaskQueueIdentity         = umpireruntime.EvidenceFieldTaskQueueIdentity
	EvidenceFieldWorkflowCorrelationID     = umpireruntime.EvidenceFieldWorkflowCorrelationID
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
	EvidenceFieldEventID:                              {},
	EvidenceFieldOpenHandleCount:                      {},
	artifactv2.ControlReceiptAttemptFieldDefinitionID: {},
}

var digestEvidenceFields = map[string]struct{}{
	EvidenceFieldEndpointIdentity:  {},
	EvidenceFieldNamespaceIdentity: {},
	EvidenceFieldTaskQueueIdentity: {},
}

// appendOutcome reports whether one evidence fact was rejected, retained, or
// stopped at the configured capacity boundary.
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

type evidenceLimit struct {
	maxRecords uint64
	maxBytes   uint64
}

// evidenceAccumulator retains bounded mechanical evidence for the internal engine.
type evidenceAccumulator struct {
	limits       map[Phase]evidenceLimit
	fieldRules   evidenceFieldRules
	phaseRecords map[Phase]uint64
	phaseBytes   map[Phase]uint64
	sources      map[string]*evidenceSourceState
	facts        map[string]retainedEvidenceFact
	knownGaps    []artifactv2.KnownGap
	totalRecords uint64
	totalBytes   uint64
}

type evidenceFieldRules struct {
	exactValues map[string]map[string]struct{}
}

// newEvidenceAccumulator creates the request-bound evidence collector used by
// the internal execution engine.
func newEvidenceAccumulator(request CheckedRunRequest) *evidenceAccumulator {
	limits := request.PhaseLimits()
	accumulator := &evidenceAccumulator{
		limits:       make(map[Phase]evidenceLimit, len(limits)),
		fieldRules:   newEvidenceFieldRules(request),
		phaseRecords: make(map[Phase]uint64, len(limits)),
		phaseBytes:   make(map[Phase]uint64, len(limits)),
		sources:      make(map[string]*evidenceSourceState, len(evidenceSourceOrder)),
		facts:        make(map[string]retainedEvidenceFact),
		knownGaps:    []artifactv2.KnownGap{},
	}
	for _, limit := range limits {
		accumulator.limits[limit.Phase()] = evidenceLimit{
			maxRecords: limit.MaxRecords(), maxBytes: limit.MaxBytes(),
		}
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

func newEvidenceFieldRules(request CheckedRunRequest) evidenceFieldRules {
	exactValues := map[string]map[string]struct{}{
		EvidenceFieldCommandKind: valueSet(
			string(CommandPrepare), string(CommandRealize), string(CommandObserve),
			string(commandIsolate), string(CommandCleanup),
		),
		EvidenceFieldErrorCode: valueSet(
			"umpire.runtime.code.canceled",
			"umpire.runtime.code.capacity",
			"umpire.runtime.code.cleanup-failed",
			"umpire.runtime.code.failed",
			"umpire.runtime.code.rejected",
			"umpire.runtime.code.timed-out",
			"umpire.runtime.code.unsupported",
		),
		EvidenceFieldRunCorrelationID: valueSet(request.RunIdentity()),
		EvidenceFieldStatus: valueSet(
			"accepted", "canceled", "closed", "complete", "failed", "incomplete",
			"not-attempted", "not-started", "partial", "rejected", "succeeded",
			"timed-out", "unsupported",
		),
		artifactv2.ControlReceiptActionFieldDefinitionID: valueSet(
			request.Program().Occurrence().ActionDefinitionID(),
		),
		artifactv2.ControlReceiptAttemptFieldDefinitionID: valueSet(
			strconv.FormatUint(request.Attempt(), 10),
		),
		artifactv2.ControlReceiptOccurrenceFieldDefinitionID: valueSet(
			request.Program().Occurrence().DefinitionID(),
		),
	}
	for _, correlation := range request.Correlations() {
		switch correlation.Kind() {
		case CorrelationWorkflow:
			exactValues[EvidenceFieldWorkflowCorrelationID] = valueSet(correlation.Identity())
		case CorrelationOperation:
			exactValues[EvidenceFieldOperationCorrelationID] = valueSet(correlation.Identity())
		}
	}
	return evidenceFieldRules{exactValues: exactValues}
}

func valueSet(values ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
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
	source, ok := a.sources[fact.SourceDefinitionID()]
	if !ok {
		return appendRejected, fmt.Errorf("evidence fact %q has unknown source", fact.DefinitionID())
	}
	if source.closed {
		return appendRejected, fmt.Errorf("evidence source %q is already closed", source.definitionID)
	}
	if (fact.SourceDefinitionID() == EvidenceSourceControlReceipt) != controlReceipt {
		return appendRejected, fmt.Errorf("control receipt source is engine-owned")
	}
	if _, duplicate := a.facts[fact.DefinitionID()]; duplicate {
		return appendRejected, fmt.Errorf("evidence fact %q is duplicated", fact.DefinitionID())
	}
	if err := a.validateEvidenceFields(fact.Fields()); err != nil {
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
	payloadBytes := evidencePayloadBytes(fact.Fields())
	if payloadBytes > artifact.MaximumEvidenceFactPayloadBytes {
		return appendRejected, fmt.Errorf("evidence fact %q exceeds payload limit", fact.DefinitionID())
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
	a.facts[fact.DefinitionID()] = retained
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
	for _, causalID := range fact.CausalDefinitionIDs() {
		cause, ok := a.facts[causalID]
		if !ok {
			return fmt.Errorf("evidence fact %q has unknown cause %q", fact.DefinitionID(), causalID)
		}
		comparison := strings.Compare(cause.fact.SourceDefinitionID(), sourceDefinitionID)
		if comparison > 0 || comparison == 0 && cause.ordinal >= ordinal {
			return fmt.Errorf("evidence fact %q has non-preceding cause %q", fact.DefinitionID(), causalID)
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

func (a *evidenceAccumulator) markSourceCapacity(sourceDefinitionID string) error {
	source, ok := a.sources[sourceDefinitionID]
	if !ok {
		return fmt.Errorf("evidence source %q is unknown", sourceDefinitionID)
	}
	if source.closed {
		return fmt.Errorf("evidence source %q is already closed", sourceDefinitionID)
	}
	a.markCapacity(source)
	return nil
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
	fields := make([]artifactv2.RawEvidenceField, len(fact.Fields()))
	for index, field := range fact.Fields() {
		disposition := "plain"
		value := any(field.Value())
		if _, numeric := numericEvidenceFields[field.DefinitionID()]; numeric {
			if _, err := strconv.ParseUint(field.Value(), 10, 64); err != nil {
				return artifactv2.RawEvidenceFact{}, fmt.Errorf("numeric evidence field %q is invalid", field.DefinitionID())
			}
			value = json.Number(field.Value())
		}
		if _, digest := digestEvidenceFields[field.DefinitionID()]; digest {
			if !artifactv2.ValidDigest(field.Value()) {
				return artifactv2.RawEvidenceFact{}, fmt.Errorf("digest evidence field %q is invalid", field.DefinitionID())
			}
			disposition = "sha256"
		}
		fields[index] = artifactv2.RawEvidenceField{
			FieldDefinitionID: field.DefinitionID(),
			Disposition:       disposition,
			Value:             value,
		}
	}
	return artifactv2.RawEvidenceFact{
		FactDefinitionID: fact.DefinitionID(), SourceDefinitionID: fact.SourceDefinitionID(),
		Ordinal: artifactv2.NaturalFromUint64(ordinal), KindDefinitionID: fact.KindDefinitionID(),
		CausalFactDefinitionIDs: slices.Clone(fact.CausalDefinitionIDs()), Fields: fields,
	}, nil
}

func (a *evidenceAccumulator) validateEvidenceFields(fields []FactField) error {
	for _, field := range fields {
		if _, ok := evidenceFieldAllowlist[field.DefinitionID()]; !ok {
			return fmt.Errorf("evidence field %q is not allowlisted", field.DefinitionID())
		}
		if _, numeric := numericEvidenceFields[field.DefinitionID()]; numeric {
			if _, err := strconv.ParseUint(field.Value(), 10, 64); err != nil {
				return fmt.Errorf("numeric evidence field %q is invalid", field.DefinitionID())
			}
		}
		if _, digest := digestEvidenceFields[field.DefinitionID()]; digest && !artifactv2.ValidDigest(field.Value()) {
			return fmt.Errorf("digest evidence field %q is invalid", field.DefinitionID())
		}
		if field.DefinitionID() == EvidenceFieldEventType && !validIdentity(field.Value()) {
			return fmt.Errorf("evidence event type %q is invalid", field.Value())
		}
		if values, exact := a.fieldRules.exactValues[field.DefinitionID()]; exact {
			if _, allowed := values[field.Value()]; !allowed {
				return fmt.Errorf("evidence field %q is not request-bound or closed", field.DefinitionID())
			}
		}
	}
	return nil
}

func evidencePayloadBytes(fields []FactField) int {
	total := 0
	for _, field := range fields {
		total += len(field.Value())
	}
	return total
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func validIdentity(value string) bool {
	if len(value) == 0 || len(value) > MaximumIdentityBytes {
		return false
	}
	segments := strings.Split(value, ".")
	if len(segments) < 2 {
		return false
	}
	for _, segment := range segments {
		if segment == "" {
			return false
		}
		for _, character := range []byte(segment) {
			if !asciiAlphaNumeric(character) && character != '-' && character != '_' {
				return false
			}
		}
	}
	return true
}

func asciiAlphaNumeric(character byte) bool {
	return character >= 'a' && character <= 'z' ||
		character >= 'A' && character <= 'Z' ||
		character >= '0' && character <= '9'
}
