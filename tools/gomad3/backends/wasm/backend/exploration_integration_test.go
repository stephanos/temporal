package backend

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/backends/wasm/internal/stockgo"
	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
	runnerbackend "go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/target"
)

const explorationGuest = `package main
import ("fmt"; "os")
func choose() int {
	a, b := make(chan int, 1), make(chan int, 1)
	a <- 0; b <- 1
	select { case n := <-a: return n; case n := <-b: return n }
}
func main() {
	a, b := choose(), choose()
	fmt.Printf("choices=%d,%d\n", a, b)
	if a == 1 && b == 1 { os.Exit(17) }
}
`

func explorationProvider(t *testing.T) *Provider {
	t.Helper()
	stock := integrationProvider(t)
	options := stock.options
	helperBytes, err := os.ReadFile(options.HelperPath)
	if err != nil || string(record.HashBytes(helperBytes)) != options.HelperSHA256 {
		t.Fatalf("helper changed before campaign snapshot: %v", err)
	}
	options.HelperPath = filepath.Join(t.TempDir(), "gomad3-wasmhost")
	if err := os.WriteFile(options.HelperPath, helperBytes, 0700); err != nil {
		t.Fatal(err)
	}
	options.CompileCacheRoot = t.TempDir()
	if err := os.Chmod(options.CompileCacheRoot, 0700); err != nil {
		t.Fatal(err)
	}
	options.Config.Profile = wasi.CooperativeProfile
	options.Config.Clock.ReadStepNanos = 0
	options.RuntimeRoot, err = filepath.Abs("../../../runtimesemantics")
	if err != nil {
		t.Fatal(err)
	}
	// Cooperative preparation needs the distribution outside GOMODCACHE.
	options.CompilerPath = stockgo.Compiler(t)
	provider, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	return provider
}

func explorationSpec(t *testing.T, provider *Provider, source string) runner.CampaignSpec {
	t.Helper()
	root := t.TempDir()
	for name, data := range map[string]string{"go.mod": "module gomad.wasm.exploration.fixture\n\ngo 1.27.1\n", "main.go": source} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return runner.CampaignSpec{
		Backend: provider, Seeds: "7", Strategy: runner.StrategyChoiceExploration, Parallel: 2,
		ExecutionTimeout: time.Minute, OverallTimeout: 5 * time.Minute, TerminateGrace: 100 * time.Millisecond,
		OnFailure: runner.PolicyAll, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20,
		ChoiceTraceLimit: 1 << 20, MaxExecutions: 16, MaxChoiceDepth: 4, MaxExplorationBytes: 8 << 20,
		Artifacts: t.TempDir(), RunnerBuild: "fn156-exploration", Coverage: runner.CoverageChoice,
		KeepSuccesses: runner.KeepSuccessesAll, SuccessArtifactLimit: 16, SuccessBytesLimit: 256 << 20,
		Target: target.Spec{Backend: Name, Kind: target.KindGoRun, Source: ".", WorkingDir: root, BuildTags: []string{"test_dep"}},
	}
}

func canonicalRounds(t *testing.T, campaign string) map[string][]byte {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(campaign, "choice-exploration", "rounds", "*", "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("missing committed rounds: %v", err)
	}
	rounds := make(map[string][]byte, len(files))
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		rounds[filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file))] = data
	}
	return rounds
}

func verifyLogicalExecutions(t *testing.T, campaign string, want uint64) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(campaign, "choice-exploration", "rounds", "*", "executions", "*.json"))
	if err != nil || uint64(len(files)) != want {
		t.Fatalf("logical execution count=%d want=%d err=%v", len(files), want, err)
	}
	seen := map[uint64]bool{}
	for _, file := range files {
		var execution struct {
			Ordinal record.Uint64String `json:"selection_ordinal"`
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &execution); err != nil {
			t.Fatal(err)
		}
		ordinal := uint64(execution.Ordinal)
		if ordinal >= want || seen[ordinal] {
			t.Fatalf("lost or duplicate logical ordinal %d", ordinal)
		}
		seen[ordinal] = true
	}
}

