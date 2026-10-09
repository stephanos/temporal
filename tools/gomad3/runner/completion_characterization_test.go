package runner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	"go.temporal.io/server/tools/gomad3/world"
)

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
	fault func(*execution.Result)
	err   error
	mu    *sync.Mutex
	last  *execution.Result
}

func (executor faultExecutor) Run(ctx context.Context, request execution.Spec) (execution.Result, error) {
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

func completionCampaign(t *testing.T, strategy Strategy, coverage CoverageMode, fault func(*execution.Result), runErr error) (CampaignSpec, faultExecutor, executionDependencies) {
	t.Helper()
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	executor := faultExecutor{t: t, fault: fault, err: runErr, mu: &sync.Mutex{}, last: &execution.Result{}}
	switch strategy {
	case StrategySeed:
		executor.base = &fakeExecutor{result: func(uint64) execution.Result {
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
	walkErr := filepath.WalkDir(summary.CampaignPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		journaled := filepath.Base(filepath.Dir(path)) == "executions" && filepath.Ext(path) == ".jsonl"
		if !journaled && entry.Name() != "partial.json" {
			return nil
		}
		contents, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if !journaled {
			var partial struct {
				State string `json:"state"`
			}
			if decodeErr := json.Unmarshal(contents, &partial); decodeErr != nil {
				return decodeErr
			}
			observed.Partials = append(observed.Partials, filepath.Base(filepath.Dir(path))+" "+partial.State)
			return nil
		}
		for _, line := range bytes.Split(bytes.TrimSuffix(contents, []byte{'\n'}), []byte{'\n'}) {
			var run campaign.ExecutionRecord
			if decodeErr := canonicaljson.DecodeCanonicalJSON(line, &run); decodeErr != nil {
				return decodeErr
			}
			observed.Journal = append(observed.Journal, run.Domain+" "+run.Reason+" "+run.Termination)
		}
		return nil
	})
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	return observed
}

// Each row fixes one captured result and pins what every strategy makes of it.
// The exploration strategies share one expectation: they fail the whole round
// before committing it, so only the cancellation diagnostic names the strategy.
func TestCompletionFaultsKeepReasonPrecedenceAndEvidence(t *testing.T) {
	const failedCampaign = "campaign recoverable-failure"
	malformedWorld := func(result *execution.Result) {
		result.WorldRecord = completionWorldRecord(t, 7)
		result.WorldRecord[len(result.WorldRecord)-1] ^= 1
	}
	seedMismatch := func(result *execution.Result) { result.WorldRecord = completionWorldRecord(t, 8) }
	malformedCoverage := func(result *execution.Result) {
		payload := []byte("not an I/O transcript")
		result.IOTranscript = deterministicio.Transcript{Bytes: payload, SHA256: sha256.Sum256(payload), Records: 1, Complete: true}
	}
	// The digest and record count are consistent, so supervision accepts the
	// trace and only the coverage projection decodes the payload.
	malformedChoices := func(result *execution.Result) {
		payload := []byte("not a choice trace")
		result.ChoiceTrace.Trace = choice.Trace{Version: choice.Version3, Bytes: payload, SHA256: sha256.Sum256(payload), Summary: choice.Summary{Terminal: choice.TerminalComplete}}
	}
	unterminated := func(result *execution.Result) { result.ChoiceTrace = execution.ChoiceTrace{} }
	watchdog := func(result *execution.Result) {
		result.Termination = execution.TerminationSignal
		result.Signal = "SIGKILL"
		result.WatchdogTimeout = true
		result.ChoiceTrace = execution.ChoiceTrace{Profile: choice.Profile, Limit: result.ChoiceTrace.Limit}
	}
	unprojectableChoices := func(result *execution.Result) { result.ChoiceTrace.Trace.Bytes = []byte("x") }
	cancelled := func(result *execution.Result) {
		result.Termination = execution.TerminationSignal
		result.Signal = "killed"
		result.Cancelled = true
		result.ChoiceTrace = execution.ChoiceTrace{Profile: choice.Profile, Limit: result.ChoiceTrace.Limit}
	}
	all := func(faults ...func(*execution.Result)) func(*execution.Result) {
		return func(result *execution.Result) {
			for _, fault := range faults {
				fault(result)
			}
		}
	}
	for _, test := range []struct {
		name            string
		coverage        CoverageMode
		fault           func(*execution.Result)
		err             error
		seed            completionObservation
		exploration     completionObservation
		simulationCause string
		statistics      *campaign.CampaignStatistics
	}{
		{
			name: "malformed World", fault: malformedWorld,
			seed:        completionObservation{Reason: "world_record", Cause: "decode World terminal: invalid character '|' after object key:value pair", Counts: [5]uint64{1, 0, 0, 0, 1}, Artifacts: []string{"gomad3.runner-failure/v1 world_record none none choices"}, Journal: []string{"runner world_record none"}, Partials: []string{"00000000000000000000-7 captured", failedCampaign}},
			exploration: completionObservation{Reason: "world_record", Cause: "decode World terminal: invalid character '|' after object key:value pair", Partials: []string{failedCampaign, "00000000000000000000 captured"}},
		},
		{
			name: "World seed mismatch", fault: seedMismatch,
			statistics:  &campaign.CampaignStatistics{Attempted: 1, DistinctFailures: 1},
			seed:        completionObservation{Reason: "world_record", Cause: "World record seed or schema does not match seed 7", Counts: [5]uint64{1, 0, 0, 0, 1}, Artifacts: []string{"gomad3.runner-failure/v1 world_record none none choices"}, Journal: []string{"runner world_record none"}, Partials: []string{"00000000000000000000-7 captured", failedCampaign}},
			exploration: completionObservation{Reason: "world_record", Cause: "World record seed or schema does not match seed 7", Partials: []string{failedCampaign, "00000000000000000000 captured"}},
		},
		{
			name: "malformed semantic coverage", coverage: CoverageSemantic, fault: malformedCoverage,
			statistics:  &campaign.CampaignStatistics{Attempted: 1},
			seed:        completionObservation{Reason: "semantic_coverage", Cause: "I/O transcript has invalid length 21", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{"00000000000000000000-7 preserve-partial", failedCampaign}},
			exploration: completionObservation{Reason: "semantic_coverage", Cause: "I/O transcript has invalid length 21", Partials: []string{failedCampaign, "00000000000000000000 captured"}},
		},
		{
			name: "malformed choice trace", coverage: CoverageChoice, fault: malformedChoices,
			seed:        completionObservation{Reason: "choice_coverage", Cause: "project choice coverage: malformed choice trace\ninvalid choice terminal values", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{"00000000000000000000-7 preserve-partial", failedCampaign}},
			exploration: completionObservation{Reason: "choice_coverage", Cause: "project choice coverage: malformed choice trace\ninvalid choice terminal values", Partials: []string{failedCampaign, "00000000000000000000 captured"}},
		},
		{
			name: "choice trace rejected by supervision", err: execution.ErrChoiceTraceMalformed,
			seed:        completionObservation{Reason: "choice_trace_malformed", Cause: "choice trace malformed", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{"00000000000000000000-7 preserve-partial", failedCampaign}},
			exploration: completionObservation{Reason: "choice_trace_malformed", Cause: "choice trace malformed", Partials: []string{failedCampaign, "00000000000000000000 exited"}},
		},
		{
			name: "unterminated choice trace rejected by supervision", err: execution.ErrChoiceTraceUnterminated,
			seed:        completionObservation{Reason: "choice_trace_unterminated", Cause: "choice trace unterminated", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{"00000000000000000000-7 preserve-partial", failedCampaign}},
			exploration: completionObservation{Reason: "choice_trace_unterminated", Cause: "choice trace unterminated", Partials: []string{failedCampaign, "00000000000000000000 exited"}},
		},
		{
			name: "missing terminal choice frame", fault: unterminated,
			seed:        completionObservation{Reason: "choice_trace_unterminated", Cause: "choice trace unterminated", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{"00000000000000000000-7 preserve-partial", failedCampaign}},
			exploration: completionObservation{Reason: "choice_trace_unterminated", Cause: "choice trace unterminated", Partials: []string{failedCampaign, "00000000000000000000 exited"}},
		},
		{
			name: "watchdog", coverage: CoverageSemanticChoice, fault: watchdog,
			seed:        completionObservation{Counts: [5]uint64{1, 0, 1, 1, 1}, Artifacts: []string{"gomad3.watchdog-timeout/v1 watchdog_timeout diagnostic none no-choices"}, Journal: []string{"watchdog watchdog_timeout timeout"}},
			exploration: completionObservation{Reason: "choice_trace_malformed", Cause: "invalid choice decision tape\nchoice trace is not complete v3 evidence", Partials: []string{failedCampaign, "00000000000000000000 captured"}},
		},
		{
			name: "cancelled execution", coverage: CoverageSemanticChoice, fault: cancelled,
			seed:            completionObservation{Counts: [5]uint64{1, 0, 1, 0, 1}, Artifacts: []string{"gomad3.runner-failure/v1 runner_cancelled none none no-choices"}, Journal: []string{"runner runner_cancelled none"}},
			exploration:     completionObservation{Reason: "runner_cancelled", Cause: "choice-exploration candidate was cancelled", Partials: []string{failedCampaign, "00000000000000000000 captured"}},
			simulationCause: "simulation exploration candidate was cancelled",
		},
		{
			name: "malformed World, semantic coverage and choice trace", coverage: CoverageSemanticChoice, fault: all(malformedWorld, malformedCoverage, malformedChoices),
			seed:        completionObservation{Reason: "world_record", Cause: "decode World terminal: invalid character '|' after object key:value pair\npublish Runner failure artifact: validate choice trace payload: malformed choice trace\ninvalid choice terminal values", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{"00000000000000000000-7 captured", failedCampaign}},
			exploration: completionObservation{Reason: "world_record", Cause: "decode World terminal: invalid character '|' after object key:value pair", Partials: []string{failedCampaign, "00000000000000000000 captured"}},
		},
		{
			name: "malformed semantic coverage and choice trace", coverage: CoverageSemanticChoice, fault: all(malformedCoverage, malformedChoices),
			seed:        completionObservation{Reason: "semantic_coverage", Cause: "I/O transcript has invalid length 21", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{"00000000000000000000-7 preserve-partial", failedCampaign}},
			exploration: completionObservation{Reason: "semantic_coverage", Cause: "I/O transcript has invalid length 21", Partials: []string{failedCampaign, "00000000000000000000 captured"}},
		},
		{
			name: "watchdog and malformed World", fault: all(watchdog, malformedWorld),
			seed:        completionObservation{Reason: "world_record", Cause: "decode World terminal: invalid character '|' after object key:value pair\nconstruct Runner failure manifest: enabled choice profile did not produce the required terminal trace", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{"00000000000000000000-7 captured", failedCampaign}},
			exploration: completionObservation{Reason: "world_record", Cause: "decode World terminal: invalid character '|' after object key:value pair", Partials: []string{failedCampaign, "00000000000000000000 captured"}},
		},
		{
			name: "watchdog and malformed semantic coverage", coverage: CoverageSemantic, fault: all(watchdog, malformedCoverage),
			seed:        completionObservation{Reason: "semantic_coverage", Cause: "I/O transcript has invalid length 21", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{"00000000000000000000-7 preserve-partial", failedCampaign}},
			exploration: completionObservation{Reason: "semantic_coverage", Cause: "I/O transcript has invalid length 21", Partials: []string{failedCampaign, "00000000000000000000 captured"}},
		},
		{
			name: "watchdog and malformed choice trace", coverage: CoverageChoice, fault: all(watchdog, unprojectableChoices),
			seed:        completionObservation{Reason: "artifact_publication", Cause: "unexpected choice trace payload", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{"00000000000000000000-7 classified", failedCampaign}},
			exploration: completionObservation{Reason: "choice_trace_malformed", Cause: "invalid choice decision tape\nchoice trace is not complete v3 evidence", Partials: []string{failedCampaign, "00000000000000000000 captured"}},
		},
		{
			name: "cancelled execution and malformed World", fault: all(cancelled, malformedWorld),
			seed:            completionObservation{Reason: "world_record", Cause: "decode World terminal: invalid character '|' after object key:value pair\nconstruct Runner failure manifest: enabled choice profile did not produce the required terminal trace", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{"00000000000000000000-7 captured", failedCampaign}},
			exploration:     completionObservation{Reason: "runner_cancelled", Cause: "choice-exploration candidate was cancelled", Partials: []string{failedCampaign, "00000000000000000000 captured"}},
			simulationCause: "simulation exploration candidate was cancelled",
		},
		{
			name: "cancelled execution and malformed semantic coverage", coverage: CoverageSemantic, fault: all(cancelled, malformedCoverage),
			seed:            completionObservation{Reason: "semantic_coverage", Cause: "I/O transcript has invalid length 21", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{"00000000000000000000-7 preserve-partial", failedCampaign}},
			exploration:     completionObservation{Reason: "runner_cancelled", Cause: "choice-exploration candidate was cancelled", Partials: []string{failedCampaign, "00000000000000000000 captured"}},
			simulationCause: "simulation exploration candidate was cancelled",
		},
		{
			name: "supervision failure and World seed mismatch", fault: seedMismatch, err: execution.ErrChoiceTraceMalformed,
			seed:        completionObservation{Reason: "choice_trace_malformed", Cause: "choice trace malformed", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{"00000000000000000000-7 preserve-partial", failedCampaign}},
			exploration: completionObservation{Reason: "choice_trace_malformed", Cause: "choice trace malformed", Partials: []string{failedCampaign, "00000000000000000000 exited"}},
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
			if len(partials) != 3 {
				t.Fatalf("cancelled partials = %v, want campaign, executions, and target", partials)
			}
			if _, err := os.Stat(filepath.Join(summary.CampaignPath, ".partial", "campaign", "partial.json")); err != nil {
				t.Fatal(err)
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
		Coverage deterministicio.SemanticCoverage
		Features choice.FeatureProjection
		Journal  []string
		Probes   []string
		Choices  []string
	}
	for _, strategy := range completionStrategies {
		t.Run(string(strategy), func(t *testing.T) {
			recording := completionWorldRecord(t, 7)
			config, executor, configDependencies := completionCampaign(t, strategy, CoverageSemanticChoice, func(result *execution.Result) { result.WorldRecord = recording }, nil)
			config.CollectExecutionEvidence = true
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
			bundle, err := execution.ComposeRecording(decoded, config.WorldTransitionLimit)
			if err != nil {
				t.Fatal(err)
			}
			coverage, err := deterministicio.SummarizeSemanticProbes([]string{completionProbe})
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
