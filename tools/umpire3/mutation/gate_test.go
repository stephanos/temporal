package mutation

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire3/execution"
	"go.temporal.io/server/tools/umpire3/execution/evidence"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

func TestCrossLayerMutationGateDiscoversMinimizesReplaysAndPromotes(t *testing.T) {
	experiment := loadMutationExperiment(t)
	baseline := "baseline"
	experiment.Actions[2].Arguments = []protocolexperiment.NamedValue{{
		Name: "reason", Value: protocolexperiment.Value{Type: protocolexperiment.ValueString, Text: &baseline},
	}}
	seeded := "seeded-adapter-corruption"
	executor := func(_ context.Context, candidate protocolexperiment.Experiment) (execution.Result, error) {
		digest, err := candidate.Digest()
		require.NoError(t, err)
		result := execution.Result{
			FormatVersion: execution.ResultFormatVersion, ExperimentDigest: digest,
			Environment: execution.EnvironmentIdentity{
				Name: "local-in-process", Capabilities: []protocolcatalog.CapabilityID{"history-observation", "nexus"},
			},
			Claim: execution.Claim{
				Kind: execution.ClaimConforming, Property: candidate.Property.Identifier,
			},
			Cleanup: execution.CleanupResult{Complete: true},
		}
		if hasArgument(candidate, "reason", seeded) {
			result.Claim.Kind = execution.ClaimViolating
			result.Claim.Checkpoint = "observe-cancellation-won"
			result.Evidence = evidence.Graph{
				FormatVersion: evidence.FormatVersion,
				Facts: []evidence.Fact{{
					Identifier: "seeded-cross-layer", Kind: "cancellation-won", Value: false,
					SourceIdentity: "history", ClockDomain: "history-sequence", SourceSequence: 1,
					ObservedAtUnixNano: 1, Reference: "history/1", EntityIdentity: "operation",
					Lineage: []string{"namespace", "operation"},
				}},
				Claims: []evidence.Claim{{Property: candidate.Property.Identifier, Verdict: "violating"}},
			}
			if traceErr := bindAcceptedSemanticTrace(candidate, &result); traceErr != nil {
				result.Claim.Kind = execution.ClaimInconclusive
				result.Claim.Reason = traceErr.Error()
				result.Evidence = evidence.Graph{}
			}
		}
		result.DeriveAssurance()
		return result, nil
	}
	request := MutationGateRequest{
		Mutation: MutationRequest{
			Experiment: experiment, Seed: 73, MaxCandidates: 32,
			Values: []protocolexperiment.Value{{Type: protocolexperiment.ValueString, Text: &seeded}},
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
	require.Contains(t, first.PromotionSource, "regression.RequireRegression")
	require.NotEmpty(t, first.ReplayBundleDigest)
	require.LessOrEqual(t, len(first.Minimized.Actions), len(experiment.Actions))
	require.Equal(t, protocolcatalog.ResultClassTraceWitness, first.ResultClass)
	require.Equal(t, protocolcatalog.TrustBadgeTestedInstance, first.TrustBadge)
	require.NoError(t, first.Validate(experiment))
	encoded, err := first.CanonicalJSON(experiment)
	require.NoError(t, err)
	decoded, err := DecodeMutationGateReport(encoded, experiment)
	require.NoError(t, err)
	require.Equal(t, first, decoded)

	var mutated map[string]any
	require.NoError(t, json.Unmarshal(encoded, &mutated))
	mutated["promotionSource"] = "package forged"
	encoded, err = json.Marshal(mutated)
	require.NoError(t, err)
	_, err = DecodeMutationGateReport(encoded, experiment)
	require.ErrorContains(t, err, "artifact digest")
}

func TestApprovedMutationAuditIsDeterministicAndSourceBound(t *testing.T) {
	experiment := loadMutationExperiment(t)
	first, err := RunApprovedMutationAudit(context.Background(), experiment)
	require.NoError(t, err)
	second, err := RunApprovedMutationAudit(context.Background(), experiment)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.NoError(t, first.Validate(experiment))
	require.NoError(t, first.ValidateCoverageGuidance())
	require.Equal(t, "adapter-response-corruption-v1", first.Discovered.Identifier)
	require.True(t, first.Replay.Reproduced)
	require.Less(t, first.ExecutionBudget, first.CandidateCount)
	require.Contains(t, first.CoverageDelta, CoveragePoint{
		Kind: CoverageProtobuf, Identifier: "actions[2].arguments[reason]",
	})
}

func TestRetainedApprovedMutationAuditMatchesFreshCampaign(t *testing.T) {
	experiment := loadMutationExperiment(t)
	retained, err := DefaultApprovedMutationAudit(experiment)
	require.NoError(t, err)
	fresh, err := RunApprovedMutationAudit(context.Background(), experiment)
	require.NoError(t, err)
	require.Equal(t, fresh, retained)
}

func hasArgument(experiment protocolexperiment.Experiment, name, expected string) bool {
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
		Executor: func(_ context.Context, candidate protocolexperiment.Experiment) (execution.Result, error) {
			digest, digestErr := candidate.Digest()
			require.NoError(t, digestErr)
			result := execution.Result{
				FormatVersion: execution.ResultFormatVersion, ExperimentDigest: digest, Claim: execution.Claim{
					Kind: execution.ClaimViolating, Property: candidate.Property.Identifier,
				}}
			result.DeriveAssurance()
			return result, nil
		},
	})
	require.ErrorContains(t, err, "unapproved mutation")
}
