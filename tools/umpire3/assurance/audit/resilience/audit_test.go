package resilience

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

func TestRunAuditMatchesRetainedControlPlaneEvidence(t *testing.T) {
	fresh, err := RunAudit(context.Background())
	require.NoError(t, err)
	retained, err := DefaultAudit()
	require.NoError(t, err)
	require.Equal(t, retained, fresh)
}

func TestAuditFailsClosedWhenARequiredCheckIsRelabeled(t *testing.T) {
	report, err := DefaultAudit()
	require.NoError(t, err)
	report.WorkerCrashRecovered = false
	report.ArtifactDigest = report.computedDigest()
	require.ErrorContains(t, report.Validate(), "requires every")

	encoded, err := report.CanonicalJSON()
	require.Error(t, err)
	require.Nil(t, encoded)

	_, err = DecodeAudit(append(defaultAuditJSON, []byte(` {}`)...), protocolexperiment.DefaultDecodeLimit)
	require.ErrorContains(t, err, "one JSON document")
}
