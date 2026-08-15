package runner

import "testing"

func TestSeedCampaignSchedulesPendingOrdinalsWithinParallelism(t *testing.T) {
	selection, err := ParseSeeds("10-13")
	if err != nil {
		t.Fatal(err)
	}
	summary := CampaignResult{SelectionCount: selection.Count()}
	campaign := newSeedCampaign(selection, map[uint64]struct{}{1: {}}, 2, PolicyAll, 1, &summary)

	first, ok := campaign.Next()
	if !ok || first != (runJob{ordinal: 0, seed: 10}) {
		t.Fatalf("first job = %#v, %t", first, ok)
	}
	second, ok := campaign.Next()
	if !ok || second != (runJob{ordinal: 2, seed: 12}) {
		t.Fatalf("second job = %#v, %t", second, ok)
	}
	if _, ok := campaign.Next(); ok || campaign.Active() != 2 {
		t.Fatalf("campaign exceeded parallelism: active = %d", campaign.Active())
	}
	campaign.FinishAttempt()
	third, ok := campaign.Next()
	if !ok || third != (runJob{ordinal: 3, seed: 13}) {
		t.Fatalf("third job = %#v, %t", third, ok)
	}
	campaign.FinishAttempt()
	campaign.FinishAttempt()
	if _, ok := campaign.Next(); ok || !campaign.Done() || summary.Attempted != 3 {
		t.Fatalf("completed campaign = %#v, active = %d", summary, campaign.Active())
	}
	campaign.Finalize()
	if summary.StopReason != StopSeedsExhausted {
		t.Fatalf("stop reason = %q", summary.StopReason)
	}
}

func TestSeedCampaignAppliesFailurePolicies(t *testing.T) {
	selection, err := ParseSeeds("0-3")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		policy     FailurePolicy
		budget     uint64
		distinct   uint64
		wantReason StopReason
		wantCancel bool
	}{
		{name: "first", policy: PolicyFirst, budget: 1, distinct: 1, wantReason: StopFirstFailure, wantCancel: true},
		{name: "budget", policy: PolicyBudget, budget: 2, distinct: 2, wantReason: StopFailureBudget},
	} {
		t.Run(test.name, func(t *testing.T) {
			summary := CampaignResult{SelectionCount: selection.Count()}
			campaign := newSeedCampaign(selection, nil, 2, test.policy, test.budget, &summary)
			if _, ok := campaign.Next(); !ok {
				t.Fatal("first job was not scheduled")
			}
			if _, ok := campaign.Next(); !ok {
				t.Fatal("second job was not scheduled")
			}
			campaign.FinishAttempt()
			if cancel := campaign.RecordFailure("watchdog", "world_replay_divergence", test.distinct); cancel != test.wantCancel {
				t.Fatalf("cancel = %t, want %t", cancel, test.wantCancel)
			}
			if !campaign.Stopped() || summary.Failures != 1 || summary.Watchdogs != 1 || summary.ReplayDivergences != 1 || summary.DistinctFailures != test.distinct || summary.StopReason != test.wantReason {
				t.Fatalf("stopped campaign = %#v", summary)
			}
			if _, ok := campaign.Next(); ok {
				t.Fatal("stopped campaign scheduled another job")
			}
			campaign.FinishAttempt()
			if !campaign.Done() {
				t.Fatal("stopped campaign did not drain active jobs")
			}
		})
	}
}

func TestSeedCampaignRestoresSatisfiedPolicyAndAggregatesResults(t *testing.T) {
	selection, err := ParseSeeds("7-8")
	if err != nil {
		t.Fatal(err)
	}
	summary := CampaignResult{SelectionCount: selection.Count(), Failures: 1, DistinctFailures: 1}
	campaign := newSeedCampaign(selection, nil, 1, PolicyFirst, 1, &summary)
	if !campaign.Stopped() || !campaign.Done() || summary.StopReason != StopFirstFailure {
		t.Fatalf("restored campaign = %#v", summary)
	}

	summary = CampaignResult{SelectionCount: selection.Count()}
	campaign = newSeedCampaign(selection, nil, 1, PolicyAll, 1, &summary)
	if _, ok := campaign.Next(); !ok {
		t.Fatal("job was not scheduled")
	}
	campaign.FinishAttempt()
	campaign.RecordSuccess()
	if _, ok := campaign.Next(); !ok {
		t.Fatal("second job was not scheduled")
	}
	campaign.FinishAttempt()
	campaign.RecordCancelled()
	if summary.Attempted != 2 || summary.Succeeded != 1 || summary.Cancelled != 1 {
		t.Fatalf("aggregate = %#v", summary)
	}
}
