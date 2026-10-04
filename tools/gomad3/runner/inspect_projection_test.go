package runner

import (
	"encoding/json"
	"testing"

	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
)

func TestInspectionCapacityProjectionPreservesReports(t *testing.T) {
	journal := campaign.ExecutionJournalPlan{MaximumExecutions: 1, MaximumBytes: 2, SegmentBytes: 3, SegmentRecords: 4, MaximumSegments: 5, MaximumPartialExecutions: 6, CapacityOutcome: "infrastructure_failure"}
	capacity := campaign.ArtifactCapacityPlan{FailureArtifacts: 11, FailureBytes: 12, SuccessArtifacts: 13, SuccessBytes: 14, TotalBytes: 15, TranscriptBytes: 16, FailureOutcome: "failure-bound", SuccessOutcome: "success-bound"}
	plan := projectCampaignPlan(openedCampaignPlan{plan: campaign.CampaignPlan{Journal: &journal, Artifacts: &capacity}})
	inspection, err := projectCampaign(campaign.Campaign{Record: campaign.CampaignRecord{Artifacts: &capacity}, Journal: &campaign.ExecutionJournalInfo{Limits: journal}})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		value any
		want  string
	}{
		{journal, `{"maximum_executions":"1","maximum_bytes":"2","segment_bytes":"3","segment_records":"4","maximum_segments":"5","maximum_partial_executions":"6","capacity_outcome":"infrastructure_failure"}`},
		{plan.Journal, `{"maximum_executions":"1","maximum_bytes":"2","segment_bytes":"3","segment_records":"4","maximum_segments":"5","maximum_partial_executions":"6","capacity_outcome":"infrastructure_failure"}`},
		{inspection.Journal.Limits, `{"maximum_executions":"1","maximum_bytes":"2","segment_bytes":"3","segment_records":"4","maximum_segments":"5","maximum_partial_executions":"6","capacity_outcome":"infrastructure_failure"}`},
		{capacity, `{"failure_artifacts":"11","failure_bytes":"12","success_artifacts":"13","success_bytes":"14","total_bytes":"15","transcript_bytes":"16","failure_outcome":"failure-bound","success_outcome":"success-bound"}`},
		{plan.ArtifactCapacity, `{"failure_artifacts":"11","failure_bytes":"12","success_artifacts":"13","success_bytes":"14","total_bytes":"15","transcript_bytes":"16","failure_outcome":"failure-bound","success_outcome":"success-bound"}`},
		{inspection.ArtifactCapacity, `{"failure_artifacts":"11","failure_bytes":"12","success_artifacts":"13","success_bytes":"14","total_bytes":"15","transcript_bytes":"16","failure_outcome":"failure-bound","success_outcome":"success-bound"}`},
	} {
		data, err := json.Marshal(test.value)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != test.want {
			t.Fatalf("capacity report = %s, want %s", data, test.want)
		}
	}
	absent, err := projectCampaign(campaign.Campaign{})
	if err != nil {
		t.Fatal(err)
	}
	if absent.ArtifactCapacity != nil || absent.Journal != nil {
		t.Fatalf("absent pointers changed: %+v", absent)
	}
	capacity.FailureBytes = 99
	journal.MaximumBytes = 99
	data, err := json.Marshal(plan.ArtifactCapacity)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"failure_artifacts":"11","failure_bytes":"12","success_artifacts":"13","success_bytes":"14","total_bytes":"15","transcript_bytes":"16","failure_outcome":"failure-bound","success_outcome":"success-bound"}` {
		t.Fatalf("report aliases source: %s", data)
	}
}
