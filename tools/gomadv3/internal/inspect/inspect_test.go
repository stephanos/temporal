package inspect

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomadv3/internal/artifact"
	"go.temporal.io/server/tools/gomadv3/internal/record"
)

func TestOpenReportsArtifactIdentityAndReplay(t *testing.T) {
	published := publishInspectArtifact(t)
	report, err := Open(published.Path)
	if err != nil {
		t.Fatal(err)
	}
	if report.Kind != "artifact" || report.Artifact == nil || report.Batch != nil {
		t.Fatalf("report = %#v", report)
	}
	observed := report.Artifact
	if observed.Seed != 7 || observed.Target.Source != "./target" || observed.Outcome.Reason != "nonzero_exit" || observed.Transcript == nil || observed.Transcript.Records != 3 {
		t.Fatalf("artifact report = %#v", observed)
	}
	if !observed.Stdout.Truncated || !strings.Contains(observed.ReplayCommand, published.Path) {
		t.Fatalf("artifact details = %#v", observed)
	}
}

func TestOpenReportsValidatedBatchRuns(t *testing.T) {
	journal, err := artifact.NewBatchJournal(context.Background(), artifact.BatchConfig{
		Root: t.TempDir(), RunID: "run-inspect", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	if err := journal.StartRuns(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendRun(artifact.RunRecord{
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit", ElapsedNanos: 5,
	}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Publish(artifact.BatchSummary{Attempted: 1, Succeeded: 1, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	report, err := Open(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if report.Kind != "batch" || report.Batch == nil || report.Artifact != nil || report.Batch.RunID != "run-inspect" || len(report.Batch.Runs) != 1 {
		t.Fatalf("report = %#v", report)
	}
}

func TestOpenReportsAndValidatesRetainedSuccessfulRuns(t *testing.T) {
	journal, err := artifact.NewBatchJournal(context.Background(), artifact.BatchConfig{
		Root: t.TempDir(), RunID: "run-inspect-success", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	retained := publishInspectArtifactAt(t, journal.SuccessesPath(), "run-inspect-success", true)
	relative, err := filepath.Rel(journal.Path(), retained.Path)
	if err != nil {
		t.Fatal(err)
	}
	bytes := record.Uint64String(retained.StoredBytes)
	if err := journal.StartRuns(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendRun(artifact.RunRecord{
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit", ElapsedNanos: 5,
		SuccessArtifact: &relative, SuccessArtifactBytes: &bytes, SemanticProbes: []string{"stdlib.os.openfile"}, NovelSemanticProbes: []string{"stdlib.os.openfile"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Publish(artifact.BatchSummary{Attempted: 1, Succeeded: 1, RetainedSuccesses: 1, RetainedSuccessBytes: retained.StoredBytes, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	report, err := Open(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if report.Batch == nil || report.Batch.RetainedSuccesses != 1 || report.Batch.RetainedSuccessBytes != retained.StoredBytes || len(report.Batch.SuccessArtifacts) != 1 || report.Batch.SuccessArtifacts[0].Path != retained.Path {
		t.Fatalf("batch report = %#v", report.Batch)
	}
}

func publishInspectArtifact(t *testing.T) artifact.Artifact {
	t.Helper()
	return publishInspectArtifactAt(t, t.TempDir(), "batch-inspect", false)
}

func publishInspectArtifactAt(t *testing.T, root, batchID string, success bool) artifact.Artifact {
	t.Helper()
	targetPath := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(targetPath, []byte("target"), 0o700); err != nil {
		t.Fatal(err)
	}
	world, payloads := record.NoneWorld()
	exitCode := record.Uint64String(2)
	artifactKind := record.ArtifactTargetFailure
	domain := "target"
	reason := "nonzero_exit"
	if success {
		exitCode = 0
		artifactKind = record.ArtifactSuccess
		domain = "success"
		reason = "success"
	}
	transcript := []byte(strings.Repeat("x", 128*3))
	manifest := record.Manifest{
		SchemaVersion: record.SchemaVersion, ArtifactKind: artifactKind, CreatedAt: "2026-08-12T12:00:00Z", BatchID: batchID, SelectionOrdinal: 0, Seed: 7, ReplayMode: record.ReplayExact,
		Runner:      record.Runner{RecordContract: "gomadv3.run-record/v1", RunnerBuild: "sha256:runner", HostOS: "darwin", HostArch: "arm64"},
		Toolchain:   record.Toolchain{GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Target:      record.Target{Kind: "go-test", Source: "./target", SHA256: record.HashBytes([]byte("target")), Size: 6, Argv: []string{"gomadv3-target"}, BuildTags: []string{"test_dep"}, BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"}},
		IOProfile:   record.IOProfile{Name: "gomadv3-deterministic/v1", ImplementationSHA256: record.HashBytes([]byte("implementation")), Inventory: "{}", InventorySHA256: record.HashBytes([]byte("{}")), Transcript: &record.IOTranscript{Schema: "gomadv3.io-transcript/v1", SHA256: record.HashBytes(transcript), Bytes: record.Uint64String(len(transcript)), Records: 3}},
		Environment: []record.Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "GOMADV3_IO_PROFILE", Value: "gomadv3-deterministic/v1"}, {Name: "TZ", Value: "UTC"}}, Limits: record.Limits{RunTimeoutNanos: 1, OverallTimeoutNanos: 2, OutputBytes: 4, WorldTransitionBytes: 64, IOTranscriptBytes: 1 << 20}, World: world,
		Outcome: record.Outcome{Domain: domain, Reason: reason, Termination: "exit", ExitCode: &exitCode},
		Streams: record.Streams{
			Stdout: record.Stream{FullSHA256: record.HashBytes([]byte("long output")), TotalBytes: 11, RetainedBytes: 4, DiscardedBytes: 7, Truncated: true},
			Stderr: record.Stream{FullSHA256: record.HashBytes(nil), TotalBytes: 0, RetainedBytes: 0},
		},
		Host: record.Host{StartedAt: "2026-08-12T12:00:00Z", FinishedAt: "2026-08-12T12:00:01Z", ElapsedNanos: 1},
	}
	published, err := (artifact.Store{Root: root}).Publish(artifact.Input{
		Manifest: manifest, TargetPath: targetPath, Stdout: []byte("long"), Stderr: nil, IOTranscript: transcript, World: payloads,
	})
	if err != nil {
		t.Fatal(err)
	}
	return published
}
