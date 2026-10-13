package runner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/artifact"
	backend "go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/backends/native"
	nativebuild "go.temporal.io/server/tools/gomad3/backends/native/build"
	"go.temporal.io/server/tools/gomad3/backends/native/preparation"
	"go.temporal.io/server/tools/gomad3/backends/native/transcript"
	"go.temporal.io/server/tools/gomad3/captured"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	guide "go.temporal.io/server/tools/gomad3/runner/internal/corpus"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad3/world"
)

const (
	probeA = "stdlib.os.openfile"
	probeB = "stdlib.os.file.read"
	probeC = "stdlib.os.file.close"
)

// retentionExecutor runs the three executions every retention campaign has,
// gives each a World record of its seed, and identifies each by its rank: the seed strategy runs seeds 1 to 3 as ranks 0
// to 2, the exploration strategies run the root as rank 0 and its two other
// alternatives as ranks 1 and 2, which share the second round when the
// campaign runs two candidates at a time.
type retentionExecutor struct {
	t        *testing.T
	strategy Strategy
	base     executionRunner
	// order maps the alternative an exploration candidate forces to its rank;
	// an alternative it does not hold is its own rank.
	order map[uint64]uint64
	// shape edits the captured result of the execution with the given rank.
	shape func(rank uint64, result *backend.ExecutionResult)
	// before runs ahead of the execution with the given rank.
	before func(rank uint64)
	// after names, per rank, the rank whose completion it waits for.
	after map[uint64]uint64
	// interrupt cancels the campaign when the execution with the rank
	// interruptAt starts; that execution then reports its cancellation.
	interrupt   context.CancelFunc
	interruptAt uint64

	mu       sync.Mutex
	finished map[uint64]chan struct{}
}

func (executor *retentionExecutor) rank(request backend.ExecutionRequest) uint64 {
	alternative := executor.alternative(request)
	if rank, ordered := executor.order[alternative]; ordered {
		return rank
	}
	return alternative
}

// alternative is the seed's position for the seed strategy and, for the
// exploration strategies, the alternative the candidate forces last.
func (executor *retentionExecutor) alternative(request backend.ExecutionRequest) uint64 {
	switch executor.strategy {
	case StrategyChoiceExploration:
		if request.Choice != nil && request.Choice.Mode == choice.ModePrefix {
			return uint64(request.Choice.ReplayPlan.Decisions[len(request.Choice.ReplayPlan.Decisions)-1].Selected)
		}
		return 0
	case StrategySimulationExploration:
		var plan struct {
			Overrides []struct {
				Selected uint64 `json:"selected"`
			} `json:"overrides"`
		}
		if request.Simulation == nil || json.Unmarshal(request.Simulation.ExplorationPlan, &plan) != nil {
			executor.t.Error("simulation exploration plan is unavailable")
			return 0
		}
		if len(plan.Overrides) == 0 {
			return 0
		}
		return plan.Overrides[len(plan.Overrides)-1].Selected
	default:
		return seedFromEnvironment(request.Env) - 1
	}
}

func (executor *retentionExecutor) completion(rank uint64) chan struct{} {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	if executor.finished == nil {
		executor.finished = make(map[uint64]chan struct{})
	}
	if executor.finished[rank] == nil {
		executor.finished[rank] = make(chan struct{})
	}
	return executor.finished[rank]
}

func (executor *retentionExecutor) Run(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	rank := executor.rank(request)
	defer close(executor.completion(rank))
	if executor.before != nil {
		executor.before(rank)
	}
	if executor.interrupt != nil && rank == executor.interruptAt {
		executor.interrupt()
		<-ctx.Done()
		result := processResult(0, "", "")
		result.Cancelled = true
		result.Termination = backend.TerminationSignal
		result.Signal = "killed"
		return result, nil
	}
	if other, waits := executor.after[rank]; waits {
		select {
		case <-executor.completion(other):
		case <-ctx.Done():
			return backend.ExecutionResult{}, ctx.Err()
		}
	}
	result, err := executor.base.Run(ctx, request)
	if err == nil {
		result.WorldRecord = completionWorldRecord(executor.t, request.World.Seed)
		if executor.shape != nil {
			executor.shape(rank, &result)
		}
	}
	return result, err
}

// retentionCampaign is a three-execution campaign of the strategy. The seed
// strategy runs all three at once; the exploration strategies run the root and
// then, with parallel candidates, both remaining alternatives in one round.
func retentionCampaign(t *testing.T, strategy Strategy, parallel bool, fail bool) (CampaignSpec, *retentionExecutor, executionDependencies) {
	t.Helper()
	config, executor, configDependencies := unorderedRetentionCampaign(t, strategy, parallel, fail)
	executor.order = explorationRanks(t, strategy)
	return config, executor, configDependencies
}

var explorationRankCache sync.Map

// explorationRanks maps each alternative an exploration strategy forces to the
// journal position of its execution. Candidates of one depth run in the order
// of their identities, which bind the platform, so the two alternatives beside
// the root run in either order. The fixture's identities are fixed, and one
// campaign whose alternatives observe distinct probes shows the order.
func explorationRanks(t *testing.T, strategy Strategy) map[uint64]uint64 {
	t.Helper()
	if strategy == StrategySeed {
		return nil
	}
	if cached, found := explorationRankCache.Load(strategy); found {
		return cached.(map[uint64]uint64)
	}
	probes := []string{probeA, probeB, probeC}
	config, executor, configDependencies := unorderedRetentionCampaign(t, strategy, false, false)
	keepSuccesses(&config, KeepSuccessesNovel)
	executor.shape = rankProbes(t, probes...)
	configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	order := make(map[uint64]uint64)
	for _, run := range journaledExecutions(t, summary.CampaignPath, true) {
		if len(run.NovelSemanticProbes) != 1 {
			t.Fatalf("execution %d found %v novel, want its own probe", run.SelectionOrdinal, run.NovelSemanticProbes)
		}
		order[uint64(slices.Index(probes, run.NovelSemanticProbes[0]))] = uint64(run.SelectionOrdinal)
	}
	if len(order) != len(probes) || order[0] != 0 {
		t.Fatalf("exploration order = %v, want the root first and every alternative once", order)
	}
	explorationRankCache.Store(strategy, order)
	return order
}

func unorderedRetentionCampaign(t *testing.T, strategy Strategy, parallel bool, fail bool) (CampaignSpec, *retentionExecutor, executionDependencies) {
	t.Helper()
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	executor := &retentionExecutor{t: t, strategy: strategy}
	width := 1
	switch strategy {
	case StrategySeed:
		executor.base = &fakeExecutor{result: func(seed uint64) backend.ExecutionResult {
			exitCode := 0
			if fail {
				exitCode = 2
			}
			result := processResult(exitCode, "", "")
			result.ChoiceTrace = completeChoiceTrace(t, preparer.prepared.BuildKey, limit, []choice.Record{{
				Ordinal: 0, Kind: choice.KindRunnable, Flags: choice.FlagDecision, Alternatives: 3, Selected: uint32(seed - 1),
			}})
			return result
		}}
		if parallel {
			width = 3
		}
	case StrategyChoiceExploration:
		base := &explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit, alternatives: 3}
		if fail {
			base.exitCode = 2
		}
		executor.base = base
		if parallel {
			width = 2
		}
	case StrategySimulationExploration:
		executor.base = &simulationExplorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit, scenarios: 3, fail: fail}
		if parallel {
			width = 2
		}
	}
	config, configDependencies := testConfig(t, preparer, executor, "7", PolicyAll, width)
	config.Strategy = strategy
	config.Coverage = CoverageSemantic
	config.ChoiceTraceLimit = limit
	config.WorldTransitionLimit = 1 << 20
	switch strategy {
	case StrategySeed:
		config.Seeds = "1-3"
	case StrategyChoiceExploration:
		config.MaxExecutions = 8
		config.MaxChoiceDepth = 4
		config.MaxExplorationBytes = 1 << 20
	case StrategySimulationExploration:
		config.MaxExecutions = 8
		config.MaxForcedDecisions = 2
		config.MaxExplorationBytes = 1 << 20
		config.MaxExplorationResultBytes = 1 << 20
		config.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 2, Scenario: 3, Network: 2, Storage: 2, Fault: 2, Crash: 2}
	}
	return config, executor, configDependencies
}

// rankProbes gives each rank a complete I/O transcript observing its probe.
func rankProbes(t *testing.T, byRank ...string) func(uint64, *backend.ExecutionResult) {
	return func(rank uint64, result *backend.ExecutionResult) {
		result.IOTranscript = semanticTranscript(t, byRank[rank])
	}
}

// reverseRankOrder makes the executions complete in the reverse of their rank order as
// far as the strategy runs them together.
func reverseRankOrder(strategy Strategy) map[uint64]uint64 {
	if strategy == StrategySeed {
		return map[uint64]uint64{0: 1, 1: 2}
	}
	return map[uint64]uint64{1: 2}
}

func keepSuccesses(config *CampaignSpec, policy KeepSuccesses) {
	config.KeepSuccesses = policy
	if policy != KeepSuccessesNone {
		config.SuccessArtifactLimit = 8
		config.SuccessBytesLimit = 64 << 20
	}
}

func resumeRetentionCampaign(t *testing.T, interrupted CampaignSpec, campaignPath string, shape func(uint64, *backend.ExecutionResult)) (CampaignResult, error) {
	t.Helper()
	_, executor, _ := retentionCampaign(t, interrupted.Strategy, interrupted.Parallel != 1, false)
	executor.shape = shape
	return exploreWith(context.Background(), CampaignSpec{
		ResumeCampaign: campaignPath, RunnerBuild: interrupted.RunnerBuild, Backend: scriptProvider(),
	}, executionDependencies{executor: executor},
	)
}

// retentionObservation is what one campaign leaves observable about retention:
// the host failure, the counters (attempted, succeeded, failures, distinct
// failures, retained successes) and, per journaled execution in journal order,
// its ordinal, outcome, whether its success was kept, what made it novel and
// which distinct failure it shares.
type retentionObservation struct {
	Reason string
	Cause  string
	Counts [5]uint64
	Runs   []string
}

// journaledExecutions reads the executions a campaign journaled: from the
// published campaign when it completed, from its execution journal otherwise.
func journaledExecutions(t *testing.T, campaignPath string, completed bool) []campaign.ExecutionRecord {
	t.Helper()
	if completed {
		batch, err := campaign.OpenCampaign(campaignPath)
		if err != nil {
			t.Fatal(err)
		}
		return batch.Executions
	}
	committed, err := campaign.ReadCommittedState(campaignPath)
	if err != nil {
		t.Fatal(err)
	}
	return committed.Executions
}

// observeRetention reads the campaign back and checks what must hold for every
// retention outcome: each kept success is an exact-replay success artifact of
// its own ordinal whose stored bytes the journal records, the summary lists the
// kept successes in journal order and counts their bytes, and every failure
// refers to a published distinct failure. It returns the observation and the
// canonical projection of the journal records and published manifests with the
// campaign identity, timestamps and store paths held fixed.
func observeRetention(t *testing.T, summary CampaignResult, err error) (retentionObservation, []byte) {
	t.Helper()
	observed := retentionObservation{Counts: [5]uint64{summary.Attempted, summary.Succeeded, summary.Failures, summary.DistinctFailures, summary.RetainedSuccesses}}
	if err != nil {
		var hostError *HostError
		if !errors.As(err, &hostError) {
			t.Fatalf("exploreWith() error = %#v, want a host failure", err)
		}
		observed.Reason = hostError.Reason
		observed.Cause = hostError.Err.Error()
	}
	fixedManifest := func(path string) record.ExecutionRecord {
		opened, openErr := artifact.OpenArtifact(path)
		if openErr != nil {
			t.Fatal(openErr)
		}
		manifest := opened.Manifest()
		if closeErr := opened.Close(); closeErr != nil {
			t.Fatal(closeErr)
		}
		manifest.CampaignID, manifest.CreatedAt, manifest.RecordHash, manifest.Host = "", "", "", record.Host{}
		return manifest
	}
	projection := struct {
		Runs      []campaign.ExecutionRecord
		Successes []record.ExecutionRecord
		Failures  []record.ExecutionRecord
	}{Runs: journaledExecutions(t, summary.CampaignPath, err == nil)}
	var kept []string
	var keptBytes uint64
	for index := range projection.Runs {
		run := &projection.Runs[index]
		run.ElapsedNanos = 0
		line := fmt.Sprintf("%d %s/%s", run.SelectionOrdinal, run.Domain, run.Reason)
		if run.SuccessArtifact != nil {
			path := filepath.Join(summary.CampaignPath, filepath.FromSlash(*run.SuccessArtifact))
			opened, openErr := artifact.OpenArtifact(path)
			if openErr != nil {
				t.Fatal(openErr)
			}
			if opened.StoredBytes() != uint64(*run.SuccessArtifactBytes) || opened.Manifest().ArtifactKind != record.ArtifactSuccess || opened.Manifest().ReplayMode != record.ReplayExact || opened.Manifest().SelectionOrdinal != run.SelectionOrdinal {
				t.Fatalf("kept success %s = %d bytes, %#v; journaled as %#v", path, opened.StoredBytes(), opened.Manifest(), *run)
			}
			if closeErr := opened.Close(); closeErr != nil {
				t.Fatal(closeErr)
			}
			keptBytes += opened.StoredBytes()
			kept = append(kept, path)
			projection.Successes = append(projection.Successes, fixedManifest(path))
			fixedPath, fixedBytes := fmt.Sprintf("successes/#%d", len(kept)-1), record.Uint64String(0)
			run.SuccessArtifact, run.SuccessArtifactBytes = &fixedPath, &fixedBytes
			line += " kept"
		}
		if len(run.NovelSemanticProbes) != 0 {
			line += " novel=" + strings.Join(run.NovelSemanticProbes, ",")
		}
		if len(run.NovelChoiceFeatures) != 0 {
			line += " +choices"
		}
		if run.Artifact != nil {
			path := filepath.Join(summary.CampaignPath, filepath.FromSlash(*run.Artifact))
			distinct := -1
			for candidate, published := range summary.Artifacts {
				if published == path {
					distinct = candidate
				}
			}
			if distinct < 0 || run.FailureSignature == nil {
				t.Fatalf("failure %#v does not refer to a published distinct failure in %v", *run, summary.Artifacts)
			}
			fixedPath := fmt.Sprintf("failures/#%d", distinct)
			run.Artifact = &fixedPath
			line += " failure#" + fmt.Sprint(distinct)
		}
		observed.Runs = append(observed.Runs, line)
	}
	if !reflect.DeepEqual(kept, summary.SuccessArtifacts) || keptBytes != summary.RetainedSuccessBytes || uint64(len(kept)) != summary.RetainedSuccesses {
		t.Fatalf("summary keeps %v (%d successes, %d bytes); the journal keeps %v (%d bytes)", summary.SuccessArtifacts, summary.RetainedSuccesses, summary.RetainedSuccessBytes, kept, keptBytes)
	}
	for _, path := range summary.Artifacts {
		projection.Failures = append(projection.Failures, fixedManifest(path))
	}
	encoded, encodeErr := canonicaljson.CanonicalJSON(projection)
	if encodeErr != nil {
		t.Fatal(encodeErr)
	}
	return observed, encoded
}

// Every strategy keeps the same executions under the same policy, and what it
// journals and publishes does not depend on the order its executions complete.
func TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy(t *testing.T) {
	for _, test := range []struct {
		name     string
		policy   KeepSuccesses
		coverage CoverageMode
		probes   [3]string
		fail     bool
		want     retentionObservation
		// other holds the simulation strategy where its fixture differs: its
		// candidates record no choice decision of their own, and their failures
		// share the failure identity of the simulation record, where the seeds
		// and the forced choices of the other strategies fail distinctly.
		other map[Strategy]retentionObservation
	}{
		{
			name: "discard", policy: KeepSuccessesNone, probes: [3]string{probeA, probeA, probeB},
			want: retentionObservation{Counts: [5]uint64{3, 3, 0, 0, 0}, Runs: []string{"0 success/world_idle", "1 success/world_idle", "2 success/world_idle"}},
		},
		{
			name: "all", policy: KeepSuccessesAll, probes: [3]string{probeA, probeA, probeB},
			want: retentionObservation{Counts: [5]uint64{3, 3, 0, 0, 3}, Runs: []string{"0 success/world_idle kept", "1 success/world_idle kept", "2 success/world_idle kept"}},
		},
		{
			name: "novel probe after a known one", policy: KeepSuccessesNovel, probes: [3]string{probeA, probeA, probeB},
			want: retentionObservation{Counts: [5]uint64{3, 3, 0, 0, 2}, Runs: []string{"0 success/world_idle kept novel=" + probeA, "1 success/world_idle", "2 success/world_idle kept novel=" + probeB}},
		},
		{
			name: "novel probe ahead of its repeat", policy: KeepSuccessesNovel, probes: [3]string{probeA, probeB, probeB},
			want: retentionObservation{Counts: [5]uint64{3, 3, 0, 0, 2}, Runs: []string{"0 success/world_idle kept novel=" + probeA, "1 success/world_idle kept novel=" + probeB, "2 success/world_idle"}},
		},
		{
			name: "novel choices", policy: KeepSuccessesNovel, coverage: CoverageSemanticChoice, probes: [3]string{probeA, probeA, probeA},
			want: retentionObservation{Counts: [5]uint64{3, 3, 0, 0, 3}, Runs: []string{"0 success/world_idle kept novel=" + probeA + " +choices", "1 success/world_idle kept +choices", "2 success/world_idle kept +choices"}},
			other: map[Strategy]retentionObservation{StrategySimulationExploration: {
				Counts: [5]uint64{3, 3, 0, 0, 1}, Runs: []string{"0 success/world_idle kept novel=" + probeA + " +choices", "1 success/world_idle", "2 success/world_idle"},
			}},
		},
		{
			name: "failures", policy: KeepSuccessesAll, probes: [3]string{probeA, probeA, probeA}, fail: true,
			want: retentionObservation{Counts: [5]uint64{3, 0, 3, 3, 0}, Runs: []string{"0 target/nonzero_exit failure#0", "1 target/nonzero_exit failure#1", "2 target/nonzero_exit failure#2"}},
			other: map[Strategy]retentionObservation{StrategySimulationExploration: {
				Counts: [5]uint64{3, 0, 3, 1, 0}, Runs: []string{"0 target/nonzero_exit failure#0", "1 target/nonzero_exit failure#0", "2 target/nonzero_exit failure#0"},
			}},
		},
	} {
		for _, strategy := range completionStrategies {
			t.Run(test.name+"/"+string(strategy), func(t *testing.T) {
				want := test.want
				if other, differs := test.other[strategy]; differs {
					want = other
				}
				var projections [][]byte
				for _, order := range []map[uint64]uint64{nil, reverseRankOrder(strategy)} {
					config, executor, configDependencies := retentionCampaign(t, strategy, true, test.fail)
					keepSuccesses(&config, test.policy)
					if test.coverage != "" {
						config.Coverage = test.coverage
					}
					executor.shape = rankProbes(t, test.probes[:]...)
					executor.after = order
					configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
					summary, err := exploreWith(context.Background(), config, configDependencies)
					observed, projection := observeRetention(t, summary, err)
					if !reflect.DeepEqual(observed, want) {
						t.Fatalf("retention = %#v, want %#v", observed, want)
					}
					projections = append(projections, projection)
				}
				if !bytes.Equal(projections[0], projections[1]) {
					t.Fatalf("projection depends on completion order:\n%s\n%s", projections[0], projections[1])
				}
				t.Logf("fixed-identity projection: %s", projections[0])
			})
		}
	}
}

