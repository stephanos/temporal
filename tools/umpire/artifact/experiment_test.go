package artifact_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

func TestExperimentV2CanonicalFixturesRoundTrip(t *testing.T) {
	for _, test := range []struct {
		fixture       string
		authoritative string
	}{
		{
			fixture:       "tools/umpire/artifact/testdata/switch-experiment-v2.json",
			authoritative: "model/Umpire/Artifact/Tests/Fixtures/SwitchExperimentSpecV2.json",
		},
		{
			fixture:       "tools/umpire/artifact/testdata/nexus-caller-closure-experiment-v2.json",
			authoritative: "model/Temporal/Feature/Nexus/Experimental/testdata/nexus-caller-closure-experiment-spec.json",
		},
	} {
		t.Run(filepath.Base(test.fixture), func(t *testing.T) {
			encoded := readExperimentV2Fixture(t, test.fixture)
			// The Go testdata remains a byte-exact mirror of its Lean-owned fixture.
			//nolint:testifylint
			require.Equal(t, readExperimentV2Fixture(t, test.authoritative), encoded)

			document, err := artifact.DecodeExperimentV2(encoded)
			require.NoError(t, err)
			reencoded, err := artifact.EncodeExperimentV2(document)
			require.NoError(t, err)
			// This is an exact persisted-byte contract; semantic JSON equality would hide spelling drift.
			//nolint:testifylint
			require.Equal(t, encoded, reencoded)
			require.True(t, bytes.HasSuffix(reencoded, []byte{'\n'}))
			require.False(t, bytes.HasSuffix(reencoded, []byte("\n\n")))
		})
	}
}

func TestExperimentV2ChecksumsUseExactPrettyPreimages(t *testing.T) {
	for _, test := range []struct {
		name               string
		fixture            string
		planChecksum       string
		experimentChecksum string
	}{
		{
			name:               "Switch",
			fixture:            "tools/umpire/artifact/testdata/switch-experiment-v2.json",
			planChecksum:       "sha256:1caad30cc09a2006600917465e4f9223529afbba7acf734c3a629b0e3723ba7d",
			experimentChecksum: "sha256:c7fc19d59b8b97922df475596bc45022e97c19d051149aa0c9aabe82dff18179",
		},
		{
			name:               "Nexus caller closure",
			fixture:            "tools/umpire/artifact/testdata/nexus-caller-closure-experiment-v2.json",
			planChecksum:       "sha256:328a90c67ca91a885a31b1e146d36af09a73cba7f729eab69a6028041a8b0bb8",
			experimentChecksum: "sha256:dde2fb35891dcc0020dbedf301805feda1b5136ec8622dd67fdc47a3d00fb1a8",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			document, err := artifact.DecodeExperimentV2(readExperimentV2Fixture(t, test.fixture))
			require.NoError(t, err)

			planChecksum := document.Plan.ArtifactChecksum
			document.Plan.ArtifactChecksum = ""
			planPreimage, err := artifact.CanonicalPretty(document.Plan)
			require.NoError(t, err)
			requireCanonicalJSONLine(t, planPreimage)
			require.Equal(t, test.planChecksum, planChecksum)
			require.Equal(t, test.planChecksum,
				independentExperimentV2Checksum("umpire.drive-plan/v2", planPreimage))

			document.Plan.ArtifactChecksum = planChecksum
			experimentChecksum := document.ArtifactChecksum
			document.ArtifactChecksum = ""
			experimentPreimage, err := artifact.CanonicalPretty(document)
			require.NoError(t, err)
			requireCanonicalJSONLine(t, experimentPreimage)
			require.Equal(t, test.experimentChecksum, experimentChecksum)
			require.Equal(t, test.experimentChecksum,
				independentExperimentV2Checksum("umpire.experiment-spec/v2", experimentPreimage))
		})
	}
}

