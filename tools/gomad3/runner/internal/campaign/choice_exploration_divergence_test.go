package campaign

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
	choiceengine "go.temporal.io/server/tools/gomad3/runner/internal/exploration/choice"
)

func journalDivergence(t *testing.T) (string, choiceengine.State, *ExplorationJournal, choiceengine.RoundSegment) {
	t.Helper()
	path := privateDirectory(t)
	initial := testExplorationState(t)
	journal, err := NewExplorationJournal(context.Background(), path, initial, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	identities := [][32]byte{sha256.Sum256([]byte("first")), sha256.Sum256([]byte("second"))}
	decision, err := choice.CanonicalDecision(0, choice.KindRunnable, 1, false, identities, identities[0], 0)
	if err != nil {
		t.Fatal(err)
	}
	trace, err := choice.BuildTrace([]choice.Record{decision.Record()}, choice.TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	tape, err := choice.ProjectReplayPlan(trace, initial.Config.Execution)
	if err != nil {
		t.Fatal(err)
	}
	root, _ := initial.NextRound()
	staged, err := journal.StageRound(root)
	if err != nil {
		t.Fatal(err)
	}
	outcome := record.HashBytes([]byte("root"))
	roundValue, depth := record.Uint64String(0), record.Uint64String(0)
	if err := staged.RecordExecution(0, ExecutionRecord{Strategy: "choice-exploration", Round: &roundValue, CandidateSHA256: root.Candidates[0].SHA256, ForcedDepth: &depth, OutcomeSHA256: outcome, Seed: 7, Domain: "success", Reason: "success", Termination: "exit"}); err != nil {
		t.Fatal(err)
	}
	state, segment, err := choiceengine.CommitRound(initial, root, []choiceengine.Result{{CandidateSHA256: root.Candidates[0].SHA256, OutcomeSHA256: outcome, Trace: &tape}})
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.CommitRound(staged, segment); err != nil {
		t.Fatal(err)
	}
	round, _ := state.NextRound()
	candidate := round.Candidates[0]
	prefix, err := candidate.PrefixReplayPlan(state.Config.Execution)
	if err != nil {
		t.Fatal(err)
	}
	expected := prefix.Decisions[0]
	observed := decision
	observed.AlternativeSetDigest = sha256.Sum256([]byte("changed"))
	divergence := &choice.Divergence{Ordinal: 0, Reason: choice.DivergenceAlternativeSet, Expected: &expected, Observed: &observed, TapeRecords: 1}
	evidence := choice.ProjectDivergenceEvidence(*divergence)
	staged, err = journal.StageRound(round)
	if err != nil {
		t.Fatal(err)
	}
	childRoundValue, childDepth := record.Uint64String(1), record.Uint64String(1)
	if err := staged.RecordExecution(0, ExecutionRecord{Strategy: "choice-exploration", Round: &childRoundValue, CandidateSHA256: candidate.SHA256, ParentCandidateSHA256: candidate.ParentSHA256, PrefixSHA256: candidate.PrefixSHA256, ForcedDepth: &childDepth, SelectionOrdinal: 1, Seed: 7, Domain: "runner", Reason: "replay_divergence", Termination: "none", Divergence: &evidence}); err != nil {
		t.Fatal(err)
	}
	final, segment, err := choiceengine.CommitRound(state, round, []choiceengine.Result{{CandidateSHA256: candidate.SHA256, Divergence: divergence}})
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.CommitRound(staged, segment); err != nil {
		t.Fatal(err)
	}
	return path, final, journal, segment
}

func TestResumeExplorationJournalRetainsDivergenceAndRejectsCorruption(t *testing.T) {
	for _, change := range []string{"none", "old schema", "run divergence"} {
		t.Run(change, func(t *testing.T) {
			path, final, journal, segment := journalDivergence(t)
			roundPath := filepath.Join(path, "choice-exploration", "rounds", "00000000000000000001")
			switch change {
			case "old schema":
				segment.Schema = "gomad3.choice-exploration-round/v1"
				data, err := canonicaljson.CanonicalJSON(segment)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(roundPath, "segment.json"), data, 0o600); err != nil {
					t.Fatal(err)
				}
			case "run divergence":
				run := journal.CommittedExecutions()[1]
				run.Divergence.Ordinal++
				data, err := canonicaljson.CanonicalJSON(run)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(roundPath, "executions", "00000000000000000000.json"), data, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			resumed, state, recovery, err := ResumeExplorationJournal(t.Context(), path, final.Config, 1<<20)
			if change != "none" {
				if err == nil || change == "old schema" && !strings.Contains(err.Error(), "schema") {
					t.Fatalf("corruption error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(state, final) {
				t.Fatal("resume changed divergent state")
			}
			if recovery != 0 {
				t.Fatalf("recovery executions = %d", recovery)
			}
			if !reflect.DeepEqual(resumed.CommittedExecutions(), journal.CommittedExecutions()) {
				t.Fatal("resume changed execution records")
			}
		})
	}
}

func TestDivergenceFieldsAreAbsentFromOrdinaryJSON(t *testing.T) {
	for _, value := range []any{ExecutionRecord{}, CampaignRecord{}, choiceengine.SegmentResult{}} {
		data, err := canonicaljson.CanonicalJSON(value)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "divergence") {
			t.Fatalf("ordinary JSON gained divergence fields: %s", data)
		}
	}
}

func TestExplorationDivergenceEvidenceCannotBeUsedAsTargetOutcome(t *testing.T) {
	_, _, journal, _ := journalDivergence(t)
	for _, test := range []struct {
		name   string
		change func(*ExecutionRecord)
	}{
		{"outcome", func(r *ExecutionRecord) { r.OutcomeSHA256 = record.HashBytes([]byte("outcome")) }},
		{"target", func(r *ExecutionRecord) { r.Domain = "target" }},
		{"signature", func(r *ExecutionRecord) { v := record.HashBytes([]byte("signature")); r.FailureSignature = &v }},
		{"artifact", func(r *ExecutionRecord) { v := "failures/x"; r.Artifact = &v }},
		{"trace", func(r *ExecutionRecord) { v := record.HashBytes([]byte("trace")); r.ChoiceTraceSHA256 = &v }},
		{"success", func(r *ExecutionRecord) { v := "successes/x"; r.SuccessArtifact = &v }},
		{"root", func(r *ExecutionRecord) {
			v := record.Uint64String(0)
			r.ForcedDepth = &v
			r.ParentCandidateSHA256 = ""
			r.PrefixSHA256 = ""
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			run := journal.CommittedExecutions()[1]
			test.change(&run)
			if err := validateExplorationExecutionSummary(run, map[record.SHA256]struct{}{}); err == nil {
				t.Fatal("accepted invalid divergence record")
			}
		})
	}
	run := journal.CommittedExecutions()[1]
	run.Strategy = "simulation-exploration"
	if err := validateSimulationExplorationExecutionSummary(run, map[record.SHA256]struct{}{}); err == nil {
		t.Fatal("accepted divergence for simulation strategy")
	}
}
