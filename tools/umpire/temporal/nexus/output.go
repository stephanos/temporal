package nexus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"

	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
	"go.temporal.io/server/tools/umpire/temporal/local"
)

var retainedFieldDisposition = map[string]string{
	umpireruntime.EvidenceFieldCancellationCallbackCount: "number",
	umpireruntime.EvidenceFieldCommandKind:               "string",
	umpireruntime.EvidenceFieldEndpointIdentity:          "sha256",
	umpireruntime.EvidenceFieldErrorCode:                 "string",
	umpireruntime.EvidenceFieldEventID:                   "number",
	umpireruntime.EvidenceFieldEventType:                 "string",
	umpireruntime.EvidenceFieldNamespaceIdentity:         "sha256",
	umpireruntime.EvidenceFieldOpenHandleCount:           "number",
	umpireruntime.EvidenceFieldOperationCorrelationID:    "string",
	umpireruntime.EvidenceFieldRunCorrelationID:          "string",
	umpireruntime.EvidenceFieldStatus:                    "string",
	umpireruntime.EvidenceFieldTaskQueueIdentity:         "sha256",
	umpireruntime.EvidenceFieldWorkflowCorrelationID:     "string",
	artifactv2.ControlReceiptActionFieldDefinitionID:     "string",
	artifactv2.ControlReceiptAttemptFieldDefinitionID:    "number",
	artifactv2.ControlReceiptOccurrenceFieldDefinitionID: "string",
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

// Run executes the exact checked caller-closure request and returns one admitted
// in-memory four-member output. It performs no publication or interpretation.
func Run(
	ctx context.Context,
	request umpireruntime.CheckedRunRequest,
) (umpireruntime.Output, error) {
	participant, err := NewParticipant(request)
	if err != nil {
		return umpireruntime.Output{}, err
	}
	output, err := umpireruntime.Run(ctx, request, local.NewFactory(), participant)
	if err != nil {
		return umpireruntime.Output{}, err
	}
	executable, ok := request.AdmittedSet().Executable()
	if !ok || validateExecutionClosure(
		executable,
		output.AdmittedSet(),
		output.ExperimentRun(),
		output.RawEvidence(),
	) != nil {
		return umpireruntime.Output{}, errors.New("umpire.temporal.nexus.invariant.execution-closure")
	}
	return output, nil
}

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
