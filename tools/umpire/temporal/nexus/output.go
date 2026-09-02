package nexus

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"

	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

var errExecutionClosure = errors.New("umpire.temporal.nexus.invariant.execution-closure")
var errCleanupLeakage = errors.New("closed cleanup does not prove zero open handles")

type executionClosureFailure struct {
	kind  string
	phase string
	code  string
}

func (*executionClosureFailure) Error() string {
	return errExecutionClosure.Error()
}

func (failure *executionClosureFailure) Kind() string {
	if failure == nil {
		return ""
	}
	return failure.kind
}

func (failure *executionClosureFailure) Phase() string {
	if failure == nil {
		return ""
	}
	return failure.phase
}

func (failure *executionClosureFailure) Code() string {
	if failure == nil {
		return ""
	}
	return failure.code
}

func (*executionClosureFailure) Is(target error) bool {
	return target == errExecutionClosure
}

func classifyExecutionClosure(err error) error {
	if errors.Is(err, errCleanupLeakage) {
		return &executionClosureFailure{
			kind:  "output-invariant",
			phase: "cleanup",
			code:  "umpire.temporal.nexus.execution-closure.cleanup-leakage",
		}
	}
	return errExecutionClosure
}

var retainedFieldDisposition = map[string]string{
	umpireruntime.EvidenceFieldCancellationCallbackCount:   "number",
	umpireruntime.EvidenceFieldCancellationCompletedCount:  "number",
	umpireruntime.EvidenceFieldCancellationRequestedCount:  "number",
	umpireruntime.EvidenceFieldCapabilityDefinitionID:      "string",
	umpireruntime.EvidenceFieldCommandKind:                 "string",
	umpireruntime.EvidenceFieldEndpointIdentity:            "sha256",
	umpireruntime.EvidenceFieldErrorCode:                   "string",
	umpireruntime.EvidenceFieldEventID:                     "number",
	umpireruntime.EvidenceFieldEventType:                   "string",
	umpireruntime.EvidenceFieldFaultDefinitionID:           "string",
	umpireruntime.EvidenceFieldFaultReceiptDefinitionID:    "string",
	umpireruntime.EvidenceFieldNamespaceIdentity:           "sha256",
	umpireruntime.EvidenceFieldOpenHandleCount:             "number",
	umpireruntime.EvidenceFieldOperationCorrelationID:      "string",
	umpireruntime.EvidenceFieldRunCorrelationID:            "string",
	umpireruntime.EvidenceFieldStatus:                      "string",
	umpireruntime.EvidenceFieldSyntheticContributionCount:  "number",
	umpireruntime.EvidenceFieldSyntheticContributionMarker: "string",
	umpireruntime.EvidenceFieldTaskQueueIdentity:           "sha256",
	umpireruntime.EvidenceFieldWorkflowCorrelationID:       "string",
	artifactv2.ControlReceiptActionFieldDefinitionID:       "string",
	artifactv2.ControlReceiptAttemptFieldDefinitionID:      "number",
	artifactv2.ControlReceiptOccurrenceFieldDefinitionID:   "string",
}

var retainedErrorCodes = map[string]struct{}{
	runtimeCodeCanceled:                  {},
	runtimeCodeFailed:                    {},
	runtimeCodeRejected:                  {},
	runtimeCodeTimedOut:                  {},
	runtimeCodeUnsupported:               {},
	"umpire.runtime.code.capacity":       {},
	"umpire.runtime.code.cleanup-failed": {},
}

var exactSourceOrder = [...]string{
	umpireruntime.EvidenceSourceCleanup,
	umpireruntime.EvidenceSourceControlReceipt,
	umpireruntime.EvidenceSourceHistory,
	umpireruntime.EvidenceSourceParticipantOutput,
}

var requiredTerminalEvents = [...]string{
	"temporal.history.NexusOperationScheduled",
	"temporal.history.NexusOperationStarted",
	"temporal.history.NexusOperationCancelRequested",
	"temporal.history.NexusOperationCancelRequestCompleted",
	"temporal.history.WorkflowExecutionCanceled",
}

