package runner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	backend "go.temporal.io/server/tools/gomad3/internal/backends"
	"go.temporal.io/server/tools/gomad3/internal/backends/native"
	"go.temporal.io/server/tools/gomad3/internal/backends/target"
	"go.temporal.io/server/tools/gomad3/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/model/choice"
	"go.temporal.io/server/tools/gomad3/internal/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/internal/runner/internal/exploration"
)

// simulationExplorationPlanVector is the retained exploration plan the Runner
// issues for fixed inputs. tools/gomad3/internal/model/cluster, which lives in another module and
// cannot share code with the Runner, checks the same file from the target
// side, so a plan identity that only one of them computes fails a test.
type simulationExplorationPlanVector struct {
	ExecutionSHA256  record.SHA256        `json:"execution_sha256"`
	ControllerSHA256 record.SHA256        `json:"controller_sha256"`
	BaseSeed         uint64               `json:"base_seed"`
	Decision         exploration.Decision `json:"decision"`
	Forced           uint32               `json:"forced"`
	RootPlan         string               `json:"root_plan"`
	ForcedPlan       string               `json:"forced_plan"`
}

func TestSimulationExplorationPlanMatchesRetainedVector(t *testing.T) {
	encoded, err := os.ReadFile(filepath.Join("testdata", "simulation_exploration_plan_vector.json"))
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var vector simulationExplorationPlanVector
	if err := decoder.Decode(&vector); err != nil {
		t.Fatal(err)
	}
	if vector.ControllerSHA256 != exploration.SimulationImplementationSHA256() {
		t.Fatalf("vector controller = %s, want this implementation %s", vector.ControllerSHA256, exploration.SimulationImplementationSHA256())
	}
	config := exploration.SimulationConfig{
		ExecutionSHA256: vector.ExecutionSHA256, ControllerSHA256: vector.ControllerSHA256, BaseSeed: vector.BaseSeed,
		Parallel: 1, MaxExecutions: 2, MaxForcedDecisions: 1, MaxExplorationBytes: 1 << 20, MaxResultBytes: 1 << 20, FailureBudget: 1,
		Limits: exploration.DimensionLimits{Runtime: 1, Scenario: 1, Network: 1, Storage: 1, Fault: 1, Crash: 1},
	}
	root, err := exploration.CanonicalCandidate(config, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	rootPlan, err := exploration.PlanForCandidate(config, root)
	if err != nil {
		t.Fatal(err)
	}
	if string(rootPlan) != vector.RootPlan {
		t.Errorf("root plan = %s, want %s", rootPlan, vector.RootPlan)
	}

	decision, err := exploration.CanonicalDecision(
		vector.Decision.Dimension, vector.Decision.Ordinal, vector.Decision.SiteSHA256, vector.Decision.Alternatives, vector.Decision.Selected,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decision, vector.Decision) {
		t.Errorf("decision = %#v, want %#v", decision, vector.Decision)
	}
	forced, err := exploration.ForceDecision(decision, vector.Forced)
	if err != nil {
		t.Fatal(err)
	}
	child, err := exploration.CanonicalCandidate(config, []exploration.ForcedDecision{forced}, root.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	forcedPlan, err := exploration.PlanForCandidate(config, child)
	if err != nil {
		t.Fatal(err)
	}
	if string(forcedPlan) != vector.ForcedPlan {
		t.Errorf("forced plan = %s, want %s", forcedPlan, vector.ForcedPlan)
	}
}

func TestSimulationExplorationExecutionIdentityBindsEveryInput(t *testing.T) {
	type inputs struct {
		config               campaignRequest
		prepared             target.Prepared
		profile              backend.Description
		choiceImplementation [sha256.Size]byte
	}
	base := inputs{
		config: campaignRequest{campaignRuntime: campaignRuntime{RunnerBuild: "runner-build"}},
		prepared: target.Prepared{
			SHA256: string(record.HashBytes([]byte("target"))), BuildKey: "toolchain-build", TargetGOOS: "linux", TargetGOARCH: "amd64",
		},
		profile: backend.Description{IOProfile: record.IOProfile{
			Name: "io-profile", ImplementationSHA256: record.HashBytes([]byte("implementation")), InventorySHA256: record.HashBytes([]byte("inventory")),
		}},
		choiceImplementation: sha256.Sum256([]byte("choice implementation")),
	}
	want, err := simulationExplorationExecutionIdentity(base.config, base.prepared, base.profile, base.choiceImplementation)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(simulationExplorationExecutionIdentityProjection{
		TargetSHA256: record.SHA256(base.prepared.SHA256), ToolchainBuildKey: base.prepared.BuildKey,
		GOOS: base.prepared.TargetGOOS, GOARCH: base.prepared.TargetGOARCH, ChoiceImplementation: record.SHA256FromSum(base.choiceImplementation),
		IOProfile:   record.IOContract{Name: base.profile.IOProfile.Name, ImplementationSHA256: base.profile.IOProfile.ImplementationSHA256, InventorySHA256: base.profile.IOProfile.InventorySHA256},
		RunnerBuild: base.config.RunnerBuild,
	})
	if err != nil {
		t.Fatal(err)
	}
	if projected := record.DomainHash("gomad3-simulation-exploration-execution/v1", encoded); want != projected {
		t.Fatalf("execution identity = %s, want stdlib projection %s", want, projected)
	}
	repeated, err := simulationExplorationExecutionIdentity(base.config, base.prepared, base.profile, base.choiceImplementation)
	if err != nil || repeated != want {
		t.Fatalf("repeated identity = %s, error = %v, want %s", repeated, err, want)
	}
	for name, mutate := range map[string]func(*inputs){
		"target":                func(value *inputs) { value.prepared.SHA256 = string(record.HashBytes([]byte("changed target"))) },
		"toolchain build":       func(value *inputs) { value.prepared.BuildKey = "changed toolchain" },
		"GOOS":                  func(value *inputs) { value.prepared.TargetGOOS = "darwin" },
		"GOARCH":                func(value *inputs) { value.prepared.TargetGOARCH = "arm64" },
		"choice implementation": func(value *inputs) { value.choiceImplementation = sha256.Sum256([]byte("changed choice")) },
		"I/O profile":           func(value *inputs) { value.profile.IOProfile.Name = "changed-profile" },
		"I/O implementation": func(value *inputs) {
			value.profile.IOProfile.ImplementationSHA256 = record.HashBytes([]byte("changed implementation"))
		},
		"I/O inventory": func(value *inputs) {
			value.profile.IOProfile.InventorySHA256 = record.HashBytes([]byte("changed inventory"))
		},
		"Runner build": func(value *inputs) { value.config.RunnerBuild = "changed-runner" },
	} {
		t.Run(name, func(t *testing.T) {
			changed := base
			mutate(&changed)
			got, err := simulationExplorationExecutionIdentity(changed.config, changed.prepared, changed.profile, changed.choiceImplementation)
			if err != nil {
				t.Fatal(err)
			}
			if got == want {
				t.Fatal("changed input did not change the execution identity")
			}
		})
	}
}

func TestSimulationExplorationExecutionIdentityRejectsInvalidOriginalStrings(t *testing.T) {
	prepared := target.Prepared{
		SHA256: string(record.HashBytes([]byte("target"))), BuildKey: "toolchain-build", TargetGOOS: "linux", TargetGOARCH: "amd64",
	}
	profile := backend.Description{IOProfile: record.IOProfile{
		Name: "io-profile", ImplementationSHA256: record.HashBytes([]byte("implementation")), InventorySHA256: record.HashBytes([]byte("inventory")),
	}}
	config := campaignRequest{campaignRuntime: campaignRuntime{RunnerBuild: string([]byte{0xff})}}
	if _, err := simulationExplorationExecutionIdentity(config, prepared, profile, sha256.Sum256([]byte("choice"))); err == nil {
		t.Fatal("simulationExplorationExecutionIdentity() accepted invalid UTF-8")
	}
}

func TestSimulationCapabilityForJobCarriesDetachedExplorationPlanToInjectedExecutor(t *testing.T) {
	plan := []byte(`{"schema":"gomad3.simulation-exploration-plan/v1"}`)
	job := runJob{simulationPlan: string(plan), simulationRecordLimit: 1 << 20, simulationRecordCount: 1}
	capability, err := simulationCapabilityForJob(&fakeExecutor{}, job)
	if err != nil {
		t.Fatal(err)
	}
	plan[0] = '!'
	if capability == nil || capability.Role != backend.SimulationRoleCoordinator || string(capability.ExplorationPlan) != `{"schema":"gomad3.simulation-exploration-plan/v1"}` || capability.ExplorationRecordLimit != 1<<20 || capability.ExplorationRecordCount != 1 {
		t.Fatalf("simulation exploration capability = %#v", capability)
	}
	if _, err := simulationCapabilityForJob(&fakeExecutor{}, runJob{simulationPlan: "plan"}); err == nil {
		t.Fatal("simulationCapabilityForJob() accepted missing record bounds")
	}
}

func TestRunSimulationExplorationExecutesRootAndEveryScenarioRank(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	executor := &simulationExplorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit}
	config, configDependencies := testConfig(t, preparer, executor, "7", PolicyAll, 1)
	config.Strategy = StrategySimulationExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 4
	config.MaxForcedDecisions = 2
	config.MaxExplorationBytes = 1 << 20
	config.MaxExplorationResultBytes = 1 << 20
	config.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 2, Scenario: 2, Network: 2, Storage: 2, Fault: 2, Crash: 2}

	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Attempted != 2 || summary.Succeeded != 2 || summary.StopReason != StopExplorationExhausted || summary.SimulationExploration == nil || summary.SimulationExploration.CommittedRounds != 2 || summary.SimulationExploration.SeenCandidates != 2 {
		t.Fatalf("simulation exploration summary = %#v", summary)
	}
	executor.mu.Lock()
	requests := append([]backend.ExecutionRequest(nil), executor.requests...)
	executor.mu.Unlock()
	if len(requests) != 2 || requests[0].Simulation == nil || len(requests[0].Simulation.ExplorationPlan) == 0 || requests[1].Simulation == nil || len(requests[1].Simulation.ExplorationPlan) == 0 || requests[0].Choice == nil || requests[0].Choice.Mode != choice.ModeRecord || requests[1].Choice == nil || requests[1].Choice.Mode != choice.ModeRecord {
		t.Fatalf("simulation exploration requests = %#v", requests)
	}
	batch, err := campaign.OpenCampaign(summary.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if batch.Record.Strategy != string(StrategySimulationExploration) || batch.Record.SimulationExploration == nil || len(batch.Executions) != 2 || batch.Executions[1].ParentCandidateSHA256 == "" || batch.Executions[1].ForcedDepth == nil || *batch.Executions[1].ForcedDepth != 1 {
		t.Fatalf("simulation exploration batch = %#v", batch)
	}
}

func TestRunSimulationExplorationRetainsExactDeduplicatedSimulationFailure(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	executor := &simulationExplorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit, fail: true}
	config, configDependencies := testConfig(t, preparer, executor, "7", PolicyAll, 1)
	config.Strategy = StrategySimulationExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 4
	config.MaxForcedDecisions = 2
	config.MaxExplorationBytes = 1 << 20
	config.MaxExplorationResultBytes = 1 << 20
	config.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 2, Scenario: 2, Network: 2, Storage: 2, Fault: 2, Crash: 2}

	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Attempted != 2 || summary.Failures != 2 || summary.DistinctFailures != 1 || len(summary.Artifacts) != 1 {
		t.Fatalf("simulation exploration failure summary = %#v", summary)
	}
	opened, err := artifact.OpenArtifact(summary.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	defer func(opened *artifact.Opened) {
		if err := opened.Close(); err != nil {
			t.Error(err)
		}
	}(opened)
	profile := opened.Manifest().SimulationProfile
	if profile == nil {
		t.Fatal("simulation exploration failure artifact omitted simulation exploration evidence")
	}
	plan, err := opened.ReadPayload(profile.Plan.File, uint64(profile.Plan.Bytes))
	if err != nil {
		t.Fatal(err)
	}
	record, err := opened.ReadPayload(profile.Record.File, uint64(profile.Record.Bytes))
	if err != nil {
		t.Fatal(err)
	}
	if err := exploration.ValidateArtifact(*profile, plan, record); err != nil {
		t.Fatal(err)
	}
	batch, err := campaign.OpenCampaign(summary.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Executions) != 2 || batch.Executions[0].Artifact == nil || batch.Executions[1].Artifact == nil || *batch.Executions[0].Artifact != *batch.Executions[1].Artifact {
		t.Fatalf("deduplicated combined failure references = %#v", batch.Executions)
	}
}

