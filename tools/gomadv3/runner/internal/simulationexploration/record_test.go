package simulationexploration

import (
	"crypto/sha256"
	"encoding/json"
	"testing"

	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/combinedfrontier"
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
	if len(decisions) != 1 || decisions[0].Dimension != combinedfrontier.DimensionRuntime || decisions[0].Ordinal != 0 || len(decisions[0].Alternatives) != 2 || decisions[0].Selected != 1 {
		t.Fatalf("runtime decisions = %#v", decisions)
	}
	canonical, err := combinedfrontier.CanonicalControlledDecision(
		combinedfrontier.DimensionRuntime, decisions[0].Ordinal, decisions[0].SiteSHA256, decisions[0].Alternatives, decisions[0].AlternativeControls, decisions[0].Selected,
	)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := evidence.CanonicalJSON(decisions[0])
	if err != nil {
		t.Fatal(err)
	}
	want, err := evidence.CanonicalJSON(canonical)
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
	config := combinedfrontier.Config{
		ExecutionSHA256: evidence.HashBytes([]byte("execution")), ControllerSHA256: combinedfrontier.ImplementationSHA256(), BaseSeed: 17,
		Parallel: 1, MaxRuns: 2, MaxForcedDecisions: 1, MaxFrontierBytes: 1 << 20, MaxResultBytes: 1 << 20, FailureBudget: 1,
		Limits: combinedfrontier.DimensionLimits{Runtime: 1, Scenario: 1, Network: 1, Storage: 1, Fault: 1, Crash: 1},
	}
	state, err := combinedfrontier.New(config)
	if err != nil {
		t.Fatal(err)
	}
	root, _ := state.NextRound()
	state, _, err = combinedfrontier.CommitRound(state, root, []combinedfrontier.Result{{
		CandidateSHA256: root.Candidates[0].SHA256, OutcomeSHA256: evidence.HashBytes([]byte("root")), Decisions: runtimeDecisions,
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

func TestResultForRecordProjectsValidatedDecisionsAndSemanticOutcome(t *testing.T) {
	config := combinedfrontier.Config{
		ExecutionSHA256: evidence.HashBytes([]byte("execution")), ControllerSHA256: combinedfrontier.ImplementationSHA256(), BaseSeed: 17,
		Parallel: 1, MaxRuns: 2, MaxForcedDecisions: 1, MaxFrontierBytes: 1 << 20, MaxResultBytes: 1 << 20, FailureBudget: 1,
		Limits: combinedfrontier.DimensionLimits{Runtime: 1, Scenario: 1, Network: 1, Storage: 1, Fault: 1, Crash: 1},
	}
	state, err := combinedfrontier.New(config)
	if err != nil {
		t.Fatal(err)
	}
	round, ok := state.NextRound()
	if !ok {
		t.Fatal("combined frontier root round is unavailable")
	}
	candidate := round.Candidates[0]
	plan, err := PlanForCandidate(config, candidate)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := combinedfrontier.CanonicalDecision(
		combinedfrontier.DimensionScenario, 0, evidence.HashBytes([]byte("route")),
		[]evidence.SHA256{evidence.HashBytes([]byte("alpha")), evidence.HashBytes([]byte("beta"))}, 1,
	)
	if err != nil {
		t.Fatal(err)
	}
	record := struct {
		Schema               string                      `json:"schema"`
		Seed                 uint64                      `json:"seed"`
		SpecSHA256           evidence.SHA256             `json:"spec_sha256"`
		Outcome              string                      `json:"outcome"`
		ExplorationPlan      json.RawMessage             `json:"exploration_plan"`
		ExplorationDecisions []combinedfrontier.Decision `json:"exploration_decisions"`
		ScenarioTape         []string                    `json:"scenario_tape"`
		Identity             evidence.SHA256             `json:"identity"`
	}{
		Schema: "gomadv3.cluster-record/v7", Seed: 17, SpecSHA256: evidence.HashBytes([]byte("candidate-bound-spec")), Outcome: "completed",
		ExplorationPlan: plan, ExplorationDecisions: []combinedfrontier.Decision{decision}, ScenarioTape: []string{"beta"}, Identity: evidence.HashBytes([]byte("record")),
	}
	encoded, err := json.Marshal(record)
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

	record.ExplorationDecisions[0].Identity = evidence.HashBytes([]byte("changed"))
	changed, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ResultForRecord(config, candidate, changed, nil); err == nil {
		t.Fatal("ResultForRecord() accepted a changed decision identity")
	}
}

func TestProjectArtifactBindsExactReplayAndNormalizedFailure(t *testing.T) {
	config := combinedfrontier.Config{
		ExecutionSHA256: evidence.HashBytes([]byte("execution")), ControllerSHA256: combinedfrontier.ImplementationSHA256(), BaseSeed: 17,
		Parallel: 1, MaxRuns: 2, MaxForcedDecisions: 1, MaxFrontierBytes: 1 << 20, MaxResultBytes: 1 << 20, FailureBudget: 1,
		Limits: combinedfrontier.DimensionLimits{Runtime: 1, Scenario: 1, Network: 1, Storage: 1, Fault: 1, Crash: 1},
	}
	state, err := combinedfrontier.New(config)
	if err != nil {
		t.Fatal(err)
	}
	round, ok := state.NextRound()
	if !ok {
		t.Fatal("combined frontier root round is unavailable")
	}
	candidate := round.Candidates[0]
	plan, err := PlanForCandidate(config, candidate)
	if err != nil {
		t.Fatal(err)
	}
	failure := evidence.HashBytes([]byte("normalized oracle failure"))
	record, err := json.Marshal(struct {
		Schema          string          `json:"schema"`
		Seed            uint64          `json:"seed"`
		SpecSHA256      evidence.SHA256 `json:"spec_sha256"`
		Outcome         string          `json:"outcome"`
		FailureIdentity evidence.SHA256 `json:"failure_identity"`
		ExplorationPlan json.RawMessage `json:"exploration_plan"`
		Identity        evidence.SHA256 `json:"identity"`
	}{
		Schema: "gomadv3.cluster-record/v7", Seed: config.BaseSeed, SpecSHA256: evidence.HashBytes([]byte("spec")),
		Outcome: "oracle_failed", FailureIdentity: failure, ExplorationPlan: plan, Identity: evidence.HashBytes([]byte("record")),
	})
	if err != nil {
		t.Fatal(err)
	}

	profile, err := ProjectArtifact(config, candidate, plan, record, nil, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if profile.CandidateSHA256 != candidate.SHA256 || profile.ControllerSHA256 != config.ControllerSHA256 || profile.ExecutionSHA256 != config.ExecutionSHA256 || profile.FailureSHA256 != failure || profile.Plan.SHA256 != evidence.HashBytes(plan) || profile.Record.SHA256 != evidence.HashBytes(record) {
		t.Fatalf("simulation profile = %#v", profile)
	}
	if err := ValidateArtifact(profile, plan, record); err != nil {
		t.Fatal(err)
	}

	changedConfig := config
	changedConfig.BaseSeed = 18
	changedState, err := combinedfrontier.New(changedConfig)
	if err != nil {
		t.Fatal(err)
	}
	changedRound, ok := changedState.NextRound()
	if !ok {
		t.Fatal("changed combined frontier root round is unavailable")
	}
	changedPlan, err := PlanForCandidate(changedConfig, changedRound.Candidates[0])
	if err != nil {
		t.Fatal(err)
	}
	profile.CandidateSHA256 = changedRound.Candidates[0].SHA256
	profile.Plan.SHA256 = evidence.HashBytes(changedPlan)
	profile.Plan.Bytes = evidence.Uint64String(len(changedPlan))
	if err := ValidateArtifact(profile, changedPlan, record); err == nil {
		t.Fatal("ValidateArtifact() accepted a record bound to a different exploration plan")
	}
}
