package simulationrecord

import (
	"crypto/sha256"
	"encoding/json"
	"testing"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
	simulationengine "go.temporal.io/server/tools/gomad3/runner/internal/exploration/simulation"
)

func TestRuntimeDecisionsProjectsBranchingChoiceTapeByRank(t *testing.T) {
	first := sha256.Sum256([]byte("first runnable"))
	second := sha256.Sum256([]byte("second runnable"))
	decision, err := choice.CanonicalDecision(0, choice.KindRunnable, 24, false, [][sha256.Size]byte{first, second}, second, 0)
	if err != nil {
		t.Fatal(err)
	}
	trace, err := choice.BuildTrace([]choice.Record{decision.Record()}, choice.TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	identity := choice.ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("target")), ToolchainBuildKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		GOOS: "darwin", GOARCH: "arm64", ImplementationSHA256: sha256.Sum256([]byte("implementation")),
	}
	tape, err := choice.ProjectReplayPlan(trace, identity)
	if err != nil {
		t.Fatal(err)
	}

	decisions, err := RuntimeDecisions(tape)
	if err != nil {
		t.Fatal(err)
	}
	if len(decisions) != 1 || decisions[0].Dimension != simulationengine.DimensionRuntime || decisions[0].Ordinal != 0 || len(decisions[0].Alternatives) != 2 || decisions[0].Selected != 1 {
		t.Fatalf("runtime decisions = %#v", decisions)
	}
	canonical, err := simulationengine.CanonicalControlledDecision(
		simulationengine.DimensionRuntime, decisions[0].Ordinal, decisions[0].SiteSHA256, decisions[0].Alternatives, decisions[0].AlternativeControls, decisions[0].Selected,
	)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := canonicaljson.CanonicalJSON(decisions[0])
	if err != nil {
		t.Fatal(err)
	}
	want, err := canonicaljson.CanonicalJSON(canonical)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != string(want) {
		t.Fatalf("runtime decision = %s, canonical = %s", actual, want)
	}
}

func TestExecutionForCandidateRestoresRuntimeRankPrefix(t *testing.T) {
	first := sha256.Sum256([]byte("first runnable"))
	second := sha256.Sum256([]byte("second runnable"))
	decision, err := choice.CanonicalDecision(0, choice.KindRunnable, 24, false, [][sha256.Size]byte{first, second}, first, 0)
	if err != nil {
		t.Fatal(err)
	}
	trace, err := choice.BuildTrace([]choice.Record{decision.Record()}, choice.TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	identity := choice.ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("target")), ToolchainBuildKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		GOOS: "darwin", GOARCH: "arm64", ImplementationSHA256: sha256.Sum256([]byte("implementation")),
	}
	tape, err := choice.ProjectReplayPlan(trace, identity)
	if err != nil {
		t.Fatal(err)
	}
	runtimeDecisions, err := RuntimeDecisions(tape)
	if err != nil {
		t.Fatal(err)
	}
	config := simulationengine.Config{
		ExecutionSHA256: record.HashBytes([]byte("execution")), ControllerSHA256: simulationengine.ImplementationSHA256(), BaseSeed: 17,
		Parallel: 1, MaxExecutions: 2, MaxForcedDecisions: 1, MaxExplorationBytes: 1 << 20, MaxResultBytes: 1 << 20, FailureBudget: 1,
		Limits: simulationengine.DimensionLimits{Runtime: 1, Scenario: 1, Network: 1, Storage: 1, Fault: 1, Crash: 1},
	}
	state, err := simulationengine.New(config)
	if err != nil {
		t.Fatal(err)
	}
	root, _ := state.NextRound()
	state, _, err = simulationengine.CommitRound(state, root, []simulationengine.Result{{
		CandidateSHA256: root.Candidates[0].SHA256, OutcomeSHA256: record.HashBytes([]byte("root")), Decisions: runtimeDecisions,
	}})
	if err != nil {
		t.Fatal(err)
	}
	children, ok := state.NextRound()
	if !ok || len(children.Candidates) != 1 {
		t.Fatalf("runtime child round = %#v", children)
	}

	execution, err := ExecutionForCandidate(config, children.Candidates[0], identity)
	if err != nil {
		t.Fatal(err)
	}
	if execution.ChoiceMode != choice.ModePrefix || execution.ChoiceReplayPlan == nil || len(execution.ChoiceReplayPlan.Decisions) != 1 || !execution.ChoiceReplayPlan.Decisions[0].RankOverride || execution.ChoiceReplayPlan.Decisions[0].Selected != 1 {
		t.Fatalf("runtime candidate execution = %#v", execution)
	}
}

