package artifact

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const canonicalStrictProbe = "{\n  \"formatVersion\": \"umpire-probe/v2\",\n  \"name\": \"probe\",\n  \"items\": [\n    \"first\"\n  ]\n}\n"

type strictProbe struct {
	FormatVersion string   `json:"formatVersion"`
	Name          string   `json:"name"`
	Items         []string `json:"items"`
}

type strictNumberProbe struct {
	FormatVersion string      `json:"formatVersion"`
	Value         json.Number `json:"value"`
}

func TestStrictJSONCanonicalPretty(t *testing.T) {
	encoded, err := CanonicalPretty(strictProbe{
		FormatVersion: "umpire-probe/v2",
		Name:          "probe",
		Items:         []string{"first"},
	})
	require.NoError(t, err)
	// This comparison is intentionally byte-exact; JSONEq would hide canonical spelling drift.
	//nolint:testifylint
	require.Equal(t, canonicalStrictProbe, string(encoded))
	require.True(t, bytes.HasSuffix(encoded, []byte{'\n'}))
	require.False(t, bytes.HasSuffix(encoded, []byte("\n\n")))
}

func TestStrictJSONAcceptsOnlyCanonicalPrettyBytes(t *testing.T) {
	decoder := Decoder[strictProbe]{Format: "umpire-probe/v2"}

	decoded, err := decoder.Decode([]byte(canonicalStrictProbe))
	require.NoError(t, err)
	require.Equal(t, strictProbe{
		FormatVersion: "umpire-probe/v2",
		Name:          "probe",
		Items:         []string{"first"},
	}, decoded)

	cases := map[string][]byte{
		"compact":              []byte(`{"formatVersion":"umpire-probe/v2","name":"probe","items":["first"]}` + "\n"),
		"alternate whitespace": []byte("{\n    \"formatVersion\": \"umpire-probe/v2\",\n    \"name\": \"probe\",\n    \"items\": [\"first\"]\n}\n"),
		"missing LF":           bytes.TrimSuffix([]byte(canonicalStrictProbe), []byte{'\n'}),
		"extra LF":             append([]byte(canonicalStrictProbe), '\n'),
		"alternate escape":     bytes.Replace([]byte(canonicalStrictProbe), []byte(`"probe"`), []byte(`"\u0070robe"`), 1),
		"escaped slash":        bytes.Replace([]byte(canonicalStrictProbe), []byte(`umpire-probe/v2`), []byte(`umpire-probe\/v2`), 1),
	}
	for name, encoded := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := decoder.Decode(encoded)
			requireErrorCode(t, err, ErrorNoncanonical)
		})
	}
}

func TestStrictJSONRejectsNoncanonicalNumberSpellings(t *testing.T) {
	decoder := Decoder[strictNumberProbe]{Format: "umpire-probe/v2"}
	for _, spelling := range []string{"1e0", "1.0", "-0"} {
		t.Run(spelling, func(t *testing.T) {
			encoded := []byte("{\n  \"formatVersion\": \"umpire-probe/v2\",\n  \"value\": " + spelling + "\n}\n")
			_, err := decoder.Decode(encoded)
			requireErrorCode(t, err, ErrorNoncanonical)
		})
	}
}

