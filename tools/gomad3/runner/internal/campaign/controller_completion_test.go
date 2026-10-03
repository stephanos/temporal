package campaign

import "testing"

type completionStep struct {
	kind     string
	domain   string
	reason   string
	distinct uint64
	cancel   bool
}

var (
	succeeded    = completionStep{kind: "success"}
	cancelled    = completionStep{kind: "cancelled"}
	unclassified = completionStep{kind: "unclassified"}
)

func failed(domain, reason string, distinct uint64, cancel bool) completionStep {
	return completionStep{kind: "failure", domain: domain, reason: reason, distinct: distinct, cancel: cancel}
}

func completeStep(controller *SeedController, step completionStep) bool {
	controller.FinishAttempt()
	switch step.kind {
	case "success":
		controller.RecordSuccess()
	case "cancelled":
		controller.RecordCancelled()
	case "failure":
		return controller.RecordFailure(step.domain, step.reason, step.distinct)
	}
	return false
}

// Each row schedules as many jobs as it completes, completes them in order and
// pins the whole statistics value, the stop state and the drain state.
func TestSeedControllerCompletionKeepsWholeStatistics(t *testing.T) {
	for _, test := range []struct {
		name        string
		policy      FailurePolicy
		budget      uint64
		initial     CampaignStatistics
		stop        bool
		steps       []completionStep
		want        CampaignStatistics
		wantStopped bool
	}{
		{name: "success", policy: FailurePolicyAll, steps: []completionStep{succeeded}, want: CampaignStatistics{Attempted: 1, Succeeded: 1}},
		{name: "cancellation", policy: FailurePolicyAll, steps: []completionStep{cancelled}, want: CampaignStatistics{Attempted: 1, Cancelled: 1}},
		{name: "unclassified attempt", policy: FailurePolicyAll, steps: []completionStep{unclassified}, want: CampaignStatistics{Attempted: 1}},
		{
			name: "watchdog", policy: FailurePolicyAll, steps: []completionStep{failed("watchdog", "watchdog_timeout", 1, false)},
			want: CampaignStatistics{Attempted: 1, Failures: 1, Watchdogs: 1, DistinctFailures: 1},
		},
		{
			name: "replay divergence", policy: FailurePolicyAll, steps: []completionStep{failed("target", "world_replay_divergence", 0, false)},
			want: CampaignStatistics{Attempted: 1, Failures: 1, ReplayDivergences: 1},
		},
		{
			name: "distinct and duplicate failures", policy: FailurePolicyAll,
			steps: []completionStep{failed("target", "nonzero_exit", 1, false), failed("target", "nonzero_exit", 1, false), succeeded, failed("target", "nonzero_exit", 2, false)},
			want:  CampaignStatistics{Attempted: 4, Succeeded: 1, Failures: 3, DistinctFailures: 2},
		},
		{
			name: "first failure cancels active work once", policy: FailurePolicyFirst,
			steps:       []completionStep{succeeded, cancelled, failed("target", "nonzero_exit", 1, true), failed("target", "nonzero_exit", 2, false), cancelled},
			want:        CampaignStatistics{Attempted: 5, Succeeded: 1, Failures: 2, Cancelled: 2, DistinctFailures: 2, StopReason: StopFirstFailure},
			wantStopped: true,
		},
		{
			name: "budget counts distinct failures", policy: FailurePolicyBudget, budget: 2,
			steps:       []completionStep{failed("target", "nonzero_exit", 1, false), failed("target", "nonzero_exit", 1, false), failed("watchdog", "watchdog_timeout", 2, false), unclassified},
			want:        CampaignStatistics{Attempted: 4, Failures: 3, Watchdogs: 1, DistinctFailures: 2, StopReason: StopFailureBudget},
			wantStopped: true,
		},
		{
			name: "budget unclassified attempts never stop", policy: FailurePolicyBudget, budget: 1,
			steps: []completionStep{unclassified, unclassified},
			want:  CampaignStatistics{Attempted: 2},
		},
		{
			name: "all never stops", policy: FailurePolicyAll,
			steps: []completionStep{failed("target", "nonzero_exit", 1, false), failed("target", "nonzero_exit", 2, false), failed("target", "nonzero_exit", 3, false)},
			want:  CampaignStatistics{Attempted: 3, Failures: 3, DistinctFailures: 3},
		},
		{
			name: "stopped campaign keeps classifying without a policy stop", policy: FailurePolicyFirst, stop: true,
			steps:       []completionStep{failed("target", "nonzero_exit", 1, false), cancelled},
			want:        CampaignStatistics{Attempted: 2, Failures: 1, Cancelled: 1, DistinctFailures: 1},
			wantStopped: true,
		},
		{
			name: "resume seeded counters accumulate", policy: FailurePolicyBudget, budget: 3,
			initial: CampaignStatistics{Attempted: 5, Succeeded: 2, Failures: 3, Watchdogs: 1, ReplayDivergences: 1, Cancelled: 0, DistinctFailures: 2},
			steps:   []completionStep{succeeded, failed("target", "world_replay_divergence", 2, false), failed("watchdog", "watchdog_timeout", 3, false)},
			want: CampaignStatistics{
				Attempted: 8, Succeeded: 3, Failures: 5, Watchdogs: 2, ReplayDivergences: 2, DistinctFailures: 3, StopReason: StopFailureBudget,
			},
			wantStopped: true,
		},
		{
			name: "resume seeded first policy without failures", policy: FailurePolicyFirst,
			initial:     CampaignStatistics{Attempted: 2, Succeeded: 1, Cancelled: 1},
			steps:       []completionStep{failed("target", "nonzero_exit", 1, true)},
			want:        CampaignStatistics{Attempted: 3, Succeeded: 1, Failures: 1, Cancelled: 1, DistinctFailures: 1, StopReason: StopFirstFailure},
			wantStopped: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			scheduled := uint64(0)
			controller, err := NewSeedController(SeedControllerConfig{
				Next: func() (SeedJob, bool) {
					if scheduled == uint64(len(test.steps)) {
						return SeedJob{}, false
					}
					job := SeedJob{Ordinal: scheduled, Seed: 100 + scheduled}
					scheduled++
					return job, true
				},
				Parallel: len(test.steps), Policy: test.policy, FailureBudget: test.budget, Initial: test.initial,
			})
			if err != nil {
				t.Fatal(err)
			}
			for ordinal := range test.steps {
				job, ok := controller.Next()
				if want := (SeedJob{Ordinal: uint64(ordinal), Seed: 100 + uint64(ordinal)}); !ok || job != want {
					t.Fatalf("job %d = %#v, %t, want %#v", ordinal, job, ok, want)
				}
			}
			if test.stop {
				controller.Stop()
			}
			for index, step := range test.steps {
				if cancel := completeStep(controller, step); cancel != step.cancel {
					t.Fatalf("completion %d cancel = %t, want %t", index, cancel, step.cancel)
				}
				if active := controller.Active(); active != len(test.steps)-index-1 {
					t.Fatalf("completion %d active = %d", index, active)
				}
			}
			if got := controller.Statistics(); got != test.want {
				t.Fatalf("statistics = %#v, want %#v", got, test.want)
			}
			if controller.Stopped() != test.wantStopped {
				t.Fatalf("stopped = %t, want %t", controller.Stopped(), test.wantStopped)
			}
			if _, ok := controller.Next(); ok || !controller.Done() {
				t.Fatal("controller did not drain")
			}
		})
	}
}

func TestSeedControllerRejectsCompletionWithoutActiveWork(t *testing.T) {
	initial := CampaignStatistics{Attempted: 3, Succeeded: 3}
	controller, err := NewSeedController(SeedControllerConfig{
		Next: func() (SeedJob, bool) { return SeedJob{}, true }, Parallel: 1, Policy: FailurePolicyFirst, Initial: initial,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []completionStep{succeeded, cancelled, unclassified, failed("target", "nonzero_exit", 1, true)} {
		func() {
			defer func() {
				if recovered := recover(); recovered != "gomad3: completed an inactive campaign attempt" {
					t.Fatalf("completion %#v recovered %#v", step, recovered)
				}
			}()
			completeStep(controller, step)
			t.Fatalf("completion %#v without active work succeeded", step)
		}()
		if got := controller.Statistics(); got != initial || controller.Stopped() || controller.Active() != 0 {
			t.Fatalf("rejected completion changed the controller: statistics = %#v, stopped = %t", got, controller.Stopped())
		}
	}
}
