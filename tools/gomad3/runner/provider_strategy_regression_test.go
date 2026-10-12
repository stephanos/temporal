package runner

import (
	"context"
	"testing"

	"go.temporal.io/server/tools/gomad3/artifact"
	backend "go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/record"
	simulationengine "go.temporal.io/server/tools/gomad3/runner/internal/exploration/simulation"
	simulationrecord "go.temporal.io/server/tools/gomad3/runner/internal/exploration/simulationrecord"
	"go.temporal.io/server/tools/gomad3/target"
)

type staticDescriptionMinimizationProvider struct{ *prefixFailureProvider }

func (provider *staticDescriptionMinimizationProvider) Describe(ctx context.Context, config backend.Configuration) (backend.Description, error) {
	description, err := provider.prefixFailureProvider.Describe(ctx, config)
	description.IOProfile.ImplementationSHA256 = record.HashBytes([]byte("static backend implementation"))
	return description, err
}

func TestExternalMinimizeUsesProvenanceProfileInsteadOfStaticDescription(t *testing.T) {
	parent, execution := backendMinimizationParent(t)
	provider := &staticDescriptionMinimizationProvider{prefixFailureProvider: execution}
	minimized, err := Minimize(t.Context(), MinimizeSpec{Backend: provider, ArtifactPath: parent.Path, OutputRoot: t.TempDir(), AttemptBudget: 8})
	if err != nil {
		t.Fatal(err)
	}
	if !minimized.Changed || minimized.Attempts != 2 || minimized.Artifact.Manifest.IOProfile.ImplementationSHA256 != parent.Manifest.IOProfile.ImplementationSHA256 || minimized.Artifact.Manifest.Minimization.FinalForcedDecisions != 1 {
		t.Fatalf("provenance-bound minimization = %+v", minimized)
	}
	replayed, err := Replay(t.Context(), ReplaySpec{Backend: provider, ArtifactPath: minimized.Artifact.Path})
	if err != nil || !replayed.Match || replayed.ChoiceReplayStatus != ChoiceReplayExact {
		t.Fatalf("external reduced exact replay = %+v, %v", replayed, err)
	}
}

type strategyRegressionProvider struct {
	backend.Provider
	preparer   *fakePreparer
	executor   executionRunner
	transcript backend.Transcript
}

func (provider *strategyRegressionProvider) Describe(ctx context.Context, config backend.Configuration) (backend.Description, error) {
	config.ForExecution = false
	return provider.Provider.Describe(ctx, config)
}

func (provider *strategyRegressionProvider) Prepare(ctx context.Context, spec target.Spec) (target.Prepared, error) {
	return provider.preparer.Prepare(ctx, spec)
}

func (provider *strategyRegressionProvider) ValidateReplay(ctx context.Context, _ *artifact.Opened) (target.Prepared, error) {
	if err := ctx.Err(); err != nil {
		return target.Prepared{}, err
	}
	return provider.preparer.prepared.CloneBackend(), nil
}

func (provider *strategyRegressionProvider) Execute(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	result, err := provider.executor.Run(ctx, request)
	if err == nil {
		result.IOTranscript = provider.transcript
	}
	return result, err
}

func TestExplorationStrategiesKeepExplicitProvider(t *testing.T) {
	for _, strategy := range []Strategy{StrategyChoiceExploration, StrategySimulationExploration} {
		t.Run(string(strategy), func(t *testing.T) {
			preparer := newFakePreparer(t)
			limit := choiceTraceLimit(t, 1)
			provider := &strategyRegressionProvider{Provider: scriptProvider(), preparer: preparer, transcript: semanticTranscript(t, "stdlib.os.openfile")}
			config, _ := testConfig(t, nil, nil, "7", PolicyAll, 1)
			config.Backend, config.Strategy, config.ChoiceTraceLimit = provider, strategy, limit
			config.MaxExecutions, config.MaxExplorationBytes = 4, 1<<20
			if strategy == StrategyChoiceExploration {
				config.MaxChoiceDepth = 2
				provider.executor = &explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit}
			} else {
				config.MaxForcedDecisions, config.MaxExplorationResultBytes = 2, 1<<20
				config.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 2, Scenario: 2, Network: 2, Storage: 2, Fault: 2, Crash: 2}
				provider.executor = &simulationExplorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit}
			}
			summary, err := Explore(t.Context(), config)
			if err != nil {
				t.Fatal(err)
			}
			if summary.Attempted != 2 || summary.Succeeded != 2 || summary.StopReason != StopExplorationExhausted {
				t.Fatalf("provider exploration = %+v", summary)
			}
			inspected, err := Inspect(summary.CampaignPath, InspectOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if len(inspected.Campaign.Executions) != 2 || inspected.Campaign.Executions[1].ParentCandidateSHA256 == "" {
				t.Fatalf("forced provider candidate was not committed: %+v", inspected.Campaign.Executions)
			}
		})
	}
}

