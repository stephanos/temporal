package runner

import (
	"context"
	"crypto/sha256"
	"errors"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/artifact"
	backend "go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/backends/native"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/runner/internal/exploration"
	"go.temporal.io/server/tools/gomad3/target"
)

type candidateDivergenceExecutor struct {
	base     *explorationExecutor
	mu       sync.Mutex
	diverged bool
	change   func(*backend.ExecutionResult, *choice.Divergence) error
}

func (e *candidateDivergenceExecutor) Run(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	result, err := e.base.Run(ctx, request)
	if err != nil || request.Choice.Mode != choice.ModePrefix {
		return result, err
	}
	e.mu.Lock()
	first := !e.diverged
	e.diverged = true
	e.mu.Unlock()
	if !first {
		return result, nil
	}
	expected := request.Choice.ReplayPlan.Decisions[0]
	tape, err := choice.ProjectReplayPlan(result.ChoiceTrace.Trace, request.Choice.ReplayPlan.Identity)
	if err != nil {
		return result, err
	}
	observed := tape.Decisions[0]
	observed.AlternativeSetDigest = sha256.Sum256([]byte("changed alternatives"))
	divergence := choice.Divergence{Ordinal: 0, Reason: choice.DivergenceAlternativeSet, Expected: &expected, Observed: &observed, TapeRecords: 1}
	if e.change != nil {
		return result, e.change(&result, &divergence)
	}
	return result, &backend.ChoiceReplayDivergenceError{Divergence: divergence}
}

func divergenceCampaignConfig(t *testing.T, policy FailurePolicy, parallel int) (CampaignSpec, executionDependencies) {
	t.Helper()
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	executor := &candidateDivergenceExecutor{base: &explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit, alternatives: 4}}
	config, configDependencies := testConfig(t, preparer, executor, "7", policy, parallel)
	config.Strategy = StrategyChoiceExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 8
	config.MaxChoiceDepth = 4
	config.MaxExplorationBytes = 1 << 20
	return config, configDependencies
}

