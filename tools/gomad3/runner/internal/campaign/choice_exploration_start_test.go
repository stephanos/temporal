package campaign

import (
	"context"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/exploration"
)

// The values below were produced by the Runner before choice exploration had
// a start ordinal, and re-derived when the choice tape header moved to v3
// and when the controller identity moved to v3 for the select-poll
// reduction. A default start ordinal keeps the initial state identity and
// the campaign plan's canonical bytes.
const (
	retainedInitialStateSHA256      record.SHA256 = "sha256:3c33957816f4bed22cba236db047973f8212092d911dbe879283774be9b84e73"
	retainedChoicePlanCanonicalHash record.SHA256 = "sha256:1c7cfef620e584c75cb5854eae44d79ec50b101717d42df44adc39a96893a76d"
)

func TestDefaultChoiceStartKeepsRetainedIdentityAndPlanBytes(t *testing.T) {
	state := testExplorationState(t)
	identity, err := exploration.ChoiceStateSHA256(state)
	if err != nil {
		t.Fatal(err)
	}
	if identity != retainedInitialStateSHA256 {
		t.Fatalf("initial state identity = %s, want %s", identity, retainedInitialStateSHA256)
	}
	encoded, err := canonicaljson.CanonicalJSON(testChoiceExplorationCampaignPlan())
	if err != nil {
		t.Fatal(err)
	}
	if digest := record.HashBytes(encoded); digest != retainedChoicePlanCanonicalHash {
		t.Fatalf("campaign plan digest = %s for %s, want %s", digest, encoded, retainedChoicePlanCanonicalHash)
	}
}

// A resumed exploration requires the start ordinal it recorded.
func TestExplorationRoundsRequireTheirStartOrdinal(t *testing.T) {
	config := testExplorationState(t).Config
	config.StartOrdinal = 1
	journal := newExplorationCampaign(t, context.Background(), "choice-exploration")
	if _, err := OpenRounds(journal, ChoiceRounds, config, 1<<20); err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	resumed, _, err := ResumeCampaignJournal(context.Background(), journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resumed.Close(); err != nil {
			t.Error(err)
		}
	}()
	for _, start := range []uint64{0, 2} {
		changed := config
		changed.StartOrdinal = start
		if _, err := OpenRounds(resumed, ChoiceRounds, changed, 1<<20); err == nil || !strings.Contains(err.Error(), "exploration plan identity or bounds changed") {
			t.Fatalf("OpenRounds(start %d) error = %v", start, err)
		}
	}
	rounds, err := OpenRounds(resumed, ChoiceRounds, config, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if rounds.State().Config.StartOrdinal != 1 {
		t.Fatalf("resumed start ordinal = %d", rounds.State().Config.StartOrdinal)
	}
}

func TestSeedCampaignPlanRejectsChoiceStartOrdinal(t *testing.T) {
	plan := testBatchPlan(nil, record.HashBytes([]byte("prepared target")), 15)
	journalPlan := recordExecutionJournalLimits(executionJournalLimits{
		MaximumExecutions: 3, MaximumBytes: 3 << 20, SegmentBytes: 1 << 20,
		SegmentRecords: 1024, MaximumSegments: 3, MaximumPartialExecutions: 2,
	})
	plan.Journal = &journalPlan
	artifacts, err := DeriveArtifactCapacityPlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	plan.Artifacts = &artifacts
	if err := validateCampaignPlan(plan); err != nil {
		t.Fatal(err)
	}
	plan.ChoiceStartOrdinal = 3
	if err := validateCampaignPlan(plan); err == nil || !strings.Contains(err.Error(), "seed campaign plan contains exploration bounds") {
		t.Fatalf("validateCampaignPlan() error = %v", err)
	}
}

func testChoiceExplorationCampaignPlan() CampaignPlan {
	plan := testBatchPlan(nil, record.HashBytes([]byte("prepared target")), 15)
	plan.Strategy = "choice-exploration"
	plan.Selection = "7"
	plan.SelectionCount = 1
	plan.MaxExecutions = 8
	plan.MaxChoiceDepth = 4
	plan.MaxExplorationBytes = 1 << 20
	plan.ChoiceExplorationImplementationSHA256 = exploration.ChoiceImplementationSHA256()
	return plan
}
