package campaignstore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomadv3/evidence"
)

func TestSegmentedJournalRotatesAndPublishesImmutableIndex(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-segmented", Selection: "7-9", SelectionCount: 3,
		Journal: RunJournalLimits{
			MaximumRuns: 3, MaximumBytes: 12 << 10, SegmentBytes: 4 << 10,
			SegmentRecords: 1, MaximumSegments: 3, MaximumPartialRuns: 1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	for ordinal, seed := range []uint64{7, 8, 9} {
		if err := journal.AppendExecution(successExecution(uint64(ordinal), seed)); err != nil {
			t.Fatal(err)
		}
	}
	if err := journal.Publish(CampaignSummary{Attempted: 3, Succeeded: 3, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(journal.Path(), "runs.jsonl")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy runs journal exists: %v", err)
	}

	indexBytes, err := os.ReadFile(filepath.Join(journal.Path(), "runs", "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index runJournalIndex
	if err := evidence.DecodeCanonicalJSON(indexBytes, &index); err != nil {
		t.Fatal(err)
	}
	if index.Schema != runJournalSchema || index.Records != 3 || len(index.Segments) != 3 {
		t.Fatalf("journal index = %#v", index)
	}
	for segmentIndex, segment := range index.Segments {
		wantName := fmt.Sprintf("%020d.jsonl", segmentIndex)
		if segment.File != wantName || segment.Records != 1 || segment.Bytes == 0 || !validRecordSHA256(segment.SHA256) {
			t.Fatalf("segment %d = %#v", segmentIndex, segment)
		}
	}

	batchBytes, err := os.ReadFile(filepath.Join(journal.Path(), "batch.json"))
	if err != nil {
		t.Fatal(err)
	}
	var batch CampaignRecord
	if err := evidence.DecodeCanonicalJSON(batchBytes, &batch); err != nil {
		t.Fatal(err)
	}
	if batch.Schema != "gomadv3.batch/v3" || batch.Journal == nil || batch.Journal.IndexSHA256 != evidence.HashBytes(indexBytes) || batch.RunsSHA256 != "" {
		t.Fatalf("batch journal identity = %#v", batch)
	}
	opened, err := OpenCampaign(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if len(opened.Runs) != 3 || opened.Runs[0].Seed != 7 || opened.Runs[1].Seed != 8 || opened.Runs[2].Seed != 9 {
		t.Fatalf("opened runs = %#v", opened.Runs)
	}

	changed := filepath.Join(journal.Path(), "runs", index.Segments[1].File)
	if err := os.WriteFile(changed, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenCampaign(journal.Path()); err == nil || !strings.Contains(err.Error(), index.Segments[1].File) {
		t.Fatalf("OpenCampaign() error = %v", err)
	}
}

func TestSegmentedJournalRejectsOversizeRecordWithoutPublishingBytes(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-segment-capacity", Selection: "7", SelectionCount: 1,
		Journal: RunJournalLimits{
			MaximumRuns: 1, MaximumBytes: 512, SegmentBytes: 512,
			SegmentRecords: 1, MaximumSegments: 1, MaximumPartialRuns: 1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	run := successExecution(0, 7)
	run.Reason = strings.Repeat("x", 512)
	err = journal.AppendExecution(run)
	var capacityErr *JournalCapacityError
	if !errors.As(err, &capacityErr) || capacityErr.Limit != JournalLimitSegmentBytes || capacityErr.Outcome != CapacityInfrastructureFailure {
		t.Fatalf("AppendExecution() error = %#v", err)
	}
	entries, err := os.ReadDir(filepath.Join(journal.Path(), "runs"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "index.json" {
		t.Fatalf("published journal entries after rejected append = %#v", entries)
	}
	partials, err := os.ReadDir(filepath.Join(journal.Path(), ".partial", "runs"))
	if err != nil {
		t.Fatal(err)
	}
	if len(partials) != 0 {
		t.Fatalf("partial journal entries after rejected append = %#v", partials)
	}
}

func TestResumeSegmentedJournalRecoversContiguousOrphanSegment(t *testing.T) {
	journal := newPreparedLifecycleJournal(t, "run-orphan-segment")
	failIndex := false
	journal.ctx = withMutationHook(context.Background(), func(point mutationPoint) error {
		if failIndex && point.Kind == mutationCreate && point.Operation == "atomic-temporary" {
			return errors.New("injected index publication failure")
		}
		return nil
	})
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendExecution(successExecution(0, 7)); err != nil {
		t.Fatal(err)
	}
	failIndex = true
	if err := journal.segmentedRuns.seal(); err == nil {
		t.Fatal("seal() succeeded despite the injected index failure")
	}
	failIndex = false
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}

	resumed, state, err := ResumeCampaignJournal(context.Background(), journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := resumed.Close(); err != nil {
			t.Error(err)
		}
	})
	if len(state.Runs) != 1 || state.Runs[0].Seed != 7 {
		t.Fatalf("resume state = %#v", state)
	}
	if err := resumed.AppendExecution(successExecution(1, 8)); err != nil {
		t.Fatal(err)
	}
	if err := resumed.Publish(CampaignSummary{Attempted: 2, Succeeded: 2, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	opened, err := OpenCampaign(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if len(opened.Runs) != 2 || opened.Runs[0].Seed != 7 || opened.Runs[1].Seed != 8 {
		t.Fatalf("published runs = %#v", opened.Runs)
	}
}

func TestResumeSegmentedJournalExcludesTornActiveTail(t *testing.T) {
	journal := newPreparedLifecycleJournal(t, "run-torn-active")
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendExecution(successExecution(0, 7)); err != nil {
		t.Fatal(err)
	}
	activePath := journal.segmentedRuns.activePath
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	active, err := os.OpenFile(activePath, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := active.Write([]byte(`{"selection_ordinal":`)); err != nil {
		active.Close()
		t.Fatal(err)
	}
	if err := active.Close(); err != nil {
		t.Fatal(err)
	}

	resumed, state, err := ResumeCampaignJournal(context.Background(), journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := resumed.Close(); err != nil {
			t.Error(err)
		}
	})
	if len(state.Runs) != 1 || state.Runs[0].Seed != 7 {
		t.Fatalf("resume state = %#v", state)
	}
	archives, err := filepath.Glob(filepath.Join(journal.Path(), ".partial", "resume", "*", "partials", "runs", filepath.Base(activePath)))
	if err != nil || len(archives) != 1 {
		t.Fatalf("archived active journal = %v, %v", archives, err)
	}
	archived, err := os.ReadFile(archives[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(archived), `{"selection_ordinal":`) {
		t.Fatalf("archived active journal lost its torn tail: %q", archived)
	}
}

func TestOpenSegmentedJournalRejectsUnindexedSegment(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-unindexed", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendExecution(successExecution(0, 7)); err != nil {
		t.Fatal(err)
	}
	if err := journal.Publish(CampaignSummary{Attempted: 1, Succeeded: 1, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	extra := filepath.Join(journal.Path(), "runs", "00000000000000000001.jsonl")
	if err := os.WriteFile(extra, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenCampaign(journal.Path()); err == nil || !IsIntegrityError(err) {
		t.Fatalf("OpenCampaign() error = %v, want integrity error", err)
	}
}

func TestCampaignJournalEnforcesPartialRunCapacityBeforeMutation(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-partial-capacity", Selection: "7-8", SelectionCount: 2,
		Journal: RunJournalLimits{
			MaximumRuns: 2, MaximumBytes: 4 << 10, SegmentBytes: 2 << 10,
			SegmentRecords: 2, MaximumSegments: 2, MaximumPartialRuns: 1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	})
	first, err := journal.BeginExecution(0, 7)
	if err != nil {
		t.Fatal(err)
	}
	_, err = journal.BeginExecution(1, 8)
	var capacityErr *JournalCapacityError
	if !errors.As(err, &capacityErr) || capacityErr.Limit != JournalLimitPartialRuns {
		t.Fatalf("BeginExecution() error = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(journal.Path(), ".partial", "00000000000000000001-8")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("second partial exists after rejected reservation: %v", err)
	}
	for _, next := range []ExecutionState{ExecutionStarting, ExecutionExited, ExecutionCaptured, ExecutionClassified} {
		if err := first.Transition(next); err != nil {
			t.Fatal(err)
		}
	}
	if err := first.Complete(); err != nil {
		t.Fatal(err)
	}
	if _, err := journal.BeginExecution(1, 8); err != nil {
		t.Fatalf("BeginExecution() after release: %v", err)
	}
}

func TestSegmentedJournalOpensBeyondLegacyJournalBound(t *testing.T) {
	const runs = 65
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-large-segmented", Selection: "generated", SelectionCount: runs,
		Journal: RunJournalLimits{
			MaximumRuns: runs, MaximumBytes: 80 << 20, SegmentBytes: 2 << 20,
			SegmentRecords: 1, MaximumSegments: runs, MaximumPartialRuns: 1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	reason := strings.Repeat("x", 1<<20)
	for ordinal := uint64(0); ordinal < runs; ordinal++ {
		run := successExecution(ordinal, ordinal+1)
		run.Reason = reason
		if err := journal.AppendExecution(run); err != nil {
			t.Fatal(err)
		}
	}
	if err := journal.Publish(CampaignSummary{Attempted: runs, Succeeded: runs, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	opened, err := OpenCampaign(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if opened.Journal == nil || opened.Journal.Bytes <= maximumRunsBytes || len(opened.Runs) != runs {
		t.Fatalf("opened large journal = %#v, runs = %d", opened.Journal, len(opened.Runs))
	}
}

func TestResumeCampaignJournalRetainsHistoricalPlanSchemas(t *testing.T) {
	for _, schema := range []string{LegacyBatchPlanSchema, OlderCampaignPlanSchema, EarlierCampaignPlanSchema, PriorCampaignPlanSchema, PreviousCampaignPlanSchema} {
		t.Run(schema, func(t *testing.T) {
			journal := newPreparedLifecycleJournal(t, "run-historical-resume")
			if err := journal.StartExecutions(); err != nil {
				t.Fatal(err)
			}
			if err := journal.AppendExecution(successExecution(0, 7)); err != nil {
				t.Fatal(err)
			}
			activePath := journal.segmentedRuns.activePath
			if err := journal.Close(); err != nil {
				t.Fatal(err)
			}
			segmented := schema == PriorCampaignPlanSchema || schema == PreviousCampaignPlanSchema
			if !segmented {
				runs, err := os.ReadFile(activePath)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(journal.Path(), "runs.jsonl"), runs, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.RemoveAll(filepath.Join(journal.Path(), "runs")); err != nil {
					t.Fatal(err)
				}
				if err := os.RemoveAll(filepath.Join(journal.Path(), ".partial", "runs")); err != nil {
					t.Fatal(err)
				}
			}

			planBytes, err := os.ReadFile(filepath.Join(journal.PreparedPath(), "plan.json"))
			if err != nil {
				t.Fatal(err)
			}
			var plan CampaignPlan
			if err := evidence.DecodeCanonicalJSON(planBytes, &plan); err != nil {
				t.Fatal(err)
			}
			plan.Schema = schema
			if !segmented {
				plan.Journal = nil
				plan.Artifacts = nil
			}
			if schema == LegacyBatchPlanSchema {
				plan.Strategy = ""
			}
			if schema == LegacyBatchPlanSchema || schema == OlderCampaignPlanSchema {
				plan.Prepared.Target.CapabilityMode = ""
				plan.Prepared.Target.CapabilityManifest = nil
			}
			encoded, err := evidence.CanonicalJSON(plan)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(journal.PreparedPath(), "plan.json"), encoded, 0o600); err != nil {
				t.Fatal(err)
			}

			resumed, state, err := ResumeCampaignJournal(context.Background(), journal.Path())
			if err != nil {
				t.Fatal(err)
			}
			if len(state.Runs) != 1 || state.Runs[0].Seed != 7 {
				t.Fatalf("resume state = %#v", state)
			}
			if err := resumed.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func successExecution(ordinal, seed uint64) ExecutionRecord {
	return ExecutionRecord{
		SelectionOrdinal: evidence.Uint64String(ordinal), Seed: evidence.Uint64String(seed),
		Domain: "success", Reason: "success", Termination: "exit",
	}
}