func TestRunChoiceExplorationDivergencePoliciesAndInspection(t *testing.T) {
	for _, policy := range []FailurePolicy{PolicyFirst, PolicyBudget, PolicyAll} {
		t.Run(string(policy), func(t *testing.T) {
			config, configDependencies := divergenceCampaignConfig(t, policy, 1)
			configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
			summary, err := exploreWith(t.Context(), config, configDependencies)
			if err != nil {
				t.Fatal(err)
			}
			wantRuns := uint64(4)
			wantStop := StopExplorationExhausted
			if policy == PolicyFirst {
				wantRuns = 2
				wantStop = StopFirstFailure
			}
			if summary.Attempted != wantRuns || summary.Succeeded != wantRuns-1 || summary.Failures != 1 || summary.ReplayDivergences != 1 || summary.DistinctFailures != 0 || summary.StopReason != wantStop {
				t.Fatalf("summary = %#v", summary)
			}
			inspected, err := Inspect(summary.CampaignPath, InspectOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if inspected.Campaign.ReplayDivergences != 1 || len(inspected.Campaign.Executions) != int(wantRuns) || len(inspected.Campaign.FailureArtifacts) != 0 {
				t.Fatalf("inspection = %#v", inspected.Campaign)
			}
			run := inspected.Campaign.Executions[1]
			if run.Reason != "replay_divergence" || run.Domain != "runner" || run.Divergence == nil || run.Divergence.Ordinal != 0 || run.Divergence.Reason != choice.DivergenceAlternativeSet || run.Divergence.Expected == nil || run.Divergence.Observed == nil || run.OutcomeSHA256 != "" {
				t.Fatalf("divergent inspection = %#v", run)
			}
		})
	}
}

func TestRunChoiceExplorationKeepsOtherErrorsAsHostErrors(t *testing.T) {
	typed := func(d choice.Divergence) error { return &backend.ChoiceReplayDivergenceError{Divergence: d} }
	for _, test := range []struct {
		name, reason string
		change       func(*backend.ExecutionResult, *choice.Divergence) error
	}{
		{"normal error", "target_supervision", func(_ *backend.ExecutionResult, _ *choice.Divergence) error { return errors.New("executor failed") }},
		{"cancelled", "runner_cancelled", func(r *backend.ExecutionResult, _ *choice.Divergence) error { r.Cancelled = true; return nil }},
		{"typed cancelled", "target_supervision", func(r *backend.ExecutionResult, d *choice.Divergence) error { r.Cancelled = true; return typed(*d) }},
		{"overflow", "choice_trace_overflow", func(_ *backend.ExecutionResult, _ *choice.Divergence) error { return backend.ErrChoiceTraceOverflow }},
		{"malformed trace", "choice_trace_malformed", func(_ *backend.ExecutionResult, _ *choice.Divergence) error { return backend.ErrChoiceTraceMalformed }},
		{"unterminated trace", "choice_trace_unterminated", func(_ *backend.ExecutionResult, _ *choice.Divergence) error {
			return backend.ErrChoiceTraceUnterminated
		}},
		{"missing identity", "target_supervision", func(_ *backend.ExecutionResult, d *choice.Divergence) error {
			d.Reason = choice.DivergenceIdentityMissing
			d.Observed = nil
			return typed(*d)
		}},
		{"duplicate identity", "target_supervision", func(_ *backend.ExecutionResult, d *choice.Divergence) error {
			d.Reason = choice.DivergenceIdentityDuplicate
			d.Observed = nil
			return typed(*d)
		}},
		{"alternative capacity", "target_supervision", func(_ *backend.ExecutionResult, d *choice.Divergence) error {
			d.Reason = choice.DivergenceAlternativeCapacity
			d.Observed = nil
			return typed(*d)
		}},
		{"tape exhaustion", "target_supervision", func(_ *backend.ExecutionResult, d *choice.Divergence) error {
			d.Reason = choice.DivergenceTapeExhausted
			d.Ordinal = 1
			d.Expected = nil
			return typed(*d)
		}},
		{"typed watchdog", "target_supervision", func(r *backend.ExecutionResult, d *choice.Divergence) error {
			r.WatchdogTimeout = true
			return typed(*d)
		}},
		{"wrong prefix", "target_supervision", func(_ *backend.ExecutionResult, d *choice.Divergence) error {
			d.Expected.SiteOffset++
			return typed(*d)
		}},
		{"malformed observed", "target_supervision", func(_ *backend.ExecutionResult, d *choice.Divergence) error {
			d.Observed.SelectedIdentity = [32]byte{}
			return typed(*d)
		}},
		{"joined failure", "target_supervision", func(_ *backend.ExecutionResult, d *choice.Divergence) error {
			return errors.Join(typed(*d), errors.New("output close failed"))
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			config, configDependencies := divergenceCampaignConfig(t, PolicyAll, 1)
			configDependencies.executor.(*candidateDivergenceExecutor).change = test.change
			configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
			summary, err := exploreWith(t.Context(), config, configDependencies)
			var host *HostError
			if !errors.As(err, &host) || host.Reason != test.reason {
				t.Fatalf("error = %v, want %s", err, test.reason)
			}
			if summary.Attempted != 1 || summary.ReplayDivergences != 0 || summary.ChoiceExploration.CommittedRounds != 1 {
				t.Fatalf("host failure committed candidate: %#v", summary)
			}
		})
	}
}

func TestProcessExplorationCompletionKeepsRunnerDomainFallback(t *testing.T) {
	config, _ := divergenceCampaignConfig(t, PolicyAll, 1)
	prepared := config.Preparer.(*fakePreparer).prepared
	implementation, err := choice.ImplementationIdentity(prepared.BuildKey)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := choiceExecutionIdentity(prepared, implementation)
	if err != nil {
		t.Fatal(err)
	}
	state, err := exploration.NewChoice(exploration.ChoiceConfig{Execution: identity, ControllerSHA256: exploration.ChoiceImplementationSHA256(), BaseSeed: 7, Parallel: 1, MaxExecutions: 8, MaxChoiceDepth: 4, MaxExplorationBytes: 1 << 20, FailurePolicy: exploration.PolicyAll, FailureBudget: 1})
	if err != nil {
		t.Fatal(err)
	}
	round, _ := state.NextRound()
	result := processResult(0, "", "")
	result.Cancelled = true
	explored := &explorationRound[exploration.ChoiceState, exploration.ChoiceRound]{
		explorationCampaign: &explorationCampaign{ctx: t.Context(), config: campaignRequestFromSpec(config), prepared: prepared, summary: &CampaignResult{}},
		state:               state, round: round, tally: &explorationTally{},
	}
	_, _, err = processExplorationCompletion(explored, 0, runCompletion{job: runJob{seed: 7}, result: result})
	var host *HostError
	if !errors.As(err, &host) || host.Reason != "runner_cancelled" {
		t.Fatalf("runner-domain fallback = %v", err)
	}
}

func TestRunChoiceExplorationResumePreservesDivergenceIdentity(t *testing.T) {
	config, configDependencies := divergenceCampaignConfig(t, PolicyAll, 1)
	ctx, cancel := context.WithCancel(t.Context())
	config.Progress = func(progress CampaignEvent) error {
		if progress.ChoiceExploration != nil && progress.ChoiceExploration.CommittedRounds == 2 {
			cancel()
		}
		return nil
	}
	partial, err := exploreWith(ctx, config, configDependencies)
	var host *HostError
	if !errors.As(err, &host) || partial.Attempted != 2 || partial.ReplayDivergences != 1 {
		t.Fatalf("partial = %#v, err=%v", partial, err)
	}
	executor := configDependencies.executor.(*candidateDivergenceExecutor)
	resumed, err := exploreWith(t.Context(), CampaignSpec{ResumeCampaign: partial.CampaignPath, RunnerBuild: config.RunnerBuild, Backend: scriptProvider()}, executionDependencies{executor: executor.base})
	if err != nil {
		t.Fatal(err)
	}
	uninterruptedConfig, uninterruptedConfigDependencies := divergenceCampaignConfig(t, PolicyAll, 1)
	uninterrupted, err := exploreWith(t.Context(), uninterruptedConfig, uninterruptedConfigDependencies)
	if err != nil {
		t.Fatal(err)
	}
	resumedBatch, err := campaign.OpenCampaign(resumed.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	uninterruptedBatch, err := campaign.OpenCampaign(uninterrupted.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resumed.ChoiceExploration, uninterrupted.ChoiceExploration) || resumedBatch.Record.ChoiceExplorationChainSHA256 != uninterruptedBatch.Record.ChoiceExplorationChainSHA256 || resumed.ReplayDivergences != 1 || resumed.DistinctFailures != 0 {
		t.Fatalf("resumed = %#v, uninterrupted = %#v", resumed, uninterrupted)
	}
	for i, run := range resumedBatch.Executions {
		run.ElapsedNanos = record.Uint64String(0)
		other := uninterruptedBatch.Executions[i]
		other.ElapsedNanos = 0
		if !reflect.DeepEqual(run, other) {
			t.Fatalf("run %d changed across resume: %#v, %#v", i, run, other)
		}
	}
}

func TestRunChoiceExplorationRetainsOnlyPrefixMismatchReasons(t *testing.T) {
	for _, reason := range []choice.DivergenceReason{choice.DivergenceKind, choice.DivergenceSite, choice.DivergenceAlternatives, choice.DivergenceSelected, choice.DivergenceAlternativeSet, choice.DivergenceTapeUnconsumed, choice.DivergenceObservation} {
		t.Run(choice.DivergenceReasonName(reason), func(t *testing.T) {
			config, configDependencies := divergenceCampaignConfig(t, PolicyAll, 1)
			configDependencies.executor.(*candidateDivergenceExecutor).change = func(_ *backend.ExecutionResult, d *choice.Divergence) error {
				d.Reason = reason
				if reason != choice.DivergenceAlternativeSet {
					d.Observed.AlternativeSetDigest = d.Expected.AlternativeSetDigest
				}
				switch reason {
				case choice.DivergenceKind:
					d.Observed.Kind = choice.KindSelectPoll
				case choice.DivergenceSite:
					d.Observed.SiteOffset++
				case choice.DivergenceAlternatives:
					d.Observed.Alternatives++
				case choice.DivergenceSelected:
					d.Observed.Selected = (d.Expected.Selected + 1) % d.Expected.Alternatives
				case choice.DivergenceTapeUnconsumed:
					d.Observed = nil
				case choice.DivergenceAlternativeSet, choice.DivergenceTapeExhausted, choice.DivergenceIdentityMissing, choice.DivergenceIdentityDuplicate, choice.DivergenceAlternativeCapacity, choice.DivergenceObservation:
				}
				return &backend.ChoiceReplayDivergenceError{Divergence: *d}
			}
			configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
			summary, err := exploreWith(t.Context(), config, configDependencies)
			if err != nil {
				t.Fatal(err)
			}
			batch, err := campaign.OpenCampaign(summary.CampaignPath)
			if err != nil {
				t.Fatal(err)
			}
			if summary.ReplayDivergences != 1 || batch.Executions[1].Divergence.Reason != reason {
				t.Fatalf("reason %d was not retained", reason)
			}
		})
	}
}

func TestRunChoiceExplorationExecutesRootAndEveryNonSelectedRank(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	executor := &explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit}
	config, configDependencies := testConfig(t, preparer, executor, "7", PolicyAll, 2)
	config.Strategy = StrategyChoiceExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 8
	config.MaxChoiceDepth = 4
	config.MaxExplorationBytes = 1 << 20

	configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Attempted != 2 || summary.Succeeded != 2 || summary.StopReason != StopExplorationExhausted || summary.ChoiceExploration == nil || summary.ChoiceExploration.CommittedRounds != 2 || summary.ChoiceExploration.SeenPrefixes != 2 {
		t.Fatalf("exploration summary = %#v", summary)
	}
	executor.mu.Lock()
	requests := append([]backend.ExecutionRequest(nil), executor.requests...)
	executor.mu.Unlock()
	if len(requests) != 2 || requests[0].Choice == nil || requests[0].Choice.Mode != choice.ModeRecord || requests[1].Choice == nil || requests[1].Choice.Mode != choice.ModePrefix || requests[1].Choice.ReplayPlan == nil || requests[1].Choice.ReplayPlan.Decisions[0].Selected != 1 {
		t.Fatalf("exploration requests = %#v", requests)
	}
	batch, err := campaign.OpenCampaign(summary.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if batch.Record.Strategy != string(StrategyChoiceExploration) || batch.Record.ChoiceExploration == nil || len(batch.Executions) != 2 || batch.Executions[1].ParentCandidateSHA256 == "" || batch.Executions[1].ForcedDepth == nil || *batch.Executions[1].ForcedDepth != 1 {
		t.Fatalf("exploration batch = %#v", batch)
	}
	explored, ok, err := inspectedRounds(summary.CampaignPath)
	if err != nil || !ok || len(explored.Rounds) != 2 || explored.Rounds[1].FirstOrdinal != 1 || explored.Rounds[1].Executions != 1 || explored.ChainSHA256 != batch.Record.ChoiceExplorationChainSHA256 {
		t.Fatalf("committed exploration rounds = %#v, %v", explored, err)
	}
}

func TestRunChoiceExplorationResumeRerunsTheWholeIncompleteRound(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	baseExecutor := &explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit}
	config, configDependencies := testConfig(t, preparer, explorationInterruptExecutor{exploration: baseExecutor}, "7", PolicyAll, 2)
	config.Strategy = StrategyChoiceExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 8
	config.MaxChoiceDepth = 4
	config.MaxExplorationBytes = 1 << 20

	partial, err := exploreWith(context.Background(), config, configDependencies)
	var hostErr *HostError
	if !errors.As(err, &hostErr) || partial.Attempted != 1 || partial.ChoiceExploration == nil || partial.ChoiceExploration.CommittedRounds != 1 {
		t.Fatalf("partial exploration = %#v, error = %v", partial, err)
	}
	resumedExecutor := &explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit}
	resumed, err := exploreWith(context.Background(), CampaignSpec{
		ResumeCampaign: partial.CampaignPath, RunnerBuild: config.RunnerBuild, Backend: scriptProvider(),
	}, executionDependencies{executor: resumedExecutor},
	)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Attempted != 2 || resumed.Succeeded != 2 || resumed.RecoveryExecutions != 1 || resumed.ChoiceExploration == nil || resumed.ChoiceExploration.CommittedRounds != 2 || resumed.StopReason != StopExplorationExhausted {
		t.Fatalf("resumed exploration = %#v", resumed)
	}
	batch, err := campaign.OpenCampaign(resumed.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Executions) != 2 || batch.Record.RecoveryExecutions != 1 {
		t.Fatalf("resumed exploration batch = %#v", batch)
	}
}

