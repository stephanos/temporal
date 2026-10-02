package campaign

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
	choiceengine "go.temporal.io/server/tools/gomad3/runner/internal/exploration/choice"
)

// The values below and testdata/pre-start-ordinal-journal were produced by the
// Runner before choice exploration had a start ordinal. The root segment was
// re-encoded through CommitRound when the choice tape header moved to v3,
// which changed its trace digest, its identity, and the after-state identity
// through the children's prefix bytes; the decisions and candidates are as
// recorded.
const (
	retainedInitialStateSHA256      record.SHA256 = "sha256:32e05cff3064bd1c6d2f76b178b48eb596076514ddea276dd1dd31825316845a"
	retainedAfterRootStateSHA256    record.SHA256 = "sha256:5a9d06998c19a945ad1cf3b1566f66775fa74f9dd67d866c548a1d3ad8259737"
	retainedChoicePlanCanonicalHash record.SHA256 = "sha256:a59870db6176bca22c6f4c4cfe2d67779e9e25b3a8b2c6bc8958cee8e37a4ea0"
)

func TestDefaultChoiceStartKeepsRetainedIdentityAndPlanBytes(t *testing.T) {
	state := testExplorationState(t)
	identity, err := choiceengine.StateSHA256(state)
	if err != nil {
		t.Fatal(err)
	}
	if identity != retainedInitialStateSHA256 {
		t.Fatalf("initial state identity = %s, want %s", identity, retainedInitialStateSHA256)
	}
	batchPath := privateDirectory(t)
	if _, err := NewExplorationJournal(context.Background(), batchPath, state, 1<<20); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(batchPath, "choice-exploration", "plan.json"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "pre-start-ordinal-journal", "choice-exploration", "plan.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("exploration plan = %s, want %s", got, want)
	}
	encoded, err := canonicaljson.CanonicalJSON(testChoiceExplorationCampaignPlan())
	if err != nil {
		t.Fatal(err)
	}
	if digest := record.HashBytes(encoded); digest != retainedChoicePlanCanonicalHash {
		t.Fatalf("campaign plan digest = %s for %s, want %s", digest, encoded, retainedChoicePlanCanonicalHash)
	}
}

func TestResumeExplorationJournalWrittenBeforeTheStartOrdinal(t *testing.T) {
	batchPath := copyRetainedExplorationJournal(t)
	journal, state, recovery, err := ResumeExplorationJournal(context.Background(), batchPath, testExplorationState(t).Config, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := choiceengine.StateSHA256(state)
	if err != nil {
		t.Fatal(err)
	}
	if identity != retainedAfterRootStateSHA256 || recovery != 0 || len(journal.CommittedExecutions()) != 1 {
		t.Fatalf("resumed state = %s, recovery = %d, executions = %d", identity, recovery, len(journal.CommittedExecutions()))
	}
}

func TestResumeExplorationJournalRequiresItsStartOrdinal(t *testing.T) {
	config := testExplorationState(t).Config
	config.StartOrdinal = 1
	state, err := choiceengine.New(config)
	if err != nil {
		t.Fatal(err)
	}
	batchPath := privateDirectory(t)
	if _, err := NewExplorationJournal(context.Background(), batchPath, state, 1<<20); err != nil {
		t.Fatal(err)
	}
	for _, start := range []uint64{0, 2} {
		changed := config
		changed.StartOrdinal = start
		if _, _, _, err := ResumeExplorationJournal(context.Background(), batchPath, changed, 1<<20); err == nil || !strings.Contains(err.Error(), "exploration plan identity or bounds changed") {
			t.Fatalf("ResumeExplorationJournal(start %d) error = %v", start, err)
		}
	}
	_, resumed, _, err := ResumeExplorationJournal(context.Background(), batchPath, config, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Config.StartOrdinal != 1 {
		t.Fatalf("resumed start ordinal = %d", resumed.Config.StartOrdinal)
	}
	retained := copyRetainedExplorationJournal(t)
	if _, _, _, err := ResumeExplorationJournal(context.Background(), retained, config, 1<<20); err == nil {
		t.Fatal("ResumeExplorationJournal() accepted a start ordinal its journal was not written with")
	}
}

func TestSeedCampaignPlanRejectsChoiceStartOrdinal(t *testing.T) {
	plan := testBatchPlan(nil, record.HashBytes([]byte("prepared target")), 15)
	journalPlan := recordExecutionJournalLimits(ExecutionJournalLimits{
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
	plan.ChoiceExplorationImplementationSHA256 = choiceengine.ImplementationSHA256()
	return plan
}

func copyRetainedExplorationJournal(t *testing.T) string {
	t.Helper()
	batchPath := copyRetainedRecords(t, "pre-start-ordinal-journal")
	if err := os.MkdirAll(filepath.Join(batchPath, ".partial", "choice-exploration"), 0o700); err != nil {
		t.Fatal(err)
	}
	return batchPath
}

// copyRetainedRecords copies a testdata directory into a private directory with
// the modes the Runner publishes records with, which a checkout does not keep.
func copyRetainedRecords(t *testing.T, name string) string {
	t.Helper()
	batchPath := privateDirectory(t)
	source := filepath.Join("testdata", name)
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil || relative == "." {
			return err
		}
		destination := filepath.Join(batchPath, relative)
		if entry.IsDir() {
			return os.Mkdir(destination, 0o700)
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, contents, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return batchPath
}
