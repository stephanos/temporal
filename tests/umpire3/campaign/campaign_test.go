package campaign

import (
	"context"
	"go/parser"
	"go/token"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	runtime "go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/scenario"
)

func TestCampaignParallelMergeIsDeterministic(t *testing.T) {
	t.Parallel()

	request := Request{
		Candidates: []Candidate{callbackCandidate("second"), callbackCandidate("first")},
		Seed:       42, MaxExecutions: 2, CompilerLimits: compilerLimits(),
		Executor: conformingExecutor,
	}
	request.Workers = 1
	serial, err := Run(context.Background(), request)
	require.NoError(t, err)
	request.Workers = 4
	parallel, err := Run(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, serial, parallel)
}

func TestCampaignRecordsReasonForEveryDroppedCandidate(t *testing.T) {
	t.Parallel()

	report, err := Run(context.Background(), Request{
		Candidates: []Candidate{callbackCandidate("one"), callbackCandidate("two")},
		Seed:       7, Workers: 1, MaxExecutions: 1, CompilerLimits: compilerLimits(),
		Executor: conformingExecutor,
	})
	require.NoError(t, err)
	require.Len(t, report.Executions, 1)
	require.Len(t, report.Dropped, 1)
	require.Equal(t, DropBudget, report.Dropped[0].Reason)
}

func TestViolationPromotionUsesMinimizedNormalRegressionPath(t *testing.T) {
	t.Parallel()

	report, err := Run(context.Background(), Request{
		Candidates: []Candidate{callbackCandidate("violation")},
		Seed:       1, Workers: 1, MaxExecutions: 1, CompilerLimits: compilerLimits(),
		MinimizeAttempts: 8,
		Executor: func(_ context.Context, experiment protocol.Experiment) (runtime.Result, []CoveragePoint, error) {
			digest, err := experiment.Digest()
			require.NoError(t, err)
			result := runtime.Result{
				FormatVersion: runtime.ResultFormatVersion, ExperimentDigest: digest,
				Claim: runtime.Claim{
					Kind: runtime.ClaimViolating, Property: experiment.Property.Identifier,
					Checkpoint: "observe-callback-response-consistent",
				}}
			result.DeriveAssurance()
			return result, nil, nil
		},
	})
	require.NoError(t, err)
	require.Len(t, report.Discoveries, 1)
	discovery := report.Discoveries[0]
	require.True(t, discovery.Minimization.Complete)
	require.Contains(t, discovery.Promotion.Source, "umpire3test.RequireRegression")
	require.Contains(t, discovery.Promotion.Source, "scenario.Action")
	_, err = parser.ParseFile(token.NewFileSet(), "promotion.go", discovery.Promotion.Source, parser.AllErrors)
	require.NoError(t, err)
}

func TestExactlyExhaustedCompletedMinimizationIsComplete(t *testing.T) {
	t.Parallel()

	require.True(t, minimizationComplete(4, 4, true))
	require.False(t, minimizationComplete(4, 4, false))
}

func TestPromotionRetainsTypedBindingsArgumentsAndFaultConfiguration(t *testing.T) {
	t.Parallel()

	file, err := os.Open("../testdata/nexus-cancellation.json")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })
	experiment, err := protocol.DecodeExperiment(file, protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	operation := "operation"
	experiment.Actions[0].Bindings = []protocol.Binding{{
		Symbol: operation, Type: string(protocol.SemanticTypeIDIdentity), Projection: "operation-id",
	}}
	experiment.Actions[2].Arguments = []protocol.NamedValue{{
		Name: "reason", Value: protocol.Value{Type: protocol.ValueString, Text: &operation},
	}}
	source, err := promotionSource(experiment)
	require.NoError(t, err)
	require.Contains(t, source, "scenario.ConfiguredFault")
	require.Contains(t, source, "scenario.Bind")
	require.Contains(t, source, "scenario.WithArgument")
	_, err = parser.ParseFile(token.NewFileSet(), "promotion.go", source, parser.AllErrors)
	require.NoError(t, err)
}

func callbackCandidate(identifier string) Candidate {
	return Candidate{
		Identifier: identifier,
		Scenario: scenario.Scenario{
			Identifier: "callback-" + identifier,
			Target:     protocol.TargetIDProtocolAtomic,
			Resources:  []scenario.Resource{{Identifier: "callback", Kind: protocol.EntityKindCallback}},
			Root: scenario.OnePath(
				scenario.Action("respond", protocol.ActionKindRecordCallbackResponse),
				scenario.Require(protocol.PropertyIDCallbackResponseConsistency),
			),
		},
		Coverage: []CoveragePoint{{Kind: CoverageAction, Identifier: "record-callback-response"}},
	}
}

func compilerLimits() scenario.Limits {
	return scenario.Limits{
		MaxPaths: 2, MaxActions: 8, MaxStates: 32, MaxMemoryBytes: 1 << 20, MaxTime: time.Second,
	}
}

func conformingExecutor(_ context.Context, experiment protocol.Experiment) (runtime.Result, []CoveragePoint, error) {
	return runtime.Result{Claim: runtime.Claim{
		Kind: runtime.ClaimConforming, Property: experiment.Property.Identifier,
	}}, []CoveragePoint{{Kind: CoverageAction, Identifier: experiment.Actions[0].Kind}}, nil
}
