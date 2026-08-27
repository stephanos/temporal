package artifactv2

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeExperimentAcceptsCanonicalSwitchAndNexusV2(t *testing.T) {
	for _, relative := range []string{
		"model/Umpire/Examples/testdata/switch-experiment-spec.json",
		"model/Temporal/Feature/Nexus/Experimental/testdata/nexus-caller-closure-experiment-spec.json",
	} {
		t.Run(filepath.Base(relative), func(t *testing.T) {
			document, err := DecodeExperiment(readRepositoryFile(t, relative))
			require.NoError(t, err)
			require.Equal(t, ExperimentFormat, document.FormatVersion)
			require.Equal(t, DrivePlanFormat, document.Plan.FormatVersion)
		})
	}
}

func TestDecodeExperimentClassifiesV1BeforeV2Fields(t *testing.T) {
	encoded := []byte(`{"formatVersion":"umpire-experiment/v1","semanticIdentity":"legacy","plan":null}` + "\n")
	_, err := DecodeExperiment(encoded)
	require.EqualError(t, err, `unsupported format "umpire-experiment/v1"`)
}

func TestDecodeExperimentRejectsNoncanonicalEncodings(t *testing.T) {
	canonical := readRepositoryFile(t, "model/Umpire/Examples/testdata/switch-experiment-spec.json")
	oneLine := bytes.TrimSuffix(canonical, []byte{'\n'})
	reordered := bytes.Replace(
		oneLine,
		[]byte(`{"formatVersion":"umpire-experiment/v2","queryBehaviorFingerprint":`),
		[]byte(`{"queryBehaviorFingerprint":`),
		1,
	)
	reordered = bytes.Replace(
		reordered,
		[]byte(`,"plan":`),
		[]byte(`,"formatVersion":"umpire-experiment/v2","plan":`),
		1,
	)
	pretty := bytes.Replace(oneLine, []byte(`,"queryBehaviorFingerprint"`), []byte(",\n  \"queryBehaviorFingerprint\""), 1)
	alternateEscape := bytes.Replace(oneLine, []byte("switch.query.exact-action"), []byte(`switch.query.exact\u002daction`), 1)
	exponent := bytes.Replace(oneLine, []byte(`"position":1`), []byte(`"position":1e0`), 1)
	legacyKey := bytes.Replace(oneLine, []byte(`"queryDefinitionId"`), []byte(`"queryIdentity"`), 1)
	unknownKey := bytes.Replace(oneLine, []byte(`{"formatVersion":`), []byte(`{"unknown":true,"formatVersion":`), 1)
	caseCollision := bytes.Replace(oneLine, []byte(`"queryDefinitionId"`), []byte(`"QueryDefinitionId"`), 1)
	duplicateKey := bytes.Replace(
		oneLine,
		[]byte(`"queryBehaviorFingerprint":`),
		[]byte(`"queryBehaviorFingerprint":"sha256:0000000000000000000000000000000000000000000000000000000000000000","queryBehaviorFingerprint":`),
		1,
	)
	malformedFingerprint := bytes.Replace(oneLine, []byte("sha256:d915"), []byte("sha256:D915"), 1)
	malformedChecksum := bytes.Replace(oneLine, []byte("sha256:9533fdb58edf1ef3702c9f909ea62a3546d65d0bf864e1a224706bb18925d984"), []byte("sha256:1234"), 1)

	cases := map[string][]byte{
		"reordered object fields":        append(reordered, '\n'),
		"leading whitespace":             append([]byte{' '}, canonical...),
		"trailing whitespace":            append(append([]byte(nil), oneLine...), ' ', '\n'),
		"pretty whitespace":              append(pretty, '\n'),
		"missing terminal LF":            oneLine,
		"extra terminal LF":              append(append([]byte(nil), canonical...), '\n'),
		"alternate string escaping":      append(alternateEscape, '\n'),
		"alternate numeric encoding":     append(exponent, '\n'),
		"legacy key":                     append(legacyKey, '\n'),
		"unknown key":                    append(unknownKey, '\n'),
		"case-colliding key":             append(caseCollision, '\n'),
		"duplicate key":                  append(duplicateKey, '\n'),
		"trailing JSON data":             append(append([]byte(nil), canonical...), []byte("{}")...),
		"malformed JSON":                 []byte("{\n"),
		"malformed behavior fingerprint": append(malformedFingerprint, '\n'),
		"malformed artifact checksum":    append(malformedChecksum, '\n'),
	}
	for name, encoded := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := DecodeExperiment(encoded)
			require.Error(t, err)
		})
	}
}

func TestDecodeExperimentVerifiesNestedAndOuterChecksumsIndependently(t *testing.T) {
	canonical := readRepositoryFile(t, "model/Umpire/Examples/testdata/switch-experiment-spec.json")
	cases := map[string]struct {
		encoded []byte
		want    string
	}{
		"nested": {encoded: bytes.Replace(canonical,
			[]byte("sha256:bfa6866e94636af51a7c0cc39b8637a896b2866c3e7f0214395f0d0d803a2d72"),
			[]byte("sha256:afa6866e94636af51a7c0cc39b8637a896b2866c3e7f0214395f0d0d803a2d72"), 1), want: "nested"},
		"outer": {encoded: bytes.Replace(canonical,
			[]byte("sha256:9533fdb58edf1ef3702c9f909ea62a3546d65d0bf864e1a224706bb18925d984"),
			[]byte("sha256:8533fdb58edf1ef3702c9f909ea62a3546d65d0bf864e1a224706bb18925d984"), 1), want: "ExperimentSpec"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := DecodeExperiment(test.encoded)
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestGoSHA256MatchesLeanGoldens(t *testing.T) {
	require.Equal(t,
		"sha256:8c09aa7f7eec82e39e6f28406acc4f640dac30a2b3bf861acfaad8d701275870",
		BehaviorFingerprint([]byte(`{"definitionId":"example.target","behavior":"start->done"}`)),
	)
	require.Equal(t,
		"sha256:3f40af6e8524a50317e0e116514d05bae3a2aef6cdbf47acc8faf071e24a9a9b",
		derive(drivePlanChecksumDomain, []byte(`{"formatVersion":"umpire-drive-plan/v2","definitionId":"example.query"}`)),
	)
	require.NotEqual(t,
		derive(experimentChecksumDomain, []byte(`{"formatVersion":"umpire-drive-plan/v2","definitionId":"example.query"}`)),
		derive(drivePlanChecksumDomain, []byte(`{"formatVersion":"umpire-drive-plan/v2","definitionId":"example.query"}`)),
	)
}

func readRepositoryFile(t *testing.T, relative string) []byte {
	t.Helper()
	root := filepath.Clean(filepath.Join("..", "..", "..", ".."))
	encoded, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(string(encoded), "\n"))
	return encoded
}
