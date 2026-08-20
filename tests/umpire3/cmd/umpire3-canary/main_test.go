package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/canary"
)

func TestDecodeApprovalRequiresStrictSealedInput(t *testing.T) {
	approval := canary.Approval{
		FormatVersion: canary.FormatVersion, Identifier: "approval", ApprovalDigest: "sha256:digest",
	}
	encoded, err := json.Marshal(approval)
	require.NoError(t, err)
	decoded, err := decodeApproval(encoded)
	require.NoError(t, err)
	require.Equal(t, approval.Identifier, decoded.Identifier)

	unknown := append(encoded[:len(encoded)-1], []byte(`,"unknown":true}`)...)
	_, err = decodeApproval(unknown)
	require.ErrorContains(t, err, "unknown field")
}

func TestCanaryConfigRequiresKillableWorkerAndRecoveryDirectory(t *testing.T) {
	err := validateConfig(config{})
	require.EqualError(t, err,
		"experiment, approval, output, recovery directory, endpoint, namespace, task queue, build, and worker command are required")
}
