package exploration

import (
	"crypto/sha256"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/record"
)

func divergenceRound(t *testing.T, policy FailurePolicy) (ChoiceState, ChoiceRound, choice.Divergence) {
	t.Helper()
	config := choiceTestConfig()
	config.Parallel, config.FailurePolicy = 3, policy
	if policy == PolicyBudget {
		config.FailureBudget = 2
	}
	state, err := NewChoice(config)
	if err != nil {
		t.Fatal(err)
	}
	root, _ := state.NextRound()
	trace := testTape(t, config.Execution, choiceTestDecision(t, 0, choice.KindRunnable, 5, 0))
	state, _, err = CommitChoiceRound(state, root, []ChoiceResult{testResult(root.Candidates[0], trace, "root")})
	if err != nil {
		t.Fatal(err)
	}
	round, _ := state.NextRound()
	prefix, err := round.Candidates[0].PrefixReplayPlan(config.Execution)
	if err != nil {
		t.Fatal(err)
	}
	expected := prefix.Decisions[0]
	observed := choiceTestDecision(t, 0, choice.KindRunnable, 5, expected.Selected)
	observed.AlternativeSetDigest = sha256.Sum256([]byte("changed alternatives"))
	return state, round, choice.Divergence{Ordinal: 0, Reason: choice.DivergenceAlternativeSet, Expected: &expected, Observed: &observed, TapeRecords: 1}
}

func TestExplorationCommitsDivergenceAndSiblingsUnderEveryPolicy(t *testing.T) {
	for _, policy := range []FailurePolicy{PolicyFirst, PolicyBudget, PolicyAll} {
		t.Run(string(policy), func(t *testing.T) {
			state, round, divergence := divergenceRound(t, policy)
			failure := record.HashBytes([]byte("failure"))
			results := []ChoiceResult{
				{CandidateSHA256: round.Candidates[0].SHA256, Divergence: &divergence},
				{CandidateSHA256: round.Candidates[1].SHA256, OutcomeSHA256: record.HashBytes([]byte("target failure")), Failed: true, FailureSHA256: failure},
				{CandidateSHA256: round.Candidates[2].SHA256, OutcomeSHA256: record.HashBytes([]byte("success"))},
			}
			next, segment, err := CommitChoiceRound(state, round, results)
			if err != nil {
				t.Fatal(err)
			}
			if next.LogicalExecutions != 4 || next.CommittedRounds != 2 || len(next.Outcomes) != 3 || !reflect.DeepEqual(next.FailureSignatures, []record.SHA256{failure}) || len(next.Queue) != 1 || len(next.Seen) != 5 {
				t.Fatalf("committed divergence = %#v", next)
			}
			if got := segment.Results[0]; got.OutcomeSHA256 != "" || got.Failed || got.FailureSHA256 != "" || len(got.TraceBytes) != 0 || !reflect.DeepEqual(got.Divergence.Divergence(), divergence) {
				t.Fatalf("divergent result = %#v", got)
			}
			_, available := next.NextRound()
			if policy == PolicyFirst && (available || next.StopReason != ChoiceStopFirstFailure) || policy != PolicyFirst && (!available || next.StopReason != "") {
				t.Fatalf("policy %s: %#v", policy, next)
			}
			replayed, err := ChoiceReplaySegment(state, segment)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(replayed, next) {
				t.Fatalf("replay = %#v, want %#v", replayed, next)
			}
			divergence.Observed.Alternatives++
			if segment.Results[0].Divergence.Observed.Alternatives != 5 {
				t.Fatal("segment aliases executor divergence")
			}
		})
	}
}

