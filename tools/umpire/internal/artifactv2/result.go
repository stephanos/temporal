package artifactv2

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

const (
	EvidenceFormat = "umpire-evidence/v2"
	ResultFormat   = "umpire-result/v2"
)

const (
	evidenceChecksumDomain          = "umpire.evidence/v2"
	resultChecksumDomain            = "umpire.result/v2"
	evaluationOutcomeChecksumDomain = "umpire.evaluation-outcome/v2"
)

type DefinitionReference struct {
	DefinitionID        string `json:"definitionId"`
	BehaviorFingerprint string `json:"behaviorFingerprint"`
}

type ModelCoordinate struct {
	Kind     string   `json:"kind"`
	Step     *Natural `json:"step"`
	Position *Natural `json:"position"`
}

type ModelTraceStep struct {
	Position       Natural      `json:"position"`
	SelectedAction ModelValue   `json:"selectedAction"`
	ModelOutcome   ModelValue   `json:"modelOutcome"`
	ResultingState ModelValue   `json:"resultingState"`
	Observations   []ModelValue `json:"observations"`
}

type ModelTrace struct {
	TraceID      string           `json:"traceId"`
	InitialState ModelValue       `json:"initialState"`
	Steps        []ModelTraceStep `json:"steps"`
}

type MeaningProvision struct {
	DefinitionID      string `json:"definitionId"`
	Kind              string `json:"kind"`
	CanonicalBehavior string `json:"canonicalBehavior"`
}

type FieldReference struct {
	KindDefinitionID  string `json:"kindDefinitionId"`
	FieldDefinitionID string `json:"fieldDefinitionId"`
}

type FieldDispositionRecord struct {
	Field                    FieldReference `json:"field"`
	Disposition              string         `json:"disposition"`
	DigestPolicyDefinitionID *string        `json:"digestPolicyDefinitionId"`
}

type EvidenceOrderingFact struct {
	FactDefinitionID        string   `json:"factDefinitionId"`
	KindDefinitionID        string   `json:"kindDefinitionId"`
	Ordinal                 Natural  `json:"ordinal"`
	CausalFactDefinitionIDs []string `json:"causalFactDefinitionIds"`
}

type EvidenceClosureFact struct {
	KindDefinitionID string  `json:"kindDefinitionId"`
	LastOrdinal      Natural `json:"lastOrdinal"`
}

type AppliedFieldDisposition struct {
	Field                    FieldReference `json:"field"`
	Kind                     string         `json:"kind"`
	NormalizedValue          *string        `json:"normalizedValue"`
	DigestPolicyDefinitionID *string        `json:"digestPolicyDefinitionId"`
	DigestToken              *string        `json:"digestToken"`
}

type EvidenceLink struct {
	Coordinate                 ModelCoordinate           `json:"coordinate"`
	MappingDefinitionID        string                    `json:"mappingDefinitionId"`
	MappingVersion             Natural                   `json:"mappingVersion"`
	MappingBehaviorFingerprint string                    `json:"mappingBehaviorFingerprint"`
	ProfileDefinitionID        string                    `json:"profileDefinitionId"`
	ProfileVersion             Natural                   `json:"profileVersion"`
	EvidenceDefinitionIDs      []string                  `json:"evidenceDefinitionIds"`
	RuleDefinitionID           string                    `json:"ruleDefinitionId"`
	BindingDefinitionIDs       []string                  `json:"bindingDefinitionIds"`
	OrderingSupport            []EvidenceOrderingFact    `json:"orderingSupport"`
	ClosureSupport             []EvidenceClosureFact     `json:"closureSupport"`
	AppliedDispositions        []AppliedFieldDisposition `json:"appliedDispositions"`
	AppliedLimit               Limit                     `json:"appliedLimit"`
	MeaningBehaviorFingerprint string                    `json:"meaningBehaviorFingerprint"`
}

type EvidenceBackedModelTrace struct {
	TraceID                    string              `json:"traceId"`
	ObservationPlan            DefinitionReference `json:"observationPlan"`
	MappingDefinitionID        string              `json:"mappingDefinitionId"`
	MappingVersion             Natural             `json:"mappingVersion"`
	MappingBehaviorFingerprint string              `json:"mappingBehaviorFingerprint"`
	Source                     SourceLocation      `json:"source"`
	ProfileDefinitionID        string              `json:"profileDefinitionId"`
	ProfileVersion             Natural             `json:"profileVersion"`
	SourceClosed               bool                `json:"sourceClosed"`
	Vocabulary                 []MeaningProvision  `json:"vocabulary"`
	AppliedLimit               Limit               `json:"appliedLimit"`
	EvidenceDefinitionIDs      []string            `json:"evidenceDefinitionIds"`
	Trace                      ModelTrace          `json:"trace"`
}

type ObservationDiagnostic struct {
	Kind                             string   `json:"kind"`
	ObservationPlanDefinitionID      string   `json:"observationPlanDefinitionId"`
	RelatedDefinitionIDs             []string `json:"relatedDefinitionIds"`
	AppliedLimit                     *Limit   `json:"appliedLimit"`
	ObservedCount                    *Natural `json:"observedCount"`
	Alternatives                     []string `json:"alternatives"`
	MissingDiscriminatorDefinitionID *string  `json:"missingDiscriminatorDefinitionId"`
}

type Evidence struct {
	FormatVersion               string                    `json:"formatVersion"`
	RunIdentity                 string                    `json:"runIdentity"`
	BehaviorFingerprint         string                    `json:"behaviorFingerprint"`
	Experiment                  ArtifactBinding           `json:"experiment"`
	RuntimeConfiguration        ArtifactBinding           `json:"runtimeConfiguration"`
	Run                         ArtifactBinding           `json:"run"`
	RawEvidence                 ArtifactBinding           `json:"rawEvidence"`
	ObservationProgram          DefinitionReference       `json:"observationProgram"`
	Mapping                     DefinitionReference       `json:"mapping"`
	ObservationEvaluationStatus string                    `json:"observationEvaluationStatus"`
	EvidenceBackedModelTrace    *EvidenceBackedModelTrace `json:"evidenceBackedModelTrace"`
	EvidenceLinks               []EvidenceLink            `json:"evidenceLinks"`
	Dispositions                []FieldDispositionRecord  `json:"dispositions"`
	Diagnostics                 []ObservationDiagnostic   `json:"diagnostics"`
	KnownGaps                   []KnownGap                `json:"knownGaps"`
	Provenance                  Provenance                `json:"provenance"`
	ProvenanceChecksum          string                    `json:"provenanceChecksum"`
	ArtifactChecksum            string                    `json:"artifactChecksum,omitempty"`
}

type ImplementationTargetReference struct {
	DefinitionID        string `json:"definitionId"`
	Kind                string `json:"kind"`
	BehaviorFingerprint string `json:"behaviorFingerprint"`
}

type ImplementationLinkDiagnostic struct {
	Kind                            string           `json:"kind"`
	Coordinate                      *ModelCoordinate `json:"coordinate"`
	RelatedDefinitionIDs            []string         `json:"relatedDefinitionIds"`
	SourceSetupBehaviorFingerprint  *string          `json:"sourceSetupBehaviorFingerprint"`
	AppliedLimit                    *Limit           `json:"appliedLimit"`
	ObservedCount                   *Natural         `json:"observedCount"`
	KnownGapCode                    *string          `json:"knownGapCode"`
	KnownGapReason                  *string          `json:"knownGapReason"`
	UnsupportedVocabularyKind       *string          `json:"unsupportedVocabularyKind"`
	EvidenceLinkBehaviorFingerprint *string          `json:"evidenceLinkBehaviorFingerprint"`
	Identity                        string           `json:"identity"`
}

type ImplementationLinkRecord struct {
	DefinitionID        string                        `json:"definitionId"`
	BehaviorFingerprint string                        `json:"behaviorFingerprint"`
	SourceTarget        ImplementationTargetReference `json:"sourceTarget"`
	DestinationTarget   ImplementationTargetReference `json:"destinationTarget"`
	Diagnostic          *ImplementationLinkDiagnostic `json:"diagnostic"`
}

type SemanticVerdictDiagnostic struct {
	Kind                  string                 `json:"kind"`
	RelatedDefinitionIDs  []string               `json:"relatedDefinitionIds"`
	ObservationDiagnostic *ObservationDiagnostic `json:"observationDiagnostic"`
}

type SemanticClauseVerdict struct {
	PropertyDefinitionID    string            `json:"propertyDefinitionId"`
	ClauseDefinitionID      string            `json:"clauseDefinitionId"`
	Status                  string            `json:"status"`
	Coordinates             []ModelCoordinate `json:"coordinates"`
	QueryLimits             Limits            `json:"queryLimits"`
	PropertyLimit           *Limit            `json:"propertyLimit"`
	EvidenceLimit           Limit             `json:"evidenceLimit"`
	ProvenanceDefinitionIDs []string          `json:"provenanceDefinitionIds"`
	EvidenceLinks           []EvidenceLink    `json:"evidenceLinks"`
}

type PropertyVerdict struct {
	QueryDefinitionID           string                     `json:"queryDefinitionId"`
	PropertyDefinitionID        string                     `json:"propertyDefinitionId"`
	PropertyBehaviorFingerprint string                     `json:"propertyBehaviorFingerprint"`
	TraceID                     *string                    `json:"traceId"`
	Status                      string                     `json:"status"`
	QueryLimits                 Limits                     `json:"queryLimits"`
	EvidenceLimit               *Limit                     `json:"evidenceLimit"`
	ProvenanceDefinitionIDs     []string                   `json:"provenanceDefinitionIds"`
	Clauses                     []SemanticClauseVerdict    `json:"clauses"`
	Diagnostic                  *SemanticVerdictDiagnostic `json:"diagnostic"`
}

