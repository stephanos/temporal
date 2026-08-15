package inspect

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomadv3/internal/artifact"
	"go.temporal.io/server/tools/gomadv3/internal/choicewire"
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

func TestOpenWithOptionsProjectsValidatedChoiceTrace(t *testing.T) {
	published := publishInspectArtifact(t)
	report, err := OpenWithOptions(published.Path, Options{Choices: true})
	if err != nil {
		t.Fatal(err)
	}
	choices := report.Artifact.Choices
	if choices == nil || choices.Records != 1 || choices.BranchingRecords != 1 || choices.Runnable != 1 || choices.SelectPoll != 0 || choices.SelectResult != 0 || len(choices.Sites) != 1 {
		t.Fatalf("choice inspection = %#v", choices)
	}
	if choices.Sites[0].Kind != "runnable" || choices.Sites[0].MaximumAlternatives != 2 || choices.Sites[0].Fingerprint == "" {
		t.Fatalf("choice site = %#v", choices.Sites)
	}
}

func TestOpenWithOptionsRejectsBatch(t *testing.T) {
	batch := writeInspectBatchForChoiceTest(t)
	if _, err := OpenWithOptions(batch, Options{Choices: true}); err == nil || !strings.Contains(err.Error(), "traced artifact") {
		t.Fatalf("batch choice inspection error = %v", err)
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
	choiceTrace := retained.Manifest.ChoiceProfile.Trace
	if err := journal.StartRuns(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendRun(artifact.RunRecord{
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit", ElapsedNanos: 5,
		SuccessArtifact: &relative, SuccessArtifactBytes: &bytes, SemanticProbes: []string{"stdlib.os.openfile"}, NovelSemanticProbes: []string{"stdlib.os.openfile"},
		ChoiceTraceSHA256: &choiceTrace.SHA256, ChoiceTraceRecords: &choiceTrace.Records, ChoiceTraceBranchingRecords: &choiceTrace.BranchingRecords,
		ChoiceTraceTerminalState: &choiceTrace.TerminalState, ChoiceTapeSHA256: &choiceTrace.TapeSHA256, ChoiceDecisions: &choiceTrace.Decisions,
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
	first := sha256.Sum256([]byte("first choice alternative"))
	second := sha256.Sum256([]byte("second choice alternative"))
	decision, err := choicewire.CanonicalDecision(0, choicewire.KindRunnable, 24, false, [][sha256.Size]byte{first, second}, second, 0)
	if err != nil {
		t.Fatal(err)
	}
	choiceRecord, err := choicewire.EncodeRecord(decision.Record())
	if err != nil {
		t.Fatal(err)
	}
	choicePayload := choiceRecord[:]
	choiceLimit := uint64(choicewire.HeaderBytes + choicewire.RecordBytes)
	choiceImplementation, err := choicewire.ImplementationIdentity(strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	choiceDigest := sha256.Sum256(choicePayload)
	choiceTrace, err := choicewire.DecodeStoredTrace(choicewire.Profile, choicePayload, choicewire.TerminalMetadata{State: choicewire.TerminalComplete, Limit: choiceLimit, Records: 1, SHA256: choiceDigest})
	if err != nil {
		t.Fatal(err)
	}
	targetSHA256, err := record.HashBytes([]byte("target")).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	choiceTape, err := choicewire.ProjectDecisionTape(choiceTrace, choicewire.ExecutionIdentity{
		TargetSHA256: targetSHA256, ToolchainBuildKey: strings.Repeat("a", 64),
		GOOS: "darwin", GOARCH: "arm64", ImplementationSHA256: choiceImplementation,
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest := record.Manifest{
		SchemaVersion: record.SchemaVersion, ArtifactKind: artifactKind, CreatedAt: "2026-08-12T12:00:00Z", BatchID: batchID, SelectionOrdinal: 0, Seed: 7, ReplayMode: record.ReplayExact,
		Runner:        record.Runner{RecordContract: record.RecordContract, RunnerBuild: "sha256:runner", HostOS: "darwin", HostArch: "arm64"},
		Toolchain:     record.Toolchain{GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Target:        record.Target{Kind: "go-test", Source: "./target", SHA256: record.HashBytes([]byte("target")), Size: 6, Argv: []string{"gomadv3-target"}, BuildTags: []string{"gomad_fixture"}, Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}, BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"}},
		IOProfile:     record.IOProfile{Name: "gomadv3-deterministic/v1", ImplementationSHA256: record.HashBytes([]byte("implementation")), Inventory: "{}", InventorySHA256: record.HashBytes([]byte("{}")), Transcript: &record.IOTranscript{Schema: "gomadv3.io-transcript/v1", SHA256: record.HashBytes(transcript), Bytes: record.Uint64String(len(transcript)), Records: 3}},
		ChoiceProfile: &record.ChoiceProfile{Name: choicewire.Profile, ImplementationSHA256: record.SHA256FromSum(choiceImplementation), Trace: record.ChoiceTrace{Schema: "gomadv3.choice-trace/v2", SHA256: record.SHA256FromSum(choiceDigest), Bytes: record.Uint64String(len(choicePayload)), Records: 1, BranchingRecords: 1, TerminalState: "complete", Limit: record.Uint64String(choiceLimit), TapeSHA256: record.SHA256FromSum(choiceTape.SHA256), Decisions: 1}},
		Environment:   []record.Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "GOMADV3_CHOICE_PROFILE", Value: choicewire.Profile}, {Name: "GOMADV3_IO_PROFILE", Value: "gomadv3-deterministic/v1"}, {Name: "TZ", Value: "UTC"}}, Limits: record.Limits{RunTimeoutNanos: 1, OverallTimeoutNanos: 2, OutputBytes: 4, WorldTransitionBytes: 64, IOTranscriptBytes: 1 << 20, ChoiceTraceBytes: record.Uint64String(choiceLimit)}, World: world,
		Outcome: record.Outcome{Domain: domain, Reason: reason, Termination: "exit", ExitCode: &exitCode},
		Streams: record.Streams{
			Stdout: record.Stream{FullSHA256: record.HashBytes([]byte("long output")), TotalBytes: 11, RetainedBytes: 4, DiscardedBytes: 7, Truncated: true},
			Stderr: record.Stream{FullSHA256: record.HashBytes(nil), TotalBytes: 0, RetainedBytes: 0},
		},
		Host: record.Host{StartedAt: "2026-08-12T12:00:00Z", FinishedAt: "2026-08-12T12:00:01Z", ElapsedNanos: 1},
	}
	published, err := (artifact.Store{Root: root}).Publish(artifact.Input{
		Manifest: manifest, TargetPath: targetPath, Stdout: []byte("long"), Stderr: nil, IOTranscript: transcript, ChoiceTrace: choicePayload, World: payloads,
	})
	if err != nil {
		t.Fatal(err)
	}
	return published
}

func writeInspectBatchForChoiceTest(t *testing.T) string {
	t.Helper()
	journal, err := artifact.NewBatchJournal(context.Background(), artifact.BatchConfig{Root: t.TempDir(), RunID: "run-choice-batch", Selection: "7", SelectionCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = journal.Close() })
	if err := journal.StartRuns(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendRun(artifact.RunRecord{SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit"}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Publish(artifact.BatchSummary{Attempted: 1, Succeeded: 1, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	return journal.Path()
}
