package assessment

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/umpire/evaluation"
)

// productionCanaryIdentity is the identity Temporal/Evaluation/CanaryTests.lean pins.
const productionCanaryIdentity = "sha256:3da213bcca87cf29ab7b1e9bcb264aa5f10f850008b6a94949200e484c1944a5"

func TestTheCanaryProfileIsLeansAndOnlyTheCanarys(t *testing.T) {
	committed, err := policy.Embedded()
	require.NoError(t, err)
	profile, err := LoadProfile(committed.EvaluationProfile)
	require.NoError(t, err)
	require.Equal(t, productionCanaryIdentity, profile.Identity)
	require.Equal(t, "dedicated-production-canary", profile.Trust)
	require.Equal(t, []string{"capability", "input", "interpretation", "claim"}, profile.BlockingKnownGaps)

	for _, name := range []string{"canary-harness", "local-ephemeral", "../profiles/production-canary", ""} {
		_, err := LoadProfile(name)
		require.ErrorIs(t, err, evaluation.ErrUnknownProfile, name)
	}
	// umpire-assess cannot select the canary's Profile.
	_, err = evaluation.LoadProfile("production-canary")
	require.ErrorIs(t, err, evaluation.ErrUnknownProfile)
}
