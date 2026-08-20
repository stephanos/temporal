package campaignstore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"go.temporal.io/server/tools/gomadv3/evidence"
)

func TestCampaignLifecycleProgressesToPublished(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-lifecycle", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	})
	assertLifecycle(t, journal.Path(), LifecyclePlanned, LifecyclePlanned, false)

	if err := journal.BeginPreparation(); err != nil {
		t.Fatal(err)
	}
	if err := journal.CompletePreparation(); err != nil {
		t.Fatal(err)
	}
	assertLifecycle(t, journal.Path(), LifecyclePrepared, LifecyclePrepared, false)

	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	assertLifecycle(t, journal.Path(), LifecycleRunning, LifecycleRunning, false)

	if err := journal.AppendExecution(ExecutionRecord{
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit",
	}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Publish(CampaignSummary{Attempted: 1, Succeeded: 1, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	assertLifecycle(t, journal.Path(), LifecyclePublished, LifecyclePublished, false)
}

func TestCampaignLifecycleRetainsRecoverableFailureProvenance(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-recoverable", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := journal.BeginPreparation(); err != nil {
		t.Fatal(err)
	}
	if err := journal.CompletePreparation(); err != nil {
		t.Fatal(err)
	}
	if err := journal.Fail("target_preparation", errors.New("build failed")); err != nil {
		t.Fatal(err)
	}

	status, err := InspectCampaignLifecycle(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if status.State != LifecycleRecoverableFailure || status.LastStableState != LifecyclePrepared || status.Reason != "target_preparation" || status.Detail != "build failed" || status.Published || status.Resumable {
		t.Fatalf("lifecycle status = %#v", status)
	}
}

func TestCampaignLifecycleRejectsIllegalTransitionWithoutMutation(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-illegal-transition", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	})
	path := filepath.Join(journal.Path(), ".partial", "batch", "partial.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.transitionLifecycle(LifecycleRunning, "", nil); err == nil {
		t.Fatal("transitionLifecycle() accepted planned -> running")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("rejected transition changed lifecycle from %q to %q", before, after)
	}
}

func TestCampaignLifecycleBoundsFailureEvidence(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-bounded-failure", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := journal.Fail(strings.Repeat("r", maximumLifecycleText+1), nil); err == nil {
		t.Fatal("Fail() accepted an oversized reason")
	}
	if err := journal.Fail("storage_failure", errors.New(strings.Repeat("é", maximumLifecycleText))); err != nil {
		t.Fatal(err)
	}
	status, err := InspectCampaignLifecycle(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if len(status.Detail) > maximumLifecycleText || !utf8.ValidString(status.Detail) {
		t.Fatalf("bounded lifecycle detail is invalid: bytes=%d", len(status.Detail))
	}
}

func TestCampaignLifecycleReadsLegacyInterruptedRecord(t *testing.T) {
	journal := newPreparedLifecycleJournal(t, "run-legacy-lifecycle")
	legacy, err := evidence.CanonicalJSON(struct {
		SchemaVersion uint32  `json:"schema_version"`
		State         string  `json:"state"`
		Reason        *string `json:"reason"`
		Detail        *string `json:"detail"`
	}{SchemaVersion: evidence.SchemaVersion, State: "running"})
	if err != nil {
		t.Fatal(err)
	}
	if err := atomicWrite(filepath.Join(journal.Path(), ".partial", "batch", "partial.json"), legacy); err != nil {
		t.Fatal(err)
	}
	preflight, err := PreflightResume(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if preflight.Lifecycle.State != LifecycleRunning || preflight.Lifecycle.LastStableState != LifecycleRunning || !preflight.Lifecycle.Resumable {
		t.Fatalf("legacy lifecycle = %#v", preflight.Lifecycle)
	}
}

func TestCampaignLifecycleRejectsCorruptState(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-corrupt-lifecycle", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := atomicWrite(filepath.Join(journal.Path(), ".partial", "batch", "partial.json"), []byte("{}")); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectCampaignLifecycle(journal.Path()); err == nil {
		t.Fatal("InspectCampaignLifecycle() accepted corrupt lifecycle state")
	} else if !IsIntegrityError(err) {
		t.Fatalf("InspectCampaignLifecycle() error = %T %v, want integrity error", err, err)
	}
}

func TestRecoverCampaignClassifiesNonRecoverableStateAsIntegrityError(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-non-recoverable", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	})

	if _, err := RecoverCampaign(context.Background(), journal.Path()); err == nil {
		t.Fatal("RecoverCampaign() accepted non-recoverable planned state")
	} else if !IsIntegrityError(err) {
		t.Fatalf("RecoverCampaign() error = %T %v, want integrity error", err, err)
	}
}

func TestRecoverCampaignPreservesContextClassification(t *testing.T) {
	journal := newPreparedLifecycleJournal(t, "run-cancelled-recovery")
	for _, test := range []struct {
		name string
		ctx  context.Context
		want error
	}{
		{name: "cancelled", ctx: cancelledContext(), want: context.Canceled},
		{name: "deadline", ctx: expiredContext(), want: context.DeadlineExceeded},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := RecoverCampaign(test.ctx, journal.Path()); !errors.Is(err, test.want) {
				t.Fatalf("RecoverCampaign() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestRecoverCampaignFinishesPublishedCleanup(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-published-recovery", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendExecution(ExecutionRecord{
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit",
	}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Publish(CampaignSummary{Attempted: 1, Succeeded: 1, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{journal.PreparedPath(), filepath.Join(journal.Path(), ".partial", "batch")} {
		if err := makePrivateDirectories(path); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(journal.PreparedPath(), "stale"), []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}

	before, err := InspectCampaignLifecycle(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if before.State != LifecyclePublished || !before.Published || !before.Repairable || before.Action != RecoveryFinalizePublication {
		t.Fatalf("published lifecycle before recovery = %#v", before)
	}
	recovered, err := RecoverCampaign(context.Background(), journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if !recovered.Changed || recovered.Action != RecoveryFinalizePublication || recovered.After.State != LifecyclePublished || recovered.After.Repairable {
		t.Fatalf("recovery result = %#v", recovered)
	}
	for _, path := range []string{journal.PreparedPath(), filepath.Join(journal.Path(), ".partial", "batch")} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("stale published state remains at %s: %v", path, err)
		}
	}
}

func TestRecoverCampaignNormalizesInterruptedCommit(t *testing.T) {
	journal := newPreparedLifecycleJournal(t, "run-commit-recovery")
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendExecution(ExecutionRecord{
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit",
	}); err != nil {
		t.Fatal(err)
	}
	if err := journal.transitionLifecycle(LifecycleCommitting, "", nil); err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}

	before, err := InspectCampaignLifecycle(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if before.State != LifecycleCommitting || !before.Repairable || before.Action != RecoveryRestoreRunning || before.Resumable {
		t.Fatalf("committing lifecycle before recovery = %#v", before)
	}
	recovered, err := RecoverCampaign(context.Background(), journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if !recovered.Changed || recovered.Action != RecoveryRestoreRunning || recovered.After.State != LifecycleRecoverableFailure || recovered.After.LastStableState != LifecycleRunning || !recovered.After.Resumable {
		t.Fatalf("recovery result = %#v", recovered)
	}
	resumed, _, err := ResumeCampaignJournal(context.Background(), journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if err := resumed.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestResumeCampaignJournalRepairsInterruptedCommitUnderLock(t *testing.T) {
	journal := newPreparedLifecycleJournal(t, "run-resume-commit")
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendExecution(ExecutionRecord{
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit",
	}); err != nil {
		t.Fatal(err)
	}
	if err := journal.transitionLifecycle(LifecycleCommitting, "", nil); err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}

	resumed, state, err := ResumeCampaignJournal(context.Background(), journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Runs) != 1 || state.Runs[0].Seed != 7 {
		t.Fatalf("resume state = %#v", state)
	}
	status, err := InspectCampaignLifecycle(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if status.State != LifecycleRunning || !status.Resumable {
		t.Fatalf("resumed lifecycle = %#v", status)
	}
	if err := resumed.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPreflightResumeReturnsValidatedLifecycleAndPlan(t *testing.T) {
	journal := newPreparedLifecycleJournal(t, "run-resume-preflight")
	preflight, err := PreflightResume(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if preflight.Lifecycle.State != LifecyclePrepared || !preflight.Lifecycle.Resumable {
		t.Fatalf("preflight lifecycle = %#v", preflight.Lifecycle)
	}
	if preflight.Plan.Selection != "7-9" || preflight.Plan.SelectionCount != 3 {
		t.Fatalf("preflight plan = %#v", preflight.Plan)
	}
}

func TestPreflightResumeRejectsUnrecoverableLifecycle(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-unrecoverable-preflight", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	})
	if _, err := PreflightResume(journal.Path()); err == nil {
		t.Fatal("PreflightResume() accepted a planned batch")
	} else if !IsIntegrityError(err) {
		t.Fatalf("PreflightResume() error = %T %v, want integrity error", err, err)
	}
}

func TestRecoverCampaignRejectsIdentitylessPlannedStateWithoutMutation(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-unrecoverable", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	})
	before, err := os.ReadFile(filepath.Join(journal.Path(), ".partial", "batch", "partial.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RecoverCampaign(context.Background(), journal.Path()); err == nil {
		t.Fatal("RecoverCampaign() accepted a planned batch without a prepared identity")
	}
	after, err := os.ReadFile(filepath.Join(journal.Path(), ".partial", "batch", "partial.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("rejected recovery changed lifecycle from %q to %q", before, after)
	}
}

func TestRecoverAndResumeShareExclusiveLock(t *testing.T) {
	journal := newPreparedLifecycleJournal(t, "run-exclusive-recovery")
	lock, err := acquireResumeLock(context.Background(), filepath.Join(journal.Path(), ".resume.lock"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := releaseResumeLock(lock); err != nil {
			t.Error(err)
		}
	}()

	if _, err := RecoverCampaign(context.Background(), journal.Path()); err == nil || !strings.Contains(err.Error(), "already being resumed") {
		t.Fatalf("RecoverCampaign() error = %v, want lock contention", err)
	}
	if _, _, err := ResumeCampaignJournal(context.Background(), journal.Path()); err == nil || !strings.Contains(err.Error(), "already being resumed") {
		t.Fatalf("ResumeCampaignJournal() error = %v, want lock contention", err)
	}
}

func newPreparedLifecycleJournal(t *testing.T, id string) *CampaignJournal {
	t.Helper()
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: id, Selection: "7-9", SelectionCount: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.BeginPreparation(); err != nil {
		t.Fatal(err)
	}
	preparedPath := filepath.Join(journal.PreparedPath(), "build", "target")
	if err := os.MkdirAll(filepath.Dir(preparedPath), 0o700); err != nil {
		t.Fatal(err)
	}
	target := []byte("prepared target")
	if err := os.WriteFile(preparedPath, target, 0o500); err != nil {
		t.Fatal(err)
	}
	if err := journal.RecordPlan(testBatchPlan(journal, evidence.HashBytes(target), uint64(len(target)))); err != nil {
		t.Fatal(err)
	}
	if err := journal.CompletePreparation(); err != nil {
		t.Fatal(err)
	}
	return journal
}

func assertLifecycle(t *testing.T, path string, state, stable LifecycleState, resumable bool) {
	t.Helper()
	status, err := InspectCampaignLifecycle(path)
	if err != nil {
		t.Fatal(err)
	}
	if status.State != state || status.LastStableState != stable || status.Resumable != resumable {
		t.Fatalf("lifecycle status = %#v", status)
	}
}

func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func expiredContext() context.Context {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	cancel()
	return ctx
}
