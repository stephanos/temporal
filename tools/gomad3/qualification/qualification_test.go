package qualification

import (
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
)

func TestBuildReportQualifiesRepeatedSuccessfulEvidence(t *testing.T) {
	evidence := successfulEvidence()
	command := []string{"gomad", "qualify", "--seed", "7", "go-test", "./pkg"}
	report, err := BuildQualificationReport(QualificationInput{
		Command: command,
		Executions: []QualificationExecution{
			{CampaignPath: "/artifacts/run-1", Evidence: evidence},
			{CampaignPath: "/artifacts/run-2", Evidence: evidence},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Schema != QualificationReportSchema || !report.Qualified || !report.Deterministic || !report.TargetSuccess || report.Seed != 7 || report.Repeat != 2 || report.EvidenceDigest == "" || report.FirstDivergence != "" {
		t.Fatalf("report = %#v", report)
	}
	if len(report.Executions) != 2 || report.Executions[0].CampaignPath != "/artifacts/run-1" || report.Executions[0].EvidenceDigest != report.EvidenceDigest {
		t.Fatalf("runs = %#v", report.Executions)
	}
	command[0] = "changed"
	if report.Command[0] != "gomad" {
		t.Fatal("report command was not copied")
	}
}

func TestBuildReportNamesFirstEvidenceDivergence(t *testing.T) {
	first := successfulEvidence()
	second := first
	second.Stdout.FullSHA256 = record.HashBytes([]byte("different"))
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify"}, Executions: []QualificationExecution{
		{CampaignPath: "/artifacts/run-1", Evidence: first},
		{CampaignPath: "/artifacts/run-2", Evidence: second},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Qualified || report.Deterministic || report.FirstDivergence != "stdout.full_sha256" || report.Executions[0].EvidenceDigest == report.Executions[1].EvidenceDigest {
		t.Fatalf("report = %#v", report)
	}
}

func TestBuildReportRecordsFailureReplay(t *testing.T) {
	evidence := successfulEvidence()
	evidence.Outcome = runner.OutcomeEvidence{Domain: "target", Reason: "nonzero_exit", Termination: "exit"}
	report, err := BuildQualificationReport(QualificationInput{
		Command: []string{"gomad", "qualify"},
		Executions: []QualificationExecution{
			{CampaignPath: "/artifacts/run-1", ArtifactPath: "/artifacts/failure-1", Evidence: evidence, Replay: &QualificationReplay{ArtifactPath: "/artifacts/failure-1", Attempted: true, Match: true}},
			{CampaignPath: "/artifacts/run-2", ArtifactPath: "/artifacts/failure-2", Evidence: evidence},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Qualified || !report.Deterministic || report.TargetSuccess || report.Executions[0].Replay == nil || !report.Executions[0].Replay.Match {
		t.Fatalf("report = %#v", report)
	}
}

func TestBuildReportRecordsPerRunSuccessfulReplay(t *testing.T) {
	evidence := successfulEvidence()
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify", "--replay-successes"}, Executions: []QualificationExecution{
		{CampaignPath: "/artifacts/run-1", ArtifactPath: "/artifacts/success-1", Evidence: evidence, Replay: &QualificationReplay{ArtifactPath: "/artifacts/success-1", Attempted: true, Match: true}},
		{CampaignPath: "/artifacts/run-2", ArtifactPath: "/artifacts/success-2", Evidence: evidence, Replay: &QualificationReplay{ArtifactPath: "/artifacts/success-2", Attempted: true, Match: true}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Schema != QualificationReportSchema || !report.Qualified || len(report.Executions) != 2 || report.Executions[0].Replay == nil || !report.Executions[0].Replay.Match || report.Executions[1].Replay == nil || !report.Executions[1].Replay.Match {
		t.Fatalf("report = %#v", report)
	}
}

func TestBuildReportRequiresExactChoiceStatusForMatchedChoiceReplay(t *testing.T) {
	evidence := successfulEvidence()
	evidence.Choices = &runner.ChoiceEvidence{Profile: "gomad3-choice-trace/v2"}
	runs := []QualificationExecution{
		{CampaignPath: "/artifacts/run-1", ArtifactPath: "/artifacts/success-1", Evidence: evidence, Replay: &QualificationReplay{ArtifactPath: "/artifacts/success-1", Attempted: true, Match: true}},
		{CampaignPath: "/artifacts/run-2", ArtifactPath: "/artifacts/success-2", Evidence: evidence, Replay: &QualificationReplay{ArtifactPath: "/artifacts/success-2", Attempted: true, Match: true}},
	}
	if _, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify", "--replay-successes"}, Executions: runs}); err == nil {
		t.Fatal("BuildQualificationReport() accepted matched choice replay without exact status")
	}
	for index := range runs {
		runs[index].Replay.ChoiceReplayStatus = ChoiceReplayExact
	}
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify", "--replay-successes"}, Executions: runs})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Qualified {
		t.Fatalf("report = %#v", report)
	}
}

func TestBuildReportRejectsMismatchedPerRunReplay(t *testing.T) {
	evidence := successfulEvidence()
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify", "--replay-successes"}, Executions: []QualificationExecution{
		{CampaignPath: "/artifacts/run-1", ArtifactPath: "/artifacts/success-1", Evidence: evidence, Replay: &QualificationReplay{ArtifactPath: "/artifacts/success-1", Attempted: true, Match: true}},
		{CampaignPath: "/artifacts/run-2", ArtifactPath: "/artifacts/success-2", Evidence: evidence, Replay: &QualificationReplay{ArtifactPath: "/artifacts/success-2", Attempted: true, Divergence: "stdout.full_sha256"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Qualified || ClassifyQualification(report) != "replay_divergence" {
		t.Fatalf("report = %#v", report)
	}
}

func TestBuildReportRejectsInvalidEvidenceSet(t *testing.T) {
	evidence := successfulEvidence()
	for _, input := range []QualificationInput{
		{Command: []string{"gomad"}, Executions: []QualificationExecution{{CampaignPath: "/one", Evidence: evidence}}},
		{Command: nil, Executions: []QualificationExecution{{CampaignPath: "/one", Evidence: evidence}, {CampaignPath: "/two", Evidence: evidence}}},
		{Command: []string{"gomad"}, Executions: []QualificationExecution{{CampaignPath: "", Evidence: evidence}, {CampaignPath: "/two", Evidence: evidence}}},
		{Command: []string{"gomad"}, Executions: []QualificationExecution{{CampaignPath: "/one", Evidence: evidence}, {CampaignPath: "/two", Evidence: withSeed(evidence, 8)}}},
		{Command: []string{"gomad"}, Executions: []QualificationExecution{{CampaignPath: "/one", Evidence: withSchema(evidence, "bad")}, {CampaignPath: "/two", Evidence: evidence}}},
	} {
		if _, err := BuildQualificationReport(input); err == nil {
			t.Fatalf("BuildQualificationReport(%#v) succeeded", input)
		}
	}
}

func TestBuildFailureRetainsFirstUnsupportedBoundary(t *testing.T) {
	report, err := BuildQualificationFailure(
		[]string{"gomad", "qualify", "go-test", "./pkg"},
		7,
		2,
		nil,
		QualificationFailure{
			Classification: "unsupported_target", Message: "example.com/target imports os/exec", Iteration: 1,
			ImportPath: "example.com/target", Capability: "imports os/exec",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.Qualified || report.Failure == nil || report.Failure.Capability != "imports os/exec" || report.Seed != 7 || report.Repeat != 2 || len(report.Executions) != 0 {
		t.Fatalf("report = %#v", report)
	}
}

func TestWriteReportRetainsCanonicalPrivateFile(t *testing.T) {
	evidence := successfulEvidence()
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify"}, Executions: []QualificationExecution{
		{CampaignPath: "/artifacts/run-1", Evidence: evidence},
		{CampaignPath: "/artifacts/run-2", Evidence: evidence},
	}})
	if err != nil {
		t.Fatal(err)
	}
	path, err := WriteQualificationReport(t.TempDir(), report)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(filepath.Dir(path)) != "v1" {
		t.Fatalf("report path = %s", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("report mode = %o", info.Mode().Perm())
	}
	opened, err := OpenQualificationReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if opened.EvidenceDigest != report.EvidenceDigest || !opened.Qualified {
		t.Fatalf("opened report = %#v", opened)
	}
}

func TestWriteRejectsInconsistentDeterministicOutcome(t *testing.T) {
	evidence := successfulEvidence()
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify"}, Executions: []QualificationExecution{
		{CampaignPath: "/artifacts/run-1", Evidence: evidence},
		{CampaignPath: "/artifacts/run-2", Evidence: evidence},
	}})
	if err != nil {
		t.Fatal(err)
	}
	report.TargetSuccess = false
	report.Qualified = false
	if _, err := WriteQualificationReport(t.TempDir(), report); err == nil {
		t.Fatal("WriteQualificationReport() accepted an outcome inconsistent with deterministic evidence")
	}
}

func successfulEvidence() runner.ExecutionEvidence {
	return runner.ExecutionEvidence{
		Schema: runner.ExecutionEvidenceSchema, Seed: 7, RunnerBuild: "sha256:runner",
		Toolchain:   record.Toolchain{GoVersion: "go1.26.4", BuildKey: "build", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Target:      record.Target{Kind: "go-test", Source: "./pkg", SHA256: "sha256:target", Size: 12, Argv: []string{"gomad3-target"}, BuildTags: []string{"gomad_fixture"}, Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}},
		IOProfile:   deterministicio.Contract{Name: "deterministic", ImplementationSHA256: "sha256:io", InventorySHA256: "sha256:inventory"},
		Environment: []record.Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "TZ", Value: "UTC"}},
		Outcome:     runner.OutcomeEvidence{Domain: "success", Reason: "success", Termination: "exit"}, GroupGone: true,
		Stdout: record.Stream{FullSHA256: "sha256:stdout"}, Stderr: record.Stream{FullSHA256: "sha256:stderr"},
		IOTranscriptSHA256: "sha256:transcript", IOTranscriptRecords: 1, IOTranscriptComplete: true,
		SemanticCoverage: deterministicio.SemanticCoverage{Schema: deterministicio.SemanticCoverageSchema, Digest: "sha256:coverage", Probes: []string{"stdlib.os.openfile"}},
	}
}

func withSeed(evidence runner.ExecutionEvidence, seed record.Uint64String) runner.ExecutionEvidence {
	evidence.Seed = seed
	return evidence
}

func withSchema(evidence runner.ExecutionEvidence, schema string) runner.ExecutionEvidence {
	evidence.Schema = schema
	return evidence
}
