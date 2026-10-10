package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
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
	shape func(rank uint64, result *execution.Result)
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

func (executor *retentionExecutor) rank(request execution.Spec) uint64 {
	alternative := executor.alternative(request)
	if rank, ordered := executor.order[alternative]; ordered {
		return rank
	}
	return alternative
}

// alternative is the seed's position for the seed strategy and, for the
// exploration strategies, the alternative the candidate forces last.
func (executor *retentionExecutor) alternative(request execution.Spec) uint64 {
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

func (executor *retentionExecutor) Run(ctx context.Context, request execution.Spec) (execution.Result, error) {
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
		result.Termination = execution.TerminationSignal
		result.Signal = "killed"
		return result, nil
	}
	if other, waits := executor.after[rank]; waits {
		select {
		case <-executor.completion(other):
		case <-ctx.Done():
			return execution.Result{}, ctx.Err()
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
		executor.base = &fakeExecutor{result: func(seed uint64) execution.Result {
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
func rankProbes(t *testing.T, byRank ...string) func(uint64, *execution.Result) {
	return func(rank uint64, result *execution.Result) {
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

func resumeRetentionCampaign(t *testing.T, interrupted CampaignSpec, campaignPath string, shape func(uint64, *execution.Result)) (CampaignResult, error) {
	t.Helper()
	_, executor, _ := retentionCampaign(t, interrupted.Strategy, interrupted.Parallel != 1, false)
	executor.shape = shape
	return exploreWith(context.Background(), CampaignSpec{
		ResumeCampaign: campaignPath, RunnerBuild: interrupted.RunnerBuild, SupervisorCommand: []string{"unused"},
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
	var runs []campaign.ExecutionRecord
	walkErr := filepath.WalkDir(campaignPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || filepath.Base(filepath.Dir(path)) != "executions" || filepath.Ext(path) != ".jsonl" {
			return walkErr
		}
		contents, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, line := range bytes.Split(bytes.TrimSuffix(contents, []byte{'\n'}), []byte{'\n'}) {
			if len(line) == 0 {
				continue
			}
			var run campaign.ExecutionRecord
			if decodeErr := canonicaljson.DecodeCanonicalJSON(line, &run); decodeErr != nil {
				return decodeErr
			}
			runs = append(runs, run)
		}
		return nil
	})
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	return runs
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
			// The seed strategy has the third completed beside it, still finds
			// its probe novel and journals it; the resumed second execution then
			// repeats a committed probe. The exploration strategies rerun the
			// round in order.
			name: "incomplete transcript ahead of its repeat", policy: KeepSuccessesNovel, probes: [3]string{probeA, probeB, probeB},
			parallel: true, incomplete: 1, blocked: 3, cancelled: 3,
			interrupted: seedAndExploration(
				retentionObservation{Reason: "success_artifact_publication", Cause: incomplete, Counts: [5]uint64{3, 1, 0, 0, 2}, Runs: []string{"0 success/world_idle kept novel=" + probeA, "2 success/world_idle kept novel=" + probeB}},
				retentionObservation{Reason: "success_artifact_publication", Cause: incomplete, Counts: [5]uint64{1, 1, 0, 0, 1}, Runs: []string{"0 success/world_idle kept novel=" + probeA}},
			),
			resumed: seedAndExploration(
				retentionObservation{Counts: [5]uint64{3, 3, 0, 0, 2}, Runs: []string{"0 success/world_idle kept novel=" + probeA, "2 success/world_idle kept novel=" + probeB, "1 success/world_idle"}},
				retentionObservation{Counts: [5]uint64{3, 3, 0, 0, 2}, Runs: []string{"0 success/world_idle kept novel=" + probeA, "1 success/world_idle kept novel=" + probeB, "2 success/world_idle"}},
			),
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
				executor.shape = func(rank uint64, result *execution.Result) {
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
					pattern := filepath.Join(config.Artifacts, "v1", "*", "successes")
					if strategy != StrategySeed {
						pattern = filepath.Join(config.Artifacts, "v1", "*", ".partial", string(strategy), fmt.Sprintf("%020d", rank), "successes")
					}
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
	if _, err := os.Stat(filepath.Join(replayer.corpus, "corpus.json")); !os.IsNotExist(err) {
		replayer.indexed = true
	}
	return replayer.result, replayer.err
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
			executor := &fakeExecutor{result: func(uint64) execution.Result {
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
			if _, err := os.Stat(filepath.Join(replayer.corpus, "corpus.json")); err == nil {
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

func (executor *pairedExecutor) Run(ctx context.Context, _ execution.Spec) (execution.Result, error) {
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
			result.Termination = execution.TerminationSignal
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