const retentionExhausted = "successful-execution retention capacity is exhausted"

// A used-up success bound fails the campaign as a capacity failure and leaves
// the kept successes where the strategy last committed them: the seed strategy
// after every execution, the exploration strategies after every round.
func TestRetentionCapacityExhaustionFailsVisiblyForEveryStrategy(t *testing.T) {
	for _, strategy := range completionStrategies {
		run := func(t *testing.T, parallel bool, configure func(*CampaignSpec)) (CampaignResult, retentionObservation) {
			t.Helper()
			config, executor, configDependencies := retentionCampaign(t, strategy, parallel, false)
			keepSuccesses(&config, KeepSuccessesAll)
			configure(&config)
			executor.shape = rankProbes(t, probeA, probeA, probeB)
			if parallel {
				executor.after = reverseRankOrder(strategy)
			}
			configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
			summary, err := exploreWith(context.Background(), config, configDependencies)
			observed, _ := observeRetention(t, summary, err)
			return summary, observed
		}
		t.Run("count/"+string(strategy), func(t *testing.T) {
			_, observed := run(t, false, func(config *CampaignSpec) { config.SuccessArtifactLimit = 1 })
			want := retentionObservation{Reason: "success_retention_capacity", Cause: retentionExhausted, Counts: [5]uint64{2, 1, 0, 0, 1}, Runs: []string{"0 success/world_idle kept"}}
			if strategy != StrategySeed {
				want.Counts[0] = 1
			}
			if !reflect.DeepEqual(observed, want) {
				t.Fatalf("retention = %#v, want %#v", observed, want)
			}
		})
		// The third execution exhausts the bound after the second was kept. The
		// seed strategy committed the second; the exploration strategies kept it
		// in the round the third then failed.
		t.Run("count inside a round/"+string(strategy), func(t *testing.T) {
			_, observed := run(t, true, func(config *CampaignSpec) { config.SuccessArtifactLimit = 2 })
			want := retentionObservation{Reason: "success_retention_capacity", Cause: retentionExhausted, Counts: [5]uint64{3, 2, 0, 0, 2}, Runs: []string{"0 success/world_idle kept", "1 success/world_idle kept"}}
			if strategy != StrategySeed {
				want.Counts, want.Runs = [5]uint64{1, 1, 0, 0, 1}, want.Runs[:1]
			}
			if !reflect.DeepEqual(observed, want) {
				t.Fatalf("retention = %#v, want %#v", observed, want)
			}
		})
		// The byte bound is the size of one kept success. A success of exactly
		// that size uses the bound up, and the next one is rejected before it is
		// published; a success of another size is rejected by the store, which
		// is offered what the committed successes left of the bound.
		t.Run("bytes/"+string(strategy), func(t *testing.T) {
			measured, _ := run(t, false, func(*CampaignSpec) {})
			if measured.RetainedSuccesses != 3 {
				t.Fatalf("measured summary = %#v", measured)
			}
			limit := measured.RetainedSuccessBytes / 3
			config, executor, configDependencies := retentionCampaign(t, strategy, false, false)
			keepSuccesses(&config, KeepSuccessesAll)
			config.SuccessBytesLimit = limit
			executor.shape = rankProbes(t, probeA, probeA, probeB)
			configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
			summary, err := exploreWith(context.Background(), config, configDependencies)
			observed, _ := observeRetention(t, summary, err)
			var capacity *artifact.CapacityError
			switch {
			case observed.Reason != "success_retention_capacity" || summary.RetainedSuccesses > 1:
				t.Fatalf("retention = %#v, want a capacity failure at the first or second success", observed)
			case summary.RetainedSuccessBytes == limit:
				if observed.Cause != retentionExhausted {
					t.Fatalf("retention = %#v, want the used-up bound rejected before publication", observed)
				}
			case !errors.As(err, &capacity) || capacity.Maximum != limit-summary.RetainedSuccessBytes:
				t.Fatalf("retention = %#v with %d of %d bytes kept, want the store offered the remaining bytes", observed, summary.RetainedSuccessBytes, limit)
			}
		})
	}
}