var duplicateDeliverySyntheticFieldOrder = [...]string{
	umpireruntime.EvidenceFieldCancellationCallbackCount,
	umpireruntime.EvidenceFieldCancellationCompletedCount,
	umpireruntime.EvidenceFieldCancellationRequestedCount,
	umpireruntime.EvidenceFieldCapabilityDefinitionID,
	umpireruntime.EvidenceFieldCommandKind,
	umpireruntime.EvidenceFieldFaultDefinitionID,
	umpireruntime.EvidenceFieldFaultReceiptDefinitionID,
	umpireruntime.EvidenceFieldOperationCorrelationID,
	umpireruntime.EvidenceFieldRunCorrelationID,
	umpireruntime.EvidenceFieldStatus,
	umpireruntime.EvidenceFieldSyntheticContributionCount,
	umpireruntime.EvidenceFieldSyntheticContributionMarker,
	umpireruntime.EvidenceFieldWorkflowCorrelationID,
}

var duplicateDeliveryOnlyFields = map[string]struct{}{
	umpireruntime.EvidenceFieldCancellationCompletedCount:  {},
	umpireruntime.EvidenceFieldCancellationRequestedCount:  {},
	umpireruntime.EvidenceFieldCapabilityDefinitionID:      {},
	umpireruntime.EvidenceFieldFaultDefinitionID:           {},
	umpireruntime.EvidenceFieldFaultReceiptDefinitionID:    {},
	umpireruntime.EvidenceFieldSyntheticContributionCount:  {},
	umpireruntime.EvidenceFieldSyntheticContributionMarker: {},
}

var faultEvaluationDispositionFields = map[string]struct{}{
	umpireruntime.EvidenceFieldCancellationCallbackCount:   {},
	umpireruntime.EvidenceFieldCancellationCompletedCount:  {},
	umpireruntime.EvidenceFieldCancellationRequestedCount:  {},
	umpireruntime.EvidenceFieldCapabilityDefinitionID:      {},
	umpireruntime.EvidenceFieldCommandKind:                 {},
	umpireruntime.EvidenceFieldEventType:                   {},
	umpireruntime.EvidenceFieldFaultDefinitionID:           {},
	umpireruntime.EvidenceFieldFaultReceiptDefinitionID:    {},
	umpireruntime.EvidenceFieldOperationCorrelationID:      {},
	umpireruntime.EvidenceFieldRunCorrelationID:            {},
	umpireruntime.EvidenceFieldStatus:                      {},
	umpireruntime.EvidenceFieldSyntheticContributionCount:  {},
	umpireruntime.EvidenceFieldSyntheticContributionMarker: {},
	umpireruntime.EvidenceFieldWorkflowCorrelationID:       {},
}

// The exact checked caller-closure request executes through the digest-bound
// runner and returns one admitted in-memory four-member output. It performs no
// publication or interpretation.
func validateExecutionClosure(
	executable artifact.ExecutableSet,
	admitted artifact.AdmittedSet,
	run artifactv2.ExperimentRun,
	rawEvidence artifactv2.RawEvidence,
) error {
	runBytes, err := artifact.EncodeExperimentRunV2(run)
	if err != nil {
		return fmt.Errorf("admit Run: %w", err)
	}
	if err := requirePrettyLine(runBytes); err != nil {
		return err
	}
	rawEvidenceBytes, err := artifact.EncodeRawEvidenceV2(rawEvidence)
	if err != nil {
		return fmt.Errorf("admit RawEvidence: %w", err)
	}
	if err := requirePrettyLine(rawEvidenceBytes); err != nil {
		return err
	}
	experiment := executable.Experiment()
	configuration := executable.RuntimeConfiguration()
	if err := artifact.ValidateRawEvidenceV2Closure(
		rawEvidence,
		experiment,
		configuration,
		run,
	); err != nil {
		return err
	}
	faulted := configuration.ConfigurationDefinitionID == duplicateDeliveryConfigurationDefinitionID
	if err := validateMechanicalEvidence(run, rawEvidence, faulted); err != nil {
		return err
	}
	if !faulted {
		if err := rejectDuplicateDeliveryEvidence(rawEvidence); err != nil {
			return err
		}
	}
	expected, err := executable.AdmitExecution(run, rawEvidence)
	if err != nil {
		return err
	}
	if admitted.Identity() != expected.Identity() ||
		admitted.Checksum() != expected.Checksum() ||
		admitted.ManifestSHA256() != expected.ManifestSHA256() ||
		!bytes.Equal(admitted.ManifestBytes(), expected.ManifestBytes()) {
		return errors.New("output set is not the exact artifact-owned execution extension")
	}
	return requirePrettyLine(admitted.ManifestBytes())
}

