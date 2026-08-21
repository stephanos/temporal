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

func TestExplorationForcedMismatchProducesValidDivergenceEvidence(t *testing.T) {
	site, alternatives, err := scenarioExplorationIdentities("route", 1, []string{"alpha", "beta"})
	require.NoError(t, err)
	override, err := NewExplorationOverride(ExplorationScenario, 0, site, alternatives, 1)
	require.NoError(t, err)
	observed, err := newExplorationDecision(ExplorationScenario, 0, site, alternatives, 0)
	require.NoError(t, err)

	divergenceErr := (&inProcessCluster{}).explorationDivergenceLocked(override, observed)
	var replayErr *ReplayDivergenceError
	require.ErrorAs(t, divergenceErr, &replayErr)
	require.NoError(t, validateDivergence(replayErr.Divergence))
	require.Equal(t, ReplayDimensionExploration, replayErr.Divergence.Dimension)
	require.Equal(t, &override, replayErr.Divergence.ExpectedExplorationOverride)
	require.Equal(t, &observed, replayErr.Divergence.ActualExploration)
}

func TestNonExplorationDivergenceRejectsExplorationEvidence(t *testing.T) {
	site, alternatives, err := scenarioExplorationIdentities("route", 1, []string{"alpha", "beta"})
	require.NoError(t, err)
	expected, err := newExplorationDecision(ExplorationScenario, 0, site, alternatives, 0)
	require.NoError(t, err)
	actual, err := newExplorationDecision(ExplorationScenario, 0, site, alternatives, 1)
	require.NoError(t, err)

	err = validateDivergence(ReplayDivergence{
		Dimension: ReplayDimensionEvidence, Ordinal: 0,
		ExpectedSHA256: expected.Identity, ActualSHA256: actual.Identity,
		ExpectedExploration: &expected, ActualExploration: &actual,
	})
	require.ErrorContains(t, err, "evidence replay divergence")
}

func TestRuntimeExplorationOverrideIsCompletedByHostController(t *testing.T) {
	site := rawSHA256([]byte("runtime site"))
	alternatives := []string{rawSHA256([]byte("first rank")), rawSHA256([]byte("second rank"))}
	override, err := NewExplorationOverride(ExplorationRuntime, 0, site, alternatives, 1)
	require.NoError(t, err)
	plan, err := NewExplorationPlan(rawSHA256([]byte("execution")), rawSHA256([]byte("controller")), 17, []ExplorationOverride{override})
	require.NoError(t, err)
	cluster := &inProcessCluster{explorationPlan: &plan, explorationConsumed: make(map[ExplorationDimension]uint64)}

	require.NoError(t, cluster.finishExplorationLocked())
}
