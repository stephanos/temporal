package artifact

import "go.temporal.io/server/tools/umpire/internal/artifactv2"

var evidenceV2Decoder = Decoder[artifactv2.Evidence]{
	Format: artifactv2.EvidenceFormat,
	NestedFormats: []NestedFormat{
		{Path: "$.experiment.formatVersion", Format: artifactv2.ExperimentFormat},
		{Path: "$.runtimeConfiguration.formatVersion", Format: artifactv2.RuntimeConfigurationFormat},
		{Path: "$.run.formatVersion", Format: artifactv2.ExperimentRunFormat},
		{Path: "$.rawEvidence.formatVersion", Format: artifactv2.RawEvidenceFormat},
	},
	Bounds:             Bounds{StringLimit: runtimeV2StringLimit},
	Validate:           artifactv2.ValidateEvidence,
	Canonical:          artifactv2.CanonicalEvidenceBytes,
	ProvenanceChecksum: artifactv2.VerifyEvidenceProvenanceChecksum,
	ArtifactChecksum:   artifactv2.VerifyEvidenceArtifactChecksum,
}

var resultV2Decoder = Decoder[artifactv2.Result]{
	Format: artifactv2.ResultFormat,
	NestedFormats: []NestedFormat{
		{Path: "$.experiment.formatVersion", Format: artifactv2.ExperimentFormat},
		{Path: "$.runtimeConfiguration.formatVersion", Format: artifactv2.RuntimeConfigurationFormat},
		{Path: "$.run.formatVersion", Format: artifactv2.ExperimentRunFormat},
		{Path: "$.rawEvidence.formatVersion", Format: artifactv2.RawEvidenceFormat},
		{Path: "$.evidence.formatVersion", Format: artifactv2.EvidenceFormat},
	},
	Bounds:             Bounds{StringLimit: runtimeV2StringLimit},
	Validate:           artifactv2.ValidateResult,
	Canonical:          artifactv2.CanonicalResultBytes,
	ProvenanceChecksum: artifactv2.VerifyResultProvenanceChecksum,
	ArtifactChecksum:   artifactv2.VerifyResultArtifactChecksum,
}

// DecodeEvidenceV2 admits only the canonical persisted bounded v2 Evidence bytes.
func DecodeEvidenceV2(encoded []byte) (artifactv2.Evidence, error) {
	return evidenceV2Decoder.Decode(encoded)
}

// EncodeEvidenceV2 returns the sole canonical persisted v2 Evidence representation.
func EncodeEvidenceV2(document artifactv2.Evidence) ([]byte, error) {
	encoded, err := artifactv2.CanonicalEvidenceBytes(document)
	if err != nil {
		return nil, wrapAdmission(ErrorMalformedValue, err)
	}
	if _, err := evidenceV2Decoder.Decode(encoded); err != nil {
		return nil, err
	}
	return encoded, nil
}

// ValidateEvidenceV2Closure checks exact parent bindings and RawEvidence-backed transport closure.
func ValidateEvidenceV2Closure(
	document artifactv2.Evidence,
	experiment artifactv2.Experiment,
	runtimeConfiguration artifactv2.RuntimeConfiguration,
	run artifactv2.ExperimentRun,
	rawEvidence artifactv2.RawEvidence,
) error {
	if err := artifactv2.ValidateEvidenceClosure(
		document,
		experiment,
		runtimeConfiguration,
		run,
		rawEvidence,
	); err != nil {
		return wrapAdmission(ErrorClosure, err)
	}
	return nil
}

// DecodeResultV2 admits only the canonical persisted bounded v2 Result bytes.
func DecodeResultV2(encoded []byte) (artifactv2.Result, error) {
	return resultV2Decoder.Decode(encoded)
}

// EncodeResultV2 returns the sole canonical persisted v2 Result representation.
func EncodeResultV2(document artifactv2.Result) ([]byte, error) {
	encoded, err := artifactv2.CanonicalResultBytes(document)
	if err != nil {
		return nil, wrapAdmission(ErrorMalformedValue, err)
	}
	if _, err := resultV2Decoder.Decode(encoded); err != nil {
		return nil, err
	}
	return encoded, nil
}

// ValidateResultV2Closure checks exact parent bindings and complete evaluation closure.
func ValidateResultV2Closure(
	document artifactv2.Result,
	experiment artifactv2.Experiment,
	runtimeConfiguration artifactv2.RuntimeConfiguration,
	run artifactv2.ExperimentRun,
	rawEvidence artifactv2.RawEvidence,
	evidence artifactv2.Evidence,
) error {
	if err := artifactv2.ValidateResultClosure(
		document,
		experiment,
		runtimeConfiguration,
		run,
		rawEvidence,
		evidence,
	); err != nil {
		return wrapAdmission(ErrorClosure, err)
	}
	return nil
}
