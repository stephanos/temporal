package runner

import (
	"context"
	"crypto/sha256"
	"errors"
	"reflect"
	"sync"
	"testing"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	choiceengine "go.temporal.io/server/tools/gomad3/runner/internal/exploration/choice"
)

type candidateDivergenceExecutor struct {
	base     *explorationExecutor
	mu       sync.Mutex
	diverged bool
	change   func(*execution.Result, *choice.Divergence) error
}

func (e *candidateDivergenceExecutor) Run(ctx context.Context, request execution.Spec) (execution.Result, error) {
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
	return result, &execution.ChoiceReplayDivergenceError{Divergence: divergence}
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
	typed := func(d choice.Divergence) error { return &execution.ChoiceReplayDivergenceError{Divergence: d} }
	for _, test := range []struct {
		name, reason string
		change       func(*execution.Result, *choice.Divergence) error
	}{
		{"normal error", "target_supervision", func(_ *execution.Result, _ *choice.Divergence) error { return errors.New("executor failed") }},
		{"cancelled", "runner_cancelled", func(r *execution.Result, _ *choice.Divergence) error { r.Cancelled = true; return nil }},
		{"typed cancelled", "target_supervision", func(r *execution.Result, d *choice.Divergence) error { r.Cancelled = true; return typed(*d) }},
		{"overflow", "choice_trace_overflow", func(_ *execution.Result, _ *choice.Divergence) error { return execution.ErrChoiceTraceOverflow }},
		{"malformed trace", "choice_trace_malformed", func(_ *execution.Result, _ *choice.Divergence) error { return execution.ErrChoiceTraceMalformed }},
		{"unterminated trace", "choice_trace_unterminated", func(_ *execution.Result, _ *choice.Divergence) error { return execution.ErrChoiceTraceUnterminated }},
		{"missing identity", "target_supervision", func(_ *execution.Result, d *choice.Divergence) error {
			d.Reason = choice.DivergenceIdentityMissing
			d.Observed = nil
			return typed(*d)
		}},
		{"duplicate identity", "target_supervision", func(_ *execution.Result, d *choice.Divergence) error {
			d.Reason = choice.DivergenceIdentityDuplicate
			d.Observed = nil
			return typed(*d)
		}},
		{"alternative capacity", "target_supervision", func(_ *execution.Result, d *choice.Divergence) error {
			d.Reason = choice.DivergenceAlternativeCapacity
			d.Observed = nil
			return typed(*d)
		}},
		{"tape exhaustion", "target_supervision", func(_ *execution.Result, d *choice.Divergence) error {
			d.Reason = choice.DivergenceTapeExhausted
			d.Ordinal = 1
			d.Expected = nil
			return typed(*d)
		}},
		{"typed watchdog", "target_supervision", func(r *execution.Result, d *choice.Divergence) error { r.WatchdogTimeout = true; return typed(*d) }},
		{"wrong prefix", "target_supervision", func(_ *execution.Result, d *choice.Divergence) error { d.Expected.SiteOffset++; return typed(*d) }},
		{"malformed observed", "target_supervision", func(_ *execution.Result, d *choice.Divergence) error {
			d.Observed.SelectedIdentity = [32]byte{}
			return typed(*d)
		}},
		{"joined failure", "target_supervision", func(_ *execution.Result, d *choice.Divergence) error {
			return errors.Join(typed(*d), errors.New("output close failed"))
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			config, configDependencies := divergenceCampaignConfig(t, PolicyAll, 1)
			configDependencies.executor.(*candidateDivergenceExecutor).change = test.change
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
	state, err := choiceengine.New(choiceengine.Config{Execution: identity, ControllerSHA256: choiceengine.ImplementationSHA256(), BaseSeed: 7, Parallel: 1, MaxExecutions: 8, MaxChoiceDepth: 4, MaxExplorationBytes: 1 << 20, FailurePolicy: choiceengine.PolicyAll, FailureBudget: 1})
	if err != nil {
		t.Fatal(err)
	}
	round, _ := state.NextRound()
	result := processResult(0, "", "")
	result.Cancelled = true
	_, err = processExplorationCompletion(t.Context(), campaignRequestFromSpec(config), prepared, nil, nil, "", "", nil, state, round, 0, runCompletion{job: runJob{seed: 7}, result: result}, &CampaignResult{}, nil, nil, nil)
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
	resumed, err := exploreWith(t.Context(), CampaignSpec{ResumeCampaign: partial.CampaignPath, RunnerBuild: config.RunnerBuild, SupervisorCommand: []string{"unused"}}, executionDependencies{executor: executor.base})
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
			configDependencies.executor.(*candidateDivergenceExecutor).change = func(_ *execution.Result, d *choice.Divergence) error {
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
				return &execution.ChoiceReplayDivergenceError{Divergence: *d}
			}
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
