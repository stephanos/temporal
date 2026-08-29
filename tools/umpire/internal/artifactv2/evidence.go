package artifactv2

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

const RawEvidenceFormat = "umpire-raw-evidence/v2"

const rawEvidenceChecksumDomain = "umpire.raw-evidence/v2"

const (
	ControlReceiptSourceDefinitionID          = "umpire.evidence.source.control-receipt"
	ControlReceiptKindDefinitionID            = "umpire.evidence.kind.control-receipt"
	ControlReceiptActionFieldDefinitionID     = "umpire.evidence.field.action-definition-id"
	ControlReceiptAttemptFieldDefinitionID    = "umpire.evidence.field.attempt"
	ControlReceiptOccurrenceFieldDefinitionID = "umpire.evidence.field.occurrence-definition-id"
	ControlReceiptStatusFieldDefinitionID     = "umpire.evidence.field.status"
)

type RawEvidenceSource struct {
	SourceDefinitionID string  `json:"sourceDefinitionId"`
	Status             string  `json:"status"`
	FactCount          Natural `json:"factCount"`
	ByteCount          Natural `json:"byteCount"`
}

type RawEvidenceField struct {
	FieldDefinitionID string `json:"fieldDefinitionId"`
	Disposition       string `json:"disposition"`
	Value             any    `json:"value"`
}

type RawEvidenceFact struct {
	FactDefinitionID        string             `json:"factDefinitionId"`
	SourceDefinitionID      string             `json:"sourceDefinitionId"`
	Ordinal                 Natural            `json:"ordinal"`
	KindDefinitionID        string             `json:"kindDefinitionId"`
	CausalFactDefinitionIDs []string           `json:"causalFactDefinitionIds"`
	Fields                  []RawEvidenceField `json:"fields"`
}

type RawEvidence struct {
	FormatVersion        string              `json:"formatVersion"`
	RunIdentity          string              `json:"runIdentity"`
	BehaviorFingerprint  string              `json:"behaviorFingerprint"`
	Experiment           ArtifactBinding     `json:"experiment"`
	RuntimeConfiguration ArtifactBinding     `json:"runtimeConfiguration"`
	Run                  ArtifactBinding     `json:"run"`
	CaptureStatus        string              `json:"captureStatus"`
	Sources              []RawEvidenceSource `json:"sources"`
	Facts                []RawEvidenceFact   `json:"facts"`
	KnownGaps            []KnownGap          `json:"knownGaps"`
	Provenance           Provenance          `json:"provenance"`
	ProvenanceChecksum   string              `json:"provenanceChecksum"`
	ArtifactChecksum     string              `json:"artifactChecksum,omitempty"`
}

func CanonicalRawEvidenceBytes(document RawEvidence) ([]byte, error) {
	return encodeJSONLine(document)
}

func ExpectedRawEvidenceChecksum(document RawEvidence) (string, error) {
	document.ArtifactChecksum = ""
	encoded, err := encodeJSONLine(document)
	if err != nil {
		return "", err
	}
	return derive(rawEvidenceChecksumDomain, encoded), nil
}

func SealRawEvidence(document RawEvidence) (RawEvidence, error) {
	provenanceChecksum, err := ExpectedProvenanceChecksum(document.Provenance)
	if err != nil {
		return RawEvidence{}, err
	}
	document.ProvenanceChecksum = provenanceChecksum
	artifactChecksum, err := ExpectedRawEvidenceChecksum(document)
	if err != nil {
		return RawEvidence{}, err
	}
	document.ArtifactChecksum = artifactChecksum
	return document, nil
}

func VerifyRawEvidenceProvenanceChecksum(document RawEvidence) error {
	expected, err := ExpectedProvenanceChecksum(document.Provenance)
	if err != nil {
		return err
	}
	if document.ProvenanceChecksum != expected {
		return fmt.Errorf("RawEvidence provenance checksum mismatch: got %q, want %q",
			document.ProvenanceChecksum, expected)
	}
	return nil
}

