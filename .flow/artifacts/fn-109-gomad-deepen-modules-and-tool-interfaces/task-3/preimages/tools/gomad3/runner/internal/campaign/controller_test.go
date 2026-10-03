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
	controller.FinishAttempt()
	controller.RecordSuccess()
	_, ok = controller.Next()
	if !ok {
		t.Fatal("third job was not scheduled")
	}
	controller.FinishAttempt()
	controller.RecordFailure("watchdog", "world_replay_divergence", 1)
	controller.FinishAttempt()
	controller.RecordCancelled()
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
			controller.FinishAttempt()
			if cancel := controller.RecordFailure("watchdog", "world_replay_divergence", test.distinct); cancel != test.wantCancel {
				t.Fatalf("cancel = %t, want %t", cancel, test.wantCancel)
			}
			want := CampaignStatistics{
				Attempted: 1, Failures: 1, Watchdogs: 1, ReplayDivergences: 1,
				DistinctFailures: test.distinct, StopReason: test.wantReason,
			}
			if got := controller.Statistics(); !controller.Stopped() || got != want {
				t.Fatalf("controller stopped = %t, statistics = %#v, want %#v", controller.Stopped(), got, want)
			}
			_, ok = controller.Next()
			if ok {
				t.Fatal("stopped controller scheduled another job")
			}
			controller.FinishAttempt()
			if !controller.Done() {
				t.Fatal("stopped controller did not drain")
			}
		})
	}
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
