package campaign

import "testing"

type completionStep struct {
	completion Completion
	cancel     bool
}

var (
	succeeded    = completionStep{completion: CompletedSuccess()}
	cancelled    = completionStep{completion: CompletedCancelled()}
	unclassified = completionStep{completion: CompletedUnclassified()}
)

func failed(domain, reason string, distinct uint64, cancel bool) completionStep {
	return completionStep{completion: CompletedFailure(domain, reason, distinct), cancel: cancel}
}

// Each row records its completions in order and pins the whole statistics
// value and the first policy stop any of them triggers; only a first-failure
// stop requests cancellation of active attempts.
func TestCampaignStatisticsRecordKeepsWholeStatistics(t *testing.T) {
	for _, test := range []struct {
		name     string
		policy   FailurePolicy
		budget   uint64
		initial  CampaignStatistics
		steps    []completionStep
		want     CampaignStatistics
		wantStop ControllerStopReason
	}{
		{name: "success", policy: failurePolicyAll, steps: []completionStep{succeeded}, want: CampaignStatistics{Attempted: 1, Succeeded: 1}},
		{name: "cancellation", policy: failurePolicyAll, steps: []completionStep{cancelled}, want: CampaignStatistics{Attempted: 1, Cancelled: 1}},
		{name: "unclassified attempt", policy: failurePolicyAll, steps: []completionStep{unclassified}, want: CampaignStatistics{Attempted: 1}},
		{
			name: "watchdog", policy: failurePolicyAll, steps: []completionStep{failed("watchdog", "watchdog_timeout", 1, false)},
			want: CampaignStatistics{Attempted: 1, Failures: 1, Watchdogs: 1, DistinctFailures: 1},
		},
		{
			name: "replay divergence", policy: failurePolicyAll, steps: []completionStep{failed("target", "world_replay_divergence", 0, false)},
			want: CampaignStatistics{Attempted: 1, Failures: 1, ReplayDivergences: 1},
		},
		{
			name: "distinct and duplicate failures", policy: failurePolicyAll,
			steps: []completionStep{failed("target", "nonzero_exit", 1, false), failed("target", "nonzero_exit", 1, false), succeeded, failed("target", "nonzero_exit", 2, false)},
			want:  CampaignStatistics{Attempted: 4, Succeeded: 1, Failures: 3, DistinctFailures: 2},
		},
		{
			name: "first failure cancels active work once", policy: failurePolicyFirst,
			steps:    []completionStep{succeeded, cancelled, failed("target", "nonzero_exit", 1, true), failed("target", "nonzero_exit", 2, true), cancelled},
			want:     CampaignStatistics{Attempted: 5, Succeeded: 1, Failures: 2, Cancelled: 2, DistinctFailures: 2},
			wantStop: StopFirstFailure,
		},
		{
			name: "budget counts distinct failures", policy: failurePolicyBudget, budget: 2,
			steps:    []completionStep{failed("target", "nonzero_exit", 1, false), failed("target", "nonzero_exit", 1, false), failed("watchdog", "watchdog_timeout", 2, false), unclassified},
			want:     CampaignStatistics{Attempted: 4, Failures: 3, Watchdogs: 1, DistinctFailures: 2},
			wantStop: StopFailureBudget,
		},
		{
			name: "budget unclassified attempts never stop", policy: failurePolicyBudget, budget: 1,
			steps: []completionStep{unclassified, unclassified},
			want:  CampaignStatistics{Attempted: 2},
		},
		{
			name: "all never stops", policy: failurePolicyAll,
			steps: []completionStep{failed("target", "nonzero_exit", 1, false), failed("target", "nonzero_exit", 2, false), failed("target", "nonzero_exit", 3, false)},
			want:  CampaignStatistics{Attempted: 3, Failures: 3, DistinctFailures: 3},
		},
		{
			name: "resume seeded counters accumulate", policy: failurePolicyBudget, budget: 3,
			initial: CampaignStatistics{Attempted: 5, Succeeded: 2, Failures: 3, Watchdogs: 1, ReplayDivergences: 1, Cancelled: 0, DistinctFailures: 2},
			steps:   []completionStep{succeeded, failed("target", "world_replay_divergence", 2, false), failed("watchdog", "watchdog_timeout", 3, false)},
			want: CampaignStatistics{
				Attempted: 8, Succeeded: 3, Failures: 5, Watchdogs: 2, ReplayDivergences: 2, DistinctFailures: 3,
			},
			wantStop: StopFailureBudget,
		},
		{
			name: "resume seeded first policy without failures", policy: failurePolicyFirst,
			initial:  CampaignStatistics{Attempted: 2, Succeeded: 1, Cancelled: 1},
			steps:    []completionStep{failed("target", "nonzero_exit", 1, true)},
			want:     CampaignStatistics{Attempted: 3, Succeeded: 1, Failures: 1, Cancelled: 1, DistinctFailures: 1},
			wantStop: StopFirstFailure,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			statistics := test.initial
			var stop ControllerStopReason
			for index, step := range test.steps {
				statistics.Record(step.completion)
				decided := PolicyStop(test.policy, test.budget, step.completion)
				if (decided == StopFirstFailure) != step.cancel {
					t.Fatalf("completion %d stop = %q, want cancellation %t", index, decided, step.cancel)
				}
				if stop == "" {
					stop = decided
				}
			}
			if statistics != test.want || stop != test.wantStop {
				t.Fatalf("statistics = %#v, stop = %q, want %#v, %q", statistics, stop, test.want, test.wantStop)
			}
		})
	}
}

