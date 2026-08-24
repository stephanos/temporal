package mutation

import (
	"bytes"
	"context"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire3/checker/finite"
	checkertrace "go.temporal.io/server/tools/umpire3/checker/trace"
	runtime "go.temporal.io/server/tools/umpire3/execution"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tools/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
	"go.temporal.io/server/tools/umpire3/scenario"
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

func TestCampaignDerivesCoverageFromTheSelectedModel(t *testing.T) {
	t.Parallel()

	candidate := callbackCandidate("derived")
	candidate.Coverage = nil
	report, err := Run(context.Background(), Request{
		Candidates: []Candidate{candidate}, Seed: 7, Workers: 1, MaxExecutions: 1,
		CompilerLimits: compilerLimits(), Executor: conformingExecutor,
	})
	require.NoError(t, err)
	require.Contains(t, report.CoverageAfter, CoveragePoint{
		Kind: CoverageTransition, Identifier: "protocol-atomic/record-callback-response",
	})
	require.Contains(t, report.CoverageAfter, CoveragePoint{
		Kind: CoverageProperty, Identifier: "protocol-atomic/callback.response-consistency",
	})
	require.Contains(t, report.CoverageAfter, CoveragePoint{
		Kind: CoverageObservation, Identifier: "protocol-atomic/observation.callback-response-consistent",
	})
}

func TestMutationCampaignPrioritizesNovelMutationSiteWithinExecutionBudget(t *testing.T) {
	t.Parallel()

	experiment := loadMutationExperiment(t)
	baseline := "baseline"
	experiment.Actions[2].Arguments = []protocolexperiment.NamedValue{{
		Name: "reason", Value: protocolexperiment.Value{Type: protocolexperiment.ValueString, Text: &baseline},
	}}
	seeded := "seeded-adapter-corruption"
	mutation := MutationRequest{
		Experiment: experiment, MaxCandidates: 32,
		Values:        []protocolexperiment.Value{{Type: protocolexperiment.ValueString, Text: &seeded}},
		TopologyKinds: []protocolcatalog.EntityKind{protocolcatalog.EntityKindCallback},
	}
	for seed := int64(1); seed <= 100; seed++ {
		mutation.Seed = seed
		generated, err := Mutate(mutation)
		require.NoError(t, err)
		if generated.Selected[0].Kind != MutationProtobufValue ||
			generated.Selected[0].Path != "actions[2].arguments[reason]" {
			break
		}
	}
	require.NotZero(t, mutation.Seed)

	covered, err := modelCoverage(experiment)
	require.NoError(t, err)
	covered = append(covered,
		CoveragePoint{Kind: CoverageKind("parameter"), Identifier: "scope.seed"},
		CoveragePoint{Kind: CoverageFault, Identifier: "faults[0].occurrence"},
		CoveragePoint{Kind: CoverageTopology, Identifier: "resources"},
	)
	found := false
	report, err := Run(context.Background(), Request{
		Mutation: &mutation, Seed: mutation.Seed, Workers: 1, MaxExecutions: 1,
		CorpusCoverage: covered,
		Executor: func(_ context.Context, candidate protocolexperiment.Experiment) (runtime.Result, []CoveragePoint, error) {
			found = hasArgument(candidate, "reason", seeded)
			return conformingExecutor(context.Background(), candidate)
		},
	})
	require.NoError(t, err)
	require.True(t, found)
	require.Len(t, report.Executions, 1)
	require.Equal(t, MutationProtobufValue, report.Executions[0].Mutation)
	require.Equal(t, "actions[2].arguments[reason]", report.Executions[0].Path)
}

func TestViolationPromotionUsesMinimizedNormalRegressionPath(t *testing.T) {
	t.Parallel()

	report, err := Run(context.Background(), Request{
		Candidates: []Candidate{callbackCandidate("violation")},
		Seed:       1, Workers: 1, MaxExecutions: 1, CompilerLimits: compilerLimits(),
		MinimizeAttempts: 8,
		Executor: func(_ context.Context, experiment protocolexperiment.Experiment) (runtime.Result, []CoveragePoint, error) {
			digest, err := experiment.Digest()
			require.NoError(t, err)
			result := runtime.Result{
				FormatVersion: runtime.ResultFormatVersion, ExperimentDigest: digest,
				Claim: runtime.Claim{
					Kind: runtime.ClaimViolating, Property: experiment.Property.Identifier,
					Checkpoint: "observe-callback-response-consistent",
				}}
			require.NoError(t, bindAcceptedSemanticTrace(experiment, &result))
			result.DeriveAssurance()
			return result, nil, nil
		},
	})
	require.NoError(t, err)
	require.Len(t, report.Discoveries, 1)
	discovery := report.Discoveries[0]
	require.True(t, discovery.Minimization.Complete)
	require.Contains(t, discovery.Promotion.Source, "regression.RequireRegression")
	require.Contains(t, discovery.Promotion.Source, "scenario.ProtocolAtomicScenario")
	require.Contains(t, discovery.Promotion.Source, "scenario.RecordCallbackResponse")
	require.NotContains(t, discovery.Promotion.Source, "tools/umpire3/protocol")
	_, err = parser.ParseFile(token.NewFileSet(), "promotion.go", discovery.Promotion.Source, parser.AllErrors)
	require.NoError(t, err)
	requirePromotionCompiles(t, discovery.Promotion.Source)
}

func TestExactlyExhaustedCompletedMinimizationIsComplete(t *testing.T) {
	t.Parallel()

	require.True(t, minimizationComplete(4, 4, true))
	require.False(t, minimizationComplete(4, 4, false))
}

func TestPromotionRetainsTypedBindingsArgumentsAndFaultConfiguration(t *testing.T) {
	t.Parallel()

	file, err := os.Open("../testdata/generated/nexus-cancellation.json")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })
	experiment, err := protocolexperiment.DecodeExperiment(file, protocolexperiment.DefaultDecodeLimit)
	require.NoError(t, err)
	operation := "operation"
	experiment.Actions[0].Bindings = []protocolexperiment.Binding{{
		Symbol: operation, Type: string(protocolcatalog.SemanticTypeIDIdentity), Projection: "operation-id",
	}}
	experiment.Actions[2].Arguments = []protocolexperiment.NamedValue{{
		Name: "reason", Value: protocolexperiment.Value{Type: protocolexperiment.ValueString, Text: &operation},
	}}
	source, err := promotionSource(experiment)
	require.NoError(t, err)
	require.Contains(t, source, "scenario.NexusCancellationScenario")
	require.Contains(t, source, "scenario.StaleWorkerCompletion")
	require.Contains(t, source, "scenario.BindIdentity")
	require.Contains(t, source, "scenario.WithReason")
	require.NotContains(t, source, "scenario.ConfiguredFault")
	require.NotContains(t, source, "tools/umpire3/protocol")
	_, err = parser.ParseFile(token.NewFileSet(), "promotion.go", source, parser.AllErrors)
	require.NoError(t, err)
	requirePromotionCompiles(t, source)
}

