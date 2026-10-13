package campaign

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/runner/internal/logstore"
)

// failureCampaign is a prepared seed campaign whose plan matches
// campaignArtifactInput, so its failure artifacts validate on resume.
type failureCampaign struct {
	journal *CampaignJournal
	input   artifact.ArtifactInput
}

func newFailureCampaign(t *testing.T) failureCampaign {
	t.Helper()
	return newFailureCampaignWith(t, failureCampaignOptions{selection: "7-9", count: 3, policy: "all", budget: 1})
}

type failureCampaignOptions struct {
	ctx             context.Context
	selection       string
	count, parallel uint64
	policy          string
	budget          uint64
}

func newFailureCampaignWith(t *testing.T, options failureCampaignOptions) failureCampaign {
	t.Helper()
	if options.ctx == nil {
		options.ctx = context.Background()
	}
	input := campaignArtifactInput(t)
	journal, err := NewCampaignJournal(options.ctx, CampaignConfig{
		Root: t.TempDir(), CampaignID: input.Manifest.CampaignID, Selection: options.selection, SelectionCount: options.count, Parallel: options.parallel,
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
	preparedPath := filepath.Join(journal.PreparedPath(), "build", "target")
	if err := os.MkdirAll(filepath.Dir(preparedPath), 0o700); err != nil {
		t.Fatal(err)
	}
	targetBytes, err := os.ReadFile(input.TargetPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(preparedPath, targetBytes, 0o500); err != nil {
		t.Fatal(err)
	}
	plan := testBatchPlan(journal, record.HashBytes(targetBytes), uint64(len(targetBytes)))
	plan.RunnerBuild = input.Manifest.Runner.RunnerBuild
	plan.Toolchain = input.Manifest.Toolchain
	plan.Prepared.Target = input.Manifest.Target
	plan.Selection, plan.SelectionCount = options.selection, record.Uint64String(options.count)
	plan.OnFailure, plan.FailureBudget = options.policy, record.Uint64String(options.budget)
	artifacts, err := DeriveArtifactCapacityPlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	plan.Artifacts = &artifacts
	if err := journal.RecordPlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := journal.CompletePreparation(); err != nil {
		t.Fatal(err)
	}
	input.TargetPath = preparedPath
	return failureCampaign{journal: journal, input: input}
}

// publishFailure publishes a failure artifact for one ordinal and returns the
// committed outcome that references it.
func (fixture failureCampaign) publishFailure(t *testing.T, ordinal, seed uint64, reason string) ExecutionRecord {
	t.Helper()
	input := fixture.input
	input.Manifest.SelectionOrdinal, input.Manifest.Seed = record.Uint64String(ordinal), record.Uint64String(seed)
	input.Manifest.Environment = append([]record.Environment(nil), input.Manifest.Environment...)
	input.Manifest.Environment[1].Value = strconv.FormatUint(seed, 10)
	if reason != "" {
		input.Manifest.Outcome.Reason = reason
	}
	published, err := artifact.PublishArtifact(artifact.Store{Root: fixture.journal.FailuresPath()}, input)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := filepath.Rel(fixture.journal.Path(), published.Path)
	if err != nil {
		t.Fatal(err)
	}
	signature := published.Manifest.Outcome.FailureSignature
	return ExecutionRecord{
		SelectionOrdinal: record.Uint64String(ordinal), Seed: record.Uint64String(seed), Domain: "target",
		Reason: published.Manifest.Outcome.Reason, Termination: published.Manifest.Outcome.Termination,
		FailureSignature: &signature, Artifact: &reference,
	}
}

func directoryNames(t *testing.T, path string) []string {
	t.Helper()
	entries, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

func TestResumeCampaignJournalReusesCommittedOutcomesAndReclaimsUncommittedWork(t *testing.T) {
	fixture := newFailureCampaign(t)
	journal := fixture.journal
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	committed := fixture.publishFailure(t, 0, 7, "")
	if err := journal.AppendExecution(committed); err != nil {
		t.Fatal(err)
	}
	cancelled := ExecutionRecord{SelectionOrdinal: 1, Seed: 8, Domain: "runner", Reason: "runner_cancelled", Termination: "none"}
	if err := journal.AppendExecution(cancelled); err != nil {
		t.Fatal(err)
	}
	orphan := fixture.publishFailure(t, 2, 9, "orphan_reason")
	incomplete, err := journal.BeginExecution(2, 9)
	if err != nil {
		t.Fatal(err)
	}
	if err := incomplete.Transition(ExecutionStarting); err != nil {
		t.Fatal(err)
	}
	diagnostics := filepath.Join(journal.Path(), "diagnostics")
	if err := os.MkdirAll(diagnostics, 0o700); err != nil {
		t.Fatal(err)
	}
	publicationTemporary := filepath.Join(diagnostics, ".publish-crashed")
	if err := os.WriteFile(publicationTemporary, []byte("uncommitted diagnostic"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(publicationTemporary, filepath.Join(diagnostics, "00000000000000000002-diagnostic.bin")); err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	failures := directoryNames(t, journal.FailuresPath())
	if len(failures) != 2 {
		t.Fatalf("published failures = %v", failures)
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
	if len(state.Executions) != 1 || state.Executions[0].Seed != 7 || state.Plan.Selection != "7-9" {
		t.Fatalf("resume state = %#v", state)
	}
	if remaining := directoryNames(t, journal.FailuresPath()); !slices.Equal(remaining, []string{filepath.Base(*committed.Artifact)}) {
		t.Fatalf("failures after resume = %v, want only the committed %s (orphan %s)", remaining, *committed.Artifact, *orphan.Artifact)
	}
	if _, err := os.Lstat(incomplete.Path()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("uncommitted attempt work survived resume: %v", err)
	}
	if remaining := directoryNames(t, diagnostics); len(remaining) != 0 {
		t.Fatalf("interrupted diagnostic publication survived resume: %v", remaining)
	}
	for _, ordinal := range []uint64{1, 2} {
		if err := resumed.AppendExecution(successExecution(ordinal, ordinal+7)); err != nil {
			t.Fatal(err)
		}
	}
	signature := *committed.FailureSignature
	if err := resumed.Publish(CampaignSummary{Attempted: 3, Succeeded: 2, Failures: 1, DistinctFailures: 1, FailureSignatures: []record.SHA256{signature}, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	opened, err := OpenCampaign(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if len(opened.Executions) != 3 || opened.Executions[1].Domain != "success" || opened.Executions[1].SelectionOrdinal != 1 {
		t.Fatalf("published outcomes = %#v", opened.Executions)
	}
}

func TestResumeCampaignJournalAcceptsRunsSharingDeduplicatedFailureEvidence(t *testing.T) {
	fixture := newFailureCampaign(t)
	journal := fixture.journal
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	first := fixture.publishFailure(t, 0, 7, "")
	second := first
	second.SelectionOrdinal, second.Seed = 1, 8
	for _, run := range []ExecutionRecord{first, second} {
		if err := journal.AppendExecution(run); err != nil {
			t.Fatal(err)
		}
	}
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
	if len(state.Executions) != 2 || state.Executions[0].Seed != 7 || state.Executions[1].Seed != 8 || *state.Executions[0].Artifact != *state.Executions[1].Artifact {
		t.Fatalf("resume state = %#v", state)
	}
	if remaining := directoryNames(t, journal.FailuresPath()); len(remaining) != 1 {
		t.Fatalf("shared failure evidence was reclaimed: %v", remaining)
	}
}

func TestCampaignRecoveryValidatesLiveEvidenceBeforeReclaimingAnything(t *testing.T) {
	fixture := newFailureCampaign(t)
	journal := fixture.journal
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	committed := fixture.publishFailure(t, 0, 7, "")
	if err := journal.AppendExecution(committed); err != nil {
		t.Fatal(err)
	}
	orphan := fixture.publishFailure(t, 1, 8, "orphan_reason")
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	live := filepath.Join(journal.Path(), filepath.FromSlash(*committed.Artifact))
	if err := os.Rename(live, live+".moved"); err != nil {
		t.Fatal(err)
	}
	if _, err := RecoverCampaign(context.Background(), journal.Path()); err == nil || !IsIntegrityError(err) {
		t.Fatalf("RecoverCampaign() missing live evidence error = %v", err)
	}
	if _, _, err := ResumeCampaignJournal(context.Background(), journal.Path()); err == nil {
		t.Fatal("ResumeCampaignJournal() accepted missing live evidence")
	}
	if _, err := os.Lstat(filepath.Join(journal.Path(), filepath.FromSlash(*orphan.Artifact))); err != nil {
		t.Fatalf("refused recovery reclaimed an orphan: %v", err)
	}
	if err := os.Rename(live+".moved", live); err != nil {
		t.Fatal(err)
	}
	recovered, err := RecoverCampaign(context.Background(), journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if !recovered.Changed || !recovered.After.Resumable {
		t.Fatalf("recovery result = %#v", recovered)
	}
	if _, err := os.Lstat(filepath.Join(journal.Path(), filepath.FromSlash(*orphan.Artifact))); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("recovery kept an unreferenced owned artifact: %v", err)
	}
	again, err := RecoverCampaign(context.Background(), journal.Path())
	if err != nil || again.Changed {
		t.Fatalf("repeated recovery = %#v, %v", again, err)
	}
}

func TestRecoverCampaignRepairsATornTailUnderTheLock(t *testing.T) {
	journal := newPreparedLifecycleJournal(t, "run-torn-recovery")
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendExecution(successExecution(0, 7)); err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	intact := logFileSize(t, journal.Path())
	payload, err := logstore.EncodeJSON(outcomeRecord{Executions: []ExecutionRecord{successExecution(1, 8)}})
	if err != nil {
		t.Fatal(err)
	}
	appendRawCampaignFrame(t, journal.Path(), kindOutcome, payload)
	logPath := filepath.Join(journal.Path(), storeDirectory, logstore.LogFile)
	if err := os.Truncate(logPath, logFileSize(t, journal.Path())-3); err != nil {
		t.Fatal(err)
	}
	torn := logFileSize(t, journal.Path())
	before, err := InspectCampaignLifecycle(journal.Path())
	if err != nil || !before.Resumable || logFileSize(t, journal.Path()) != torn {
		t.Fatalf("torn lifecycle = %#v, %v", before, err)
	}
	recovered, err := RecoverCampaign(context.Background(), journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if !recovered.Changed || logFileSize(t, journal.Path()) != intact {
		t.Fatalf("recovery result = %#v, log %d bytes, want %d", recovered, logFileSize(t, journal.Path()), intact)
	}
	resumed, state, err := ResumeCampaignJournal(context.Background(), journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Executions) != 1 {
		t.Fatalf("torn outcome was applied: %#v", state.Executions)
	}
	if err := resumed.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRecoverCampaignClassifiesNonRecoverableStateAsIntegrityError(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-non-recoverable", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	before := logFileSize(t, journal.Path())
	if _, err := RecoverCampaign(context.Background(), journal.Path()); err == nil {
		t.Fatal("RecoverCampaign() accepted non-recoverable planned state")
	} else if !IsIntegrityError(err) {
		t.Fatalf("RecoverCampaign() error = %T %v, want integrity error", err, err)
	}
	if after := logFileSize(t, journal.Path()); after != before {
		t.Fatalf("rejected recovery changed the log from %d to %d bytes", before, after)
	}
}

func TestRecoverCampaignPreservesContextClassification(t *testing.T) {
	journal := newPreparedLifecycleJournal(t, "run-cancelled-recovery")
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
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
	journal := newPreparedLifecycleJournal(t, "run-published-recovery")
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	for ordinal, seed := range []uint64{7, 8, 9} {
		if err := journal.AppendExecution(successExecution(uint64(ordinal), seed)); err != nil {
			t.Fatal(err)
		}
	}
	controller := &mutationController{inject: func(point mutationPoint) error {
		if point.Kind == mutationDelete && point.Operation == "prepared-target" {
			return errors.New("injected cleanup failure")
		}
		return nil
	}, enabled: true}
	journal.ctx = withMutationHook(context.Background(), controller.observe)
	if err := journal.Publish(CampaignSummary{Attempted: 3, Succeeded: 3, StopReason: "seeds_exhausted"}); err == nil {
		t.Fatal("Publish() ignored its interrupted cleanup")
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := InspectCampaignLifecycle(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if before.State != lifecyclePublished || !before.Published || !before.Repairable || before.Action != recoveryFinalizePublication {
		t.Fatalf("published lifecycle before recovery = %#v", before)
	}
	recovered, err := RecoverCampaign(context.Background(), journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if !recovered.Changed || recovered.Action != recoveryFinalizePublication || recovered.After.State != lifecyclePublished || recovered.After.Repairable {
		t.Fatalf("recovery result = %#v", recovered)
	}
	if _, err := os.Lstat(journal.PreparedPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("published prepared target remains: %v", err)
	}
	again, err := RecoverCampaign(context.Background(), journal.Path())
	if err != nil || again.Changed {
		t.Fatalf("repeated recovery = %#v, %v", again, err)
	}
	if _, _, err := ResumeCampaignJournal(context.Background(), journal.Path()); err == nil || !IsIntegrityError(err) {
		t.Fatalf("ResumeCampaignJournal(published) error = %v", err)
	}
}

func TestPreflightResumeReturnsValidatedLifecycleAndPlan(t *testing.T) {
	journal := newPreparedLifecycleJournal(t, "run-resume-preflight")
	preflight, err := PreflightResume(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if preflight.Lifecycle.State != lifecyclePrepared || !preflight.Lifecycle.Resumable {
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
		t.Fatal("PreflightResume() accepted a planned campaign")
	} else if !IsIntegrityError(err) {
		t.Fatalf("PreflightResume() error = %T %v, want integrity error", err, err)
	}
}

func TestRecoverAndResumeShareTheWritersExclusiveLock(t *testing.T) {
	journal := newPreparedLifecycleJournal(t, "run-exclusive-recovery")
	if _, err := RecoverCampaign(context.Background(), journal.Path()); err == nil || !strings.Contains(err.Error(), "already being resumed") || !errors.Is(err, logstore.ErrInUse) {
		t.Fatalf("RecoverCampaign() error = %v, want lock contention", err)
	}
	if _, _, err := ResumeCampaignJournal(context.Background(), journal.Path()); err == nil || !strings.Contains(err.Error(), "already being resumed") {
		t.Fatalf("ResumeCampaignJournal() error = %v, want lock contention", err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	resumed, _, err := ResumeCampaignJournal(context.Background(), journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ResumeCampaignJournal(context.Background(), journal.Path()); err == nil || !strings.Contains(err.Error(), "already being resumed") {
		t.Fatalf("second ResumeCampaignJournal() error = %v, want lock contention", err)
	}
	if err := resumed.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestResumeCampaignJournalRepeatsResumeIdempotently(t *testing.T) {
	journal := newPreparedLifecycleJournal(t, "run-repeated-resume")
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendExecution(successExecution(0, 7)); err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	for attempt := range 3 {
		resumed, state, err := ResumeCampaignJournal(context.Background(), journal.Path())
		if err != nil {
			t.Fatalf("resume %d: %v", attempt, err)
		}
		if len(state.Executions) != 1 || state.Executions[0].Seed != 7 {
			t.Fatalf("resume %d state = %#v", attempt, state)
		}
		if err := resumed.Close(); err != nil {
			t.Fatal(err)
		}
		assertLifecycle(t, journal.Path(), lifecycleRunning, lifecycleRunning, true)
	}
}