func TestCampaignStatisticsRejectsCompletionWithoutClassification(t *testing.T) {
	for _, completion := range []Completion{{}, {Kind: completionFailure + 1}} {
		statistics := CampaignStatistics{Attempted: 3, Succeeded: 3}
		func() {
			defer func() {
				if recovered := recover(); recovered != "gomad: completed a campaign attempt without a classification" {
					t.Fatalf("completion %#v recovered %#v", completion, recovered)
				}
			}()
			statistics.Record(completion)
			t.Fatalf("completion %#v was recorded", completion)
		}()
		if statistics != (CampaignStatistics{Attempted: 3, Succeeded: 3}) {
			t.Fatalf("rejected completion changed statistics: %#v", statistics)
		}
	}
}

func TestCampaignStatisticsFinalizeKeepsACommittedStop(t *testing.T) {
	exhausted := CampaignStatistics{Attempted: 1}
	exhausted.Finalize()
	stopped := CampaignStatistics{Attempted: 1, StopReason: StopFirstFailure}
	stopped.Finalize()
	if exhausted.StopReason != StopSeedsExhausted || stopped.StopReason != StopFirstFailure {
		t.Fatalf("finalized stops = %q, %q", exhausted.StopReason, stopped.StopReason)
	}
}

func TestPolicyStopDecidesOneOrderedCompletion(t *testing.T) {
	for _, test := range []struct {
		policy     FailurePolicy
		budget     uint64
		completion Completion
		want       ControllerStopReason
	}{
		{failurePolicyFirst, 1, CompletedFailure("target", "nonzero_exit", 1), StopFirstFailure},
		{failurePolicyFirst, 1, CompletedSuccess(), ""},
		{failurePolicyFirst, 1, CompletedCancelled(), ""},
		{failurePolicyFirst, 1, CompletedUnclassified(), ""},
		{failurePolicyBudget, 2, CompletedFailure("target", "nonzero_exit", 1), ""},
		{failurePolicyBudget, 2, CompletedFailure("watchdog", "watchdog_timeout", 2), StopFailureBudget},
		{failurePolicyBudget, 2, CompletedFailure("watchdog", "watchdog_timeout", 3), StopFailureBudget},
		{failurePolicyAll, 1, CompletedFailure("target", "nonzero_exit", 5), ""},
	} {
		if got := PolicyStop(test.policy, test.budget, test.completion); got != test.want {
			t.Fatalf("PolicyStop(%s, %d, %#v) = %q, want %q", test.policy, test.budget, test.completion, got, test.want)
		}
	}
}
