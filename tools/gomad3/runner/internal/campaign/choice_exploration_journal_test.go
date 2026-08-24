package campaign

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
	choiceengine "go.temporal.io/server/tools/gomad3/runner/internal/exploration/choice"
)

func TestExplorationJournalCommitsAndReplaysWholeRounds(t *testing.T) {
	batchPath := privateDirectory(t)
	state := testExplorationState(t)
	journal, err := NewExplorationJournal(context.Background(), batchPath, state, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	round, ok := state.NextRound()
	if !ok {
		t.Fatal("root exploration round is unavailable")
	}
	staged, err := journal.StageRound(round)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(staged.Path(), "round.json")); err != nil {
		t.Fatal(err)
	}
	next, segment, err := choiceengine.CommitRound(state, round, []choiceengine.Result{{
		CandidateSHA256: round.Candidates[0].SHA256, OutcomeSHA256: record.HashBytes([]byte("success")),
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
	if _, err := os.Stat(filepath.Join(batchPath, "choice-exploration", "rounds", "00000000000000000000", "segment.json")); err != nil {
		t.Fatal(err)
	}
	resumed, recovered, recoveryExecutions, err := ResumeExplorationJournal(context.Background(), batchPath, state.Config, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if recoveryExecutions != 0 {
		t.Fatalf("recovery executions = %d", recoveryExecutions)
	}
	got, err := choiceengine.StateSHA256(recovered)
	if err != nil {
		t.Fatal(err)
	}
	want, err := choiceengine.StateSHA256(next)
	if err != nil {
		t.Fatal(err)
	}
	if got != want || resumed.State().Summary() != next.Summary() {
		t.Fatalf("resumed exploration = %#v, want %#v", recovered, next)
	}
}

func TestResumeExplorationJournalArchivesIncompleteRoundAtomically(t *testing.T) {
	batchPath := privateDirectory(t)
	state := testExplorationState(t)
	journal, err := NewExplorationJournal(context.Background(), batchPath, state, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	round, _ := state.NextRound()
	staged, err := journal.StageRound(round)
	if err != nil {
		t.Fatal(err)
	}
	resumed, recovered, recoveryExecutions, err := ResumeExplorationJournal(context.Background(), batchPath, state.Config, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if recoveryExecutions != 0 || recovered.Summary() != state.Summary() || resumed.State().Summary() != state.Summary() {
		t.Fatalf("resume = %#v recovery=%d", recovered, recoveryExecutions)
	}
	if _, err := os.Stat(staged.Path()); !os.IsNotExist(err) {
		t.Fatalf("incomplete round remains: %v", err)
	}
	archives, err := filepath.Glob(filepath.Join(batchPath, ".partial", "resume", "*", "partials", "choice-exploration", filepath.Base(staged.Path())))
	if err != nil || len(archives) != 1 {
		t.Fatalf("exploration archives = %v, %v", archives, err)
	}
}

func TestResumeExplorationJournalRejectsCorruptOrNonContiguousSegments(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*testing.T, string)
	}{
		{name: "corrupt", mutate: func(t *testing.T, path string) {
			segment := filepath.Join(path, "choice-exploration", "rounds", "00000000000000000000", "segment.json")
			if err := os.WriteFile(segment, []byte("{}"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "gap", mutate: func(t *testing.T, path string) {
			oldPath := filepath.Join(path, "choice-exploration", "rounds", "00000000000000000000")
			newPath := filepath.Join(path, "choice-exploration", "rounds", "00000000000000000001")
			if err := os.Rename(oldPath, newPath); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			batchPath := privateDirectory(t)
			state := testExplorationState(t)
			journal, err := NewExplorationJournal(context.Background(), batchPath, state, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			round, _ := state.NextRound()
			staged, err := journal.StageRound(round)
			if err != nil {
				t.Fatal(err)
			}
			_, segment, err := choiceengine.CommitRound(state, round, []choiceengine.Result{{CandidateSHA256: round.Candidates[0].SHA256, OutcomeSHA256: record.HashBytes([]byte("success"))}})
			if err != nil {
				t.Fatal(err)
			}
			if err := journal.CommitRound(staged, segment); err != nil {
				t.Fatal(err)
			}
			test.mutate(t, batchPath)
			if _, _, _, err := ResumeExplorationJournal(context.Background(), batchPath, state.Config, 1<<20); err == nil {
				t.Fatal("ResumeExplorationJournal() accepted invalid segment state")
			}
		})
	}
}

func TestExplorationJournalRejectsChangedPlanAndOversizedSegment(t *testing.T) {
	batchPath := privateDirectory(t)
	state := testExplorationState(t)
	journal, err := NewExplorationJournal(context.Background(), batchPath, state, 128)
	if err != nil {
		t.Fatal(err)
	}
	round, _ := state.NextRound()
	staged, err := journal.StageRound(round)
	if err != nil {
		t.Fatal(err)
	}
	_, segment, err := choiceengine.CommitRound(state, round, []choiceengine.Result{{CandidateSHA256: round.Candidates[0].SHA256, OutcomeSHA256: record.HashBytes([]byte(strings.Repeat("x", 16)))}})
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.CommitRound(staged, segment); err == nil || !strings.Contains(err.Error(), "capacity") {
		t.Fatalf("CommitRound() error = %v", err)
	}
	changed := state.Config
	changed.BaseSeed++
	if _, _, _, err := ResumeExplorationJournal(context.Background(), batchPath, changed, 128); err == nil {
		t.Fatal("ResumeExplorationJournal() accepted a changed plan")
	}
}

func TestResumeExplorationJournalRejectsRunProjectionDivergence(t *testing.T) {
	batchPath := privateDirectory(t)
	state := testExplorationState(t)
	journal, err := NewExplorationJournal(context.Background(), batchPath, state, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	round, _ := state.NextRound()
	staged, err := journal.StageRound(round)
	if err != nil {
		t.Fatal(err)
	}
	outcome := record.HashBytes([]byte("success"))
	roundValue := record.Uint64String(round.Index)
	depth := record.Uint64String(0)
	if err := staged.RecordExecution(0, ExecutionRecord{
		Strategy: "choice-exploration", Round: &roundValue, CandidateSHA256: round.Candidates[0].SHA256, ForcedDepth: &depth, OutcomeSHA256: outcome,
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit",
	}); err != nil {
		t.Fatal(err)
	}
	_, segment, err := choiceengine.CommitRound(state, round, []choiceengine.Result{{CandidateSHA256: round.Candidates[0].SHA256, OutcomeSHA256: outcome}})
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.CommitRound(staged, segment); err != nil {
		t.Fatal(err)
	}
	runPath := filepath.Join(batchPath, "choice-exploration", "rounds", "00000000000000000000", "executions", "00000000000000000000.json")
	contents, err := os.ReadFile(runPath)
	if err != nil {
		t.Fatal(err)
	}
	var run ExecutionRecord
	if err := canonicaljson.DecodeCanonicalJSON(contents, &run); err != nil {
		t.Fatal(err)
	}
	run.OutcomeSHA256 = record.HashBytes([]byte("changed"))
	contents, err = canonicaljson.CanonicalJSON(run)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := ResumeExplorationJournal(context.Background(), batchPath, state.Config, 1<<20); err == nil {
		t.Fatal("ResumeExplorationJournal() accepted a divergent run projection")
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

func testExplorationState(t *testing.T) choiceengine.State {
	t.Helper()
	state, err := choiceengine.New(choiceengine.Config{
		Execution: choice.ExecutionIdentity{
			TargetSHA256: sha256.Sum256([]byte("target")), ToolchainBuildKey: strings.Repeat("a", 64),
			GOOS: "darwin", GOARCH: "arm64", ImplementationSHA256: sha256.Sum256([]byte("controller")),
		},
		ControllerSHA256: choiceengine.ImplementationSHA256(),
		BaseSeed:         7, Parallel: 2, MaxExecutions: 8, MaxChoiceDepth: 4, MaxExplorationBytes: 1 << 20,
		FailurePolicy: choiceengine.PolicyAll, FailureBudget: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	return state
}
