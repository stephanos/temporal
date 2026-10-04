package campaign

import "testing"

func TestSeedControllerSchedulesAndAggregatesDeterministically(t *testing.T) {
	jobs := []SeedJob{{Ordinal: 0, Seed: 10}, {Ordinal: 2, Seed: 12}, {Ordinal: 3, Seed: 13}}
	next := 0
	controller, err := NewSeedController(SeedControllerConfig{
		Next: func() (SeedJob, bool) {
			if next == len(jobs) {
				return SeedJob{}, false
			}
			job := jobs[next]
			next++
			return job, true
		},
		Parallel: 2, Policy: FailurePolicyAll, FailureBudget: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	first, ok := controller.Next()
	if !ok || first != jobs[0] {
		t.Fatalf("first job = %#v, %t", first, ok)
	}
	second, ok := controller.Next()
	if !ok || second != jobs[1] {
		t.Fatalf("second job = %#v, %t", second, ok)
	}
	_, ok = controller.Next()
	if ok {
		t.Fatal("controller exceeded parallelism")
	}
	controller.Complete(Completion{Kind: CompletionSuccess})
	_, ok = controller.Next()
	if !ok {
		t.Fatal("third job was not scheduled")
	}
	controller.Complete(Completion{Kind: CompletionFailure, Domain: "watchdog", Reason: "world_replay_divergence", DistinctFailures: 1})
	controller.Complete(Completion{Kind: CompletionCancelled})
	_, ok = controller.Next()
	if ok || !controller.Done() {
		t.Fatal("controller did not exhaust")
	}
	controller.Finalize()
	want := CampaignStatistics{
		Attempted: 3, Succeeded: 1, Failures: 1, Watchdogs: 1, ReplayDivergences: 1,
		Cancelled: 1, DistinctFailures: 1, StopReason: StopSeedsExhausted,
	}
	if got := controller.Statistics(); got != want {
		t.Fatalf("statistics = %#v, want %#v", got, want)
	}
}

func TestSeedControllerAppliesFailurePolicies(t *testing.T) {
	for _, test := range []struct {
		name       string
		policy     FailurePolicy
		budget     uint64
		distinct   uint64
		wantReason ControllerStopReason
		wantCancel bool
	}{
		{name: "first", policy: FailurePolicyFirst, budget: 1, distinct: 1, wantReason: StopFirstFailure, wantCancel: true},
		{name: "budget", policy: FailurePolicyBudget, budget: 2, distinct: 2, wantReason: StopFailureBudget},
		{name: "all", policy: FailurePolicyAll, budget: 1, distinct: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			controller, err := NewSeedController(SeedControllerConfig{
				Next:     func() (SeedJob, bool) { return SeedJob{}, true },
				Parallel: 2, Policy: test.policy, FailureBudget: test.budget,
			})
			if err != nil {
				t.Fatal(err)
			}
			_, ok := controller.Next()
			if !ok {
				t.Fatal("first job was not scheduled")
			}
			_, ok = controller.Next()
			if !ok {
				t.Fatal("second job was not scheduled")
			}
			if cancel := controller.Complete(Completion{Kind: CompletionFailure, Domain: "watchdog", Reason: "world_replay_divergence", DistinctFailures: test.distinct}); cancel != test.wantCancel {
				t.Fatalf("cancel = %t, want %t", cancel, test.wantCancel)
			}
			want := CampaignStatistics{
				Attempted: 1, Failures: 1, Watchdogs: 1, ReplayDivergences: 1,
				DistinctFailures: test.distinct, StopReason: test.wantReason,
			}
			if got := controller.Statistics(); controller.Stopped() != (test.wantReason != "") || got != want || controller.Active() != 1 {
				t.Fatalf("controller stopped = %t, statistics = %#v, want %#v", controller.Stopped(), got, want)
			}
			_, ok = controller.Next()
			if ok != (test.wantReason == "") {
				t.Fatal("stopped controller scheduled another job")
			}
			if test.wantReason == "" {
				controller.Complete(Completion{Kind: CompletionUnclassified})
			}
			controller.Complete(Completion{Kind: CompletionUnclassified})
			if test.wantReason == "" {
				controller.Stop()
			}
			if !controller.Done() {
				t.Fatal("stopped controller did not drain")
			}
		})
	}
}

func TestSeedControllerCompletesUnclassifiedAndDuplicateFailureAtomically(t *testing.T) {
	initial := CampaignStatistics{Attempted: 4, Succeeded: 2, Failures: 2, DistinctFailures: 1}
	controller, err := NewSeedController(SeedControllerConfig{
		Next:     func() (SeedJob, bool) { return SeedJob{Ordinal: 4, Seed: 14}, true },
		Parallel: 2, Policy: FailurePolicyBudget, FailureBudget: 2, Initial: initial,
	})
	if err != nil {
		t.Fatal(err)
	}
	controller.Next()
	controller.Next()
	if cancel := controller.Complete(Completion{Kind: CompletionUnclassified}); cancel {
		t.Fatal("unclassified completion cancelled active work")
	}
	want := CampaignStatistics{Attempted: 5, Succeeded: 2, Failures: 2, DistinctFailures: 1}
	if got := controller.Statistics(); got != want || controller.Active() != 1 {
		t.Fatalf("statistics = %#v, active = %d, want %#v and 1", got, controller.Active(), want)
	}
	if cancel := controller.Complete(Completion{Kind: CompletionFailure, Domain: "target", Reason: "exit_nonzero", DistinctFailures: 1}); cancel {
		t.Fatal("duplicate failure cancelled active work")
	}
	want = CampaignStatistics{Attempted: 6, Succeeded: 2, Failures: 3, DistinctFailures: 1}
	if got := controller.Statistics(); got != want || controller.Active() != 0 || controller.Stopped() {
		t.Fatalf("statistics = %#v, active = %d, stopped = %t, want %#v", got, controller.Active(), controller.Stopped(), want)
	}
}

func TestSeedControllerRejectsInactiveCompletionBeforeChangingStatistics(t *testing.T) {
	controller, err := NewSeedController(SeedControllerConfig{
		Next: func() (SeedJob, bool) { return SeedJob{}, false }, Parallel: 1, Policy: FailurePolicyAll,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recovered := recover(); recovered != "gomad3: completed an inactive campaign attempt" {
			t.Fatalf("panic = %#v", recovered)
		}
		if got := controller.Statistics(); got != (CampaignStatistics{}) {
			t.Fatalf("statistics after rejected completion = %#v", got)
		}
	}()
	controller.Complete(Completion{Kind: CompletionSuccess})
}

func TestSeedControllerRestoresSatisfiedPolicy(t *testing.T) {
	controller, err := NewSeedController(SeedControllerConfig{
		Next: func() (SeedJob, bool) { return SeedJob{}, true }, Parallel: 1,
		Policy: FailurePolicyFirst, FailureBudget: 1,
		Initial: CampaignStatistics{Failures: 1, DistinctFailures: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !controller.Stopped() || !controller.Done() || controller.Statistics().StopReason != StopFirstFailure {
		t.Fatalf("restored controller = %#v", controller)
	}
}

func TestSeedControllerRestoresAdmissionAndWholeStatistics(t *testing.T) {
	for _, test := range []struct {
		name    string
		policy  FailurePolicy
		budget  uint64
		stopped bool
		want    CampaignStatistics
	}{
		{name: "all admits retained failures", policy: FailurePolicyAll, budget: 1,
			want: CampaignStatistics{Attempted: 8, Succeeded: 3, Failures: 4, Watchdogs: 2, ReplayDivergences: 1, Cancelled: 1, DistinctFailures: 2}},
		{name: "budget below threshold admits", policy: FailurePolicyBudget, budget: 3,
			want: CampaignStatistics{Attempted: 8, Succeeded: 3, Failures: 4, Watchdogs: 2, ReplayDivergences: 1, Cancelled: 1, DistinctFailures: 2}},
		{name: "budget at threshold stops", policy: FailurePolicyBudget, budget: 2, stopped: true,
			want: CampaignStatistics{Attempted: 8, Succeeded: 3, Failures: 4, Watchdogs: 2, ReplayDivergences: 1, Cancelled: 1, DistinctFailures: 2, StopReason: StopFailureBudget}},
		{name: "budget above threshold stops", policy: FailurePolicyBudget, budget: 1, stopped: true,
			want: CampaignStatistics{Attempted: 8, Succeeded: 3, Failures: 4, Watchdogs: 2, ReplayDivergences: 1, Cancelled: 1, DistinctFailures: 2, StopReason: StopFailureBudget}},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			controller, err := NewSeedController(SeedControllerConfig{
				Next: func() (SeedJob, bool) {
					calls++
					return SeedJob{Ordinal: 8, Seed: 18}, true
				},
				Parallel: 1, Policy: test.policy, FailureBudget: test.budget,
				Initial: CampaignStatistics{Attempted: 8, Succeeded: 3, Failures: 4, Watchdogs: 2, ReplayDivergences: 1, Cancelled: 1, DistinctFailures: 2},
			})
			if err != nil {
				t.Fatal(err)
			}
			if got := controller.Statistics(); got != test.want || controller.Stopped() != test.stopped || controller.Done() != test.stopped || controller.Active() != 0 {
				t.Fatalf("restored statistics = %#v, stopped = %t, done = %t, active = %d, want %#v, stopped = %t", got, controller.Stopped(), controller.Done(), controller.Active(), test.want, test.stopped)
			}
			job, ok := controller.Next()
			if test.stopped {
				if ok || job != (SeedJob{}) || calls != 0 {
					t.Fatalf("stopped admission = %#v, %t, source calls = %d", job, ok, calls)
				}
			} else if !ok || job != (SeedJob{Ordinal: 8, Seed: 18}) || calls != 1 || controller.Active() != 1 {
				t.Fatalf("resumed admission = %#v, %t, source calls = %d, active = %d", job, ok, calls, controller.Active())
			}
			if got := controller.Statistics(); got != test.want {
				t.Fatalf("admission changed statistics = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestSeedControllerRejectsInvalidConfiguration(t *testing.T) {
	validNext := func() (SeedJob, bool) { return SeedJob{}, false }
	for _, config := range []SeedControllerConfig{
		{},
		{Next: validNext},
		{Next: validNext, Parallel: 1, Policy: FailurePolicyBudget},
		{Next: validNext, Parallel: 1, Policy: "unknown"},
	} {
		_, err := NewSeedController(config)
		if err == nil {
			t.Fatalf("NewSeedController(%#v) succeeded", config)
		}
	}
}
