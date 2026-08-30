package runevaluation

import (
	"errors"
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
	Format:   checkerResponseFormat,
	Validate: validateCheckerResponse,
}

func encodeCheckerRequest(request checkerRequest) ([]byte, error) {
	if err := validateCheckerRequest(request); err != nil {
		return nil, err
	}
	encoded, err := artifact.CanonicalPretty(request)
	if err != nil {
		return nil, errors.New("encode checker request")
	}
	if len(encoded) > maximumCheckerProtocolBytes {
		return nil, errors.New("checker request is oversized")
	}
	return encoded, nil
}

func decodeCheckerResponse(encoded []byte, request checkerRequest) (checkerResponse, error) {
	response, err := checkerResponseDecoder.Decode(encoded)
	if err != nil {
		return checkerResponse{}, err
	}
	if response.CheckerIdentity != request.CheckerIdentity ||
		response.CheckerVersion != request.CheckerVersion ||
		response.CheckerBehaviorFingerprint != request.CheckerBehaviorFingerprint {
		return checkerResponse{}, errors.New("checker response handshake drifted")
	}
	if response.ExperimentArtifactChecksum != request.Experiment.ArtifactChecksum ||
		response.RuntimeConfigurationArtifactChecksum != request.RuntimeConfiguration.ArtifactChecksum ||
		response.RunArtifactChecksum != request.Run.ArtifactChecksum ||
		response.RawEvidenceArtifactChecksum != request.RawEvidence.ArtifactChecksum ||
		response.ExperimentBehaviorFingerprint != request.Experiment.BehaviorFingerprint ||
		response.RuntimeConfigurationBehaviorFingerprint != request.RuntimeConfiguration.BehaviorFingerprint ||
		response.RunIdentity != request.RunIdentity {
		return checkerResponse{}, errors.New("checker response binding drifted")
	}
	return response, nil
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