func TestSimulationMinimizeKeepsProviderProfileAndExactReplay(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	provider := &strategyRegressionProvider{Provider: scriptProvider(), preparer: preparer,
		transcript: semanticTranscript(t, "stdlib.os.openfile"),
		executor:   &simulationExplorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit, fail: true}}
	config, _ := testConfig(t, nil, nil, "7", PolicyAll, 1)
	config.Backend, config.Strategy, config.ChoiceTraceLimit = provider, StrategySimulationExploration, limit
	config.MaxExecutions, config.MaxForcedDecisions = 4, 2
	config.MaxExplorationBytes, config.MaxExplorationResultBytes = 1<<20, 1<<20
	config.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 2, Scenario: 2, Network: 2, Storage: 2, Fault: 2, Crash: 2}
	summary, err := Explore(t.Context(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Attempted != 2 || summary.Failures != 2 || len(summary.Artifacts) != 1 {
		t.Fatalf("provider failure discovery = %+v", summary)
	}
	parent := providerForcedSimulationArtifact(t, provider, summary.Artifacts[0])
	preflight, err := Replay(t.Context(), ReplaySpec{Backend: provider, ArtifactPath: parent, VerifyOnly: true})
	if err != nil || !preflight.Verified || preflight.ChoiceReplayStatus != ChoiceReplayAvailable {
		t.Fatalf("provider minimization preflight = %+v, %v", preflight, err)
	}
	minimized, err := Minimize(t.Context(), MinimizeSpec{Backend: provider, ArtifactPath: parent, OutputRoot: t.TempDir(), AttemptBudget: 8})
	if err != nil {
		t.Fatal(err)
	}
	if !minimized.Changed || minimized.Attempts == 0 || minimized.Artifact.Manifest.IOProfile.Name != "gomad3-deterministic/v1" || minimized.Artifact.Manifest.SimulationProfile == nil {
		t.Fatalf("provider minimization = %+v", minimized)
	}
	replayed, err := Replay(t.Context(), ReplaySpec{Backend: provider, ArtifactPath: minimized.Artifact.Path})
	if err != nil || !replayed.Match || replayed.ChoiceReplayStatus != ChoiceReplayExact {
		t.Fatalf("minimized provider exact replay = %+v, %v", replayed, err)
	}
}

func providerForcedSimulationArtifact(t *testing.T, provider *strategyRegressionProvider, root string) string {
	t.Helper()
	opened, err := artifact.OpenArtifact(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := opened.Close(); err != nil {
			t.Error(err)
		}
	}()
	manifest := opened.Manifest()
	profile := manifest.SimulationProfile
	observedRecord, err := opened.ReadPayload(profile.Record.File, uint64(profile.Record.Bytes))
	if err != nil {
		t.Fatal(err)
	}
	capability, _, err := choiceCapabilityForArtifact(opened)
	if err != nil {
		t.Fatal(err)
	}
	config := simulationengine.Config{ExecutionSHA256: profile.ExecutionSHA256, ControllerSHA256: profile.ControllerSHA256,
		BaseSeed: uint64(manifest.Seed), Parallel: 1, MaxExecutions: 4, MaxForcedDecisions: 2,
		MaxExplorationBytes: 1 << 20, MaxResultBytes: 1 << 20, FailureBudget: 4,
		Limits: simulationengine.DimensionLimits{Runtime: 2, Scenario: 2, Network: 2, Storage: 2, Fault: 2, Crash: 2}}
	rootCandidate, err := simulationengine.CanonicalCandidate(config, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if rootCandidate.SHA256 != profile.CandidateSHA256 {
		t.Fatal("fixture root candidate identity changed")
	}
	rootResult, err := simulationrecord.ResultForRecord(config, rootCandidate, observedRecord, nil)
	if err != nil {
		t.Fatal(err)
	}
	forced, err := simulationengine.ForceDecision(rootResult.Decisions[0], 1)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := simulationengine.CanonicalCandidate(config, []simulationengine.ForcedDecision{forced}, rootCandidate.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	execution, err := simulationrecord.ExecutionForCandidate(config, candidate, capability.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	request := backend.ExecutionRequest{Target: provider.preparer.prepared, Seed: uint64(manifest.Seed),
		Choice: capability, Simulation: &backend.SimulationCapability{Role: backend.SimulationRoleCoordinator,
			ExplorationPlan: execution.SimulationPlan, ExplorationRecordLimit: uint64(profile.Record.Limit), ExplorationRecordCount: 1}}
	observed, err := provider.Execute(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	retained, err := simulationrecord.ProjectArtifact(config, candidate, execution.SimulationPlan, observed.SimulationRecords[0], nil, uint64(profile.Record.Limit))
	if err != nil {
		t.Fatal(err)
	}
	manifest.SimulationProfile = &retained
	world, err := readWorldPayloads(opened)
	if err != nil {
		t.Fatal(err)
	}
	published, err := artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, artifact.ArtifactInput{
		Manifest: manifest, TargetPath: provider.preparer.prepared.Path, World: world,
		Stdout: observed.Stdout.Bytes, Stderr: observed.Stderr.Bytes, IOTranscript: observed.IOTranscript.Bytes, ChoiceTrace: observed.ChoiceTrace.Trace.Bytes,
		Simulation: &artifact.SimulationPayloads{Plan: execution.SimulationPlan, Record: observed.SimulationRecords[0]},
	})
	if err != nil {
		t.Fatal(err)
	}
	return published.Path
}