func VerifyRawEvidenceArtifactChecksum(document RawEvidence) error {
	expected, err := ExpectedRawEvidenceChecksum(document)
	if err != nil {
		return err
	}
	if document.ArtifactChecksum != expected {
		return fmt.Errorf("RawEvidence artifact checksum mismatch: got %q, want %q",
			document.ArtifactChecksum, expected)
	}
	return nil
}

func ValidateRawEvidence(document RawEvidence) error {
	if document.FormatVersion != RawEvidenceFormat {
		return fmt.Errorf("unsupported format %q", document.FormatVersion)
	}
	if !validDefinitionID(document.RunIdentity) || !ValidDigest(document.BehaviorFingerprint) ||
		!ValidDigest(document.ProvenanceChecksum) || !ValidDigest(document.ArtifactChecksum) {
		return errors.New("RawEvidence identity, checksum, or behavior fingerprint is malformed")
	}
	if err := validateArtifactBinding("experiment", document.Experiment, ExperimentFormat); err != nil {
		return err
	}
	if err := validateArtifactBinding("runtime configuration", document.RuntimeConfiguration,
		RuntimeConfigurationFormat); err != nil {
		return err
	}
	if err := validateArtifactBinding("Run", document.Run, ExperimentRunFormat); err != nil {
		return err
	}
	if err := validateRawEvidenceSources(document.Sources); err != nil {
		return err
	}
	if err := validateRawEvidenceFacts(document.Sources, document.Facts); err != nil {
		return err
	}
	expectedStatus := expectedCaptureStatus(document.Sources)
	if document.CaptureStatus != expectedStatus {
		return fmt.Errorf("capture status %q is inconsistent with sources; expected %q",
			document.CaptureStatus, expectedStatus)
	}
	if document.KnownGaps == nil {
		return errors.New("RawEvidence known gaps must not be null")
	}
	if err := validateKnownGaps(document.KnownGaps); err != nil {
		return err
	}
	return validateProvenance(document.Provenance)
}

func validateRawEvidenceSources(sources []RawEvidenceSource) error {
	if len(sources) == 0 {
		return errors.New("at least one RawEvidence source is required")
	}
	if !slices.IsSortedFunc(sources, func(left, right RawEvidenceSource) int {
		return strings.Compare(left.SourceDefinitionID, right.SourceDefinitionID)
	}) {
		return errors.New("RawEvidence sources are not in canonical order")
	}
	for index, source := range sources {
		if index > 0 && source.SourceDefinitionID == sources[index-1].SourceDefinitionID {
			return fmt.Errorf("duplicate RawEvidence source %q", source.SourceDefinitionID)
		}
		if !validDefinitionID(source.SourceDefinitionID) ||
			validateNaturalBytes([]byte(source.FactCount)) != nil ||
			validateNaturalBytes([]byte(source.ByteCount)) != nil {
			return fmt.Errorf("RawEvidence source %q is malformed", source.SourceDefinitionID)
		}
		switch source.Status {
		case "closed", "partial", "failed":
		default:
			return fmt.Errorf("RawEvidence source status %q is invalid", source.Status)
		}
	}
	return nil
}

func expectedCaptureStatus(sources []RawEvidenceSource) string {
	for _, source := range sources {
		if source.Status == "failed" {
			return "failed"
		}
	}
	for _, source := range sources {
		if source.Status == "partial" {
			return "partial"
		}
	}
	return "closed"
}

