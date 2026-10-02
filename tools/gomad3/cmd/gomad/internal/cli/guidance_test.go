package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/runner"
)

func TestFullyAnsweredGuidanceReportsZeroExecutionsAndSuccessStatus(t *testing.T) {
	summary := runner.CampaignResult{CampaignPath: "/campaign", StopReason: runner.StopSeedsExhausted, Guidance: &runner.GuidanceSummary{Requested: 4, Answered: 4}}
	for _, jsonOutput := range []bool{false, true} {
		var stdout, stderr bytes.Buffer
		if err := newExploreReporter(jsonOutput, &stdout, &stderr).Result(summary); err != nil {
			t.Fatal(err)
		}
		if exploreSummaryStatus(summary) != 0 || stderr.Len() != 0 {
			t.Fatalf("status=%d stderr=%s", exploreSummaryStatus(summary), &stderr)
		}
		if jsonOutput {
			var event exploreEvent
			if err := json.Unmarshal(stdout.Bytes(), &event); err != nil {
				t.Fatal(err)
			}
			if event.Classification != "success" || event.Guidance == nil || *event.Guidance != *summary.Guidance {
				t.Fatalf("event=%#v", event)
			}
			for _, want := range []string{`"requested":4`, `"answered":4`, `"guided":0`, `"new_executions":0`} {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("output=%s missing=%s", &stdout, want)
				}
			}
		} else {
			for _, want := range []string{"requested=4 answered=4 guided=0 new-executions=0", "all requested seeds are answered", "--guide-regression", "exit 0"} {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("output=%s missing=%s", &stdout, want)
				}
			}
		}
	}
}

func TestGuidanceReportsNoCorpusSelectionAndNewExecutions(t *testing.T) {
	var stdout, stderr bytes.Buffer
	summary := runner.CampaignResult{SelectionCount: 3, Attempted: 3, Succeeded: 3, StopReason: runner.StopSeedsExhausted, Guidance: &runner.GuidanceSummary{Requested: 4, Answered: 1, NewExecutions: 3}}
	if err := newExploreReporter(false, &stdout, &stderr).Result(summary); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "guidance selected no corpus seeds") || !strings.Contains(stdout.String(), "new-executions=3") {
		t.Fatalf("output=%s", &stdout)
	}
}

func TestResumeGuidanceModePreservesExplicitFalseAndTrue(t *testing.T) {
	for _, value := range []string{"false", "true"} {
		var got *bool
		dependencies := resumeDependencies{identity: func(string) (string, string, string, error) { return "/toolchain", "/gomad", "runner", nil }, run: func(_ context.Context, spec runner.ResumeSpec) (runner.CampaignResult, error) {
			got = spec.GuideRegression
			return runner.CampaignResult{StopReason: runner.StopSeedsExhausted}, nil
		}}
		var stdout, stderr bytes.Buffer
		status := runResumeWith([]string{"--guide-regression=" + value, "/campaign"}, &stdout, &stderr, dependencies)
		if status != 0 || got == nil || *got != (value == "true") {
			t.Fatalf("status=%d mode=%v", status, got)
		}
	}
}

func TestGuideRegressionRequiresGuidance(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := runExplore([]string{"--json", "--guide-regression", "go-run", "./fixture"}, &stdout, &stderr)
	if status != 2 || !strings.Contains(stdout.String(), "--guide-regression requires --guide") {
		t.Fatalf("status=%d stdout=%s stderr=%s", status, &stdout, &stderr)
	}
}

func TestEmptyShardDoesNotReportPartiallyAnsweredPlanAsFullyAnswered(t *testing.T) {
	var stdout, stderr bytes.Buffer
	summary := runner.CampaignResult{SelectionCount: 0, StopReason: runner.StopSeedsExhausted, Guidance: &runner.GuidanceSummary{Requested: 2, Answered: 1}}
	if err := newExploreReporter(false, &stdout, &stderr).Result(summary); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout.String(), "all requested seeds are answered") || !strings.Contains(stdout.String(), "requested=2 answered=1") || !strings.Contains(stdout.String(), "guidance selected no corpus seeds") {
		t.Fatalf("empty shard output=%s", &stdout)
	}
}
