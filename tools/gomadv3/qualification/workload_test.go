package qualification

import (
	"context"
	"fmt"
	"testing"

	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner"
)

func TestRunWorkloadRepeatsOneCampaignAndPublishesTheClaim(t *testing.T) {
	var calls int
	var progress []WorkloadProgress
	result, err := RunWorkload(context.Background(), WorkloadSpec{
		Command: []string{"gomad", "qualify"}, Seed: 7, Repeat: 2, ArtifactRoot: "/artifacts",
		Campaign: runner.CampaignSpec{Seeds: "7"},
		Progress: func(event WorkloadProgress) error {
			progress = append(progress, event)
			return nil
		},
		Explore: func(context.Context, runner.CampaignSpec) (runner.CampaignResult, error) {
			calls++
			evidence := workloadEvidence(7)
			return runner.CampaignResult{CampaignPath: fmt.Sprintf("/artifacts/run-%d", calls), ExecutionEvidence: &evidence}, nil
		},
		Write: func(root string, report QualificationReport) (string, error) {
			requireTestEqual(t, "/artifacts", root)
			requireTestEqual(t, true, report.Qualified)
			return "/artifacts/qualification.json", nil
		},
	})

	requireTestNoError(t, err)
	requireTestEqual(t, 2, calls)
	requireTestEqual(t, []WorkloadProgress{{Iteration: 1, Repeat: 2}, {Iteration: 2, Repeat: 2}}, progress)
	requireTestEqual(t, "/artifacts/qualification.json", result.ReportPath)
	requireTestEqual(t, true, result.Report.Qualified)
}

func workloadEvidence(seed uint64) runner.ExecutionEvidence {
	return runner.ExecutionEvidence{
		Schema: runner.ExecutionEvidenceSchema, Seed: evidence.Uint64String(seed), RunnerBuild: "sha256:runner",
		Toolchain:   evidence.Toolchain{GoVersion: "go1.26.4", BuildKey: "build", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Target:      evidence.Target{Kind: "go-test", Source: "./pkg", SHA256: "sha256:target", Size: 12, Argv: []string{"gomadv3-target"}, BuildTags: []string{}},
		IOProfile:   runner.IOProfileEvidence{Name: "deterministic", ImplementationSHA256: "sha256:io", InventorySHA256: "sha256:inventory"},
		Environment: []evidence.Environment{{Name: "GOMADSEED", Value: fmt.Sprintf("%d", seed)}, {Name: "TZ", Value: "UTC"}},
		Outcome:     runner.OutcomeEvidence{Domain: "success", Reason: "success", Termination: "exit"}, GroupGone: true,
		Stdout: evidence.Stream{FullSHA256: "sha256:stdout"}, Stderr: evidence.Stream{FullSHA256: "sha256:stderr"},
		IOTranscriptSHA256: "sha256:transcript", IOTranscriptRecords: 1, IOTranscriptComplete: true,
		SemanticCoverage: deterministicio.SemanticCoverage{Schema: deterministicio.SemanticCoverageSchema, Digest: "sha256:coverage", Probes: []string{}},
	}
}