func TestWASMChoiceCampaignCanonicalRoundsAndDetachedReplay(t *testing.T) {
	provider := explorationProvider(t)
	spec := explorationSpec(t, provider, explorationGuest)
	first, err := runner.Explore(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if first.Failures == 0 || first.Succeeded == 0 || first.ReplayDivergences != 0 || first.RetainedSuccesses != first.Succeeded || first.ChoiceExploration == nil || first.ChoiceExploration.DeepestPrefix < 2 {
		t.Fatalf("bounded branching canary campaign: %+v", first)
	}
	spec.Artifacts = t.TempDir()
	second, err := runner.Explore(t.Context(), spec)
	if err != nil || !reflect.DeepEqual(first.ChoiceExploration, second.ChoiceExploration) || !reflect.DeepEqual(canonicalRounds(t, first.CampaignPath), canonicalRounds(t, second.CampaignPath)) {
		t.Fatalf("canonical campaigns differ: first=%+v second=%+v err=%v", first.ChoiceExploration, second.ChoiceExploration, err)
	}
	verifyLogicalExecutions(t, first.CampaignPath, first.Attempted)
	bounded := spec
	bounded.Artifacts = t.TempDir()
	bounded.MaxExecutions = 1
	limited, err := runner.Explore(t.Context(), bounded)
	if err != nil || limited.Attempted != 1 || limited.StopReason != runner.StopMaxExecutions || limited.ChoiceExploration == nil || limited.ChoiceExploration.OmittedByExecutionBound == 0 {
		t.Fatalf("execution envelope was not reported: %+v err=%v", limited, err)
	}
	bounded.Artifacts = t.TempDir()
	bounded.MaxExecutions, bounded.MaxChoiceDepth = 16, 1
	limited, err = runner.Explore(t.Context(), bounded)
	if err != nil || limited.StopReason != runner.StopChoiceDepthComplete || limited.ChoiceExploration == nil || limited.ChoiceExploration.OmittedByDepth == 0 || limited.ChoiceExploration.DeepestPrefix != 1 {
		t.Fatalf("depth envelope was not reported: %+v err=%v", limited, err)
	}
	rootRoundKey := filepath.Join("00000000000000000000", "round.json")
	rootRound := canonicalRounds(t, first.CampaignPath)[rootRoundKey]
	if len(rootRound) == 0 {
		t.Fatal("missing canonical root round for byte-capacity envelope")
	}
	bounded.Artifacts = t.TempDir()
	bounded.MaxChoiceDepth = spec.MaxChoiceDepth
	// Admit the initial queue and its journal envelope, but no larger forced prefix.
	bounded.MaxExplorationBytes = uint64(len(rootRound))
	limited, err = runner.Explore(t.Context(), bounded)
	if err != nil || limited.StopReason != runner.StopExplorationCapacity || limited.Attempted != 1 || limited.ChoiceExploration == nil {
		t.Fatalf("byte envelope was not reported: %+v err=%v", limited, err)
	}
	capacity := limited.ChoiceExploration
	if capacity.MaxExplorationBytes != bounded.MaxExplorationBytes || capacity.OmittedByCapacity == 0 || capacity.Pending != 0 || capacity.PendingBytes != 0 || capacity.SeenPrefixes != 1 || capacity.LogicalExecutions != 1 || capacity.CommittedRounds != 1 || capacity.DeepestPrefix != 0 || capacity.BoundedComplete {
		t.Fatalf("byte-capacity queue/state escaped its envelope: %+v", capacity)
	}
	verifyLogicalExecutions(t, limited.CampaignPath, 1)
	capacityRounds := canonicalRounds(t, limited.CampaignPath)
	if len(capacityRounds) != 2 || !reflect.DeepEqual(rootRound, capacityRounds[rootRoundKey]) {
		t.Fatalf("byte-capacity campaign changed canonical root order: %v", capacityRounds)
	}
	for _, directory := range []string{spec.Target.WorkingDir, provider.options.CacheRoot} {
		if err := os.Rename(directory, directory+"-unavailable"); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := os.Rename(directory+"-unavailable", directory); err != nil {
				t.Error(err)
			}
		})
	}
	provider.options.CompilerPath = filepath.Join(t.TempDir(), "unavailable-go")
	provider.options.RuntimeRoot = filepath.Join(t.TempDir(), "unavailable-runtime")
	for _, path := range append(first.Artifacts, first.SuccessArtifacts...) {
		replay, err := runner.Replay(t.Context(), runner.ReplaySpec{Backend: provider, ArtifactPath: path})
		if err != nil || !replay.Match || replay.ChoiceReplayStatus != runner.ChoiceReplayExact || replay.ObservedRepetition {
			t.Fatalf("detached exact replay: %+v err=%v", replay, err)
		}
	}
	incompatible := *provider
	incompatible.options.Config.Profile = wasi.StockProfile
	tracked := &minimizationProvider{Provider: &incompatible}
	_, err = runner.Minimize(t.Context(), runner.MinimizeSpec{Backend: tracked, ArtifactPath: first.Artifacts[0], OutputRoot: t.TempDir(), AttemptBudget: 8})
	if err == nil || !strings.Contains(err.Error(), "replay profile mismatch") || len(tracked.controls) != 0 {
		t.Fatalf("incompatible replay profile must reject before minimization execution: %v controls=%v", err, tracked.controls)
	}
	t.Logf("logical=%d rounds=%d failures=%d retained-successes=%d", first.Attempted, first.ChoiceExploration.CommittedRounds, first.Failures, first.RetainedSuccesses)
}

