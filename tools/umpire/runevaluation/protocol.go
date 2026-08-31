package runevaluation

import (
	"errors"
	"io"
	"reflect"
	"slices"
	"strings"

	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

const (
	checkerRequestFormat        = "umpire-semantic-check-request/v2"
	checkerResponseFormat       = "umpire-semantic-check-response/v2"
	checkerIdentity             = "temporal.nexus.caller-closure.run-evaluation"
	checkerBehaviorFingerprint  = "sha256:e649a5e059ef42806eb661deb1c1ccba08ec5202425d7a824f7e25026f8134da"
	checkerVersion              = "2"
	maximumCheckerProtocolBytes = 32 << 20
)

type definitionReference struct {
	DefinitionID        string `json:"definitionId"`
	BehaviorFingerprint string `json:"behaviorFingerprint"`
}

type propertyReference struct {
	DefinitionID             string   `json:"definitionId"`
	BehaviorFingerprint      string   `json:"behaviorFingerprint"`
	RequirementDefinitionIDs []string `json:"requirementDefinitionIds"`
}

type checkerRequest struct {
	FormatVersion              string                         `json:"formatVersion"`
	CheckerIdentity            string                         `json:"checkerIdentity"`
	CheckerVersion             artifactv2.Natural             `json:"checkerVersion"`
	CheckerBehaviorFingerprint string                         `json:"checkerBehaviorFingerprint"`
	Experiment                 artifactv2.ArtifactBinding     `json:"experiment"`
	RuntimeConfiguration       artifactv2.ArtifactBinding     `json:"runtimeConfiguration"`
	Run                        artifactv2.ArtifactBinding     `json:"run"`
	RawEvidence                artifactv2.ArtifactBinding     `json:"rawEvidence"`
	RunIdentity                string                         `json:"runIdentity"`
	Query                      definitionReference            `json:"query"`
	Properties                 []propertyReference            `json:"properties"`
	ObservationProgram         definitionReference            `json:"observationProgram"`
	Mapping                    definitionReference            `json:"mapping"`
	PhaseOutcomes              []artifactv2.PhaseOutcome      `json:"phaseOutcomes"`
	ControlAttempts            []artifactv2.ControlAttempt    `json:"controlAttempts"`
	SourceClosures             []artifactv2.SourceClosure     `json:"sourceClosures"`
	CaptureStatus              string                         `json:"captureStatus"`
	Sources                    []artifactv2.RawEvidenceSource `json:"sources"`
	Facts                      []artifactv2.RawEvidenceFact   `json:"facts"`
	RunKnownGaps               []artifactv2.KnownGap          `json:"runKnownGaps"`
	RawEvidenceKnownGaps       []artifactv2.KnownGap          `json:"rawEvidenceKnownGaps"`
}

type checkerResponse struct {
	FormatVersion                           string                               `json:"formatVersion"`
	CheckerIdentity                         string                               `json:"checkerIdentity"`
	CheckerVersion                          artifactv2.Natural                   `json:"checkerVersion"`
	CheckerBehaviorFingerprint              string                               `json:"checkerBehaviorFingerprint"`
	ExperimentArtifactChecksum              string                               `json:"experimentArtifactChecksum"`
	RuntimeConfigurationArtifactChecksum    string                               `json:"runtimeConfigurationArtifactChecksum"`
	RunArtifactChecksum                     string                               `json:"runArtifactChecksum"`
	RawEvidenceArtifactChecksum             string                               `json:"rawEvidenceArtifactChecksum"`
	ExperimentBehaviorFingerprint           string                               `json:"experimentBehaviorFingerprint"`
	RuntimeConfigurationBehaviorFingerprint string                               `json:"runtimeConfigurationBehaviorFingerprint"`
	RunIdentity                             string                               `json:"runIdentity"`
	ObservationEvaluationStatus             string                               `json:"observationEvaluationStatus"`
	ImplementationLink                      artifactv2.ImplementationLinkRecord  `json:"implementationLink"`
	ImplementationLinkStatus                string                               `json:"implementationLinkStatus"`
	EvidenceBackedModelTrace                *artifactv2.EvidenceBackedModelTrace `json:"evidenceBackedModelTrace"`
	EvidenceLinks                           []artifactv2.EvidenceLink            `json:"evidenceLinks"`
	Dispositions                            []artifactv2.FieldDispositionRecord  `json:"dispositions"`
	Diagnostics                             []artifactv2.ObservationDiagnostic   `json:"diagnostics"`
	ObservationKnownGaps                    []artifactv2.KnownGap                `json:"observationKnownGaps"`
	PropertyVerdicts                        []artifactv2.PropertyVerdict         `json:"propertyVerdicts"`
	QuerySummary                            artifactv2.QuerySummary              `json:"querySummary"`
	SemanticStatus                          string                               `json:"semanticStatus"`
	ResultKnownGaps                         []artifactv2.KnownGap                `json:"resultKnownGaps"`
	EvaluationOutcomeChecksum               *string                              `json:"evaluationOutcomeChecksum"`
}

var checkerResponseDecoder = artifact.Decoder[checkerResponse]{
	Format:    checkerResponseFormat,
	Validate:  validateCheckerResponse,
	Canonical: canonicalCheckerResponse,
}

func canonicalCheckerResponse(response checkerResponse) ([]byte, error) {
	encoded := newBoundedCapture(maximumCheckerProtocolBytes, nil)
	if err := writeCanonicalPrettyJSON(encoded, response); err != nil {
		return nil, err
	}
	if encoded.exceeded() {
		return nil, errors.New("checker response is oversized")
	}
	return encoded.take(), nil
}

func encodeCheckerRequest(request checkerRequest) ([]byte, error) {
	if err := validateCheckerRequest(request); err != nil {
		return nil, err
	}
	encoded := newBoundedCapture(maximumCheckerProtocolBytes, nil)
	if err := writeCanonicalCheckerRequest(request, encoded); err != nil {
		return nil, errors.New("encode checker request")
	}
	if encoded.exceeded() {
		return nil, errors.New("checker request is oversized")
	}
	return encoded.take(), nil
}

func writeCanonicalCheckerRequest(request checkerRequest, writer io.Writer) error {
	return writeCanonicalPrettyJSON(writer, request)
}

func decodeCheckerResponse(encoded []byte, request checkerRequest) (checkerResponse, error) {
	response, err := checkerResponseDecoder.Decode(encoded)
	if err != nil {
		return checkerResponse{}, err
	}
	if err := validateCheckerResponseForRequest(response, request); err != nil {
		return checkerResponse{}, err
	}
	return response, nil
}

func validateCheckerResponseProjection(response checkerResponse, request checkerRequest) error {
	if err := artifactv2.ValidateEvidence(projectCheckerEvidence(response, request)); err != nil {
		return errors.New("checker response Evidence projection is invalid")
	}
	if err := artifactv2.ValidateResult(projectCheckerResult(response, request)); err != nil {
		return errors.New("checker response Result projection is invalid")
	}
	if err := validateCheckerSemanticBindings(response, request); err != nil {
		return err
	}
	if err := validateCheckerQueryPartition(response, request); err != nil {
		return err
	}
	return validateCheckerKnownGapUnion(response, request)
}

func projectCheckerEvidence(response checkerResponse, request checkerRequest) artifactv2.Evidence {
	return artifactv2.Evidence{
		FormatVersion:               artifactv2.EvidenceFormat,
		RunIdentity:                 response.RunIdentity,
		BehaviorFingerprint:         checkerBehaviorFingerprint,
		Experiment:                  request.Experiment,
		RuntimeConfiguration:        request.RuntimeConfiguration,
		Run:                         request.Run,
		RawEvidence:                 request.RawEvidence,
		ObservationProgram:          artifactDefinitionReference(request.ObservationProgram),
		Mapping:                     artifactDefinitionReference(request.Mapping),
		ObservationEvaluationStatus: response.ObservationEvaluationStatus,
		EvidenceBackedModelTrace:    response.EvidenceBackedModelTrace,
		EvidenceLinks:               response.EvidenceLinks,
		Dispositions:                response.Dispositions,
		Diagnostics:                 response.Diagnostics,
		KnownGaps:                   response.ObservationKnownGaps,
		Provenance:                  checkerValidationProvenance(),
		ProvenanceChecksum:          request.RawEvidence.ProvenanceChecksum,
		ArtifactChecksum:            request.RawEvidence.ArtifactChecksum,
	}
}

func projectCheckerResult(response checkerResponse, request checkerRequest) artifactv2.Result {
	return artifactv2.Result{
		FormatVersion:               artifactv2.ResultFormat,
		RunIdentity:                 response.RunIdentity,
		BehaviorFingerprint:         checkerBehaviorFingerprint,
		Experiment:                  request.Experiment,
		RuntimeConfiguration:        request.RuntimeConfiguration,
		Run:                         request.Run,
		RawEvidence:                 request.RawEvidence,
		Evidence:                    checkerEvidenceBinding(request),
		OperationalStatus:           "succeeded",
		ObservationEvaluationStatus: response.ObservationEvaluationStatus,
		ImplementationLink:          response.ImplementationLink,
		ImplementationLinkStatus:    response.ImplementationLinkStatus,
		PropertyVerdicts:            response.PropertyVerdicts,
		QuerySummary:                response.QuerySummary,
		SemanticStatus:              response.SemanticStatus,
		Limits:                      []artifactv2.StagedLimit{},
		KnownGaps:                   response.ResultKnownGaps,
		CleanupStatus:               "complete",
		EvaluationOutcomeChecksum:   response.EvaluationOutcomeChecksum,
		Provenance:                  checkerValidationProvenance(),
		ProvenanceChecksum:          request.Run.ProvenanceChecksum,
		ArtifactChecksum:            request.Run.ArtifactChecksum,
	}
}

func artifactDefinitionReference(reference definitionReference) artifactv2.DefinitionReference {
	return artifactv2.DefinitionReference{
		DefinitionID:        reference.DefinitionID,
		BehaviorFingerprint: reference.BehaviorFingerprint,
	}
}

func checkerEvidenceBinding(request checkerRequest) artifactv2.ArtifactBinding {
	return artifactv2.ArtifactBinding{
		FormatVersion:       artifactv2.EvidenceFormat,
		ArtifactChecksum:    request.Run.ArtifactChecksum,
		BehaviorFingerprint: checkerBehaviorFingerprint,
		ProvenanceChecksum:  request.Run.ProvenanceChecksum,
	}
}

func checkerValidationProvenance() artifactv2.Provenance {
	one := artifactv2.NaturalFromUint64(1)
	return artifactv2.Provenance{
		SourceDefinitionIDs: []string{},
		SourceLocations: []artifactv2.SourceLocation{{
			Path: "checker-response", Line: one, Column: one, Provenance: "generated",
		}},
	}
}

func validateCheckerSemanticBindings(response checkerResponse, request checkerRequest) error {
	implementationLink := response.ImplementationLink
	implementationLink.Diagnostic = nil
	if !reflect.DeepEqual(implementationLink, callerClosureImplementationLink()) {
		return errors.New("checker response Implementation Link binding drifted")
	}
	if response.QuerySummary.QueryDefinitionID != request.Query.DefinitionID {
		return errors.New("checker response query binding drifted")
	}
	if trace := response.EvidenceBackedModelTrace; trace != nil {
		if trace.ObservationPlan != artifactDefinitionReference(request.Mapping) ||
			trace.MappingDefinitionID != request.Mapping.DefinitionID ||
			trace.MappingBehaviorFingerprint != request.Mapping.BehaviorFingerprint {
			return errors.New("checker response Observation binding drifted")
		}
		profileVersion := artifactv2.NaturalFromUint64(1)
		if trace.ProfileDefinitionID != callerClosureCheckedProfileID ||
			trace.ProfileVersion != profileVersion {
			return errors.New("checker response profile binding drifted")
		}
		for _, link := range response.EvidenceLinks {
			if link.ProfileDefinitionID != callerClosureCheckedProfileID ||
				link.ProfileVersion != profileVersion {
				return errors.New("checker response profile binding drifted")
			}
		}
	}
	for _, verdict := range response.PropertyVerdicts {
		if err := validateCheckerPropertyBinding(verdict, response, request); err != nil {
			return err
		}
	}
	return nil
}

func validateCheckerPropertyBinding(
	verdict artifactv2.PropertyVerdict,
	response checkerResponse,
	request checkerRequest,
) error {
	propertyIndex := slices.IndexFunc(request.Properties, func(property propertyReference) bool {
		return property.DefinitionID == verdict.PropertyDefinitionID
	})
	if propertyIndex < 0 || request.Properties[propertyIndex].BehaviorFingerprint !=
		verdict.PropertyBehaviorFingerprint || verdict.QueryDefinitionID != request.Query.DefinitionID {
		return errors.New("checker response Property binding drifted")
	}
	if verdict.TraceID != nil &&
		(response.EvidenceBackedModelTrace == nil ||
			*verdict.TraceID != response.EvidenceBackedModelTrace.TraceID ||
			verdict.EvidenceLimit == nil ||
			*verdict.EvidenceLimit != response.EvidenceBackedModelTrace.AppliedLimit) {
		return errors.New("checker response Property trace binding drifted")
	}
	for _, clause := range verdict.Clauses {
		for _, link := range clause.EvidenceLinks {
			if !slices.ContainsFunc(response.EvidenceLinks, func(candidate artifactv2.EvidenceLink) bool {
				return reflect.DeepEqual(candidate, link)
			}) {
				return errors.New("checker response Property Evidence Link drifted")
			}
		}
	}
	return nil
}

func validateCheckerQueryPartition(response checkerResponse, request checkerRequest) error {
	required := make([]string, len(request.Properties))
	properties := make(map[string]propertyReference, len(request.Properties))
	for index, property := range request.Properties {
		required[index] = property.DefinitionID
		properties[property.DefinitionID] = property
	}
	if !slices.Equal(response.QuerySummary.RequiredPropertyDefinitionIDs, required) {
		return errors.New("checker response required Property partition drifted")
	}

	counts := make(map[string]int, len(response.PropertyVerdicts))
	unexpected := make([]string, 0)
	divergent := make([]string, 0)
	wrongQuery := make([]string, 0)
	traceIDs := make([]string, 0, len(response.PropertyVerdicts))
	for _, verdict := range response.PropertyVerdicts {
		counts[verdict.PropertyDefinitionID]++
		property, found := properties[verdict.PropertyDefinitionID]
		if !found {
			unexpected = append(unexpected, verdict.PropertyDefinitionID)
		} else if property.BehaviorFingerprint != verdict.PropertyBehaviorFingerprint {
			divergent = append(divergent, verdict.PropertyDefinitionID)
		}
		if verdict.QueryDefinitionID != request.Query.DefinitionID ||
			!reflect.DeepEqual(verdict.QueryLimits, response.QuerySummary.QueryLimits) {
			wrongQuery = append(wrongQuery, verdict.PropertyDefinitionID)
		}
		if verdict.TraceID != nil {
			traceIDs = append(traceIDs, *verdict.TraceID)
		}
	}
	return validateCheckerQueryPartitionValues(response.QuerySummary, required, counts,
		unexpected, divergent, wrongQuery, traceIDs)
}

func validateCheckerQueryPartitionValues(
	summary artifactv2.QuerySummary,
	required []string,
	counts map[string]int,
	unexpected []string,
	divergent []string,
	wrongQuery []string,
	traceIDs []string,
) error {
	missing := make([]string, 0)
	duplicates := make([]string, 0)
	for _, propertyID := range required {
		switch counts[propertyID] {
		case 0:
			missing = append(missing, propertyID)
		case 1:
		default:
			duplicates = append(duplicates, propertyID)
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
		return errors.New("checker response query structural partition drifted")
	}
	return nil
}

func validateCheckerKnownGapUnion(response checkerResponse, request checkerRequest) error {
	expected := make([]artifactv2.KnownGap, 0,
		len(request.RunKnownGaps)+len(request.RawEvidenceKnownGaps)+len(response.ObservationKnownGaps))
	expected = append(expected, request.RunKnownGaps...)
	expected = append(expected, request.RawEvidenceKnownGaps...)
	expected = append(expected, response.ObservationKnownGaps...)
	slices.SortFunc(expected, compareCheckerKnownGap)
	expected = slices.CompactFunc(expected, func(left, right artifactv2.KnownGap) bool {
		return compareCheckerKnownGap(left, right) == 0
	})
	if !reflect.DeepEqual(expected, response.ResultKnownGaps) {
		return errors.New("checker response Known Gap union drifted")
	}
	return nil
}

func compareCheckerKnownGap(left, right artifactv2.KnownGap) int {
	for _, comparison := range []int{
		checkerCompareInt(checkerKnownGapKindRank(left.Kind), checkerKnownGapKindRank(right.Kind)),
		strings.Compare(left.Code, right.Code),
		strings.Compare(checkerPointerValue(left.Subject), checkerPointerValue(right.Subject)),
		strings.Compare(checkerPointerValue(left.Detail), checkerPointerValue(right.Detail)),
	} {
		if comparison != 0 {
			return comparison
		}
	}
	return 0
}

func checkerKnownGapKindRank(kind string) int {
	switch kind {
	case "capability-contract":
		return 0
	case "input":
		return 1
	case "interpretation":
		return 2
	case "claim":
		return 3
	default:
		return 4
	}
}

func checkerCompareInt(left, right int) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func checkerPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func validateCheckerResponse(response checkerResponse) error {
	if response.FormatVersion != checkerResponseFormat || response.CheckerIdentity != checkerIdentity ||
		response.CheckerVersion != checkerVersion ||
		response.CheckerBehaviorFingerprint != checkerBehaviorFingerprint {
		return errors.New("checker response handshake is invalid")
	}
	if err := validateCheckerResponseBindings(response); err != nil {
		return err
	}
	if err := validateCheckerResponseSemanticValues(response); err != nil {
		return err
	}
	return validateCheckerResponseCollections(response)
}

func validateCheckerResponseBindings(response checkerResponse) error {
	for _, digest := range []string{
		response.ExperimentArtifactChecksum,
		response.RuntimeConfigurationArtifactChecksum,
		response.RunArtifactChecksum,
		response.RawEvidenceArtifactChecksum,
		response.ExperimentBehaviorFingerprint,
		response.RuntimeConfigurationBehaviorFingerprint,
	} {
		if !artifactv2.ValidDigest(digest) {
			return errors.New("checker response digest is invalid")
		}
	}
	if response.EvaluationOutcomeChecksum != nil &&
		!artifactv2.ValidDigest(*response.EvaluationOutcomeChecksum) {
		return errors.New("checker response outcome checksum is invalid")
	}
	if !validDefinitionID(response.RunIdentity) {
		return errors.New("checker response Run identity is invalid")
	}
	return nil
}

func validateCheckerResponseSemanticValues(response checkerResponse) error {
	if !oneOf(response.ObservationEvaluationStatus, "accepted", "unknown", "conflict", "unsupported") ||
		!oneOf(response.ImplementationLinkStatus,
			"applied", "not-evaluated", "invalid", "unknown", "conflict", "unsupported") ||
		!oneOf(response.SemanticStatus, "satisfied", "violated", "incomplete") ||
		!oneOf(response.QuerySummary.Status, "satisfied", "violated", "incomplete") {
		return errors.New("checker response status is invalid")
	}
	for _, verdict := range response.PropertyVerdicts {
		if !oneOf(verdict.Status, "satisfied", "violated", "unknown", "conflict", "unsupported") {
			return errors.New("checker response Property status is invalid")
		}
	}
	return nil
}

func validateCheckerResponseCollections(response checkerResponse) error {
	if response.EvidenceLinks == nil || response.Dispositions == nil || response.Diagnostics == nil ||
		response.ObservationKnownGaps == nil || response.PropertyVerdicts == nil ||
		response.ResultKnownGaps == nil || response.QuerySummary.RequiredPropertyDefinitionIDs == nil ||
		response.QuerySummary.PropertyVerdicts == nil ||
		response.QuerySummary.MissingPropertyDefinitionIDs == nil ||
		response.QuerySummary.DuplicatePropertyDefinitionIDs == nil ||
		response.QuerySummary.UnexpectedPropertyDefinitionIDs == nil ||
		response.QuerySummary.DivergentPropertyDefinitionIDs == nil ||
		response.QuerySummary.WrongQueryResultDefinitionIDs == nil ||
		response.QuerySummary.TraceIDs == nil {
		return errors.New("checker response collection is null")
	}
	return nil
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func validateCheckerRequest(request checkerRequest) error {
	if request.FormatVersion != checkerRequestFormat || request.CheckerIdentity != checkerIdentity ||
		request.CheckerVersion != checkerVersion ||
		request.CheckerBehaviorFingerprint != checkerBehaviorFingerprint {
		return errors.New("checker request handshake is invalid")
	}
	if err := validateCheckerRequestBindings(request); err != nil {
		return err
	}
	if err := validateCheckerRequestSemantics(request); err != nil {
		return err
	}
	if request.CaptureStatus != "closed" && request.CaptureStatus != "partial" &&
		request.CaptureStatus != "failed" {
		return errors.New("checker request capture status is invalid")
	}
	return nil
}

func validateCheckerRequestBindings(request checkerRequest) error {
	for _, binding := range []struct {
		value  artifactv2.ArtifactBinding
		format string
	}{
		{value: request.Experiment, format: artifactv2.ExperimentFormat},
		{value: request.RuntimeConfiguration, format: artifactv2.RuntimeConfigurationFormat},
		{value: request.Run, format: artifactv2.ExperimentRunFormat},
		{value: request.RawEvidence, format: artifactv2.RawEvidenceFormat},
	} {
		if binding.value.FormatVersion != binding.format ||
			!artifactv2.ValidDigest(binding.value.ArtifactChecksum) ||
			!artifactv2.ValidDigest(binding.value.BehaviorFingerprint) ||
			!artifactv2.ValidDigest(binding.value.ProvenanceChecksum) {
			return errors.New("checker request artifact binding is invalid")
		}
	}
	return nil
}

func validateCheckerRequestSemantics(request checkerRequest) error {
	if !validDefinitionID(request.RunIdentity) || !validReference(request.Query) ||
		!validReference(request.ObservationProgram) || !validReference(request.Mapping) {
		return errors.New("checker request semantic binding is invalid")
	}
	if request.Properties == nil || request.PhaseOutcomes == nil || request.ControlAttempts == nil ||
		request.SourceClosures == nil || request.Sources == nil || request.Facts == nil ||
		request.RunKnownGaps == nil || request.RawEvidenceKnownGaps == nil {
		return errors.New("checker request collection is null")
	}
	for _, property := range request.Properties {
		if !validDefinitionID(property.DefinitionID) ||
			!artifactv2.ValidDigest(property.BehaviorFingerprint) ||
			property.RequirementDefinitionIDs == nil {
			return errors.New("checker request property binding is invalid")
		}
		for _, definitionID := range property.RequirementDefinitionIDs {
			if !validDefinitionID(definitionID) {
				return errors.New("checker request requirement binding is invalid")
			}
		}
	}
	return nil
}

func validReference(reference definitionReference) bool {
	return validDefinitionID(reference.DefinitionID) &&
		artifactv2.ValidDigest(reference.BehaviorFingerprint)
}

func validDefinitionID(value string) bool {
	segments := strings.Split(value, ".")
	if len(segments) < 2 {
		return false
	}
	for _, segment := range segments {
		if segment == "" {
			return false
		}
		for _, character := range []byte(segment) {
			if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' ||
				character >= '0' && character <= '9' || character == '-' || character == '_' {
				continue
			}
			return false
		}
	}
	return true
}
