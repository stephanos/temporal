package artifact

import "go.temporal.io/server/tools/umpire/internal/artifactv2"

var experimentV2Decoder = Decoder[artifactv2.Experiment]{
	Format:           artifactv2.ExperimentFormat,
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
	return artifactv2.CanonicalExperimentBytes(document)
}