func validateRawEvidenceFacts(sources []RawEvidenceSource, facts []RawEvidenceFact) error {
	if facts == nil {
		return errors.New("RawEvidence facts must not be null")
	}
	if !slices.IsSortedFunc(facts, compareRawEvidenceFact) {
		return errors.New("RawEvidence facts are not in canonical order")
	}
	sourceByID := make(map[string]RawEvidenceSource, len(sources))
	expectedOrdinal := make(map[string]uint64, len(sources))
	for _, source := range sources {
		sourceByID[source.SourceDefinitionID] = source
	}
	seenFacts := make(map[string]struct{}, len(facts))
	for _, fact := range facts {
		if !validDefinitionID(fact.FactDefinitionID) || !validDefinitionID(fact.SourceDefinitionID) ||
			!validDefinitionID(fact.KindDefinitionID) || validateNaturalBytes([]byte(fact.Ordinal)) != nil {
			return fmt.Errorf("RawEvidence fact %q is malformed", fact.FactDefinitionID)
		}
		if _, duplicate := seenFacts[fact.FactDefinitionID]; duplicate {
			return fmt.Errorf("duplicate RawEvidence fact %q", fact.FactDefinitionID)
		}
		if _, ok := sourceByID[fact.SourceDefinitionID]; !ok {
			return fmt.Errorf("RawEvidence fact %q has unknown source %q",
				fact.FactDefinitionID, fact.SourceDefinitionID)
		}
		expected := NaturalFromUint64(expectedOrdinal[fact.SourceDefinitionID])
		if fact.Ordinal != expected {
			return fmt.Errorf("RawEvidence fact %q has ordinal %s; expected %s",
				fact.FactDefinitionID, fact.Ordinal, expected)
		}
		if err := validateCausalFactDefinitionIDs(fact, seenFacts); err != nil {
			return err
		}
		if err := validateRawEvidenceFields(fact); err != nil {
			return err
		}
		seenFacts[fact.FactDefinitionID] = struct{}{}
		expectedOrdinal[fact.SourceDefinitionID]++
	}
	for _, source := range sources {
		actual := NaturalFromUint64(expectedOrdinal[source.SourceDefinitionID])
		if source.FactCount != actual {
			return fmt.Errorf("RawEvidence source %q declares %s facts; found %s",
				source.SourceDefinitionID, source.FactCount, actual)
		}
	}
	return nil
}

func compareRawEvidenceFact(left, right RawEvidenceFact) int {
	if comparison := strings.Compare(left.SourceDefinitionID, right.SourceDefinitionID); comparison != 0 {
		return comparison
	}
	if comparison := compareNatural(left.Ordinal, right.Ordinal); comparison != 0 {
		return comparison
	}
	return strings.Compare(left.FactDefinitionID, right.FactDefinitionID)
}

func validateCausalFactDefinitionIDs(fact RawEvidenceFact, seenFacts map[string]struct{}) error {
	if fact.CausalFactDefinitionIDs == nil {
		return fmt.Errorf("RawEvidence fact %q causal fact definition IDs must not be null",
			fact.FactDefinitionID)
	}
	if !slices.IsSorted(fact.CausalFactDefinitionIDs) {
		return fmt.Errorf("RawEvidence fact %q causal fact definition IDs are not in canonical order",
			fact.FactDefinitionID)
	}
	for index, causalID := range fact.CausalFactDefinitionIDs {
		if index > 0 && causalID == fact.CausalFactDefinitionIDs[index-1] {
			return fmt.Errorf("RawEvidence fact %q repeats causal fact %q", fact.FactDefinitionID, causalID)
		}
		if !validDefinitionID(causalID) {
			return fmt.Errorf("RawEvidence fact %q has malformed causal fact %q",
				fact.FactDefinitionID, causalID)
		}
		if _, precedes := seenFacts[causalID]; !precedes {
			return fmt.Errorf("RawEvidence fact %q has forward or dangling causal fact %q",
				fact.FactDefinitionID, causalID)
		}
	}
	return nil
}

