package artifact

import (
	"strings"

	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

var runtimeConfigurationV2Decoder = Decoder[artifactv2.RuntimeConfiguration]{
	Format: artifactv2.RuntimeConfigurationFormat,
	NestedFormats: []NestedFormat{
		{Path: "$.experiment.formatVersion", Format: artifactv2.ExperimentFormat},
	},
	Bounds:             Bounds{StringLimit: runtimeV2StringLimit},
	Validate:           artifactv2.ValidateRuntimeConfiguration,
	Canonical:          artifactv2.CanonicalRuntimeConfigurationBytes,
	ProvenanceChecksum: artifactv2.VerifyRuntimeConfigurationProvenanceChecksum,
	ArtifactChecksum:   artifactv2.VerifyRuntimeConfigurationArtifactChecksum,
}

var experimentRunV2Decoder = Decoder[artifactv2.ExperimentRun]{
	Format: artifactv2.ExperimentRunFormat,
	NestedFormats: []NestedFormat{
		{Path: "$.experiment.formatVersion", Format: artifactv2.ExperimentFormat},
		{Path: "$.runtimeConfiguration.formatVersion", Format: artifactv2.RuntimeConfigurationFormat},
	},
	Bounds:             Bounds{StringLimit: runtimeV2StringLimit},
	Validate:           artifactv2.ValidateExperimentRun,
	Canonical:          artifactv2.CanonicalExperimentRunBytes,
	ProvenanceChecksum: artifactv2.VerifyExperimentRunProvenanceChecksum,
	ArtifactChecksum:   artifactv2.VerifyExperimentRunArtifactChecksum,
}

// DecodeRuntimeConfigurationV2 admits only the canonical persisted v2 RuntimeConfiguration bytes.
func DecodeRuntimeConfigurationV2(encoded []byte) (artifactv2.RuntimeConfiguration, error) {
	return runtimeConfigurationV2Decoder.Decode(encoded)
}

// EncodeRuntimeConfigurationV2 returns the sole canonical persisted v2 RuntimeConfiguration representation.
func EncodeRuntimeConfigurationV2(document artifactv2.RuntimeConfiguration) ([]byte, error) {
	encoded, err := artifactv2.CanonicalRuntimeConfigurationBytes(document)
	if err != nil {
		return nil, wrapAdmission(ErrorMalformedValue, err)
	}
	if _, err := runtimeConfigurationV2Decoder.Decode(encoded); err != nil {
		return nil, err
	}
	return encoded, nil
}

// ValidateRuntimeConfigurationV2Closure checks the configuration's exact ExperimentSpec binding.
func ValidateRuntimeConfigurationV2Closure(
	document artifactv2.RuntimeConfiguration,
	experiment artifactv2.Experiment,
) error {
	if err := artifactv2.ValidateRuntimeConfigurationExperimentClosure(document, experiment); err != nil {
		return wrapAdmission(ErrorClosure, err)
	}
	return nil
}

// DecodeExperimentRunV2 admits only the canonical persisted v2 ExperimentRun bytes.
func DecodeExperimentRunV2(encoded []byte) (artifactv2.ExperimentRun, error) {
	return experimentRunV2Decoder.Decode(encoded)
}

// EncodeExperimentRunV2 returns the sole canonical persisted v2 ExperimentRun representation.
func EncodeExperimentRunV2(document artifactv2.ExperimentRun) ([]byte, error) {
	encoded, err := artifactv2.CanonicalExperimentRunBytes(document)
	if err != nil {
		return nil, wrapAdmission(ErrorMalformedValue, err)
	}
	if _, err := experimentRunV2Decoder.Decode(encoded); err != nil {
		return nil, err
	}
	return encoded, nil
}

// ValidateExperimentRunV2Closure checks the Run's exact ExperimentSpec and RuntimeConfiguration bindings.
func ValidateExperimentRunV2Closure(
	document artifactv2.ExperimentRun,
	experiment artifactv2.Experiment,
	runtimeConfiguration artifactv2.RuntimeConfiguration,
) error {
	if err := artifactv2.ValidateExperimentRunClosure(document, experiment, runtimeConfiguration); err != nil {
		return wrapAdmission(ErrorClosure, err)
	}
	return nil
}

func runtimeV2StringLimit(path JSONPath) int {
	fieldPath := string(path)
	switch {
	case strings.HasSuffix(fieldPath, ".detail"),
		strings.HasSuffix(fieldPath, ".knownGapReason"):
		return MaximumDiagnosticBytes
	case strings.HasSuffix(fieldPath, ".definitionId"),
		strings.HasSuffix(fieldPath, "DefinitionId"),
		strings.HasSuffix(fieldPath, "DefinitionIds[*]"),
		strings.HasSuffix(fieldPath, ".alternatives[*]"),
		strings.HasSuffix(fieldPath, "Fingerprint"),
		strings.HasSuffix(fieldPath, "Checksum"),
		strings.HasSuffix(fieldPath, ".identity"),
		strings.HasSuffix(fieldPath, ".formatVersion"),
		strings.HasSuffix(fieldPath, ".runIdentity"),
		strings.HasSuffix(fieldPath, ".code"),
		strings.HasSuffix(fieldPath, ".subject"):
		return MaximumIdentityBytes
	default:
		return 0
	}
}
