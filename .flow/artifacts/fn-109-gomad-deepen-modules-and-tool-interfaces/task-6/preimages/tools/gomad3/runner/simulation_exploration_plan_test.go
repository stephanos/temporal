package runner

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
	simulationengine "go.temporal.io/server/tools/gomad3/runner/internal/exploration/simulation"
	simulationrecord "go.temporal.io/server/tools/gomad3/runner/internal/exploration/simulationrecord"
)

// simulationExplorationPlanVector is the retained exploration plan the Runner
// issues for fixed inputs. tools/gomad3sim, which lives in another module and
// cannot share code with the Runner, checks the same file from the target
// side, so a plan identity that only one of them computes fails a test.
type simulationExplorationPlanVector struct {
	ExecutionSHA256  record.SHA256             `json:"execution_sha256"`
	ControllerSHA256 record.SHA256             `json:"controller_sha256"`
	BaseSeed         uint64                    `json:"base_seed"`
	Decision         simulationengine.Decision `json:"decision"`
	Forced           uint32                    `json:"forced"`
	RootPlan         string                    `json:"root_plan"`
	ForcedPlan       string                    `json:"forced_plan"`
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
	if vector.ControllerSHA256 != simulationengine.ImplementationSHA256() {
		t.Fatalf("vector controller = %s, want this implementation %s", vector.ControllerSHA256, simulationengine.ImplementationSHA256())
	}
	config := simulationengine.Config{
		ExecutionSHA256: vector.ExecutionSHA256, ControllerSHA256: vector.ControllerSHA256, BaseSeed: vector.BaseSeed,
		Parallel: 1, MaxExecutions: 2, MaxForcedDecisions: 1, MaxExplorationBytes: 1 << 20, MaxResultBytes: 1 << 20, FailureBudget: 1,
		Limits: simulationengine.DimensionLimits{Runtime: 1, Scenario: 1, Network: 1, Storage: 1, Fault: 1, Crash: 1},
	}
	root, err := simulationengine.CanonicalCandidate(config, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	rootPlan, err := simulationrecord.PlanForCandidate(config, root)
	if err != nil {
		t.Fatal(err)
	}
	if string(rootPlan) != vector.RootPlan {
		t.Errorf("root plan = %s, want %s", rootPlan, vector.RootPlan)
	}

	decision, err := simulationengine.CanonicalDecision(
		vector.Decision.Dimension, vector.Decision.Ordinal, vector.Decision.SiteSHA256, vector.Decision.Alternatives, vector.Decision.Selected,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decision, vector.Decision) {
		t.Errorf("decision = %#v, want %#v", decision, vector.Decision)
	}
	forced, err := simulationengine.ForceDecision(decision, vector.Forced)
	if err != nil {
		t.Fatal(err)
	}
	child, err := simulationengine.CanonicalCandidate(config, []simulationengine.ForcedDecision{forced}, root.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	forcedPlan, err := simulationrecord.PlanForCandidate(config, child)
	if err != nil {
		t.Fatal(err)
	}
	if string(forcedPlan) != vector.ForcedPlan {
		t.Errorf("forced plan = %s, want %s", forcedPlan, vector.ForcedPlan)
	}
}
