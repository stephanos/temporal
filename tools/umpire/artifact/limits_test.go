package artifact

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type arrayBoundaryProbe struct {
	FormatVersion string   `json:"formatVersion"`
	Values        []string `json:"values"`
}

type objectBoundaryProbe struct {
	FormatVersion string            `json:"formatVersion"`
	Values        map[string]string `json:"values"`
}

type stringBoundaryProbe struct {
	FormatVersion string `json:"formatVersion"`
	Value         string `json:"value"`
}

type depthBoundaryProbe struct {
	FormatVersion string `json:"formatVersion"`
	Value         any    `json:"value"`
}

type tokenBoundaryProbe struct {
	FormatVersion string  `json:"formatVersion"`
	Values        [][]int `json:"values"`
}

func TestStrictJSONEveryDeclaredCeilingHasItsExactValue(t *testing.T) {
	limits := map[string]struct {
		got  int
		want int
	}{
		"document bytes":        {MaximumDocumentBytes, 32 << 20},
		"tokens":                {MaximumJSONTokens, 1_048_576},
		"depth":                 {MaximumJSONDepth, 32},
		"array items":           {MaximumJSONArrayItems, 4_096},
		"object members":        {MaximumJSONObjectMembers, 256},
		"string bytes":          {MaximumJSONStringBytes, 1 << 20},
		"set members":           {MaximumArtifactSetMembers, 6},
		"evidence sources":      {MaximumEvidenceSources, 64},
		"evidence facts":        {MaximumEvidenceFacts, 4_096},
		"fields per fact":       {MaximumFieldsPerEvidenceFact, 128},
		"payload per fact":      {MaximumEvidenceFactPayloadBytes, 1 << 20},
		"aggregate raw payload": {MaximumRawEvidencePayloadBytes, 16 << 20},
		"identity bytes":        {MaximumIdentityBytes, 512},
		"diagnostic bytes":      {MaximumDiagnosticBytes, 4_096},
	}
	for name, limit := range limits {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, limit.want, limit.got)
		})
	}
}

func TestStrictJSONStructuralCeilingsUseEncodedBoundaries(t *testing.T) {
	t.Run("document bytes N and N+1", func(t *testing.T) {
		encoded := canonicalDocumentOfSize(t, MaximumDocumentBytes)
		require.NoError(t, checkDocumentBytes(encoded, standardStructuralLimits))
		requireErrorCode(t, checkDocumentBytes(append(encoded, ' '), standardStructuralLimits), ErrorByteLimit)
	})

	t.Run("token ceiling brackets the only representable boundary", func(t *testing.T) {
		atNMinusOne := canonicalTokenDocument(t, 3_963)
		atNPlusOne := canonicalTokenDocument(t, 3_964)
		metrics, err := measureJSON(atNMinusOne)
		require.NoError(t, err)
		require.Equal(t, MaximumJSONTokens-1, metrics.tokens)
		metrics, err = measureJSON(atNPlusOne)
		require.NoError(t, err)
		require.Equal(t, MaximumJSONTokens+1, metrics.tokens)

		decoder := Decoder[tokenBoundaryProbe]{Format: "umpire-token-boundary/v2"}
		_, err = decoder.Decode(atNMinusOne)
		require.NoError(t, err)
		_, err = decoder.Decode(atNPlusOne)
		requireErrorCode(t, err, ErrorTokenLimit)
	})

	t.Run("depth N and N+1", func(t *testing.T) {
		decoder := Decoder[depthBoundaryProbe]{Format: "umpire-depth-boundary/v2"}
		atN := canonicalDepthDocument(t, MaximumJSONDepth-2)
		metrics, err := measureJSON(atN)
		require.NoError(t, err)
		require.Equal(t, MaximumJSONDepth, metrics.depth)
		_, err = decoder.Decode(atN)
		require.NoError(t, err)

		atNPlusOne := canonicalDepthDocument(t, MaximumJSONDepth-1)
		metrics, err = measureJSON(atNPlusOne)
		require.NoError(t, err)
		require.Equal(t, MaximumJSONDepth+1, metrics.depth)
		_, err = decoder.Decode(atNPlusOne)
		requireErrorCode(t, err, ErrorDepthLimit)
	})

	t.Run("array items N and N+1", func(t *testing.T) {
		testArrayBoundary(t, MaximumJSONArrayItems, MaximumJSONArrayItems)
	})

	t.Run("object members N and N+1", func(t *testing.T) {
		decoder := Decoder[objectBoundaryProbe]{Format: "umpire-object-boundary/v2"}
		atN := canonicalObjectDocument(t, MaximumJSONObjectMembers)
		_, err := decoder.Decode(atN)
		require.NoError(t, err)
		atNPlusOne := canonicalObjectDocument(t, MaximumJSONObjectMembers+1)
		_, err = decoder.Decode(atNPlusOne)
		requireErrorCode(t, err, ErrorCollectionLimit)
	})

	t.Run("string bytes N and N+1", func(t *testing.T) {
		testStringBoundary(t, MaximumJSONStringBytes, MaximumJSONStringBytes)
	})
}

