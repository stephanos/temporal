package qualify

import (
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomadv3/internal/ioprofile"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/runner"
)

func TestBuildReportQualifiesRepeatedSuccessfulEvidence(t *testing.T) {
	evidence := successfulEvidence()
	command := []string{"gomad", "qualify", "--seed", "7", "go-test", "./pkg"}
	report, err := Build(Input{
		Command: command,
		Runs: []Run{
			{BatchPath: "/artifacts/run-1", Evidence: evidence},
			{BatchPath: "/artifacts/run-2", Evidence: evidence},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Schema != ReportSchema || !report.Qualified || !report.Deterministic || !report.TargetSuccess || report.Seed != 7 || report.Repeat != 2 || report.EvidenceDigest == "" || report.FirstDivergence != "" {
		t.Fatalf("report = %#v", report)
	}
	if len(report.Runs) != 2 || report.Runs[0].BatchPath != "/artifacts/run-1" || report.Runs[0].EvidenceDigest != report.EvidenceDigest {
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
	second.Stdout.FullSHA256 = record.HashBytes([]byte("different"))
	report, err := Build(Input{Command: []string{"gomad", "qualify"}, Runs: []Run{
		{BatchPath: "/artifacts/run-1", Evidence: first},
		{BatchPath: "/artifacts/run-2", Evidence: second},
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
	report, err := Build(Input{
		Command: []string{"gomad", "qualify"},
		Runs: []Run{
			{BatchPath: "/artifacts/run-1", ArtifactPath: "/artifacts/failure-1", Evidence: evidence},
			{BatchPath: "/artifacts/run-2", ArtifactPath: "/artifacts/failure-2", Evidence: evidence},
		},
		Replay: &Replay{ArtifactPath: "/artifacts/failure-1", Attempted: true, Match: true},
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
	report, err := Build(Input{Command: []string{"gomad", "qualify", "--replay-successes"}, Runs: []Run{
		{BatchPath: "/artifacts/run-1", ArtifactPath: "/artifacts/success-1", Evidence: evidence, Replay: &Replay{ArtifactPath: "/artifacts/success-1", Attempted: true, Match: true}},
		{BatchPath: "/artifacts/run-2", ArtifactPath: "/artifacts/success-2", Evidence: evidence, Replay: &Replay{ArtifactPath: "/artifacts/success-2", Attempted: true, Match: true}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Schema != "gomadv3.qualification/v2" || !report.Qualified || len(report.Runs) != 2 || report.Runs[0].Replay == nil || !report.Runs[0].Replay.Match || report.Runs[1].Replay == nil || !report.Runs[1].Replay.Match {
		t.Fatalf("report = %#v", report)
	}
}

func TestBuildReportRejectsMismatchedPerRunReplay(t *testing.T) {
	evidence := successfulEvidence()
	report, err := Build(Input{Command: []string{"gomad", "qualify", "--replay-successes"}, Runs: []Run{
		{BatchPath: "/artifacts/run-1", ArtifactPath: "/artifacts/success-1", Evidence: evidence, Replay: &Replay{ArtifactPath: "/artifacts/success-1", Attempted: true, Match: true}},
		{BatchPath: "/artifacts/run-2", ArtifactPath: "/artifacts/success-2", Evidence: evidence, Replay: &Replay{ArtifactPath: "/artifacts/success-2", Attempted: true, Divergence: "stdout.full_sha256"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Qualified || Classify(report) != "replay_divergence" {
		t.Fatalf("report = %#v", report)
	}
}

func TestBuildReportRejectsInvalidEvidenceSet(t *testing.T) {
	evidence := successfulEvidence()
	for _, input := range []Input{
		{Command: []string{"gomad"}, Runs: []Run{{BatchPath: "/one", Evidence: evidence}}},
		{Command: nil, Runs: []Run{{BatchPath: "/one", Evidence: evidence}, {BatchPath: "/two", Evidence: evidence}}},
		{Command: []string{"gomad"}, Runs: []Run{{BatchPath: "", Evidence: evidence}, {BatchPath: "/two", Evidence: evidence}}},
		{Command: []string{"gomad"}, Runs: []Run{{BatchPath: "/one", Evidence: evidence}, {BatchPath: "/two", Evidence: withSeed(evidence, 8)}}},
		{Command: []string{"gomad"}, Runs: []Run{{BatchPath: "/one", Evidence: withSchema(evidence, "bad")}, {BatchPath: "/two", Evidence: evidence}}},
	} {
		if _, err := Build(input); err == nil {
			t.Fatalf("Build(%#v) succeeded", input)
		}
	}
}

func TestBuildFailureRetainsFirstUnsupportedBoundary(t *testing.T) {
	report, err := BuildFailure(
		[]string{"gomad", "qualify", "go-test", "./pkg"},
		7,
		2,
		nil,
		Failure{
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
	report, err := Build(Input{Command: []string{"gomad", "qualify"}, Runs: []Run{
		{BatchPath: "/artifacts/run-1", Evidence: evidence},
		{BatchPath: "/artifacts/run-2", Evidence: evidence},
	}})
	if err != nil {
		t.Fatal(err)
	}
	path, err := Write(t.TempDir(), report)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(filepath.Dir(path)) != "v2" {
		t.Fatalf("report path = %s", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("report mode = %o", info.Mode().Perm())
	}
	opened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if opened.EvidenceDigest != report.EvidenceDigest || !opened.Qualified {
		t.Fatalf("opened report = %#v", opened)
	}
}

func TestOpenNormalizesLegacyFailureReplay(t *testing.T) {
	evidence := successfulEvidence()
	evidence.Outcome = runner.OutcomeEvidence{Domain: "target", Reason: "nonzero_exit", Termination: "exit"}
	digest, err := evidenceDigest(evidence)
	if err != nil {
		t.Fatal(err)
	}
	legacy := legacyReport{
		Schema: LegacyReportSchema, Deterministic: true, Seed: 7, Repeat: 2,
		Command: []string{"gomad", "qualify"}, EvidenceDigest: digest, Evidence: &evidence,
		Runs: []legacyRunReport{
			{BatchPath: "/artifacts/run-1", ArtifactPath: "/artifacts/failure-1", EvidenceDigest: digest},
			{BatchPath: "/artifacts/run-2", ArtifactPath: "/artifacts/failure-2", EvidenceDigest: digest},
		},
		Replay: &Replay{ArtifactPath: "/artifacts/failure-1", Attempted: true, Match: true},
	}
	encoded, err := record.CanonicalJSON(legacy)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "qualification-v1.json")
	if err := os.WriteFile(path, append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	opened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Schema != ReportSchema || opened.Qualified || opened.TargetSuccess || opened.Runs[0].Replay == nil || !opened.Runs[0].Replay.Match || opened.Runs[1].Replay != nil {
		t.Fatalf("opened report = %#v", opened)
	}
}

func TestWriteRejectsInconsistentDeterministicOutcome(t *testing.T) {
	evidence := successfulEvidence()
	report, err := Build(Input{Command: []string{"gomad", "qualify"}, Runs: []Run{
		{BatchPath: "/artifacts/run-1", Evidence: evidence},
		{BatchPath: "/artifacts/run-2", Evidence: evidence},
	}})
	if err != nil {
		t.Fatal(err)
	}
	report.TargetSuccess = false
	report.Qualified = false
	if _, err := Write(t.TempDir(), report); err == nil {
		t.Fatal("Write() accepted an outcome inconsistent with deterministic evidence")
	}
}

func successfulEvidence() runner.RunEvidence {
	return runner.RunEvidence{
		Schema: runner.RunEvidenceSchema, Seed: 7, RunnerBuild: "sha256:runner",
		Toolchain:   record.Toolchain{GoVersion: "go1.26.4", BuildKey: "build", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Target:      record.Target{Kind: "go-test", Source: "./pkg", SHA256: "sha256:target", Size: 12, Argv: []string{"gomadv3-target"}, BuildTags: []string{"gomad_fixture"}, Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}},
		IOProfile:   runner.IOProfileEvidence{Name: "deterministic", ImplementationSHA256: "sha256:io", InventorySHA256: "sha256:inventory"},
		Environment: []record.Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "TZ", Value: "UTC"}},
		Outcome:     runner.OutcomeEvidence{Domain: "success", Reason: "success", Termination: "exit"}, GroupGone: true,
		Stdout: record.Stream{FullSHA256: "sha256:stdout"}, Stderr: record.Stream{FullSHA256: "sha256:stderr"},
		IOTranscriptSHA256: "sha256:transcript", IOTranscriptRecords: 1, IOTranscriptComplete: true,
		SemanticCoverage: ioprofile.SemanticCoverage{Schema: ioprofile.SemanticCoverageSchema, Digest: "sha256:coverage", Probes: []string{"stdlib.os.openfile"}},
	}
}

func withSeed(evidence runner.RunEvidence, seed record.Uint64String) runner.RunEvidence {
	evidence.Seed = seed
	return evidence
}

func withSchema(evidence runner.RunEvidence, schema string) runner.RunEvidence {
	evidence.Schema = schema
	return evidence
}