func TestRunChoiceExplorationLeavesNoOpSelectPollsUnexpandedAcrossResume(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 3)
	baseExecutor := &explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit, noOpSelect: true}
	config, configDependencies := testConfig(t, preparer, explorationInterruptExecutor{exploration: baseExecutor}, "7", PolicyAll, 2)
	config.Strategy = StrategyChoiceExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 8
	config.MaxChoiceDepth = 4
	config.MaxExplorationBytes = 1 << 20

	partial, err := exploreWith(context.Background(), config, configDependencies)
	var hostErr *HostError
	if !errors.As(err, &hostErr) || partial.ChoiceExploration == nil || partial.ChoiceExploration.CommittedRounds != 1 {
		t.Fatalf("partial exploration = %#v, error = %v", partial, err)
	}
	resumedExecutor := &explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit, noOpSelect: true}
	resumed, err := exploreWith(context.Background(), CampaignSpec{
		ResumeCampaign: partial.CampaignPath, RunnerBuild: config.RunnerBuild, Backend: scriptProvider(),
	}, executionDependencies{executor: resumedExecutor},
	)
	if err != nil {
		t.Fatal(err)
	}
	// The runnable decision has one alternative to expand; the select's does
	// not, in the root's round or in the resumed one.
	if resumed.Attempted != 2 || resumed.StopReason != StopExplorationExhausted || resumed.ChoiceExploration == nil || resumed.ChoiceExploration.SeenPrefixes != 2 || resumed.ChoiceExploration.CommittedRounds != 2 || resumed.ChoiceExploration.OmittedBySelectReadiness != 2 {
		t.Fatalf("resumed exploration = %#v", resumed)
	}
	batch, err := campaign.OpenCampaign(resumed.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Executions) != 2 || batch.Record.ChoiceExploration == nil || batch.Record.ChoiceExploration.OmittedBySelectReadiness != 2 || batch.Record.ChoiceExploration.OmittedByDepth != 0 {
		t.Fatalf("resumed exploration record = %#v", batch.Record.ChoiceExploration)
	}
	resumedExecutor.mu.Lock()
	defer resumedExecutor.mu.Unlock()
	for _, request := range resumedExecutor.requests {
		if request.Choice == nil || request.Choice.ReplayPlan == nil {
			continue
		}
		if forced := request.Choice.ReplayPlan.Decisions; forced[len(forced)-1].Kind == choice.KindSelectPoll {
			t.Fatalf("resumed exploration forced the no-op select-poll decision: %#v", forced)
		}
	}
}