func TestCandidateForArtifactRestoresRuntimeControlAndFaultOverride(t *testing.T) {
	identity := choice.ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("target")), ToolchainBuildKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		GOOS: "darwin", GOARCH: "arm64", ImplementationSHA256: sha256.Sum256([]byte("implementation")),
	}
	unforcedRuntime := testRuntimeDecision(t, 0, 0)
	unforcedTrace, err := choice.BuildTrace([]choice.Record{unforcedRuntime.Record()}, choice.TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	unforcedTape, err := choice.ProjectReplayPlan(unforcedTrace, identity)
	if err != nil {
		t.Fatal(err)
	}
	runtimeDecisions, err := RuntimeDecisions(unforcedTape)
	if err != nil {
		t.Fatal(err)
	}
	runtimeOverride, err := simulationengine.ForceDecision(runtimeDecisions[0], 1)
	if err != nil {
		t.Fatal(err)
	}
	faultDecision, err := simulationengine.CanonicalDecision(
		simulationengine.DimensionFault, 0, record.HashBytes([]byte("fault site")),
		[]record.SHA256{record.HashBytes([]byte("default")), record.HashBytes([]byte("drop"))}, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	faultOverride, err := simulationengine.ForceDecision(faultDecision, 1)
	if err != nil {
		t.Fatal(err)
	}
	config := testCombinedConfig()
	candidate, err := simulationengine.CanonicalCandidate(config, []simulationengine.ForcedDecision{runtimeOverride, faultOverride}, "")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanForCandidate(config, candidate)
	if err != nil {
		t.Fatal(err)
	}
	observedRuntime := testRuntimeDecision(t, 0, 1)
	observedTrace, err := choice.BuildTrace([]choice.Record{observedRuntime.Record()}, choice.TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	observedTape, err := choice.ProjectReplayPlan(observedTrace, identity)
	if err != nil {
		t.Fatal(err)
	}
	profile := record.SimulationProfile{
		ControllerSHA256: config.ControllerSHA256, ExecutionSHA256: config.ExecutionSHA256, CandidateSHA256: candidate.SHA256,
	}

	gotConfig, gotCandidate, err := CandidateForArtifact(profile, plan, &observedTape)
	if err != nil {
		t.Fatal(err)
	}
	if gotConfig.ExecutionSHA256 != config.ExecutionSHA256 || gotConfig.BaseSeed != config.BaseSeed || gotCandidate.SHA256 != candidate.SHA256 || len(gotCandidate.Overrides) != 2 {
		t.Fatalf("reconstructed candidate = %#v with config %#v", gotCandidate, gotConfig)
	}
	if string(gotCandidate.Overrides[0].Control) != string(runtimeOverride.Control) || gotCandidate.Overrides[1].Dimension != simulationengine.DimensionFault {
		t.Fatalf("reconstructed overrides = %#v", gotCandidate.Overrides)
	}
}

func TestResultForRecordProjectsValidatedDecisionsAndSemanticOutcome(t *testing.T) {
	config := simulationengine.Config{
		ExecutionSHA256: record.HashBytes([]byte("execution")), ControllerSHA256: simulationengine.ImplementationSHA256(), BaseSeed: 17,
		Parallel: 1, MaxExecutions: 2, MaxForcedDecisions: 1, MaxExplorationBytes: 1 << 20, MaxResultBytes: 1 << 20, FailureBudget: 1,
		Limits: simulationengine.DimensionLimits{Runtime: 1, Scenario: 1, Network: 1, Storage: 1, Fault: 1, Crash: 1},
	}
	state, err := simulationengine.New(config)
	if err != nil {
		t.Fatal(err)
	}
	round, ok := state.NextRound()
	if !ok {
		t.Fatal("simulation exploration root round is unavailable")
	}
	candidate := round.Candidates[0]
	plan, err := PlanForCandidate(config, candidate)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := simulationengine.CanonicalDecision(
		simulationengine.DimensionScenario, 0, record.HashBytes([]byte("route")),
		[]record.SHA256{record.HashBytes([]byte("alpha")), record.HashBytes([]byte("beta"))}, 1,
	)
	if err != nil {
		t.Fatal(err)
	}
	recordValue := struct {
		Schema               string                      `json:"schema"`
		Seed                 uint64                      `json:"seed"`
		SpecSHA256           record.SHA256               `json:"spec_sha256"`
		Outcome              string                      `json:"outcome"`
		ExplorationPlan      json.RawMessage             `json:"exploration_plan"`
		ExplorationDecisions []simulationengine.Decision `json:"exploration_decisions"`
		ScenarioTape         []string                    `json:"scenario_tape"`
		Identity             record.SHA256               `json:"identity"`
	}{
		Schema: "gomad3.cluster-record/v7", Seed: 17, SpecSHA256: record.HashBytes([]byte("candidate-bound-spec")), Outcome: "completed",
		ExplorationPlan: plan, ExplorationDecisions: []simulationengine.Decision{decision}, ScenarioTape: []string{"beta"}, Identity: record.HashBytes([]byte("record")),
	}
	encoded, err := json.Marshal(recordValue)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ResultForRecord(config, candidate, encoded, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.CandidateSHA256 != candidate.SHA256 || result.Failed || result.Diverged || len(result.Decisions) != 1 || result.Decisions[0].Identity != decision.Identity {
		t.Fatalf("result = %#v", result)
	}
	if _, err := result.OutcomeSHA256.Bytes(); err != nil {
		t.Fatalf("outcome identity = %q: %v", result.OutcomeSHA256, err)
	}

	recordValue.ExplorationDecisions[0].Identity = record.HashBytes([]byte("changed"))
	changed, err := json.Marshal(recordValue)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ResultForRecord(config, candidate, changed, nil); err == nil {
		t.Fatal("ResultForRecord() accepted a changed decision identity")
	}
}

func TestProjectArtifactBindsExactReplayAndNormalizedFailure(t *testing.T) {
	config := simulationengine.Config{
		ExecutionSHA256: record.HashBytes([]byte("execution")), ControllerSHA256: simulationengine.ImplementationSHA256(), BaseSeed: 17,
		Parallel: 1, MaxExecutions: 2, MaxForcedDecisions: 1, MaxExplorationBytes: 1 << 20, MaxResultBytes: 1 << 20, FailureBudget: 1,
		Limits: simulationengine.DimensionLimits{Runtime: 1, Scenario: 1, Network: 1, Storage: 1, Fault: 1, Crash: 1},
	}
	state, err := simulationengine.New(config)
	if err != nil {
		t.Fatal(err)
	}
	round, ok := state.NextRound()
	if !ok {
		t.Fatal("simulation exploration root round is unavailable")
	}
	candidate := round.Candidates[0]
	plan, err := PlanForCandidate(config, candidate)
	if err != nil {
		t.Fatal(err)
	}
	failure := record.HashBytes([]byte("normalized oracle failure"))
	recordBytes, err := json.Marshal(struct {
		Schema          string          `json:"schema"`
		Seed            uint64          `json:"seed"`
		SpecSHA256      record.SHA256   `json:"spec_sha256"`
		Outcome         string          `json:"outcome"`
		FailureIdentity record.SHA256   `json:"failure_identity"`
		ExplorationPlan json.RawMessage `json:"exploration_plan"`
		Identity        record.SHA256   `json:"identity"`
	}{
		Schema: "gomad3.cluster-record/v7", Seed: config.BaseSeed, SpecSHA256: record.HashBytes([]byte("spec")),
		Outcome: "oracle_failed", FailureIdentity: failure, ExplorationPlan: plan, Identity: record.HashBytes([]byte("record")),
	})
	if err != nil {
		t.Fatal(err)
	}

	profile, err := ProjectArtifact(config, candidate, plan, recordBytes, nil, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if profile.CandidateSHA256 != candidate.SHA256 || profile.ControllerSHA256 != config.ControllerSHA256 || profile.ExecutionSHA256 != config.ExecutionSHA256 || profile.FailureSHA256 != failure || profile.Plan.SHA256 != record.HashBytes(plan) || profile.Record.SHA256 != record.HashBytes(recordBytes) {
		t.Fatalf("simulation profile = %#v", profile)
	}
	if err := ValidateArtifact(profile, plan, recordBytes); err != nil {
		t.Fatal(err)
	}

	changedConfig := config
	changedConfig.BaseSeed = 18
	changedState, err := simulationengine.New(changedConfig)
	if err != nil {
		t.Fatal(err)
	}
	changedRound, ok := changedState.NextRound()
	if !ok {
		t.Fatal("changed simulation exploration root round is unavailable")
	}
	changedPlan, err := PlanForCandidate(changedConfig, changedRound.Candidates[0])
	if err != nil {
		t.Fatal(err)
	}
	profile.CandidateSHA256 = changedRound.Candidates[0].SHA256
	profile.Plan.SHA256 = record.HashBytes(changedPlan)
	profile.Plan.Bytes = record.Uint64String(len(changedPlan))
	if err := ValidateArtifact(profile, changedPlan, recordBytes); err == nil {
		t.Fatal("ValidateArtifact() accepted a record bound to a different exploration plan")
	}
}

func TestProjectArtifactRequiresEveryForcedDecisionRecord(t *testing.T) {
	config := testCombinedConfig()
	decision, err := simulationengine.CanonicalDecision(
		simulationengine.DimensionFault, 0, record.HashBytes([]byte("fault site")),
		[]record.SHA256{record.HashBytes([]byte("none")), record.HashBytes([]byte("drop"))}, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	forced, err := simulationengine.ForceDecision(decision, 1)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := simulationengine.CanonicalCandidate(config, []simulationengine.ForcedDecision{forced}, "")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanForCandidate(config, candidate)
	if err != nil {
		t.Fatal(err)
	}
	record, err := json.Marshal(struct {
		Schema          string          `json:"schema"`
		Seed            uint64          `json:"seed"`
		SpecSHA256      record.SHA256   `json:"spec_sha256"`
		Outcome         string          `json:"outcome"`
		FailureIdentity record.SHA256   `json:"failure_identity"`
		ExplorationPlan json.RawMessage `json:"exploration_plan"`
		Identity        record.SHA256   `json:"identity"`
	}{
		Schema: "gomad3.cluster-record/v7", Seed: config.BaseSeed, SpecSHA256: record.HashBytes([]byte("spec")),
		Outcome: "oracle_failed", FailureIdentity: record.HashBytes([]byte("failure")), ExplorationPlan: plan,
		Identity: record.HashBytes([]byte("record")),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ProjectArtifact(config, candidate, plan, record, nil, 1<<20); err == nil {
		t.Fatal("ProjectArtifact() accepted a record that omitted its forced fault decision")
	}
}

func testCombinedConfig() simulationengine.Config {
	return simulationengine.Config{
		ExecutionSHA256: record.HashBytes([]byte("execution")), ControllerSHA256: simulationengine.ImplementationSHA256(), BaseSeed: 17,
		Parallel: 1, MaxExecutions: 8, MaxForcedDecisions: 8, MaxExplorationBytes: 1 << 20, MaxResultBytes: 1 << 20, FailureBudget: 1,
		Limits: simulationengine.DimensionLimits{Runtime: 8, Scenario: 8, Network: 8, Storage: 8, Fault: 8, Crash: 8},
	}
}

func testRuntimeDecision(t *testing.T, ordinal uint64, selected uint32) choice.Decision {
	t.Helper()
	first := sha256.Sum256([]byte("first runnable"))
	second := sha256.Sum256([]byte("second runnable"))
	decision, err := choice.CanonicalDecision(ordinal, choice.KindRunnable, 24, false, [][sha256.Size]byte{first, second}, [][sha256.Size]byte{first, second}[selected], 0)
	if err != nil {
		t.Fatal(err)
	}
	return decision
}