func requirePromotionCompiles(t *testing.T, source string) {
	t.Helper()
	directory, err := os.MkdirTemp(".", ".promotion-")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, os.RemoveAll(directory)) })
	require.NoError(t, os.WriteFile(filepath.Join(directory, "promotion.go"), []byte(source), 0o600))
	command := exec.Command("go", "test", "-tags", "test_dep", ".")
	command.Dir = directory
	output, err := command.CombinedOutput()
	require.NoError(t, err, strings.TrimSpace(string(output)))
}

func TestCheckedBackendTraceUsesCampaignAndNormalRegressionPromotion(t *testing.T) {
	t.Parallel()

	encoded, err := os.ReadFile("../checker/veil/testdata/retained/nexus-cancellation-mutated-concrete.json")
	require.NoError(t, err)
	backend, err := protocolchecker.DecodeBackendResult(bytes.NewReader(encoded), protocolexperiment.DefaultDecodeLimit)
	require.NoError(t, err)
	trace, err := checkertrace.FromBackendResult(backend)
	require.NoError(t, err)
	report, err := Run(context.Background(), Request{
		Traces: []protocolchecker.SemanticTrace{trace},
		Seed:   1, Workers: 1, MaxExecutions: 1, CompilerLimits: compilerLimits(),
		MinimizeAttempts: 128,
		Executor: func(_ context.Context, experiment protocolexperiment.Experiment) (runtime.Result, []CoveragePoint, error) {
			digest, digestErr := experiment.Digest()
			require.NoError(t, digestErr)
			violating := false
			if len(experiment.Faults) != 0 {
				kinds := make(map[string]struct{}, len(experiment.Actions))
				for _, action := range experiment.Actions {
					kinds[action.Kind] = struct{}{}
				}
				_, hasOwnership := kinds[string(protocolcatalog.ActionKindAcquireOwnership)]
				_, hasReturn := kinds[string(protocolcatalog.ActionKindWorkerReturnsSuccess)]
				_, hasPersist := kinds[string(protocolcatalog.ActionKindPersistSuccess)]
				violating = hasOwnership && hasReturn && hasPersist
			}
			claimKind := runtime.ClaimConforming
			if violating {
				claimKind = runtime.ClaimViolating
			}
			result := runtime.Result{
				FormatVersion:    runtime.ResultFormatVersion,
				ExperimentDigest: digest,
				Claim: runtime.Claim{Kind: claimKind,
					Property:   experiment.Property.Identifier,
					Checkpoint: "observe-stale-success-absent"},
			}
			if violating {
				if traceErr := bindAcceptedSemanticTrace(experiment, &result); traceErr != nil {
					result.Claim.Kind = runtime.ClaimConforming
				}
			}
			result.DeriveAssurance()
			return result, nil, nil
		},
	})
	require.NoError(t, err)
	require.Len(t, report.Discoveries, 1)
	require.Empty(t, report.Discoveries[0].PromotionBlock)
	require.Contains(t, report.Discoveries[0].Promotion.Source, "regression.RequireRegression")
	require.Contains(t, report.Discoveries[0].Promotion.Source, "scenario.StaleWorkerCompletion")
	require.NotContains(t, report.Discoveries[0].Promotion.Source, "tools/umpire3/protocol")
	_, err = parser.ParseFile(token.NewFileSet(), "promotion.go",
		report.Discoveries[0].Promotion.Source, parser.AllErrors)
	require.NoError(t, err)
	requirePromotionCompiles(t, report.Discoveries[0].Promotion.Source)
}

