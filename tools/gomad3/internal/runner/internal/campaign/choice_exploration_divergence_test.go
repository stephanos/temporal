package campaign

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/model/choice"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/runner/internal/exploration"
)

// journalDivergence commits a root round whose forced child then diverges
// from its prefix, and returns the campaign, the final state and the
// committed executions.
func journalDivergence(t *testing.T) (*CampaignJournal, exploration.ChoiceState, []ExecutionRecord) {
	t.Helper()
	initial := testExplorationState(t)
	journal := newExplorationCampaign(t, context.Background(), "choice-exploration")
	rounds, err := OpenRounds(journal, ChoiceRounds, initial.Config, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	root, _ := initial.NextRound()
	results, runs := choiceRoundOutcomes(t, initial, root)
	state, segment, err := exploration.CommitChoiceRound(initial, root, results)
	if err != nil {
		t.Fatal(err)
	}
	if err := rounds.Start(root); err != nil {
		t.Fatal(err)
	}
	if err := rounds.Commit(root, segment, runs); err != nil {
		t.Fatal(err)
	}
	round, _ := state.NextRound()
	candidate := round.Candidates[0]
	prefix, err := candidate.PrefixReplayPlan(state.Config.Execution)
	if err != nil {
		t.Fatal(err)
	}
	expected := prefix.Decisions[0]
	identities := [][32]byte{sha256.Sum256([]byte("first")), sha256.Sum256([]byte("second"))}
	observed, err := choice.CanonicalDecision(0, choice.KindRunnable, 1, false, identities, identities[0], 0)
	if err != nil {
		t.Fatal(err)
	}
	observed.AlternativeSetDigest = sha256.Sum256([]byte("changed"))
	divergence := &choice.Divergence{Ordinal: 0, Reason: choice.DivergenceAlternativeSet, Expected: &expected, Observed: &observed, TapeRecords: 1}
	evidence := choice.ProjectDivergenceEvidence(*divergence)
	childRoundValue, childDepth := record.Uint64String(1), record.Uint64String(1)
	child := ExecutionRecord{Strategy: "choice-exploration", Round: &childRoundValue, CandidateSHA256: candidate.SHA256, ParentCandidateSHA256: candidate.ParentSHA256, PrefixSHA256: candidate.PrefixSHA256, ForcedDepth: &childDepth, SelectionOrdinal: 1, Seed: 7, Domain: "runner", Reason: "replay_divergence", Termination: "none", Divergence: &evidence}
	final, segment, err := exploration.CommitChoiceRound(state, round, []exploration.ChoiceResult{{CandidateSHA256: candidate.SHA256, Divergence: divergence}})
	if err != nil {
		t.Fatal(err)
	}
	if err := rounds.Start(round); err != nil {
		t.Fatal(err)
	}
	changed := child
	changedEvidence := evidence
	changedEvidence.Ordinal++
	changed.Divergence = &changedEvidence
	if err := rounds.Commit(round, segment, []ExecutionRecord{changed}); err == nil {
		t.Fatal("committed a divergence that differs from its segment")
	}
	oldSchema := segment
	oldSchema.Schema = "gomad3.choice-exploration-round/v1"
	if err := rounds.Commit(round, oldSchema, []ExecutionRecord{child}); err == nil || !strings.Contains(err.Error(), "schema") {
		t.Fatalf("old segment schema error = %v", err)
	}
	if err := rounds.Commit(round, segment, []ExecutionRecord{child}); err != nil {
		t.Fatal(err)
	}
	return journal, final, append(runs, child)
}

// A replay divergence is committed with its round and replays unchanged; a
// divergence or segment that does not match is refused.
func TestExplorationRoundsRetainDivergenceAndRejectCorruption(t *testing.T) {
	journal, final, executions := journalDivergence(t)
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	resumed, resumeState, err := ResumeCampaignJournal(t.Context(), journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resumed.Close(); err != nil {
			t.Error(err)
		}
	}()
	rounds, err := OpenRounds(resumed, ChoiceRounds, final.Config, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rounds.State(), final) {
		t.Fatal("resume changed divergent state")
	}
	if rounds.RecoveryExecutions() != 0 || !reflect.DeepEqual(resumeState.Executions, executions) {
		t.Fatalf("resumed executions = %#v", resumeState.Executions)
	}
}

func TestDivergenceFieldsAreAbsentFromOrdinaryJSON(t *testing.T) {
	for _, value := range []any{ExecutionRecord{}, CampaignRecord{}, exploration.ChoiceSegmentResult{}} {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "divergence") {
			t.Fatalf("ordinary JSON gained divergence fields: %s", data)
		}
	}
}

func TestExplorationDivergenceEvidenceCannotBeUsedAsTargetOutcome(t *testing.T) {
	_, _, executions := journalDivergence(t)
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
			run := executions[1]
			test.change(&run)
			if err := validateExplorationExecutionSummary(run, map[record.SHA256]struct{}{}); err == nil {
				t.Fatal("accepted invalid divergence record")
			}
		})
	}
	run := executions[1]
	run.Strategy = "simulation-exploration"
	if err := validateSimulationExplorationExecutionSummary(run, map[record.SHA256]struct{}{}); err == nil {
		t.Fatal("accepted divergence for simulation strategy")
	}
}