func rejectDuplicateDeliveryEvidence(rawEvidence artifactv2.RawEvidence) error {
	for _, fact := range rawEvidence.Facts {
		if fact.KindDefinitionID == umpireruntime.EvidenceKindParticipantCommandSyntheticDuplicate {
			return errors.New("normal execution contains a synthetic duplicate contribution")
		}
		for _, field := range fact.Fields {
			if _, faultOnly := duplicateDeliveryOnlyFields[field.FieldDefinitionID]; faultOnly {
				return fmt.Errorf("normal execution contains fault-only field %q", field.FieldDefinitionID)
			}
		}
	}
	return nil
}

func rawEvidenceFactsWithField(
	rawEvidence artifactv2.RawEvidence,
	fieldDefinitionID string,
) []artifactv2.RawEvidenceFact {
	facts := make([]artifactv2.RawEvidenceFact, 0, 1)
	for _, fact := range rawEvidence.Facts {
		for _, field := range fact.Fields {
			if field.FieldDefinitionID == fieldDefinitionID {
				facts = append(facts, fact)
				break
			}
		}
	}
	return facts
}

func exactHistoryEventFact(
	rawEvidence artifactv2.RawEvidence,
	eventType string,
) (artifactv2.RawEvidenceFact, error) {
	matches := make([]artifactv2.RawEvidenceFact, 0, 1)
	for _, fact := range rawEvidence.Facts {
		if fact.SourceDefinitionID != umpireruntime.EvidenceSourceHistory {
			continue
		}
		actual, err := rawStringField(fact, umpireruntime.EvidenceFieldEventType)
		if err == nil && actual == eventType {
			matches = append(matches, fact)
		}
	}
	if len(matches) != 1 {
		return artifactv2.RawEvidenceFact{}, fmt.Errorf(
			"faulted execution has %d %s history facts",
			len(matches),
			eventType,
		)
	}
	return matches[0], nil
}

func rawEvidenceFieldDefinitionIDs(fact artifactv2.RawEvidenceFact) []string {
	definitionIDs := make([]string, len(fact.Fields))
	for index, field := range fact.Fields {
		definitionIDs[index] = field.FieldDefinitionID
	}
	return definitionIDs
}

func requestCorrelations(request umpireruntime.CheckedRunRequest) adapterCorrelations {
	result := adapterCorrelations{}
	for _, correlation := range request.Correlations() {
		switch correlation.Kind() {
		case umpireruntime.CorrelationWorkflow:
			result.workflow = correlation.Identity()
		case umpireruntime.CorrelationOperation:
			result.operation = correlation.Identity()
		default:
			continue
		}
	}
	return result
}

func requirePrettyLine(encoded []byte) error {
	if len(encoded) < 2 || encoded[0] != '{' || encoded[len(encoded)-1] != '\n' ||
		encoded[len(encoded)-2] == '\n' || !bytes.Contains(encoded, []byte("\n  \"")) {
		return errors.New("artifact is not deterministic pretty JSON with one terminal line feed")
	}
	return nil
}