// replaceWithFile makes a store root unusable until the returned function
// puts it back.
func replaceWithFile(t *testing.T, root string) func() {
	t.Helper()
	held := root + ".held"
	existed := true
	if err := os.Rename(root, held); os.IsNotExist(err) {
		existed = false
	} else if err != nil {
		t.Error(err)
	}
	if err := os.WriteFile(root, nil, 0o600); err != nil {
		t.Error(err)
	}
	return func() {
		if err := os.Remove(root); err != nil {
			t.Fatal(err)
		}
		if existed {
			if err := os.Rename(held, root); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// A campaign that fails or is cancelled while it retains a success leaves the
// counters and the novelty it judges later executions by where the strategy
// last committed them, and resuming it retains what an uninterrupted campaign
// retains.
func TestRetentionFailureLeavesNoveltyAndCountersAtTheCommittedState(t *testing.T) {
	const incomplete = "retained success requires a complete I/O transcript for exact replay"
	seedAndExploration := func(seed, exploration retentionObservation) map[Strategy]retentionObservation {
		return map[Strategy]retentionObservation{StrategySeed: seed, StrategyChoiceExploration: exploration, StrategySimulationExploration: exploration}
	}
	everyStrategy := func(observed retentionObservation) map[Strategy]retentionObservation {
		return seedAndExploration(observed, observed)
	}
	for _, test := range []struct {
		name     string
		policy   KeepSuccesses
		probes   [3]string
		parallel bool
		// incomplete is the rank whose transcript is incomplete, blocked the
		// rank that finds its success store unusable and cancelled the rank
		// that cancels the campaign; each is off at 3.
		incomplete, blocked, cancelled uint64
		interrupted                    map[Strategy]retentionObservation
		resumed                        map[Strategy]retentionObservation
	}{
		{
			// The second execution is kept before the third fails. The seed
			// strategy has committed it; the exploration strategies kept it in
			// the round that then failed, and keep it again on resume.
			name: "incomplete transcript behind a kept success", policy: KeepSuccessesNovel, probes: [3]string{probeA, probeB, probeC},
			parallel: true, incomplete: 2, blocked: 3, cancelled: 3,
			interrupted: seedAndExploration(
				retentionObservation{Reason: "success_artifact_publication", Cause: incomplete, Counts: [5]uint64{3, 2, 0, 0, 2}, Runs: []string{"0 success/world_idle kept novel=" + probeA, "1 success/world_idle kept novel=" + probeB}},
				retentionObservation{Reason: "success_artifact_publication", Cause: incomplete, Counts: [5]uint64{1, 1, 0, 0, 1}, Runs: []string{"0 success/world_idle kept novel=" + probeA}},
			),
			resumed: everyStrategy(retentionObservation{Counts: [5]uint64{3, 3, 0, 0, 3}, Runs: []string{"0 success/world_idle kept novel=" + probeA, "1 success/world_idle kept novel=" + probeB, "2 success/world_idle kept novel=" + probeC}}),
		},
		{
			// The second execution fails to retain the probe the third repeats.
			// The seed strategy has the third completed beside it with a
			// durable receipt, but commits outcomes only in ordinal order: the
			// resumed second execution retains the probe and the received third
			// then repeats it. The exploration strategies rerun the round in
			// order. Both resume to the uninterrupted retention.
			name: "incomplete transcript ahead of its repeat", policy: KeepSuccessesNovel, probes: [3]string{probeA, probeB, probeB},
			parallel: true, incomplete: 1, blocked: 3, cancelled: 3,
			interrupted: seedAndExploration(
				retentionObservation{Reason: "success_artifact_publication", Cause: incomplete, Counts: [5]uint64{2, 1, 0, 0, 1}, Runs: []string{"0 success/world_idle kept novel=" + probeA}},
				retentionObservation{Reason: "success_artifact_publication", Cause: incomplete, Counts: [5]uint64{1, 1, 0, 0, 1}, Runs: []string{"0 success/world_idle kept novel=" + probeA}},
			),
			resumed: everyStrategy(retentionObservation{Counts: [5]uint64{3, 3, 0, 0, 2}, Runs: []string{"0 success/world_idle kept novel=" + probeA, "1 success/world_idle kept novel=" + probeB, "2 success/world_idle"}}),
		},
		{
			name: "unusable success store", policy: KeepSuccessesAll, probes: [3]string{probeA, probeA, probeB},
			incomplete: 3, blocked: 1, cancelled: 3,
			interrupted: seedAndExploration(
				retentionObservation{Reason: "success_artifact_publication", Counts: [5]uint64{2, 1, 0, 0, 1}, Runs: []string{"0 success/world_idle kept"}},
				retentionObservation{Reason: "success_artifact_publication", Counts: [5]uint64{1, 1, 0, 0, 1}, Runs: []string{"0 success/world_idle kept"}},
			),
			resumed: everyStrategy(retentionObservation{Counts: [5]uint64{3, 3, 0, 0, 3}, Runs: []string{"0 success/world_idle kept", "1 success/world_idle kept", "2 success/world_idle kept"}}),
		},
		{
			name: "cancellation", policy: KeepSuccessesNovel, probes: [3]string{probeA, probeB, probeB},
			incomplete: 3, blocked: 3, cancelled: 2,
			interrupted: seedAndExploration(
				retentionObservation{Reason: "cancelled", Cause: "context canceled", Counts: [5]uint64{3, 2, 0, 0, 2}, Runs: []string{"0 success/world_idle kept novel=" + probeA, "1 success/world_idle kept novel=" + probeB}},
				retentionObservation{Reason: "cancelled", Cause: "context canceled", Counts: [5]uint64{2, 2, 0, 0, 2}, Runs: []string{"0 success/world_idle kept novel=" + probeA, "1 success/world_idle kept novel=" + probeB}},
			),
			resumed: everyStrategy(retentionObservation{Counts: [5]uint64{3, 3, 0, 0, 2}, Runs: []string{"0 success/world_idle kept novel=" + probeA, "1 success/world_idle kept novel=" + probeB, "2 success/world_idle"}}),
		},
	} {
		for _, strategy := range completionStrategies {
			t.Run(test.name+"/"+string(strategy), func(t *testing.T) {
				config, executor, configDependencies := retentionCampaign(t, strategy, test.parallel, false)
				keepSuccesses(&config, test.policy)
				shape := rankProbes(t, test.probes[:]...)
				executor.shape = func(rank uint64, result *backend.ExecutionResult) {
					shape(rank, result)
					result.IOTranscript.Complete = rank != test.incomplete
				}
				if test.parallel {
					executor.after = reverseRankOrder(strategy)
				}
				restore := func() {}
				executor.before = func(rank uint64) {
					if rank != test.blocked {
						return
					}
					// Every strategy publishes into its campaign's success store.
					pattern := filepath.Join(config.Artifacts, "v1", "*", "successes")
					roots, err := filepath.Glob(filepath.Dir(pattern))
					if err != nil || len(roots) != 1 {
						t.Errorf("success store parents = %v, %v", roots, err)
						return
					}
					restore = replaceWithFile(t, filepath.Join(roots[0], "successes"))
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if test.cancelled < 3 {
					executor.interrupt, executor.interruptAt = cancel, test.cancelled
					config.TerminateGrace = 10 * time.Millisecond
				}
				partial, err := exploreWith(ctx, config, configDependencies)
				restore()
				observed, _ := observeRetention(t, partial, err)
				if test.blocked < 3 {
					// The store's diagnostic names the campaign directory.
					observed.Cause = ""
				}
				if want := test.interrupted[strategy]; !reflect.DeepEqual(observed, want) {
					t.Fatalf("interrupted retention = %#v, want %#v", observed, want)
				}
				resumed, err := resumeRetentionCampaign(t, config, partial.CampaignPath, shape)
				observed, projection := observeRetention(t, resumed, err)
				if want := test.resumed[strategy]; !reflect.DeepEqual(observed, want) {
					t.Fatalf("resumed retention = %#v, want %#v", observed, want)
				}
				t.Logf("fixed-identity projection: %s", projection)
			})
		}
	}
}

// replayRecorder replays a guided corpus case as its test directs and records
// what the corpus held when the replay was requested.
type replayRecorder struct {
	t        *testing.T
	corpus   string
	result   ReplayResult
	err      error
	calls    int
	indexed  bool
	manifest record.ExecutionRecord
}

func (replayer *replayRecorder) Replay(_ context.Context, config ReplaySpec) (ReplayResult, error) {
	replayer.calls++
	opened, err := artifact.OpenArtifact(config.ArtifactPath)
	if err != nil {
		replayer.t.Errorf("guided corpus case was not published before its replay: %v", err)
		return ReplayResult{}, err
	}
	defer func(opened *artifact.Opened) {
		if err := opened.Close(); err != nil {
			replayer.t.Error(err)
		}
	}(opened)
	replayer.manifest = opened.Manifest()
	if corpusAdmissions(replayer.t, replayer.corpus) != 0 {
		replayer.indexed = true
	}
	return replayer.result, replayer.err
}

// corpusAdmissions counts the admissions committed to the corpus log at
// path, reading it without the writer lock.
func corpusAdmissions(t *testing.T, path string) uint64 {
	t.Helper()
	read, err := guide.ReadLog(path)
	if err != nil {
		t.Fatalf("read guided corpus log: %v", err)
	}
	return uint64(read.Snapshot.Generation)
}

// A guided campaign publishes a completion as a corpus case, replays it, and
// advances the corpus index only when the replay was exact. A case that is not
// replayable is not offered, and a case whose replay fails is discarded.
func TestGuidedAdmissionReplaysBeforeTheCorpusAdvances(t *testing.T) {
	type admission struct {
		Reason  string
		Replays int
		Added   uint64
		Entries uint64
		Indexed bool
		Cases   int
	}
	for _, test := range []struct {
		name     string
		complete bool
		result   ReplayResult
		err      error
		want     admission
	}{
		{name: "exact replay", complete: true, result: ReplayResult{Verified: true, Match: true}, want: admission{Replays: 1, Added: 1, Entries: 1, Indexed: true, Cases: 1}},
		{name: "diverged replay", complete: true, result: ReplayResult{Verified: true, Divergence: "stdout"}, want: admission{Reason: "guided_corpus", Replays: 1}},
		{name: "unverified replay", complete: true, result: ReplayResult{Match: true}, want: admission{Reason: "guided_corpus", Replays: 1}},
		{name: "failed replay", complete: true, err: errors.New("replay is unavailable"), want: admission{Reason: "guided_corpus", Replays: 1}},
		{name: "incomplete transcript", want: admission{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			replayer := &replayRecorder{t: t, corpus: filepath.Join(t.TempDir(), "corpus"), result: test.result, err: test.err}
			executor := &fakeExecutor{result: func(uint64) backend.ExecutionResult {
				result := processResult(0, "", "")
				result.IOTranscript = semanticTranscript(t, probeA)
				result.IOTranscript.Complete = test.complete
				return result
			}}
			config, configDependencies := testConfig(t, newFakePreparer(t), executor, "1", PolicyAll, 1)
			config.Coverage = CoverageSemantic
			config.Guide = true
			config.Corpus = replayer.corpus
			config.Replayer = replayer
			summary, err := exploreWith(context.Background(), config, configDependencies)
			observed := admission{Replays: replayer.calls, Added: summary.CorpusAdded, Entries: summary.CorpusEntries}
			var hostError *HostError
			if errors.As(err, &hostError) {
				observed.Reason = hostError.Reason
			} else if err != nil {
				t.Fatal(err)
			}
			if replayer.indexed {
				t.Fatal("the corpus index advanced before the replay")
			}
			if corpusAdmissions(t, replayer.corpus) != 0 {
				observed.Indexed = true
			}
			cases, err := os.ReadDir(filepath.Join(replayer.corpus, "cases"))
			if err != nil {
				t.Fatal(err)
			}
			observed.Cases = len(cases)
			if observed != test.want {
				t.Fatalf("admission = %#v, want %#v", observed, test.want)
			}
			if observed.Added == 1 {
				manifest := replayer.manifest
				manifest.CampaignID, manifest.CreatedAt, manifest.RecordHash, manifest.Host = "", "", "", record.Host{}
				encoded, err := canonicaljson.CanonicalJSON(manifest)
				if err != nil {
					t.Fatal(err)
				}
				t.Logf("fixed-identity projection: %s", encoded)
			}
		})
	}
}

// pairedExecutor holds every execution until a second one runs beside it, so a
// campaign that runs two executions at a time is observed at exactly that
// width. Every execution returns the same bounded payload.
type pairedExecutor struct {
	mu            sync.Mutex
	waiting       chan struct{}
	active        int
	maximumActive int
}

func (executor *pairedExecutor) Run(ctx context.Context, _ backend.ExecutionRequest) (backend.ExecutionResult, error) {
	executor.mu.Lock()
	executor.active++
	executor.maximumActive = max(executor.maximumActive, executor.active)
	partner, own := executor.waiting, chan struct{}(nil)
	if partner != nil {
		executor.waiting = nil
		close(partner)
	} else {
		own = make(chan struct{})
		executor.waiting = own
	}
	executor.mu.Unlock()
	result := processResult(0, "payload", "")
	result.IOTranscript = completeEmptyTranscript()
	if own != nil {
		select {
		case <-own:
		case <-ctx.Done():
			result.Cancelled = true
			result.Termination = backend.TerminationSignal
			result.Signal = "killed"
		}
	}
	executor.mu.Lock()
	executor.active--
	executor.mu.Unlock()
	return result, nil
}

// The number of selected seeds changes neither how many executions run at once
// nor where a fixed success bound stops the campaign: at the fourth success,
// with the fifth execution already started beside it.
func TestRunBoundsActiveExecutionsAndRetentionAtTenAndOneHundredJobs(t *testing.T) {
	type bounded struct {
		Reason        string
		Attempted     uint64
		Retained      uint64
		MaximumActive int
	}
	run := func(t *testing.T, jobs uint64, configure func(*CampaignSpec)) (CampaignResult, bounded) {
		t.Helper()
		executor := &pairedExecutor{}
		config, configDependencies := testConfig(t, newFakePreparer(t), executor, fmt.Sprintf("1-%d", jobs), PolicyAll, 2)
		config.OverallTimeout = time.Minute
		configure(&config)
		configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
		summary, err := exploreWith(context.Background(), config, configDependencies)
		observed := bounded{Attempted: summary.Attempted, Retained: summary.RetainedSuccesses, MaximumActive: executor.maximumActive}
		var hostError *HostError
		if errors.As(err, &hostError) {
			observed.Reason = hostError.Reason
		} else if err != nil {
			t.Fatal(err)
		}
		return summary, observed
	}
	measured, _ := run(t, 2, func(config *CampaignSpec) { keepSuccesses(config, KeepSuccessesAll) })
	if measured.RetainedSuccesses != 2 {
		t.Fatalf("measured summary = %#v", measured)
	}
	threeAndAHalf := measured.RetainedSuccessBytes / 2 * 7 / 2
	for _, jobs := range []uint64{10, 100} {
		for _, test := range []struct {
			name      string
			configure func(*CampaignSpec)
			want      bounded
		}{
			{name: "discard", configure: func(config *CampaignSpec) { keepSuccesses(config, KeepSuccessesNone) }, want: bounded{Attempted: jobs, MaximumActive: 2}},
			{
				name: "success count",
				configure: func(config *CampaignSpec) {
					keepSuccesses(config, KeepSuccessesAll)
					config.SuccessArtifactLimit = 3
				},
				want: bounded{Reason: "success_retention_capacity", Attempted: 5, Retained: 3, MaximumActive: 2},
			},
			{
				name: "success bytes",
				configure: func(config *CampaignSpec) {
					keepSuccesses(config, KeepSuccessesAll)
					config.SuccessArtifactLimit = 1000
					config.SuccessBytesLimit = threeAndAHalf
				},
				want: bounded{Reason: "success_retention_capacity", Attempted: 5, Retained: 3, MaximumActive: 2},
			},
		} {
			t.Run(fmt.Sprintf("%d jobs/%s", jobs, test.name), func(t *testing.T) {
				if _, observed := run(t, jobs, test.configure); observed != test.want {
					t.Fatalf("bounded campaign = %#v, want %#v", observed, test.want)
				}
			})
		}
	}
}

// The original fixture pins selections, environments, errors and plan records.
// Its flat coordinator projection normalizes the omitted seed strategy.
const legacyCampaignOptionsCharacterizationPath = "testdata/campaign_options_characterization.json"

// legacyCharacterizationChildTimeout stands in for the child deadline that the
// isolated parent derives from its own remaining time.
const legacyCharacterizationChildTimeout = 987654321 * time.Nanosecond

// legacyCharacterizationCoordinator is a coordinator command that cannot start, so
// an isolated Explore that passes every request check stops at the process
// boundary with a stable error.
const legacyCharacterizationCoordinator = "/nonexistent/gomad3-characterization-coordinator"

type legacyCampaignOptionsCase struct {
	name string
	spec CampaignSpec
	// dependencies are the private substitutions the request runs with.
	dependencies dependencies
	// frozenGuidance attaches the frozen guidance a resumed or sharded
	// campaign restores from its recorded plan.
	frozenGuidance *campaign.GuidancePlan
	// explore also runs Explore through the isolated path, whose request
	// checks precede and follow campaign validation.
	explore            bool
	coordinatorCommand []string
	// exploreLocal also runs Explore through the local path.
	exploreLocal bool
	// pinBytes keeps the complete request bytes in the table, not only their
	// digest, so a representative request of each shape stays readable.
	pinBytes bool
}

type dependencies = executionDependencies

type legacyCampaignOptionsObservation struct {
	Name           string        `json:"name"`
	Selection      string        `json:"selection"`
	SelectionCount uint64        `json:"selection_count"`
	Environment    string        `json:"environment"`
	Error          string        `json:"error"`
	IsolatedError  string        `json:"isolated_error,omitempty"`
	LocalError     string        `json:"local_error,omitempty"`
	RequestSHA256  record.SHA256 `json:"request_sha256"`
	Request        string        `json:"request,omitempty"`
	LegacySHA256   record.SHA256 `json:"legacy_request_sha256"`
	LegacyRequest  string        `json:"legacy_request,omitempty"`
	PlanSHA256     record.SHA256 `json:"plan_sha256,omitempty"`
	Plan           string        `json:"plan,omitempty"`
}

func TestCampaignOptionsLegacyCharacterization(t *testing.T) {
	cases := legacyCampaignOptionsCases(t)
	observed := make([]legacyCampaignOptionsObservation, 0, len(cases))
	for _, test := range cases {
		observed = append(observed, legacyObserveCampaignOptions(t, test))
	}
	contents, err := os.ReadFile(legacyCampaignOptionsCharacterizationPath)
	if err != nil {
		t.Fatal(err)
	}
	var pinned []legacyCampaignOptionsObservation
	if err := json.Unmarshal(contents, &pinned); err != nil {
		t.Fatal(err)
	}
	if len(pinned) != len(observed) {
		t.Fatalf("characterization table has %d cases, pinned %d", len(observed), len(pinned))
	}
	for index, got := range observed {
		want := pinned[index]
		t.Run(got.Name, func(t *testing.T) {
			if got.Name != want.Name {
				t.Fatalf("case %d = %q, pinned %q", index, got.Name, want.Name)
			}
			if got.Selection != want.Selection || got.SelectionCount != want.SelectionCount || got.Environment != want.Environment {
				t.Errorf("selection %q/%d environment %q, pinned %q/%d %q", got.Selection, got.SelectionCount, got.Environment, want.Selection, want.SelectionCount, want.Environment)
			}
			if got.Error != want.Error || got.IsolatedError != want.IsolatedError || got.LocalError != want.LocalError {
				t.Errorf("errors validate=%q isolated=%q local=%q, pinned validate=%q isolated=%q local=%q", got.Error, got.IsolatedError, got.LocalError, want.Error, want.IsolatedError, want.LocalError)
			}
			if got.RequestSHA256 == "" || strings.Contains(got.Request, `"Options"`) {
				t.Errorf("flat coordinator request %s %s", got.RequestSHA256, got.Request)
			}
			if got.PlanSHA256 != want.PlanSHA256 || got.Plan != want.Plan {
				t.Errorf("campaign plan record %s %s, pinned %s %s", got.PlanSHA256, got.Plan, want.PlanSHA256, want.Plan)
			}
			if got.LegacySHA256 != want.LegacySHA256 || got.LegacyRequest != want.LegacyRequest {
				t.Errorf("legacy coordinator request projection %s %s, pinned %s %s", got.LegacySHA256, got.LegacyRequest, want.LegacySHA256, want.LegacyRequest)
			}
		})
	}
}

func legacyObserveCampaignOptions(t *testing.T, test legacyCampaignOptionsCase) legacyCampaignOptionsObservation {
	t.Helper()
	observation := legacyCampaignOptionsObservation{Name: test.name}
	selection, environment, err := legacyCharacterizeValidation(test.spec, test.dependencies, test.frozenGuidance)
	observation.Selection, observation.SelectionCount = selection.String(), selection.Count()
	entries := make([]string, len(environment))
	for index, entry := range environment {
		entries[index] = entry.Name + "=" + entry.Value
	}
	observation.Environment = strings.Join(entries, " ")
	observation.Error = legacyCharacterizationError(t, err)
	if err == nil && test.spec.ResumeCampaign == "" {
		plan, err := legacyCharacterizePlanRecord(test.spec, test.frozenGuidance, selection, environment)
		if err != nil {
			t.Fatalf("%s: campaign plan record: %v", test.name, err)
		}
		observation.PlanSHA256 = record.HashBytes(plan)
		if test.pinBytes {
			observation.Plan = string(plan)
		}
	}
	request, legacy, err := legacyCharacterizeCoordinatorRequest(test.spec)
	if err != nil {
		t.Fatalf("%s: encode coordinator request: %v", test.name, err)
	}
	observation.RequestSHA256, observation.LegacySHA256 = record.HashBytes(request), record.HashBytes(legacy)
	if test.pinBytes {
		observation.Request, observation.LegacyRequest = string(request), string(legacy)
	}
	if test.explore {
		isolated := test.spec
		coordinator := []string{legacyCharacterizationCoordinator, "__coordinator"}
		if test.coordinatorCommand != nil {
			coordinator = test.coordinatorCommand
		}
		if isolated.Backend != nil {
			isolated.Backend = native.New(native.Config{SupervisorCommand: []string{"/bin/gomad", "__supervisor"}, CoordinatorCommand: coordinator, RunnerBuild: isolated.RunnerBuild})
		}
		_, err := exploreWith(context.Background(), isolated, test.dependencies)
		observation.IsolatedError = legacyCharacterizationError(t, err)
	}
	if test.exploreLocal {
		_, err := exploreWith(context.Background(), test.spec, test.dependencies)
		observation.LocalError = legacyCharacterizationError(t, err)
	}
	return observation
}

// legacyCharacterizationError removes the only host-dependent text an error can
// carry, the working directory a relative resume path resolves against.
func legacyCharacterizationError(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		return ""
	}
	workingDir, wdErr := os.Getwd()
	if wdErr != nil {
		t.Fatal(wdErr)
	}
	return strings.ReplaceAll(err.Error(), workingDir, "$WORKDIR")
}

func legacyCampaignOptionsCases(t *testing.T) []legacyCampaignOptionsCase {
	t.Helper()
	seed := func() CampaignSpec {
		return CampaignSpec{
			Seeds: "7-9", Parallel: 2, ExecutionTimeout: 3 * time.Second, OverallTimeout: time.Minute, TerminateGrace: time.Second,
			OnFailure: PolicyAll, FailureBudget: 1, OutputLimit: 4096, WorldTransitionLimit: 8192,
			Artifacts: "/artifacts", Environment: []string{"ZETA=1", "ALPHA=2"},
			Target:  target.Spec{Kind: target.KindGoRun, Source: "./cmd/target", WorkingDir: "/", ToolchainRoot: "/toolchain", BuildTags: []string{"tag"}},
			Backend: native.New(native.Config{SupervisorCommand: []string{"/bin/gomad", "__supervisor"}}), RunnerBuild: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			ProgressInterval: 250 * time.Millisecond,
		}
	}
	choiceSpec := func() CampaignSpec {
		spec := seed()
		spec.Strategy = StrategyChoiceExploration
		spec.Seeds = "7"
		spec.ChoiceTraceLimit = backend.MinimumChoiceTraceBytes
		spec.MaxExecutions = 8
		spec.MaxChoiceDepth = 4
		spec.ChoiceStartOrdinal = 2
		spec.MaxExplorationBytes = 1 << 20
		return spec
	}
	simulationSpec := func() CampaignSpec {
		spec := seed()
		spec.Strategy = StrategySimulationExploration
		spec.Seeds = "89"
		spec.OnFailure = PolicyBudget
		spec.FailureBudget = 4
		spec.ChoiceTraceLimit = 9 << 20
		spec.MaxExecutions = 16
		spec.MaxForcedDecisions = 3
		spec.MaxExplorationBytes = 2 << 20
		spec.MaxExplorationResultBytes = 3 << 20
		spec.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 5, Scenario: 11, Network: 13, Storage: 17, Fault: 19, Crash: 23}
		return spec
	}
	resumeSpec := func() CampaignSpec {
		return CampaignSpec{
			ResumeCampaign: "/campaigns/v1/resumed", RunnerBuild: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Target: target.Spec{ToolchainRoot: "/toolchain"}, Backend: native.New(native.Config{SupervisorCommand: []string{"/bin/gomad", "__supervisor"}}), ProgressInterval: time.Second,
		}
	}
	planIdentity := record.SHA256("sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	frozen := &campaign.GuidancePlan{Corpus: "/corpus", RequestedSelection: "7-9", RequestedCount: 3, AnsweredSeeds: []record.Uint64String{8}, AnsweredCount: 1, GuidedCount: 1}
	with := func(base func() CampaignSpec, configure func(*CampaignSpec)) CampaignSpec {
		spec := base()
		configure(&spec)
		return spec
	}
	falseOverride := false
	cases := []legacyCampaignOptionsCase{
		// Accepted requests for each strategy and campaign shape.
		{name: "seed default strategy", spec: seed(), explore: true, pinBytes: true},
		{name: "seed explicit strategy", spec: with(seed, func(spec *CampaignSpec) { spec.Strategy = StrategySeed }), explore: true},
		{name: "seed budget policy", spec: with(seed, func(spec *CampaignSpec) { spec.OnFailure, spec.FailureBudget = PolicyBudget, 3 })},
		{name: "seed first policy", spec: with(seed, func(spec *CampaignSpec) { spec.OnFailure = PolicyFirst })},
		{name: "seed observation", spec: with(seed, func(spec *CampaignSpec) {
			spec.ChoiceTraceLimit, spec.Diagnostics, spec.ClockTick = 1<<20, true, record.ClockTickForward
			spec.IOTranscriptLimit, spec.Coverage = 128<<20, CoverageSemanticChoice
		}), explore: true},
		{name: "seed strict clock tick", spec: with(seed, func(spec *CampaignSpec) { spec.ClockTick = record.ClockTickStrict })},
		{name: "seed read-only mounts", spec: with(seed, func(spec *CampaignSpec) { spec.IOROMounts = []string{"usr=/mnt/usr"} })},
		{name: "seed read-only mount limits", spec: with(seed, func(spec *CampaignSpec) {
			spec.IOROMounts = []string{"usr=/mnt/usr"}
			spec.IOROMountLimits = captured.DefaultLimits()
		})},
		{name: "seed keep all successes", spec: with(seed, func(spec *CampaignSpec) {
			spec.KeepSuccesses, spec.SuccessArtifactLimit, spec.SuccessBytesLimit = KeepSuccessesAll, 2, 1<<20
		})},
		{name: "seed keep novel successes", spec: with(seed, func(spec *CampaignSpec) {
			spec.Coverage, spec.KeepSuccesses, spec.SuccessArtifactLimit, spec.SuccessBytesLimit = CoverageSemantic, KeepSuccessesNovel, 2, 1<<20
		})},
		{name: "seed execution evidence", spec: with(seed, func(spec *CampaignSpec) {
			spec.Seeds, spec.Coverage, spec.CollectExecutionEvidence = "7", CoverageSemantic, true
		})},
		{name: "guided", spec: with(seed, func(spec *CampaignSpec) {
			spec.Guide, spec.GuideRegression, spec.Corpus, spec.Coverage = true, true, "/corpus", CoverageSemantic
			spec.GuideSnapshotSHA256 = planIdentity
		}), explore: true},
		{name: "sharded", spec: with(seed, func(spec *CampaignSpec) {
			spec.Shard, spec.PlanSHA256 = CampaignShard{Index: 1, Count: 3}, planIdentity
		}), explore: true, pinBytes: true},
		{name: "sharded frozen guidance", spec: with(seed, func(spec *CampaignSpec) {
			spec.Shard, spec.PlanSHA256 = CampaignShard{Index: 0, Count: 2}, planIdentity
			spec.Guide, spec.Corpus, spec.Coverage = true, "/corpus", CoverageSemantic
		}), frozenGuidance: frozen},
		{name: "choice exploration", spec: choiceSpec(), explore: true, pinBytes: true},
		{name: "simulation exploration", spec: simulationSpec(), explore: true, pinBytes: true},
		{name: "resume", spec: resumeSpec(), pinBytes: true},
		{name: "resume explicit regression override", spec: with(resumeSpec, func(spec *CampaignSpec) { spec.GuideRegressionOverride = &falseOverride })},
		{name: "resume missing campaign", spec: with(resumeSpec, func(spec *CampaignSpec) { spec.ResumeCampaign = filepath.Join("missing", "v1", "resumed") }), explore: true, exploreLocal: true},

		// Isolated request checks around campaign validation.
		{name: "isolated injected executor", spec: seed(), dependencies: dependencies{executor: &fakeExecutor{}}, explore: true},
		{name: "isolated injected preparer", spec: with(seed, func(spec *CampaignSpec) { spec.Preparer = targetPreparer{} }), explore: true},
		{name: "isolated empty native coordinator command", spec: seed(), coordinatorCommand: []string{""}, explore: true},
		{name: "shared validation before native coordinator command", spec: with(seed, func(spec *CampaignSpec) { spec.Parallel = 0 }), coordinatorCommand: []string{""}, explore: true},

		// Resume rejections.
		{name: "resume with preparer", spec: with(resumeSpec, func(spec *CampaignSpec) { spec.Preparer = targetPreparer{} })},
		{name: "resume without Runner build", spec: with(resumeSpec, func(spec *CampaignSpec) { spec.RunnerBuild = "" })},
		{name: "resume without provider", spec: with(resumeSpec, func(spec *CampaignSpec) { spec.Backend = nil })},
		{name: "resume ignores fresh request faults", spec: with(resumeSpec, func(spec *CampaignSpec) { spec.Seeds, spec.Parallel, spec.Strategy = "bad", -1, "bogus" })},

		// Selection and sharding rejections.
		{name: "malformed seeds", spec: with(seed, func(spec *CampaignSpec) { spec.Seeds = "x" })},
		{name: "empty seeds", spec: with(seed, func(spec *CampaignSpec) { spec.Seeds = "" })},
		{name: "shard index outside count", spec: with(seed, func(spec *CampaignSpec) {
			spec.Shard, spec.PlanSHA256 = CampaignShard{Index: 3, Count: 3}, planIdentity
		})},
		{name: "shard without plan identity", spec: with(seed, func(spec *CampaignSpec) { spec.Shard = CampaignShard{Index: 0, Count: 2} })},
		{name: "shard malformed plan identity", spec: with(seed, func(spec *CampaignSpec) {
			spec.Shard, spec.PlanSHA256 = CampaignShard{Index: 0, Count: 2}, "sha256:short"
		})},
		{name: "shard choice strategy", spec: with(choiceSpec, func(spec *CampaignSpec) {
			spec.Shard, spec.PlanSHA256 = CampaignShard{Index: 0, Count: 2}, planIdentity
		})},
		{name: "shard guidance without frozen plan", spec: with(seed, func(spec *CampaignSpec) {
			spec.Shard, spec.PlanSHA256 = CampaignShard{Index: 0, Count: 2}, planIdentity
			spec.Guide, spec.Corpus, spec.Coverage = true, "/corpus", CoverageSemantic
		})},
		{name: "shard first failure policy", spec: with(seed, func(spec *CampaignSpec) {
			spec.Shard, spec.PlanSHA256, spec.OnFailure = CampaignShard{Index: 0, Count: 2}, planIdentity, PolicyFirst
		})},

		// Strategy bound rejections.
		{name: "seed max executions", spec: with(seed, func(spec *CampaignSpec) { spec.MaxExecutions = 1 })},
		{name: "seed choice depth", spec: with(seed, func(spec *CampaignSpec) { spec.MaxChoiceDepth = 1 })},
		{name: "seed choice start ordinal", spec: with(seed, func(spec *CampaignSpec) { spec.ChoiceStartOrdinal = 1 })},
		{name: "seed forced decisions", spec: with(seed, func(spec *CampaignSpec) { spec.MaxForcedDecisions = 1 })},
		{name: "seed exploration bytes", spec: with(seed, func(spec *CampaignSpec) { spec.MaxExplorationBytes = 1 })},
		{name: "seed exploration result bytes", spec: with(seed, func(spec *CampaignSpec) { spec.MaxExplorationResultBytes = 1 })},
		{name: "seed dimension limits", spec: with(seed, func(spec *CampaignSpec) { spec.SimulationDimensionLimits.Crash = 1 })},
		{name: "choice forced decisions", spec: with(choiceSpec, func(spec *CampaignSpec) { spec.MaxForcedDecisions = 1 })},
		{name: "choice result bytes", spec: with(choiceSpec, func(spec *CampaignSpec) { spec.MaxExplorationResultBytes = 1 })},
		{name: "choice dimension limits", spec: with(choiceSpec, func(spec *CampaignSpec) { spec.SimulationDimensionLimits.Runtime = 1 })},
		{name: "choice multiple seeds", spec: with(choiceSpec, func(spec *CampaignSpec) { spec.Seeds = "7-8" })},
		{name: "choice guidance", spec: with(choiceSpec, func(spec *CampaignSpec) { spec.Guide, spec.Corpus, spec.Coverage = true, "/corpus", CoverageSemantic })},
		{name: "choice without trace", spec: with(choiceSpec, func(spec *CampaignSpec) { spec.ChoiceTraceLimit = 0 })},
		{name: "choice without max executions", spec: with(choiceSpec, func(spec *CampaignSpec) { spec.MaxExecutions = 0 })},
		{name: "choice without depth", spec: with(choiceSpec, func(spec *CampaignSpec) { spec.MaxChoiceDepth = 0 })},
		{name: "choice without exploration bytes", spec: with(choiceSpec, func(spec *CampaignSpec) { spec.MaxExplorationBytes = 0 })},
		{name: "simulation multiple seeds", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.Seeds = "89-90" })},
		{name: "simulation guidance", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.Guide, spec.Corpus, spec.Coverage = true, "/corpus", CoverageSemantic })},
		{name: "simulation without trace", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.ChoiceTraceLimit = 0 })},
		{name: "simulation without max executions", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.MaxExecutions = 0 }), explore: true},
		{name: "simulation choice depth", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.MaxChoiceDepth = 1 })},
		{name: "simulation choice start ordinal", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.ChoiceStartOrdinal = 1 })},
		{name: "simulation without forced decisions", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.MaxForcedDecisions = 0 }), explore: true},
		{name: "simulation without exploration bytes", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.MaxExplorationBytes = 0 })},
		{name: "simulation without result bytes", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.MaxExplorationResultBytes = 0 }), explore: true},
		{name: "simulation without runtime dimension", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.SimulationDimensionLimits.Runtime = 0 })},
		{name: "simulation without scenario dimension", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.SimulationDimensionLimits.Scenario = 0 })},
		{name: "simulation without network dimension", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.SimulationDimensionLimits.Network = 0 })},
		{name: "simulation without storage dimension", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.SimulationDimensionLimits.Storage = 0 })},
		{name: "simulation without fault dimension", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.SimulationDimensionLimits.Fault = 0 })},
		{name: "simulation without crash dimension", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.SimulationDimensionLimits.Crash = 0 }), explore: true},
		{name: "simulation without any dimension", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.SimulationDimensionLimits = SimulationDimensionLimits{} })},
		{name: "unknown strategy", spec: with(seed, func(spec *CampaignSpec) { spec.Strategy = "bogus" }), explore: true},

		// Execution limit rejections.
		{name: "zero parallelism", spec: with(seed, func(spec *CampaignSpec) { spec.Parallel = 0 }), explore: true},
		{name: "zero execution timeout", spec: with(seed, func(spec *CampaignSpec) { spec.ExecutionTimeout = 0 })},
		{name: "zero overall timeout", spec: with(seed, func(spec *CampaignSpec) { spec.OverallTimeout = 0 })},
		{name: "negative termination grace", spec: with(seed, func(spec *CampaignSpec) { spec.TerminateGrace = -1 })},
		{name: "termination grace beyond execution timeout", spec: with(seed, func(spec *CampaignSpec) { spec.TerminateGrace = 4 * time.Second })},
		{name: "zero output limit", spec: with(seed, func(spec *CampaignSpec) { spec.OutputLimit = 0 })},
		{name: "zero World transition limit", spec: with(seed, func(spec *CampaignSpec) { spec.WorldTransitionLimit = 0 })},
		{name: "invalid I/O transcript limit", spec: with(seed, func(spec *CampaignSpec) { spec.IOTranscriptLimit = 1 })},

		// Observation rejections.
		{name: "diagnostics without choice trace", spec: with(seed, func(spec *CampaignSpec) { spec.Diagnostics = true })},
		{name: "diagnostics with choice exploration", spec: with(choiceSpec, func(spec *CampaignSpec) { spec.Diagnostics = true })},
		{name: "choice trace below minimum", spec: with(seed, func(spec *CampaignSpec) { spec.ChoiceTraceLimit = 1 })},
		{name: "choice trace above maximum", spec: with(seed, func(spec *CampaignSpec) { spec.ChoiceTraceLimit = backend.MaximumChoiceTraceBytes + 1 })},
		{name: "probes without coverage", spec: with(seed, func(spec *CampaignSpec) { spec.RequiredSemanticProbes = []string{"probe"} })},
		{name: "unknown required probe", spec: with(seed, func(spec *CampaignSpec) {
			spec.Coverage, spec.RequiredSemanticProbes = CoverageSemantic, []string{"gomad3.characterization.unknown"}
		})},
		{name: "probes with choice coverage", spec: with(seed, func(spec *CampaignSpec) {
			spec.Coverage, spec.ChoiceTraceLimit, spec.RequiredSemanticProbes = CoverageChoice, 1<<20, []string{"probe"}
		})},
		{name: "unknown coverage", spec: with(seed, func(spec *CampaignSpec) { spec.Coverage = "bogus" })},
		{name: "choice coverage without trace", spec: with(seed, func(spec *CampaignSpec) { spec.Coverage = CoverageChoice })},
		{name: "execution evidence with multiple seeds", spec: with(seed, func(spec *CampaignSpec) { spec.Coverage, spec.CollectExecutionEvidence = CoverageSemantic, true })},
		{name: "execution evidence without semantic coverage", spec: with(seed, func(spec *CampaignSpec) { spec.Seeds, spec.CollectExecutionEvidence = "7", true })},
		{name: "invalid clock tick", spec: with(seed, func(spec *CampaignSpec) { spec.ClockTick = "sometimes" })},

		// Retention rejections.
		{name: "disabled retention with count", spec: with(seed, func(spec *CampaignSpec) { spec.SuccessArtifactLimit = 1 })},
		{name: "disabled retention with bytes", spec: with(seed, func(spec *CampaignSpec) { spec.SuccessBytesLimit = 1 })},
		{name: "novel retention without coverage", spec: with(seed, func(spec *CampaignSpec) {
			spec.KeepSuccesses, spec.SuccessArtifactLimit, spec.SuccessBytesLimit = KeepSuccessesNovel, 1, 1
		})},
		{name: "novel retention without bytes", spec: with(seed, func(spec *CampaignSpec) {
			spec.Coverage, spec.KeepSuccesses, spec.SuccessArtifactLimit = CoverageSemantic, KeepSuccessesNovel, 1
		})},
		{name: "all retention without count", spec: with(seed, func(spec *CampaignSpec) { spec.KeepSuccesses, spec.SuccessBytesLimit = KeepSuccessesAll, 1 })},
		{name: "all retention without bytes", spec: with(seed, func(spec *CampaignSpec) { spec.KeepSuccesses, spec.SuccessArtifactLimit = KeepSuccessesAll, 1 })},
		{name: "unknown retention", spec: with(seed, func(spec *CampaignSpec) { spec.KeepSuccesses = "bogus" })},
		{name: "empty artifact root", spec: with(seed, func(spec *CampaignSpec) { spec.Artifacts = "" })},
		{name: "empty Runner build", spec: with(seed, func(spec *CampaignSpec) { spec.RunnerBuild = "" })},

		// Guidance rejections.
		{name: "guidance without corpus", spec: with(seed, func(spec *CampaignSpec) { spec.Guide, spec.Coverage = true, CoverageSemantic })},
		{name: "guidance without coverage", spec: with(seed, func(spec *CampaignSpec) { spec.Guide, spec.Corpus = true, "/corpus" })},
		{name: "regression without guidance", spec: with(seed, func(spec *CampaignSpec) { spec.GuideRegression = true })},
		{name: "corpus without guidance", spec: with(seed, func(spec *CampaignSpec) { spec.Corpus = "/corpus" })},
		{name: "snapshot without guidance", spec: with(seed, func(spec *CampaignSpec) { spec.GuideSnapshotSHA256 = planIdentity })},

		// Failure policy rejections.
		{name: "all policy with budget", spec: with(seed, func(spec *CampaignSpec) { spec.FailureBudget = 2 })},
		{name: "first policy with budget", spec: with(seed, func(spec *CampaignSpec) { spec.OnFailure, spec.FailureBudget = PolicyFirst, 0 })},
		{name: "zero budget", spec: with(seed, func(spec *CampaignSpec) { spec.OnFailure, spec.FailureBudget = PolicyBudget, 0 })},
		{name: "unknown failure policy", spec: with(seed, func(spec *CampaignSpec) { spec.OnFailure = "bogus" })},

		// Process wiring and target environment rejections.
		{name: "no backend provider", spec: with(seed, func(spec *CampaignSpec) { spec.Backend = nil })},
		{name: "injected executor without backend provider", spec: with(seed, func(spec *CampaignSpec) { spec.Backend = nil }), dependencies: dependencies{executor: &fakeExecutor{}}},
		{name: "mounts without working directory", spec: with(seed, func(spec *CampaignSpec) { spec.IOROMounts, spec.Target.WorkingDir = []string{"usr=/mnt/usr"}, "" })},
		{name: "malformed mount", spec: with(seed, func(spec *CampaignSpec) { spec.IOROMounts = []string{"usr"} })},
		{name: "missing mount source", spec: with(seed, func(spec *CampaignSpec) { spec.IOROMounts = []string{"gomad3-characterization-missing=/mnt/missing"} })},
		{name: "invalid mount limits", spec: with(seed, func(spec *CampaignSpec) {
			spec.IOROMounts, spec.IOROMountLimits = []string{"usr=/mnt/usr"}, captured.Limits{Files: 1}
		})},
		{name: "malformed environment entry", spec: with(seed, func(spec *CampaignSpec) { spec.Environment = []string{"NOVALUE"} })},
		{name: "reserved environment name", spec: with(seed, func(spec *CampaignSpec) { spec.Environment = []string{"GOMAXPROCS=2"} })},
		{name: "reserved environment prefix", spec: with(seed, func(spec *CampaignSpec) { spec.Environment = []string{"LD_AUDIT=x"} })},
		{name: "duplicate environment name", spec: with(seed, func(spec *CampaignSpec) { spec.Environment = []string{"A=1", "A=2"} })},
		{name: "clock tick as target environment", spec: with(seed, func(spec *CampaignSpec) { spec.Environment = []string{record.ClockTickEnvironment + "=forward"} })},

		// Simultaneous faults pin which check wins.
		{name: "precedence seeds before strategy", spec: with(seed, func(spec *CampaignSpec) { spec.Seeds, spec.Strategy, spec.Parallel = "x", "bogus", 0 })},
		{name: "precedence shard before strategy bounds", spec: with(seed, func(spec *CampaignSpec) { spec.Shard, spec.MaxExecutions = CampaignShard{Index: 0, Count: 2}, 1 })},
		{name: "precedence strategy before limits", spec: with(seed, func(spec *CampaignSpec) { spec.Strategy, spec.Parallel, spec.OutputLimit = "bogus", 0, 0 })},
		{name: "precedence limits before identity", spec: with(seed, func(spec *CampaignSpec) { spec.Parallel, spec.Artifacts, spec.RunnerBuild = 0, "", "" })},
		{name: "precedence coverage before retention", spec: with(seed, func(spec *CampaignSpec) { spec.Coverage, spec.KeepSuccesses = "bogus", "bogus" })},
		{name: "precedence policy before provider", spec: with(seed, func(spec *CampaignSpec) { spec.OnFailure, spec.Backend = "bogus", nil })},
		{name: "precedence mounts before environment", spec: with(seed, func(spec *CampaignSpec) { spec.IOROMounts, spec.Environment = []string{"usr"}, []string{"NOVALUE"} })},
		{name: "precedence environment before clock tick", spec: with(seed, func(spec *CampaignSpec) { spec.Environment, spec.ClockTick = []string{"NOVALUE"}, "sometimes" })},
		{name: "precedence transcript before clock tick", spec: with(seed, func(spec *CampaignSpec) { spec.IOTranscriptLimit, spec.ClockTick = 1, "sometimes" })},
	}
	names := make(map[string]struct{}, len(cases))
	for _, test := range cases {
		if _, duplicate := names[test.name]; duplicate {
			t.Fatalf("duplicate characterization case %q", test.name)
		}
		names[test.name] = struct{}{}
	}
	return cases
}