func TestStrictJSONPerFamilyCeilingsUseEncodedNNPlusOneBoundaries(t *testing.T) {
	collectionCases := map[string]int{
		"set members":      MaximumArtifactSetMembers,
		"evidence sources": MaximumEvidenceSources,
		"evidence facts":   MaximumEvidenceFacts,
		"fields per fact":  MaximumFieldsPerEvidenceFact,
	}
	for name, limit := range collectionCases {
		t.Run(name, func(t *testing.T) {
			testArrayBoundary(t, limit, limit)
		})
	}

	stringCases := map[string]int{
		"identity bytes":   MaximumIdentityBytes,
		"diagnostic bytes": MaximumDiagnosticBytes,
	}
	for name, limit := range stringCases {
		t.Run(name, func(t *testing.T) {
			testStringBoundary(t, limit, limit)
		})
	}

	payloadCases := map[string]int{
		"payload per fact":      MaximumEvidenceFactPayloadBytes,
		"aggregate raw payload": MaximumRawEvidencePayloadBytes,
	}
	for name, limit := range payloadCases {
		t.Run(name, func(t *testing.T) {
			testPayloadBoundary(t, limit)
		})
	}
}

func canonicalDocumentOfSize(t *testing.T, size int) []byte {
	t.Helper()
	probe := arrayBoundaryProbe{
		FormatVersion: "umpire-document-boundary/v2",
		Values:        make([]string, MaximumJSONArrayItems),
	}
	empty, err := CanonicalPretty(probe)
	require.NoError(t, err)
	remaining := size - len(empty)
	require.Positive(t, remaining)
	for index := range probe.Values {
		chunk := min(remaining, MaximumJSONStringBytes)
		probe.Values[index] = strings.Repeat("x", chunk)
		remaining -= chunk
	}
	require.Zero(t, remaining)
	encoded, err := CanonicalPretty(probe)
	require.NoError(t, err)
	require.Len(t, encoded, size)
	return encoded
}

func canonicalTokenDocument(t *testing.T, finalItems int) []byte {
	t.Helper()
	values := make([][]int, 128)
	for index := range 127 {
		values[index] = make([]int, MaximumJSONArrayItems)
	}
	values[len(values)-1] = make([]int, finalItems)
	encoded, err := CanonicalPretty(tokenBoundaryProbe{
		FormatVersion: "umpire-token-boundary/v2",
		Values:        values,
	})
	require.NoError(t, err)
	return encoded
}

func canonicalDepthDocument(t *testing.T, arrays int) []byte {
	t.Helper()
	var value any
	for range arrays {
		value = []any{value}
	}
	encoded, err := CanonicalPretty(depthBoundaryProbe{
		FormatVersion: "umpire-depth-boundary/v2",
		Value:         value,
	})
	require.NoError(t, err)
	return encoded
}

func canonicalObjectDocument(t *testing.T, members int) []byte {
	t.Helper()
	values := make(map[string]string, members)
	for index := range members {
		values[fmt.Sprintf("key-%03d", index)] = ""
	}
	encoded, err := CanonicalPretty(objectBoundaryProbe{
		FormatVersion: "umpire-object-boundary/v2",
		Values:        values,
	})
	require.NoError(t, err)
	return encoded
}

func testArrayBoundary(t *testing.T, limit, items int) {
	t.Helper()
	decoder := Decoder[arrayBoundaryProbe]{
		Format: "umpire-array-boundary/v2",
		Bounds: Bounds{CollectionLimit: func(path JSONPath, kind CollectionKind) int {
			if path == "$.values" && kind == CollectionArray {
				return limit
			}
			return 0
		}},
	}
	atN, err := CanonicalPretty(arrayBoundaryProbe{
		FormatVersion: "umpire-array-boundary/v2",
		Values:        make([]string, items),
	})
	require.NoError(t, err)
	_, err = decoder.Decode(atN)
	require.NoError(t, err)
	atNPlusOne, err := CanonicalPretty(arrayBoundaryProbe{
		FormatVersion: "umpire-array-boundary/v2",
		Values:        make([]string, items+1),
	})
	require.NoError(t, err)
	_, err = decoder.Decode(atNPlusOne)
	requireErrorCode(t, err, ErrorCollectionLimit)
}

func testStringBoundary(t *testing.T, limit, bytes int) {
	t.Helper()
	decoder := Decoder[stringBoundaryProbe]{
		Format: "umpire-string-boundary/v2",
		Bounds: Bounds{StringLimit: func(path JSONPath) int {
			if path == "$.value" {
				return limit
			}
			return 0
		}},
	}
	atN, err := CanonicalPretty(stringBoundaryProbe{
		FormatVersion: "umpire-string-boundary/v2",
		Value:         strings.Repeat("x", bytes),
	})
	require.NoError(t, err)
	_, err = decoder.Decode(atN)
	require.NoError(t, err)
	atNPlusOne, err := CanonicalPretty(stringBoundaryProbe{
		FormatVersion: "umpire-string-boundary/v2",
		Value:         strings.Repeat("x", bytes+1),
	})
	require.NoError(t, err)
	_, err = decoder.Decode(atNPlusOne)
	requireErrorCode(t, err, ErrorStringLimit)
}

func testPayloadBoundary(t *testing.T, limit int) {
	t.Helper()
	total := limit
	decoder := Decoder[arrayBoundaryProbe]{
		Format: "umpire-payload-boundary/v2",
		Bounds: Bounds{PayloadLimit: func([]byte) error {
			if total > limit {
				return errors.New("decoded payload exceeds limit")
			}
			return nil
		}},
	}
	encoded, err := CanonicalPretty(arrayBoundaryProbe{FormatVersion: "umpire-payload-boundary/v2"})
	require.NoError(t, err)
	_, err = decoder.Decode(encoded)
	require.NoError(t, err)
	total++
	_, err = decoder.Decode(encoded)
	requireErrorCode(t, err, ErrorPayloadLimit)
}
