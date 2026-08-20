package campaign

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/environment"
	"go.temporal.io/server/tests/umpire3/evidence"
	"go.temporal.io/server/tests/umpire3/protocol"
	umpire3runtime "go.temporal.io/server/tests/umpire3/runtime"
)

func TestCrossLayerMutationGateDiscoversMinimizesReplaysAndPromotes(t *testing.T) {
	experiment := loadMutationExperiment(t)
	baseline := "baseline"
	experiment.Actions[2].Arguments = []protocol.NamedValue{{
		Name: "reason", Value: protocol.Value{Type: protocol.ValueString, Text: &baseline},
	}}
	seeded := "seeded-adapter-corruption"
	executor := func(_ context.Context, candidate protocol.Experiment) (umpire3runtime.Result, error) {
		digest, err := candidate.Digest()
		require.NoError(t, err)
		result := umpire3runtime.Result{
			FormatVersion: protocol.FormatVersion, ExperimentDigest: digest,
			Environment: umpire3runtime.EnvironmentProfile{
				Name: "local-in-process", Capabilities: []string{"history-observation", "nexus"},
			},
			Claim: umpire3runtime.Claim{
				Kind: umpire3runtime.ClaimConforming, Property: candidate.Property.Identifier,
			},
			Cleanup: environment.CleanupResult{Complete: true},
		}
		if hasArgument(candidate, "reason", seeded) {
			result.Claim.Kind = umpire3runtime.ClaimViolating
			result.Claim.Checkpoint = "observe-cancellation-won"
			result.Evidence = evidence.Graph{
				FormatVersion: evidence.FormatVersion,
				Facts: []evidence.Fact{{
					Identifier: "seeded-cross-layer", Kind: "cancellation-won", Value: false,
					SourceIdentity: "history", ClockDomain: "history-sequence", SourceSequence: 1,
					Reference: "history/1", EntityIdentity: "operation", Lineage: []string{"namespace", "operation"},
				}},
				Claims: []evidence.Claim{{Property: candidate.Property.Identifier, Verdict: "violating"}},
			}
		}
		return result, nil
	}
	request := MutationGateRequest{
		Mutation: MutationRequest{
			Experiment: experiment, Seed: 73, MaxCandidates: 32,
			Values: []protocol.Value{{Type: protocol.ValueString, Text: &seeded}},
		},
		Approved: []ApprovedMutation{{
			Identifier: "adapter-response-corruption-v1", Layer: "temporal-adapter",
			Kind: MutationProtobufValue, Path: "actions[2].arguments[reason]",
		}},
		Executor: executor,
	}
	first, err := RunMutationGate(context.Background(), request)
	require.NoError(t, err)
	second, err := RunMutationGate(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, "adapter-response-corruption-v1", first.Discovered.Identifier)
	require.True(t, first.Replay.Reproduced)
	require.Contains(t, first.PromotionSource, "umpire3test.RequireRegression")
	require.NotEmpty(t, first.ReplayBundleDigest)
	require.LessOrEqual(t, len(first.Minimized.Actions), len(experiment.Actions))
}

func hasArgument(experiment protocol.Experiment, name, expected string) bool {
	for _, action := range experiment.Actions {
		for _, argument := range action.Arguments {
			if argument.Name == name && argument.Value.Text != nil && *argument.Value.Text == expected {
				return true
			}
		}
	}
	return false
}

func TestMutationGateRejectsUnapprovedViolation(t *testing.T) {
	experiment := loadMutationExperiment(t)
	_, err := RunMutationGate(context.Background(), MutationGateRequest{
		Mutation: MutationRequest{Experiment: experiment, Seed: 1, MaxCandidates: 8},
		Approved: []ApprovedMutation{{Identifier: "approved", Layer: "model", Kind: MutationSchedule, Path: "other"}},
		Executor: func(_ context.Context, candidate protocol.Experiment) (umpire3runtime.Result, error) {
			digest, digestErr := candidate.Digest()
			require.NoError(t, digestErr)
			return umpire3runtime.Result{ExperimentDigest: digest, Claim: umpire3runtime.Claim{
				Kind: umpire3runtime.ClaimViolating, Property: candidate.Property.Identifier,
			}}, nil
		},
	})
	require.ErrorContains(t, err, "unapproved mutation")
}
