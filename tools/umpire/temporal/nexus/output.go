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

type cancellationLifecycle struct {
	requested artifactv2.RawEvidenceFact
	completed artifactv2.RawEvidenceFact
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
	if err := validateMechanicalEvidence(run, rawEvidence); err != nil {
		return err
	}
	if configuration.ConfigurationDefinitionID == duplicateDeliveryConfigurationDefinitionID {
		if err := validateDuplicateDeliveryEvidence(executable, run, rawEvidence); err != nil {
			return err
		}
	} else if err := rejectDuplicateDeliveryEvidence(rawEvidence); err != nil {
		return err
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

func validateDuplicateDeliveryEvidence(
	executable artifact.ExecutableSet,
	run artifactv2.ExperimentRun,
	rawEvidence artifactv2.RawEvidence,
) error {
	if err := validateDuplicateDeliveryRequest(executable.Experiment(), run); err != nil {
		return err
	}
	syntheticFacts := rawEvidenceFactsByKind(rawEvidence, duplicateObservationFactKind)
	if run.OperationalStatus != "succeeded" && len(syntheticFacts) == 0 {
		return nil
	}
	if run.OperationalStatus == "succeeded" && !acceptedControlAttempt(run.ControlAttempts[0]) {
		return errors.New("successful faulted execution has no accepted control receipt")
	}
	synthetic, err := exactDuplicateDeliverySyntheticFact(syntheticFacts, run.RunIdentity)
	if err != nil {
		return err
	}
	callback, err := exactMechanicalCallbackFact(rawEvidence, synthetic)
	if err != nil {
		return err
	}
	lifecycle, err := exactCancellationLifecycle(rawEvidence)
	if err != nil {
		return err
	}
	return validateFaultRealizationCorrelations(
		executable, run.RunIdentity, synthetic, callback, lifecycle.requested, lifecycle.completed,
	)
}

func validateDuplicateDeliveryRequest(
	experiment artifactv2.Experiment,
	run artifactv2.ExperimentRun,
) error {
	if len(experiment.Plan.RequestedFaults) != 1 ||
		experiment.Plan.RequestedFaults[0].DefinitionID != duplicateDeliveryFaultDefinitionID ||
		experiment.Plan.RequestedFaults[0].Value != forceCloseOccurrenceDefinitionID {
		return errors.New("faulted execution does not bind the exact requested fault")
	}
	if len(run.ControlAttempts) != 1 ||
		run.ControlAttempts[0].OccurrenceDefinitionID != forceCloseOccurrenceDefinitionID ||
		run.ControlAttempts[0].ActionDefinitionID != forceCloseActionDefinitionID ||
		run.ControlAttempts[0].Attempt.String() != "1" {
		return errors.New("faulted execution does not bind the exact control occurrence")
	}
	return nil
}

func acceptedControlAttempt(attempt artifactv2.ControlAttempt) bool {
	return attempt.Status == "accepted" && attempt.ReceiptFactDefinitionID != nil
}

func exactDuplicateDeliverySyntheticFact(
	facts []artifactv2.RawEvidenceFact,
	runIdentity string,
) (artifactv2.RawEvidenceFact, error) {
	if len(facts) != 1 {
		return artifactv2.RawEvidenceFact{}, fmt.Errorf(
			"faulted execution has %d synthetic contribution facts", len(facts))
	}
	synthetic := facts[0]
	if !slices.Equal(rawEvidenceFieldDefinitionIDs(synthetic),
		duplicateDeliverySyntheticFieldOrder[:]) {
		return artifactv2.RawEvidenceFact{},
			errors.New("synthetic contribution does not have the exact closed fields")
	}
	for definitionID, expected := range map[string]string{
		umpireruntime.EvidenceFieldCancellationCompletedCount:  "1",
		umpireruntime.EvidenceFieldCancellationRequestedCount:  "1",
		umpireruntime.EvidenceFieldCapabilityDefinitionID:      "nexus.capability.cancellation",
		umpireruntime.EvidenceFieldCommandKind:                 string(umpireruntime.CommandRealize),
		umpireruntime.EvidenceFieldFaultDefinitionID:           duplicateDeliveryFaultDefinitionID,
		umpireruntime.EvidenceFieldFaultReceiptDefinitionID:    duplicateDeliveryFaultReceiptDefinitionID,
		umpireruntime.EvidenceFieldRunCorrelationID:            runIdentity,
		umpireruntime.EvidenceFieldStatus:                      "accepted",
		umpireruntime.EvidenceFieldSyntheticContributionCount:  "1",
		umpireruntime.EvidenceFieldSyntheticContributionMarker: duplicateObservationFactKind,
	} {
		actual, err := rawFieldString(synthetic, definitionID)
		if err != nil || actual != expected {
			return artifactv2.RawEvidenceFact{}, fmt.Errorf(
				"synthetic contribution field %q is not exact", definitionID)
		}
	}
	return synthetic, nil
}

func exactMechanicalCallbackFact(
	rawEvidence artifactv2.RawEvidence,
	synthetic artifactv2.RawEvidenceFact,
) (artifactv2.RawEvidenceFact, error) {
	callbackFacts := rawEvidenceFactsWithField(
		rawEvidence,
		umpireruntime.EvidenceFieldCancellationCallbackCount,
	)
	if len(callbackFacts) != 1 {
		return artifactv2.RawEvidenceFact{}, fmt.Errorf(
			"faulted execution has %d mechanical callback facts", len(callbackFacts))
	}
	callback := callbackFacts[0]
	callbackCount, err := rawNaturalField(callback, umpireruntime.EvidenceFieldCancellationCallbackCount)
	if err != nil || callbackCount != 1 {
		return artifactv2.RawEvidenceFact{},
			errors.New("mechanical cancellation callback count is not one")
	}
	syntheticOrdinal, syntheticOrdinalErr := rawOrdinal(synthetic)
	callbackOrdinal, callbackOrdinalErr := rawOrdinal(callback)
	if !slices.Equal(synthetic.CausalFactDefinitionIDs, []string{callback.FactDefinitionID}) ||
		synthetic.SourceDefinitionID != callback.SourceDefinitionID ||
		syntheticOrdinalErr != nil || callbackOrdinalErr != nil ||
		syntheticOrdinal <= callbackOrdinal {
		return artifactv2.RawEvidenceFact{},
			errors.New("synthetic contribution does not follow the real callback fact")
	}
	return callback, nil
}

func exactCancellationLifecycle(
	rawEvidence artifactv2.RawEvidence,
) (cancellationLifecycle, error) {
	requested, err := exactHistoryEventFact(
		rawEvidence,
		"temporal.history.NexusOperationCancelRequested",
	)
	if err != nil {
		return cancellationLifecycle{}, err
	}
	completed, err := exactHistoryEventFact(
		rawEvidence,
		"temporal.history.NexusOperationCancelRequestCompleted",
	)
	if err != nil {
		return cancellationLifecycle{}, err
	}
	completedOrdinal, completedOrdinalErr := rawOrdinal(completed)
	requestedOrdinal, requestedOrdinalErr := rawOrdinal(requested)
	if !slices.Equal(completed.CausalFactDefinitionIDs, []string{requested.FactDefinitionID}) ||
		completedOrdinalErr != nil || requestedOrdinalErr != nil ||
		completedOrdinal <= requestedOrdinal {
		return cancellationLifecycle{}, errors.New("completed cancellation does not follow its request")
	}
	return cancellationLifecycle{requested: requested, completed: completed}, nil
}

func validateFaultRealizationCorrelations(
	executable artifact.ExecutableSet,
	runIdentity string,
	facts ...artifactv2.RawEvidenceFact,
) error {
	request, err := CheckRequest(executable.AdmittedSet(), runIdentity)
	if err != nil {
		return fmt.Errorf("recheck faulted execution request: %w", err)
	}
	correlations := requestCorrelations(request)
	for definitionID, expected := range map[string]string{
		umpireruntime.EvidenceFieldOperationCorrelationID: correlations.operation,
		umpireruntime.EvidenceFieldRunCorrelationID:       runIdentity,
		umpireruntime.EvidenceFieldWorkflowCorrelationID:  correlations.workflow,
	} {
		if expected == "" {
			return fmt.Errorf("synthetic contribution correlation %q is unavailable", definitionID)
		}
		for _, fact := range facts {
			actual, fieldErr := rawStringField(fact, definitionID)
			if fieldErr != nil || actual != expected {
				return fmt.Errorf("fault realization correlation %q does not close", definitionID)
			}
		}
	}
	return nil
}

func rejectDuplicateDeliveryEvidence(rawEvidence artifactv2.RawEvidence) error {
	if len(rawEvidenceFactsByKind(rawEvidence, duplicateObservationFactKind)) != 0 {
		return errors.New("normal execution contains a synthetic duplicate contribution")
	}
	for _, fact := range rawEvidence.Facts {
		for _, field := range fact.Fields {
			if _, faultOnly := duplicateDeliveryOnlyFields[field.FieldDefinitionID]; faultOnly {
				return fmt.Errorf("normal execution contains fault-only field %q", field.FieldDefinitionID)
			}
		}
	}
	return nil
}

func rawEvidenceFactsByKind(
	rawEvidence artifactv2.RawEvidence,
	kindDefinitionID string,
) []artifactv2.RawEvidenceFact {
	facts := make([]artifactv2.RawEvidenceFact, 0, 1)
	for _, fact := range rawEvidence.Facts {
		if fact.KindDefinitionID == kindDefinitionID {
			facts = append(facts, fact)
		}
	}
	return facts
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

func rawFieldString(fact artifactv2.RawEvidenceFact, definitionID string) (string, error) {
	if value, err := rawStringField(fact, definitionID); err == nil {
		return value, nil
	}
	value, err := rawNaturalField(fact, definitionID)
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(value, 10), nil
}

func rawOrdinal(fact artifactv2.RawEvidenceFact) (uint64, error) {
	return strconv.ParseUint(fact.Ordinal.String(), 10, 64)
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
		if err := validateRetainedFields(fact); err != nil {
			return err
		}
	}
	if err := validateHistoryClosure(rawEvidence); err != nil {
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

func validateRetainedFields(fact artifactv2.RawEvidenceFact) error {
	for _, field := range fact.Fields {
		kind, allowed := retainedFieldDisposition[field.FieldDefinitionID]
		if !allowed {
			return fmt.Errorf("evidence field %q is not allowlisted", field.FieldDefinitionID)
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

func validateHistoryClosure(rawEvidence artifactv2.RawEvidence) error {
	historyStatus := ""
	for _, source := range rawEvidence.Sources {
		if source.SourceDefinitionID == umpireruntime.EvidenceSourceHistory {
			historyStatus = source.Status
		}
	}
	historyFacts := make([]artifactv2.RawEvidenceFact, 0)
	for _, fact := range rawEvidence.Facts {
		if fact.SourceDefinitionID == umpireruntime.EvidenceSourceHistory {
			historyFacts = append(historyFacts, fact)
		}
	}
	previousFact := ""
	previousEventID := uint64(0)
	eventCounts := make(map[string]uint64)
	lastEventType := ""
	for index, fact := range historyFacts {
		if fact.KindDefinitionID != "umpire.evidence.kind.workflow-history-event" {
			return errors.New("history fact has the wrong mechanical kind")
		}
		if index == 0 {
			if len(fact.CausalFactDefinitionIDs) != 0 {
				return errors.New("first history fact has a causal predecessor")
			}
		} else if !slices.Equal(fact.CausalFactDefinitionIDs, []string{previousFact}) {
			return errors.New("history facts do not form one gapless causal chain")
		}
		eventID, err := rawNaturalField(fact, umpireruntime.EvidenceFieldEventID)
		if err != nil || eventID <= previousEventID {
			return errors.New("history event IDs are not strictly increasing")
		}
		eventType, err := rawStringField(fact, umpireruntime.EvidenceFieldEventType)
		if err != nil {
			return err
		}
		previousFact = fact.FactDefinitionID
		previousEventID = eventID
		lastEventType = eventType
		eventCounts[eventType]++
	}
	if historyStatus != "closed" {
		return nil
	}
	if lastEventType != "temporal.history.WorkflowExecutionCanceled" {
		return errors.New("closed history is not terminal caller cancellation")
	}
	for _, eventType := range requiredTerminalEvents {
		if eventCounts[eventType] != 1 {
			return fmt.Errorf("closed history has %d %s events", eventCounts[eventType], eventType)
		}
	}
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
		return errors.New("closed cleanup does not prove zero open handles")
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
		if !ok || value == "" {
			return "", fmt.Errorf("field %q is not one string", definitionID)
		}
		return value, nil
	}
	return "", fmt.Errorf("field %q is missing", definitionID)
}
