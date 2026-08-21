package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/canary"
)

func TestDecodeApprovalRequiresStrictSealedInput(t *testing.T) {
	approval := canary.Approval{
		FormatVersion: canary.FormatVersion, Identifier: "approval", ApprovalDigest: "sha256:digest",
		Signature: base64.RawStdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize)),
	}
	encoded, err := json.Marshal(approval)
	require.NoError(t, err)
	decoded, err := decodeApproval(encoded)
	require.NoError(t, err)
	require.Equal(t, approval.Identifier, decoded.Identifier)

	unknown := append(encoded[:len(encoded)-1], []byte(`,"unknown":true}`)...)
	_, err = decodeApproval(unknown)
	require.ErrorContains(t, err, "unknown field")

	approval.Signature = ""
	encoded, err = json.Marshal(approval)
	require.NoError(t, err)
	_, err = decodeApproval(encoded)
	require.ErrorContains(t, err, "unsealed")
}

func TestCanaryConfigRequiresKillableWorkerAndRecoveryDirectory(t *testing.T) {
	err := validateConfig(config{})
	require.EqualError(t, err,
		"experiment, approval, approval authority and public key, output, recovery directory, endpoint, namespace, task queue, build, and worker command are required")
}

func TestCanaryFlagsAcceptApprovalAuthorityKey(t *testing.T) {
	_, err := parseFlags([]string{
		"-approval-authority", "release-controller",
		"-approval-public-key", "authority.pem",
	})
	require.NoError(t, err)
}

func TestCanaryWorkerEnvironmentForwardsOnlyTheAPIKey(t *testing.T) {
	t.Setenv("UMPIRE3_TEMPORAL_API_KEY", "approved-credential")
	t.Setenv("UNRELATED_CONTROLLER_SECRET", "must-not-cross-boundary")

	require.Equal(t, []string{"UMPIRE3_TEMPORAL_API_KEY=approved-credential"},
		canaryWorkerEnvironment())
}