func TestCheckedCanonicalTraceReceiptUsesCampaignSource(t *testing.T) {
	t.Parallel()

	view, found, err := finite.DefaultFirstOrderView(protocolcatalog.TargetIDNexusCancellation,
		"stale-completion-guard-removed")
	require.NoError(t, err)
	require.True(t, found)
	input := protocolchecker.TraceReplayInput{
		FormatVersion: protocolchecker.TraceReplayInputFormatVersion,
		Target:        view.Target,
		Property:      view.Property,
		World:         view.World,
		Variant:       view.Variant,
		SemanticHash:  view.SemanticHash,
		Actions: []protocolcatalog.ActionKind{
			protocolcatalog.ActionKindDispatchTask,
			protocolcatalog.ActionKindAcquireOwnership,
			protocolcatalog.ActionKindWorkerReturnsSuccess,
			protocolcatalog.ActionKindPersistSuccess,
		},
	}
	digest, err := input.Digest()
	require.NoError(t, err)
	receipt := protocolchecker.TraceReplayReceipt{
		FormatVersion: protocolchecker.TraceReplayReceiptFormatVersion,
		TraceDigest:   digest,
		Target:        input.Target,
		Property:      input.Property,
		World:         input.World,
		Variant:       input.Variant,
		SemanticHash:  input.SemanticHash,
		Actions:       input.Actions,
		Status:        protocolchecker.TraceReplayAccepted,
		TrustBadge:    protocolcatalog.TrustBadgeCheckedCertificate,
		Axioms:        []string{},
	}
	trace, err := checkertrace.FromTraceReplayReceipt(
		protocolchecker.SemanticTraceProducerExact, receipt)
	require.NoError(t, err)
	report, err := Run(context.Background(), Request{
		Traces:         []protocolchecker.SemanticTrace{trace},
		Seed:           1,
		Workers:        1,
		MaxExecutions:  1,
		CompilerLimits: compilerLimits(),
		Executor:       conformingExecutor,
	})
	require.NoError(t, err)
	require.Len(t, report.Executions, 1)
	require.Equal(t, scenario.SemanticTraceIdentifier(trace), report.Executions[0].CandidateID)
}