func TestExplorationRejectsInvalidCandidateDivergence(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*ChoiceResult)
	}{
		{"outcome", func(r *ChoiceResult) { r.OutcomeSHA256 = record.HashBytes([]byte("outcome")) }},
		{"failed", func(r *ChoiceResult) { r.Failed = true }},
		{"signature", func(r *ChoiceResult) { r.FailureSHA256 = record.HashBytes([]byte("signature")) }},
		{"trace", func(r *ChoiceResult) { r.Trace = &choice.ReplayPlan{} }},
		{"expected", func(r *ChoiceResult) { r.Divergence.Expected.Selected++ }},
		{"ordinal", func(r *ChoiceResult) { r.Divergence.Ordinal++ }},
		{"tape count", func(r *ChoiceResult) { r.Divergence.TapeRecords++ }},
		{"reason", func(r *ChoiceResult) { r.Divergence.Reason = choice.DivergenceSite }},
		{"missing expected", func(r *ChoiceResult) { r.Divergence.Expected = nil }},
		{"missing observed", func(r *ChoiceResult) { r.Divergence.Observed = nil }},
		{"observation beyond prefix boundary", func(r *ChoiceResult) {
			r.Divergence.Reason = choice.DivergenceObservation
			r.Divergence.Ordinal = 2
			r.Divergence.Expected = nil
			r.Divergence.Observed.Ordinal = 2
		}},
		{"observation at prefix boundary", func(r *ChoiceResult) {
			r.Divergence.Reason = choice.DivergenceObservation
			r.Divergence.Ordinal = 1
			r.Divergence.Expected = nil
			r.Divergence.Observed.Ordinal = 1
		}},
		{"observed ordinal", func(r *ChoiceResult) { r.Divergence.Observed.Ordinal++ }},
		{"rank override", func(r *ChoiceResult) { r.Divergence.Observed.RankOverride = true }},
		{"observed kind", func(r *ChoiceResult) { r.Divergence.Observed.Kind = 0 }},
		{"observed count", func(r *ChoiceResult) { r.Divergence.Observed.Alternatives = 0 }},
		{"observed selection", func(r *ChoiceResult) { r.Divergence.Observed.Selected = r.Divergence.Observed.Alternatives }},
		{"observed identity", func(r *ChoiceResult) { r.Divergence.Observed.SelectedIdentity = [32]byte{} }},
		{"observed digest", func(r *ChoiceResult) { r.Divergence.Observed.AlternativeSetDigest = [32]byte{} }},
		{"observed site", func(r *ChoiceResult) { r.Divergence.Observed.SiteMissing = true; r.Divergence.Observed.SiteOffset = 1 }},
		{"expected kind", func(r *ChoiceResult) { r.Divergence.Expected.Kind = 0 }},
		{"expected count", func(r *ChoiceResult) { r.Divergence.Expected.Alternatives = 0 }},
		{"expected rank override", func(r *ChoiceResult) { r.Divergence.Expected.RankOverride = false }},
		{"missing identity", func(r *ChoiceResult) {
			r.Divergence.Reason = choice.DivergenceIdentityMissing
			r.Divergence.Observed = nil
		}},
		{"duplicate identity", func(r *ChoiceResult) {
			r.Divergence.Reason = choice.DivergenceIdentityDuplicate
			r.Divergence.Observed = nil
		}},
		{"alternative capacity", func(r *ChoiceResult) {
			r.Divergence.Reason = choice.DivergenceAlternativeCapacity
			r.Divergence.Observed = nil
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, round, divergence := divergenceRound(t, PolicyAll)
			result := ChoiceResult{CandidateSHA256: round.Candidates[0].SHA256, Divergence: &divergence}
			test.change(&result)
			results := []ChoiceResult{result, {CandidateSHA256: round.Candidates[1].SHA256, OutcomeSHA256: record.HashBytes([]byte("success"))}, {CandidateSHA256: round.Candidates[2].SHA256, OutcomeSHA256: record.HashBytes([]byte("success"))}}
			if _, _, err := CommitChoiceRound(state, round, results); err == nil {
				t.Fatal("accepted invalid divergence")
			}
		})
	}
	state, _, divergence := divergenceRound(t, PolicyAll)
	initial, err := NewChoice(state.Config)
	if err != nil {
		t.Fatal(err)
	}
	root, _ := initial.NextRound()
	if _, _, err := CommitChoiceRound(initial, root, []ChoiceResult{{CandidateSHA256: root.Candidates[0].SHA256, Divergence: &divergence}}); err == nil {
		t.Fatal("accepted root divergence")
	}
}

func TestExplorationRejectsPreviousSegmentSchemaVisibly(t *testing.T) {
	state, round, divergence := divergenceRound(t, PolicyAll)
	results := []ChoiceResult{{CandidateSHA256: round.Candidates[0].SHA256, Divergence: &divergence}, {CandidateSHA256: round.Candidates[1].SHA256, OutcomeSHA256: record.HashBytes([]byte("success"))}, {CandidateSHA256: round.Candidates[2].SHA256, OutcomeSHA256: record.HashBytes([]byte("success"))}}
	_, segment, err := CommitChoiceRound(state, round, results)
	if err != nil {
		t.Fatal(err)
	}
	segment.Schema = "gomad3.choice-exploration-round/v1"
	if _, err := ChoiceReplaySegment(state, segment); err == nil || !strings.Contains(err.Error(), "schema") {
		t.Fatalf("old schema error = %v", err)
	}
}
