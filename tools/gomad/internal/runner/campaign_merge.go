package runner

import (
	"context"
	"errors"

	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/inspection"
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
	PlanSHA256       record.SHA256          `json:"plan_sha256"`
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
	// JournalSegments described the retired segmented journal; it is always zero.
	JournalSegments uint64 `json:"journal_segments"`
	SelectionCount  uint64 `json:"selection_count"`
}

func MergeCampaignShards(ctx context.Context, spec CampaignMergeSpec) (CampaignMergeResult, error) {
	opened, err := campaign.OpenPortablePlan(spec.PlanPath)
	if err != nil {
		return CampaignMergeResult{}, err
	}
	selection, err := parseCampaignSelection(opened.Plan.Selection, uint64(opened.Plan.SelectionCount))
	if err != nil || selection.Count() != uint64(opened.Plan.SelectionCount) {
		return CampaignMergeResult{}, errors.Join(errors.New("campaign plan seed selection is invalid"), err)
	}
	if opened.Plan.Journal == nil || opened.Plan.Artifacts == nil {
		return CampaignMergeResult{}, errors.New("campaign plan capacities are incomplete")
	}
	merged, err := campaign.MergeCampaigns(ctx, campaign.MergeSpec{
		Output: spec.Output, Guidance: opened.Plan.Guidance, PlanSHA256: opened.Identity, Selection: opened.Plan.Selection, SelectionCount: selection.Count(),
		Journal: *opened.Plan.Journal, Artifacts: *opened.Plan.Artifacts, Partial: spec.Partial, ShardPaths: append([]string(nil), spec.Shards...), SeedAt: selection.SeedAt,
	})
	if err != nil {
		return CampaignMergeResult{}, err
	}
	return campaignMergeResult(merged), nil
}

func campaignMergeResult(merged campaign.MergedCampaign) CampaignMergeResult {
	projected := inspection.ProjectMerge(merged)
	missing := make([]CampaignOrdinalRange, len(projected.Missing))
	for index, value := range projected.Missing {
		missing[index] = CampaignOrdinalRange(value)
	}
	return CampaignMergeResult{
		Path: projected.Path, PlanSHA256: projected.PlanSHA256, Partial: projected.Partial, Missing: missing, Shards: projected.Shards, Attempted: projected.Attempted,
		Succeeded: projected.Succeeded, Failures: projected.Failures, Watchdogs: projected.Watchdogs, Cancelled: projected.Cancelled, DistinctFailures: projected.DistinctFailures,
		RetainedEvidence: projected.RetainedEvidence, RetainedBytes: projected.RetainedBytes, JournalBytes: projected.JournalBytes, SelectionCount: projected.SelectionCount,
	}
}
