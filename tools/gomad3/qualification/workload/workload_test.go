package workload

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/qualification"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
)

func TestRunWorkloadRepeatsOneCampaignAndPublishesTheClaim(t *testing.T) {
	var calls int
	var progress []Progress
	result, err := Run(context.Background(), Spec{
		Command: []string{"gomad", "qualify"}, Seed: 7, Repeat: 2, ArtifactRoot: "/artifacts",
		Campaign: runner.CampaignSpec{Seeds: "7"},
		Progress: func(event Progress) error {
			progress = append(progress, event)
			return nil
		},
		Explore: func(context.Context, runner.CampaignSpec) (runner.CampaignResult, error) {
			calls++
			evidence := workloadEvidence(7)
			return runner.CampaignResult{CampaignPath: fmt.Sprintf("/artifacts/run-%d", calls), ExecutionEvidence: &evidence}, nil
		},
		Write: func(root string, report qualification.QualificationReport) (string, error) {
			if root != "/artifacts" || !report.Qualified {
				t.Fatalf("root = %q, report = %#v", root, report)
			}
			return "/artifacts/qualification.json", nil
		},
	})

	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || !reflect.DeepEqual(progress, []Progress{{Iteration: 1, Repeat: 2}, {Iteration: 2, Repeat: 2}}) || result.ReportPath != "/artifacts/qualification.json" || !result.Report.Qualified {
		t.Fatalf("calls = %d, progress = %#v, result = %#v", calls, progress, result)
	}
}

func workloadEvidence(seed uint64) runner.ExecutionEvidence {
	return runner.ExecutionEvidence{
		Schema: runner.ExecutionEvidenceSchema, Seed: record.Uint64String(seed), RunnerBuild: "sha256:runner",
		Toolchain:   record.Toolchain{GoVersion: "go1.26.4", BuildKey: "build", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Target:      record.Target{Kind: "go-test", Source: "./pkg", SHA256: "sha256:target", Size: 12, Argv: []string{"gomad3-target"}, BuildTags: []string{}},
		IOProfile:   deterministicio.Contract{Name: "deterministic", ImplementationSHA256: "sha256:io", InventorySHA256: "sha256:inventory"},
		Environment: []record.Environment{{Name: "GOMADSEED", Value: fmt.Sprintf("%d", seed)}, {Name: "TZ", Value: "UTC"}},
		Outcome:     runner.OutcomeEvidence{Domain: "success", Reason: "success", Termination: "exit"}, GroupGone: true,
		Stdout: record.Stream{FullSHA256: "sha256:stdout"}, Stderr: record.Stream{FullSHA256: "sha256:stderr"},
		IOTranscriptSHA256: "sha256:transcript", IOTranscriptRecords: 1, IOTranscriptComplete: true,
		SemanticCoverage: deterministicio.SemanticCoverage{Schema: deterministicio.SemanticCoverageSchema, Digest: "sha256:coverage", Probes: []string{}},
	}
}