func TestStrictJSONRejectsEveryStructuralClass(t *testing.T) {
	collectionOne := func(path JSONPath, kind CollectionKind) int {
		if path == "$.items" && kind == CollectionArray {
			return 1
		}
		return 0
	}
	stringFour := func(path JSONPath) int {
		if path == "$.name" {
			return 4
		}
		return 0
	}
	cases := map[string]struct {
		decoder Decoder[strictProbe]
		encoded []byte
		want    ErrorCode
	}{
		"syntax": {
			decoder: Decoder[strictProbe]{Format: "umpire-probe/v2"},
			encoded: []byte("{\n"),
			want:    ErrorSyntax,
		},
		"duplicate key": {
			decoder: Decoder[strictProbe]{Format: "umpire-probe/v2"},
			encoded: []byte(`{"formatVersion":"umpire-probe/v2","formatVersion":"umpire-probe/v2"}`),
			want:    ErrorDuplicateKey,
		},
		"case collision": {
			decoder: Decoder[strictProbe]{Format: "umpire-probe/v2"},
			encoded: []byte(`{"FormatVersion":"umpire-probe/v2"}`),
			want:    ErrorCaseCollision,
		},
		"unsupported format": {
			decoder: Decoder[strictProbe]{Format: "umpire-probe/v2"},
			encoded: []byte(`{"formatVersion":"umpire-probe/v1","unknown":true}`),
			want:    ErrorUnsupportedFormat,
		},
		"wrong family": {
			decoder: Decoder[strictProbe]{Format: "umpire-probe/v2"},
			encoded: []byte(`{"formatVersion":"umpire-other/v2","unknown":true}`),
			want:    ErrorWrongFamily,
		},
		"unknown field": {
			decoder: Decoder[strictProbe]{Format: "umpire-probe/v2"},
			encoded: []byte(`{"formatVersion":"umpire-probe/v2","unknown":true}`),
			want:    ErrorUnknownField,
		},
		"collection limit": {
			decoder: Decoder[strictProbe]{
				Format: "umpire-probe/v2",
				Bounds: Bounds{CollectionLimit: collectionOne},
			},
			encoded: []byte(`{"formatVersion":"umpire-probe/v2","name":"ok","items":["a","b"]}`),
			want:    ErrorCollectionLimit,
		},
		"string limit": {
			decoder: Decoder[strictProbe]{
				Format: "umpire-probe/v2",
				Bounds: Bounds{StringLimit: stringFour},
			},
			encoded: []byte(canonicalStrictProbe),
			want:    ErrorStringLimit,
		},
		"payload limit": {
			decoder: Decoder[strictProbe]{
				Format: "umpire-probe/v2",
				Bounds: Bounds{PayloadLimit: func([]byte) error { return errors.New("too large") }},
			},
			encoded: []byte(canonicalStrictProbe),
			want:    ErrorPayloadLimit,
		},
		"malformed value": {
			decoder: Decoder[strictProbe]{Format: "umpire-probe/v2"},
			encoded: []byte("{\n  \"formatVersion\": \"umpire-probe/v2\",\n  \"name\": 7,\n  \"items\": []\n}\n"),
			want:    ErrorMalformedValue,
		},
		"noncanonical": {
			decoder: Decoder[strictProbe]{Format: "umpire-probe/v2"},
			encoded: []byte(`{"formatVersion":"umpire-probe/v2","name":"probe","items":["first"]}` + "\n"),
			want:    ErrorNoncanonical,
		},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := test.decoder.Decode(test.encoded)
			requireErrorCode(t, err, test.want)
		})
	}
}

func TestStrictJSONUsesCanonicalAndValidationHooks(t *testing.T) {
	custom := []byte(canonicalStrictProbe)
	decoder := Decoder[strictProbe]{
		Format: "umpire-probe/v2",
		Validate: func(value strictProbe) error {
			if value.Name != "probe" {
				return errors.New("wrong name")
			}
			return nil
		},
		Canonical: func(strictProbe) ([]byte, error) {
			return bytes.Clone(custom), nil
		},
	}

	decoded, err := decoder.Decode(custom)
	require.NoError(t, err)
	require.Equal(t, "probe", decoded.Name)

	decoder.Validate = func(strictProbe) error { return errors.New("invalid") }
	_, err = decoder.Decode(custom)
	requireErrorCode(t, err, ErrorMalformedValue)
}

func TestStrictJSONCountsPunctuationScalarsAndRootDepth(t *testing.T) {
	cases := map[string]struct {
		encoded    string
		wantTokens int
		wantDepth  int
	}{
		"root scalar":  {encoded: `null`, wantTokens: 1, wantDepth: 1},
		"empty array":  {encoded: `[]`, wantTokens: 2, wantDepth: 1},
		"array":        {encoded: `[null,true]`, wantTokens: 5, wantDepth: 2},
		"empty object": {encoded: `{}`, wantTokens: 2, wantDepth: 1},
		"object":       {encoded: `{"a":null}`, wantTokens: 5, wantDepth: 2},
		"nested":       {encoded: `{"a":[null]}`, wantTokens: 7, wantDepth: 3},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			metrics, err := measureJSON([]byte(test.encoded))
			require.NoError(t, err)
			require.Equal(t, test.wantTokens, metrics.tokens)
			require.Equal(t, test.wantDepth, metrics.depth)
		})
	}
}

