//go:build !gomad3_toolchain

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
// the same bytes. The tagged build runs under a seeded runtime, which replaces
// the host filesystem, so the vector is read only in the untagged build.
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
