package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/runner"
)

func TestChoiceExplorationDivergenceClassificationAndStatus(t *testing.T) {
	for _, test := range []struct {
		name           string
		summary        runner.CampaignResult
		classification string
		status         int
	}{
		{"choice confidence failure", runner.CampaignResult{Failures: 1, ReplayDivergences: 1, ChoiceExploration: &runner.ChoiceExplorationSummary{}}, "replay_divergence", 3},
		{"choice mixed failure", runner.CampaignResult{Failures: 2, ReplayDivergences: 1, ChoiceExploration: &runner.ChoiceExplorationSummary{}}, "mixed_failure", 3},
		{"confidence counter without target failure", runner.CampaignResult{ReplayDivergences: 1, ChoiceExploration: &runner.ChoiceExplorationSummary{}}, "replay_divergence", 3},
		{"world replay divergence", runner.CampaignResult{Failures: 1, ReplayDivergences: 1}, "replay_divergence", 1},
		{"target failure", runner.CampaignResult{Failures: 1}, "target_failure", 1},
		{"success", runner.CampaignResult{}, "success", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := classifyExploreSummary(test.summary); got != test.classification {
				t.Fatalf("classification = %s", got)
			}
			if got := exploreSummaryStatus(test.summary); got != test.status {
				t.Fatalf("status = %d", got)
			}
			var stdout, stderr bytes.Buffer
			status := runResumeWith([]string{"--json", "campaign"}, &stdout, &stderr, resumeDependencies{
				identity: func(string) (string, string, string, error) { return "toolchain", "gomad", "runner", nil },
				run:      func(context.Context, runner.ResumeSpec) (runner.CampaignResult, error) { return test.summary, nil },
			})
			if status != test.status {
				t.Fatalf("resume status=%d output=%s stderr=%s", status, stdout.String(), stderr.String())
			}
			var result exploreEvent
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Classification != test.classification {
				t.Fatalf("reported classification=%s", result.Classification)
			}
		})
	}
}

func TestInspectReportsCandidateReplayDivergence(t *testing.T) {
	candidate := runner.ExecutionInspection{Strategy: string(runner.StrategyChoiceExploration), Domain: "runner", Reason: "replay_divergence", Divergence: &choice.DivergenceEvidence{Ordinal: 7, Reason: choice.DivergenceAlternativeSet}}
	inspected := runner.CampaignInspection{ReplayDivergences: 1, Executions: []runner.ExecutionInspection{candidate}}
	var output bytes.Buffer
	printer := &inspectionPrinter{output: &output}
	printCampaignInspection(printer, &inspected)
	if printer.err != nil {
		t.Fatal(printer.err)
	}
	for _, expected := range []string{"reason=replay_divergence", "divergence-ordinal=7", "divergence-reason=alternative_set"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("inspect missing %s: %s", expected, output.String())
		}
	}
	encoded, err := json.Marshal(inspected)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"\"divergence\"", "\"ordinal\":7", "\"reason\":5", "\"replay_divergences\":1"} {
		if !bytes.Contains(encoded, []byte(expected)) {
			t.Fatalf("inspect JSON missing %s: %s", expected, encoded)
		}
	}
}
