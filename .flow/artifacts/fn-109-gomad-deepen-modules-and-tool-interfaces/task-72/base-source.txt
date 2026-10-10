package runner

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
)

type injectedCampaign struct {
	CampaignSpec
	dependencies executionDependencies
}

func injectedTestConfig(t *testing.T, preparer Preparer, executor executionRunner, seeds string, policy FailurePolicy, parallel int) injectedCampaign {
	spec, dependencies := testConfig(t, preparer, executor, seeds, policy, parallel)
	return injectedCampaign{CampaignSpec: spec, dependencies: dependencies}
}

func injectedCompletionCampaign(t *testing.T, strategy Strategy, coverage CoverageMode, fault func(*execution.Result), runErr error) (injectedCampaign, faultExecutor) {
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
	scripts map[uint64]func(context.Context) (execution.Result, error)
}

func (executor scriptedSeedExecutor) Run(ctx context.Context, request execution.Spec) (execution.Result, error) {
	if script := executor.scripts[seedFromEnvironment(request.Env)]; script != nil {
		return script(ctx)
	}
	return processResult(0, "", ""), nil
}

func cancelledByCampaign(ctx context.Context) (execution.Result, error) {
	<-ctx.Done()
	result := processResult(0, "", "")
	result.Cancelled = true
	result.Termination = execution.TerminationSignal
	result.Signal = "killed"
	return result, nil
}

