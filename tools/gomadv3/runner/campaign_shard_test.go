package runner

import (
	"testing"
)

func TestCampaignShardPartitionsCanonicalOrdinals(t *testing.T) {
	selection, err := ParseSeeds("9-11,2,40-43")
	if err != nil {
		t.Fatal(err)
	}

	seen := make(map[uint64]uint64, selection.Count())
	expected := []uint64{3, 3, 2}
	for index := uint64(0); index < 3; index++ {
		shard := CampaignShard{Index: index, Count: 3}
		if err := shard.Validate(); err != nil {
			t.Fatal(err)
		}
		if got := shard.SelectionCount(selection.Count()); got != expected[index] {
			t.Fatalf("SelectionCount() = %d, want %d", got, expected[index])
		}
		for ordinal := uint64(0); ordinal < selection.Count(); ordinal++ {
			if shard.Owns(ordinal) {
				seen[ordinal]++
			}
		}
	}
	if len(seen) != int(selection.Count()) {
		t.Fatalf("partition contains %d ordinals, want %d", len(seen), selection.Count())
	}
	for ordinal := uint64(0); ordinal < selection.Count(); ordinal++ {
		if seen[ordinal] != 1 {
			t.Fatalf("ordinal %d ownership count = %d", ordinal, seen[ordinal])
		}
	}
}

func TestCampaignShardRejectsInvalidAssignments(t *testing.T) {
	for _, shard := range []CampaignShard{{}, {Index: 1, Count: 1}, {Index: 3, Count: 3}} {
		if err := shard.Validate(); err == nil {
			t.Fatalf("CampaignShard%+v passed validation", shard)
		}
	}
	if err := (CampaignShard{Index: 0, Count: 1}).Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestPendingJobsRetainsGlobalOrdinalsForShard(t *testing.T) {
	selection, err := ParseSeeds("100-105")
	if err != nil {
		t.Fatal(err)
	}
	jobs := pendingJobs{seeds: selection.Iterator(), shard: CampaignShard{Index: 1, Count: 3}, completed: map[uint64]struct{}{4: {}}}

	job, ok := jobs.Next()
	if !ok || job != (runJob{ordinal: 1, seed: 101}) {
		t.Fatalf("Next() = %#v, %t", job, ok)
	}
	_, ok = jobs.Next()
	if ok {
		t.Fatal("Next() returned a completed shard ordinal")
	}
}
