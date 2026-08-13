package artifact

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/record"
)

func TestBatchJournalPublishesTheCanonicalBatchLifecycle(t *testing.T) {
	journal, err := NewBatchJournal(context.Background(), BatchConfig{
		Root: t.TempDir(), RunID: "run-fixed", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	if err := journal.BeginPreparation(); err != nil {
		t.Fatal(err)
	}
	if err := journal.CompletePreparation(); err != nil {
		t.Fatal(err)
	}
	if err := journal.StartRuns(); err != nil {
		t.Fatal(err)
	}
	run, err := journal.BeginRun(0, 7)
	if err != nil {
		t.Fatal(err)
	}
	partialPath := filepath.Join(run.Path(), "partial.json")
	assertFileContents(t, partialPath, `{"schema_version":2,"seed":"7","selection_ordinal":"0","state":"staging"}`)
	if err := run.Transition(RunStarting); err != nil {
		t.Fatal(err)
	}
	stdout, err := run.CreateOutput("stdout")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stdout.Write([]byte("output")); err != nil {
		t.Fatal(err)
	}
	if err := stdout.Close(); err != nil {
		t.Fatal(err)
	}
	for _, state := range []RunState{RunExited, RunCaptured, RunClassified} {
		if err := run.Transition(state); err != nil {
			t.Fatal(err)
		}
	}
	if err := journal.AppendRun(RunRecord{
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit", ElapsedNanos: 5,
	}); err != nil {
		t.Fatal(err)
	}
	if err := run.Complete(); err != nil {
		t.Fatal(err)
	}
	if err := journal.Publish(BatchSummary{Attempted: 1, Succeeded: 1, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	const wantRuns = "{\"artifact\":null,\"domain\":\"success\",\"elapsed_nanos\":\"5\",\"failure_signature\":null,\"io_transcript_records\":null,\"io_transcript_sha256\":null,\"reason\":\"success\",\"seed\":\"7\",\"selection_ordinal\":\"0\",\"termination\":\"exit\"}\n"
	assertFileContents(t, filepath.Join(journal.Path(), "runs.jsonl"), wantRuns)
	const wantBatch = `{"attempted":"1","cancelled":"0","distinct_failures":"0","failure_signatures":[],"failures":"0","run_id":"run-fixed","runs_sha256":"sha256:52c0817ad9d6287383d15753b8c4ff3a4b5c2c805bf742d83bdb06c1d9ef8d44","schema":"gomadv3.batch/v1","schema_version":2,"selection":"7","selection_count":"1","stop_reason":"seeds_exhausted","succeeded":"1","watchdogs":"0"}`
	assertFileContents(t, filepath.Join(journal.Path(), "batch.json"), wantBatch)
	for _, removed := range []string{journal.PreparedPath(), filepath.Join(journal.Path(), ".partial", "batch"), run.Path()} {
		if _, err := os.Stat(removed); !os.IsNotExist(err) {
			t.Fatalf("completed path %q remains: %v", removed, err)
		}
	}
}

func TestOpenBatchValidatesPublishedJournal(t *testing.T) {
	journal, err := NewBatchJournal(context.Background(), BatchConfig{
		Root: t.TempDir(), RunID: "run-open", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	if err := journal.StartRuns(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendRun(RunRecord{
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit", ElapsedNanos: 5,
	}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Publish(BatchSummary{Attempted: 1, Succeeded: 1, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	opened, err := OpenBatch(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if opened.Path != journal.Path() || opened.Record.RunID != "run-open" || len(opened.Runs) != 1 || opened.Runs[0].Seed != 7 {
		t.Fatalf("opened batch = %#v", opened)
	}
}

func TestBatchJournalRecordsCanonicalResumePlan(t *testing.T) {
	journal, err := NewBatchJournal(context.Background(), BatchConfig{
		Root: t.TempDir(), RunID: "run-resume-plan", Selection: "7-9", SelectionCount: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	if err := journal.BeginPreparation(); err != nil {
		t.Fatal(err)
	}
	preparedPath := filepath.Join(journal.PreparedPath(), "build", "target")
	if err := os.MkdirAll(filepath.Dir(preparedPath), 0o700); err != nil {
		t.Fatal(err)
	}
	targetBytes := []byte("prepared target")
	if err := os.WriteFile(preparedPath, targetBytes, 0o500); err != nil {
		t.Fatal(err)
	}
	plan := testBatchPlan(journal, record.HashBytes(targetBytes), uint64(len(targetBytes)))
	if err := journal.RecordPlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := journal.CompletePreparation(); err != nil {
		t.Fatal(err)
	}
	opened, err := ReadResumePlan(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if opened.Selection != "7-9" || opened.Prepared.Target.SHA256 != plan.Prepared.Target.SHA256 || opened.Prepared.Path != ".prepared/build/target" || opened.OverallTimeoutNanos != record.Uint64String(10*time.Second) {
		t.Fatalf("resume plan = %#v", opened)
	}
	if err := os.Chmod(preparedPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(preparedPath, []byte("changed"), 0o500); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(preparedPath, 0o500); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadResumePlan(journal.Path()); err == nil || !strings.Contains(err.Error(), "prepared target identity") {
		t.Fatalf("ReadResumePlan() error = %v", err)
	}
}

func TestResumeBatchJournalReusesVerifiedRunsAndArchivesIncompleteWork(t *testing.T) {
	journal, err := NewBatchJournal(context.Background(), BatchConfig{
		Root: t.TempDir(), RunID: "run-resume", Selection: "7-9", SelectionCount: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.BeginPreparation(); err != nil {
		t.Fatal(err)
	}
	preparedPath := filepath.Join(journal.PreparedPath(), "build", "target")
	if err := os.MkdirAll(filepath.Dir(preparedPath), 0o700); err != nil {
		t.Fatal(err)
	}
	targetBytes := []byte("prepared target")
	if err := os.WriteFile(preparedPath, targetBytes, 0o500); err != nil {
		t.Fatal(err)
	}
	if err := journal.RecordPlan(testBatchPlan(journal, record.HashBytes(targetBytes), uint64(len(targetBytes)))); err != nil {
		t.Fatal(err)
	}
	if err := journal.CompletePreparation(); err != nil {
		t.Fatal(err)
	}
	if err := journal.StartRuns(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendRun(RunRecord{SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit"}); err != nil {
		t.Fatal(err)
	}
	incomplete, err := journal.BeginRun(1, 8)
	if err != nil {
		t.Fatal(err)
	}
	if err := incomplete.Transition(RunStarting); err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}

	resumed, state, err := ResumeBatchJournal(context.Background(), journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	defer resumed.Close()
	if len(state.Runs) != 1 || state.Runs[0].Seed != 7 || state.Plan.Selection != "7-9" {
		t.Fatalf("resume state = %#v", state)
	}
	for ordinal, seed := range []uint64{8, 9} {
		run, err := resumed.BeginRun(uint64(ordinal+1), seed)
		if err != nil {
			t.Fatal(err)
		}
		for _, next := range []RunState{RunStarting, RunExited, RunCaptured, RunClassified} {
			if err := run.Transition(next); err != nil {
				t.Fatal(err)
			}
		}
		if err := resumed.AppendRun(RunRecord{SelectionOrdinal: record.Uint64String(ordinal + 1), Seed: record.Uint64String(seed), Domain: "success", Reason: "success", Termination: "exit"}); err != nil {
			t.Fatal(err)
		}
		if err := run.Complete(); err != nil {
			t.Fatal(err)
		}
	}
	if err := resumed.Publish(BatchSummary{Attempted: 3, Succeeded: 3, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	opened, err := OpenBatch(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if len(opened.Runs) != 3 {
		t.Fatalf("published runs = %#v", opened.Runs)
	}
	archives, err := filepath.Glob(filepath.Join(journal.Path(), ".partial", "resume", "*", "partials", filepath.Base(incomplete.Path())))
	if err != nil || len(archives) != 1 {
		t.Fatalf("archived partials = %v, %v", archives, err)
	}
}

func testBatchPlan(journal *BatchJournal, digest record.SHA256, size uint64) BatchPlan {
	return BatchPlan{
		Schema: BatchPlanSchema, Selection: "7-9", SelectionCount: 3, Parallel: 2,
		RunTimeoutNanos: record.Uint64String(time.Second), OverallTimeoutNanos: record.Uint64String(10 * time.Second), TerminateGraceNanos: record.Uint64String(100 * time.Millisecond),
		OnFailure: "all", FailureBudget: 1, OutputBytes: 64, WorldTransitionBytes: 1024,
		RunnerBuild: string(record.HashBytes([]byte("runner"))), Toolchain: record.Toolchain{GoVersion: "go1.26.4", BuildKey: "build", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Prepared: PreparedTargetPlan{
			Path:   ".prepared/build/target",
			Target: record.Target{Kind: "go-test", Source: "./pkg", SHA256: digest, Size: record.Uint64String(size), Argv: []string{"gomadv3-target"}, BuildTags: []string{"gomad_fixture"}, Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}},
		},
		IOProfile:   IOProfilePlan{Name: "gomadv3-deterministic/v1", ImplementationSHA256: record.HashBytes([]byte("io")), InventorySHA256: record.HashBytes([]byte("inventory"))},
		Environment: []record.Environment{{Name: "GOMADV3_IO_PROFILE", Value: "gomadv3-deterministic/v1"}},
		Coverage:    "semantic", RequiredSemanticProbes: []string{"stdlib.os.openfile"}, KeepSuccesses: "none",
	}
}

func TestOpenBatchRejectsChangedRuns(t *testing.T) {
	journal, err := NewBatchJournal(context.Background(), BatchConfig{
		Root: t.TempDir(), RunID: "run-tampered", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	if err := journal.StartRuns(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendRun(RunRecord{
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit", ElapsedNanos: 5,
	}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Publish(BatchSummary{Attempted: 1, Succeeded: 1, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	runs := filepath.Join(journal.Path(), "runs.jsonl")
	if err := os.WriteFile(runs, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenBatch(journal.Path()); err == nil || !strings.Contains(err.Error(), "runs digest") {
		t.Fatalf("OpenBatch() error = %v", err)
	}
}

func TestBatchJournalPreservesExplicitFailureState(t *testing.T) {
	journal, err := NewBatchJournal(context.Background(), BatchConfig{
		Root: t.TempDir(), RunID: "run-failed", Selection: "1", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	if err := journal.BeginPreparation(); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("build failed")
	if err := journal.FailPreparation("target_preparation", failure); err != nil {
		t.Fatal(err)
	}
	const wantFailure = `{"detail":"build failed","reason":"target_preparation","schema_version":2,"state":"failed"}`
	assertFileContents(t, filepath.Join(journal.Path(), ".partial", "preparation", "partial.json"), wantFailure)
	if err := journal.Fail("target_preparation", failure); err != nil {
		t.Fatal(err)
	}
	assertFileContents(t, filepath.Join(journal.Path(), ".partial", "batch", "partial.json"), wantFailure)
}

func TestRunJournalRejectsInvalidStateTransitions(t *testing.T) {
	journal, err := NewBatchJournal(context.Background(), BatchConfig{
		Root: t.TempDir(), RunID: "run-state", Selection: "1", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	run, err := journal.BeginRun(0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := run.Transition(RunCaptured); err == nil {
		t.Fatal("Transition(captured) succeeded before starting")
	}
	if err := run.Complete(); err == nil {
		t.Fatal("Complete() succeeded before classification")
	}
}

func assertFileContents(t *testing.T, path, want string) {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != want {
		t.Fatalf("%s = %q, want %q", filepath.Base(path), contents, want)
	}
}
