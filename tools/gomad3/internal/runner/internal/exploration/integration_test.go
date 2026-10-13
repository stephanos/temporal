//go:build integration

package exploration_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/backends/native/execution"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/runner/internal/exploration"
)

func rootSimulationTarget(t *testing.T) (root, toolchainRoot, source string) {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root, filepath.Join(root, ".toolchain"), "./internal/model/cluster"
}

func TestRootSimulationTargetResolvesInGomadModule(t *testing.T) {
	root, _, source := rootSimulationTarget(t)
	command := exec.CommandContext(t.Context(), "go", "list", "-mod=readonly", "-tags", "test_dep,gomad3_toolchain", "-f", "{{.Module.Path}}|{{.ImportPath}}", source)
	command.Dir = root
	command.Env = append(filteredSimulationBuildEnvironment(os.Environ()), "GOTOOLCHAIN=local", "GOWORK=off", "GOFLAGS=")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("resolve root simulation target: %v: %s", err, output)
	}
	const expected = "go.temporal.io/server/tools/gomad3|go.temporal.io/server/tools/gomad3/internal/model/cluster"
	if strings.TrimSpace(string(output)) != expected {
		t.Fatalf("root simulation target ownership = %q, want %q", output, expected)
	}
}

func TestRootProcessSimulationUsesRunnerTransport(t *testing.T) {
	root, toolchainRoot, source := rootSimulationTarget(t)
	target := filepath.Join(t.TempDir(), "gomad3sim.test")
	command := exec.CommandContext(context.Background(), filepath.Join(toolchainRoot, "bin", "go"), "test", "-c", "-trimpath", "-tags", "test_dep,gomad3_toolchain", "-o", target, source)
	command.Dir = root
	command.Env = append(filteredSimulationBuildEnvironment(os.Environ()), "CGO_ENABLED=0", "GOEXPERIMENT=nogreenteagc", "GOTOOLCHAIN=local", "GOWORK=off")
	if output, buildErr := command.CombinedOutput(); buildErr != nil {
		t.Fatalf("build root simulation target: %v: %s", buildErr, output)
	}

	for _, testName := range []string{
		"TestScenarioChoicePlanForcesRankBoundDecisionAndExactlyReplays",
		"TestScenarioChoicePlanRejectsChangedDecisionBeforeSelection",
		"TestProcessExplorationConsumesExternalPlanAndPublishesRecord",
		"TestProcessBackendCarriesForwardClockAcrossActivationAndQuiescence",
		"TestProcessBackendResetsGlobalsDescriptorsAndGoroutines",
		"TestProcessAndInProcessBackendsHaveEquivalentDetachedModels",
		"TestProcessBackendRoutesTCPThroughSharedHostModel",
		"TestProcessNetworkHandleOperationParity",
		"TestProcessNetworkHandleQueuedAccept",
		"TestProcessNetworkHandleShortReadsAndHalfClose",
		"TestProcessNetworkHandleEmptyIO",
		"TestProcessNetworkHandleDeadlines",
		"TestProcessNetworkHandlePartialWrite",
		"TestProcessNetworkHandleRefusedDial",
		"TestProcessNetworkHandleByteBoundaries",
		"TestProcessNetworkHandleFaultConformance",
		"TestProcessNetworkHandleCapacityPreservesIdentities",
		"TestProcessNetworkHandleConnectionCapacity",
		"TestProcessNetworkHandleTinyWritesUseByteCapacity",
		"TestProcessNetworkHandleByteCapacity",
		"TestProcessNetworkHandleRevocationDropsDelayedData",
		"TestProcessNetworkHandleCrashDropsDelayedData",
		"TestProcessNetworkHandleReplayRejectsWriteBeforeMutation",
		"TestProcessNetworkHeldFIFO",
		"TestProcessNetworkStallExpiry",
		"TestProcessNetworkStallOrdering",
		"TestProcessNetworkPartitionedDialStall",
		"TestProcessNetworkFramedPartition",
		"TestProcessFilesystemHandlePartialIOAndLifetime",
		"TestProcessFilesystemHandleDirectoryAndChdir",
		"TestProcessFilesystemHandleAccessAndBounds",
		"TestProcessFilesystemHandleReplayRejectsWriteBeforeMutation",
		"TestProcessBackendSynchronizesNodeClockWithModelDelay",
		"TestProcessBackendRoutesListenThroughSharedHostModel",
		"TestProcessBackendPreservesHostVolumeAcrossRestart",
		"TestProcessBackendCrashDrainsInflightModelOperationDeterministically",
		"TestProcessBackendModelDigestsIgnoreCompletionOrder",
	} {
		environments := []struct {
			suffix string
			value  []string
		}{{value: []string{"GOMADSEED=89", "TZ=UTC"}}}
		if testName == "TestProcessBackendCarriesForwardClockAcrossActivationAndQuiescence" || testName == "TestProcessBackendSynchronizesNodeClockWithModelDelay" {
			environments[0].suffix = "/strict"
			environments = append(environments, struct {
				suffix string
				value  []string
			}{suffix: "/forward", value: []string{"GOMADSEED=89", "TZ=UTC", record.ClockTickEnvironment + "=" + record.ClockTickForward}})
		}
		for _, environment := range environments {
			t.Run(testName+environment.suffix, func(t *testing.T) {
				simulation := &execution.SimulationCapability{Role: execution.SimulationRoleCoordinator}
				var explorationConfig exploration.SimulationConfig
				var explorationCandidate exploration.SimulationCandidate
				if testName == "TestProcessExplorationConsumesExternalPlanAndPublishesRecord" {
					explorationConfig, explorationCandidate, simulation.ExplorationPlan = rootExplorationPlan(t, 89)
					simulation.ExplorationRecordLimit = 1 << 20
					simulation.ExplorationRecordCount = 1
				}
				result, runErr := execution.Run(context.Background(), execution.Spec{
					SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
					BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
					Command:           target, Args: []string{"-test.run=^" + testName + "$", "-test.v", "-test.timeout=20s"}, Argv0: "gomad3sim.test", Dir: t.TempDir(),
					Env: environment.value, ExecutionTimeout: 30 * time.Second, TerminateGrace: 2 * time.Second, OutputLimit: 1 << 20,
					World:      execution.WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: 89},
					Simulation: simulation,
				})
				if runErr != nil {
					t.Fatalf("run root simulation target: %v: stdout=%s stderr=%s", runErr, result.Stdout.Bytes, result.Stderr.Bytes)
				}
				if result.Termination != execution.TerminationExit || result.ExitCode != 0 || !result.GroupGone || !strings.Contains(string(result.Stdout.Bytes), "--- PASS: "+testName) {
					t.Fatalf("root simulation termination=%s exit=%d signal=%s watchdog=%t group-gone=%t stdout=%s stderr=%s", result.Termination, result.ExitCode, result.Signal, result.WatchdogTimeout, result.GroupGone, result.Stdout.Bytes, result.Stderr.Bytes)
				}
				if testName == "TestProcessExplorationConsumesExternalPlanAndPublishesRecord" && len(result.SimulationRecords) != 1 {
					t.Fatalf("simulation exploration records = %d, want 1", len(result.SimulationRecords))
				}
				if testName == "TestProcessExplorationConsumesExternalPlanAndPublishesRecord" {
					if _, err := exploration.ResultForRecord(explorationConfig, explorationCandidate, result.SimulationRecords[0], nil); err != nil {
						t.Fatalf("project simulation exploration record: %v", err)
					}
				}
			})
		}
	}
}