func validateRawEvidenceFields(fact RawEvidenceFact) error {
	if fact.Fields == nil {
		return fmt.Errorf("RawEvidence fact %q fields must not be null", fact.FactDefinitionID)
	}
	if !slices.IsSortedFunc(fact.Fields, func(left, right RawEvidenceField) int {
		return strings.Compare(left.FieldDefinitionID, right.FieldDefinitionID)
	}) {
		return fmt.Errorf("RawEvidence fact %q fields are not in canonical order", fact.FactDefinitionID)
	}
	for index, field := range fact.Fields {
		if index > 0 && field.FieldDefinitionID == fact.Fields[index-1].FieldDefinitionID {
			return fmt.Errorf("RawEvidence fact %q repeats field %q",
				fact.FactDefinitionID, field.FieldDefinitionID)
		}
		if !validDefinitionID(field.FieldDefinitionID) {
			return fmt.Errorf("RawEvidence field definition ID %q is invalid", field.FieldDefinitionID)
		}
		if err := validateRawEvidenceFieldValue(field); err != nil {
			return fmt.Errorf("RawEvidence field %q: %w", field.FieldDefinitionID, err)
		}
	}
	return nil
}

func validateRawEvidenceFieldValue(field RawEvidenceField) error {
	switch field.Disposition {
	case "plain":
		switch value := field.Value.(type) {
		case nil, bool, string:
			return nil
		case json.Number:
			if !validCanonicalInteger(string(value)) {
				return errors.New("plain number is not a canonical integer")
			}
			return nil
		default:
			return errors.New("plain value is not null, Boolean, canonical integer, or string")
		}
	case "redacted", "rejected":
		if field.Value != nil {
			return fmt.Errorf("%s value must be null", field.Disposition)
		}
		return nil
	case "sha256":
		value, ok := field.Value.(string)
		if !ok || !ValidDigest(value) {
			return errors.New("sha256 value must use checksum spelling")
		}
		return nil
	default:
		return fmt.Errorf("disposition %q is invalid", field.Disposition)
	}
}

