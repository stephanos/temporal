package artifact

import (
	"strings"

	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

var experimentV2Decoder = Decoder[artifactv2.Experiment]{
	Format: artifactv2.ExperimentFormat,
	NestedFormats: []NestedFormat{
		{Path: "$.plan.formatVersion", Format: artifactv2.DrivePlanFormat},
	},
	Bounds:           Bounds{StringLimit: experimentV2StringLimit},
	Validate:         artifactv2.ValidateExperiment,
	Canonical:        artifactv2.CanonicalExperimentBytes,
	ArtifactChecksum: artifactv2.VerifyExperimentChecksums,
	Closure:          artifactv2.ValidateExperimentClosure,
}

// DecodeExperimentV2 admits only the canonical persisted v2 ExperimentSpec bytes.
func DecodeExperimentV2(encoded []byte) (artifactv2.Experiment, error) {
	return experimentV2Decoder.Decode(encoded)
}

// EncodeExperimentV2 returns the sole canonical persisted v2 ExperimentSpec representation.
func EncodeExperimentV2(document artifactv2.Experiment) ([]byte, error) {
	encoded, err := artifactv2.CanonicalExperimentBytes(document)
	if err != nil {
		return nil, wrapAdmission(ErrorMalformedValue, err)
	}
	if _, err := experimentV2Decoder.Decode(encoded); err != nil {
		return nil, err
	}
	return encoded, nil
}

func experimentV2StringLimit(path JSONPath) int {
	fieldPath := string(path)
	switch {
	case strings.HasSuffix(fieldPath, ".detail"):
		return MaximumDiagnosticBytes
	case strings.HasSuffix(fieldPath, "DefinitionId"),
		strings.HasSuffix(fieldPath, "DefinitionIds[*]"),
		strings.HasSuffix(fieldPath, "Fingerprint"),
		strings.HasSuffix(fieldPath, "Checksum"),
		strings.HasSuffix(fieldPath, ".formatVersion"),
		strings.HasSuffix(fieldPath, ".code"),
		strings.HasSuffix(fieldPath, ".subject"):
		return MaximumIdentityBytes
	default:
		return 0
	}
}
