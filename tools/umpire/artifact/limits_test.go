package artifact

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStrictJSONEveryLimitHasExactNNPlusOneBehavior(t *testing.T) {
	limits := map[string]int{
		"document bytes":        MaximumDocumentBytes,
		"tokens":                MaximumJSONTokens,
		"depth":                 MaximumJSONDepth,
		"array items":           MaximumJSONArrayItems,
		"object members":        MaximumJSONObjectMembers,
		"string bytes":          MaximumJSONStringBytes,
		"set members":           MaximumArtifactSetMembers,
		"evidence sources":      MaximumEvidenceSources,
		"evidence facts":        MaximumEvidenceFacts,
		"fields per fact":       MaximumFieldsPerEvidenceFact,
		"payload per fact":      MaximumEvidenceFactPayloadBytes,
		"aggregate raw payload": MaximumRawEvidencePayloadBytes,
		"identity bytes":        MaximumIdentityBytes,
		"diagnostic bytes":      MaximumDiagnosticBytes,
	}
	for name, limit := range limits {
		t.Run(name, func(t *testing.T) {
			require.False(t, exceeds(limit, limit))
			require.True(t, exceeds(limit+1, limit))
		})
	}
}

func TestStrictJSONDocumentBytesIncludeTerminalLF(t *testing.T) {
	encoded := make([]byte, MaximumDocumentBytes)
	encoded[len(encoded)-1] = '\n'
	require.NoError(t, checkDocumentBytes(encoded, standardStructuralLimits))

	encoded = append(encoded, 'x')
	requireErrorCode(t, checkDocumentBytes(encoded, standardStructuralLimits), ErrorByteLimit)
}
