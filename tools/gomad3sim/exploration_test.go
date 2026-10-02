package gomad3sim

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// The Runner issues exploration plans and reads back exploration decisions,
// but lives in the nested tools/gomad3 module and shares no code with this
// package. Its retained vector is the one place both sides are checked against
// the same bytes.
func TestExplorationIdentitiesMatchRunnerIssuedPlanVector(t *testing.T) {
	encoded, err := os.ReadFile(filepath.Join("..", "gomad3", "runner", "testdata", "simulation_exploration_plan_vector.json"))
	require.NoError(t, err)
	var vector struct {
		ExecutionSHA256  string              `json:"execution_sha256"`
		ControllerSHA256 string              `json:"controller_sha256"`
		BaseSeed         uint64              `json:"base_seed"`
		Decision         ExplorationDecision `json:"decision"`
		Forced           uint32              `json:"forced"`
		RootPlan         string              `json:"root_plan"`
		ForcedPlan       string              `json:"forced_plan"`
	}
	require.NoError(t, json.Unmarshal(encoded, &vector))

	site, alternatives, err := scenarioExplorationIdentities("route", 1, []string{"alpha", "beta"})
	require.NoError(t, err)
	decision, err := newExplorationDecision(ExplorationScenario, 0, site, alternatives, 0)
	require.NoError(t, err)
	require.Equal(t, vector.Decision, decision)

	root, err := NewExplorationPlan(vector.ExecutionSHA256, vector.ControllerSHA256, vector.BaseSeed, nil)
	require.NoError(t, err)
	override, err := NewExplorationOverride(ExplorationScenario, 0, site, alternatives, vector.Forced)
	require.NoError(t, err)
	forced, err := NewExplorationPlan(vector.ExecutionSHA256, vector.ControllerSHA256, vector.BaseSeed, []ExplorationOverride{override})
	require.NoError(t, err)
	for _, test := range []struct {
		name   string
		plan   ExplorationPlan
		issued string
	}{{name: "root", plan: root, issued: vector.RootPlan}, {name: "forced", plan: forced, issued: vector.ForcedPlan}} {
		t.Run(test.name, func(t *testing.T) {
			computed, err := EncodeExplorationPlan(test.plan)
			require.NoError(t, err)
			require.Equal(t, test.issued, string(computed))
			decoded, err := DecodeExplorationPlan([]byte(test.issued))
			require.NoError(t, err)
			require.Equal(t, test.plan, decoded)
		})
	}
}

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

func TestExplorationEvidenceProvesOnlyTheOverridesTheClusterDecides(t *testing.T) {
	runtimeOverride, err := NewExplorationOverride(ExplorationRuntime, 0, rawSHA256([]byte("runtime site")), []string{rawSHA256([]byte("first rank")), rawSHA256([]byte("second rank"))}, 1)
	require.NoError(t, err)
	site, alternatives, err := scenarioExplorationIdentities("route", 1, []string{"alpha", "beta"})
	require.NoError(t, err)
	scenarioOverride, err := NewExplorationOverride(ExplorationScenario, 0, site, alternatives, 1)
	require.NoError(t, err)
	plan, err := NewExplorationPlan(rawSHA256([]byte("execution")), rawSHA256([]byte("controller")), 17, []ExplorationOverride{runtimeOverride, scenarioOverride})
	require.NoError(t, err)
	forced, err := newExplorationDecision(ExplorationScenario, 0, site, alternatives, 1)
	require.NoError(t, err)
	observed, err := newExplorationDecision(ExplorationScenario, 0, site, alternatives, 0)
	require.NoError(t, err)

	require.NoError(t, validateExplorationEvidence(&plan, []ExplorationDecision{forced}, DefaultLimits()))
	require.ErrorContains(t, validateExplorationEvidence(&plan, []ExplorationDecision{observed}, DefaultLimits()), "does not prove a forced decision")
	require.ErrorContains(t, validateExplorationEvidence(&plan, nil, DefaultLimits()), "does not prove a forced decision")
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
