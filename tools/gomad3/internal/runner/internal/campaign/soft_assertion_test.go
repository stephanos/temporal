package campaign

import (
	"reflect"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
)

func TestSoftAssertionReceiptSurvivesResumeWithoutReexecution(t *testing.T) {
	fixture := newFailureCampaignWith(t, failureCampaignOptions{selection: "7", count: 1, parallel: 1, policy: "all", budget: 1, softAssertionReportOnly: true})
	journal := fixture.journal
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.Admit([]SeedJob{{Ordinal: 0, Seed: 7}}); err != nil {
		t.Fatal(err)
	}
	observed := successExecution(0, 7)
	observed.SoftAssertions = &record.SoftAssertions{Messages: []string{"invalid state"}, ReportOnly: true}
	if err := journal.RecordReceipt(Receipt{Observation: observed}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	resumed, _, err := ResumeCampaignJournal(t.Context(), journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resumed.Close(); err != nil {
			t.Error(err)
		}
	}()
	frontier := resumed.Frontier()
	if len(frontier.Retry()) != 0 || !reflect.DeepEqual(frontier.Receipts[0].Observation.SoftAssertions, observed.SoftAssertions) {
		t.Fatalf("receipt lost assertion annotation: %+v", frontier)
	}
	frontier.Receipts[0].Observation.SoftAssertions.Messages[0] = "mutated caller copy"
	if resumed.Frontier().Receipts[0].Observation.SoftAssertions.Messages[0] != "invalid state" {
		t.Fatal("receipt report was not detached")
	}
	if err := resumed.CommitOutcome(resumed.Frontier().Receipts[0].Observation, nil); err != nil {
		t.Fatal(err)
	}
	if len(resumed.Frontier().Receipts) != 0 {
		t.Fatal("committed receipt remained pending")
	}
}

func TestSoftAssertionReceiptRequiresSealedPolicyAndBoundedMessages(t *testing.T) {
	for _, policy := range []bool{false, true} {
		fixture := newFailureCampaignWith(t, failureCampaignOptions{selection: "7", count: 1, parallel: 1, policy: "all", budget: 1, softAssertionReportOnly: policy})
		for _, messages := range [][]string{{"a"}, {"z", "a"}, {""}, {string([]byte{0xff})}} {
			run := successExecution(0, 7)
			run.SoftAssertions = &record.SoftAssertions{Messages: messages, ReportOnly: true}
			err := fixture.journal.state.validateSoftAssertionReport(run)
			valid := policy && reflect.DeepEqual(messages, []string{"a"})
			if valid && err != nil || !valid && err == nil {
				t.Fatalf("policy=%t messages=%q annotation admission=%v", policy, messages, err)
			}
		}
	}
}
