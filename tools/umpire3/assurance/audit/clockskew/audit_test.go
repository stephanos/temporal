package clockskew

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuditExecutesRestrictedFaultAndRejectsTimestampOrdering(t *testing.T) {
	first, err := RunAudit()
	require.NoError(t, err)
	second, err := RunAudit()
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.NoError(t, first.Validate())
	require.Equal(t, []string{"install", "activate", "release", "cleanup"}, first.FaultLifecycle)
	require.Equal(t, []string{"clock-a", "clock-b"}, first.ClockDomains)
	require.True(t, first.InvertedTimestamps)
	require.True(t, first.CausalOrderAccepted)
	require.True(t, first.TimestampOnlyRejected)
	require.NotEmpty(t, first.ArtifactDigest)
}