type QuerySummary struct {
	QueryDefinitionID               string            `json:"queryDefinitionId"`
	Status                          string            `json:"status"`
	QueryLimits                     Limits            `json:"queryLimits"`
	RequiredPropertyDefinitionIDs   []string          `json:"requiredPropertyDefinitionIds"`
	PropertyVerdicts                []PropertyVerdict `json:"propertyVerdicts"`
	MissingPropertyDefinitionIDs    []string          `json:"missingPropertyDefinitionIds"`
	DuplicatePropertyDefinitionIDs  []string          `json:"duplicatePropertyDefinitionIds"`
	UnexpectedPropertyDefinitionIDs []string          `json:"unexpectedPropertyDefinitionIds"`
	DivergentPropertyDefinitionIDs  []string          `json:"divergentPropertyDefinitionIds"`
	WrongQueryResultDefinitionIDs   []string          `json:"wrongQueryResultDefinitionIds"`
	TraceIDs                        []string          `json:"traceIds"`
}

type StagedLimit struct {
	Stage string `json:"stage"`
	Limit Limit  `json:"limit"`
}

type Result struct {
	FormatVersion               string                   `json:"formatVersion"`
	RunIdentity                 string                   `json:"runIdentity"`
	BehaviorFingerprint         string                   `json:"behaviorFingerprint"`
	Experiment                  ArtifactBinding          `json:"experiment"`
	RuntimeConfiguration        ArtifactBinding          `json:"runtimeConfiguration"`
	Run                         ArtifactBinding          `json:"run"`
	RawEvidence                 ArtifactBinding          `json:"rawEvidence"`
	Evidence                    ArtifactBinding          `json:"evidence"`
	OperationalStatus           string                   `json:"operationalStatus"`
	ObservationEvaluationStatus string                   `json:"observationEvaluationStatus"`
	ImplementationLink          ImplementationLinkRecord `json:"implementationLink"`
	ImplementationLinkStatus    string                   `json:"implementationLinkStatus"`
	PropertyVerdicts            []PropertyVerdict        `json:"propertyVerdicts"`
	QuerySummary                QuerySummary             `json:"querySummary"`
	SemanticStatus              string                   `json:"semanticStatus"`
	Limits                      []StagedLimit            `json:"limits"`
	KnownGaps                   []KnownGap               `json:"knownGaps"`
	CleanupStatus               string                   `json:"cleanupStatus"`
	EvaluationOutcomeChecksum   *string                  `json:"evaluationOutcomeChecksum"`
	Provenance                  Provenance               `json:"provenance"`
	ProvenanceChecksum          string                   `json:"provenanceChecksum"`
	ArtifactChecksum            string                   `json:"artifactChecksum,omitempty"`
}

func CanonicalEvidenceBytes(document Evidence) ([]byte, error) {
	return encodeJSONLine(document)
}

func ExpectedEvidenceChecksum(document Evidence) (string, error) {
	document.ArtifactChecksum = ""
	encoded, err := encodeJSONLine(document)
	if err != nil {
		return "", err
	}
	return derive(evidenceChecksumDomain, encoded), nil
}

func SealEvidence(document Evidence) (Evidence, error) {
	provenanceChecksum, err := ExpectedProvenanceChecksum(document.Provenance)
	if err != nil {
		return Evidence{}, err
	}
	document.ProvenanceChecksum = provenanceChecksum
	artifactChecksum, err := ExpectedEvidenceChecksum(document)
	if err != nil {
		return Evidence{}, err
	}
	document.ArtifactChecksum = artifactChecksum
	return document, nil
}

func VerifyEvidenceProvenanceChecksum(document Evidence) error {
	expected, err := ExpectedProvenanceChecksum(document.Provenance)
	if err != nil {
		return err
	}
	if document.ProvenanceChecksum != expected {
		return fmt.Errorf("Evidence provenance checksum mismatch: got %q, want %q",
			document.ProvenanceChecksum, expected)
	}
	return nil
}

func VerifyEvidenceArtifactChecksum(document Evidence) error {
	expected, err := ExpectedEvidenceChecksum(document)
	if err != nil {
		return err
	}
	if document.ArtifactChecksum != expected {
		return fmt.Errorf("Evidence artifact checksum mismatch: got %q, want %q",
			document.ArtifactChecksum, expected)
	}
	return nil
}

func CanonicalResultBytes(document Result) ([]byte, error) {
	return encodeJSONLine(document)
}

func ExpectedResultChecksum(document Result) (string, error) {
	document.ArtifactChecksum = ""
	encoded, err := encodeJSONLine(document)
	if err != nil {
		return "", err
	}
	return derive(resultChecksumDomain, encoded), nil
}

func SealResult(document Result) (Result, error) {
	provenanceChecksum, err := ExpectedProvenanceChecksum(document.Provenance)
	if err != nil {
		return Result{}, err
	}
	document.ProvenanceChecksum = provenanceChecksum
	artifactChecksum, err := ExpectedResultChecksum(document)
	if err != nil {
		return Result{}, err
	}
	document.ArtifactChecksum = artifactChecksum
	return document, nil
}

func VerifyResultProvenanceChecksum(document Result) error {
	expected, err := ExpectedProvenanceChecksum(document.Provenance)
	if err != nil {
		return err
	}
	if document.ProvenanceChecksum != expected {
		return fmt.Errorf("Result provenance checksum mismatch: got %q, want %q",
			document.ProvenanceChecksum, expected)
	}
	return nil
}

func VerifyResultArtifactChecksum(document Result) error {
	expected, err := ExpectedResultChecksum(document)
	if err != nil {
		return err
	}
	if document.ArtifactChecksum != expected {
		return fmt.Errorf("Result artifact checksum mismatch: got %q, want %q",
			document.ArtifactChecksum, expected)
	}
	return nil
}

func ValidateEvidence(document Evidence) error {
	if document.FormatVersion != EvidenceFormat {
		return fmt.Errorf("unsupported format %q", document.FormatVersion)
	}
	if !validDefinitionID(document.RunIdentity) || !ValidDigest(document.BehaviorFingerprint) ||
		!ValidDigest(document.ProvenanceChecksum) || !ValidDigest(document.ArtifactChecksum) {
		return errors.New("Evidence identity, checksum, or behavior fingerprint is malformed")
	}
	for _, binding := range []struct {
		label  string
		value  ArtifactBinding
		format string
	}{
		{label: "experiment", value: document.Experiment, format: ExperimentFormat},
		{label: "runtime configuration", value: document.RuntimeConfiguration, format: RuntimeConfigurationFormat},
		{label: "Run", value: document.Run, format: ExperimentRunFormat},
		{label: "RawEvidence", value: document.RawEvidence, format: RawEvidenceFormat},
	} {
		if err := validateArtifactBinding(binding.label, binding.value, binding.format); err != nil {
			return err
		}
	}
	if err := validateDefinitionReference("observation program", document.ObservationProgram); err != nil {
		return err
	}
	if err := validateDefinitionReference("mapping", document.Mapping); err != nil {
		return err
	}
	if document.EvidenceLinks == nil || document.Dispositions == nil || document.Diagnostics == nil ||
		document.KnownGaps == nil {
		return errors.New("Evidence arrays must not be null")
	}
	if err := validateFieldDispositionRecords(document.Dispositions); err != nil {
		return err
	}
	if err := validateObservationStatusMatrix(document); err != nil {
		return err
	}
	if err := validateKnownGaps(document.KnownGaps); err != nil {
		return err
	}
	return validateProvenance(document.Provenance)
}

func validateObservationStatusMatrix(document Evidence) error {
	switch document.ObservationEvaluationStatus {
	case "accepted":
		if document.EvidenceBackedModelTrace == nil || len(document.EvidenceLinks) == 0 ||
			len(document.Diagnostics) != 0 {
			return errors.New("accepted Evidence requires one complete trace, links, and no diagnostics")
		}
		if err := validateEvidenceBackedModelTrace(*document.EvidenceBackedModelTrace); err != nil {
			return err
		}
		trace := *document.EvidenceBackedModelTrace
		if trace.ObservationPlan != document.Mapping ||
			trace.MappingDefinitionID != document.Mapping.DefinitionID ||
			trace.MappingBehaviorFingerprint != document.Mapping.BehaviorFingerprint {
			return errors.New("evidence-backed Model Trace does not match the Evidence mapping")
		}
		if err := validateEvidenceLinks(document.EvidenceLinks, *document.EvidenceBackedModelTrace); err != nil {
			return err
		}
	case "unknown", "conflict", "unsupported":
		if document.EvidenceBackedModelTrace != nil || len(document.EvidenceLinks) != 0 ||
			len(document.Diagnostics) != 1 {
			return fmt.Errorf("%s Evidence requires no trace or links and exactly one diagnostic",
				document.ObservationEvaluationStatus)
		}
		diagnostic := document.Diagnostics[0]
		if err := validateObservationDiagnostic(diagnostic); err != nil {
			return err
		}
		status, ok := observationFailureStatus(diagnostic.Kind)
		if !ok || status != document.ObservationEvaluationStatus {
			return fmt.Errorf("observation diagnostic %q does not match status %q",
				diagnostic.Kind, document.ObservationEvaluationStatus)
		}
	case "":
		return errors.New("observation evaluation status is empty")
	default:
		return fmt.Errorf("observation evaluation status %q is invalid", document.ObservationEvaluationStatus)
	}
	for _, diagnostic := range document.Diagnostics {
		if diagnostic.ObservationPlanDefinitionID != document.Mapping.DefinitionID {
			return errors.New("observation diagnostic plan does not match mapping")
		}
	}
	return nil
}

func validateDefinitionReference(label string, reference DefinitionReference) error {
	if !validDefinitionID(reference.DefinitionID) || !ValidDigest(reference.BehaviorFingerprint) {
		return fmt.Errorf("%s reference is malformed", label)
	}
	return nil
}

