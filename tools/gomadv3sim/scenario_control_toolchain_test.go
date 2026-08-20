//go:build gomadv3_toolchain

package gomadv3sim

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScenarioChoicePlanForcesRankBoundDecisionAndExactlyReplays(t *testing.T) {
	boot := uniqueBootID("scenario-control")
	require.NoError(t, RegisterBoot(boot, func(context.Context, NodeContext) error { return nil }))
	plan, err := NewScenarioChoicePlan([]ScenarioChoiceOverride{{
		Ordinal: 0, ID: "route", Occurrence: 1, Alternatives: []string{"alpha", "beta"}, Selected: 0,
	}})
	require.NoError(t, err)
	spec := Spec{
		Schema: SpecSchema, Backend: BackendInProcess, Fidelity: FidelitySimulationModel, Seed: 17, Limits: DefaultLimits(),
		Nodes: []NodeSpec{{ID: "server", Boot: boot, Address: "10.0.0.1"}}, ScenarioChoices: &plan,
	}
	selected := ""
	alpha, err := NewScenarioStep("alpha", func(context.Context, Cluster) error { selected = "alpha"; return nil })
	require.NoError(t, err)
	beta, err := NewScenarioStep("beta", func(context.Context, Cluster) error { selected = "beta"; return nil })
	require.NoError(t, err)
	scenario := Choose("route", alpha, beta)

	result, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, result.Outcome)
	require.Equal(t, "alpha", selected)
	require.Equal(t, plan, result.Record.ScenarioChoices)
	require.EqualValues(t, 0, result.Scenarios[0].Selected)

	replay, err := ReplayPlanFor(result.Record)
	require.NoError(t, err)
	replaySpec := spec
	replaySpec.Replay = &replay
	selected = ""
	replayed, err := Run(context.Background(), replaySpec, scenario)
	require.NoError(t, err)
	require.Equal(t, "alpha", selected)
	require.Equal(t, result.Record.Identity, replayed.Record.Identity)
}

func TestScenarioChoicePlanRejectsChangedDecisionBeforeSelection(t *testing.T) {
	boot := uniqueBootID("scenario-control-divergence")
	require.NoError(t, RegisterBoot(boot, func(context.Context, NodeContext) error { return nil }))
	plan, err := NewScenarioChoicePlan([]ScenarioChoiceOverride{{
		Ordinal: 0, ID: "route", Occurrence: 1, Alternatives: []string{"alpha", "beta"}, Selected: 0,
	}})
	require.NoError(t, err)
	spec := Spec{
		Schema: SpecSchema, Backend: BackendInProcess, Fidelity: FidelitySimulationModel, Seed: 17, Limits: DefaultLimits(),
		Nodes: []NodeSpec{{ID: "server", Boot: boot, Address: "10.0.0.1"}}, ScenarioChoices: &plan,
	}
	selected := false
	alpha, err := NewScenarioStep("alpha", func(context.Context, Cluster) error { selected = true; return nil })
	require.NoError(t, err)
	beta, err := NewScenarioStep("beta", func(context.Context, Cluster) error { selected = true; return nil })
	require.NoError(t, err)

	result, err := Run(context.Background(), spec, Choose("renamed-route", alpha, beta))
	require.NoError(t, err)
	require.Equal(t, OutcomeReplayDiverged, result.Outcome)
	require.False(t, selected)
	require.NotNil(t, result.Divergence)
	require.Equal(t, ReplayDimensionScenario, result.Divergence.Dimension)
	require.NotNil(t, result.Divergence.ExpectedScenario)
	require.Equal(t, "route", result.Divergence.ExpectedScenario.ID)
}
