package assessment

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/testpilot/evaluation"
	"go.temporal.io/server/tools/canary/policy"
)

// productionCanaryIdentity is the canary profile identity this test pins; changing it is a deliberate
// profile change.
const productionCanaryIdentity = "sha256:6265dc08b1e426948cd6e619e7ea830a57a0ca1e3a351b3fdf910e1cb721263c"

func TestTheCanaryProfileIsTheCanarysOwn(t *testing.T) {
	committed, err := policy.Embedded()
	require.NoError(t, err)
	profile, err := LoadProfile(committed.EvaluationProfile)
	require.NoError(t, err)
	require.Equal(t, productionCanaryIdentity, profile.Identity)
	require.Equal(t, "dedicated-production-canary", profile.Trust)
	require.Equal(t, []string{"capability", "input", "interpretation", "claim"}, profile.BlockingKnownGaps)
	require.Equal(t, evaluation.DecisionRejected, profile.UnsupportedRule)

	for _, name := range []string{"canary-harness", "local-ephemeral", "../profiles/production-canary", ""} {
		_, err := LoadProfile(name)
		require.ErrorIs(t, err, evaluation.ErrUnknownProfile, name)
	}
	// umpire-assess cannot select the canary's Profile.
	_, err = evaluation.LoadProfile("production-canary")
	require.ErrorIs(t, err, evaluation.ErrUnknownProfile)
}