func validateEvidenceBackedModelTrace(trace EvidenceBackedModelTrace) error {
	if !validTraceID(trace.TraceID) || trace.TraceID != trace.Trace.TraceID ||
		!validDefinitionID(trace.MappingDefinitionID) || trace.MappingVersion.IsZero() ||
		!ValidDigest(trace.MappingBehaviorFingerprint) || !validDefinitionID(trace.ProfileDefinitionID) ||
		trace.ProfileVersion.IsZero() || !trace.SourceClosed {
		return errors.New("Evidence-backed Model Trace identity or closure is malformed")
	}
	if err := validateDefinitionReference("observation plan", trace.ObservationPlan); err != nil {
		return err
	}
	if err := validateProvenance(Provenance{SourceDefinitionIDs: []string{trace.MappingDefinitionID},
		SourceLocations: []SourceLocation{trace.Source}}); err != nil {
		return fmt.Errorf("Evidence-backed Model Trace source: %w", err)
	}
	if trace.Vocabulary == nil || trace.EvidenceDefinitionIDs == nil || trace.Trace.Steps == nil {
		return errors.New("Evidence-backed Model Trace arrays must not be null")
	}
	if err := validateMeaningProvisions(trace.Vocabulary); err != nil {
		return err
	}
	if err := validateDefinitionIDSet("Evidence definition ID", trace.EvidenceDefinitionIDs); err != nil {
		return err
	}
	if len(trace.EvidenceDefinitionIDs) == 0 {
		return errors.New("Evidence-backed Model Trace requires Evidence identities")
	}
	if err := validateEvidenceLimit(trace.AppliedLimit); err != nil {
		return err
	}
	if err := validateModelValue("trace initial state", trace.Trace.InitialState); err != nil {
		return err
	}
	for index, step := range trace.Trace.Steps {
		expected := NaturalFromUint64(uint64(index + 1))
		if step.Position != expected || step.Observations == nil {
			return fmt.Errorf("Model Trace step %d is not contiguous or has null observations", index+1)
		}
		for _, value := range []struct {
			label string
			value ModelValue
		}{
			{label: "selected action", value: step.SelectedAction},
			{label: "model outcome", value: step.ModelOutcome},
			{label: "resulting state", value: step.ResultingState},
		} {
			if err := validateModelValue(value.label, value.value); err != nil {
				return err
			}
		}
		if err := validateModelValues("observation", step.Observations); err != nil {
			return err
		}
	}
	return validateTraceVocabulary(trace.Trace, trace.Vocabulary)
}

func validateTraceVocabulary(trace ModelTrace, vocabulary []MeaningProvision) error {
	provided := make(map[string]string, len(vocabulary))
	for _, meaning := range vocabulary {
		provided[meaning.DefinitionID] = meaning.Kind
	}
	type typedModelValue struct {
		value ModelValue
		kind  string
	}
	values := []typedModelValue{{value: trace.InitialState, kind: "state"}}
	for _, step := range trace.Steps {
		values = append(values,
			typedModelValue{value: step.SelectedAction, kind: "action"},
			typedModelValue{value: step.ModelOutcome, kind: "outcome"},
			typedModelValue{value: step.ResultingState, kind: "state"},
		)
		for _, observation := range step.Observations {
			values = append(values, typedModelValue{value: observation, kind: "observation"})
		}
	}
	for _, expected := range values {
		if provided[expected.value.DefinitionID] != expected.kind {
			return fmt.Errorf("Model Trace value %q has no matching vocabulary provision",
				expected.value.DefinitionID)
		}
	}
	return nil
}

func validateMeaningProvisions(values []MeaningProvision) error {
	if !slices.IsSortedFunc(values, func(left, right MeaningProvision) int {
		return strings.Compare(left.DefinitionID, right.DefinitionID)
	}) {
		return errors.New("meaning provisions are not in canonical order")
	}
	for index, value := range values {
		if index > 0 && value.DefinitionID == values[index-1].DefinitionID {
			return fmt.Errorf("duplicate meaning provision %q", value.DefinitionID)
		}
		if !validDefinitionID(value.DefinitionID) || !validDefinitionKind(value.Kind) || value.CanonicalBehavior == "" {
			return fmt.Errorf("meaning provision %q is malformed", value.DefinitionID)
		}
	}
	return nil
}

func validDefinitionKind(kind string) bool {
	switch kind {
	case "state", "action", "outcome", "observation", "relation", "capability", "provider",
		"law", "connector", "target", "kernel", "experiment-space", "variation-axis", "choice",
		"fault", "coverage-goal":
		return true
	default:
		return false
	}
}

func validateFieldDispositionRecords(records []FieldDispositionRecord) error {
	if !slices.IsSortedFunc(records, func(left, right FieldDispositionRecord) int {
		return compareFieldReference(left.Field, right.Field)
	}) {
		return errors.New("field dispositions are not in canonical order")
	}
	for index, record := range records {
		if index > 0 && record.Field == records[index-1].Field {
			return fmt.Errorf("duplicate field disposition for %q", record.Field.FieldDefinitionID)
		}
		if err := validateFieldReference(record.Field); err != nil {
			return err
		}
		switch record.Disposition {
		case "hash":
			if record.DigestPolicyDefinitionID == nil || !validDefinitionID(*record.DigestPolicyDefinitionID) {
				return errors.New("hash disposition requires one digest policy Definition ID")
			}
		case "retain", "redact", "reject":
			if record.DigestPolicyDefinitionID != nil {
				return fmt.Errorf("%s disposition must not carry a digest policy", record.Disposition)
			}
		default:
			return fmt.Errorf("field disposition %q is invalid", record.Disposition)
		}
	}
	return nil
}

func validateFieldReference(reference FieldReference) error {
	if !validDefinitionID(reference.KindDefinitionID) || !validDefinitionID(reference.FieldDefinitionID) {
		return errors.New("field reference is malformed")
	}
	return nil
}

func compareFieldReference(left, right FieldReference) int {
	if comparison := strings.Compare(left.KindDefinitionID, right.KindDefinitionID); comparison != 0 {
		return comparison
	}
	return strings.Compare(left.FieldDefinitionID, right.FieldDefinitionID)
}

func validateEvidenceLinks(links []EvidenceLink, trace EvidenceBackedModelTrace) error {
	expected := expectedModelCoordinates(trace.Trace)
	if len(links) != len(expected) {
		return errors.New("Evidence Links are not a bijection with Model Trace coordinates")
	}
	for index, link := range links {
		if compareModelCoordinate(link.Coordinate, expected[index]) != 0 {
			return fmt.Errorf("Evidence Link %d is not in canonical Model coordinate order", index)
		}
		if err := validateEvidenceLink(link, trace); err != nil {
			return err
		}
		if index > 0 && !reflect.DeepEqual(link.OrderingSupport, links[0].OrderingSupport) {
			return errors.New("Evidence Links do not share exact ordering support")
		}
		if index > 0 && !reflect.DeepEqual(link.ClosureSupport, links[0].ClosureSupport) {
			return errors.New("Evidence Links do not share exact closure support")
		}
	}
	if len(links[0].ClosureSupport) == 0 {
		return errors.New("Evidence Links require closure support")
	}
	lastOrdinalByKind := make(map[string]Natural)
	for _, link := range links {
		for _, fact := range link.OrderingSupport {
			current, found := lastOrdinalByKind[fact.KindDefinitionID]
			if !found || compareNatural(current, fact.Ordinal) < 0 {
				lastOrdinalByKind[fact.KindDefinitionID] = fact.Ordinal
			}
		}
	}
	closureByKind := make(map[string]Natural, len(links[0].ClosureSupport))
	for _, closure := range links[0].ClosureSupport {
		closureByKind[closure.KindDefinitionID] = closure.LastOrdinal
		expected, found := lastOrdinalByKind[closure.KindDefinitionID]
		if !found {
			expected = NaturalFromUint64(0)
		}
		if closure.LastOrdinal != expected {
			return fmt.Errorf("Evidence closure support %q has a stale terminal ordinal",
				closure.KindDefinitionID)
		}
	}
	for kind := range lastOrdinalByKind {
		if _, found := closureByKind[kind]; !found {
			return fmt.Errorf("Evidence ordering kind %q has no closure support", kind)
		}
	}
	return nil
}

func validateEvidenceLink(link EvidenceLink, trace EvidenceBackedModelTrace) error {
	if err := validateModelCoordinate(link.Coordinate); err != nil {
		return err
	}
	if link.MappingDefinitionID != trace.MappingDefinitionID || link.MappingVersion != trace.MappingVersion ||
		link.MappingBehaviorFingerprint != trace.MappingBehaviorFingerprint ||
		link.ProfileDefinitionID != trace.ProfileDefinitionID || link.ProfileVersion != trace.ProfileVersion ||
		!validDefinitionID(link.RuleDefinitionID) || !ValidDigest(link.MeaningBehaviorFingerprint) ||
		link.AppliedLimit != trace.AppliedLimit {
		return errors.New("Evidence Link identity does not match its Evidence-backed Model Trace")
	}
	for _, ids := range []struct {
		label string
		value []string
	}{
		{label: "Evidence Link Evidence definition ID", value: link.EvidenceDefinitionIDs},
		{label: "Evidence Link binding definition ID", value: link.BindingDefinitionIDs},
	} {
		if ids.value == nil {
			return fmt.Errorf("%ss must not be null", ids.label)
		}
		if err := validateDefinitionIDSet(ids.label, ids.value); err != nil {
			return err
		}
	}
	if len(link.EvidenceDefinitionIDs) == 0 || link.OrderingSupport == nil || link.ClosureSupport == nil ||
		link.AppliedDispositions == nil {
		return errors.New("Evidence Link support arrays are incomplete")
	}
	for _, evidenceDefinitionID := range link.EvidenceDefinitionIDs {
		if !slices.Contains(trace.EvidenceDefinitionIDs, evidenceDefinitionID) {
			return errors.New("Evidence Link identity is absent from its Evidence-backed Model Trace")
		}
	}
	if err := validateOrderingSupport(link.OrderingSupport); err != nil {
		return err
	}
	orderingEvidenceIDs := make([]string, len(link.OrderingSupport))
	for index, fact := range link.OrderingSupport {
		orderingEvidenceIDs[index] = fact.FactDefinitionID
	}
	if !slices.Equal(orderingEvidenceIDs, trace.EvidenceDefinitionIDs) {
		return errors.New("Evidence Link ordering support does not match its Evidence-backed Model Trace")
	}
	if err := validateClosureSupport(link.ClosureSupport); err != nil {
		return err
	}
	if err := validateAppliedDispositions(link.AppliedDispositions); err != nil {
		return err
	}
	return nil
}

