package runner

import (
	"context"
	"errors"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/campaignstore"
)

type CampaignMergeSpec struct {
	PlanPath string
	Shards   []string
	Output   string
	Partial  bool
}

type CampaignOrdinalRange struct {
	Start uint64 `json:"start"`
	End   uint64 `json:"end"`
}

type CampaignMergeResult struct {
	Path             string                 `json:"path"`
	PlanSHA256       evidence.SHA256        `json:"plan_sha256"`
	Partial          bool                   `json:"partial"`
	Missing          []CampaignOrdinalRange `json:"missing"`
	Shards           uint64                 `json:"shards"`
	Attempted        uint64                 `json:"attempted"`
	Succeeded        uint64                 `json:"succeeded"`
	Failures         uint64                 `json:"failures"`
	Watchdogs        uint64                 `json:"watchdogs"`
	Cancelled        uint64                 `json:"cancelled"`
	DistinctFailures uint64                 `json:"distinct_failures"`
	RetainedEvidence uint64                 `json:"retained_evidence"`
	RetainedBytes    uint64                 `json:"retained_bytes"`
	JournalBytes     uint64                 `json:"journal_bytes"`
	JournalSegments  uint64                 `json:"journal_segments"`
	SelectionCount   uint64                 `json:"selection_count"`
}

func MergeCampaignShards(ctx context.Context, spec CampaignMergeSpec) (CampaignMergeResult, error) {
	opened, err := openCampaignPlan(spec.PlanPath)
	if err != nil {
		return CampaignMergeResult{}, err
	}
	selection, err := ParseSeeds(opened.plan.Selection)
	if err != nil || selection.Count() != uint64(opened.plan.SelectionCount) {
		return CampaignMergeResult{}, errors.Join(errors.New("campaign plan seed selection is invalid"), err)
	}
	if opened.plan.Journal == nil || opened.plan.Artifacts == nil {
		return CampaignMergeResult{}, errors.New("campaign plan capacities are incomplete")
	}
	merged, err := campaignstore.MergeCampaigns(ctx, campaignstore.MergeSpec{
		Output: spec.Output, PlanSHA256: opened.identity, Selection: opened.plan.Selection, SelectionCount: selection.Count(),
		Journal: *opened.plan.Journal, Artifacts: *opened.plan.Artifacts, Partial: spec.Partial, ShardPaths: append([]string(nil), spec.Shards...), SeedAt: selection.SeedAt,
	})
	if err != nil {
		return CampaignMergeResult{}, err
	}
	return campaignMergeResult(merged), nil
}

func campaignMergeResult(merged campaignstore.MergedCampaign) CampaignMergeResult {
	record := merged.Record
	missing := make([]CampaignOrdinalRange, len(record.Missing))
	for index, value := range record.Missing {
		missing[index] = CampaignOrdinalRange{Start: uint64(value.Start), End: uint64(value.End)}
	}
	return CampaignMergeResult{
		Path: merged.Path, PlanSHA256: record.PlanSHA256, Partial: record.Partial, Missing: missing, Shards: uint64(len(record.Shards)), Attempted: uint64(record.Attempted),
		Succeeded: uint64(record.Succeeded), Failures: uint64(record.Failures), Watchdogs: uint64(record.Watchdogs), Cancelled: uint64(record.Cancelled), DistinctFailures: uint64(record.DistinctFailures),
		RetainedEvidence: uint64(record.RetainedEvidence), RetainedBytes: uint64(record.EvidenceBytes), JournalBytes: uint64(record.Journal.Bytes), JournalSegments: uint64(record.Journal.Segments), SelectionCount: uint64(record.SelectionCount),
	}
}
