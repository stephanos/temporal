package qualification

import (
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner"
)

func TestBuildReportQualifiesRepeatedSuccessfulEvidence(t *testing.T) {
	evidence := successfulEvidence()
	command := []string{"gomad", "qualify", "--seed", "7", "go-test", "./pkg"}
	report, err := BuildQualificationReport(QualificationInput{
		Command: command,
		Runs: []QualificationExecution{
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
	if len(report.Runs) != 2 || report.Runs[0].CampaignPath != "/artifacts/run-1" || report.Runs[0].EvidenceDigest != report.EvidenceDigest {
		t.Fatalf("runs = %#v", report.Runs)
	}
	command[0] = "changed"
	if report.Command[0] != "gomad" {
		t.Fatal("report command was not copied")
	}
}

func TestBuildReportNamesFirstEvidenceDivergence(t *testing.T) {
	first := successfulEvidence()
	second := first
	second.Stdout.FullSHA256 = evidence.HashBytes([]byte("different"))
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify"}, Runs: []QualificationExecution{
		{CampaignPath: "/artifacts/run-1", Evidence: first},
		{CampaignPath: "/artifacts/run-2", Evidence: second},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Qualified || report.Deterministic || report.FirstDivergence != "stdout.full_sha256" || report.Runs[0].EvidenceDigest == report.Runs[1].EvidenceDigest {
		t.Fatalf("report = %#v", report)
	}
}

func TestBuildReportRecordsFailureReplay(t *testing.T) {
	evidence := successfulEvidence()
	evidence.Outcome = runner.OutcomeEvidence{Domain: "target", Reason: "nonzero_exit", Termination: "exit"}
	report, err := BuildQualificationReport(QualificationInput{
		Command: []string{"gomad", "qualify"},
		Runs: []QualificationExecution{
			{CampaignPath: "/artifacts/run-1", ArtifactPath: "/artifacts/failure-1", Evidence: evidence},
			{CampaignPath: "/artifacts/run-2", ArtifactPath: "/artifacts/failure-2", Evidence: evidence},
		},
		Replay: &QualificationReplay{ArtifactPath: "/artifacts/failure-1", Attempted: true, Match: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Qualified || !report.Deterministic || report.TargetSuccess || report.Runs[0].Replay == nil || !report.Runs[0].Replay.Match {
		t.Fatalf("report = %#v", report)
	}
}

func TestBuildReportRecordsPerRunSuccessfulReplay(t *testing.T) {
	evidence := successfulEvidence()
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify", "--replay-successes"}, Runs: []QualificationExecution{
		{CampaignPath: "/artifacts/run-1", ArtifactPath: "/artifacts/success-1", Evidence: evidence, Replay: &QualificationReplay{ArtifactPath: "/artifacts/success-1", Attempted: true, Match: true}},
		{CampaignPath: "/artifacts/run-2", ArtifactPath: "/artifacts/success-2", Evidence: evidence, Replay: &QualificationReplay{ArtifactPath: "/artifacts/success-2", Attempted: true, Match: true}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Schema != "gomadv3.qualification/v4" || !report.Qualified || len(report.Runs) != 2 || report.Runs[0].Replay == nil || !report.Runs[0].Replay.Match || report.Runs[1].Replay == nil || !report.Runs[1].Replay.Match {
		t.Fatalf("report = %#v", report)
	}
}

func TestBuildReportRequiresExactChoiceStatusForMatchedChoiceReplay(t *testing.T) {
	evidence := successfulEvidence()
	evidence.Choices = &runner.ChoiceEvidence{Profile: "gomadv3-choice-trace/v2"}
	runs := []QualificationExecution{
		{CampaignPath: "/artifacts/run-1", ArtifactPath: "/artifacts/success-1", Evidence: evidence, Replay: &QualificationReplay{ArtifactPath: "/artifacts/success-1", Attempted: true, Match: true}},
		{CampaignPath: "/artifacts/run-2", ArtifactPath: "/artifacts/success-2", Evidence: evidence, Replay: &QualificationReplay{ArtifactPath: "/artifacts/success-2", Attempted: true, Match: true}},
	}
	if _, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify", "--replay-successes"}, Runs: runs}); err == nil {
		t.Fatal("BuildQualificationReport() accepted matched choice replay without exact status")
	}
	for index := range runs {
		runs[index].Replay.ChoiceReplayStatus = ChoiceReplayExact
	}
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify", "--replay-successes"}, Runs: runs})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Qualified {
		t.Fatalf("report = %#v", report)
	}
}

func TestBuildReportRejectsMismatchedPerRunReplay(t *testing.T) {
	evidence := successfulEvidence()
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify", "--replay-successes"}, Runs: []QualificationExecution{
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
		{Command: []string{"gomad"}, Runs: []QualificationExecution{{CampaignPath: "/one", Evidence: evidence}}},
		{Command: nil, Runs: []QualificationExecution{{CampaignPath: "/one", Evidence: evidence}, {CampaignPath: "/two", Evidence: evidence}}},
		{Command: []string{"gomad"}, Runs: []QualificationExecution{{CampaignPath: "", Evidence: evidence}, {CampaignPath: "/two", Evidence: evidence}}},
		{Command: []string{"gomad"}, Runs: []QualificationExecution{{CampaignPath: "/one", Evidence: evidence}, {CampaignPath: "/two", Evidence: withSeed(evidence, 8)}}},
		{Command: []string{"gomad"}, Runs: []QualificationExecution{{CampaignPath: "/one", Evidence: withSchema(evidence, "bad")}, {CampaignPath: "/two", Evidence: evidence}}},
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
	if report.Qualified || report.Failure == nil || report.Failure.Capability != "imports os/exec" || report.Seed != 7 || report.Repeat != 2 || len(report.Runs) != 0 {
		t.Fatalf("report = %#v", report)
	}
}

func TestWriteReportRetainsCanonicalPrivateFile(t *testing.T) {
	evidence := successfulEvidence()
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify"}, Runs: []QualificationExecution{
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
	if filepath.Base(filepath.Dir(path)) != "v4" {
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

func TestOpenNormalizesLegacyFailureReplay(t *testing.T) {
	runRecord := successfulEvidence()
	runRecord.Outcome = runner.OutcomeEvidence{Domain: "target", Reason: "nonzero_exit", Termination: "exit"}
	digest, err := evidenceDigest(runRecord)
	if err != nil {
		t.Fatal(err)
	}
	legacy := legacyReport{
		Schema: LegacyQualificationReportSchema, Deterministic: true, Seed: 7, Repeat: 2,
		Command: []string{"gomad", "qualify"}, EvidenceDigest: digest, Evidence: &runRecord,
		Runs: []legacyRunReport{
			{CampaignPath: "/artifacts/run-1", ArtifactPath: "/artifacts/failure-1", EvidenceDigest: digest},
			{CampaignPath: "/artifacts/run-2", ArtifactPath: "/artifacts/failure-2", EvidenceDigest: digest},
		},
		Replay: &QualificationReplay{ArtifactPath: "/artifacts/failure-1", Attempted: true, Match: true},
	}
	encoded, err := evidence.CanonicalJSON(legacy)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "qualification-v1.json")
	if err := os.WriteFile(path, append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	opened, err := OpenQualificationReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Schema != QualificationReportSchema || opened.Qualified || opened.TargetSuccess || opened.Runs[0].Replay == nil || !opened.Runs[0].Replay.Match || opened.Runs[1].Replay != nil {
		t.Fatalf("opened report = %#v", opened)
	}
}

func TestOpenPreviousReportUsesLegacyEvidenceDigest(t *testing.T) {
	runRecord := successfulEvidence()
	runRecord.Schema = runner.LegacyExecutionEvidenceSchema
	encodedEvidence, err := evidence.CanonicalJSON(runRecord)
	if err != nil {
		t.Fatal(err)
	}
	digest := evidence.DomainHash(LegacyExecutionEvidenceDigestDomain, encodedEvidence)
	report := QualificationReport{
		Schema: PreviousQualificationReportSchema, Qualified: true, Deterministic: true, TargetSuccess: true,
		Seed: runRecord.Seed, Repeat: 2, Command: []string{"gomad", "qualify"}, EvidenceDigest: digest, Evidence: &runRecord,
		Runs: []QualificationExecutionReport{{CampaignPath: "/artifacts/run-1", EvidenceDigest: digest}, {CampaignPath: "/artifacts/run-2", EvidenceDigest: digest}},
	}
	encoded, err := evidence.CanonicalJSON(report)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "qualification-v2.json")
	if err := os.WriteFile(path, append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	opened, err := OpenQualificationReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Schema != QualificationReportSchema || opened.EvidenceDigest != digest || opened.Evidence.Schema != runner.LegacyExecutionEvidenceSchema {
		t.Fatalf("normalized previous report = %#v", opened)
	}
}

func TestWriteRejectsInconsistentDeterministicOutcome(t *testing.T) {
	evidence := successfulEvidence()
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify"}, Runs: []QualificationExecution{
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
		Toolchain:   evidence.Toolchain{GoVersion: "go1.26.4", BuildKey: "build", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Target:      evidence.Target{Kind: "go-test", Source: "./pkg", SHA256: "sha256:target", Size: 12, Argv: []string{"gomadv3-target"}, BuildTags: []string{"gomad_fixture"}, Adapters: []evidence.TargetAdapter{}, Compatibility: []evidence.CompatibilityPack{}},
		IOProfile:   runner.IOProfileEvidence{Name: "deterministic", ImplementationSHA256: "sha256:io", InventorySHA256: "sha256:inventory"},
		Environment: []evidence.Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "TZ", Value: "UTC"}},
		Outcome:     runner.OutcomeEvidence{Domain: "success", Reason: "success", Termination: "exit"}, GroupGone: true,
		Stdout: evidence.Stream{FullSHA256: "sha256:stdout"}, Stderr: evidence.Stream{FullSHA256: "sha256:stderr"},
		IOTranscriptSHA256: "sha256:transcript", IOTranscriptRecords: 1, IOTranscriptComplete: true,
		SemanticCoverage: deterministicio.SemanticCoverage{Schema: deterministicio.SemanticCoverageSchema, Digest: "sha256:coverage", Probes: []string{"stdlib.os.openfile"}},
	}
}

func withSeed(evidence runner.ExecutionEvidence, seed evidence.Uint64String) runner.ExecutionEvidence {
	evidence.Seed = seed
	return evidence
}

func withSchema(evidence runner.ExecutionEvidence, schema string) runner.ExecutionEvidence {
	evidence.Schema = schema
	return evidence
}