func TestStrictJSONScannerUsesBoundedBookkeeping(t *testing.T) {
	encoded := []byte(strings.Repeat("[", 4_096) + "null" + strings.Repeat("]", 4_096))
	var scanErr error
	allocations := testing.AllocsPerRun(5, func() {
		_, scanErr = measureJSON(encoded)
	})

	require.NoError(t, scanErr)
	require.Zero(t, allocations)
}

func TestStrictJSONStopsObjectBookkeepingAtNPlusOne(t *testing.T) {
	near := overlimitObjectDocument(t, MaximumJSONObjectMembers+1)
	far := overlimitObjectDocument(t, MaximumJSONArrayItems)
	decoder := Decoder[objectBoundaryProbe]{Format: "umpire-map-boundary/v2"}
	_, err := decoder.Decode(near)
	requireErrorCode(t, err, ErrorCollectionLimit)
	_, err = decoder.Decode(far)
	requireErrorCode(t, err, ErrorCollectionLimit)

	allocatedBytes := func(encoded []byte) int64 {
		result := testing.Benchmark(func(benchmark *testing.B) {
			benchmark.ReportAllocs()
			for range benchmark.N {
				_, _ = decoder.Decode(encoded)
			}
		})
		return result.AllocedBytesPerOp()
	}
	require.LessOrEqual(t, allocatedBytes(far), allocatedBytes(near)+1_024)
}

func overlimitObjectDocument(t *testing.T, members int) []byte {
	t.Helper()
	var encoded strings.Builder
	encoded.WriteString(`{"formatVersion":"umpire-map-boundary/v2","values":{`)
	for index := range members {
		if index > 0 {
			encoded.WriteByte(',')
		}
		_, err := fmt.Fprintf(&encoded, `"key-%06d":""`, index)
		require.NoError(t, err)
	}
	encoded.WriteString("}}")
	return []byte(encoded.String())
}

func TestStrictJSONCountsDecodedStringBytesWithoutMaterializingStrings(t *testing.T) {
	metrics, err := measureJSON([]byte(`"é\u00e9\uD83D\uDE00"`))
	require.NoError(t, err)
	require.Equal(t, 8, metrics.stringBytes)
}