func TestExperimentV2RejectsOneAtATimeMutations(t *testing.T) {
	canonical := readExperimentV2Fixture(t,
		"tools/umpire/artifact/testdata/switch-experiment-v2.json")
	withoutLF := bytes.TrimSuffix(canonical, []byte{'\n'})
	var compact bytes.Buffer
	require.NoError(t, json.Compact(&compact, canonical))
	compact.WriteByte('\n')
	earlierFormat := "umpire-experiment/" + "v" + "1"
	earlierWithInvalidField := replaceExperimentV2Once(t, canonical,
		`"umpire-experiment/v2"`, `"`+earlierFormat+`"`)
	earlierWithInvalidField = replaceExperimentV2Once(t, earlierWithInvalidField,
		"{\n  \"formatVersion\":", "{\n  \"legacy\": true,\n  \"formatVersion\":")
	legacyKey := "semantic" + "Identity"

	cases := map[string]struct {
		encoded []byte
		code    artifact.ErrorCode
	}{
		"malformed JSON": {
			encoded: []byte("{\n"),
			code:    artifact.ErrorSyntax,
		},
		"pre-v2 before invalid field": {
			encoded: earlierWithInvalidField,
			code:    artifact.ErrorUnsupportedFormat,
		},
		"unsupported later major": {
			encoded: replaceExperimentV2Once(t, canonical,
				`"umpire-experiment/v2"`, `"umpire-experiment/v3"`),
			code: artifact.ErrorUnsupportedFormat,
		},
		"wrong family": {
			encoded: replaceExperimentV2Once(t, canonical,
				`"umpire-experiment/v2"`, `"umpire-result/v2"`),
			code: artifact.ErrorWrongFamily,
		},
		"duplicate key": {
			encoded: replaceExperimentV2Once(t, canonical,
				"\n  \"queryBehaviorFingerprint\": ",
				"\n  \"queryBehaviorFingerprint\": \"sha256:0000000000000000000000000000000000000000000000000000000000000000\",\n  \"queryBehaviorFingerprint\": "),
			code: artifact.ErrorDuplicateKey,
		},
		"case collision": {
			encoded: replaceExperimentV2Once(t, canonical,
				"\n  \"queryBehaviorFingerprint\"", "\n  \"QueryBehaviorFingerprint\""),
			code: artifact.ErrorCaseCollision,
		},
		"legacy key": {
			encoded: replaceExperimentV2Once(t, canonical,
				"\n  \"queryBehaviorFingerprint\"", "\n  \""+legacyKey+"\""),
			code: artifact.ErrorUnknownField,
		},
		"unknown key": {
			encoded: replaceExperimentV2Once(t, canonical,
				"{\n  \"formatVersion\":", "{\n  \"unknown\": true,\n  \"formatVersion\":"),
			code: artifact.ErrorUnknownField,
		},
		"compact JSON": {
			encoded: compact.Bytes(),
			code:    artifact.ErrorNoncanonical,
		},
		"alternate indentation": {
			encoded: replaceExperimentV2Once(t, canonical,
				"{\n  \"formatVersion\"", "{\n    \"formatVersion\""),
			code: artifact.ErrorNoncanonical,
		},
		"reordered fields": {
			encoded: replaceExperimentV2Once(t, canonical,
				"{\n  \"formatVersion\": \"umpire-experiment/v2\",\n  \"queryBehaviorFingerprint\": \"sha256:d915da489735c26fcb295cbbd5e246f6758f612eb7141d448ab84716b02766d0\",",
				"{\n  \"queryBehaviorFingerprint\": \"sha256:d915da489735c26fcb295cbbd5e246f6758f612eb7141d448ab84716b02766d0\",\n  \"formatVersion\": \"umpire-experiment/v2\","),
			code: artifact.ErrorNoncanonical,
		},
		"alternate string escape": {
			encoded: replaceExperimentV2Once(t, canonical,
				`"queryDefinitionId": "switch.query.exact-action"`,
				`"queryDefinitionId": "switch.query.exact\u002daction"`),
			code: artifact.ErrorNoncanonical,
		},
		"missing terminal LF": {
			encoded: withoutLF,
			code:    artifact.ErrorNoncanonical,
		},
		"extra terminal LF": {
			encoded: append(bytes.Clone(canonical), '\n'),
			code:    artifact.ErrorNoncanonical,
		},
		"malformed Definition ID": {
			encoded: resealedExperimentV2Mutation(t, func(document *artifactv2.Experiment) {
				document.Plan.QueryDefinitionID = "unnamespaced"
			}),
			code: artifact.ErrorMalformedValue,
		},
		"malformed Behavior Fingerprint": {
			encoded: resealedExperimentV2Mutation(t, func(document *artifactv2.Experiment) {
				document.Plan.BehaviorFingerprint = "sha256:ABC"
			}),
			code: artifact.ErrorMalformedValue,
		},
		"invalid Limit": {
			encoded: resealedExperimentV2Mutation(t, func(document *artifactv2.Experiment) {
				document.Plan.ExpandedLimits.Search.Unit = "unbounded"
			}),
			code: artifact.ErrorMalformedValue,
		},
		"invalid Known Gap": {
			encoded: resealedExperimentV2Mutation(t, func(document *artifactv2.Experiment) {
				document.Plan.KnownGaps[0].Kind = "free-form"
			}),
			code: artifact.ErrorMalformedValue,
		},
		"invalid occurrence": {
			encoded: resealedExperimentV2Mutation(t, func(document *artifactv2.Experiment) {
				document.Plan.LinearExtension[0].DefinitionID = "unnamespaced"
			}),
			code: artifact.ErrorMalformedValue,
		},
		"invalid checkpoint": {
			encoded: resealedExperimentV2Mutation(t, func(document *artifactv2.Experiment) {
				document.Plan.Checkpoints[0].Observations = nil
			}),
			code: artifact.ErrorMalformedValue,
		},
		"invalid Property": {
			encoded: resealedExperimentV2Mutation(t, func(document *artifactv2.Experiment) {
				document.Properties[0].DefinitionID = "unnamespaced"
			}),
			code: artifact.ErrorMalformedValue,
		},
		"invalid requirement": {
			encoded: resealedExperimentV2Mutation(t, func(document *artifactv2.Experiment) {
				document.Properties[0].RequirementDefinitionIDs = []string{""}
			}),
			code: artifact.ErrorMalformedValue,
		},
		"invalid provenance": {
			encoded: resealedExperimentV2Mutation(t, func(document *artifactv2.Experiment) {
				document.Provenance.SourceLocations[0].Line = "0"
			}),
			code: artifact.ErrorMalformedValue,
		},
		"nested checksum drift": {
			encoded: replaceExperimentV2Once(t, canonical,
				"sha256:1caad30cc09a2006600917465e4f9223529afbba7acf734c3a629b0e3723ba7d",
				"sha256:2caad30cc09a2006600917465e4f9223529afbba7acf734c3a629b0e3723ba7d"),
			code: artifact.ErrorArtifactChecksum,
		},
		"outer checksum drift": {
			encoded: replaceExperimentV2Once(t, canonical,
				"sha256:c7fc19d59b8b97922df475596bc45022e97c19d051149aa0c9aabe82dff18179",
				"sha256:d7fc19d59b8b97922df475596bc45022e97c19d051149aa0c9aabe82dff18179"),
			code: artifact.ErrorArtifactChecksum,
		},
		"closure drift": {
			encoded: resealedExperimentV2Mutation(t, func(document *artifactv2.Experiment) {
				document.QueryBehaviorFingerprint = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
			}),
			code: artifact.ErrorClosure,
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := artifact.DecodeExperimentV2(test.encoded)
			require.Error(t, err)
			code, ok := artifact.CodeOf(err)
			require.True(t, ok, err)
			require.Equal(t, test.code, code)
		})
	}
}