func TestRunChoiceExplorationResumesWithItsStartAndReportsItUnreached(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	config, configDependencies := testConfig(t, preparer, explorationRootInterruptExecutor{}, "7", PolicyAll, 1)
	config.Strategy = StrategyChoiceExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 8
	config.MaxChoiceDepth = 4
	config.ChoiceStartOrdinal = 1
	config.MaxExplorationBytes = 1 << 20

	partial, err := exploreWith(context.Background(), config, configDependencies)
	var hostErr *HostError
	if !errors.As(err, &hostErr) || partial.ChoiceExploration == nil || partial.ChoiceExploration.CommittedRounds != 0 || partial.ChoiceExploration.StartOrdinal != 1 {
		t.Fatalf("partial exploration = %#v, error = %v", partial, err)
	}
	resumed, err := exploreWith(context.Background(), CampaignSpec{
		ResumeCampaign: partial.CampaignPath, RunnerBuild: config.RunnerBuild, Backend: scriptProvider(),
	}, executionDependencies{executor: &explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Attempted != 1 || resumed.StopReason != StopChoiceStartUnreached || resumed.ChoiceExploration == nil || resumed.ChoiceExploration.StartOrdinal != 1 || resumed.ChoiceExploration.BoundedComplete || resumed.ChoiceExploration.Pending != 0 {
		t.Fatalf("resumed exploration = %#v", resumed)
	}
	batch, err := campaign.OpenCampaign(resumed.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if batch.Record.StopReason != string(StopChoiceStartUnreached) || batch.Record.ChoiceExploration == nil || batch.Record.ChoiceExploration.StartOrdinal != 1 {
		t.Fatalf("published exploration = %#v", batch.Record)
	}
}

func TestInspectChoicesListsReplayPlanDecisionOrdinals(t *testing.T) {
	limit := choiceTraceLimit(t, 4)
	preparer := newFakePreparer(t)
	executor := &fakeExecutor{result: func(uint64) backend.ExecutionResult {
		result := processResult(1, "failure", "")
		result.ChoiceTrace = completeChoiceTrace(t, preparer.prepared.BuildKey, limit, []choice.Record{
			{Ordinal: 0, Kind: choice.KindRunnable, Flags: choice.FlagDecision, SiteOffset: 0x10, Alternatives: 1},
			{Ordinal: 1, Kind: choice.KindRunnable, Flags: choice.FlagDecision, SiteOffset: 0x20, Alternatives: 2, Selected: 1},
			{Ordinal: 2, Kind: choice.KindSelectPoll, Flags: choice.FlagDecision, SiteOffset: 0x30, Alternatives: 3, Selected: 2},
			{Ordinal: 3, Kind: choice.KindSelectResult, Flags: choice.FlagObservation, SiteOffset: 0x30, Alternatives: 3, Selected: 1},
		})
		return result
	}}
	config, configDependencies := testConfig(t, preparer, executor, "1", PolicyAll, 1)
	config.ChoiceTraceLimit = limit
	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	report, err := Inspect(summary.Artifacts[0], InspectOptions{Choices: true})
	if err != nil {
		t.Fatal(err)
	}
	want := []ChoiceDecision{
		{Ordinal: 0, Kind: "runnable", SiteOffset: 0x20, Alternatives: 2, Selected: 0},
		{Ordinal: 1, Kind: "select-poll", SiteOffset: 0x30, Alternatives: 3, Selected: 2},
	}
	if report.Artifact == nil || report.Artifact.Choices == nil || !slices.Equal(report.Artifact.Choices.ReplayDecisions, want) {
		t.Fatalf("choice inspection = %#v", report.Artifact.Choices)
	}
}

func TestRunChoiceExplorationDivergingPrefixRetainsCompletedRound(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	executor := &explorationDivergenceExecutor{
		exploration:      &explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit, alternatives: 4},
		beforeDivergence: make(chan struct{}), afterDivergence: make(chan struct{}),
	}
	config, configDependencies := testConfig(t, preparer, executor, "7", PolicyAll, 3)
	config.Strategy = StrategyChoiceExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 8
	config.MaxChoiceDepth = 4
	config.MaxExplorationBytes = 1 << 20

	configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatalf("diverging exploration discarded its completed round: %v", err)
	}
	if summary.Attempted != 4 || summary.Succeeded != 3 || summary.Failures != 1 || summary.ReplayDivergences != 1 || summary.DistinctFailures != 0 || summary.ChoiceExploration == nil || summary.ChoiceExploration.CommittedRounds != 2 {
		t.Fatalf("diverging exploration = %#v", summary)
	}
	if len(summary.Artifacts) != 0 || len(summary.SuccessArtifacts) != 0 {
		t.Fatalf("diverging exploration retained candidate artifacts: %#v", summary)
	}
	batch, err := campaign.OpenCampaign(summary.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Executions) != 4 {
		t.Fatalf("committed executions = %d", len(batch.Executions))
	}
	var divergences int
	for _, run := range batch.Executions {
		if run.Reason == "replay_divergence" {
			divergences++
			if run.Domain != "runner" || run.OutcomeSHA256 != "" || run.FailureSignature != nil || run.Artifact != nil || run.ChoiceTraceSHA256 != nil || run.CandidateSHA256 == "" || run.ForcedDepth == nil || *run.ForcedDepth != 1 {
				t.Fatalf("divergent execution = %#v", run)
			}
		}
	}
	if divergences != 1 {
		t.Fatalf("retained divergences = %d", divergences)
	}
}

func TestRunChoiceExplorationExpandsCompleteTargetFailures(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	executor := &explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit, exitCode: 2}
	config, configDependencies := testConfig(t, preparer, executor, "7", PolicyAll, 2)
	config.Strategy = StrategyChoiceExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 8
	config.MaxChoiceDepth = 4
	config.MaxExplorationBytes = 1 << 20

	configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Attempted != 2 || summary.Failures != 2 || summary.Succeeded != 0 || summary.StopReason != StopExplorationExhausted || len(summary.Artifacts) != 2 {
		t.Fatalf("exploration target failures = %#v", summary)
	}
}

func TestRunChoiceExplorationFindsPinnedOutcomesWithinSeedSamplingBudget(t *testing.T) {
	toolchainRoot, err := filepath.Abs(filepath.Join("..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	testdata, err := filepath.Abs(filepath.Join("..", "backends", "native", "conformance", "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	supervisor := filepath.Join(t.TempDir(), "gomad")
	command := exec.Command(filepath.Join(toolchainRoot, "bin", "go"), "build", "-trimpath", "-o", supervisor, "./cmd/gomad")
	command.Dir = filepath.Join(testdata, "..", "..", "..", "..")
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

func distinctSuccessStdoutOutcomes(t *testing.T, paths []string) map[record.SHA256]struct{} {
	t.Helper()
	outcomes := make(map[record.SHA256]struct{}, len(paths))
	for _, path := range paths {
		artifact, err := artifact.OpenArtifact(path)
		if err != nil {
			t.Fatal(err)
		}
		outcomes[artifact.Manifest().Streams.Stdout.FullSHA256] = struct{}{}
		if err := artifact.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return outcomes
}
