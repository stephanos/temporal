package simulationexploration

import (
	"encoding/json"
	"testing"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/combinedfrontier"
)

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