func validateOrderingSupport(facts []EvidenceOrderingFact) error {
	if !slices.IsSortedFunc(facts, func(left, right EvidenceOrderingFact) int {
		return strings.Compare(left.FactDefinitionID, right.FactDefinitionID)
	}) {
		return errors.New("Evidence ordering support is not in canonical order")
	}
	for index, fact := range facts {
		if index > 0 && fact.FactDefinitionID == facts[index-1].FactDefinitionID {
			return fmt.Errorf("duplicate Evidence ordering fact %q", fact.FactDefinitionID)
		}
		if !validDefinitionID(fact.FactDefinitionID) || !validDefinitionID(fact.KindDefinitionID) ||
			fact.CausalFactDefinitionIDs == nil {
			return errors.New("Evidence ordering fact is malformed")
		}
		if err := validateDefinitionIDSet("causal fact definition ID", fact.CausalFactDefinitionIDs); err != nil {
			return err
		}
	}
	return nil
}

func validateClosureSupport(facts []EvidenceClosureFact) error {
	if !slices.IsSortedFunc(facts, func(left, right EvidenceClosureFact) int {
		return strings.Compare(left.KindDefinitionID, right.KindDefinitionID)
	}) {
		return errors.New("Evidence closure support is not in canonical order")
	}
	for index, fact := range facts {
		if index > 0 && fact.KindDefinitionID == facts[index-1].KindDefinitionID {
			return fmt.Errorf("duplicate Evidence closure fact %q", fact.KindDefinitionID)
		}
		if !validDefinitionID(fact.KindDefinitionID) {
			return errors.New("Evidence closure fact is malformed")
		}
	}
	return nil
}

func validateAppliedDispositions(dispositions []AppliedFieldDisposition) error {
	if !slices.IsSortedFunc(dispositions, func(left, right AppliedFieldDisposition) int {
		return compareFieldReference(left.Field, right.Field)
	}) {
		return errors.New("applied field dispositions are not in canonical order")
	}
	for index, disposition := range dispositions {
		if index > 0 && disposition.Field == dispositions[index-1].Field {
			return fmt.Errorf("duplicate applied field disposition %q", disposition.Field.FieldDefinitionID)
		}
		if err := validateFieldReference(disposition.Field); err != nil {
			return err
		}
		switch disposition.Kind {
		case "retained":
			if disposition.NormalizedValue == nil || disposition.DigestPolicyDefinitionID != nil ||
				disposition.DigestToken != nil {
				return errors.New("retained disposition has an invalid value matrix")
			}
		case "redacted":
			if disposition.NormalizedValue != nil || disposition.DigestPolicyDefinitionID != nil ||
				disposition.DigestToken != nil {
				return errors.New("redacted disposition must not carry material")
			}
		case "digest-token":
			if disposition.NormalizedValue != nil || disposition.DigestPolicyDefinitionID == nil ||
				disposition.DigestToken == nil || !validDefinitionID(*disposition.DigestPolicyDefinitionID) ||
				*disposition.DigestToken == "" {
				return errors.New("digest-token disposition has an invalid value matrix")
			}
		default:
			return fmt.Errorf("applied field disposition kind %q is invalid", disposition.Kind)
		}
	}
	return nil
}

func expectedModelCoordinates(trace ModelTrace) []ModelCoordinate {
	coordinates := []ModelCoordinate{{Kind: "initial-state"}}
	for index, step := range trace.Steps {
		position := NaturalFromUint64(uint64(index + 1))
		coordinates = append(coordinates,
			ModelCoordinate{Kind: "selected-action", Step: naturalPointer(position)},
			ModelCoordinate{Kind: "model-outcome", Step: naturalPointer(position)},
			ModelCoordinate{Kind: "resulting-state", Step: naturalPointer(position)},
		)
		for observationIndex := range step.Observations {
			observationPosition := NaturalFromUint64(uint64(observationIndex + 1))
			coordinates = append(coordinates, ModelCoordinate{
				Kind: "observation", Step: naturalPointer(position), Position: naturalPointer(observationPosition),
			})
		}
	}
	slices.SortFunc(coordinates, compareModelCoordinate)
	return coordinates
}

func naturalPointer(value Natural) *Natural {
	copy := value
	return &copy
}

func validateModelCoordinate(coordinate ModelCoordinate) error {
	switch coordinate.Kind {
	case "initial-state":
		if coordinate.Step != nil || coordinate.Position != nil {
			return errors.New("initial-state coordinate must have null step and position")
		}
	case "selected-action", "model-outcome", "resulting-state":
		if coordinate.Step == nil || coordinate.Step.IsZero() || coordinate.Position != nil {
			return fmt.Errorf("%s coordinate has an invalid step or position", coordinate.Kind)
		}
	case "observation":
		if coordinate.Step == nil || coordinate.Step.IsZero() || coordinate.Position == nil ||
			coordinate.Position.IsZero() {
			return errors.New("observation coordinate requires positive step and position")
		}
	default:
		return fmt.Errorf("Model coordinate kind %q is invalid", coordinate.Kind)
	}
	return nil
}

func compareModelCoordinate(left, right ModelCoordinate) int {
	if comparison := compareInt(modelCoordinateRank(left.Kind), modelCoordinateRank(right.Kind)); comparison != 0 {
		return comparison
	}
	if comparison := compareOptionalNatural(left.Step, right.Step); comparison != 0 {
		return comparison
	}
	return compareOptionalNatural(left.Position, right.Position)
}

func modelCoordinateRank(kind string) int {
	switch kind {
	case "initial-state":
		return 0
	case "selected-action":
		return 1
	case "model-outcome":
		return 2
	case "resulting-state":
		return 3
	case "observation":
		return 4
	default:
		return 5
	}
}

func compareOptionalNatural(left, right *Natural) int {
	if left == nil && right == nil {
		return 0
	}
	if left == nil {
		return -1
	}
	if right == nil {
		return 1
	}
	return compareNatural(*left, *right)
}

func validateEvidenceLimit(limit Limit) error {
	if limit.Value.IsZero() || limit.Unit != "evidence-records" {
		return errors.New("Evidence Limit must be positive evidence-records")
	}
	return nil
}

func validateObservationDiagnostic(diagnostic ObservationDiagnostic) error {
	if !validDefinitionID(diagnostic.ObservationPlanDefinitionID) || diagnostic.RelatedDefinitionIDs == nil ||
		diagnostic.Alternatives == nil {
		return errors.New("observation diagnostic is malformed")
	}
	if err := validateDefinitionIDSet("observation diagnostic related Definition ID",
		diagnostic.RelatedDefinitionIDs); err != nil {
		return err
	}
	if err := validateDefinitionIDSet("observation diagnostic alternative", diagnostic.Alternatives); err != nil {
		return err
	}
	switch diagnostic.Kind {
	case "evidence-bound-exhausted":
		if diagnostic.AppliedLimit == nil || diagnostic.ObservedCount == nil ||
			len(diagnostic.Alternatives) != 0 || diagnostic.MissingDiscriminatorDefinitionID != nil {
			return errors.New("evidence-bound-exhausted diagnostic has an invalid nullable field matrix")
		}
		if err := validateEvidenceLimit(*diagnostic.AppliedLimit); err != nil {
			return err
		}
	case "compatible-alternatives":
		if diagnostic.AppliedLimit != nil || diagnostic.ObservedCount != nil ||
			len(diagnostic.Alternatives) == 0 || diagnostic.MissingDiscriminatorDefinitionID == nil ||
			!validDefinitionID(*diagnostic.MissingDiscriminatorDefinitionID) {
			return errors.New("compatible-alternatives diagnostic has an invalid nullable field matrix")
		}
	default:
		if _, ok := observationFailureStatus(diagnostic.Kind); !ok {
			return fmt.Errorf("observation diagnostic kind %q is invalid", diagnostic.Kind)
		}
		if diagnostic.AppliedLimit != nil || diagnostic.ObservedCount != nil ||
			len(diagnostic.Alternatives) != 0 || diagnostic.MissingDiscriminatorDefinitionID != nil {
			return fmt.Errorf("observation diagnostic %q has an invalid nullable field matrix", diagnostic.Kind)
		}
	}
	return nil
}

func observationFailureStatus(kind string) (string, bool) {
	switch kind {
	case "profile-mismatch", "profile-version-mismatch", "kind-mismatch", "field-mismatch",
		"raw-value-leakage", "redacted-value-leakage", "rejected-value-leakage",
		"rejected-field-present", "digest-policy-mismatch", "disallowed-raw-material":
		return "unsupported", true
	case "duplicate-evidence-identity", "contradictory-fact", "contradictory-binding",
		"contradictory-order", "misdirected-fault-receipt", "duplicate-model-coordinate",
		"extra-model-coordinate", "inconsistent-evidence-link", "digest-collision":
		return "conflict", true
	case "empty-evidence", "evidence-bound-exhausted", "missing-initial-state", "missing-closure",
		"sequence-gap", "missing-causal-parent", "normalization-failure", "unresolved-binding",
		"incomparable-ordering", "compatible-alternatives", "zero-usable-interpretations",
		"absent-model-coordinate", "unconsumed-reference", "missing-closure-support",
		"missing-order-support":
		return "unknown", true
	default:
		return "", false
	}
}

