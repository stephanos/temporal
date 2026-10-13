package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/runner"
)

func TestSoftAssertionProgressReportsEscapedObservations(t *testing.T) {
	seed := uint64(0)
	progress := runner.CampaignEvent{Phase: runner.ProgressRunning, ExecutionSeed: &seed, SoftAssertions: &record.SoftAssertions{Messages: []string{"invalid\nstate"}, ReportOnly: true}}
	for _, jsonOutput := range []bool{false, true} {
		var stdout, stderr bytes.Buffer
		reporter := newExploreReporter(jsonOutput, &stdout, &stderr)
		if err := reporter.Progress(progress); err != nil {
			t.Fatal(err)
		}
		if jsonOutput {
			var event exploreEvent
			if err := json.Unmarshal(stdout.Bytes(), &event); err != nil {
				t.Fatal(err)
			}
			if event.ExecutionSeed == nil || *event.ExecutionSeed != 0 || event.SoftAssertions == nil || event.SoftAssertions.Messages[0] != "invalid\nstate" || !event.SoftAssertions.ReportOnly || stderr.Len() != 0 {
				t.Fatalf("JSON oracle report: %+v stderr=%q", event, stderr.String())
			}
		} else if stdout.Len() != 0 || !strings.Contains(stderr.String(), `seed=0 report-only=true inconclusive=false messages=["invalid\nstate"]`) {
			t.Fatalf("human oracle report: stdout=%q stderr=%q", stdout.String(), stderr.String())
		}
	}
	progress.ExecutionSeed = nil
	if err := newExploreReporter(false, &bytes.Buffer{}, &bytes.Buffer{}).Progress(progress); err == nil {
		t.Fatal("oracle observation without a seed admitted")
	}
}