// legacyCharacterizeValidation, legacyCharacterizeCoordinatorRequest and the exploreWith
// calls, which carry a case's private substitutions, are the only parts of
// this test that name the implementation under characterization.

// legacyCharacterizationPrepared is a fixed prepared target, so a campaign plan
// record depends only on the campaign request it maps.
func legacyCharacterizationPrepared() target.Prepared {
	return target.Prepared{
		Path: "/campaigns/v1/run/prepared/target", Kind: target.KindGoRun, Source: "./cmd/target",
		SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 4096, Argv: []string{"target"},
		BuildTags: []string{"tag"}, GoVersion: "go1.27.1", BuildKey: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", TargetGOOS: "linux", TargetGOARCH: "amd64",
	}
}

// legacyCharacterizationPlan clears the plan members that identify this Runner's
// implementation rather than the campaign request: they legitimately change
// with a rebuilt toolchain or the development platform.
func legacyCharacterizationPlan(plan campaign.CampaignPlan) ([]byte, error) {
	plan.IOProfile = record.IOContract{}
	plan.ChoiceExplorationImplementationSHA256 = ""
	plan.SimulationExplorationImplementationSHA256 = ""
	if plan.ChoiceProfile != nil {
		profile := *plan.ChoiceProfile
		profile.ImplementationSHA256 = ""
		plan.ChoiceProfile = &profile
	}
	return canonicaljson.CanonicalJSON(plan)
}

