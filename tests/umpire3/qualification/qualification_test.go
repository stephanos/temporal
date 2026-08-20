package qualification

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
	umpire3runtime "go.temporal.io/server/tests/umpire3/runtime"
)

func TestQualifyRejectsIncompleteInput(t *testing.T) {
	_, err := Qualify(Request{})
	require.EqualError(t, err, "release, experiment, result, and profile are required")
}

func TestQualificationRejectsSelfAttestedRuntimeAssurance(t *testing.T) {
	encoded, err := os.ReadFile("../testdata/update-lifecycle.json")
	require.NoError(t, err)
	experiment, err := protocol.DecodeExperiment(bytes.NewReader(encoded), protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	digest, err := experiment.Digest()
	require.NoError(t, err)
	result := umpire3runtime.Result{
		FormatVersion: umpire3runtime.ResultFormatVersion, ExperimentDigest: digest,
		ResultClass: protocol.ResultClassFiniteExhaustive, TrustBadge: protocol.TrustBadgeKernel,
		Claim: umpire3runtime.Claim{
			Kind: umpire3runtime.ClaimConforming, Property: experiment.Property.Identifier,
		},
	}
	require.ErrorContains(t, validateResult("remote-deployment", experiment, result), "assurance")
}

func TestQualifyRejectsIncompleteCanaryEnvelope(t *testing.T) {
	_, err := DecodeResult([]byte(`{"formatVersion":"umpire3/canary/v1","runtime":{},"complete":false}`))
	require.ErrorContains(t, err, "canary result is incomplete")
}
