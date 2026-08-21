package simulationexploration

import (
	"fmt"
	"testing"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/combinedfrontier"
)

func TestPlanForCandidateEncodesCanonicalRootContract(t *testing.T) {
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
	if !ok || len(round.Candidates) != 1 {
		t.Fatal("combined frontier root candidate is unavailable")
	}

	encoded, err := PlanForCandidate(config, round.Candidates[0])
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf(`{"schema":"gomadv3.simulation-exploration-plan/v1","execution_sha256":"%s","controller_sha256":"%s","base_seed":17,"overrides":null,"candidate_sha256":"%s"}`, config.ExecutionSHA256, config.ControllerSHA256, round.Candidates[0].SHA256)
	if string(encoded) != want {
		t.Fatalf("plan = %s, want %s", encoded, want)
	}

	changed := round.Candidates[0]
	changed.SHA256 = evidence.HashBytes([]byte("changed"))
	if _, err := PlanForCandidate(config, changed); err == nil {
		t.Fatal("PlanForCandidate() accepted a changed candidate identity")
	}
}
