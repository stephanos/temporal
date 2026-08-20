//nolint:revive // The package name is the public Umpire3 runtime.Run seam.
package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestMinimizeActionsRemovesIrrelevantStepsAndPreservesViolation(t *testing.T) {
	experiment := loadExperiment(t)
	experiment.Scope.Bounds.MaxDepth = 10
	experiment.Actions = append(experiment.Actions[:5], append([]protocol.Action{
		{Identifier: "irrelevant-crash", Kind: "crash-owner", RequiredCapabilities: []string{"failover-control"}},
		{Identifier: "irrelevant-recover", Kind: "recover-owner", RequiredCapabilities: []string{"failover-control"}},
	}, experiment.Actions[5:]...)...)

	minimized, err := MinimizeActions(context.Background(), experiment, func(_ context.Context, candidate protocol.Experiment) (Result, error) {
		kinds := make(map[string]bool, len(candidate.Actions))
		for _, action := range candidate.Actions {
			kinds[action.Kind] = true
		}
		claim := ClaimInconclusive
		if kinds["commit-cancellation"] && kinds["worker-returns-success"] && kinds["persist-success"] {
			claim = ClaimViolating
		}
		return Result{Claim: Claim{Kind: claim, Property: candidate.Property.Identifier}}, nil
	})
	require.NoError(t, err)
	require.Len(t, minimized.Actions, 3)
	require.Equal(t, []string{"commit-cancellation", "worker-returns-success", "persist-success"}, []string{
		minimized.Actions[0].Kind,
		minimized.Actions[1].Kind,
		minimized.Actions[2].Kind,
	})
}

func TestMinimizeActionsRejectsDifferentPropertyViolation(t *testing.T) {
	experiment := loadExperiment(t)
	_, err := MinimizeActions(context.Background(), experiment, func(context.Context, protocol.Experiment) (Result, error) {
		return Result{Claim: Claim{Kind: ClaimViolating, Property: "different.property"}}, nil
	})
	require.ErrorContains(t, err, "original experiment")
}

func TestMinimizeExperimentRemovesUnusedResourcesFaultsAndBindings(t *testing.T) {
	experiment := loadExperiment(t)
	experiment.Scope.Bounds.MaxDepth++
	experiment.Actions = append(experiment.Actions, protocol.Action{
		Identifier: "unused-fault", Kind: "crash-owner", RequiredCapabilities: []string{"failover-control"},
		Arguments: map[string]string{"unused": "value"}, Bindings: map[string]string{"unused": "resource"},
	})
	experiment.Resources = append(experiment.Resources, protocol.Resource{Identifier: "unused", Kind: "worker"})

	minimized, err := MinimizeExperiment(context.Background(), experiment,
		func(_ context.Context, candidate protocol.Experiment) (Result, error) {
			return Result{Claim: Claim{
				Kind: ClaimViolating, Property: candidate.Property.Identifier, Checkpoint: "no-stale-success",
			}}, nil
		})
	require.NoError(t, err)
	require.Len(t, minimized.Actions, 1)
	require.Len(t, minimized.Resources, 1)
	require.Empty(t, minimized.Actions[0].Arguments)
	require.Empty(t, minimized.Actions[0].Bindings)
}
