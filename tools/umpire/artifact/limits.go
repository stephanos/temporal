package artifact

import "fmt"

const (
	MaximumDocumentBytes            = 32 << 20
	MaximumJSONTokens               = 1_048_576
	MaximumJSONDepth                = 32
	MaximumJSONArrayItems           = 4_096
	MaximumJSONObjectMembers        = 256
	MaximumJSONStringBytes          = 1 << 20
	MaximumArtifactSetMembers       = 6
	MaximumEvidenceSources          = 64
	MaximumEvidenceFacts            = 4_096
	MaximumFieldsPerEvidenceFact    = 128
	MaximumEvidenceFactPayloadBytes = 1 << 20
	MaximumRawEvidencePayloadBytes  = 16 << 20
	MaximumIdentityBytes            = 512
	MaximumDiagnosticBytes          = 4_096
)

type structuralLimits struct {
	documentBytes int
	tokens        int
	depth         int
	arrayItems    int
	objectMembers int
	stringBytes   int
}

var standardStructuralLimits = structuralLimits{
	documentBytes: MaximumDocumentBytes,
	tokens:        MaximumJSONTokens,
	depth:         MaximumJSONDepth,
	arrayItems:    MaximumJSONArrayItems,
	objectMembers: MaximumJSONObjectMembers,
	stringBytes:   MaximumJSONStringBytes,
}

func checkDocumentBytes(encoded []byte, limits structuralLimits) error {
	if exceeds(len(encoded), limits.documentBytes) {
		return wrapAdmission(ErrorByteLimit, fmt.Errorf("document has %d bytes; limit is %d", len(encoded), limits.documentBytes))
	}
	return nil
}

func exceeds(actual, maximum int) bool {
	return actual > maximum
}

func tighterLimit(general, specific int) int {
	if specific <= 0 || specific >= general {
		return general
	}
	return specific
}
