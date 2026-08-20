package campaignstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/combinedfrontier"
)

func TestCombinedFrontierJournalCommitsAndResumesWholeRounds(t *testing.T) {
	batchPath := privateDirectory(t)
	state := testCombinedFrontierState(t)
	journal, err := NewCombinedFrontierJournal(context.Background(), batchPath, state, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	round, ok := state.NextRound()
	if !ok {
		t.Fatal("combined frontier root round is unavailable")
	}
	staged, err := journal.StageRound(round)
	if err != nil {
		t.Fatal(err)
	}
	next, segment, err := combinedfrontier.CommitRound(state, round, []combinedfrontier.Result{{
		CandidateSHA256: round.Candidates[0].SHA256,
		OutcomeSHA256:   evidence.HashBytes([]byte("success")),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.CommitRound(staged, segment); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(staged.Path()); !os.IsNotExist(err) {
		t.Fatalf("staged round remains after commit: %v", err)
	}
	resumed, recovered, recoveryExecutions, err := ResumeCombinedFrontierJournal(context.Background(), batchPath, state.Config, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if recoveryExecutions != 0 {
		t.Fatalf("recovery executions = %d", recoveryExecutions)
	}
	got, err := combinedfrontier.StateSHA256(recovered)
	if err != nil {
		t.Fatal(err)
	}
	want, err := combinedfrontier.StateSHA256(next)
	if err != nil {
		t.Fatal(err)
	}
	if got != want || resumed.State().Summary() != next.Summary() {
		t.Fatalf("resumed combined frontier = %#v, want %#v", recovered, next)
	}
}

func TestResumeCombinedFrontierJournalDiscardsOnlyIncompleteRound(t *testing.T) {
	batchPath := privateDirectory(t)
	state := testCombinedFrontierState(t)
	journal, err := NewCombinedFrontierJournal(context.Background(), batchPath, state, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	round, _ := state.NextRound()
	staged, err := journal.StageRound(round)
	if err != nil {
		t.Fatal(err)
	}
	resumed, recovered, recoveryExecutions, err := ResumeCombinedFrontierJournal(context.Background(), batchPath, state.Config, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if recoveryExecutions != 1 || recovered.Summary() != state.Summary() || resumed.State().Summary() != state.Summary() {
		t.Fatalf("resume = %#v recovery=%d", recovered, recoveryExecutions)
	}
	if _, err := os.Stat(staged.Path()); !os.IsNotExist(err) {
		t.Fatalf("incomplete round remains: %v", err)
	}
}

func TestResumeCombinedFrontierJournalRejectsCorruptSegment(t *testing.T) {
	batchPath := privateDirectory(t)
	state := testCombinedFrontierState(t)
	journal, err := NewCombinedFrontierJournal(context.Background(), batchPath, state, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	round, _ := state.NextRound()
	staged, err := journal.StageRound(round)
	if err != nil {
		t.Fatal(err)
	}
	_, segment, err := combinedfrontier.CommitRound(state, round, []combinedfrontier.Result{{
		CandidateSHA256: round.Candidates[0].SHA256,
		OutcomeSHA256:   evidence.HashBytes([]byte("success")),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.CommitRound(staged, segment); err != nil {
		t.Fatal(err)
	}
	segmentPath := filepath.Join(batchPath, "combined-frontier", "rounds", "00000000000000000000", "segment.json")
	if err := os.WriteFile(segmentPath, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := ResumeCombinedFrontierJournal(context.Background(), batchPath, state.Config, 1<<20); err == nil {
		t.Fatal("ResumeCombinedFrontierJournal() accepted a corrupt segment")
	}
}

func testCombinedFrontierState(t *testing.T) combinedfrontier.State {
	t.Helper()
	state, err := combinedfrontier.New(combinedfrontier.Config{
		ExecutionSHA256: evidence.HashBytes([]byte("execution")), ControllerSHA256: combinedfrontier.ImplementationSHA256(),
		BaseSeed: 7, Parallel: 2, MaxRuns: 8, MaxForcedDecisions: 4, MaxFrontierBytes: 1 << 20,
		MaxResultBytes: 1 << 20, FailureBudget: 4,
		Limits: combinedfrontier.DimensionLimits{Runtime: 4, Scenario: 4, Network: 4, Storage: 4, Fault: 4, Crash: 4},
	})
	if err != nil {
		t.Fatal(err)
	}
	return state
}
