package simulationrecord

import (
	"fmt"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
	simulationengine "go.temporal.io/server/tools/gomad3/runner/internal/exploration/simulation"
)

func TestPlanForCandidateEncodesCanonicalRootContract(t *testing.T) {
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