func ValidateResult(document Result) error {
	if document.FormatVersion != ResultFormat {
		return fmt.Errorf("unsupported format %q", document.FormatVersion)
	}
	if !validDefinitionID(document.RunIdentity) || !ValidDigest(document.BehaviorFingerprint) ||
		!ValidDigest(document.ProvenanceChecksum) || !ValidDigest(document.ArtifactChecksum) {
		return errors.New("Result identity, checksum, or behavior fingerprint is malformed")
	}
	for _, binding := range []struct {
		label  string
		value  ArtifactBinding
		format string
	}{
		{label: "experiment", value: document.Experiment, format: ExperimentFormat},
		{label: "runtime configuration", value: document.RuntimeConfiguration, format: RuntimeConfigurationFormat},
		{label: "Run", value: document.Run, format: ExperimentRunFormat},
		{label: "RawEvidence", value: document.RawEvidence, format: RawEvidenceFormat},
		{label: "Evidence", value: document.Evidence, format: EvidenceFormat},
	} {
		if err := validateArtifactBinding(binding.label, binding.value, binding.format); err != nil {
			return err
		}
	}
	if document.PropertyVerdicts == nil || document.QuerySummary.PropertyVerdicts == nil ||
		document.Limits == nil || document.KnownGaps == nil {
		return errors.New("Result arrays must not be null")
	}
	if document.OperationalStatus != "succeeded" && document.OperationalStatus != "failed" &&
		document.OperationalStatus != "incomplete" {
		return fmt.Errorf("operational status %q is invalid", document.OperationalStatus)
	}
	if document.ObservationEvaluationStatus != "accepted" &&
		document.ObservationEvaluationStatus != "unknown" &&
		document.ObservationEvaluationStatus != "conflict" &&
		document.ObservationEvaluationStatus != "unsupported" {
		return fmt.Errorf("observation evaluation status %q is invalid", document.ObservationEvaluationStatus)
	}
	if err := validateImplementationLink(document.ImplementationLink, document.ImplementationLinkStatus); err != nil {
		return err
	}
	if err := validatePropertyVerdicts(document.PropertyVerdicts); err != nil {
		return err
	}
	if err := validateQuerySummary(document.QuerySummary, document.PropertyVerdicts); err != nil {
		return err
	}
	if document.SemanticStatus != document.QuerySummary.Status {
		return errors.New("semantic status does not match Query summary status")
	}
	if err := validateResultStatusMatrix(document); err != nil {
		return err
	}
	if err := validateStagedLimits(document.Limits); err != nil {
		return err
	}
	if err := validateKnownGaps(document.KnownGaps); err != nil {
		return err
	}
	if document.CleanupStatus != "complete" && document.CleanupStatus != "incomplete" &&
		document.CleanupStatus != "failed" {
		return fmt.Errorf("cleanup status %q is invalid", document.CleanupStatus)
	}
	return validateProvenance(document.Provenance)
}

type implementationLinkDiagnosticTargetIdentity struct {
	ID                  string `json:"id"`
	Kind                string `json:"kind"`
	BehaviorFingerprint string `json:"behaviorFingerprint"`
}

type implementationLinkDiagnosticIdentityView struct {
	ImplementationLinkID                  string                                     `json:"implementationLinkId"`
	ImplementationLinkBehaviorFingerprint string                                     `json:"implementationLinkBehaviorFingerprint"`
	SourceTarget                          implementationLinkDiagnosticTargetIdentity `json:"sourceTarget"`
	DestinationTarget                     implementationLinkDiagnosticTargetIdentity `json:"destinationTarget"`
	Kind                                  string                                     `json:"kind"`
	Status                                string                                     `json:"status"`
	Coordinate                            *string                                    `json:"coordinate"`
	RelatedDefinitionIDs                  []string                                   `json:"relatedDefinitionIds"`
	SourceSetupBehaviorFingerprint        *string                                    `json:"sourceSetupBehaviorFingerprint"`
	AppliedLimit                          *Limit                                     `json:"appliedLimit"`
	ObservedCount                         *Natural                                   `json:"observedCount"`
	KnownGapCode                          *string                                    `json:"knownGapCode"`
	KnownGapReason                        *string                                    `json:"knownGapReason"`
	UnsupportedVocabularyKind             *string                                    `json:"unsupportedVocabularyKind"`
	EvidenceLinkBehaviorFingerprint       *string                                    `json:"evidenceLinkBehaviorFingerprint"`
}

// ExpectedImplementationLinkDiagnosticIdentity verifies the frozen transport projection only.
// It does not apply an Implementation Link or derive any diagnostic from model values.
func ExpectedImplementationLinkDiagnosticIdentity(record ImplementationLinkRecord) (string, error) {
	if record.Diagnostic == nil {
		return "", errors.New("Implementation Link diagnostic identity requires a diagnostic")
	}
	diagnostic := *record.Diagnostic
	status, ok := implementationLinkFailureStatus(diagnostic.Kind)
	if !ok {
		return "", fmt.Errorf("Implementation Link diagnostic kind %q is invalid", diagnostic.Kind)
	}
	var coordinate *string
	if diagnostic.Coordinate != nil {
		value, err := modelCoordinateIdentityName(*diagnostic.Coordinate)
		if err != nil {
			return "", err
		}
		coordinate = &value
	}
	view := implementationLinkDiagnosticIdentityView{
		ImplementationLinkID:                  record.DefinitionID,
		ImplementationLinkBehaviorFingerprint: record.BehaviorFingerprint,
		SourceTarget: implementationLinkDiagnosticTargetIdentity{
			ID: record.SourceTarget.DefinitionID, Kind: record.SourceTarget.Kind,
			BehaviorFingerprint: record.SourceTarget.BehaviorFingerprint,
		},
		DestinationTarget: implementationLinkDiagnosticTargetIdentity{
			ID: record.DestinationTarget.DefinitionID, Kind: record.DestinationTarget.Kind,
			BehaviorFingerprint: record.DestinationTarget.BehaviorFingerprint,
		},
		Kind:                            diagnostic.Kind,
		Status:                          status,
		Coordinate:                      coordinate,
		RelatedDefinitionIDs:            diagnostic.RelatedDefinitionIDs,
		SourceSetupBehaviorFingerprint:  diagnostic.SourceSetupBehaviorFingerprint,
		AppliedLimit:                    diagnostic.AppliedLimit,
		ObservedCount:                   diagnostic.ObservedCount,
		KnownGapCode:                    diagnostic.KnownGapCode,
		KnownGapReason:                  diagnostic.KnownGapReason,
		UnsupportedVocabularyKind:       diagnostic.UnsupportedVocabularyKind,
		EvidenceLinkBehaviorFingerprint: diagnostic.EvidenceLinkBehaviorFingerprint,
	}
	encoded, err := encodeJSONLine(view)
	if err != nil {
		return "", err
	}
	return BehaviorFingerprint(encoded), nil
}

func modelCoordinateIdentityName(coordinate ModelCoordinate) (string, error) {
	if err := validateModelCoordinate(coordinate); err != nil {
		return "", err
	}
	switch coordinate.Kind {
	case "initial-state":
		return coordinate.Kind, nil
	case "selected-action", "model-outcome", "resulting-state":
		return coordinate.Kind + ":" + coordinate.Step.String(), nil
	case "observation":
		return coordinate.Kind + ":" + coordinate.Step.String() + ":" + coordinate.Position.String(), nil
	default:
		return "", fmt.Errorf("Model coordinate kind %q is invalid", coordinate.Kind)
	}
}

func validateImplementationLink(record ImplementationLinkRecord, status string) error {
	if !validDefinitionID(record.DefinitionID) || !ValidDigest(record.BehaviorFingerprint) {
		return errors.New("Implementation Link identity is malformed")
	}
	for _, target := range []ImplementationTargetReference{record.SourceTarget, record.DestinationTarget} {
		if !validDefinitionID(target.DefinitionID) || !validDefinitionKind(target.Kind) ||
			!ValidDigest(target.BehaviorFingerprint) {
			return errors.New("Implementation Link target reference is malformed")
		}
	}
	switch status {
	case "applied", "not-evaluated":
		if record.Diagnostic != nil {
			return fmt.Errorf("%s Implementation Link must not carry a diagnostic", status)
		}
	case "invalid", "unknown", "conflict", "unsupported":
		if record.Diagnostic == nil {
			return fmt.Errorf("%s Implementation Link requires a diagnostic", status)
		}
		if err := validateImplementationLinkDiagnostic(*record.Diagnostic); err != nil {
			return err
		}
		expected, ok := implementationLinkFailureStatus(record.Diagnostic.Kind)
		if !ok || expected != status {
			return fmt.Errorf("Implementation Link diagnostic %q does not match status %q",
				record.Diagnostic.Kind, status)
		}
		expectedIdentity, err := ExpectedImplementationLinkDiagnosticIdentity(record)
		if err != nil {
			return err
		}
		if record.Diagnostic.Identity != expectedIdentity {
			return errors.New("Implementation Link diagnostic identity does not match its persisted projection")
		}
	default:
		return fmt.Errorf("Implementation Link status %q is invalid", status)
	}
	return nil
}

func validateImplementationLinkDiagnostic(diagnostic ImplementationLinkDiagnostic) error {
	if diagnostic.RelatedDefinitionIDs == nil || !ValidDigest(diagnostic.Identity) {
		return errors.New("Implementation Link diagnostic is malformed")
	}
	if err := validateDefinitionIDSet("Implementation Link diagnostic related Definition ID",
		diagnostic.RelatedDefinitionIDs); err != nil {
		return err
	}
	if diagnostic.Coordinate != nil {
		if err := validateModelCoordinate(*diagnostic.Coordinate); err != nil {
			return err
		}
	}
	if diagnostic.SourceSetupBehaviorFingerprint != nil &&
		!ValidDigest(*diagnostic.SourceSetupBehaviorFingerprint) {
		return errors.New("Implementation Link source setup fingerprint is malformed")
	}
	if diagnostic.AppliedLimit != nil {
		if err := validatePositiveLimit(*diagnostic.AppliedLimit); err != nil {
			return err
		}
	}
	if diagnostic.KnownGapCode != nil && !validDefinitionID(*diagnostic.KnownGapCode) {
		return errors.New("Implementation Link Known Gap code is malformed")
	}
	if diagnostic.UnsupportedVocabularyKind != nil && !validDefinitionKind(*diagnostic.UnsupportedVocabularyKind) {
		return errors.New("Implementation Link unsupported vocabulary kind is malformed")
	}
	if diagnostic.EvidenceLinkBehaviorFingerprint != nil &&
		!ValidDigest(*diagnostic.EvidenceLinkBehaviorFingerprint) {
		return errors.New("Implementation Link Evidence Link fingerprint is malformed")
	}
	if diagnostic.Kind == "known-gap" {
		if diagnostic.KnownGapCode == nil || diagnostic.KnownGapReason == nil {
			return errors.New("known-gap Implementation Link diagnostic requires code and reason")
		}
	} else if diagnostic.KnownGapCode != nil || diagnostic.KnownGapReason != nil {
		return errors.New("non-known-gap Implementation Link diagnostic must not carry Known Gap fields")
	}
	if diagnostic.Kind == "unsupported-vocabulary" {
		if diagnostic.UnsupportedVocabularyKind == nil {
			return errors.New("unsupported-vocabulary diagnostic requires a vocabulary kind")
		}
	} else if diagnostic.UnsupportedVocabularyKind != nil {
		return errors.New("non-unsupported-vocabulary diagnostic must not carry a vocabulary kind")
	}
	if diagnostic.Kind == "limit-reached" {
		if diagnostic.AppliedLimit == nil || diagnostic.ObservedCount == nil {
			return errors.New("limit-reached diagnostic requires a Limit and observed count")
		}
	} else if diagnostic.AppliedLimit != nil || diagnostic.ObservedCount != nil {
		return errors.New("non-limit diagnostic must not carry Limit fields")
	}
	return nil
}

