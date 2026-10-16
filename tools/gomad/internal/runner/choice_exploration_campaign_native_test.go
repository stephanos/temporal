//go:build gomad_native

package runner

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends/native"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
)

func TestRunChoiceExplorationFindsPinnedOutcomesWithinSeedSamplingBudget(t *testing.T) {
	toolchainRoot, err := filepath.Abs(filepath.Join("..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	testdata, err := filepath.Abs(filepath.Join("..", "backends", "native", "conformance", "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	supervisor := filepath.Join(t.TempDir(), "gomad")
	command := exec.Command(filepath.Join(toolchainRoot, "bin", "go"), "build", "-trimpath", "-o", supervisor, "./cmd/gomad")
	command.Dir = filepath.Join(testdata, "..", "..", "..", "..", "..")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build pinned supervisor: %v: %s", err, output)
	}
	config := CampaignSpec{
		Parallel:         2,
		ExecutionTimeout: 10 * time.Second, OverallTimeout: time.Minute, TerminateGrace: time.Second,
		OnFailure: PolicyAll, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20,
		KeepSuccesses: KeepSuccessesAll, SuccessArtifactLimit: 16, SuccessBytesLimit: 128 << 20,
		Target: target.Spec{
			Kind: target.KindGoRun, Source: "./choice_exploration", WorkingDir: testdata, ToolchainRoot: toolchainRoot,
		},
		Backend: native.New(native.Config{ToolchainRoot: toolchainRoot, SupervisorCommand: []string{supervisor, "__supervisor"}}), RunnerBuild: "sha256:" + strings.Repeat("0", 64),
	}
	seedConfig := config
	seedConfig.Strategy = StrategySeed
	seedConfig.Seeds = "211-226"
	seedConfig.Artifacts = t.TempDir()
	seedSummary, err := Explore(context.Background(), seedConfig)
	if err != nil {
		t.Fatal(err)
	}
	explorationConfig := config
	explorationConfig.Strategy = StrategyChoiceExploration
	explorationConfig.Seeds = "211"
	explorationConfig.ChoiceTraceLimit = 1 << 20
	explorationConfig.MaxExecutions = 16
	explorationConfig.MaxChoiceDepth = 32
	explorationConfig.MaxExplorationBytes = 4 << 20
	explorationConfig.Artifacts = t.TempDir()
	explorationSummary, err := Explore(context.Background(), explorationConfig)
	if err != nil {
		t.Fatal(err)
	}
	seedOutcomes := distinctSuccessStdoutOutcomes(t, seedSummary.SuccessArtifacts)
	explorationOutcomes := distinctSuccessStdoutOutcomes(t, explorationSummary.SuccessArtifacts)
	if seedSummary.Attempted != 16 || len(seedOutcomes) != 2 {
		t.Fatalf("equal-budget seed summary = %#v, outcomes = %v", seedSummary, seedOutcomes)
	}
	if explorationSummary.Attempted != 2 || explorationSummary.StopReason != StopExplorationExhausted || len(explorationOutcomes) != 2 || explorationSummary.ChoiceExploration == nil || explorationSummary.ChoiceExploration.DeduplicatedOutcomes != 2 {
		t.Fatalf("equal-budget exploration summary = %#v, outcomes = %v", explorationSummary, explorationOutcomes)
	}
	for outcome := range seedOutcomes {
		if _, ok := explorationOutcomes[outcome]; !ok {
			t.Fatalf("seed outcome %s missing from exploration outcomes %v", outcome, explorationOutcomes)
		}
	}
	if uint64(len(explorationOutcomes))*seedSummary.Attempted < uint64(len(seedOutcomes))*explorationSummary.Attempted {
		t.Fatalf("exploration outcomes per execution below seed sampling: seed=%d/%d exploration=%d/%d", len(seedOutcomes), seedSummary.Attempted, len(explorationOutcomes), explorationSummary.Attempted)
	}
}
