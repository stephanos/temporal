//go:build integration

package execution

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/combinedfrontier"
	"go.temporal.io/server/tools/gomadv3/runner/internal/simulationexploration"
)

func TestRootProcessSimulationUsesRunnerTransport(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	toolchainRoot := filepath.Join(root, "tools", "gomadv3", ".toolchain")
	target := filepath.Join(t.TempDir(), "gomadv3sim.test")
	command := exec.CommandContext(context.Background(), filepath.Join(toolchainRoot, "bin", "go"), "test", "-c", "-trimpath", "-tags", "test_dep,gomadv3_toolchain", "-o", target, "./tools/gomadv3sim")
	command.Dir = root
	command.Env = append(filteredSimulationBuildEnvironment(os.Environ()), "CGO_ENABLED=0", "GOTOOLCHAIN=local", "GOWORK=off")
	if output, buildErr := command.CombinedOutput(); buildErr != nil {
		t.Fatalf("build root simulation target: %v: %s", buildErr, output)
	}

	for _, testName := range []string{
		"TestScenarioChoicePlanForcesRankBoundDecisionAndExactlyReplays",
		"TestScenarioChoicePlanRejectsChangedDecisionBeforeSelection",
		"TestProcessExplorationConsumesExternalPlanAndPublishesRecord",
		"TestProcessBackendResetsGlobalsDescriptorsAndGoroutines",
		"TestProcessAndInProcessBackendsHaveEquivalentDetachedModels",
		"TestProcessBackendRoutesTCPThroughSharedHostModel",
		"TestProcessBackendSynchronizesNodeClockWithModelDelay",
		"TestProcessBackendRoutesListenThroughSharedHostModel",
		"TestProcessBackendPreservesHostVolumeAcrossRestart",
		"TestProcessBackendCrashDrainsInflightModelOperationDeterministically",
		"TestProcessBackendModelDigestsIgnoreCompletionOrder",
	} {
		t.Run(testName, func(t *testing.T) {
			simulation := &SimulationCapability{Role: SimulationRoleCoordinator}
			var explorationConfig combinedfrontier.Config
			var explorationCandidate combinedfrontier.Candidate
			if testName == "TestProcessExplorationConsumesExternalPlanAndPublishesRecord" {
				explorationConfig, explorationCandidate, simulation.ExplorationPlan = rootExplorationPlan(t, 89)
				simulation.ExplorationRecordLimit = 1 << 20
				simulation.ExplorationRecordCount = 1
			}
			result, runErr := Run(context.Background(), Spec{
				SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
				BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
				Command:           target, Args: []string{"-test.run=^" + testName + "$", "-test.v", "-test.timeout=20s"}, Argv0: "gomadv3sim.test", Dir: t.TempDir(),
				Env: []string{"GOMADSEED=89", "TZ=UTC"}, RunTimeout: 30 * time.Second, TerminateGrace: 2 * time.Second, OutputLimit: 1 << 20,
				World:      WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: 89},
				Simulation: simulation,
			})
			if runErr != nil {
				t.Fatalf("run root simulation target: %v: stdout=%s stderr=%s", runErr, result.Stdout.Bytes, result.Stderr.Bytes)
			}
			if result.Termination != TerminationExit || result.ExitCode != 0 || !result.GroupGone || !strings.Contains(string(result.Stdout.Bytes), "--- PASS: "+testName) {
				t.Fatalf("root simulation termination=%s exit=%d signal=%s watchdog=%t group-gone=%t stdout=%s stderr=%s", result.Termination, result.ExitCode, result.Signal, result.WatchdogTimeout, result.GroupGone, result.Stdout.Bytes, result.Stderr.Bytes)
			}
			if testName == "TestProcessExplorationConsumesExternalPlanAndPublishesRecord" && len(result.SimulationRecords) != 1 {
				t.Fatalf("simulation exploration records = %d, want 1", len(result.SimulationRecords))
			}
			if testName == "TestProcessExplorationConsumesExternalPlanAndPublishesRecord" {
				if _, err := simulationexploration.ResultForRecord(explorationConfig, explorationCandidate, result.SimulationRecords[0], nil); err != nil {
					t.Fatalf("project simulation exploration record: %v", err)
				}
			}
		})
	}
}

func rootExplorationPlan(t *testing.T, seed uint64) (combinedfrontier.Config, combinedfrontier.Candidate, []byte) {
	t.Helper()
	config := combinedfrontier.Config{
		ExecutionSHA256:  evidence.HashBytes([]byte("root process exploration integration")),
		ControllerSHA256: combinedfrontier.ImplementationSHA256(), BaseSeed: seed,
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
	encoded, err := simulationexploration.PlanForCandidate(config, round.Candidates[0])
	if err != nil {
		t.Fatal(err)
	}
	return config, round.Candidates[0], encoded
}

func filteredSimulationBuildEnvironment(environment []string) []string {
	filtered := make([]string, 0, len(environment))
	for _, entry := range environment {
		name, _, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		switch name {
		case "CGO_ENABLED", "GOMADSEED", "GOMADV3_CHILD_SEED", "GOTOOLCHAIN", "GOWORK":
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}
