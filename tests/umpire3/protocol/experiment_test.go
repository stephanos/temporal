package protocol

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeNexusExperiment(t *testing.T) {
	encoded, err := os.ReadFile("../testdata/nexus-cancellation.json")
	require.NoError(t, err)

	experiment, err := DecodeExperiment(bytes.NewReader(encoded), DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, "nexus-cancellation-stale-completion-v1", experiment.ExperimentID)
	require.Len(t, experiment.Actions, 8)
	require.Len(t, experiment.Checkpoints, 3)
	require.NoError(t, experiment.Validate())
}

func TestExperimentCanonicalEncodingIsStable(t *testing.T) {
	encoded, err := os.ReadFile("../testdata/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := DecodeExperiment(bytes.NewReader(encoded), DefaultDecodeLimit)
	require.NoError(t, err)

	first, err := experiment.CanonicalJSON()
	require.NoError(t, err)
	second, err := experiment.CanonicalJSON()
	require.NoError(t, err)
	require.Equal(t, first, second)

	firstDigest, err := experiment.Digest()
	require.NoError(t, err)
	experiment.Scope.Bounds.MaxDepth++
	secondDigest, err := experiment.Digest()
	require.NoError(t, err)
	require.NotEqual(t, firstDigest, secondDigest)
	experiment.Scope.Assumptions = append(experiment.Scope.Assumptions, Assumption{
		Identifier:    "changed-assumption",
		StatementHash: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
	})
	assumptionDigest, err := experiment.Digest()
	require.NoError(t, err)
	require.NotEqual(t, secondDigest, assumptionDigest)
}

func TestDecodeExperimentRejectsUnknownField(t *testing.T) {
	_, err := DecodeExperiment(bytes.NewBufferString(`{
  "formatVersion":"umpire3/v1",
  "experimentID":"unknown-field",
  "unexpected":true
}`), DefaultDecodeLimit)
	require.ErrorContains(t, err, "unknown field")
}

func TestDecodeExperimentRejectsOversizedInput(t *testing.T) {
	_, err := DecodeExperiment(bytes.NewBufferString(`{"formatVersion":"umpire3/v1"}`), 8)
	require.ErrorContains(t, err, "exceeds")
}

func TestExperimentRejectsIncompleteTrace(t *testing.T) {
	experiment := Experiment{FormatVersion: FormatVersion, ExperimentID: "incomplete"}
	require.Error(t, experiment.Validate())
}

func TestExperimentRejectsSensitiveActionData(t *testing.T) {
	encoded, err := os.ReadFile("../testdata/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := DecodeExperiment(bytes.NewReader(encoded), DefaultDecodeLimit)
	require.NoError(t, err)
	experiment.Actions[0].Arguments = map[string]string{"authorizationHeader": "secret"}
	require.ErrorContains(t, experiment.Validate(), "sensitive")
}

func TestExperimentRejectsUnknownActionVocabulary(t *testing.T) {
	encoded, err := os.ReadFile("../testdata/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := DecodeExperiment(bytes.NewReader(encoded), DefaultDecodeLimit)
	require.NoError(t, err)
	experiment.Actions[0].Kind = "unknown-action"
	require.ErrorContains(t, experiment.Validate(), "unknown action")
}