func TestSupervisorHelper(t *testing.T) {
	if os.Getenv("GOMAD3_PROCESS_SUPERVISOR") != "1" {
		t.Skip("supervisor subprocess only")
	}
	if err := execution.SupervisorMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	os.Exit(0)
}

func TestTargetBootstrapHelper(t *testing.T) {
	if os.Getenv("GOMAD3_TARGET_BOOTSTRAP") != "1" {
		t.Skip("target bootstrap subprocess only")
	}
	if err := execution.BootstrapMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	os.Exit(0)
}

func rootExplorationPlan(t *testing.T, seed uint64) (exploration.SimulationConfig, exploration.SimulationCandidate, []byte) {
	t.Helper()
	config := exploration.SimulationConfig{
		ExecutionSHA256:  record.HashBytes([]byte("root process exploration integration")),
		ControllerSHA256: exploration.SimulationImplementationSHA256(), BaseSeed: seed,
		Parallel: 1, MaxExecutions: 2, MaxForcedDecisions: 1, MaxExplorationBytes: 1 << 20, MaxResultBytes: 1 << 20, FailureBudget: 1,
		Limits: exploration.DimensionLimits{Runtime: 1, Scenario: 1, Network: 1, Storage: 1, Fault: 1, Crash: 1},
	}
	state, err := exploration.NewSimulation(config)
	if err != nil {
		t.Fatal(err)
	}
	round, ok := state.NextRound()
	if !ok || len(round.Candidates) != 1 {
		t.Fatal("simulation exploration root candidate is unavailable")
	}
	encoded, err := exploration.PlanForCandidate(config, round.Candidates[0])
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
		case "CGO_ENABLED", "GOEXPERIMENT", "GOMADSEED", "GOMAD3_CHILD_SEED", "GOTOOLCHAIN", "GOWORK":
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}