type interruptRoundProvider struct {
	*Provider
	calls  atomic.Uint64
	cancel context.CancelFunc
}

func (p *interruptRoundProvider) Run(ctx context.Context, request runnerbackend.Request) (runnerbackend.Result, error) {
	result, err := p.Provider.Run(ctx, request)
	if p.calls.Add(1) == 2 {
		p.cancel()
	}
	return result, err
}

func TestWASMChoiceCampaignInterruptedRoundRecoveryNoDuplicates(t *testing.T) {
	provider := explorationProvider(t)
	spec := explorationSpec(t, provider, explorationGuest)
	baseline, err := runner.Explore(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	spec.Artifacts = t.TempDir()
	spec.Backend = &interruptRoundProvider{Provider: provider, cancel: cancel}
	partial, err := runner.Explore(ctx, spec)
	if !errors.Is(err, context.Canceled) || partial.Attempted != 1 {
		t.Fatalf("interrupt inside second round: %+v err=%v", partial, err)
	}
	if _, err := runner.Recover(t.Context(), partial.CampaignPath); err != nil {
		t.Fatal(err)
	}
	resumed, err := runner.Resume(t.Context(), runner.ResumeSpec{Backend: provider, CampaignPath: partial.CampaignPath, RunnerBuild: spec.RunnerBuild})
	if err != nil || resumed.Attempted != baseline.Attempted || resumed.RecoveryExecutions == 0 || !reflect.DeepEqual(baseline.ChoiceExploration, resumed.ChoiceExploration) || !reflect.DeepEqual(canonicalRounds(t, baseline.CampaignPath), canonicalRounds(t, resumed.CampaignPath)) {
		t.Fatalf("resumed campaign differs: baseline=%+v resumed=%+v err=%v", baseline, resumed, err)
	}
	verifyLogicalExecutions(t, resumed.CampaignPath, resumed.Attempted)
	if _, err := runner.Resume(t.Context(), runner.ResumeSpec{Backend: provider, CampaignPath: resumed.CampaignPath, RunnerBuild: spec.RunnerBuild}); err == nil {
		t.Fatal("published campaign was executed twice")
	}
	t.Logf("logical=%d recovery=%d canonical-rounds=%d", resumed.Attempted, resumed.RecoveryExecutions, resumed.ChoiceExploration.CommittedRounds)
}

func TestWASMChoiceCampaignFuelCapacityIsNotApplicationFailure(t *testing.T) {
	provider := explorationProvider(t)
	provider.options.Fuel = 100000000
	source := strings.Replace(explorationGuest, `if a == 1 && b == 1 { os.Exit(17) }`, `if os.Getenv("CPU_LOOP_EXIT") == "yes" { os.Exit(0) }; for {}`, 1)
	spec := explorationSpec(t, provider, source)
	result, err := runner.Explore(t.Context(), spec)
	var failure *runnerbackend.FailureError
	if !errors.As(err, &failure) || failure.Termination != runnerbackend.Capacity || result.Failures != 0 || result.Attempted != 0 || len(result.Artifacts) != 0 {
		t.Fatalf("CPU loop must stop with fuel capacity, not an application invariant: %+v err=%v", result, err)
	}
	if !strings.Contains(strings.ToLower(failure.Message), "fuel") {
		t.Fatalf("CPU loop exhausted a different capacity: %v", err)
	}
	files, globErr := filepath.Glob(filepath.Join(result.CampaignPath, "choice-exploration", "rounds", "*", "segment.json"))
	if globErr != nil || len(files) != 0 {
		t.Fatalf("capacity round was committed: %v err=%v", files, globErr)
	}
	if strings.Contains(err.Error(), "nonzero_exit") {
		t.Fatalf("fuel capacity fabricated a guest exit: %v", err)
	}
}
