package canary_test

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/testpilot/evaluation"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	"go.temporal.io/server/tools/canary/assessment"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/canary/recovery"
)

// versionField is a document's format version, compact or indented, and the space before its next
// key; the first match is the document's own, a nested one comes later.
var versionField = regexp.MustCompile(`"version":(\s*)1,(\s*)`)

// canaryDocument is one format the canary reads or writes: its canonical bytes and its strict
// decoder.
type canaryDocument struct {
	name    string
	encoded []byte
	decode  func([]byte) error
}

func canaryDocuments(t *testing.T) []canaryDocument {
	t.Helper()
	root := repositoryRoot(t)
	read := func(relative string) []byte {
		encoded, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		require.NoError(t, err)
		return encoded
	}
	parseProfile := func(encoded []byte) error {
		_, err := evaluation.ParseProfile(encoded)
		return err
	}
	record, err := recovery.Encode(&recovery.Record{Version: recovery.Version, InvocationID: "1234567-1", Phase: recovery.PhaseStarted})
	require.NoError(t, err)
	return []canaryDocument{
		{"the policy", read("tools/canary/policy/production-canary.json"), func(encoded []byte) error {
			_, err := policy.Decode(encoded)
			return err
		}},
		{"the production-canary Profile", read("tools/canary/assessment/profiles/production-canary.json"), parseProfile},
		{"the canary-harness Profile", read("tools/canary/testharness/profiles/canary-harness.json"), parseProfile},
		{"a receipt", renderedReceipt(t), func(encoded []byte) error {
			_, err := evaluation.DecodeReceipt(encoded)
			return err
		}},
		{"a provenance", read("tools/canary/assessment/testdata/provenance/released.json"), func(encoded []byte) error {
			_, err := assessment.DecodeProvenance(encoded)
			return err
		}},
		{"the recovery record", record, func(encoded []byte) error {
			_, err := recovery.Decode(encoded)
			return err
		}},
	}
}

// renderedReceipt is fn-26's receipt for the test cluster's recorded canary Run under the
// production-canary Profile, as the controller renders one.
func renderedReceipt(t *testing.T) []byte {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join(repositoryRoot(t), "tools", "canary", "assessment", "testdata", "nexusCallerCanary-syncCompletion-run.json"))
	require.NoError(t, err)
	decoded, err := recordedrun.Decode(encoded)
	require.NoError(t, err)
	canary, err := policy.Embedded()
	require.NoError(t, err)
	profile, err := assessment.LoadProfile(canary.EvaluationProfile)
	require.NoError(t, err)
	subject, err := assessment.Admit(canary, decoded.Driver, decoded.Run)
	require.NoError(t, err)
	receipt, err := evaluation.Render(subject, *profile, evaluation.Assess(subject, *profile))
	require.NoError(t, err)
	return receipt
}

// Every document the canary reads or writes is at format version 1, decodes from its canonical
// bytes, and refuses the prior version, the next one, and every alias of 1 -- a string, a float, a
// missing field -- so no document is read under a version it does not name.
func TestEveryCanaryDocumentRejectsEveryOtherVersion(t *testing.T) {
	for _, document := range canaryDocuments(t) {
		t.Run(document.name, func(t *testing.T) {
			require.NoError(t, document.decode(document.encoded), "the canonical bytes decode")
			match := versionField.FindSubmatchIndex(document.encoded)
			require.NotNil(t, match, "the document names its format version")
			for alias, replacement := range map[string]string{
				"the prior version":   `"version":${1}0,${2}`,
				"the next version":    `"version":${1}2,${2}`,
				"a string version":    `"version":${1}"1",${2}`,
				"a float version":     `"version":${1}1.0,${2}`,
				"a negative version":  `"version":${1}-1,${2}`,
				"no version at all":   ``,
				"a case-folded field": `"Version":${1}1,${2}`,
			} {
				t.Run(alias, func(t *testing.T) {
					var mutated []byte
					mutated = versionField.Expand(mutated, []byte(replacement), document.encoded, match)
					mutated = append(append(bytes.Clone(document.encoded[:match[0]]), mutated...), document.encoded[match[1]:]...)
					require.NotEqual(t, document.encoded, mutated)
					require.Error(t, document.decode(mutated))
				})
			}
		})
	}
}