func readExperimentV2Fixture(t *testing.T, relative string) []byte {
	t.Helper()
	repositoryRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	encoded, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(relative)))
	require.NoError(t, err)
	return encoded
}

func resealedExperimentV2Mutation(t *testing.T, mutate func(*artifactv2.Experiment)) []byte {
	t.Helper()
	encoded := readExperimentV2Fixture(t,
		"tools/umpire/artifact/testdata/switch-experiment-v2.json")
	document, err := artifactv2.DecodeExperiment(encoded)
	require.NoError(t, err)
	mutate(&document)
	document, err = artifactv2.SealExperiment(document)
	require.NoError(t, err)
	encoded, err = artifactv2.CanonicalExperimentBytes(document)
	require.NoError(t, err)
	return encoded
}

func replaceExperimentV2Once(t *testing.T, encoded []byte, old, replacement string) []byte {
	t.Helper()
	require.Equal(t, 1, bytes.Count(encoded, []byte(old)))
	return bytes.Replace(encoded, []byte(old), []byte(replacement), 1)
}

func independentExperimentV2Checksum(domain string, preimage []byte) string {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(domain))
	_, _ = hasher.Write([]byte{'\n'})
	_, _ = hasher.Write(preimage)
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil))
}

func requireCanonicalJSONLine(t *testing.T, encoded []byte) {
	t.Helper()
	require.True(t, bytes.HasSuffix(encoded, []byte{'\n'}))
	require.False(t, bytes.HasSuffix(encoded, []byte("\n\n")))
}
