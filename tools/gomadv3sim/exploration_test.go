package gomadv3sim

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExplorationPlanCanonicalRoundTripAndDetachedInput(t *testing.T) {
	site, alternatives, err := scenarioExplorationIdentities("route", 1, []string{"alpha", "beta"})
	require.NoError(t, err)
	override, err := NewExplorationOverride(ExplorationScenario, 0, site, alternatives, 1)
	require.NoError(t, err)
	plan, err := NewExplorationPlan(rawSHA256([]byte("execution")), rawSHA256([]byte("controller")), 17, []ExplorationOverride{override})
	require.NoError(t, err)
	wantSelected := plan.Overrides[0].SelectedSHA256
	alternatives[0] = rawSHA256([]byte("changed"))

	encoded, err := EncodeExplorationPlan(plan)
	require.NoError(t, err)
	decoded, err := DecodeExplorationPlan(encoded)
	require.NoError(t, err)
	require.Equal(t, plan, decoded)
	require.Equal(t, wantSelected, decoded.Overrides[0].SelectedSHA256)

	_, err = DecodeExplorationPlan(append(encoded, '\n'))
	require.Error(t, err)
}

func TestExplorationPlanRejectsChangedForcedDecisionIdentity(t *testing.T) {
	site, alternatives, err := scenarioExplorationIdentities("route", 1, []string{"alpha", "beta"})
	require.NoError(t, err)
	override, err := NewExplorationOverride(ExplorationScenario, 0, site, alternatives, 1)
	require.NoError(t, err)
	plan, err := NewExplorationPlan(rawSHA256([]byte("execution")), rawSHA256([]byte("controller")), 17, []ExplorationOverride{override})
	require.NoError(t, err)
	plan.Overrides[0].SelectedSHA256 = alternatives[0]

	_, err = EncodeExplorationPlan(plan)
	require.Error(t, err)
}