func validateMechanicalEvidence(
	run artifactv2.ExperimentRun,
	rawEvidence artifactv2.RawEvidence,
	faulted bool,
) error {
	if len(rawEvidence.Sources) != len(exactSourceOrder) ||
		len(run.SourceClosures) != len(exactSourceOrder) {
		return errors.New("execution evidence does not have exactly four sources")
	}
	for index, sourceDefinitionID := range exactSourceOrder {
		if rawEvidence.Sources[index].SourceDefinitionID != sourceDefinitionID ||
			run.SourceClosures[index].SourceDefinitionID != sourceDefinitionID {
			return errors.New("execution evidence sources are not the exact closed set")
		}
	}
	if len(run.ControlAttempts) != 1 {
		return errors.New("execution evidence must bind exactly one control attempt")
	}
	if !equalKnownGaps(run.KnownGaps, rawEvidence.KnownGaps) {
		return errors.New("Run and RawEvidence known gaps diverge")
	}
	if run.OperationalStatus == "succeeded" &&
		(len(run.KnownGaps) != 0 || rawEvidence.CaptureStatus != "closed") {
		return errors.New("successful execution cannot contain a gap or partial capture")
	}
	if err := validateSourceByteCounts(rawEvidence); err != nil {
		return err
	}
	for _, fact := range rawEvidence.Facts {
		if err := validateRetainedFields(fact, faulted); err != nil {
			return err
		}
	}
	if err := validateHistoryClosure(rawEvidence, !faulted); err != nil {
		return err
	}
	return validateCleanupClosure(run, rawEvidence)
}

func validateSourceByteCounts(rawEvidence artifactv2.RawEvidence) error {
	byteCounts := make(map[string]uint64, len(rawEvidence.Sources))
	for _, fact := range rawEvidence.Facts {
		encoded, err := artifact.CanonicalPretty(fact)
		if err != nil {
			return fmt.Errorf("encode evidence fact: %w", err)
		}
		byteCounts[fact.SourceDefinitionID] += uint64(len(encoded))
	}
	for _, source := range rawEvidence.Sources {
		if source.ByteCount.String() != strconv.FormatUint(byteCounts[source.SourceDefinitionID], 10) {
			return fmt.Errorf("evidence source %q byte count is not exact", source.SourceDefinitionID)
		}
	}
	return nil
}

