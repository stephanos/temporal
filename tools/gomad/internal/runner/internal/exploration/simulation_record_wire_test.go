package exploration

import (
	"fmt"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/evidence/record"
)

func TestPlanForCandidateEncodesCanonicalRootContract(t *testing.T) {
	config := SimulationConfig{
		ExecutionSHA256: record.HashBytes([]byte("execution")), ControllerSHA256: SimulationImplementationSHA256(), BaseSeed: 17,
		Parallel: 1, MaxExecutions: 2, MaxForcedDecisions: 1, MaxExplorationBytes: 1 << 20, MaxResultBytes: 1 << 20, FailureBudget: 1,
		Limits: DimensionLimits{Runtime: 1, Scenario: 1, Network: 1, Storage: 1, Fault: 1, Crash: 1},
	}
	state, err := NewSimulation(config)
	if err != nil {
		t.Fatal(err)
	}
	round, ok := state.NextRound()
	if !ok || len(round.Candidates) != 1 {
		t.Fatal("simulation exploration root candidate is unavailable")
	}

	encoded, err := PlanForCandidate(config, round.Candidates[0])
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf(`{"schema":"gomad3.simulation-exploration-plan/v1","execution_sha256":"%s","controller_sha256":"%s","base_seed":17,"overrides":null,"candidate_sha256":"%s"}`, config.ExecutionSHA256, config.ControllerSHA256, round.Candidates[0].SHA256)
	if string(encoded) != want {
		t.Fatalf("plan = %s, want %s", encoded, want)
	}

	changed := round.Candidates[0]
	changed.SHA256 = record.HashBytes([]byte("changed"))
	if _, err := PlanForCandidate(config, changed); err == nil {
		t.Fatal("PlanForCandidate() accepted a changed candidate identity")
	}
}

func TestPlanForCandidatePreservesNilAndEmptyOverrideCollections(t *testing.T) {
	config := simulationTestConfig()
	nilCandidate, err := CanonicalCandidate(config, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	emptyCandidate, err := CanonicalCandidate(config, []ForcedDecision{}, "")
	if err != nil {
		t.Fatal(err)
	}
	if nilCandidate.SHA256 == emptyCandidate.SHA256 {
		t.Fatal("nil and empty override collections produced one candidate identity")
	}
	nilPlan, err := PlanForCandidate(config, nilCandidate)
	if err != nil {
		t.Fatal(err)
	}
	emptyPlan, err := PlanForCandidate(config, emptyCandidate)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(nilPlan), `"overrides":null`) || !strings.Contains(string(emptyPlan), `"overrides":[]`) {
		t.Fatalf("nil plan = %s; empty plan = %s", nilPlan, emptyPlan)
	}
	if _, err := validateRetainedPlan(nilPlan); err != nil {
		t.Fatalf("validate nil-override plan: %v", err)
	}
	if _, err := validateRetainedPlan(emptyPlan); err != nil {
		t.Fatalf("validate empty-override plan: %v", err)
	}
}

func TestCombinedAlternativeSetIdentityMatchesPairedFormula(t *testing.T) {
	decision := simulationTestDecision(t, DimensionRuntime, 3, 3, 1)
	want, err := alternativeSetIdentity(decision.Dimension, decision.Ordinal, decision.SiteSHA256, decision.Alternatives)
	if err != nil {
		t.Fatal(err)
	}
	if got := combinedAlternativeSetIdentity(decision.Ordinal, decision.SiteSHA256, decision.Alternatives); got != want {
		t.Fatalf("combined alternative-set identity = %s, want paired identity %s", got, want)
	}
}
