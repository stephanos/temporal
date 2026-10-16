package runner

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/campaign"
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

type environmentRecordingProvider struct {
	*boundaryProvider
	environment []string
}

func (p *environmentRecordingProvider) ValidatePrepared(_ target.Spec, _ target.Prepared, environment []string) error {
	p.environment = environment
	return nil
}

func TestCampaignPlanPreparerValidatesWithRecordedEnvironment(t *testing.T) {
	source := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(source, []byte("prepared"), 0o500); err != nil {
		t.Fatal(err)
	}
	provider := &environmentRecordingProvider{boundaryProvider: &boundaryProvider{}}
	recorded := []string{"MODE=retained", "GOMAD_IO_PROFILE=fixture", "GOMAD_CHOICE_PROFILE=fixture"}
	preparer := &campaignPlanPreparer{source: target.Prepared{Path: source, Size: uint64(len("prepared"))}, provider: provider, environment: recorded}
	if _, err := preparer.Prepare(t.Context(), target.Spec{PreparationRoot: filepath.Join(t.TempDir(), "shard")}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(provider.environment, recorded) {
		t.Fatalf("shard target validated with environment %q; want %q", provider.environment, recorded)
	}
}

// A resumed shard schedules the ordinals it owns that are neither committed
// nor already admitted, and its summary mirrors its recorded statistics.
func TestPendingJobsSkipsCommittedAndAdmittedOrdinalsAndSynchronizesStatistics(t *testing.T) {
	selection, err := ParseSeeds("10-16")
	if err != nil {
		t.Fatal(err)
	}
	jobs := pendingJobs{seeds: selection.Iterator(), shard: CampaignShard{Index: 0, Count: 2}, completed: map[uint64]struct{}{2: {}, 4: {}}}
	var scheduled []runJob
	for {
		job, ok := jobs.Next()
		if !ok {
			break
		}
		scheduled = append(scheduled, job)
	}
	if want := []runJob{{ordinal: 0, seed: 10}, {ordinal: 6, seed: 16}}; !slices.Equal(scheduled, want) {
		t.Fatalf("scheduled = %#v, want %#v", scheduled, want)
	}
	statistics := campaign.CampaignStatistics{}
	statistics.Record(campaign.CompletedSuccess())
	statistics.Record(campaign.CompletedCancelled())
	statistics.Finalize()
	summary := CampaignResult{SelectionCount: selection.Count()}
	synchronizeCampaignStatistics(&summary, statistics)
	if summary.Attempted != 2 || summary.Succeeded != 1 || summary.Cancelled != 1 || summary.StopReason != StopSeedsExhausted {
		t.Fatalf("summary = %#v", summary)
	}
}