func TestCheckedTemporalLassoUsesCampaignSource(t *testing.T) {
	t.Parallel()

	view, found, err := protocolchecker.DefaultTemporalView("delivery-fairness-removed")
	require.NoError(t, err)
	require.True(t, found)
	input := protocolchecker.TemporalLassoReplayInput{
		FormatVersion: protocolchecker.TemporalLassoReplayInputFormatVersion,
		Target:        view.Target,
		Property:      view.Property,
		World:         view.World,
		Variant:       view.Variant,
		SemanticHash:  view.SemanticHash,
		Lasso: protocolchecker.TemporalLasso{
			States:    []string{"unavailable", "ready"},
			Actions:   []protocolcatalog.ActionKind{protocolcatalog.ActionKindRecoverOwner, ""},
			LoopStart: 1,
		},
	}
	digest, err := input.Digest()
	require.NoError(t, err)
	receipt := protocolchecker.TemporalLassoReplayReceipt{
		FormatVersion: protocolchecker.TemporalLassoReplayReceiptFormatVersion,
		LassoDigest:   digest,
		Target:        input.Target,
		Property:      input.Property,
		World:         input.World,
		Variant:       input.Variant,
		SemanticHash:  input.SemanticHash,
		Lasso:         input.Lasso,
		Status:        protocolchecker.TraceReplayAccepted,
		TrustBadge:    protocolcatalog.TrustBadgeCheckedCertificate,
		Axioms:        []string{},
	}
	trace, err := checkertrace.FromTemporalLassoReplayReceipt(
		protocolchecker.SemanticTraceProducerLeanTemporal, receipt)
	require.NoError(t, err)
	report, err := Run(context.Background(), Request{
		Traces:         []protocolchecker.SemanticTrace{trace},
		Seed:           1,
		Workers:        1,
		MaxExecutions:  1,
		CompilerLimits: compilerLimits(),
		Executor:       conformingExecutor,
	})
	require.NoError(t, err)
	require.Len(t, report.Executions, 1)
	require.Equal(t, scenario.SemanticTraceIdentifier(trace), report.Executions[0].CandidateID)
}

func TestLiveSemanticTraceCampaignRetainsExactCompiledIntent(t *testing.T) {
	t.Parallel()

	experiment := loadMutationExperiment(t)
	digest, err := experiment.Digest()
	require.NoError(t, err)
	result := runtime.Result{
		FormatVersion: runtime.ResultFormatVersion, ExperimentDigest: digest,
		Claim: runtime.Claim{
			Kind: runtime.ClaimViolating, Property: experiment.Property.Identifier,
		},
	}
	require.NoError(t, bindAcceptedSemanticTrace(experiment, &result))
	require.NotNil(t, result.Trace)
	var executedDigest string
	report, err := Run(context.Background(), Request{
		Traces: []protocolchecker.SemanticTrace{*result.Trace}, Seed: 1, Workers: 1,
		MaxExecutions: 1, CompilerLimits: compilerLimits(),
		Executor: func(_ context.Context, candidate protocolexperiment.Experiment) (runtime.Result, []CoveragePoint, error) {
			var digestErr error
			executedDigest, digestErr = candidate.Digest()
			if digestErr != nil {
				return runtime.Result{}, nil, digestErr
			}
			return conformingExecutor(context.Background(), candidate)
		},
	})
	require.NoError(t, err)
	require.Len(t, report.Executions, 1)
	require.Equal(t, digest, executedDigest)
	require.Equal(t, digest, report.Executions[0].Digest)
}

func callbackCandidate(identifier string) Candidate {
	return Candidate{
		Identifier: identifier,
		Scenario: scenario.Scenario{
			Identifier: "callback-" + identifier,
			Target:     protocolcatalog.TargetIDProtocolAtomic,
			Resources:  []scenario.Resource{{Identifier: "callback", Kind: protocolcatalog.EntityKindCallback}},
			Root: scenario.OnePath(
				scenario.Action("respond", protocolcatalog.ActionKindRecordCallbackResponse),
				scenario.Require(protocolcatalog.PropertyIDCallbackResponseConsistency),
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

func conformingExecutor(_ context.Context, experiment protocolexperiment.Experiment) (runtime.Result, []CoveragePoint, error) {
	return runtime.Result{Claim: runtime.Claim{
		Kind: runtime.ClaimConforming, Property: experiment.Property.Identifier,
	}}, []CoveragePoint{{Kind: CoverageAction, Identifier: experiment.Actions[0].Kind}}, nil
}
