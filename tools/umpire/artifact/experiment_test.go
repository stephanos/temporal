package artifact_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
			planChecksum:       "sha256:a695f9f6cc79ba49a721d1764519e2167b5fe66278666238c6da862b1a33b835",
			experimentChecksum: "sha256:ac3fde668a79ff0433106e28f8ec9579a36f9f7d0ab09845d01b563289b560fd",
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
	nestedEarlierFormat := "umpire-drive-plan/" + "v" + "1"
	nestedEarlierWithUnknown := replaceExperimentV2Once(t, canonical,
		"\"plan\": {\n    \"formatVersion\": \"umpire-drive-plan/v2\",",
		"\"plan\": {\n    \"unknown\": true,\n    \"formatVersion\": \""+nestedEarlierFormat+"\",")

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
		"nested pre-v2 before unknown field": {
			encoded: nestedEarlierWithUnknown,
			code:    artifact.ErrorUnsupportedFormat,
		},
		"unsupported nested later major": {
			encoded: replaceExperimentV2Once(t, canonical,
				`"umpire-drive-plan/v2"`, `"umpire-drive-plan/v3"`),
			code: artifact.ErrorUnsupportedFormat,
		},
		"wrong nested family": {
			encoded: replaceExperimentV2Once(t, canonical,
				`"umpire-drive-plan/v2"`, `"umpire-result/v2"`),
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
				"{\n  \"formatVersion\": \"umpire-experiment/v2\",\n  \"queryBehaviorFingerprint\": \"sha256:c296b131ab2a42992a13cc733b050536389c20b3589e7d59cfa70c88f1ae423b\",",
				"{\n  \"queryBehaviorFingerprint\": \"sha256:c296b131ab2a42992a13cc733b050536389c20b3589e7d59cfa70c88f1ae423b\",\n  \"formatVersion\": \"umpire-experiment/v2\","),
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
		"non-ASCII Definition ID": {
			encoded: resealedExperimentV2Mutation(t, func(document *artifactv2.Experiment) {
				document.Plan.QueryDefinitionID = "switch.query.exact-actiøn"
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
				document.Properties[0].RequirementDefinitionIDs = []string{"unnamespaced"}
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
				"sha256:a695f9f6cc79ba49a721d1764519e2167b5fe66278666238c6da862b1a33b835",
				"sha256:2caad30cc09a2006600917465e4f9223529afbba7acf734c3a629b0e3723ba7d"),
			code: artifact.ErrorArtifactChecksum,
		},
		"outer checksum drift": {
			encoded: replaceExperimentV2Once(t, canonical,
				"sha256:ac3fde668a79ff0433106e28f8ec9579a36f9f7d0ab09845d01b563289b560fd",
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

func TestExperimentV2StringBounds(t *testing.T) {
	identityAtLimit := "a." + strings.Repeat("x", artifact.MaximumIdentityBytes-2)
	identityOverLimit := identityAtLimit + "x"
	detailAtLimit := strings.Repeat("x", artifact.MaximumDiagnosticBytes)
	detailOverLimit := detailAtLimit + "x"

	for _, test := range []struct {
		name      string
		atLimit   func(*artifactv2.Experiment)
		overLimit func(*artifactv2.Experiment)
	}{
		{
			name: "identity",
			atLimit: func(document *artifactv2.Experiment) {
				document.Plan.QueryDefinitionID = identityAtLimit
			},
			overLimit: func(document *artifactv2.Experiment) {
				document.Plan.QueryDefinitionID = identityOverLimit
			},
		},
		{
			name: "diagnostic detail",
			atLimit: func(document *artifactv2.Experiment) {
				document.Plan.KnownGaps[0].Detail = &detailAtLimit
			},
			overLimit: func(document *artifactv2.Experiment) {
				document.Plan.KnownGaps[0].Detail = &detailOverLimit
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := artifact.DecodeExperimentV2(resealedExperimentV2Mutation(t, test.atLimit))
			require.NoError(t, err)

			_, err = artifact.DecodeExperimentV2(resealedExperimentV2Mutation(t, test.overLimit))
			requireExperimentV2ErrorCode(t, err, artifact.ErrorStringLimit)
		})
	}
}

func TestExperimentV2RejectsMalformedDefinitionIDSets(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*artifactv2.Experiment)
	}{
		{
			name: "plan capability requirement",
			mutate: func(document *artifactv2.Experiment) {
				document.Plan.CapabilityRequirementDefinitionIDs = []string{"unnamespaced"}
			},
		},
		{
			name: "plan source definition",
			mutate: func(document *artifactv2.Experiment) {
				document.Plan.Provenance.SourceDefinitionIDs = []string{"unnamespaced"}
			},
		},
		{
			name: "property requirement",
			mutate: func(document *artifactv2.Experiment) {
				document.Properties[0].RequirementDefinitionIDs = []string{"unnamespaced"}
			},
		},
		{
			name: "observation requirement",
			mutate: func(document *artifactv2.Experiment) {
				document.ObservationRequirementDefinitionIDs = []string{"unnamespaced"}
			},
		},
		{
			name: "experiment source definition",
			mutate: func(document *artifactv2.Experiment) {
				document.Provenance.SourceDefinitionIDs = []string{"nonascii.réquirement"}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := artifact.DecodeExperimentV2(resealedExperimentV2Mutation(t, test.mutate))
			requireExperimentV2ErrorCode(t, err, artifact.ErrorMalformedValue)
		})
	}
}

func TestExperimentV2EncodeRejectsInvalidValues(t *testing.T) {
	canonical := readExperimentV2Fixture(t,
		"tools/umpire/artifact/testdata/switch-experiment-v2.json")
	document, err := artifactv2.DecodeExperiment(canonical)
	require.NoError(t, err)

	for _, test := range []struct {
		name   string
		mutate func(*artifactv2.Experiment)
		code   artifact.ErrorCode
	}{
		{
			name: "unsupported outer format",
			mutate: func(document *artifactv2.Experiment) {
				document.FormatVersion = "umpire-experiment/" + "v" + "1"
			},
			code: artifact.ErrorUnsupportedFormat,
		},
		{
			name: "unsupported nested format",
			mutate: func(document *artifactv2.Experiment) {
				document.Plan.FormatVersion = "umpire-drive-plan/" + "v" + "1"
			},
			code: artifact.ErrorUnsupportedFormat,
		},
		{
			name: "malformed field",
			mutate: func(document *artifactv2.Experiment) {
				document.Plan.QueryDefinitionID = "unnamespaced"
			},
			code: artifact.ErrorMalformedValue,
		},
		{
			name: "stale checksum",
			mutate: func(document *artifactv2.Experiment) {
				document.Plan.ArtifactChecksum = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
			},
			code: artifact.ErrorArtifactChecksum,
		},
		{
			name: "closure drift",
			mutate: func(document *artifactv2.Experiment) {
				document.QueryBehaviorFingerprint = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
				sealed, sealErr := artifactv2.SealExperiment(*document)
				require.NoError(t, sealErr)
				*document = sealed
			},
			code: artifact.ErrorClosure,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			mutated := document
			test.mutate(&mutated)
			encoded, err := artifact.EncodeExperimentV2(mutated)
			require.Nil(t, encoded)
			requireExperimentV2ErrorCode(t, err, test.code)
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

func requireExperimentV2ErrorCode(t *testing.T, err error, expected artifact.ErrorCode) {
	t.Helper()
	require.Error(t, err)
	code, ok := artifact.CodeOf(err)
	require.True(t, ok, err)
	require.Equal(t, expected, code)
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