// Completions are counted in selection order, so a campaign whose earlier seed
// fails the host still counts the attempts it drains afterwards. Each row pins
// every statistic, including those of attempts that end before classification.
func TestSeedCompletionKeepsCampaignStatistics(t *testing.T) {
	const failedCampaign = "campaign recoverable-failure"
	supervisionErr := errors.New("supervisor broke")
	failSupervision := func(context.Context) (execution.Result, error) {
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
				executor := &fakeExecutor{result: func(seed uint64) execution.Result {
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
				Completion: completionObservation{Counts: [5]uint64{3, 0, 1, 0, 1}, Artifacts: []string{"gomad3.target-failure/v1 nonzero_exit exact none no-choices"}, Journal: []string{"target nonzero_exit exit", "runner runner_cancelled none", "runner runner_cancelled none"}, Partials: []string{"00000000000000000001-2 preserve-partial", "00000000000000000002-3 preserve-partial"}},
				Statistics: campaign.CampaignStatistics{Attempted: 3, Failures: 1, Cancelled: 2, DistinctFailures: 1, StopReason: campaign.StopFirstFailure},
			},
		},
		{
			name: "failure budget",
			configure: func(t *testing.T) (context.Context, injectedCampaign) {
				executor := &fakeExecutor{result: func(seed uint64) execution.Result {
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
				return context.Background(), injectedTestConfig(t, newFakePreparer(t), scriptedSeedExecutor{scripts: map[uint64]func(context.Context) (execution.Result, error){1: failSupervision}}, "1", PolicyAll, 1)
			},
			want: seedCompletionObservation{
				Completion: completionObservation{Reason: "target_supervision", Cause: "supervisor broke", Counts: [5]uint64{1, 0, 0, 0, 1}, Artifacts: []string{"gomad3.runner-failure/v1 target_supervision none none no-choices"}, Journal: []string{"runner target_supervision none"}, Partials: []string{"00000000000000000000-1 preserve-partial", failedCampaign}},
				Statistics: campaign.CampaignStatistics{Attempted: 1, DistinctFailures: 1},
			},
		},
		{
			name: "supervision failure drains a cancelled attempt",
			configure: func(t *testing.T) (context.Context, injectedCampaign) {
				executor := scriptedSeedExecutor{scripts: map[uint64]func(context.Context) (execution.Result, error){1: failSupervision, 2: cancelledByCampaign}}
				return context.Background(), injectedTestConfig(t, newFakePreparer(t), executor, "1-2", PolicyAll, 2)
			},
			want: seedCompletionObservation{
				Completion: completionObservation{Reason: "target_supervision", Cause: "supervisor broke", Counts: [5]uint64{2, 0, 0, 0, 1}, Artifacts: []string{"gomad3.runner-failure/v1 target_supervision none none no-choices"}, Journal: []string{"runner target_supervision none", "runner target_supervision none"}, Partials: []string{"00000000000000000000-1 preserve-partial", "00000000000000000001-2 preserve-partial", failedCampaign}},
				Statistics: campaign.CampaignStatistics{Attempted: 2, Cancelled: 1, DistinctFailures: 1},
			},
		},
		{
			name: "supervision failure drains a success",
			configure: func(t *testing.T) (context.Context, injectedCampaign) {
				executor := scriptedSeedExecutor{scripts: map[uint64]func(context.Context) (execution.Result, error){1: failSupervision}}
				return context.Background(), injectedTestConfig(t, newFakePreparer(t), executor, "1-2", PolicyAll, 2)
			},
			want: seedCompletionObservation{
				Completion: completionObservation{Reason: "target_supervision", Cause: "supervisor broke", Counts: [5]uint64{2, 0, 0, 0, 0}, Artifacts: []string{"gomad3.runner-failure/v1 target_supervision none none no-choices"}, Journal: []string{"runner target_supervision none", "success success exit"}, Partials: []string{"00000000000000000000-1 preserve-partial", failedCampaign}},
				Statistics: campaign.CampaignStatistics{Attempted: 2},
			},
		},
		{
			name: "supervision failure drains a failure",
			configure: func(t *testing.T) (context.Context, injectedCampaign) {
				executor := scriptedSeedExecutor{scripts: map[uint64]func(context.Context) (execution.Result, error){1: failSupervision, 2: func(context.Context) (execution.Result, error) {
					return processResult(1, "failed", ""), nil
				}}}
				return context.Background(), injectedTestConfig(t, newFakePreparer(t), executor, "1-2", PolicyFirst, 2)
			},
			want: seedCompletionObservation{
				Completion: completionObservation{Reason: "target_supervision", Cause: "supervisor broke", Counts: [5]uint64{2, 0, 1, 0, 2}, Artifacts: []string{"gomad3.runner-failure/v1 target_supervision none none no-choices", "gomad3.target-failure/v1 nonzero_exit exact none no-choices"}, Journal: []string{"runner target_supervision none", "target nonzero_exit exit"}, Partials: []string{"00000000000000000000-1 preserve-partial", failedCampaign}},
				Statistics: campaign.CampaignStatistics{Attempted: 2, Failures: 1, DistinctFailures: 2},
			},
		},
		{
			name: "prepared target integrity",
			configure: func(t *testing.T) (context.Context, injectedCampaign) {
				return context.Background(), injectedTestConfig(t, newFakePreparer(t), mutatingExecutor{}, "1", PolicyAll, 1)
			},
			want: seedCompletionObservation{
				Completion: completionObservation{Reason: "prepared_target_integrity", Cause: "prepared target changed after preparation", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{"00000000000000000000-1 captured", failedCampaign}},
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
				Completion: completionObservation{Reason: "cancelled", Cause: "context canceled", Counts: [5]uint64{1, 0, 0, 0, 0}, Partials: []string{"00000000000000000000-1 starting", failedCampaign}},
				Statistics: campaign.CampaignStatistics{Attempted: 1},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, config := test.configure(t)
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
	malformedWorld := func(result *execution.Result) {
		result.WorldRecord = completionWorldRecord(t, 7)
		result.WorldRecord[len(result.WorldRecord)-1] ^= 1
	}
	watchdog := func(result *execution.Result) {
		result.Termination = execution.TerminationSignal
		result.Signal = "SIGKILL"
		result.WatchdogTimeout = true
		result.ChoiceTrace = execution.ChoiceTrace{Profile: choice.Profile, Limit: result.ChoiceTrace.Limit}
	}
	cancelled := func(result *execution.Result) {
		result.Termination = execution.TerminationSignal
		result.Signal = "killed"
		result.Cancelled = true
		result.ChoiceTrace = execution.ChoiceTrace{Profile: choice.Profile, Limit: result.ChoiceTrace.Limit}
	}
	for _, test := range []struct {
		name     string
		coverage CoverageMode
		fault    func(*execution.Result)
		err      error
		want     campaign.CampaignStatistics
	}{
		{name: "malformed World", fault: malformedWorld, want: campaign.CampaignStatistics{Attempted: 1, DistinctFailures: 1}},
		{name: "supervision rejected the choice trace", err: execution.ErrChoiceTraceMalformed, want: campaign.CampaignStatistics{Attempted: 1}},
		{name: "watchdog", coverage: CoverageSemanticChoice, fault: watchdog, want: campaign.CampaignStatistics{Attempted: 1, Failures: 1, Watchdogs: 1, DistinctFailures: 1, StopReason: campaign.StopSeedsExhausted}},
		{name: "cancelled execution", coverage: CoverageSemanticChoice, fault: cancelled, want: campaign.CampaignStatistics{Attempted: 1, Failures: 1, DistinctFailures: 1, StopReason: campaign.StopSeedsExhausted}},
	} {
		t.Run(test.name, func(t *testing.T) {
			config, _ := injectedCompletionCampaign(t, StrategySeed, test.coverage, test.fault, test.err)
			summary, err := exploreWith(context.Background(), config.CampaignSpec, config.dependencies)
			if observed := observeSeedCompletion(t, summary, err).Statistics; observed != test.want {
				t.Fatalf("statistics = %#v, want %#v", observed, test.want)
			}
		})
	}
}
