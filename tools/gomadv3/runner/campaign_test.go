package runner

import "testing"

func TestShardedSeedControllerSchedulesPendingOrdinalsAndSynchronizesStatistics(t *testing.T) {
	selection, err := ParseSeeds("10-14")
	if err != nil {
		t.Fatal(err)
	}

	summary := CampaignResult{SelectionCount: selection.Count()}
	controller, err := newShardedSeedController(selection, CampaignShard{Index: 0, Count: 2}, map[uint64]struct{}{2: {}}, 2, PolicyAll, 1, summary)
	if err != nil {
		t.Fatal(err)
	}
	synchronizeCampaignStatistics(&summary, controller.Statistics())

	first, ok := controller.Next()
	if !ok || first.Ordinal != 0 || first.Seed != 10 {
		t.Fatalf("first job = %#v, %t", first, ok)
	}
	second, ok := controller.Next()
	if !ok || second.Ordinal != 4 || second.Seed != 14 {
		t.Fatalf("second job = %#v, %t", second, ok)
	}
	_, ok = controller.Next()
	if ok {
		t.Fatal("controller exceeded parallelism")
	}

	controller.FinishAttempt()
	controller.RecordSuccess()
	controller.FinishAttempt()
	controller.RecordCancelled()
	_, ok = controller.Next()
	if ok || !controller.Done() {
		t.Fatal("controller did not exhaust")
	}
	controller.Finalize()
	synchronizeCampaignStatistics(&summary, controller.Statistics())

	if summary.Attempted != 2 || summary.Succeeded != 1 || summary.Cancelled != 1 || summary.StopReason != StopSeedsExhausted {
		t.Fatalf("summary = %#v", summary)
	}
}
