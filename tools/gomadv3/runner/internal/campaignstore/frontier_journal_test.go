package campaignstore

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/frontier"
)

func TestFrontierJournalCommitsAndReplaysWholeRounds(t *testing.T) {
	batchPath := privateDirectory(t)
	state := testFrontierState(t)
	journal, err := NewFrontierJournal(context.Background(), batchPath, state, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	round, ok := state.NextRound()
	if !ok {
		t.Fatal("root frontier round is unavailable")
	}
	staged, err := journal.StageRound(round)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(staged.Path(), "round.json")); err != nil {
		t.Fatal(err)
	}
	next, segment, err := frontier.CommitRound(state, round, []frontier.Result{{
		CandidateSHA256: round.Candidates[0].SHA256, OutcomeSHA256: evidence.HashBytes([]byte("success")),
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
	if _, err := os.Stat(filepath.Join(batchPath, "frontier", "rounds", "00000000000000000000", "segment.json")); err != nil {
		t.Fatal(err)
	}
	resumed, recovered, recoveryExecutions, err := ResumeFrontierJournal(context.Background(), batchPath, state.Config, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if recoveryExecutions != 0 {
		t.Fatalf("recovery executions = %d", recoveryExecutions)
	}
	got, err := frontier.StateSHA256(recovered)
	if err != nil {
		t.Fatal(err)
	}
	want, err := frontier.StateSHA256(next)
	if err != nil {
		t.Fatal(err)
	}
	if got != want || resumed.State().Summary() != next.Summary() {
		t.Fatalf("resumed frontier = %#v, want %#v", recovered, next)
	}
}

func TestResumeFrontierJournalArchivesIncompleteRoundAtomically(t *testing.T) {
	batchPath := privateDirectory(t)
	state := testFrontierState(t)
	journal, err := NewFrontierJournal(context.Background(), batchPath, state, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	round, _ := state.NextRound()
	staged, err := journal.StageRound(round)
	if err != nil {
		t.Fatal(err)
	}
	resumed, recovered, recoveryExecutions, err := ResumeFrontierJournal(context.Background(), batchPath, state.Config, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if recoveryExecutions != 0 || recovered.Summary() != state.Summary() || resumed.State().Summary() != state.Summary() {
		t.Fatalf("resume = %#v recovery=%d", recovered, recoveryExecutions)
	}
	if _, err := os.Stat(staged.Path()); !os.IsNotExist(err) {
		t.Fatalf("incomplete round remains: %v", err)
	}
	archives, err := filepath.Glob(filepath.Join(batchPath, ".partial", "resume", "*", "partials", "frontier", filepath.Base(staged.Path())))
	if err != nil || len(archives) != 1 {
		t.Fatalf("frontier archives = %v, %v", archives, err)
	}
}

func TestResumeFrontierJournalRejectsCorruptOrNonContiguousSegments(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*testing.T, string)
	}{
		{name: "corrupt", mutate: func(t *testing.T, path string) {
			segment := filepath.Join(path, "frontier", "rounds", "00000000000000000000", "segment.json")
			if err := os.WriteFile(segment, []byte("{}"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "gap", mutate: func(t *testing.T, path string) {
			oldPath := filepath.Join(path, "frontier", "rounds", "00000000000000000000")
			newPath := filepath.Join(path, "frontier", "rounds", "00000000000000000001")
			if err := os.Rename(oldPath, newPath); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			batchPath := privateDirectory(t)
			state := testFrontierState(t)
			journal, err := NewFrontierJournal(context.Background(), batchPath, state, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			round, _ := state.NextRound()
			staged, err := journal.StageRound(round)
			if err != nil {
				t.Fatal(err)
			}
			_, segment, err := frontier.CommitRound(state, round, []frontier.Result{{CandidateSHA256: round.Candidates[0].SHA256, OutcomeSHA256: evidence.HashBytes([]byte("success"))}})
			if err != nil {
				t.Fatal(err)
			}
			if err := journal.CommitRound(staged, segment); err != nil {
				t.Fatal(err)
			}
			test.mutate(t, batchPath)
			if _, _, _, err := ResumeFrontierJournal(context.Background(), batchPath, state.Config, 1<<20); err == nil {
				t.Fatal("ResumeFrontierJournal() accepted invalid segment state")
			}
		})
	}
}

func TestFrontierJournalRejectsChangedPlanAndOversizedSegment(t *testing.T) {
	batchPath := privateDirectory(t)
	state := testFrontierState(t)
	journal, err := NewFrontierJournal(context.Background(), batchPath, state, 128)
	if err != nil {
		t.Fatal(err)
	}
	round, _ := state.NextRound()
	staged, err := journal.StageRound(round)
	if err != nil {
		t.Fatal(err)
	}
	_, segment, err := frontier.CommitRound(state, round, []frontier.Result{{CandidateSHA256: round.Candidates[0].SHA256, OutcomeSHA256: evidence.HashBytes([]byte(strings.Repeat("x", 16)))}})
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.CommitRound(staged, segment); err == nil || !strings.Contains(err.Error(), "capacity") {
		t.Fatalf("CommitRound() error = %v", err)
	}
	changed := state.Config
	changed.BaseSeed++
	if _, _, _, err := ResumeFrontierJournal(context.Background(), batchPath, changed, 128); err == nil {
		t.Fatal("ResumeFrontierJournal() accepted a changed plan")
	}
}

func TestResumeFrontierJournalRejectsRunProjectionDivergence(t *testing.T) {
	batchPath := privateDirectory(t)
	state := testFrontierState(t)
	journal, err := NewFrontierJournal(context.Background(), batchPath, state, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	round, _ := state.NextRound()
	staged, err := journal.StageRound(round)
	if err != nil {
		t.Fatal(err)
	}
	outcome := evidence.HashBytes([]byte("success"))
	roundValue := evidence.Uint64String(round.Index)
	depth := evidence.Uint64String(0)
	if err := staged.RecordExecution(0, ExecutionRecord{
		Strategy: "choice-frontier", Round: &roundValue, CandidateSHA256: round.Candidates[0].SHA256, ForcedDepth: &depth, OutcomeSHA256: outcome,
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit",
	}); err != nil {
		t.Fatal(err)
	}
	_, segment, err := frontier.CommitRound(state, round, []frontier.Result{{CandidateSHA256: round.Candidates[0].SHA256, OutcomeSHA256: outcome}})
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.CommitRound(staged, segment); err != nil {
		t.Fatal(err)
	}
	runPath := filepath.Join(batchPath, "frontier", "rounds", "00000000000000000000", "runs", "00000000000000000000.json")
	contents, err := os.ReadFile(runPath)
	if err != nil {
		t.Fatal(err)
	}
	var run ExecutionRecord
	if err := evidence.DecodeCanonicalJSON(contents, &run); err != nil {
		t.Fatal(err)
	}
	run.OutcomeSHA256 = evidence.HashBytes([]byte("changed"))
	contents, err = evidence.CanonicalJSON(run)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := ResumeFrontierJournal(context.Background(), batchPath, state.Config, 1<<20); err == nil {
		t.Fatal("ResumeFrontierJournal() accepted a divergent run projection")
	}
}

func privateDirectory(t *testing.T) string {
	t.Helper()
	path := t.TempDir()
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func testFrontierState(t *testing.T) frontier.State {
	t.Helper()
	state, err := frontier.New(frontier.Config{
		Execution: choice.ExecutionIdentity{
			TargetSHA256: sha256.Sum256([]byte("target")), ToolchainBuildKey: strings.Repeat("a", 64),
			GOOS: "darwin", GOARCH: "arm64", ImplementationSHA256: sha256.Sum256([]byte("controller")),
		},
		ControllerSHA256: frontier.ImplementationSHA256(),
		BaseSeed:         7, Parallel: 2, MaxRuns: 8, MaxChoiceDepth: 4, MaxFrontierBytes: 1 << 20,
		FailurePolicy: frontier.PolicyAll, FailureBudget: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	return state
}