func TestRunSimulationExplorationResumePreservesCommittedCandidates(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	baseExecutor := &simulationExplorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit}
	config, configDependencies := testConfig(t, preparer, simulationExplorationInterruptExecutor{exploration: baseExecutor}, "7", PolicyAll, 1)
	config.Strategy = StrategySimulationExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 4
	config.MaxForcedDecisions = 2
	config.MaxExplorationBytes = 1 << 20
	config.MaxExplorationResultBytes = 1 << 20
	config.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 2, Scenario: 2, Network: 2, Storage: 2, Fault: 2, Crash: 2}

	partial, err := exploreWith(context.Background(), config, configDependencies)
	var hostErr *HostError
	if !errors.As(err, &hostErr) || partial.Attempted != 1 || partial.SimulationExploration == nil || partial.SimulationExploration.CommittedRounds != 1 {
		t.Fatalf("partial simulation exploration = %#v, error = %v", partial, err)
	}
	resumedExecutor := &simulationExplorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit}
	resumed, err := exploreWith(context.Background(), CampaignSpec{
		ResumeCampaign: partial.CampaignPath, RunnerBuild: config.RunnerBuild, Backend: scriptProvider(),
	}, executionDependencies{executor: resumedExecutor},
	)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Attempted != 2 || resumed.Succeeded != 2 || resumed.RecoveryExecutions != 1 || resumed.SimulationExploration == nil || resumed.SimulationExploration.CommittedRounds != 2 || resumed.StopReason != StopExplorationExhausted {
		t.Fatalf("resumed simulation exploration = %#v", resumed)
	}
	resumedExecutor.mu.Lock()
	resumedRequests := len(resumedExecutor.requests)
	resumedExecutor.mu.Unlock()
	if resumedRequests != 1 {
		t.Fatalf("resumed simulation exploration executed %d candidates, want only the uncommitted candidate", resumedRequests)
	}
}

func TestSimulationCapabilityRequiresProviderSupport(t *testing.T) {
	for _, test := range []struct {
		name     string
		provider backend.Provider
		want     bool
	}{
		{name: "peer without simulation", provider: &boundaryProvider{}, want: false},
		{name: "simulation provider", provider: native.New(native.Config{}), want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			capability, err := simulationCapabilityForJob(processExecutor{provider: test.provider}, runJob{})
			if err != nil || (capability != nil) != test.want {
				t.Fatalf("simulation capability=%+v error=%v want enabled=%t", capability, err, test.want)
			}
		})
	}
}