func TestStrictJSONAppliesStableErrorPrecedence(t *testing.T) {
	base := Decoder[strictProbe]{Format: "umpire-probe/v2"}
	fail := func() error { return errors.New("failed") }
	collectionOne := func(path JSONPath, kind CollectionKind) int {
		if path == "$.items" && kind == CollectionArray {
			return 1
		}
		return 0
	}
	shortStrings := func(path JSONPath) int {
		if path == "$.name" || path == "$.items[*]" {
			return 1
		}
		return 0
	}
	cases := map[string]struct {
		decoder Decoder[strictProbe]
		encoded []byte
		limits  structuralLimits
		want    ErrorCode
	}{
		"byte before syntax": {
			decoder: base,
			encoded: []byte("not-json"),
			limits:  structuralLimits{documentBytes: 7, tokens: MaximumJSONTokens, depth: MaximumJSONDepth, arrayItems: MaximumJSONArrayItems, objectMembers: MaximumJSONObjectMembers, stringBytes: MaximumJSONStringBytes},
			want:    ErrorByteLimit,
		},
		"syntax before token": {
			decoder: base,
			encoded: []byte(`{"formatVersion":"umpire-probe/v2","items":[0,0`),
			limits:  structuralLimits{documentBytes: MaximumDocumentBytes, tokens: 3, depth: MaximumJSONDepth, arrayItems: MaximumJSONArrayItems, objectMembers: MaximumJSONObjectMembers, stringBytes: MaximumJSONStringBytes},
			want:    ErrorSyntax,
		},
		"token before depth": {
			decoder: Decoder[strictProbe]{Format: "umpire-probe/v2"},
			encoded: []byte(`{"formatVersion":"umpire-probe/v2","name":"probe","items":[[["first"]]]}`),
			limits:  structuralLimits{documentBytes: MaximumDocumentBytes, tokens: 10, depth: 2, arrayItems: MaximumJSONArrayItems, objectMembers: MaximumJSONObjectMembers, stringBytes: MaximumJSONStringBytes},
			want:    ErrorTokenLimit,
		},
		"depth before duplicate": {
			decoder: Decoder[strictProbe]{Format: "umpire-probe/v2"},
			encoded: []byte(`{"formatVersion":"umpire-probe/v2","items":[[["first"]]],"name":"a","name":"b"}`),
			limits:  structuralLimits{documentBytes: MaximumDocumentBytes, tokens: MaximumJSONTokens, depth: 2, arrayItems: MaximumJSONArrayItems, objectMembers: MaximumJSONObjectMembers, stringBytes: MaximumJSONStringBytes},
			want:    ErrorDepthLimit,
		},
		"duplicate before case collision": {
			decoder: base,
			encoded: []byte(`{"formatVersion":"umpire-probe/v2","formatVersion":"umpire-probe/v2","Name":"probe"}`),
			limits:  standardStructuralLimits,
			want:    ErrorDuplicateKey,
		},
		"case collision before unsupported format": {
			decoder: base,
			encoded: []byte(`{"FormatVersion":"umpire-probe/v1"}`),
			limits:  standardStructuralLimits,
			want:    ErrorCaseCollision,
		},
		"unsupported format before wrong family": {
			decoder: base,
			encoded: []byte(`{"formatVersion":"umpire-other/v1"}`),
			limits:  standardStructuralLimits,
			want:    ErrorUnsupportedFormat,
		},
		"wrong family before unknown field": {
			decoder: base,
			encoded: []byte(`{"formatVersion":"umpire-other/v2","unknown":true}`),
			limits:  standardStructuralLimits,
			want:    ErrorWrongFamily,
		},
		"unknown field before collection limit": {
			decoder: Decoder[strictProbe]{Format: "umpire-probe/v2", Bounds: Bounds{CollectionLimit: collectionOne}},
			encoded: []byte(`{"formatVersion":"umpire-probe/v2","unknown":true,"items":["a","b"]}`),
			limits:  standardStructuralLimits,
			want:    ErrorUnknownField,
		},
		"collection limit before string limit": {
			decoder: Decoder[strictProbe]{
				Format: "umpire-probe/v2",
				Bounds: Bounds{CollectionLimit: collectionOne, StringLimit: shortStrings},
			},
			encoded: []byte(`{"formatVersion":"umpire-probe/v2","name":"p","items":["aa","bb"]}`),
			limits:  standardStructuralLimits,
			want:    ErrorCollectionLimit,
		},
		"string limit before payload limit": {
			decoder: Decoder[strictProbe]{
				Format: "umpire-probe/v2",
				Bounds: Bounds{
					StringLimit:  shortStrings,
					PayloadLimit: func([]byte) error { return fail() },
				},
			},
			encoded: []byte(canonicalStrictProbe),
			limits:  standardStructuralLimits,
			want:    ErrorStringLimit,
		},
		"payload limit before malformed value": {
			decoder: Decoder[strictProbe]{
				Format: "umpire-probe/v2",
				Bounds: Bounds{PayloadLimit: func([]byte) error { return fail() }},
			},
			encoded: []byte(`{"formatVersion":"umpire-probe/v2","name":7,"items":[]}`),
			limits:  standardStructuralLimits,
			want:    ErrorPayloadLimit,
		},
		"malformed before noncanonical": {
			decoder: Decoder[strictProbe]{Format: "umpire-probe/v2", Validate: func(strictProbe) error { return fail() }},
			encoded: []byte(`{"formatVersion":"umpire-probe/v2","name":"probe","items":["first"]}`),
			limits:  standardStructuralLimits,
			want:    ErrorMalformedValue,
		},
		"noncanonical before provenance checksum": {
			decoder: Decoder[strictProbe]{Format: "umpire-probe/v2", ProvenanceChecksum: func(strictProbe) error { return fail() }},
			encoded: []byte(`{"formatVersion":"umpire-probe/v2","name":"probe","items":["first"]}`),
			limits:  standardStructuralLimits,
			want:    ErrorNoncanonical,
		},
		"provenance before artifact checksum": {
			decoder: Decoder[strictProbe]{
				Format:             "umpire-probe/v2",
				ProvenanceChecksum: func(strictProbe) error { return fail() },
				ArtifactChecksum:   func(strictProbe) error { return fail() },
			},
			encoded: []byte(canonicalStrictProbe),
			limits:  standardStructuralLimits,
			want:    ErrorProvenanceChecksum,
		},
		"artifact checksum before closure": {
			decoder: Decoder[strictProbe]{
				Format:           "umpire-probe/v2",
				ArtifactChecksum: func(strictProbe) error { return fail() },
				Closure:          func(strictProbe) error { return fail() },
			},
			encoded: []byte(canonicalStrictProbe),
			limits:  standardStructuralLimits,
			want:    ErrorArtifactChecksum,
		},
		"closure": {
			decoder: Decoder[strictProbe]{Format: "umpire-probe/v2", Closure: func(strictProbe) error { return fail() }},
			encoded: []byte(canonicalStrictProbe),
			limits:  standardStructuralLimits,
			want:    ErrorClosure,
		},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := decodeWithStructuralLimits(test.decoder, test.encoded, test.limits)
			requireErrorCode(t, err, test.want)
		})
	}
}

