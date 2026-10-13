package runner

import (
	"errors"
	"reflect"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/runner/internal/campaign"
)

func TestSoftAssertionResumedReceiptReportsDetachedObservation(t *testing.T) {
	report := &record.SoftAssertions{Messages: []string{"invalid state"}, ReportOnly: true}
	receipt := campaign.Receipt{Observation: campaign.ExecutionRecord{Seed: 7, SoftAssertions: report}}
	local := localCampaign{}
	observed, err := local.resumedObservation(receipt)
	if err != nil {
		t.Fatal(err)
	}
	var events []CampaignEvent
	if err := reportSoftAssertions(func(event CampaignEvent) error {
		events = append(events, event)
		return nil
	}, "/campaign", observed.run); err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].ExecutionSeed == nil || *events[0].ExecutionSeed != 7 || !reflect.DeepEqual(events[0].SoftAssertions, report) {
		t.Fatalf("resumed receipt report = %+v", events)
	}
	events[0].SoftAssertions.Messages[0] = "caller changed report"
	if report.Messages[0] != "invalid state" {
		t.Fatal("callback shared durable receipt metadata")
	}
	want := errors.New("callback failed")
	if err := reportSoftAssertions(func(CampaignEvent) error { return want }, "/campaign", observed.run); !errors.Is(err, want) {
		t.Fatalf("callback error was lost: %v", err)
	}
}

func TestSoftAssertionProgressSkipsAbsentOrUnobservedPolicy(t *testing.T) {
	for _, report := range []*record.SoftAssertions{nil, {ReportOnly: true}, {Messages: []string{"invalid state"}}} {
		if err := reportSoftAssertions(func(CampaignEvent) error {
			t.Fatal("unexpected observation")
			return nil
		}, "/campaign", campaign.ExecutionRecord{SoftAssertions: report}); err != nil {
			t.Fatal(err)
		}
	}
}