func legacyCharacterizationJournalPlan(spec CampaignSpec, selection SeedSelection) (campaign.ExecutionJournalPlan, error) {
	return campaign.DeriveExecutionJournalPlan(string(NormalizeStrategy(spec.Strategy)), selection.Count(), spec.MaxExecutions, uint64(spec.Parallel))
}

func legacyCharacterizationMounts(spec CampaignSpec) ([]captured.Mapping, captured.Limits, error) {
	mounts, err := captured.ParseMappings(spec.IOROMounts, spec.Target.WorkingDir)
	limits := spec.IOROMountLimits
	if limits == (captured.Limits{}) {
		limits = captured.DefaultLimits()
	}
	return mounts, limits, err
}

func legacyCharacterizeValidation(spec CampaignSpec, dependency dependencies, frozenGuidance *campaign.GuidancePlan) (SeedSelection, []record.Environment, error) {
	run := campaignRequestFromSpecWith(spec, dependency)
	run.guidancePlan = frozenGuidance
	return validateCampaignRequest(run)
}

func legacyCharacterizePlanRecord(spec CampaignSpec, frozenGuidance *campaign.GuidancePlan, selection SeedSelection, environment []record.Environment) ([]byte, error) {
	journalPlan, err := legacyCharacterizationJournalPlan(spec, selection)
	if err != nil {
		return nil, err
	}
	mounts, limits, err := legacyCharacterizationMounts(spec)
	if err != nil {
		return nil, err
	}
	run := campaignRequestFromSpec(spec)
	run.guidancePlan, run.IOROMountLimits = frozenGuidance, limits
	plan, err := campaignPlanRecord(run, journalPlan, "prepared/target", legacyCharacterizationPrepared(), environment, mounts, selection.Count())
	if err != nil {
		return nil, err
	}
	return legacyCharacterizationPlan(plan)
}

// legacyCharacterizeCoordinatorRequest returns the request bytes the isolated
// parent sends and its projection with the documented strategy default.
func legacyCharacterizeCoordinatorRequest(spec CampaignSpec) (request, legacy json.RawMessage, err error) {
	request, err = json.Marshal(campaignRequestFromSpec(spec).coordinatorConfig(legacyCharacterizationChildTimeout))
	if err != nil {
		return nil, nil, err
	}
	var members map[string]json.RawMessage
	if err := json.Unmarshal(request, &members); err != nil {
		return nil, nil, err
	}
	if _, nested := members["Options"]; nested {
		return nil, nil, errors.New("coordinator request unexpectedly nests options")
	}
	legacy, err = legacyCoordinatorProjection(members)
	return request, legacy, err
}

func legacyCoordinatorProjection(members map[string]json.RawMessage) (json.RawMessage, error) {
	if string(members["Strategy"]) == `""` {
		members["Strategy"] = json.RawMessage(fmt.Sprintf("%q", StrategySeed))
	}
	return json.Marshal(members)
}

const completionProbe = "stdlib.os.openfile"

var completionStrategies = []Strategy{StrategySeed, StrategyChoiceExploration, StrategySimulationExploration}

// completionObservation is what one campaign leaves observable after its
// executions completed: the host failure, the counters (attempted, succeeded,
// failures, watchdogs, distinct failures), every published failure artifact,
// every journaled execution and every partial that outlived the campaign.
type completionObservation struct {
	Reason    string
	Cause     string
	Counts    [5]uint64
	Artifacts []string
	Journal   []string
	Partials  []string
}

// faultExecutor completes every execution of its strategy's well-formed
// executor with one fixed fault applied to the captured result.
type faultExecutor struct {
	t     *testing.T
	base  executionRunner
	fault func(*backend.ExecutionResult)
	err   error
	mu    *sync.Mutex
	last  *backend.ExecutionResult
}

func (executor faultExecutor) Run(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	result, err := executor.base.Run(ctx, request)
	if err != nil {
		return result, err
	}
	result.IOTranscript = semanticTranscript(executor.t, completionProbe)
	if executor.fault != nil {
		executor.fault(&result)
	}
	executor.mu.Lock()
	*executor.last = result
	executor.mu.Unlock()
	return result, executor.err
}

func completionCampaign(t *testing.T, strategy Strategy, coverage CoverageMode, fault func(*backend.ExecutionResult), runErr error) (CampaignSpec, faultExecutor, executionDependencies) {
	t.Helper()
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	executor := faultExecutor{t: t, fault: fault, err: runErr, mu: &sync.Mutex{}, last: &backend.ExecutionResult{}}
	switch strategy {
	case StrategySeed:
		executor.base = &fakeExecutor{result: func(uint64) backend.ExecutionResult {
			result := processResult(0, "", "")
			result.ChoiceTrace = completeChoiceTrace(t, preparer.prepared.BuildKey, limit, []choice.Record{{
				Ordinal: 0, Kind: choice.KindRunnable, Flags: choice.FlagDecision, Alternatives: 2,
			}})
			return result
		}}
	case StrategyChoiceExploration:
		executor.base = &explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit}
	case StrategySimulationExploration:
		executor.base = &simulationExplorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit}
	}
	config, configDependencies := testConfig(t, preparer, executor, "7", PolicyAll, 1)
	config.Strategy = strategy
	config.Coverage = coverage
	config.ChoiceTraceLimit = limit
	config.WorldTransitionLimit = 1 << 20
	switch strategy {
	case StrategySeed:
	case StrategyChoiceExploration:
		config.MaxExecutions = 8
		config.MaxChoiceDepth = 4
		config.MaxExplorationBytes = 1 << 20
	case StrategySimulationExploration:
		config.MaxExecutions = 4
		config.MaxForcedDecisions = 2
		config.MaxExplorationBytes = 1 << 20
		config.MaxExplorationResultBytes = 1 << 20
		config.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 2, Scenario: 2, Network: 2, Storage: 2, Fault: 2, Crash: 2}
	}
	return config, executor, configDependencies
}

func completionWorldRecord(t *testing.T, seed uint64) []byte {
	t.Helper()
	core, err := world.New(world.Config{Seed: world.Seed(seed), Limits: world.Limits{MaxRequests: 10, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 10, MaxPayloadBytes: 1024, MaxStringBytes: 64}})
	if err != nil {
		t.Fatal(err)
	}
	initial := core.Snapshot()
	if _, err := core.Quiesce(); err != nil {
		t.Fatal(err)
	}
	recording, err := world.EncodeRecording(world.Recording{Initial: initial, Final: core.Snapshot(), Terminal: world.Terminal{Kind: world.TerminalIdle}})
	if err != nil {
		t.Fatal(err)
	}
	return recording
}