func implementationLinkFailureStatus(kind string) (string, bool) {
	switch kind {
	case "stale-source-target", "stale-destination-target", "behavior-fingerprint-drift",
		"source-setup-mismatch", "non-authoritative-source-initial", "non-authoritative-source-step",
		"invalid-coordinate":
		return "invalid", true
	case "absent-coordinate", "limit-reached":
		return "unknown", true
	case "duplicate-coordinate", "contradictory-coordinate", "multiple-mappings", "evidence-link-mismatch":
		return "conflict", true
	case "known-gap", "unsupported-vocabulary":
		return "unsupported", true
	default:
		return "", false
	}
}

func validatePropertyVerdicts(verdicts []PropertyVerdict) error {
	if !slices.IsSortedFunc(verdicts, func(left, right PropertyVerdict) int {
		return strings.Compare(left.PropertyDefinitionID, right.PropertyDefinitionID)
	}) {
		return errors.New("Property verdicts are not in canonical order")
	}
	for index, verdict := range verdicts {
		if index > 0 && verdict.PropertyDefinitionID == verdicts[index-1].PropertyDefinitionID {
			return fmt.Errorf("duplicate Property verdict %q", verdict.PropertyDefinitionID)
		}
		if err := validatePropertyVerdict(verdict); err != nil {
			return err
		}
	}
	return nil
}

func validatePropertyVerdict(verdict PropertyVerdict) error {
	if !validDefinitionID(verdict.QueryDefinitionID) || !validDefinitionID(verdict.PropertyDefinitionID) ||
		!ValidDigest(verdict.PropertyBehaviorFingerprint) || verdict.ProvenanceDefinitionIDs == nil ||
		verdict.Clauses == nil {
		return fmt.Errorf("Property verdict %q is malformed", verdict.PropertyDefinitionID)
	}
	if err := validateLimits(verdict.QueryLimits); err != nil {
		return err
	}
	if err := validateDefinitionIDSet("Property verdict provenance Definition ID",
		verdict.ProvenanceDefinitionIDs); err != nil {
		return err
	}
	switch verdict.Status {
	case "satisfied", "violated":
		if verdict.TraceID == nil || !validTraceID(*verdict.TraceID) || verdict.EvidenceLimit == nil ||
			len(verdict.Clauses) == 0 || verdict.Diagnostic != nil {
			return fmt.Errorf("resolved Property verdict %q has an invalid nullable field matrix",
				verdict.PropertyDefinitionID)
		}
		if err := validateEvidenceLimit(*verdict.EvidenceLimit); err != nil {
			return err
		}
		if err := validateSemanticClauses(verdict); err != nil {
			return err
		}
	case "unknown", "conflict", "unsupported":
		if len(verdict.Clauses) != 0 || verdict.Diagnostic == nil {
			return fmt.Errorf("unresolved Property verdict %q requires a diagnostic and no clauses",
				verdict.PropertyDefinitionID)
		}
		if verdict.TraceID != nil && !validTraceID(*verdict.TraceID) {
			return errors.New("unresolved Property verdict trace ID is malformed")
		}
		if verdict.EvidenceLimit != nil {
			if err := validateEvidenceLimit(*verdict.EvidenceLimit); err != nil {
				return err
			}
		}
		if err := validateSemanticVerdictDiagnostic(*verdict.Diagnostic, verdict.Status); err != nil {
			return err
		}
		tracePresent := verdict.TraceID != nil
		limitPresent := verdict.EvidenceLimit != nil
		switch verdict.Diagnostic.Kind {
		case "query-property-mismatch":
			if tracePresent || limitPresent {
				return errors.New("query-property-mismatch must not carry trace context")
			}
		default:
			if !tracePresent || !limitPresent {
				return fmt.Errorf("semantic diagnostic %q requires trace context", verdict.Diagnostic.Kind)
			}
		}
	default:
		return fmt.Errorf("Property verdict status %q is invalid", verdict.Status)
	}
	return nil
}

func validateSemanticClauses(verdict PropertyVerdict) error {
	if !slices.IsSortedFunc(verdict.Clauses, func(left, right SemanticClauseVerdict) int {
		return strings.Compare(left.ClauseDefinitionID, right.ClauseDefinitionID)
	}) {
		return errors.New("semantic clauses are not in canonical order")
	}
	allSatisfied := true
	for index, clause := range verdict.Clauses {
		if index > 0 && clause.ClauseDefinitionID == verdict.Clauses[index-1].ClauseDefinitionID {
			return fmt.Errorf("duplicate semantic clause %q", clause.ClauseDefinitionID)
		}
		if clause.PropertyDefinitionID != verdict.PropertyDefinitionID ||
			!validDefinitionID(clause.ClauseDefinitionID) ||
			(clause.Status != "satisfied" && clause.Status != "violated") ||
			clause.Coordinates == nil || clause.ProvenanceDefinitionIDs == nil || clause.EvidenceLinks == nil {
			return fmt.Errorf("semantic clause %q is malformed", clause.ClauseDefinitionID)
		}
		if err := validateLimits(clause.QueryLimits); err != nil {
			return err
		}
		if !reflect.DeepEqual(clause.QueryLimits, verdict.QueryLimits) || clause.EvidenceLimit != *verdict.EvidenceLimit {
			return fmt.Errorf("semantic clause %q Limits do not match its Property verdict", clause.ClauseDefinitionID)
		}
		if clause.PropertyLimit != nil {
			if err := validatePositiveLimit(*clause.PropertyLimit); err != nil {
				return err
			}
		}
		if err := validateCoordinateSet(clause.Coordinates); err != nil {
			return err
		}
		if err := validateDefinitionIDSet("semantic clause provenance Definition ID",
			clause.ProvenanceDefinitionIDs); err != nil {
			return err
		}
		if len(clause.EvidenceLinks) != len(clause.Coordinates) {
			return fmt.Errorf("semantic clause %q links do not match coordinates", clause.ClauseDefinitionID)
		}
		for linkIndex, link := range clause.EvidenceLinks {
			if compareModelCoordinate(link.Coordinate, clause.Coordinates[linkIndex]) != 0 {
				return fmt.Errorf("semantic clause %q Evidence Links are not coordinate-aligned",
					clause.ClauseDefinitionID)
			}
			if err := validateModelCoordinate(link.Coordinate); err != nil {
				return err
			}
		}
		allSatisfied = allSatisfied && clause.Status == "satisfied"
	}
	if (verdict.Status == "satisfied") != allSatisfied {
		return fmt.Errorf("Property verdict %q status does not summarize its clauses",
			verdict.PropertyDefinitionID)
	}
	return nil
}

func validateCoordinateSet(coordinates []ModelCoordinate) error {
	if !slices.IsSortedFunc(coordinates, compareModelCoordinate) {
		return errors.New("Model coordinates are not in canonical order")
	}
	for index, coordinate := range coordinates {
		if index > 0 && compareModelCoordinate(coordinates[index-1], coordinate) == 0 {
			return errors.New("duplicate Model coordinate")
		}
		if err := validateModelCoordinate(coordinate); err != nil {
			return err
		}
	}
	return nil
}

func validateSemanticVerdictDiagnostic(diagnostic SemanticVerdictDiagnostic, status string) error {
	if diagnostic.RelatedDefinitionIDs == nil {
		return errors.New("semantic verdict diagnostic related Definition IDs must not be null")
	}
	if err := validateDefinitionIDSet("semantic verdict diagnostic related Definition ID",
		diagnostic.RelatedDefinitionIDs); err != nil {
		return err
	}
	expected := ""
	switch diagnostic.Kind {
	case "observation-evaluation-failure":
		if diagnostic.ObservationDiagnostic == nil {
			return errors.New("observation-evaluation-failure requires its observation diagnostic")
		}
		if err := validateObservationDiagnostic(*diagnostic.ObservationDiagnostic); err != nil {
			return err
		}
		expected, _ = observationFailureStatus(diagnostic.ObservationDiagnostic.Kind)
	case "query-property-mismatch", "missing-capability", "missing-vocabulary", "ambiguous-vocabulary",
		"digest-mismatch":
		expected = "unsupported"
	case "invalid-evidence-bound", "missing-logical-time":
		expected = "unknown"
	default:
		return fmt.Errorf("semantic verdict diagnostic kind %q is invalid", diagnostic.Kind)
	}
	if diagnostic.Kind != "observation-evaluation-failure" && diagnostic.ObservationDiagnostic != nil {
		return errors.New("non-observation semantic diagnostic must not carry an observation diagnostic")
	}
	if expected != status {
		return fmt.Errorf("semantic diagnostic %q does not match status %q", diagnostic.Kind, status)
	}
	return nil
}

