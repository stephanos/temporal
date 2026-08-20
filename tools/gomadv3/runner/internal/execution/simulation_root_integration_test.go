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
			result, runErr := Run(context.Background(), Spec{
				SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
				BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
				Command:           target, Args: []string{"-test.run=^" + testName + "$", "-test.v", "-test.timeout=20s"}, Argv0: "gomadv3sim.test", Dir: t.TempDir(),
				Env: []string{"GOMADSEED=89", "TZ=UTC"}, RunTimeout: 30 * time.Second, TerminateGrace: 2 * time.Second, OutputLimit: 1 << 20,
				World:      WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: 89},
				Simulation: &SimulationCapability{Role: SimulationRoleCoordinator},
			})
			if runErr != nil {
				t.Fatalf("run root simulation target: %v: stdout=%s stderr=%s", runErr, result.Stdout.Bytes, result.Stderr.Bytes)
			}
			if result.Termination != TerminationExit || result.ExitCode != 0 || !result.GroupGone || !strings.Contains(string(result.Stdout.Bytes), "--- PASS: "+testName) {
				t.Fatalf("root simulation termination=%s exit=%d signal=%s watchdog=%t group-gone=%t stdout=%s stderr=%s", result.Termination, result.ExitCode, result.Signal, result.WatchdogTimeout, result.GroupGone, result.Stdout.Bytes, result.Stderr.Bytes)
			}
		})
	}
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
