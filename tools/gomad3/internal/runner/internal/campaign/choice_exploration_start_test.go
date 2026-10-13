package campaign

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/runner/internal/exploration"
)

// The values below were produced by the Runner before choice exploration had
// a start ordinal, and re-derived when the choice tape header moved to v3
// and when the controller identity moved to v3 for the select-poll
// reduction. The initial state identity was re-derived when choice state
// projections moved to encoding/json. A default start ordinal keeps that
// identity. World binary snapshots add a 64-byte envelope to each of the two
// snapshots reserved per Artifact, rotating the plan's capacity and JSON hash.
const (
	retainedInitialStateSHA256 record.SHA256 = "sha256:bfa322bbbdacdf4ff0e9752fc0b8721e798fdfb6ad43013e047b419d899d4738"
	retainedChoicePlanJSONHash record.SHA256 = "sha256:fd169c192d69477d5dd2f6463e29ee4d2edcf0c67e49eb83f5d228af63b31771"
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
	encoded, err := json.Marshal(testChoiceExplorationCampaignPlan())
	if err != nil {
		t.Fatal(err)
	}
	if digest := record.HashBytes(encoded); digest != retainedChoicePlanJSONHash {
		t.Fatalf("campaign plan digest = %s for %s, want %s", digest, encoded, retainedChoicePlanJSONHash)
	}
}

func TestChoicePlanHashRotationOnlyAddsWorldEnvelopeCapacity(t *testing.T) {
	plan := testChoiceExplorationCampaignPlan()
	if plan.Artifacts.FailureArtifacts != 3 || plan.Artifacts.FailureBytes != 654315354 || plan.Artifacts.TotalBytes != plan.Artifacts.FailureBytes {
		t.Fatalf("unexpected retained capacity projection: %+v", plan.Artifacts)
	}
	prior := *plan.Artifacts
	prior.FailureBytes -= 3 * 2 * 64
	prior.TotalBytes -= 3 * 2 * 64
	plan.Artifacts = &prior
	encoded, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	const previous = "sha256:afd609dfe85609a457d678d14ee350b0f3b22f63c95f68a2e80e6af814708d65"
	if digest := record.HashBytes(encoded); digest != previous {
		t.Fatalf("plan inputs changed beyond World envelope capacity: %s", digest)
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