func equalKnownGaps(left []artifactv2.KnownGap, right []artifactv2.KnownGap) bool {
	return slices.EqualFunc(left, right, func(left artifactv2.KnownGap, right artifactv2.KnownGap) bool {
		return left.Kind == right.Kind && left.Code == right.Code &&
			optionalString(left.Subject) == optionalString(right.Subject) &&
			optionalString(left.Detail) == optionalString(right.Detail)
	})
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func validateRetainedFields(fact artifactv2.RawEvidenceFact, faulted bool) error {
	for _, field := range fact.Fields {
		kind, allowed := retainedFieldDisposition[field.FieldDefinitionID]
		if !allowed {
			return fmt.Errorf("evidence field %q is not allowlisted", field.FieldDefinitionID)
		}
		if faulted && faultEvaluationDispositionMayBeOpaque(fact, field.FieldDefinitionID) {
			continue
		}
		switch kind {
		case "sha256":
			value, ok := field.Value.(string)
			if field.Disposition != "sha256" || !ok || !artifactv2.ValidDigest(value) {
				return fmt.Errorf("evidence field %q is not one digest token", field.FieldDefinitionID)
			}
		case "number":
			value, ok := field.Value.(json.Number)
			if field.Disposition != "plain" || !ok {
				return fmt.Errorf("evidence field %q is not one plain natural", field.FieldDefinitionID)
			}
			if _, err := strconv.ParseUint(string(value), 10, 64); err != nil {
				return fmt.Errorf("evidence field %q is not one plain natural", field.FieldDefinitionID)
			}
		case "string":
			value, ok := field.Value.(string)
			if field.Disposition != "plain" || !ok || value == "" {
				return fmt.Errorf("evidence field %q is not one plain closed string", field.FieldDefinitionID)
			}
			if field.FieldDefinitionID == umpireruntime.EvidenceFieldErrorCode {
				if _, ok := retainedErrorCodes[value]; !ok {
					return fmt.Errorf("evidence error code %q is not closed", value)
				}
			}
		}
	}
	return nil
}

func faultEvaluationDispositionMayBeOpaque(
	fact artifactv2.RawEvidenceFact,
	fieldDefinitionID string,
) bool {
	if _, semantic := faultEvaluationDispositionFields[fieldDefinitionID]; !semantic {
		return false
	}
	switch fact.SourceDefinitionID {
	case umpireruntime.EvidenceSourceParticipantOutput:
		return true
	case umpireruntime.EvidenceSourceHistory:
		return fieldDefinitionID == umpireruntime.EvidenceFieldEventType ||
			fieldDefinitionID == umpireruntime.EvidenceFieldOperationCorrelationID ||
			fieldDefinitionID == umpireruntime.EvidenceFieldRunCorrelationID ||
			fieldDefinitionID == umpireruntime.EvidenceFieldWorkflowCorrelationID
	default:
		return false
	}
}

type historyFactSummary struct {
	eventCounts         map[string]uint64
	lastEventType       string
	allEventTypesUsable bool
}

func validateHistoryClosure(rawEvidence artifactv2.RawEvidence, requireCallerClosure bool) error {
	historyStatus, historyFacts := historyEvidence(rawEvidence)
	summary, err := validateHistoryFactChain(historyFacts, requireCallerClosure)
	if err != nil {
		return err
	}
	if historyStatus != "closed" {
		return nil
	}
	if !requireCallerClosure {
		return validateFaultedLifecycleCounts(summary)
	}
	if err := validateCallerClosureLifecycle(summary); err != nil {
		return err
	}
	return validateCancellationCallbackCount(rawEvidence)
}

func historyEvidence(
	rawEvidence artifactv2.RawEvidence,
) (historyStatus string, historyFacts []artifactv2.RawEvidenceFact) {
	for _, source := range rawEvidence.Sources {
		if source.SourceDefinitionID == umpireruntime.EvidenceSourceHistory {
			historyStatus = source.Status
		}
	}
	for _, fact := range rawEvidence.Facts {
		if fact.SourceDefinitionID == umpireruntime.EvidenceSourceHistory {
			historyFacts = append(historyFacts, fact)
		}
	}
	return historyStatus, historyFacts
}

func validateHistoryFactChain(
	historyFacts []artifactv2.RawEvidenceFact,
	requireEventTypes bool,
) (historyFactSummary, error) {
	previousFact := ""
	previousEventID := uint64(0)
	summary := historyFactSummary{
		eventCounts:         make(map[string]uint64),
		allEventTypesUsable: true,
	}
	for index, fact := range historyFacts {
		if fact.KindDefinitionID != "umpire.evidence.kind.workflow-history-event" {
			return historyFactSummary{}, errors.New("history fact has the wrong mechanical kind")
		}
		if index == 0 {
			if len(fact.CausalFactDefinitionIDs) != 0 {
				return historyFactSummary{}, errors.New("first history fact has a causal predecessor")
			}
		} else if !slices.Equal(fact.CausalFactDefinitionIDs, []string{previousFact}) {
			return historyFactSummary{}, errors.New("history facts do not form one gapless causal chain")
		}
		eventID, err := rawNaturalField(fact, umpireruntime.EvidenceFieldEventID)
		if err != nil || eventID <= previousEventID {
			return historyFactSummary{}, errors.New("history event IDs are not strictly increasing")
		}
		eventType, err := rawStringField(fact, umpireruntime.EvidenceFieldEventType)
		if err != nil {
			if requireEventTypes {
				return historyFactSummary{}, err
			}
			summary.allEventTypesUsable = false
		} else {
			summary.lastEventType = eventType
			summary.eventCounts[eventType]++
		}
		previousFact = fact.FactDefinitionID
		previousEventID = eventID
	}
	return summary, nil
}

func validateCallerClosureLifecycle(summary historyFactSummary) error {
	if summary.lastEventType != "temporal.history.WorkflowExecutionCanceled" {
		return errors.New("closed history is not terminal caller cancellation")
	}
	for _, eventType := range requiredTerminalEvents {
		if summary.eventCounts[eventType] != 1 {
			return fmt.Errorf(
				"closed history has %d %s events",
				summary.eventCounts[eventType],
				eventType,
			)
		}
	}
	return nil
}

func validateCancellationCallbackCount(rawEvidence artifactv2.RawEvidence) error {
	callbacks := uint64(0)
	callbackFields := 0
	for _, fact := range rawEvidence.Facts {
		for _, field := range fact.Fields {
			if field.FieldDefinitionID != umpireruntime.EvidenceFieldCancellationCallbackCount {
				continue
			}
			value, _ := field.Value.(json.Number)
			parsed, err := strconv.ParseUint(string(value), 10, 64)
			if err != nil {
				return errors.New("cancellation callback count is malformed")
			}
			callbacks = parsed
			callbackFields++
		}
	}
	if callbackFields != 1 || callbacks != 1 {
		return errors.New("closed history requires exactly one cancellation callback")
	}
	return nil
}

func validateFaultedLifecycleCounts(
	summary historyFactSummary,
) error {
	for _, eventType := range requiredTerminalEvents {
		if summary.eventCounts[eventType] > 1 {
			return fmt.Errorf(
				"faulted history has %d %s events",
				summary.eventCounts[eventType],
				eventType,
			)
		}
	}
	if !summary.allEventTypesUsable {
		return nil
	}
	if summary.lastEventType != "temporal.history.WorkflowExecutionCanceled" {
		return errors.New("closed faulted history is not terminal caller cancellation")
	}
	for _, eventType := range requiredTerminalEvents {
		if summary.eventCounts[eventType] != 1 {
			return fmt.Errorf(
				"faulted history has %d %s events",
				summary.eventCounts[eventType],
				eventType,
			)
		}
	}
	return nil
}

func validateCleanupClosure(
	run artifactv2.ExperimentRun,
	rawEvidence artifactv2.RawEvidence,
) error {
	cleanupStatus := ""
	for _, source := range rawEvidence.Sources {
		if source.SourceDefinitionID == umpireruntime.EvidenceSourceCleanup {
			cleanupStatus = source.Status
		}
	}
	if cleanupStatus != "closed" {
		return nil
	}
	openHandles := uint64(0)
	openHandleFields := 0
	for _, fact := range rawEvidence.Facts {
		if fact.SourceDefinitionID != umpireruntime.EvidenceSourceCleanup {
			continue
		}
		for _, field := range fact.Fields {
			if field.FieldDefinitionID != umpireruntime.EvidenceFieldOpenHandleCount {
				continue
			}
			value, _ := field.Value.(json.Number)
			parsed, err := strconv.ParseUint(string(value), 10, 64)
			if err != nil {
				return errors.New("cleanup open-handle count is malformed")
			}
			openHandles = parsed
			openHandleFields++
		}
	}
	if openHandleFields != 1 || openHandles != 0 || !run.Cleanup.OpenHandleCount.IsZero() {
		return errCleanupLeakage
	}
	return nil
}

func rawNaturalField(fact artifactv2.RawEvidenceFact, definitionID string) (uint64, error) {
	for _, field := range fact.Fields {
		if field.FieldDefinitionID != definitionID {
			continue
		}
		value, ok := field.Value.(json.Number)
		if !ok {
			return 0, fmt.Errorf("field %q is not a natural", definitionID)
		}
		return strconv.ParseUint(string(value), 10, 64)
	}
	return 0, fmt.Errorf("field %q is missing", definitionID)
}

func rawStringField(fact artifactv2.RawEvidenceFact, definitionID string) (string, error) {
	for _, field := range fact.Fields {
		if field.FieldDefinitionID != definitionID {
			continue
		}
		value, ok := field.Value.(string)
		if field.Disposition != "plain" || !ok || value == "" {
			return "", fmt.Errorf("field %q is not one string", definitionID)
		}
		return value, nil
	}
	return "", fmt.Errorf("field %q is missing", definitionID)
}
