package artifact

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
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
	assertFileContents(t, partialPath, `{"schema_version":1,"seed":"7","selection_ordinal":"0","state":"staging"}`)
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
	const wantBatch = `{"attempted":"1","cancelled":"0","distinct_failures":"0","failure_signatures":[],"failures":"0","run_id":"run-fixed","runs_sha256":"sha256:52c0817ad9d6287383d15753b8c4ff3a4b5c2c805bf742d83bdb06c1d9ef8d44","schema":"gomadv3.batch/v1","schema_version":1,"selection":"7","selection_count":"1","stop_reason":"seeds_exhausted","succeeded":"1","watchdogs":"0"}`
	assertFileContents(t, filepath.Join(journal.Path(), "batch.json"), wantBatch)
	for _, removed := range []string{journal.PreparedPath(), filepath.Join(journal.Path(), ".partial", "batch"), run.Path()} {
		if _, err := os.Stat(removed); !os.IsNotExist(err) {
			t.Fatalf("completed path %q remains: %v", removed, err)
		}
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
	const wantFailure = `{"detail":"build failed","reason":"target_preparation","schema_version":1,"state":"failed"}`
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