func TestStrictJSONBoundOverridesCanOnlyTighten(t *testing.T) {
	decoder := Decoder[strictProbe]{
		Format: "umpire-probe/v2",
		Bounds: Bounds{
			CollectionLimit: func(JSONPath, CollectionKind) int { return MaximumJSONArrayItems + 1 },
			StringLimit:     func(JSONPath) int { return MaximumJSONStringBytes + 1 },
		},
	}

	collectionLimits := standardStructuralLimits
	collectionLimits.arrayItems = 0
	analysis, err := inspectJSON(
		[]byte(canonicalStrictProbe),
		schemaFor[strictProbe](),
		decoder.Bounds,
		collectionLimits,
	)
	require.NoError(t, err)
	require.True(t, analysis.collectionLimit)

	stringLimits := standardStructuralLimits
	stringLimits.stringBytes = 0
	analysis, err = inspectJSON(
		[]byte(canonicalStrictProbe),
		schemaFor[strictProbe](),
		decoder.Bounds,
		stringLimits,
	)
	require.NoError(t, err)
	require.True(t, analysis.stringLimit)
}

func FuzzStrictJSONNoPanicOrPermissiveSuccess(f *testing.F) {
	f.Add([]byte(canonicalStrictProbe))
	f.Add([]byte(`{"formatVersion":"umpire-probe/v2"}`))
	f.Add([]byte("{\n"))
	f.Add([]byte(`{"formatVersion":"umpire-probe/v2","formatVersion":"umpire-probe/v2"}`))
	f.Add(append([]byte(`{"formatVersion":"umpire-probe/v2","items":`),
		[]byte("[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[null]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]}")...))
	f.Add([]byte{'{', '"', 0xff, '"', ':', '0', '}'})
	decoder := Decoder[strictProbe]{Format: "umpire-probe/v2"}
	f.Fuzz(func(t *testing.T, encoded []byte) {
		decoded, err := decoder.Decode(encoded)
		if err != nil {
			_, ok := CodeOf(err)
			require.True(t, ok)
			return
		}
		canonical, canonicalErr := CanonicalPretty(decoded)
		require.NoError(t, canonicalErr)
		require.Equal(t, encoded, canonical)
		require.Equal(t, "umpire-probe/v2", decoded.FormatVersion)
	})
}

func requireErrorCode(t *testing.T, err error, want ErrorCode) {
	t.Helper()
	require.Error(t, err)
	got, ok := CodeOf(err)
	require.True(t, ok, "error %v has no admission code", err)
	require.Equal(t, want, got)
}
