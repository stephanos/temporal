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
	roundValue := evidence.Uint64String(round.Index)
	depth := evidence.Uint64String(len(round.Candidates[0].Overrides))
	outcome := evidence.HashBytes([]byte("success"))
	run := ExecutionRecord{
		Strategy: "combined-frontier", Round: &roundValue, CandidateSHA256: round.Candidates[0].SHA256,
		ParentCandidateSHA256: round.Candidates[0].ParentSHA256, ForcedDepth: &depth, OutcomeSHA256: outcome,
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "exit_zero", Termination: "exit",
	}
	if err := staged.RecordExecution(0, run); err != nil {
		t.Fatal(err)
	}
	next, segment, err := combinedfrontier.CommitRound(state, round, []combinedfrontier.Result{{
		CandidateSHA256: round.Candidates[0].SHA256,
		OutcomeSHA256:   outcome,
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
	committed := resumed.CommittedExecutions()
	if len(committed) != 1 {
		t.Fatalf("committed executions = %#v", committed)
	}
	left, err := evidence.CanonicalJSON(committed[0])
	if err != nil {
		t.Fatal(err)
	}
	right, err := evidence.CanonicalJSON(run)
	if err != nil {
		t.Fatal(err)
	}
	if string(left) != string(right) {
		t.Fatalf("committed execution = %s, want %s", left, right)
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
	if _, err := staged.BeginExecution(0, 7); err != nil {
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

func TestResumeCombinedFrontierJournalDoesNotCountUnstartedRoundCandidates(t *testing.T) {
	batchPath := privateDirectory(t)
	state := testCombinedFrontierState(t)
	journal, err := NewCombinedFrontierJournal(context.Background(), batchPath, state, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	round, _ := state.NextRound()
	if _, err := journal.StageRound(round); err != nil {
		t.Fatal(err)
	}

	_, _, recoveryExecutions, err := ResumeCombinedFrontierJournal(context.Background(), batchPath, state.Config, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if recoveryExecutions != 0 {
		t.Fatalf("unstarted recovery executions = %d, want 0", recoveryExecutions)
	}
}

func TestInspectCombinedFrontierReportsPendingAndStagedWorkWithoutMutation(t *testing.T) {
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
	if _, err := staged.BeginExecution(0, state.Config.BaseSeed); err != nil {
		t.Fatal(err)
	}

	inspected, err := InspectCombinedFrontier(batchPath)
	if err != nil {
		t.Fatal(err)
	}
	if inspected.Summary != state.Summary() || inspected.ImplementationSHA256 != combinedfrontier.ImplementationSHA256() || inspected.ChainSHA256 == "" || len(inspected.Pending) != 1 || inspected.Pending[0].SHA256 != round.Candidates[0].SHA256 {
		t.Fatalf("combined frontier inspection = %#v", inspected)
	}
	if inspected.StagedRound == nil || inspected.StagedRound.Index != 0 || inspected.StagedRound.Candidates != 1 || inspected.StagedRound.Attempted != 1 {
		t.Fatalf("staged combined frontier inspection = %#v", inspected.StagedRound)
	}
	if _, err := os.Stat(staged.Path()); err != nil {
		t.Fatalf("inspection mutated staged round: %v", err)
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