func observeCompletion(t *testing.T, summary CampaignResult, err error) completionObservation {
	t.Helper()
	observed := completionObservation{Counts: [5]uint64{summary.Attempted, summary.Succeeded, summary.Failures, summary.Watchdogs, summary.DistinctFailures}}
	if err != nil {
		var hostError *HostError
		if !errors.As(err, &hostError) {
			t.Fatalf("exploreWith() error = %#v, want a host failure", err)
		}
		observed.Reason = hostError.Reason
		observed.Cause = hostError.Err.Error()
	}
	for _, path := range summary.Artifacts {
		opened, openErr := artifact.OpenArtifact(path)
		if openErr != nil {
			t.Fatal(openErr)
		}
		manifest := opened.Manifest()
		if closeErr := opened.Close(); closeErr != nil {
			t.Fatal(closeErr)
		}
		choices := "no-choices"
		if manifest.ChoiceProfile != nil {
			choices = "choices"
		}
		observed.Artifacts = append(observed.Artifacts, manifest.ArtifactKind+" "+manifest.Outcome.Reason+" "+manifest.ReplayMode+" "+manifest.World.Terminal.Kind+" "+choices)
	}
	// Attempts keep their state only in memory; the campaign log holds the
	// lifecycle and committed outcomes.
	if committed, readErr := campaign.ReadCommittedState(summary.CampaignPath); readErr == nil {
		if !committed.Lifecycle.Published {
			observed.Partials = append(observed.Partials, "campaign "+string(committed.Lifecycle.State))
		}
		for _, run := range committed.Executions {
			observed.Journal = append(observed.Journal, run.Domain+" "+run.Reason+" "+run.Termination)
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		t.Fatal(readErr)
	}
	return observed
}

// Each row fixes one captured result and pins what every strategy makes of it.
// The exploration strategies share one expectation: they fail the whole round
// before committing it, so only the cancellation diagnostic names the strategy.
func TestCompletionFaultsKeepReasonPrecedenceAndEvidence(t *testing.T) {
	const failedCampaign = "campaign recoverable-failure"
	malformedWorld := func(result *backend.ExecutionResult) {
		result.WorldRecord = completionWorldRecord(t, 7)
		result.WorldRecord[len(result.WorldRecord)-1] ^= 1
	}
	seedMismatch := func(result *backend.ExecutionResult) { result.WorldRecord = completionWorldRecord(t, 8) }
	malformedCoverage := func(result *backend.ExecutionResult) {
		payload := []byte("not an I/O transcript")
		result.IOTranscript = backend.Transcript{Bytes: payload, SHA256: sha256.Sum256(payload), Records: 1, Complete: true}
	}
	// The digest and record count are consistent, so supervision accepts the
	// trace and only the coverage projection decodes the payload.
	malformedChoices := func(result *backend.ExecutionResult) {
		payload := []byte("not a choice trace")
		result.ChoiceTrace.Trace = choice.Trace{Version: choice.Version3, Bytes: payload, SHA256: sha256.Sum256(payload), Summary: choice.Summary{Terminal: choice.TerminalComplete}}
	}
	unterminated := func(result *backend.ExecutionResult) { result.ChoiceTrace = backend.ChoiceTrace{} }
	watchdog := func(result *backend.ExecutionResult) {
		result.Termination = backend.TerminationSignal
		result.Signal = "SIGKILL"
		result.WatchdogTimeout = true
		result.ChoiceTrace = backend.ChoiceTrace{Profile: choice.Profile, Limit: result.ChoiceTrace.Limit}
	}
	unprojectableChoices := func(result *backend.ExecutionResult) { result.ChoiceTrace.Trace.Bytes = []byte("x") }
	cancelled := func(result *backend.ExecutionResult) {
		result.Termination = backend.TerminationSignal
		result.Signal = "killed"
		result.Cancelled = true
		result.ChoiceTrace = backend.ChoiceTrace{Profile: choice.Profile, Limit: result.ChoiceTrace.Limit}
	}
	all := func(faults ...func(*backend.ExecutionResult)) func(*backend.ExecutionResult) {
		return func(result *backend.ExecutionResult) {
			for _, fault := range faults {
				fault(result)
			}
		}
	}
	for _, test := range []struct {
		name            string
		coverage        CoverageMode
		fault           func(*backend.ExecutionResult)
		err             error
		seed            completionObservation
		exploration     completionObservation
		simulationCause string
		statistics      *campaign.CampaignStatistics
	}{
		{
			name: "malformed World", fault: malformedWorld,
			seed:        completionObservation{Reason: "world_record", Cause: "decode World terminal: invalid character '|' after object key:value pair", Counts: [5]uint64{1, 0, 0, 0, 1}, Artifacts: []string{"gomad3.runner-failure/v1 world_record none none choices"}, Journal: []string{"runner world_record none"}, Partials: []string{failedCampaign}},
			exploration: completionObservation{Reason: "world_record", Cause: "decode World terminal: invalid character '|' after object key:value pair", Partials: []string{failedCampaign}},
		},
		{
			name: "World seed mismatch", fault: seedMismatch,
			statistics:  &campaign.CampaignStatistics{Attempted: 1, DistinctFailures: 1},
			seed:        completionObservation{Reason: "world_record", Cause: "World record seed or schema does not match seed 7", Counts: [5]uint64{1, 0, 0, 0, 1}, Artifacts: []string{"gomad3.runner-failure/v1 world_record none none choices"}, Journal: []string{"runner world_record none"}, Partials: []string{failedCampaign}},
			exploration: completionObservation{Reason: "world_record", Cause: "World record seed or schema does not match seed 7", Partials: []string{failedCampaign}},
		},
		{
			name: "malformed semantic coverage", coverage: CoverageSemantic, fault: malformedCoverage,
			statistics:  &campaign.CampaignStatistics{Attempted: 1},
			seed:        completionObservation{Reason: "semantic_coverage", Cause: "I/O transcript has invalid length 21", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{failedCampaign}},
			exploration: completionObservation{Reason: "semantic_coverage", Cause: "I/O transcript has invalid length 21", Partials: []string{failedCampaign}},
		},
		{
			name: "malformed choice trace", coverage: CoverageChoice, fault: malformedChoices,
			seed:        completionObservation{Reason: "choice_coverage", Cause: "project choice coverage: malformed choice trace\ninvalid choice terminal values", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{failedCampaign}},
			exploration: completionObservation{Reason: "choice_coverage", Cause: "project choice coverage: malformed choice trace\ninvalid choice terminal values", Partials: []string{failedCampaign}},
		},
		{
			name: "choice trace rejected by supervision", err: backend.ErrChoiceTraceMalformed,
			seed:        completionObservation{Reason: "choice_trace_malformed", Cause: "choice trace malformed", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{failedCampaign}},
			exploration: completionObservation{Reason: "choice_trace_malformed", Cause: "choice trace malformed", Partials: []string{failedCampaign}},
		},
		{
			name: "unterminated choice trace rejected by supervision", err: backend.ErrChoiceTraceUnterminated,
			seed:        completionObservation{Reason: "choice_trace_unterminated", Cause: "choice trace unterminated", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{failedCampaign}},
			exploration: completionObservation{Reason: "choice_trace_unterminated", Cause: "choice trace unterminated", Partials: []string{failedCampaign}},
		},
		{
			name: "missing terminal choice frame", fault: unterminated,
			seed:        completionObservation{Reason: "choice_trace_unterminated", Cause: "choice trace unterminated", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{failedCampaign}},
			exploration: completionObservation{Reason: "choice_trace_unterminated", Cause: "choice trace unterminated", Partials: []string{failedCampaign}},
		},
		{
			name: "watchdog", coverage: CoverageSemanticChoice, fault: watchdog,
			seed:        completionObservation{Counts: [5]uint64{1, 0, 1, 1, 1}, Artifacts: []string{"gomad3.watchdog-timeout/v1 watchdog_timeout diagnostic none no-choices"}, Journal: []string{"watchdog watchdog_timeout timeout"}},
			exploration: completionObservation{Reason: "choice_trace_malformed", Cause: "invalid choice decision tape\nchoice trace is not complete v3 evidence", Partials: []string{failedCampaign}},
		},
		{
			name: "cancelled execution", coverage: CoverageSemanticChoice, fault: cancelled,
			// A cancellation the campaign did not request fails the host for
			// every strategy; it is never a committed cancellation.
			seed:            completionObservation{Reason: "runner_cancelled", Cause: "seed execution was cancelled", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{failedCampaign}},
			exploration:     completionObservation{Reason: "runner_cancelled", Cause: "choice-exploration candidate was cancelled", Partials: []string{failedCampaign}},
			simulationCause: "simulation exploration candidate was cancelled",
		},
		{
			name: "malformed World, semantic coverage and choice trace", coverage: CoverageSemanticChoice, fault: all(malformedWorld, malformedCoverage, malformedChoices),
			seed:        completionObservation{Reason: "world_record", Cause: "decode World terminal: invalid character '|' after object key:value pair\npublish Runner failure artifact: validate choice trace payload: malformed choice trace\ninvalid choice terminal values", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{failedCampaign}},
			exploration: completionObservation{Reason: "world_record", Cause: "decode World terminal: invalid character '|' after object key:value pair", Partials: []string{failedCampaign}},
		},
		{
			name: "malformed semantic coverage and choice trace", coverage: CoverageSemanticChoice, fault: all(malformedCoverage, malformedChoices),
			seed:        completionObservation{Reason: "semantic_coverage", Cause: "I/O transcript has invalid length 21", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{failedCampaign}},
			exploration: completionObservation{Reason: "semantic_coverage", Cause: "I/O transcript has invalid length 21", Partials: []string{failedCampaign}},
		},
		{
			name: "watchdog and malformed World", fault: all(watchdog, malformedWorld),
			seed:        completionObservation{Reason: "world_record", Cause: "decode World terminal: invalid character '|' after object key:value pair\nconstruct Runner failure manifest: enabled choice profile did not produce the required terminal trace", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{failedCampaign}},
			exploration: completionObservation{Reason: "world_record", Cause: "decode World terminal: invalid character '|' after object key:value pair", Partials: []string{failedCampaign}},
		},
		{
			name: "watchdog and malformed semantic coverage", coverage: CoverageSemantic, fault: all(watchdog, malformedCoverage),
			seed:        completionObservation{Reason: "semantic_coverage", Cause: "I/O transcript has invalid length 21", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{failedCampaign}},
			exploration: completionObservation{Reason: "semantic_coverage", Cause: "I/O transcript has invalid length 21", Partials: []string{failedCampaign}},
		},
		{
			name: "watchdog and malformed choice trace", coverage: CoverageChoice, fault: all(watchdog, unprojectableChoices),
			seed:        completionObservation{Reason: "artifact_publication", Cause: "unexpected choice trace payload", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{failedCampaign}},
			exploration: completionObservation{Reason: "choice_trace_malformed", Cause: "invalid choice decision tape\nchoice trace is not complete v3 evidence", Partials: []string{failedCampaign}},
		},
		{
			name: "cancelled execution and malformed World", fault: all(cancelled, malformedWorld),
			seed:            completionObservation{Reason: "world_record", Cause: "decode World terminal: invalid character '|' after object key:value pair\nconstruct Runner failure manifest: enabled choice profile did not produce the required terminal trace", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{failedCampaign}},
			exploration:     completionObservation{Reason: "runner_cancelled", Cause: "choice-exploration candidate was cancelled", Partials: []string{failedCampaign}},
			simulationCause: "simulation exploration candidate was cancelled",
		},
		{
			name: "cancelled execution and malformed semantic coverage", coverage: CoverageSemantic, fault: all(cancelled, malformedCoverage),
			seed:            completionObservation{Reason: "semantic_coverage", Cause: "I/O transcript has invalid length 21", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{failedCampaign}},
			exploration:     completionObservation{Reason: "runner_cancelled", Cause: "choice-exploration candidate was cancelled", Partials: []string{failedCampaign}},
			simulationCause: "simulation exploration candidate was cancelled",
		},
		{
			name: "supervision failure and World seed mismatch", fault: seedMismatch, err: backend.ErrChoiceTraceMalformed,
			seed:        completionObservation{Reason: "choice_trace_malformed", Cause: "choice trace malformed", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{failedCampaign}},
			exploration: completionObservation{Reason: "choice_trace_malformed", Cause: "choice trace malformed", Partials: []string{failedCampaign}},
		},
	} {
		for _, strategy := range completionStrategies {
			t.Run(test.name+"/"+string(strategy), func(t *testing.T) {
				want := test.seed
				if strategy != StrategySeed {
					want = test.exploration
					if strategy == StrategySimulationExploration && test.simulationCause != "" {
						want.Cause = test.simulationCause
					}
				}
				config, _, configDependencies := completionCampaign(t, strategy, test.coverage, test.fault, test.err)
				configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
				summary, err := exploreWith(context.Background(), config, configDependencies)
				if observed := observeCompletion(t, summary, err); !reflect.DeepEqual(observed, want) {
					t.Fatalf("completion = %#v, want %#v", observed, want)
				}
				if strategy == StrategySeed && test.statistics != nil {
					if observed := observeSeedCompletion(t, summary, err).Statistics; observed != *test.statistics {
						t.Fatalf("statistics = %#v, want %#v", observed, *test.statistics)
					}
				}
			})
		}
	}
}

// A context cancelled while executions run fails the Campaign as a
// cancellation. A seed Campaign leaves a resumable plan and its partials; an
// exploration round fails ahead of the cancelled results its candidates then
// report.
func TestCancellationIsAHostFailure(t *testing.T) {
	for _, strategy := range completionStrategies {
		t.Run(string(strategy), func(t *testing.T) {
			var config CampaignSpec
			var configDependencies executionDependencies
			if strategy == StrategySeed {
				config, configDependencies = testConfig(t, newFakePreparer(t), blockingExecutor{}, "1", PolicyAll, 1)
			} else {
				config, _, configDependencies = completionCampaign(t, strategy, CoverageNone, nil, nil)
				configDependencies.executor = blockingExecutor{}
			}
			ctx := cancelOnProgress(t, &config, func(progress CampaignEvent) bool { return progress.Running == 1 })
			config.TerminateGrace = 10 * time.Millisecond
			configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
			summary, err := exploreWith(ctx, config, configDependencies)
			if strategy != StrategySeed {
				observed := observeCompletion(t, summary, err)
				// The round returns while its candidate is still exiting, so the
				// candidate's partial state is not settled yet.
				observed.Partials = nil
				if want := (completionObservation{Reason: "cancelled", Cause: "context canceled"}); !reflect.DeepEqual(observed, want) {
					t.Fatalf("completion = %#v, want %#v", observed, want)
				}
				return
			}
			var hostError *HostError
			if !errors.As(err, &hostError) || hostError.Reason != "cancelled" || !errors.Is(err, context.Canceled) {
				t.Fatalf("exploreWith() error = %#v", err)
			}
			if summary.Failures != 0 || len(summary.Artifacts) != 0 {
				t.Fatalf("cancelled summary = %#v", summary)
			}
			plan, planErr := campaign.ReadResumePlan(summary.CampaignPath)
			if planErr != nil {
				t.Fatal(planErr)
			}
			if plan.Selection != "1" || plan.RunnerBuild != config.RunnerBuild || plan.Prepared.Target.SHA256 == "" {
				t.Fatalf("resume plan = %#v", plan)
			}
			partials, readErr := os.ReadDir(filepath.Join(summary.CampaignPath, ".partial"))
			if readErr != nil {
				t.Fatal(readErr)
			}
			if len(partials) != 1 || !partials[0].IsDir() {
				t.Fatalf("cancelled partials = %v, want only the target attempt's work", partials)
			}
			committed, err := campaign.ReadCommittedState(summary.CampaignPath)
			if err != nil {
				t.Fatal(err)
			}
			if committed.Lifecycle.State != "recoverable-failure" || committed.Lifecycle.Reason != "cancelled" {
				t.Fatalf("cancelled campaign lifecycle = %#v", committed.Lifecycle)
			}
		})
	}
}

// A well-formed completion carries its World record, semantic probe and choice
// features into the execution evidence and the journal for fixed identities.
func TestCompletionProjectsWorldCoverageAndChoicesForEveryStrategy(t *testing.T) {
	type projection struct {
		Outcome  OutcomeEvidence
		World    record.World
		Coverage record.SemanticCoverage
		Features choice.FeatureProjection
		Journal  []string
		Probes   []string
		Choices  []string
	}
	for _, strategy := range completionStrategies {
		t.Run(string(strategy), func(t *testing.T) {
			recording := completionWorldRecord(t, 7)
			config, executor, configDependencies := completionCampaign(t, strategy, CoverageSemanticChoice, func(result *backend.ExecutionResult) { result.WorldRecord = recording }, nil)
			config.CollectExecutionEvidence = true
			configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
			summary, err := exploreWith(context.Background(), config, configDependencies)
			if err != nil {
				t.Fatal(err)
			}
			batch, err := campaign.OpenCampaign(summary.CampaignPath)
			if err != nil {
				t.Fatal(err)
			}
			evidence := summary.ExecutionEvidence
			if evidence == nil || evidence.Choices == nil || len(batch.Executions) == 0 {
				t.Fatalf("execution evidence = %#v, executions = %#v", evidence, batch.Executions)
			}
			last := batch.Executions[len(batch.Executions)-1]
			observed := projection{
				Outcome: evidence.Outcome, World: evidence.World, Coverage: evidence.SemanticCoverage,
				Features: choice.FeatureProjection{
					Values: evidence.Choices.Features, AdjacentPairsObserved: uint64(evidence.Choices.AdjacentPairsObserved), AdjacentPairsTruncated: evidence.Choices.AdjacentPairsTruncated,
				},
				Probes: last.SemanticProbes, Choices: last.ChoiceFeatures,
			}
			for _, run := range batch.Executions {
				observed.Journal = append(observed.Journal, run.Domain+" "+run.Reason+" "+run.Termination)
			}

			decoded, err := world.DecodeRecording(recording)
			if err != nil {
				t.Fatal(err)
			}
			bundle, err := backend.ComposeRecording(decoded, config.WorldTransitionLimit)
			if err != nil {
				t.Fatal(err)
			}
			coverage, err := transcript.SummarizeSemanticProbes([]string{completionProbe})
			if err != nil {
				t.Fatal(err)
			}
			executor.mu.Lock()
			trace := executor.last.ChoiceTrace
			executor.mu.Unlock()
			projected, err := choice.ProjectTrace(trace.Trace, trace.Limit, sha256.Sum256([]byte("fake prepared target")))
			if err != nil {
				t.Fatal(err)
			}
			exitCode := record.Uint64String(0)
			want := projection{
				Outcome: OutcomeEvidence{Domain: "success", Reason: "world_idle", Termination: "exit", ExitCode: &exitCode},
				World:   cloneWorld(bundle.Manifest), Coverage: coverage, Features: projected.Features,
				Probes: []string{completionProbe}, Choices: []string{},
			}
			for _, feature := range projected.Features.Values {
				want.Choices = append(want.Choices, feature.ID())
			}
			sort.Strings(want.Choices)
			for range batch.Executions {
				want.Journal = append(want.Journal, "success world_idle exit")
			}
			if !reflect.DeepEqual(observed, want) {
				t.Fatalf("projection = %#v, want %#v", observed, want)
			}
			for index := range batch.Executions {
				batch.Executions[index].ElapsedNanos = 0
			}
			encoded, err := canonicaljson.CanonicalJSON(struct {
				Evidence   *ExecutionEvidence
				Executions []campaign.ExecutionRecord
			}{evidence, batch.Executions})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("fixed-input canonical record inputs: %s", encoded)
		})
	}
}

type injectedCampaign struct {
	CampaignSpec
	dependencies executionDependencies
}

func injectedTestConfig(t *testing.T, preparer Preparer, executor executionRunner, seeds string, policy FailurePolicy, parallel int) injectedCampaign {
	spec, dependencies := testConfig(t, preparer, executor, seeds, policy, parallel)
	return injectedCampaign{CampaignSpec: spec, dependencies: dependencies}
}

func injectedCompletionCampaign(t *testing.T, strategy Strategy, coverage CoverageMode, fault func(*backend.ExecutionResult), runErr error) (injectedCampaign, faultExecutor) {
	spec, executor, dependencies := completionCampaign(t, strategy, coverage, fault, runErr)
	return injectedCampaign{CampaignSpec: spec, dependencies: dependencies}, executor
}

// seedCompletionObservation is what a seed campaign leaves observable once its
// completions were counted: the completion evidence and every controller
// statistic as the summary reports it.
type seedCompletionObservation struct {
	Completion completionObservation
	Statistics campaign.CampaignStatistics
}

func observeSeedCompletion(t *testing.T, summary CampaignResult, err error) seedCompletionObservation {
	t.Helper()
	return seedCompletionObservation{
		Completion: observeCompletion(t, summary, err),
		Statistics: campaign.CampaignStatistics{
			Attempted: summary.Attempted, Succeeded: summary.Succeeded, Failures: summary.Failures, Watchdogs: summary.Watchdogs,
			ReplayDivergences: summary.ReplayDivergences, Cancelled: summary.Cancelled, DistinctFailures: summary.DistinctFailures,
			StopReason: campaign.ControllerStopReason(summary.StopReason),
		},
	}
}

// scriptedSeedExecutor runs each seed's script; a seed without one succeeds.
type scriptedSeedExecutor struct {
	scripts map[uint64]func(context.Context) (backend.ExecutionResult, error)
}

func (executor scriptedSeedExecutor) Run(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	if script := executor.scripts[seedFromEnvironment(request.Env)]; script != nil {
		return script(ctx)
	}
	return processResult(0, "", ""), nil
}

func cancelledByCampaign(ctx context.Context) (backend.ExecutionResult, error) {
	<-ctx.Done()
	result := processResult(0, "", "")
	result.Cancelled = true
	result.Termination = backend.TerminationSignal
	result.Signal = "killed"
	return result, nil
}

// Completions are counted in selection order, so a campaign whose earlier seed
// fails the host still counts the attempts it drains afterwards. Each row pins
// every statistic, including those of attempts that end before classification.
func TestSeedCompletionKeepsCampaignStatistics(t *testing.T) {
	const failedCampaign = "campaign recoverable-failure"
	supervisionErr := errors.New("supervisor broke")
	failSupervision := func(context.Context) (backend.ExecutionResult, error) {
		return processResult(0, "", ""), supervisionErr
	}
	for _, test := range []struct {
		name      string
		configure func(*testing.T) (context.Context, injectedCampaign)
		want      seedCompletionObservation
	}{
		{
			name: "successes",
			configure: func(t *testing.T) (context.Context, injectedCampaign) {
				return context.Background(), injectedTestConfig(t, newFakePreparer(t), &fakeExecutor{}, "1-3", PolicyAll, 2)
			},
			want: seedCompletionObservation{
				Completion: completionObservation{Counts: [5]uint64{3, 3, 0, 0, 0}, Journal: []string{"success success exit", "success success exit", "success success exit"}},
				Statistics: campaign.CampaignStatistics{Attempted: 3, Succeeded: 3, StopReason: campaign.StopSeedsExhausted},
			},
		},
		{
			name: "distinct and duplicate failures",
			configure: func(t *testing.T) (context.Context, injectedCampaign) {
				executor := &fakeExecutor{result: func(seed uint64) backend.ExecutionResult {
					switch seed {
					case 1:
						return processResult(0, "", "")
					case 4:
						return processResult(1, "different", "")
					}
					return processResult(1, "same", "")
				}}
				return context.Background(), injectedTestConfig(t, newFakePreparer(t), executor, "1-4", PolicyAll, 1)
			},
			want: seedCompletionObservation{
				Completion: completionObservation{Counts: [5]uint64{4, 1, 3, 0, 2}, Artifacts: []string{"gomad3.target-failure/v1 nonzero_exit exact none no-choices", "gomad3.target-failure/v1 nonzero_exit exact none no-choices"}, Journal: []string{"success success exit", "target nonzero_exit exit", "target nonzero_exit exit", "target nonzero_exit exit"}},
				Statistics: campaign.CampaignStatistics{Attempted: 4, Succeeded: 1, Failures: 3, DistinctFailures: 2, StopReason: campaign.StopSeedsExhausted},
			},
		},
		{
			name: "first failure cancels active",
			configure: func(t *testing.T) (context.Context, injectedCampaign) {
				return context.Background(), injectedTestConfig(t, newFakePreparer(t), newFirstFailureExecutor(3), "1-10", PolicyFirst, 3)
			},
			want: seedCompletionObservation{
				Completion: completionObservation{Counts: [5]uint64{3, 0, 1, 0, 1}, Artifacts: []string{"gomad3.target-failure/v1 nonzero_exit exact none no-choices"}, Journal: []string{"target nonzero_exit exit", "runner runner_cancelled none", "runner runner_cancelled none"}},
				Statistics: campaign.CampaignStatistics{Attempted: 3, Failures: 1, Cancelled: 2, DistinctFailures: 1, StopReason: campaign.StopFirstFailure},
			},
		},
		{
			name: "failure budget",
			configure: func(t *testing.T) (context.Context, injectedCampaign) {
				executor := &fakeExecutor{result: func(seed uint64) backend.ExecutionResult {
					if seed == 4 {
						return processResult(1, "different", "")
					}
					return processResult(1, "same", "")
				}}
				config := injectedTestConfig(t, newFakePreparer(t), executor, "1-10", PolicyBudget, 1)
				config.FailureBudget = 2
				return context.Background(), config
			},
			want: seedCompletionObservation{
				Completion: completionObservation{Counts: [5]uint64{4, 0, 4, 0, 2}, Artifacts: []string{"gomad3.target-failure/v1 nonzero_exit exact none no-choices", "gomad3.target-failure/v1 nonzero_exit exact none no-choices"}, Journal: []string{"target nonzero_exit exit", "target nonzero_exit exit", "target nonzero_exit exit", "target nonzero_exit exit"}},
				Statistics: campaign.CampaignStatistics{Attempted: 4, Failures: 4, DistinctFailures: 2, StopReason: campaign.StopFailureBudget},
			},
		},
		{
			name: "supervision failure",
			configure: func(t *testing.T) (context.Context, injectedCampaign) {
				return context.Background(), injectedTestConfig(t, newFakePreparer(t), scriptedSeedExecutor{scripts: map[uint64]func(context.Context) (backend.ExecutionResult, error){1: failSupervision}}, "1", PolicyAll, 1)
			},
			want: seedCompletionObservation{
				Completion: completionObservation{Reason: "target_supervision", Cause: "supervisor broke", Counts: [5]uint64{1, 0, 0, 0, 1}, Artifacts: []string{"gomad3.runner-failure/v1 target_supervision none none no-choices"}, Journal: []string{"runner target_supervision none"}, Partials: []string{failedCampaign}},
				Statistics: campaign.CampaignStatistics{Attempted: 1, DistinctFailures: 1},
			},
		},
		{
			name: "supervision failure drains a cancelled attempt",
			configure: func(t *testing.T) (context.Context, injectedCampaign) {
				executor := scriptedSeedExecutor{scripts: map[uint64]func(context.Context) (backend.ExecutionResult, error){1: failSupervision, 2: cancelledByCampaign}}
				return context.Background(), injectedTestConfig(t, newFakePreparer(t), executor, "1-2", PolicyAll, 2)
			},
			want: seedCompletionObservation{
				Completion: completionObservation{Reason: "target_supervision", Cause: "supervisor broke", Counts: [5]uint64{2, 0, 0, 0, 1}, Artifacts: []string{"gomad3.runner-failure/v1 target_supervision none none no-choices"}, Journal: []string{"runner target_supervision none", "runner target_supervision none"}, Partials: []string{failedCampaign}},
				Statistics: campaign.CampaignStatistics{Attempted: 2, Cancelled: 1, DistinctFailures: 1},
			},
		},
		{
			name: "supervision failure drains a success",
			configure: func(t *testing.T) (context.Context, injectedCampaign) {
				executor := scriptedSeedExecutor{scripts: map[uint64]func(context.Context) (backend.ExecutionResult, error){1: failSupervision}}
				return context.Background(), injectedTestConfig(t, newFakePreparer(t), executor, "1-2", PolicyAll, 2)
			},
			want: seedCompletionObservation{
				Completion: completionObservation{Reason: "target_supervision", Cause: "supervisor broke", Counts: [5]uint64{2, 0, 0, 0, 0}, Artifacts: []string{"gomad3.runner-failure/v1 target_supervision none none no-choices"}, Journal: []string{"runner target_supervision none", "success success exit"}, Partials: []string{failedCampaign}},
				Statistics: campaign.CampaignStatistics{Attempted: 2},
			},
		},
		{
			name: "supervision failure drains a failure",
			configure: func(t *testing.T) (context.Context, injectedCampaign) {
				executor := scriptedSeedExecutor{scripts: map[uint64]func(context.Context) (backend.ExecutionResult, error){1: failSupervision, 2: func(context.Context) (backend.ExecutionResult, error) {
					return processResult(1, "failed", ""), nil
				}}}
				return context.Background(), injectedTestConfig(t, newFakePreparer(t), executor, "1-2", PolicyFirst, 2)
			},
			want: seedCompletionObservation{
				Completion: completionObservation{Reason: "target_supervision", Cause: "supervisor broke", Counts: [5]uint64{2, 0, 1, 0, 2}, Artifacts: []string{"gomad3.runner-failure/v1 target_supervision none none no-choices", "gomad3.target-failure/v1 nonzero_exit exact none no-choices"}, Journal: []string{"runner target_supervision none", "target nonzero_exit exit"}, Partials: []string{failedCampaign}},
				Statistics: campaign.CampaignStatistics{Attempted: 2, Failures: 1, DistinctFailures: 2},
			},
		},
		{
			name: "prepared target integrity",
			configure: func(t *testing.T) (context.Context, injectedCampaign) {
				return context.Background(), injectedTestConfig(t, newFakePreparer(t), mutatingExecutor{}, "1", PolicyAll, 1)
			},
			want: seedCompletionObservation{
				Completion: completionObservation{Reason: "prepared_target_integrity", Cause: "prepared target changed after preparation", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{failedCampaign}},
				Statistics: campaign.CampaignStatistics{Attempted: 1},
			},
		},
		{
			name: "campaign cancelled while running",
			configure: func(t *testing.T) (context.Context, injectedCampaign) {
				config := injectedTestConfig(t, newFakePreparer(t), blockingExecutor{}, "1", PolicyAll, 1)
				config.TerminateGrace = 10 * time.Millisecond
				ctx := cancelOnProgress(t, &config.CampaignSpec, func(progress CampaignEvent) bool { return progress.Running == 1 })
				return ctx, config
			},
			want: seedCompletionObservation{
				Completion: completionObservation{Reason: "cancelled", Cause: "context canceled", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{failedCampaign}},
				Statistics: campaign.CampaignStatistics{Attempted: 1},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, config := test.configure(t)
			config.dependencies = scriptedPreparationDependencies(t, config.Preparer, config.dependencies.executor)
			summary, err := exploreWith(ctx, config.CampaignSpec, config.dependencies)
			observed := observeSeedCompletion(t, summary, err)
			if !reflect.DeepEqual(observed, test.want) {
				t.Fatalf("seed completion = %#v, want %#v", observed, test.want)
			}
		})
	}
}

// The completion faults that end a seed attempt before or during
// classification keep every statistic, not only the counters the shared
// completion table pins.
func TestSeedCompletionFaultsKeepCampaignStatistics(t *testing.T) {
	malformedWorld := func(result *backend.ExecutionResult) {
		result.WorldRecord = completionWorldRecord(t, 7)
		result.WorldRecord[len(result.WorldRecord)-1] ^= 1
	}
	watchdog := func(result *backend.ExecutionResult) {
		result.Termination = backend.TerminationSignal
		result.Signal = "SIGKILL"
		result.WatchdogTimeout = true
		result.ChoiceTrace = backend.ChoiceTrace{Profile: choice.Profile, Limit: result.ChoiceTrace.Limit}
	}
	cancelled := func(result *backend.ExecutionResult) {
		result.Termination = backend.TerminationSignal
		result.Signal = "killed"
		result.Cancelled = true
		result.ChoiceTrace = backend.ChoiceTrace{Profile: choice.Profile, Limit: result.ChoiceTrace.Limit}
	}
	for _, test := range []struct {
		name     string
		coverage CoverageMode
		fault    func(*backend.ExecutionResult)
		err      error
		want     campaign.CampaignStatistics
	}{
		{name: "malformed World", fault: malformedWorld, want: campaign.CampaignStatistics{Attempted: 1, DistinctFailures: 1}},
		{name: "supervision rejected the choice trace", err: backend.ErrChoiceTraceMalformed, want: campaign.CampaignStatistics{Attempted: 1}},
		{name: "watchdog", coverage: CoverageSemanticChoice, fault: watchdog, want: campaign.CampaignStatistics{Attempted: 1, Failures: 1, Watchdogs: 1, DistinctFailures: 1, StopReason: campaign.StopSeedsExhausted}},
		{name: "cancelled execution", coverage: CoverageSemanticChoice, fault: cancelled, want: campaign.CampaignStatistics{Attempted: 1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			config, _ := injectedCompletionCampaign(t, StrategySeed, test.coverage, test.fault, test.err)
			config.dependencies = scriptedPreparationDependencies(t, config.Preparer, config.dependencies.executor)
			summary, err := exploreWith(context.Background(), config.CampaignSpec, config.dependencies)
			if observed := observeSeedCompletion(t, summary, err).Statistics; observed != test.want {
				t.Fatalf("statistics = %#v, want %#v", observed, test.want)
			}
		})
	}
}

type campaignOptionsObservation struct {
	Name        string               `json:"name"`
	Selection   string               `json:"selection"`
	Count       uint64               `json:"count"`
	Environment []record.Environment `json:"environment"`
	Error       string               `json:"error"`
	Request     string               `json:"request"`
}

type campaignOptionsRequestBytes struct {
	Name    string `json:"name"`
	Request string `json:"request"`
}

func campaignOptionsCases() []struct {
	name   string
	config CampaignSpec
} {
	base := CampaignSpec{
		Seeds: "7", Parallel: 1, ExecutionTimeout: 30 * time.Second, OverallTimeout: time.Minute,
		TerminateGrace: time.Second, OnFailure: PolicyAll, FailureBudget: 1,
		OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20,
		Artifacts: "/tmp/gomad-campaign-options", Environment: []string{"MODE=baseline"},
		Backend:     native.New(native.Config{SupervisorCommand: []string{"supervisor"}}),
		RunnerBuild: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
	choice := base
	choice.Strategy = StrategyChoiceExploration
	choice.ChoiceTraceLimit = 1 << 20
	choice.MaxExecutions, choice.MaxChoiceDepth, choice.MaxExplorationBytes = 8, 4, 1<<20
	simulation := base
	simulation.Strategy = StrategySimulationExploration
	simulation.ChoiceTraceLimit = 1 << 20
	simulation.MaxExecutions, simulation.MaxForcedDecisions = 8, 4
	simulation.MaxExplorationBytes, simulation.MaxExplorationResultBytes = 1<<20, 2<<20
	simulation.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 1, Scenario: 2, Network: 3, Storage: 4, Fault: 5, Crash: 6}
	guided := base
	guided.Seeds = "7-10"
	guided.Guide, guided.GuideRegression = true, true
	guided.Corpus, guided.Coverage = "/tmp/gomad-corpus", CoverageSemantic
	sharded := base
	sharded.Shard = CampaignShard{Index: 1, Count: 2}
	sharded.PlanSHA256 = record.SHA256("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	resume := base
	resume.ResumeCampaign = "/tmp/gomad-resume"
	cases := []struct {
		name   string
		config CampaignSpec
	}{
		{"seed-default", base}, {"choice", choice}, {"simulation", simulation}, {"guided", guided}, {"sharded", sharded}, {"resume", resume},
	}
	add := func(name string, source CampaignSpec, change func(*CampaignSpec)) {
		change(&source)
		cases = append(cases, struct {
			name   string
			config CampaignSpec
		}{name, source})
	}
	add("resume-preparer", resume, func(c *CampaignSpec) { c.Preparer = targetPreparer{} })
	add("resume-build", resume, func(c *CampaignSpec) { c.RunnerBuild = "" })
	add("resume-provider", resume, func(c *CampaignSpec) { c.Backend = nil })
	add("seeds", base, func(c *CampaignSpec) { c.Seeds = "" })
	add("shard-index", sharded, func(c *CampaignSpec) { c.Shard.Index = 2 })
	add("shard-identity-missing", sharded, func(c *CampaignSpec) { c.PlanSHA256 = "" })
	add("shard-identity-invalid", sharded, func(c *CampaignSpec) { c.PlanSHA256 = "bad" })
	add("shard-strategy", sharded, func(c *CampaignSpec) { c.Strategy = StrategyChoiceExploration })
	add("shard-guidance", sharded, func(c *CampaignSpec) { c.Guide = true })
	add("shard-policy", sharded, func(c *CampaignSpec) { c.OnFailure = PolicyFirst })
	add("seed-bounds", base, func(c *CampaignSpec) { c.MaxExecutions = 1 })
	add("choice-simulation-bounds", choice, func(c *CampaignSpec) { c.MaxForcedDecisions = 1 })
	add("choice-seeds", choice, func(c *CampaignSpec) { c.Seeds = "7-8" })
	add("choice-guide", choice, func(c *CampaignSpec) { c.Guide = true })
	add("choice-trace", choice, func(c *CampaignSpec) { c.ChoiceTraceLimit = 0 })
	add("choice-executions", choice, func(c *CampaignSpec) { c.MaxExecutions = 0 })
	add("choice-depth", choice, func(c *CampaignSpec) { c.MaxChoiceDepth = 0 })
	add("choice-bytes", choice, func(c *CampaignSpec) { c.MaxExplorationBytes = 0 })
	add("simulation-seeds", simulation, func(c *CampaignSpec) { c.Seeds = "7-8" })
	add("simulation-guide", simulation, func(c *CampaignSpec) { c.Guide = true })
	add("simulation-trace", simulation, func(c *CampaignSpec) { c.ChoiceTraceLimit = 0 })
	add("simulation-executions", simulation, func(c *CampaignSpec) { c.MaxExecutions = 0 })
	add("simulation-choice-depth", simulation, func(c *CampaignSpec) { c.MaxChoiceDepth = 1 })
	add("simulation-start-ordinal", simulation, func(c *CampaignSpec) { c.ChoiceStartOrdinal = 1 })
	add("simulation-forced", simulation, func(c *CampaignSpec) { c.MaxForcedDecisions = 0 })
	add("simulation-bytes", simulation, func(c *CampaignSpec) { c.MaxExplorationBytes = 0 })
	add("simulation-result-bytes", simulation, func(c *CampaignSpec) { c.MaxExplorationResultBytes = 0 })
	for _, dimension := range []struct {
		name string
		zero func(*SimulationDimensionLimits)
	}{
		{"runtime", func(l *SimulationDimensionLimits) { l.Runtime = 0 }},
		{"scenario", func(l *SimulationDimensionLimits) { l.Scenario = 0 }},
		{"network", func(l *SimulationDimensionLimits) { l.Network = 0 }},
		{"storage", func(l *SimulationDimensionLimits) { l.Storage = 0 }},
		{"fault", func(l *SimulationDimensionLimits) { l.Fault = 0 }},
		{"crash", func(l *SimulationDimensionLimits) { l.Crash = 0 }},
	} {
		add("simulation-dimension-"+dimension.name, simulation, func(c *CampaignSpec) { dimension.zero(&c.SimulationDimensionLimits) })
	}
	add("strategy-unknown", base, func(c *CampaignSpec) { c.Strategy = "unknown" })
	add("parallel", base, func(c *CampaignSpec) { c.Parallel = 0 })
	add("timeout", base, func(c *CampaignSpec) { c.ExecutionTimeout = 0 })
	add("grace", base, func(c *CampaignSpec) { c.TerminateGrace = time.Minute })
	add("output", base, func(c *CampaignSpec) { c.OutputLimit = 0 })
	add("diagnostics", base, func(c *CampaignSpec) { c.Diagnostics = true })
	add("choice-capacity", base, func(c *CampaignSpec) { c.ChoiceTraceLimit = 1 })
	add("artifacts", base, func(c *CampaignSpec) { c.Artifacts = "" })
	add("coverage-probes", base, func(c *CampaignSpec) { c.RequiredSemanticProbes = []string{"probe"} })
	add("coverage-unknown-probe", base, func(c *CampaignSpec) { c.Coverage = CoverageSemantic; c.RequiredSemanticProbes = []string{"probe"} })
	add("coverage-choice-probes", base, func(c *CampaignSpec) { c.Coverage = CoverageChoice; c.RequiredSemanticProbes = []string{"probe"} })
	add("coverage-choice-trace", base, func(c *CampaignSpec) { c.Coverage = CoverageChoice })
	add("coverage-unknown", base, func(c *CampaignSpec) { c.Coverage = "unknown" })
	add("retention-disabled-limit", base, func(c *CampaignSpec) { c.SuccessArtifactLimit = 1 })
	add("retention-novel", base, func(c *CampaignSpec) { c.KeepSuccesses = KeepSuccessesNovel })
	add("retention-all", base, func(c *CampaignSpec) { c.KeepSuccesses = KeepSuccessesAll })
	add("retention-unknown", base, func(c *CampaignSpec) { c.KeepSuccesses = "unknown" })
	add("execution-evidence", base, func(c *CampaignSpec) { c.CollectExecutionEvidence = true })
	add("guide-corpus", base, func(c *CampaignSpec) { c.Guide = true })
	add("unguided-corpus", base, func(c *CampaignSpec) { c.Corpus = "/tmp/corpus" })
	add("failure-budget-first", base, func(c *CampaignSpec) { c.FailureBudget = 2 })
	add("failure-budget-zero", base, func(c *CampaignSpec) { c.OnFailure = PolicyBudget; c.FailureBudget = 0 })
	add("failure-policy-unknown", base, func(c *CampaignSpec) { c.OnFailure = "unknown" })
	add("provider", base, func(c *CampaignSpec) { c.Backend = nil })
	add("mount-working-dir", base, func(c *CampaignSpec) { c.IOROMounts = []string{"/tmp=data"} })
	add("mount-invalid", base, func(c *CampaignSpec) { c.Target.WorkingDir = "/"; c.IOROMounts = []string{"invalid"} })
	add("mount-limits", base, func(c *CampaignSpec) {
		c.Target.WorkingDir = "/"
		c.IOROMounts = []string{"/tmp=data"}
		c.IOROMountLimits = captured.Limits{PathBytes: 1}
	})
	add("environment-invalid", base, func(c *CampaignSpec) { c.Environment = []string{"bad"} })
	add("environment-reserved", base, func(c *CampaignSpec) { c.Environment = []string{"GOMADSEED=1"} })
	add("environment-duplicate", base, func(c *CampaignSpec) { c.Environment = []string{"MODE=one", "MODE=two"} })
	add("transcript", base, func(c *CampaignSpec) { c.IOTranscriptLimit = 1 })
	add("clock", base, func(c *CampaignSpec) { c.ClockTick = "unknown" })
	add("seeds-before-parallel", base, func(c *CampaignSpec) { c.Seeds = ""; c.Parallel = 0 })
	add("simulation-forced-before-result", simulation, func(c *CampaignSpec) { c.MaxForcedDecisions = 0; c.MaxExplorationResultBytes = 0 })
	add("retention-before-environment", base, func(c *CampaignSpec) { c.SuccessArtifactLimit = 1; c.Environment = []string{"bad"} })
	add("resume-build-before-provider", resume, func(c *CampaignSpec) { c.RunnerBuild = ""; c.Backend = nil })
	add("guided-frozen-selection", guided, func(c *CampaignSpec) {
		c.guidancePlan = &campaign.GuidancePlan{RequestedCount: 4, AnsweredCount: 1}
		c.Seeds = "7-9"
	})
	return cases
}

func TestCampaignOptionsCharacterization(t *testing.T) {
	var observed []campaignOptionsObservation
	cases := campaignOptionsCases()
	for _, test := range cases {
		selection, environment, err := validateConfig(test.config)
		request, marshalErr := json.Marshal(campaignRequestFromSpec(test.config).coordinatorConfig(test.config.OverallTimeout))
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		row := campaignOptionsObservation{Name: test.name, Selection: selection.String(), Count: selection.Count(), Environment: environment, Request: string(request)}
		if err != nil {
			row.Error = err.Error()
		}
		observed = append(observed, row)
	}
	const fixture = "testdata/campaign-options-before.json"
	if os.Getenv("GOMAD3_CAPTURE_OPTIONS_BASELINE") == "1" {
		data, err := json.MarshalIndent(observed, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fixture, append(data, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	var baseline []campaignOptionsObservation
	if err := json.Unmarshal(data, &baseline); err != nil {
		t.Fatal(err)
	}
	if len(observed) != len(baseline) {
		t.Fatalf("characterization rows = %d, want %d", len(observed), len(baseline))
	}
	var requestBytes []campaignOptionsRequestBytes
	for index, got := range observed {
		want := baseline[index]
		if want.Name == "mount-limits" {
			if source, err := os.Lstat("/tmp"); err == nil && source.IsDir() {
				want.Error = "read-only mount limits must be positive"
			}
		}
		gotRequest, wantRequest := got.Request, want.Request
		requestBytes = append(requestBytes, campaignOptionsRequestBytes{Name: got.Name, Request: gotRequest})
		got.Request, want.Request = "", ""
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s selection/environment/error = %#v, want %#v", want.Name, got, want)
		}
		var gotFields, wantFields map[string]json.RawMessage
		if err := json.Unmarshal([]byte(gotRequest), &gotFields); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(wantRequest), &wantFields); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(gotFields, wantFields) {
			t.Errorf("%s coordinator fields changed: got %s, want %s", want.Name, gotRequest, wantRequest)
		}
		decoder := json.NewDecoder(bytes.NewReader([]byte(gotRequest)))
		decoder.DisallowUnknownFields()
		var decoded coordinatorConfig
		if err := decoder.Decode(&decoded); err != nil {
			t.Fatal(err)
		}
		original := campaignRequestFromSpec(cases[index].config)
		transported := decoded.campaignRequest(original.Backend, original.RunnerBuild)
		if !reflect.DeepEqual(transported.campaignOptions, original.campaignOptions) || transported.Backend != original.Backend || transported.RunnerBuild != original.RunnerBuild {
			t.Errorf("%s coordinator decode changed the campaign request", want.Name)
		}
	}
	const afterFixture = "testdata/campaign-options-request-after.json"
	if os.Getenv("GOMAD3_CAPTURE_OPTIONS_AFTER") == "1" {
		data, err := json.MarshalIndent(requestBytes, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(afterFixture, append(data, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	data, err = os.ReadFile(afterFixture)
	if err != nil {
		t.Fatal(err)
	}
	var expectedRequests []campaignOptionsRequestBytes
	if err := json.Unmarshal(data, &expectedRequests); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(requestBytes, expectedRequests) {
		t.Fatal("coordinator request bytes changed")
	}
}

// These tests pin the behavior that keys on whether an operation runs its
// executions through the supervisor process or through a substituted
// executor, so moving the substitution out of the public requests cannot
// change it.

func TestInjectionCharacterizationSimulationRoleKeysOnTheProcessExecutor(t *testing.T) {
	capability, err := simulationCapabilityForJob(processExecutor{provider: scriptProvider()}, runJob{})
	if err != nil || !reflect.DeepEqual(capability, &backend.SimulationCapability{Role: backend.SimulationRoleCoordinator}) {
		t.Fatalf("process executor simulation capability = %#v, %v", capability, err)
	}
	capability, err = simulationCapabilityForJob(&fakeExecutor{}, runJob{})
	if err != nil || capability != nil {
		t.Fatalf("substituted executor simulation capability = %#v, %v", capability, err)
	}
}

func TestInjectionCharacterizationReplayRequiresSupervisorOnlyForTheProcessExecutor(t *testing.T) {
	artifactPath, observed := publishReplayArtifactForTarget(t, nil, replayArtifactTarget{})
	provider := native.New(native.Config{ToolchainRoot: toolchainRoot(t)})
	_, err := Replay(context.Background(), ReplaySpec{Backend: provider, ArtifactPath: artifactPath, ToolchainRoot: toolchainRoot(t)})
	if err == nil || err.Error() != "supervisor command is required" {
		t.Fatalf("replay without supervisor or executor error = %v", err)
	}
	if _, err := Replay(context.Background(), ReplaySpec{Backend: provider, ArtifactPath: artifactPath, VerifyOnly: true, ToolchainRoot: toolchainRoot(t)}); err != nil {
		t.Fatalf("verify-only replay without supervisor: %v", err)
	}
	executor := &fakeReplayExecutor{result: observed}
	result, err := replayWith(context.Background(), ReplaySpec{Backend: provider, ArtifactPath: artifactPath, ToolchainRoot: toolchainRoot(t)}, executionDependencies{executor: executor})
	if err != nil || !result.Match {
		t.Fatalf("substituted replay without supervisor = %#v, %v", result, err)
	}
	description, err := provider.Describe(t.Context(), backend.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	if executor.calls != 1 || executor.request.Simulation != nil || executor.request.IO == nil || !reflect.DeepEqual(executor.request.IO.Profile, description.IOProfile) {
		t.Fatalf("substituted replay request calls=%d simulation=%#v profile=%#v", executor.calls, executor.request.Simulation, executor.request.IO)
	}
}

func TestInjectionCharacterizationMinimizeRequiresSupervisorOnlyForTheProcessExecutor(t *testing.T) {
	spec, dependency := minimizationSpec(t, minimizationParent(t), t.TempDir())
	spec.Backend, dependency.executor = native.New(native.Config{ToolchainRoot: spec.ToolchainRoot}), nil
	if _, err := minimizeWith(context.Background(), spec, dependency); err == nil || err.Error() != "supervisor command is required" {
		t.Fatalf("minimize without supervisor or executor error = %v", err)
	}
	spec, dependency = minimizationSpec(t, minimizationParent(t), t.TempDir())
	spec.Backend = native.New(native.Config{ToolchainRoot: spec.ToolchainRoot})
	result, err := minimizeWith(context.Background(), spec, dependency)
	if err != nil || !result.Changed {
		t.Fatalf("substituted minimize without supervisor = %#v, %v", result, err)
	}
}

var errReplayReachedSubstitutedExecutor = errors.New("replay reached the substituted executor")

// replayForwardingExecutor evaluates minimization candidates like
// minimizationExecutor and fails every replay execution, which it recognizes
// by its replayed I/O transcript.
type replayForwardingExecutor struct {
	candidates minimizationExecutor
	replays    int
}

func (executor *replayForwardingExecutor) Run(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	if request.IO != nil && request.IO.Transcript != nil && request.IO.Transcript.Replay {
		executor.replays++
		return backend.ExecutionResult{}, errReplayReachedSubstitutedExecutor
	}
	return executor.candidates.Run(ctx, request)
}

func TestInjectionCharacterizationMinimizeDefaultReplayerUsesItsExecutor(t *testing.T) {
	spec, dependency := minimizationSpec(t, minimizationParent(t), t.TempDir())
	executor := &replayForwardingExecutor{}
	dependency.executor, spec.Replayer = executor, nil
	_, err := minimizeWith(context.Background(), spec, dependency)
	if !errors.Is(err, errReplayReachedSubstitutedExecutor) || !strings.HasPrefix(err.Error(), "replay minimization candidate: execute replay target: ") {
		t.Fatalf("minimize default replay error = %v", err)
	}
	if executor.replays != 1 || executor.candidates.calls == 0 {
		t.Fatalf("minimize executor replays=%d candidates=%d", executor.replays, executor.candidates.calls)
	}
}

func TestInjectionCharacterizationShardChecksToolchainOnlyForTheProcessExecutor(t *testing.T) {
	config, _ := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
	planPath := filepath.Join(t.TempDir(), "campaign.plan.json")
	if _, err := CreateCampaignPlan(context.Background(), CampaignPlanSpec{Campaign: config, Output: planPath}); err != nil {
		t.Fatal(err)
	}
	missingRoot := filepath.Join(t.TempDir(), "missing-toolchain")
	_, err := RunCampaignShard(context.Background(), CampaignShardSpec{
		Backend:  native.New(native.Config{ToolchainRoot: missingRoot, SupervisorCommand: []string{"unused"}}),
		PlanPath: planPath, Shard: CampaignShard{Index: 0, Count: 1}, Artifacts: t.TempDir(), RunnerBuild: config.RunnerBuild,
		ToolchainRoot: missingRoot,
	})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("process shard with a missing toolchain error = %v", err)
	}
	executor := &fakeExecutor{result: func(uint64) backend.ExecutionResult { return processResult(0, "", "") }}
	result, err := runCampaignShardWith(context.Background(), CampaignShardSpec{
		Backend:  scriptProvider(),
		PlanPath: planPath, Shard: CampaignShard{Index: 0, Count: 1}, Artifacts: t.TempDir(), RunnerBuild: config.RunnerBuild,
		ToolchainRoot: missingRoot,
	}, executionDependencies{executor: executor})
	if err != nil || result.Succeeded != 1 || len(executor.requests) != 1 {
		t.Fatalf("substituted shard with a missing toolchain = %#v requests=%d, %v", result, len(executor.requests), err)
	}
}

func TestInjectionCharacterizationResumeChecksToolchainAndSupervisorOnlyForTheProcessExecutor(t *testing.T) {
	config, dependency := testConfig(t, newFakePreparer(t), &resumeInterruptExecutor{}, "7-8", PolicyAll, 1)
	ctx := cancelOnProgress(t, &config, func(event CampaignEvent) bool { return event.Succeeded == 1 })
	partial, err := exploreWith(ctx, config, dependency)
	if err == nil {
		t.Fatal("Explore() was not interrupted")
	}
	missingRoot := filepath.Join(t.TempDir(), "missing-toolchain")
	_, err = Resume(context.Background(), ResumeSpec{Backend: native.New(native.Config{ToolchainRoot: missingRoot}), CampaignPath: partial.CampaignPath, RunnerBuild: config.RunnerBuild, ToolchainRoot: missingRoot})
	if err == nil || err.Error() != "supervisor command is required" {
		t.Fatalf("resume without supervisor or executor error = %v", err)
	}
	_, err = Resume(context.Background(), ResumeSpec{Backend: native.New(native.Config{ToolchainRoot: missingRoot, SupervisorCommand: []string{"unused"}}), CampaignPath: partial.CampaignPath, RunnerBuild: config.RunnerBuild, ToolchainRoot: missingRoot})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("process resume with a missing toolchain error = %v", err)
	}
	executor := &fakeExecutor{result: func(uint64) backend.ExecutionResult { return processResult(0, "", "") }}
	resumed, err := resumeWith(context.Background(), ResumeSpec{Backend: scriptProvider(), CampaignPath: partial.CampaignPath, RunnerBuild: config.RunnerBuild, ToolchainRoot: missingRoot}, executionDependencies{executor: executor})
	if err != nil || resumed.Attempted != 2 || resumed.Succeeded != 2 || len(executor.requests) != 1 {
		t.Fatalf("substituted resume with a missing toolchain = %#v requests=%d, %v", resumed, len(executor.requests), err)
	}
}

func TestInjectionCharacterizationIsolatedExploreRejectsEverySubstitution(t *testing.T) {
	for name, configure := range map[string]func(*CampaignSpec, *executionDependencies){
		"executor": func(_ *CampaignSpec, dependency *executionDependencies) { dependency.executor = &fakeExecutor{} },
		"preparer": func(config *CampaignSpec, _ *executionDependencies) { config.Preparer = targetPreparer{} },
		"replayer": func(config *CampaignSpec, _ *executionDependencies) { config.Replayer = artifactReplayer{} },
	} {
		t.Run(name, func(t *testing.T) {
			config, dependency := testConfig(t, nil, nil, "1", PolicyAll, 1)
			config.Backend = native.New(native.Config{CoordinatorCommand: []string{"unused", "__coordinator"}})
			configure(&config, &dependency)
			if _, err := exploreWith(context.Background(), config, dependency); err == nil || err.Error() != "isolated Runner does not accept injected preparation or execution" {
				t.Fatalf("isolated Explore with an injected %s error = %v", name, err)
			}
		})
	}
}

func TestInjectionCharacterizationIsolatedPreparationDependencies(t *testing.T) {
	for _, test := range []struct {
		name          string
		prepare       bool
		execution     bool
		invalidSeeds  bool
		missingResume bool
	}{
		{name: "prepare only", prepare: true},
		{name: "execution only", execution: true},
		{name: "combined", prepare: true, execution: true},
		{name: "injection before seed validation", prepare: true, execution: true, invalidSeeds: true},
		{name: "resume preflight before injection", prepare: true, execution: true, missingResume: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			config, dependencies := testConfig(t, nil, nil, "7", PolicyAll, 1)
			marker := filepath.Join(t.TempDir(), "coordinator-started")
			command := filepath.Join(t.TempDir(), "coordinator")
			if err := os.WriteFile(command, []byte("#!/bin/sh\n: > \"$1\"\nexit 91\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			config.Backend = native.New(native.Config{CoordinatorCommand: []string{command, marker}})
			if test.prepare {
				dependencies.prepare = func(context.Context, preparationRequest) (target.Prepared, error) {
					calls.Add(1)
					return target.Prepared{}, errors.New("isolated preparation callback invoked")
				}
			}
			if test.execution {
				dependencies.executor = preparationFixtureExecutor{run: func(context.Context, backend.ExecutionRequest) (backend.ExecutionResult, error) {
					calls.Add(1)
					return backend.ExecutionResult{}, errors.New("isolated execution callback invoked")
				}}
			}
			if test.invalidSeeds {
				config.Seeds = "invalid"
			}
			if test.missingResume {
				config.ResumeCampaign = filepath.Join(t.TempDir(), "missing-campaign")
			}
			result, err := exploreWith(context.Background(), config, dependencies)
			if test.missingResume {
				if !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("resume preflight error = %v", err)
				}
			} else if err == nil || err.Error() != "isolated Runner does not accept injected preparation or execution" {
				t.Fatalf("isolated injection error = %v", err)
			}
			if (dependencies.executor != nil) != test.execution || calls.Load() != 0 || !reflect.DeepEqual(result, CampaignResult{}) {
				t.Fatalf("isolated execution ran: executor=%v callbacks=%d result=%#v", dependencies.executor, calls.Load(), result)
			}
			if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("coordinator started: %v", err)
			}
		})
	}
}

type preparationEquivalenceSnapshot struct {
	Fresh           target.Prepared `json:"fresh"`
	Cache           target.Prepared `json:"cache"`
	FreshProjection record.Target   `json:"fresh_projection"`
	CacheProjection record.Target   `json:"cache_projection"`
}

func legacyPreparationForEquivalence(ctx context.Context, spec target.Spec, environment []string) (target.Prepared, error) {
	profile := transcript.Default()
	preparedSpec, adapters, err := profile.PrepareTargetBuildAdapters(ctx, spec)
	if err != nil {
		return target.Prepared{}, err
	}
	prepared, err := nativebuild.Prepare(ctx, preparedSpec)
	if err != nil {
		return target.Prepared{}, err
	}
	prepared.Adapters = make([]record.TargetAdapter, len(adapters))
	for index, adapter := range adapters {
		prepared.Adapters[index] = record.TargetAdapter{Module: adapter.Module, Version: adapter.Version, Sum: adapter.Sum}
	}
	if err := profile.ValidatePreparedTarget(preparedSpec, prepared, environment); err != nil {
		return target.Prepared{}, err
	}
	return prepared, nil
}

func preparationForEquivalence(ctx context.Context, spec target.Spec, environment []string) (target.Prepared, error) {
	if os.Getenv("GOMAD3_PREPARATION_USE_OWNER") == "1" {
		return preparation.Prepare(ctx, preparation.Request{Target: spec, Environment: environment})
	}
	return legacyPreparationForEquivalence(ctx, spec, environment)
}

func TestLegacyPreparationFixedInputSnapshot(t *testing.T) {
	path := os.Getenv("GOMAD3_PREPARATION_SNAPSHOT")
	if path == "" {
		t.Skip("fixed-input snapshot requested separately")
	}
	fixture := os.Getenv("GOMAD3_PREPARATION_FIXTURE")
	if fixture == "" {
		t.Fatal("fixed-input fixture is required")
	}
	result := preparationEquivalenceSnapshot{}
	for index, name := range []string{"fresh", "cache"} {
		if err := os.MkdirAll(filepath.Join(filepath.Dir(path), name), 0o700); err != nil {
			t.Fatal(err)
		}
		prepared, err := preparationForEquivalence(t.Context(), target.Spec{
			Kind: target.KindGoRun, Source: ".", WorkingDir: fixture,
			PreparationRoot: filepath.Join(filepath.Dir(path), name), ToolchainRoot: toolchainRoot(t),
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		prepared.Path = "<prepared-path>"
		if index == 0 {
			result.Fresh = prepared
			result.FreshProjection = prepared.RecordTarget()
		} else {
			result.Cache = prepared
			result.CacheProjection = prepared.RecordTarget()
		}
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyAdapterPreparationFixedInputSnapshot(t *testing.T) {
	path := os.Getenv("GOMAD3_PREPARATION_ADAPTER_SNAPSHOT")
	if path == "" {
		t.Skip("fixed-input adapter snapshot requested separately")
	}
	fixture, err := filepath.Abs(filepath.Join("..", "backends/native/transcript", "testdata", "sprig"))
	if err != nil {
		t.Fatal(err)
	}
	result := preparationEquivalenceSnapshot{}
	for index, name := range []string{"fresh", "cache"} {
		if err := os.MkdirAll(filepath.Join(filepath.Dir(path), name), 0o700); err != nil {
			t.Fatal(err)
		}
		prepared, err := preparationForEquivalence(t.Context(), target.Spec{
			Kind: target.KindGoTest, Source: ".", WorkingDir: fixture,
			PreparationRoot: filepath.Join(filepath.Dir(path), name), ToolchainRoot: toolchainRoot(t),
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		prepared.Path = "<prepared-path>"
		if index == 0 {
			result.Fresh = prepared
			result.FreshProjection = prepared.RecordTarget()
		} else {
			result.Cache = prepared
			result.CacheProjection = prepared.RecordTarget()
		}
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}