func validateQuerySummary(summary QuerySummary, verdicts []PropertyVerdict) error {
	if !validDefinitionID(summary.QueryDefinitionID) || summary.RequiredPropertyDefinitionIDs == nil ||
		summary.MissingPropertyDefinitionIDs == nil || summary.DuplicatePropertyDefinitionIDs == nil ||
		summary.UnexpectedPropertyDefinitionIDs == nil || summary.DivergentPropertyDefinitionIDs == nil ||
		summary.WrongQueryResultDefinitionIDs == nil || summary.TraceIDs == nil {
		return errors.New("Query summary is malformed")
	}
	if err := validateLimits(summary.QueryLimits); err != nil {
		return err
	}
	for _, ids := range []struct {
		label string
		value []string
	}{
		{label: "required Property Definition ID", value: summary.RequiredPropertyDefinitionIDs},
		{label: "missing Property Definition ID", value: summary.MissingPropertyDefinitionIDs},
		{label: "duplicate Property Definition ID", value: summary.DuplicatePropertyDefinitionIDs},
		{label: "unexpected Property Definition ID", value: summary.UnexpectedPropertyDefinitionIDs},
		{label: "divergent Property Definition ID", value: summary.DivergentPropertyDefinitionIDs},
		{label: "wrong Query result Definition ID", value: summary.WrongQueryResultDefinitionIDs},
	} {
		if err := validateDefinitionIDSet(ids.label, ids.value); err != nil {
			return err
		}
	}
	if !slices.IsSorted(summary.TraceIDs) {
		return errors.New("Query summary trace IDs are not in canonical order")
	}
	for index, traceID := range summary.TraceIDs {
		if !validTraceID(traceID) || (index > 0 && traceID == summary.TraceIDs[index-1]) {
			return errors.New("Query summary trace IDs are malformed or duplicated")
		}
	}
	if !reflect.DeepEqual(summary.PropertyVerdicts, verdicts) {
		return errors.New("Query summary Property verdicts are not byte-identical to Result verdicts")
	}
	structuralErrors := len(summary.MissingPropertyDefinitionIDs) + len(summary.DuplicatePropertyDefinitionIDs) +
		len(summary.UnexpectedPropertyDefinitionIDs) + len(summary.DivergentPropertyDefinitionIDs) +
		len(summary.WrongQueryResultDefinitionIDs)
	resolved := len(verdicts) == len(summary.RequiredPropertyDefinitionIDs) && len(verdicts) > 0
	allSatisfied := resolved
	for _, verdict := range verdicts {
		resolved = resolved && (verdict.Status == "satisfied" || verdict.Status == "violated") &&
			verdict.QueryDefinitionID == summary.QueryDefinitionID && reflect.DeepEqual(verdict.QueryLimits, summary.QueryLimits)
		allSatisfied = allSatisfied && verdict.Status == "satisfied"
	}
	if resolved {
		traceID := verdicts[0].TraceID
		resolved = traceID != nil && len(summary.TraceIDs) == 1 && summary.TraceIDs[0] == *traceID
		for _, verdict := range verdicts[1:] {
			resolved = resolved && verdict.TraceID != nil && *verdict.TraceID == *traceID
		}
	}
	expected := "incomplete"
	if structuralErrors == 0 && resolved {
		if allSatisfied {
			expected = "satisfied"
		} else {
			expected = "violated"
		}
	}
	if summary.Status != expected {
		return fmt.Errorf("Query summary status %q is inconsistent; expected %q", summary.Status, expected)
	}
	return nil
}

func validateResultStatusMatrix(document Result) error {
	semanticsResolved := document.SemanticStatus == "satisfied" || document.SemanticStatus == "violated"
	if semanticsResolved != (document.EvaluationOutcomeChecksum != nil) {
		return errors.New("evaluation outcome checksum nullability does not match resolved semantics")
	}
	if document.EvaluationOutcomeChecksum != nil && !ValidDigest(*document.EvaluationOutcomeChecksum) {
		return errors.New("evaluation outcome checksum is malformed")
	}
	if document.ObservationEvaluationStatus != "accepted" {
		if document.ImplementationLinkStatus != "not-evaluated" || len(document.PropertyVerdicts) != 0 ||
			document.SemanticStatus != "incomplete" {
			return errors.New("non-accepted observation must skip Implementation Link and Property evaluation")
		}
	} else if document.ImplementationLinkStatus != "applied" {
		if len(document.PropertyVerdicts) != 0 || document.SemanticStatus != "incomplete" {
			return errors.New("non-applied Implementation Link must skip Property evaluation")
		}
	} else if len(document.PropertyVerdicts) == 0 {
		return errors.New("accepted observation and applied Implementation Link require Property verdicts")
	}
	if semanticsResolved && (document.ObservationEvaluationStatus != "accepted" ||
		document.ImplementationLinkStatus != "applied") {
		return errors.New("resolved semantics require accepted observation and applied Implementation Link")
	}
	return nil
}

func validTraceID(value string) bool {
	return value != ""
}

func validateStagedLimits(limits []StagedLimit) error {
	stages := []string{"observation-evaluation", "implementation-link", "query", "property"}
	last := -1
	for _, staged := range limits {
		index := slices.Index(stages, staged.Stage)
		if index < 0 || index <= last {
			return errors.New("Result Limits are not in closed canonical stage order")
		}
		last = index
		if err := validatePositiveLimit(staged.Limit); err != nil {
			return err
		}
	}
	return nil
}

func validatePositiveLimit(limit Limit) error {
	if limit.Value.IsZero() {
		return errors.New("Limit must be positive")
	}
	if limit.Unit == "evidence-records" {
		return nil
	}
	if !validLimitUnit(limit.Unit) {
		return fmt.Errorf("Limit unit %q is invalid", limit.Unit)
	}
	return nil
}

func RawEvidenceArtifactBinding(document RawEvidence) ArtifactBinding {
	return ArtifactBinding{
		FormatVersion:       document.FormatVersion,
		ArtifactChecksum:    document.ArtifactChecksum,
		BehaviorFingerprint: document.BehaviorFingerprint,
		ProvenanceChecksum:  document.ProvenanceChecksum,
	}
}

func EvidenceArtifactBinding(document Evidence) ArtifactBinding {
	return ArtifactBinding{
		FormatVersion:       document.FormatVersion,
		ArtifactChecksum:    document.ArtifactChecksum,
		BehaviorFingerprint: document.BehaviorFingerprint,
		ProvenanceChecksum:  document.ProvenanceChecksum,
	}
}

func ValidateEvidenceClosure(
	document Evidence,
	experiment Experiment,
	runtimeConfiguration RuntimeConfiguration,
	run ExperimentRun,
	rawEvidence RawEvidence,
) error {
	if err := ValidateRawEvidenceClosure(rawEvidence, experiment, runtimeConfiguration, run); err != nil {
		return err
	}
	experimentBinding, err := ExperimentArtifactBinding(experiment)
	if err != nil {
		return err
	}
	if document.Experiment != experimentBinding ||
		document.RuntimeConfiguration != RuntimeConfigurationArtifactBinding(runtimeConfiguration) ||
		document.Run != ExperimentRunArtifactBinding(run) ||
		document.RawEvidence != RawEvidenceArtifactBinding(rawEvidence) {
		return errors.New("Evidence input Artifact binding does not match its bound member")
	}
	if document.RunIdentity != run.RunIdentity || document.RunIdentity != rawEvidence.RunIdentity {
		return errors.New("Evidence run identity does not match its bound Run")
	}
	if document.ObservationProgram.DefinitionID != runtimeConfiguration.Observation.ProgramDefinitionID ||
		document.ObservationProgram.BehaviorFingerprint != runtimeConfiguration.Observation.ProgramBehaviorFingerprint {
		return errors.New("Evidence Observation program is stale")
	}
	if document.ObservationEvaluationStatus != "accepted" {
		return nil
	}
	dispositions, err := validateEvidenceDispositionsAgainstRaw(document, rawEvidence)
	if err != nil {
		return err
	}
	trace := *document.EvidenceBackedModelTrace
	if trace.ObservationPlan != document.Mapping || trace.MappingDefinitionID != document.Mapping.DefinitionID ||
		trace.MappingBehaviorFingerprint != document.Mapping.BehaviorFingerprint {
		return errors.New("Evidence-backed Model Trace does not match the bound Observation identities")
	}
	return validateEvidenceLinksAgainstRaw(document, rawEvidence, dispositions)
}

func validateEvidenceDispositionsAgainstRaw(
	document Evidence,
	rawEvidence RawEvidence,
) (map[FieldReference]FieldDispositionRecord, error) {
	dispositions := make(map[FieldReference]FieldDispositionRecord, len(document.Dispositions))
	for _, disposition := range document.Dispositions {
		dispositions[disposition.Field] = disposition
		if !rawEvidenceHasField(rawEvidence, disposition.Field) {
			return nil, fmt.Errorf("Evidence disposition field %q is absent from RawEvidence",
				disposition.Field.FieldDefinitionID)
		}
		expectedRawDisposition := map[string]string{
			"retain": "plain",
			"redact": "redacted",
			"hash":   "sha256",
			"reject": "rejected",
		}[disposition.Disposition]
		if rawEvidenceFieldDisposition(rawEvidence, disposition.Field) != expectedRawDisposition {
			return nil, fmt.Errorf("Evidence disposition %q does not match RawEvidence field %q",
				disposition.Disposition, disposition.Field.FieldDefinitionID)
		}
	}
	for _, fact := range rawEvidence.Facts {
		for _, field := range fact.Fields {
			if field.Disposition != "rejected" {
				continue
			}
			reference := FieldReference{
				KindDefinitionID: fact.KindDefinitionID, FieldDefinitionID: field.FieldDefinitionID,
			}
			if dispositions[reference].Disposition != "reject" {
				return nil, fmt.Errorf("rejected RawEvidence field %q has no reject disposition",
					field.FieldDefinitionID)
			}
		}
	}
	return dispositions, nil
}

