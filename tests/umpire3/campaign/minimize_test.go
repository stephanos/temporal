package campaign

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	umpire3execution "go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestMinimizeActionsRemovesIrrelevantStepsAndPreservesViolation(t *testing.T) {
	experiment := loadMutationExperiment(t)
	experiment.Scope.Bounds.MaxDepth = 10
	experiment.Actions = append(experiment.Actions[:5], append([]protocol.Action{
		{Identifier: "irrelevant-crash", Kind: "crash-owner", RequiredCapabilities: []string{"failover-control"}},
		{Identifier: "irrelevant-recover", Kind: "recover-owner", RequiredCapabilities: []string{"failover-control"}},
	}, experiment.Actions[5:]...)...)

	minimized, err := MinimizeActions(context.Background(), experiment, func(_ context.Context, candidate protocol.Experiment) (umpire3execution.Result, error) {
		kinds := make(map[string]bool, len(candidate.Actions))
		for _, action := range candidate.Actions {
			kinds[action.Kind] = true
		}
		claim := umpire3execution.ClaimInconclusive
		if kinds["commit-cancellation"] && kinds["worker-returns-success"] && kinds["persist-success"] {
			claim = umpire3execution.ClaimViolating
		}
		return umpire3execution.Result{Claim: umpire3execution.Claim{Kind: claim, Property: candidate.Property.Identifier}}, nil
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
	experiment := loadMutationExperiment(t)
	_, err := MinimizeActions(context.Background(), experiment, func(context.Context, protocol.Experiment) (umpire3execution.Result, error) {
		return umpire3execution.Result{Claim: umpire3execution.Claim{Kind: umpire3execution.ClaimViolating, Property: "different.property"}}, nil
	})
	require.ErrorContains(t, err, "original experiment")
}

func TestMinimizeActionsPreservesGroundedViolationIdentity(t *testing.T) {
	experiment := loadMutationExperiment(t)
	fullActionCount := len(experiment.Actions)
	minimized, err := MinimizeActions(context.Background(), experiment,
		func(_ context.Context, candidate protocol.Experiment) (umpire3execution.Result, error) {
			binding := "different-operation"
			if len(candidate.Actions) == fullActionCount {
				binding = "original-operation"
			}
			return umpire3execution.Result{
				Bindings: map[string]string{"operation": binding},
				Claim: umpire3execution.Claim{
					Kind: umpire3execution.ClaimViolating, Property: candidate.Property.Identifier, Checkpoint: "no-stale-success",
				},
			}, nil
		})
	require.NoError(t, err)
	require.Len(t, minimized.Actions, fullActionCount)
}

func TestMinimizeExperimentRemovesUnusedResourcesFaultsAndBindings(t *testing.T) {
	experiment := loadMutationExperiment(t)
	experiment.Scope.Bounds.MaxDepth++
	reason := "unused reason"
	experiment.Actions[2].Arguments = []protocol.NamedValue{{
		Name:  "reason",
		Value: protocol.Value{Type: protocol.ValueString, Text: &reason},
	}}
	experiment.Actions[0].Bindings = []protocol.Binding{{
		Symbol: "unused", Type: "identity", Projection: "operation-id",
	}}
	experiment.Actions = append(experiment.Actions, protocol.Action{
		Identifier: "unused-fault", Kind: "crash-owner", RequiredCapabilities: []string{"failover-control"},
	})
	experiment.Resources = append(experiment.Resources, protocol.Resource{Identifier: "unused", Kind: "nexus-worker"})

	minimized, err := MinimizeExperiment(context.Background(), experiment,
		func(_ context.Context, candidate protocol.Experiment) (umpire3execution.Result, error) {
			kinds := make(map[string]bool, len(candidate.Actions))
			for _, action := range candidate.Actions {
				kinds[action.Kind] = true
			}
			kind := umpire3execution.ClaimInconclusive
			if kinds["request-cancellation"] && kinds["persist-success"] {
				kind = umpire3execution.ClaimViolating
			}
			return umpire3execution.Result{Claim: umpire3execution.Claim{
				Kind: kind, Property: candidate.Property.Identifier, Checkpoint: "no-stale-success",
			}}, nil
		})
	require.NoError(t, err)
	require.Len(t, minimized.Actions, 2)
	require.Len(t, minimized.Resources, 1)
	for _, action := range minimized.Actions {
		require.Empty(t, action.Arguments)
		require.Empty(t, action.Bindings)
	}
}

func TestMinimizeExperimentShrinksOrderPayloadPolicyAndFaultConfiguration(t *testing.T) {
	experiment := loadMutationExperiment(t)
	reason := "not-minimal"
	faultArgument := int64(99)
	listItem := "not-minimal"
	experiment.Actions[2].Arguments = []protocol.NamedValue{{
		Name: "reason", Value: protocol.Value{Type: protocol.ValueString, Text: &reason},
	}}
	experiment.Policies[0].Arguments = []protocol.NamedValue{{
		Name: "commands", Value: protocol.Value{Type: protocol.ValueList, Elements: []protocol.Value{{
			Type: protocol.ValueString, Text: &listItem,
		}}},
	}}
	experiment.Faults[0].Arguments = []protocol.NamedValue{{
		Name: "response-code", Value: protocol.Value{Type: protocol.ValueInteger, Integer: &faultArgument},
	}}
	experiment.Faults[0].Occurrence = protocol.FaultOccurrence{First: 3, Count: 4}
	experiment.Faults[0].Scope.Endpoints = []string{"first", "second"}
	experiment.Faults[0].Scope.Attempts = []int{4, 7}

	minimized, err := MinimizeExperiment(context.Background(), experiment,
		func(_ context.Context, candidate protocol.Experiment) (umpire3execution.Result, error) {
			kind := umpire3execution.ClaimInconclusive
			if len(candidate.Actions) == len(experiment.Actions) &&
				len(candidate.Actions[2].Arguments) == 1 &&
				len(candidate.Policies) == 1 && len(candidate.Policies[0].Arguments) == 1 &&
				len(candidate.Faults) == 1 && len(candidate.Faults[0].Arguments) == 1 {
				kind = umpire3execution.ClaimViolating
			}
			return umpire3execution.Result{Claim: umpire3execution.Claim{
				Kind: kind, Property: candidate.Property.Identifier, Checkpoint: "same-violation",
			}}, nil
		})
	require.NoError(t, err)
	require.Empty(t, minimized.Order)
	require.Empty(t, *minimized.Actions[2].Arguments[0].Value.Text)
	require.Empty(t, minimized.Policies[0].Arguments[0].Value.Elements)
	require.EqualValues(t, 0, *minimized.Faults[0].Arguments[0].Value.Integer)
	require.Equal(t, protocol.FaultOccurrence{First: 1, Count: 1}, minimized.Faults[0].Occurrence)
	require.Equal(t, []string{"operation"}, minimized.Faults[0].Scope.Resources)
	require.Equal(t, []string{"first"}, minimized.Faults[0].Scope.Endpoints)
	require.Equal(t, []int{4}, minimized.Faults[0].Scope.Attempts)
}