func validCanonicalInteger(value string) bool {
	if value == "0" {
		return true
	}
	value = strings.TrimPrefix(value, "-")
	if value == "" || value[0] == '0' {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func ExperimentRunArtifactBinding(document ExperimentRun) ArtifactBinding {
	return ArtifactBinding{
		FormatVersion:       document.FormatVersion,
		ArtifactChecksum:    document.ArtifactChecksum,
		BehaviorFingerprint: document.BehaviorFingerprint,
		ProvenanceChecksum:  document.ProvenanceChecksum,
	}
}

func ValidateRawEvidenceClosure(
	document RawEvidence,
	experiment Experiment,
	runtimeConfiguration RuntimeConfiguration,
	run ExperimentRun,
) error {
	if err := ValidateExperimentRunClosure(run, experiment, runtimeConfiguration); err != nil {
		return err
	}
	experimentBinding, err := ExperimentArtifactBinding(experiment)
	if err != nil {
		return err
	}
	if document.Experiment != experimentBinding {
		return errors.New("RawEvidence experiment binding does not match ExperimentSpec")
	}
	if document.RuntimeConfiguration != RuntimeConfigurationArtifactBinding(runtimeConfiguration) {
		return errors.New("RawEvidence runtime configuration binding does not match RuntimeConfiguration")
	}
	if document.Run != ExperimentRunArtifactBinding(run) {
		return errors.New("RawEvidence Run binding does not match ExperimentRun")
	}
	if document.RunIdentity != run.RunIdentity {
		return errors.New("RawEvidence run identity does not match ExperimentRun")
	}
	if err := validateRawEvidenceSourceClosure(document.Sources, run.SourceClosures); err != nil {
		return err
	}
	return ValidateRawEvidenceRunReceipts(document, run)
}

func validateRawEvidenceSourceClosure(sources []RawEvidenceSource, closures []SourceClosure) error {
	if len(sources) != len(closures) {
		return errors.New("RawEvidence sources do not match ExperimentRun source closures")
	}
	for index, source := range sources {
		closure := closures[index]
		if source.SourceDefinitionID != closure.SourceDefinitionID || source.Status != closure.Status ||
			source.FactCount != closure.RecordCount || source.ByteCount != closure.ByteCount {
			return fmt.Errorf("RawEvidence source %q does not match its ExperimentRun closure",
				source.SourceDefinitionID)
		}
	}
	return nil
}

func ValidateRawEvidenceRunReceipts(document RawEvidence, run ExperimentRun) error {
	factsByID := make(map[string][]RawEvidenceFact, len(document.Facts))
	for _, fact := range document.Facts {
		factsByID[fact.FactDefinitionID] = append(factsByID[fact.FactDefinitionID], fact)
	}
	referenced := make(map[string]struct{}, len(run.ControlAttempts))
	for _, attempt := range run.ControlAttempts {
		if attempt.Status == "not-attempted" {
			continue
		}
		if attempt.ReceiptFactDefinitionID == nil {
			return fmt.Errorf("attempted control %q has no receipt fact", attempt.OccurrenceDefinitionID)
		}
		receiptID := *attempt.ReceiptFactDefinitionID
		facts := factsByID[receiptID]
		if len(facts) != 1 {
			return fmt.Errorf("control receipt %q resolves to %d RawEvidence facts", receiptID, len(facts))
		}
		if _, duplicate := referenced[receiptID]; duplicate {
			return fmt.Errorf("control receipt %q is referenced more than once", receiptID)
		}
		referenced[receiptID] = struct{}{}
		if err := validateControlReceiptFact(facts[0], attempt); err != nil {
			return err
		}
	}
	for _, fact := range document.Facts {
		isReceiptSource := fact.SourceDefinitionID == ControlReceiptSourceDefinitionID
		isReceiptKind := fact.KindDefinitionID == ControlReceiptKindDefinitionID
		if !isReceiptSource && !isReceiptKind {
			continue
		}
		if !isReceiptSource || !isReceiptKind {
			return fmt.Errorf("RawEvidence fact %q has a crossed control-receipt source or kind",
				fact.FactDefinitionID)
		}
		if _, ok := referenced[fact.FactDefinitionID]; !ok {
			return fmt.Errorf("RawEvidence control-receipt fact %q is not bound to one attempted control",
				fact.FactDefinitionID)
		}
	}
	return nil
}

func validateControlReceiptFact(fact RawEvidenceFact, attempt ControlAttempt) error {
	if fact.SourceDefinitionID != ControlReceiptSourceDefinitionID ||
		fact.KindDefinitionID != ControlReceiptKindDefinitionID {
		return fmt.Errorf("control receipt %q has the wrong source or kind", fact.FactDefinitionID)
	}
	if len(fact.Fields) != 4 {
		return fmt.Errorf("control receipt %q must contain exactly four binding fields",
			fact.FactDefinitionID)
	}
	values := make(map[string]any, len(fact.Fields))
	for _, field := range fact.Fields {
		if _, duplicate := values[field.FieldDefinitionID]; duplicate {
			return fmt.Errorf("control receipt %q repeats field %q",
				fact.FactDefinitionID, field.FieldDefinitionID)
		}
		if field.Disposition != "plain" {
			return fmt.Errorf("control receipt %q field %q is not plain",
				fact.FactDefinitionID, field.FieldDefinitionID)
		}
		values[field.FieldDefinitionID] = field.Value
	}
	expected := map[string]any{
		ControlReceiptActionFieldDefinitionID:     attempt.ActionDefinitionID,
		ControlReceiptAttemptFieldDefinitionID:    json.Number(attempt.Attempt.String()),
		ControlReceiptOccurrenceFieldDefinitionID: attempt.OccurrenceDefinitionID,
		ControlReceiptStatusFieldDefinitionID:     attempt.Status,
	}
	for fieldDefinitionID, expectedValue := range expected {
		value, ok := values[fieldDefinitionID]
		if !ok || value != expectedValue {
			return fmt.Errorf("control receipt %q field %q does not match ExperimentRun",
				fact.FactDefinitionID, fieldDefinitionID)
		}
	}
	return nil
}