func validateEvidenceLinksAgainstRaw(
	document Evidence,
	rawEvidence RawEvidence,
	dispositions map[FieldReference]FieldDispositionRecord,
) error {
	facts := make(map[string]RawEvidenceFact, len(rawEvidence.Facts))
	for _, fact := range rawEvidence.Facts {
		facts[fact.FactDefinitionID] = fact
	}
	appliedFields := make(map[FieldReference]struct{}, len(document.Dispositions))
	for _, link := range document.EvidenceLinks {
		for _, support := range link.OrderingSupport {
			fact, ok := facts[support.FactDefinitionID]
			if !ok || support.KindDefinitionID != fact.KindDefinitionID || support.Ordinal != fact.Ordinal ||
				!slices.Equal(support.CausalFactDefinitionIDs, fact.CausalFactDefinitionIDs) {
				return fmt.Errorf("Evidence ordering support %q is stale", support.FactDefinitionID)
			}
		}
		for _, applied := range link.AppliedDispositions {
			declaration, ok := dispositions[applied.Field]
			if !ok {
				return fmt.Errorf("applied field %q has no disposition declaration", applied.Field.FieldDefinitionID)
			}
			appliedFields[applied.Field] = struct{}{}
			switch applied.Kind {
			case "retained":
				if declaration.Disposition != "retain" ||
					rawEvidenceFieldDisposition(rawEvidence, applied.Field) != "plain" {
					return errors.New("retained Evidence attempts to expose prohibited raw field material")
				}
			case "redacted":
				if declaration.Disposition != "redact" ||
					rawEvidenceFieldDisposition(rawEvidence, applied.Field) != "redacted" {
					return errors.New("redacted Evidence disposition does not match RawEvidence")
				}
			case "digest-token":
				if declaration.Disposition != "hash" ||
					declaration.DigestPolicyDefinitionID == nil || applied.DigestPolicyDefinitionID == nil ||
					*declaration.DigestPolicyDefinitionID != *applied.DigestPolicyDefinitionID ||
					rawEvidenceFieldDisposition(rawEvidence, applied.Field) != "sha256" {
					return errors.New("digest Evidence disposition does not match RawEvidence")
				}
			}
		}
	}
	for field, disposition := range dispositions {
		_, applied := appliedFields[field]
		if disposition.Disposition == "reject" {
			if applied {
				return errors.New("rejected Evidence field must not contribute to a Model Fact")
			}
		}
	}
	return nil
}

func rawEvidenceHasField(document RawEvidence, reference FieldReference) bool {
	for _, fact := range document.Facts {
		if fact.KindDefinitionID != reference.KindDefinitionID {
			continue
		}
		for _, field := range fact.Fields {
			if field.FieldDefinitionID == reference.FieldDefinitionID {
				return true
			}
		}
	}
	return false
}

func rawEvidenceFieldDisposition(document RawEvidence, reference FieldReference) string {
	result := ""
	for _, fact := range document.Facts {
		if fact.KindDefinitionID != reference.KindDefinitionID {
			continue
		}
		for _, field := range fact.Fields {
			if field.FieldDefinitionID == reference.FieldDefinitionID {
				if result != "" && result != field.Disposition {
					return "conflict"
				}
				result = field.Disposition
			}
		}
	}
	return result
}

type evaluationOutcomeView struct {
	Plan                     DrivePlan                `json:"plan"`
	EvidenceBackedModelTrace EvidenceBackedModelTrace `json:"evidenceBackedModelTrace"`
	EvidenceLinks            []EvidenceLink           `json:"evidenceLinks"`
	ObservationProgram       DefinitionReference      `json:"observationProgram"`
	Mapping                  DefinitionReference      `json:"mapping"`
	ImplementationLink       ImplementationLinkRecord `json:"implementationLink"`
	QuerySummary             QuerySummary             `json:"querySummary"`
	Properties               []Property               `json:"properties"`
	PropertyVerdicts         []PropertyVerdict        `json:"propertyVerdicts"`
	Limits                   []StagedLimit            `json:"limits"`
}

func ExpectedEvaluationOutcomeChecksum(result Result, evidence Evidence, experiment Experiment) (string, error) {
	if evidence.EvidenceBackedModelTrace == nil {
		return "", errors.New("evaluation outcome requires an Evidence-backed Model Trace")
	}
	view := evaluationOutcomeView{
		Plan:                     experiment.Plan,
		EvidenceBackedModelTrace: *evidence.EvidenceBackedModelTrace,
		EvidenceLinks:            evidence.EvidenceLinks,
		ObservationProgram:       evidence.ObservationProgram,
		Mapping:                  evidence.Mapping,
		ImplementationLink:       result.ImplementationLink,
		QuerySummary:             result.QuerySummary,
		Properties:               experiment.Properties,
		PropertyVerdicts:         result.PropertyVerdicts,
		Limits:                   result.Limits,
	}
	encoded, err := encodeJSONLine(view)
	if err != nil {
		return "", err
	}
	return derive(evaluationOutcomeChecksumDomain, encoded), nil
}

func ValidateResultClosure(
	document Result,
	experiment Experiment,
	runtimeConfiguration RuntimeConfiguration,
	run ExperimentRun,
	rawEvidence RawEvidence,
	evidence Evidence,
) error {
	if err := ValidateEvidenceClosure(evidence, experiment, runtimeConfiguration, run, rawEvidence); err != nil {
		return err
	}
	experimentBinding, err := ExperimentArtifactBinding(experiment)
	if err != nil {
		return err
	}
	if document.Experiment != experimentBinding ||
		document.RuntimeConfiguration != RuntimeConfigurationArtifactBinding(runtimeConfiguration) ||
		document.Run != ExperimentRunArtifactBinding(run) ||
		document.RawEvidence != RawEvidenceArtifactBinding(rawEvidence) ||
		document.Evidence != EvidenceArtifactBinding(evidence) {
		return errors.New("Result input Artifact binding does not match its bound member")
	}
	if document.RunIdentity != run.RunIdentity || document.RunIdentity != evidence.RunIdentity ||
		document.OperationalStatus != run.OperationalStatus ||
		document.ObservationEvaluationStatus != evidence.ObservationEvaluationStatus ||
		document.CleanupStatus != run.Cleanup.Status {
		return errors.New("Result Run, operational, Observation, or cleanup field is stale")
	}
	propertyIDs := make([]string, len(experiment.Properties))
	for index, property := range experiment.Properties {
		propertyIDs[index] = property.DefinitionID
	}
	if !slices.Equal(document.QuerySummary.RequiredPropertyDefinitionIDs, propertyIDs) {
		return errors.New("Result required Properties do not match ExperimentSpec")
	}
	if err := validateQuerySummaryClosure(document.QuerySummary, experiment); err != nil {
		return err
	}
	for _, verdict := range document.PropertyVerdicts {
		propertyIndex := slices.IndexFunc(experiment.Properties, func(property Property) bool {
			return property.DefinitionID == verdict.PropertyDefinitionID
		})
		if propertyIndex < 0 || experiment.Properties[propertyIndex].BehaviorFingerprint !=
			verdict.PropertyBehaviorFingerprint {
			return fmt.Errorf("Result Property verdict %q is stale", verdict.PropertyDefinitionID)
		}
		if verdict.TraceID != nil {
			if evidence.EvidenceBackedModelTrace == nil ||
				*verdict.TraceID != evidence.EvidenceBackedModelTrace.TraceID ||
				verdict.EvidenceLimit == nil ||
				*verdict.EvidenceLimit != evidence.EvidenceBackedModelTrace.AppliedLimit {
				return fmt.Errorf("Result Property verdict %q has stale Evidence trace context",
					verdict.PropertyDefinitionID)
			}
		}
		for _, clause := range verdict.Clauses {
			for _, link := range clause.EvidenceLinks {
				if !slices.ContainsFunc(evidence.EvidenceLinks, func(candidate EvidenceLink) bool {
					return reflect.DeepEqual(candidate, link)
				}) {
					return errors.New("Result clause contains an Evidence Link absent from Evidence")
				}
			}
		}
	}
	if document.EvaluationOutcomeChecksum != nil {
		expected, err := ExpectedEvaluationOutcomeChecksum(document, evidence, experiment)
		if err != nil {
			return err
		}
		if *document.EvaluationOutcomeChecksum != expected {
			return fmt.Errorf("evaluation outcome checksum mismatch: got %q, want %q",
				*document.EvaluationOutcomeChecksum, expected)
		}
	}
	return nil
}

func validateQuerySummaryClosure(summary QuerySummary, experiment Experiment) error {
	propertyByID := make(map[string]Property, len(experiment.Properties))
	counts := make(map[string]int, len(summary.PropertyVerdicts))
	unexpected := make([]string, 0)
	divergent := make([]string, 0)
	wrongQuery := make([]string, 0)
	traceIDs := make([]string, 0, len(summary.PropertyVerdicts))
	for _, property := range experiment.Properties {
		propertyByID[property.DefinitionID] = property
	}
	for _, verdict := range summary.PropertyVerdicts {
		counts[verdict.PropertyDefinitionID]++
		property, found := propertyByID[verdict.PropertyDefinitionID]
		if !found {
			unexpected = append(unexpected, verdict.PropertyDefinitionID)
		} else if property.BehaviorFingerprint != verdict.PropertyBehaviorFingerprint {
			divergent = append(divergent, verdict.PropertyDefinitionID)
		}
		if verdict.QueryDefinitionID != experiment.Plan.QueryDefinitionID ||
			!reflect.DeepEqual(verdict.QueryLimits, experiment.Plan.ExpandedLimits) {
			wrongQuery = append(wrongQuery, verdict.PropertyDefinitionID)
		}
		if verdict.TraceID != nil {
			traceIDs = append(traceIDs, *verdict.TraceID)
		}
	}
	missing := make([]string, 0)
	duplicates := make([]string, 0)
	for _, property := range experiment.Properties {
		switch counts[property.DefinitionID] {
		case 0:
			missing = append(missing, property.DefinitionID)
		case 1:
		default:
			duplicates = append(duplicates, property.DefinitionID)
		}
	}
	for _, values := range [][]string{unexpected, divergent, wrongQuery, traceIDs} {
		slices.Sort(values)
	}
	unexpected = slices.Compact(unexpected)
	divergent = slices.Compact(divergent)
	wrongQuery = slices.Compact(wrongQuery)
	traceIDs = slices.Compact(traceIDs)
	if !slices.Equal(summary.MissingPropertyDefinitionIDs, missing) ||
		!slices.Equal(summary.DuplicatePropertyDefinitionIDs, duplicates) ||
		!slices.Equal(summary.UnexpectedPropertyDefinitionIDs, unexpected) ||
		!slices.Equal(summary.DivergentPropertyDefinitionIDs, divergent) ||
		!slices.Equal(summary.WrongQueryResultDefinitionIDs, wrongQuery) ||
		!slices.Equal(summary.TraceIDs, traceIDs) {
		return errors.New("query summary structural partition is stale")
	}
	return nil
}
