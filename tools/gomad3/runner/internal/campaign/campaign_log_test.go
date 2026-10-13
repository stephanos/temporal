package campaign

import (
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/logstore"
)

func TestCampaignLogPublishesLifecycleAndOutcomesAsItsOnlyState(t *testing.T) {
	journal := newPreparedLifecycleJournal(t, "run-fixed")
	assertLifecycle(t, journal.Path(), lifecyclePrepared, lifecyclePrepared, true)
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	assertLifecycle(t, journal.Path(), lifecycleRunning, lifecycleRunning, true)
	run, err := journal.BeginExecution(0, 7)
	if err != nil {
		t.Fatal(err)
	}
	if err := run.Transition(ExecutionStarting); err != nil {
		t.Fatal(err)
	}
	stdout, err := run.CreateOutput("stdout")
	if err != nil {
		t.Fatal(err)
	}
	if err := run.CloseOutput("stdout", stdout); err != nil {
		t.Fatal(err)
	}
	for _, state := range []executionState{ExecutionExited, ExecutionCaptured, ExecutionClassified} {
		if err := run.Transition(state); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Lstat(filepath.Join(run.Path(), "partial.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("attempt work kept a state file: %v", err)
	}
	for ordinal, seed := range []uint64{7, 8, 9} {
		if err := journal.AppendExecution(successExecution(uint64(ordinal), seed)); err != nil {
			t.Fatal(err)
		}
	}
	if err := run.Complete(); err != nil {
		t.Fatal(err)
	}
	if err := journal.Publish(CampaignSummary{Attempted: 3, Succeeded: 3, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	assertLifecycle(t, journal.Path(), lifecyclePublished, lifecyclePublished, false)

	kinds, report := campaignLogRecords(t, journal.Path())
	want := []logstore.Kind{kindCampaignOpen, kindPlan, kindLifecycle, kindLifecycle, kindOutcome, kindOutcome, kindOutcome, kindPublish}
	if !slices.Equal(kinds, want) || report.Tail != logstore.TailNone {
		t.Fatalf("campaign log = %v %#v, want %v", kinds, report, want)
	}
	store, err := os.ReadDir(filepath.Join(journal.Path(), storeDirectory))
	if err != nil {
		t.Fatal(err)
	}
	if len(store) != 2 || store[0].Name() != logstore.LockFile || store[1].Name() != logstore.LogFile {
		t.Fatalf("campaign store = %v, want only its log and lock", store)
	}
	for _, removed := range []string{"campaign.json", "executions", filepath.Join(".partial", "campaign"), filepath.Join(".partial", "executions"), ".prepared", ".resume.lock", run.Path()} {
		if !filepath.IsAbs(removed) {
			removed = filepath.Join(journal.Path(), removed)
		}
		if _, err := os.Lstat(removed); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("campaign keeps %s beside its log: %v", removed, err)
		}
	}
	opened, err := OpenCampaign(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if opened.Record.CampaignID != "run-fixed" || opened.Record.Selection != "7-9" || len(opened.Executions) != 3 || opened.Executions[2].Seed != 9 || opened.Record.Artifacts == nil {
		t.Fatalf("opened campaign = %#v", opened)
	}
	info := opened.Journal
	if info == nil || info.Schema != CampaignLogFormat || info.Records != 3 || info.Bytes == 0 || info.IndexSHA256 != "" || info.Segments != 0 || info.Limits != journal.ExecutionJournalPlan() {
		t.Fatalf("published journal info = %#v", info)
	}
	if err := journal.Publish(CampaignSummary{Attempted: 3, Succeeded: 3, StopReason: "seeds_exhausted"}); err == nil {
		t.Fatal("Publish() published twice")
	}
}

func TestCampaignLogRoundTripsChoiceProfileAndRunSummary(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{Root: t.TempDir(), CampaignID: "run-choices", Selection: "7", SelectionCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	digest := record.HashBytes([]byte("choices"))
	records := record.Uint64String(4)
	branching := record.Uint64String(2)
	decisions := record.Uint64String(3)
	tapeDigest := record.HashBytes([]byte("choice tape"))
	terminal := "complete"
	if err := journal.AppendExecution(ExecutionRecord{
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit",
		ChoiceTraceSHA256: &digest, ChoiceTraceRecords: &records, ChoiceTraceBranchingRecords: &branching, ChoiceTraceTerminalState: &terminal,
		ChoiceTapeSHA256: &tapeDigest, ChoiceDecisions: &decisions,
	}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Publish(CampaignSummary{Attempted: 1, Succeeded: 1, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	opened, err := OpenCampaign(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if len(opened.Executions) != 1 || opened.Executions[0].ChoiceTraceBranchingRecords == nil || *opened.Executions[0].ChoiceTraceBranchingRecords != 2 || opened.Executions[0].ChoiceTapeSHA256 == nil || *opened.Executions[0].ChoiceDecisions != 3 {
		t.Fatalf("choice run summary = %#v", opened.Executions)
	}
}

func TestChoiceProfileMatchesResumePlanExactly(t *testing.T) {
	plan := CampaignPlan{ChoiceProfile: &ChoiceProfilePlan{Name: choice.Profile, ImplementationSHA256: record.HashBytes([]byte("choice implementation")), Limit: 8 << 20}}
	manifest := record.ExecutionRecord{ChoiceProfile: &record.ChoiceProfile{
		Name: choice.Profile, ImplementationSHA256: plan.ChoiceProfile.ImplementationSHA256,
		Trace: record.ChoiceTrace{Limit: 8 << 20},
	}}
	if !choiceProfileMatchesPlan(plan, manifest) {
		t.Fatal("matching choice profile was rejected")
	}
	manifest.ChoiceProfile.Trace.Limit++
	if choiceProfileMatchesPlan(plan, manifest) {
		t.Fatal("changed choice limit was accepted")
	}
	if choiceProfileMatchesPlan(CampaignPlan{}, manifest) || choiceProfileMatchesPlan(plan, record.ExecutionRecord{}) {
		t.Fatal("enabled and disabled choice profiles were treated as compatible")
	}
}

func TestValidateBatchPlanRejectsChangedArtifactCapacity(t *testing.T) {
	plan := testBatchPlan(nil, record.HashBytes([]byte("target")), 1)
	journalPlan := recordExecutionJournalLimits(executionJournalLimits{
		MaximumExecutions: 3, MaximumBytes: 3 << 20, SegmentBytes: 1 << 20,
		SegmentRecords: 1024, MaximumSegments: 3, MaximumPartialExecutions: 2,
	})
	plan.Journal = &journalPlan
	if err := validateCampaignPlan(plan); err != nil {
		t.Fatal(err)
	}
	changed := *plan.Artifacts
	changed.FailureBytes++
	plan.Artifacts = &changed
	if err := validateCampaignPlan(plan); err == nil {
		t.Fatal("validateCampaignPlan() accepted changed artifact capacity")
	}
}

func TestCampaignLogRecordsTheFrozenResumePlanAndRefusesChangedTargets(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-resume-plan", Selection: "7-9", SelectionCount: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err := journal.BeginPreparation(); err != nil {
		t.Fatal(err)
	}
	preparedPath := filepath.Join(journal.PreparedPath(), "build", "target")
	if err := os.MkdirAll(filepath.Dir(preparedPath), 0o700); err != nil {
		t.Fatal(err)
	}
	targetBytes := []byte("prepared target")
	if err := os.WriteFile(preparedPath, targetBytes, 0o500); err != nil {
		t.Fatal(err)
	}
	plan := testBatchPlan(journal, record.HashBytes(targetBytes), uint64(len(targetBytes)))
	plan.Toolchain.BuildKey = strings.Repeat("a", 64)
	implementation, err := choice.ImplementationIdentity(plan.Toolchain.BuildKey)
	if err != nil {
		t.Fatal(err)
	}
	plan.ChoiceProfile = &ChoiceProfilePlan{Name: choice.Profile, ImplementationSHA256: record.SHA256FromSum(implementation), Limit: 8 << 20}
	artifacts, err := DeriveArtifactCapacityPlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	plan.Artifacts = &artifacts
	if _, err := ReadResumePlan(journal.Path()); err == nil || err.Error() != "campaign preparation did not complete" {
		t.Fatalf("ReadResumePlan() before the plan error = %v", err)
	}
	if err := journal.RecordPlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := journal.RecordPlan(plan); err == nil || err.Error() != "campaign plan already exists" || errors.Unwrap(err) != nil {
		t.Fatalf("RecordPlan() existing-plan error = %v", err)
	}
	if _, err := ReadResumePlan(journal.Path()); err == nil || err.Error() != "campaign preparation did not complete" {
		t.Fatalf("ReadResumePlan() during preparation error = %v", err)
	}
	if err := journal.CompletePreparation(); err != nil {
		t.Fatal(err)
	}
	opened, err := ReadResumePlan(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(opened, plan) {
		t.Fatalf("resume plan = %#v, want %#v", opened, plan)
	}
	if err := os.Chmod(preparedPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(preparedPath, []byte("changed"), 0o500); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(preparedPath, 0o500); err != nil {
		t.Fatal(err)
	}
	const wantTarget = "prepared target identity does not match campaign plan\ntarget metadata does not match its manifest"
	rejected, err := ReadResumePlan(journal.Path())
	if err == nil || err.Error() != wantTarget || !reflect.DeepEqual(rejected, CampaignPlan{}) {
		t.Fatalf("ReadResumePlan() error = %v", err)
	}
	if status, err := InspectCampaignLifecycle(journal.Path()); err != nil || status.Resumable {
		t.Fatalf("changed target lifecycle = %#v, %v", status, err)
	}
}

func TestCampaignLogRetainsRecoverableFailureProvenance(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-recoverable", Selection: "1", SelectionCount: 1,
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
	if err := journal.FailPreparation("target_preparation", errors.New("build failed")); err != nil {
		t.Fatal(err)
	}
	status, err := InspectCampaignLifecycle(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if status.State != lifecycleRecoverableFailure || status.LastStableState != lifecyclePlanned || status.Reason != "target_preparation" || status.Detail != "build failed" || status.Resumable {
		t.Fatalf("failed preparation lifecycle = %#v", status)
	}
	if err := journal.Fail("target_preparation", nil); err != nil {
		t.Fatal(err)
	}
	if status, err := InspectCampaignLifecycle(journal.Path()); err != nil || status.Detail != "" || status.LastStableState != lifecyclePlanned {
		t.Fatalf("repeated failure lifecycle = %#v, %v", status, err)
	}

	running := newPreparedLifecycleJournal(t, "run-recoverable-running")
	if err := running.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := running.Fail("cancelled", context.Canceled); err != nil {
		t.Fatal(err)
	}
	assertLifecycle(t, running.Path(), lifecycleRecoverableFailure, lifecycleRunning, true)
}

func TestCampaignLogRejectsIllegalRecordsWithoutAppending(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-illegal-transition", Selection: "7-9", SelectionCount: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	})
	before := logFileSize(t, journal.Path())
	for name, attempt := range map[string]func() error{
		"planned to running":    func() error { return journal.transition(context.Background(), lifecycleRunning, "", nil) },
		"failure without cause": func() error { return journal.transition(context.Background(), lifecycleRecoverableFailure, "", nil) },
		"outcome before start": func() error {
			return journal.append(kindOutcome, outcomeRecord{Executions: []ExecutionRecord{successExecution(0, 7)}})
		},
		"second envelope": func() error { return journal.append(kindCampaignOpen, journal.state.open) },
	} {
		if err := attempt(); err == nil {
			t.Fatalf("%s was accepted", name)
		}
	}
	if after := logFileSize(t, journal.Path()); after != before {
		t.Fatalf("rejected records changed the log from %d to %d bytes", before, after)
	}
	if err := journal.StartExecutions(); err != nil {
		t.Fatalf("writer was poisoned by rejected records: %v", err)
	}
	if err := journal.AppendExecution(successExecution(0, 7)); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendExecution(successExecution(0, 7)); err == nil || !strings.Contains(err.Error(), "duplicated") {
		t.Fatalf("duplicate ordinal error = %v", err)
	}
}

func TestCampaignLogBoundsFailureEvidence(t *testing.T) {
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
	if len(status.Detail) > maximumLifecycleText || len(status.Detail) < maximumLifecycleText-1 || !utf8.ValidString(status.Detail) {
		t.Fatalf("bounded lifecycle detail is invalid: bytes=%d", len(status.Detail))
	}
}

func TestCampaignLogEnforcesExecutionLimitsBeforeAppending(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-journal-capacity", Selection: "7", SelectionCount: 1,
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
	before := logFileSize(t, journal.Path())
	err = journal.AppendExecution(ExecutionRecord{
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: strings.Repeat("x", defaultExecutionSegmentBytes), Termination: "exit",
	})
	var capacityErr *JournalCapacityError
	if !errors.As(err, &capacityErr) || capacityErr.Limit != journalLimitSegmentBytes || capacityErr.Outcome != CapacityInfrastructureFailure {
		t.Fatalf("AppendExecution() oversized error = %v", err)
	}
	if after := logFileSize(t, journal.Path()); after != before {
		t.Fatalf("rejected execution changed the log from %d to %d bytes", before, after)
	}
	if err := journal.AppendExecution(ExecutionRecord{SelectionOrdinal: 1, Seed: 8, Domain: "success", Reason: "success", Termination: "exit"}); err == nil || !strings.Contains(err.Error(), "out of range") && !errors.As(err, &capacityErr) {
		t.Fatalf("AppendExecution() beyond the selection error = %v", err)
	}
	if err := journal.AppendExecution(successExecution(0, 7)); err != nil {
		t.Fatal(err)
	}
	err = journal.AppendExecution(successExecution(1, 8))
	if !errors.As(err, &capacityErr) || capacityErr.Limit != journalLimitExecutions {
		t.Fatalf("AppendExecution() beyond the execution capacity error = %v", err)
	}
}

func TestCampaignLogRetiresSegmentGroupingLimits(t *testing.T) {
	limits := executionJournalLimits{
		MaximumExecutions: 5, MaximumBytes: 5 << 20, SegmentBytes: 1 << 20,
		SegmentRecords: 1, MaximumSegments: 1, MaximumPartialExecutions: 1,
	}
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-retired-segments", Selection: "1-5", SelectionCount: 5, Journal: limits,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	})
	if plan := journal.ExecutionJournalPlan(); plan.SegmentRecords != 1 || plan.MaximumSegments != 1 {
		t.Fatalf("retired segment values were not retained: %#v", plan)
	}
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	for ordinal := range uint64(5) {
		if err := journal.AppendExecution(successExecution(ordinal, ordinal+1)); err != nil {
			t.Fatalf("execution %d beyond one segment record: %v", ordinal, err)
		}
	}
	if err := journal.Publish(CampaignSummary{Attempted: 5, Succeeded: 5, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
}

func TestCampaignJournalEnforcesPartialRunCapacityBeforeMutation(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-partials", Selection: "1-3", SelectionCount: 3, Parallel: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	})
	first, err := journal.BeginExecution(0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := journal.BeginExecution(1, 2); err != nil {
		t.Fatal(err)
	}
	_, err = journal.BeginExecution(2, 3)
	var capacityErr *JournalCapacityError
	if !errors.As(err, &capacityErr) || capacityErr.Limit != journalLimitPartialExecutions || capacityErr.Required != 3 || capacityErr.Maximum != 2 {
		t.Fatalf("BeginExecution() over capacity error = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(journal.Path(), ".partial", executionWorkName(2, 3))); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("rejected attempt created work: %v", err)
	}
	for _, state := range []executionState{ExecutionStarting, ExecutionExited, ExecutionCaptured, ExecutionClassified} {
		if err := first.Transition(state); err != nil {
			t.Fatal(err)
		}
	}
	if err := first.Complete(); err != nil {
		t.Fatal(err)
	}
	if _, err := journal.BeginExecution(2, 3); err != nil {
		t.Fatalf("BeginExecution() after a released slot: %v", err)
	}
}

func TestExecutionJournalRejectsInvalidStateTransitions(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-state", Selection: "1", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	}()
	run, err := journal.BeginExecution(0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := run.Transition(ExecutionCaptured); err == nil {
		t.Fatal("Transition(captured) succeeded before starting")
	}
	if err := run.Complete(); err == nil {
		t.Fatal("Complete() succeeded before classification")
	}
}

func TestCampaignLogClassifiesDamageAndObservesTornTailsReadOnly(t *testing.T) {
	journal := newPreparedLifecycleJournal(t, "run-damage")
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendExecution(successExecution(0, 7)); err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(journal.Path(), storeDirectory, logstore.LogFile)
	intact, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}

	torn := append(append([]byte(nil), intact...), "GMLG"...)
	if err := os.WriteFile(logPath, torn, 0o600); err != nil {
		t.Fatal(err)
	}
	status, err := InspectCampaignLifecycle(journal.Path())
	if err != nil || status.State != lifecycleRunning || !status.Resumable {
		t.Fatalf("torn-tail lifecycle = %#v, %v", status, err)
	}
	if after, err := os.ReadFile(logPath); err != nil || len(after) != len(torn) {
		t.Fatalf("read-only inspection changed the torn log: %d bytes, %v", len(after), err)
	}

	damaged := append([]byte(nil), intact...)
	damaged[30] ^= 0xff
	if err := os.WriteFile(logPath, damaged, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectCampaignLifecycle(journal.Path()); !IsIntegrityError(err) || !errors.Is(err, logstore.ErrCorrupt) {
		t.Fatalf("damaged log error = %T %v, want integrity corruption", err, err)
	}
	if _, err := RecoverCampaign(context.Background(), journal.Path()); !IsIntegrityError(err) {
		t.Fatalf("RecoverCampaign() damaged log error = %v", err)
	}
	if after, err := os.ReadFile(logPath); err != nil || string(after) != string(damaged) {
		t.Fatalf("recovery mutated a corrupt log: %v", err)
	}
}

func TestCampaignLogRejectsSemanticallyInvalidCompleteRecords(t *testing.T) {
	journal := newPreparedLifecycleJournal(t, "run-invalid-record")
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	payload, err := logstore.EncodeJSON(outcomeRecord{Executions: []ExecutionRecord{successExecution(5, 7)}})
	if err != nil {
		t.Fatal(err)
	}
	appendRawCampaignFrame(t, journal.Path(), kindOutcome, payload)
	before := logFileSize(t, journal.Path())
	if _, err := InspectCampaignLifecycle(journal.Path()); !IsIntegrityError(err) || !errors.Is(err, logstore.ErrCorrupt) || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("invalid complete outcome error = %v", err)
	}
	if _, _, err := ResumeCampaignJournal(context.Background(), journal.Path()); err == nil {
		t.Fatal("ResumeCampaignJournal() accepted an invalid complete outcome")
	}
	if after := logFileSize(t, journal.Path()); after != before {
		t.Fatalf("refused resume changed the log from %d to %d bytes", before, after)
	}
}

// appendRawCampaignFrame appends a well-formed frame that the campaign's own
// writer would refuse, as a coherent foreign writer or damage could.
func appendRawCampaignFrame(t *testing.T, path string, kind logstore.Kind, payload []byte) {
	t.Helper()
	table := crc32.MakeTable(crc32.Castagnoli)
	frame := make([]byte, 16+len(payload)+4)
	copy(frame, "GMLG")
	frame[4] = 1
	binary.LittleEndian.PutUint16(frame[6:], uint16(kind))
	binary.LittleEndian.PutUint32(frame[8:], uint32(len(payload)))
	binary.LittleEndian.PutUint32(frame[12:], crc32.Checksum(frame[:12], table))
	copy(frame[16:], payload)
	binary.LittleEndian.PutUint32(frame[16+len(payload):], crc32.Update(crc32.Checksum(frame[:16], table), table, payload))
	file, err := os.OpenFile(filepath.Join(path, storeDirectory, logstore.LogFile), os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(frame); err != nil {
		t.Fatal(errors.Join(err, file.Close()))
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestCampaignStorageRefusesLegacyAndUnknownFormatsBeforeMutation(t *testing.T) {
	for name, marker := range map[string]string{
		"published campaign": "campaign.json",
		"execution journal":  "executions",
		"lifecycle state":    filepath.Join(".partial", "campaign"),
	} {
		t.Run(name, func(t *testing.T) {
			path := privateDirectory(t)
			if err := os.MkdirAll(filepath.Join(path, filepath.Dir(marker)), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(path, marker), []byte("{}"), 0o600); err != nil {
				t.Fatal(err)
			}
			for operation, open := range map[string]func() error{
				"open":    func() error { _, err := OpenCampaign(path); return err },
				"inspect": func() error { _, err := InspectCampaignLifecycle(path); return err },
				"plan":    func() error { _, err := ReadResumePlan(path); return err },
				"resume":  func() error { _, _, err := ResumeCampaignJournal(context.Background(), path); return err },
				"recover": func() error { _, err := RecoverCampaign(context.Background(), path); return err },
				"kind":    func() error { _, err := StoreKind(path); return err },
			} {
				err := open()
				var unsupported *UnsupportedFormatError
				if !errors.As(err, &unsupported) || unsupported.Format != LegacyCampaignFormat || !errors.Is(err, ErrUnsupportedFormat) || !strings.Contains(err.Error(), LegacyCampaignFormat) {
					t.Fatalf("%s legacy campaign error = %v", operation, err)
				}
			}
			if _, err := os.Lstat(filepath.Join(path, storeDirectory)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("refusal created a log store: %v", err)
			}
		})
	}

	merged := privateDirectory(t)
	if err := os.WriteFile(filepath.Join(merged, "merge.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	var unsupported *UnsupportedFormatError
	if _, err := OpenMergedCampaign(merged); !errors.As(err, &unsupported) || unsupported.Format != LegacyMergedCampaignFormat {
		t.Fatalf("legacy merged campaign error = %v", err)
	}

	journal := newPreparedLifecycleJournal(t, "run-future-format")
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(journal.Path(), storeDirectory, logstore.LogFile)
	for format, want := range map[string]*UnsupportedFormatError{
		`{"format":"gomad3.campaign-log/v2","campaign_id":"x"}`: {Format: "gomad3.campaign-log/v2"},
		`{"campaign_id":"x"}`: {Malformed: true},
	} {
		if err := os.WriteFile(logPath, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		appendRawCampaignFrame(t, journal.Path(), kindCampaignOpen, []byte(format))
		before, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatal(err)
		}
		for operation, open := range map[string]func() error{
			"inspect": func() error { _, err := InspectCampaignLifecycle(journal.Path()); return err },
			"resume":  func() error { _, _, err := ResumeCampaignJournal(context.Background(), journal.Path()); return err },
			"recover": func() error { _, err := RecoverCampaign(context.Background(), journal.Path()); return err },
		} {
			err := open()
			if !errors.As(err, &unsupported) || *unsupported != *want || !IsIntegrityError(err) {
				t.Fatalf("%s envelope %s error = %v", operation, format, err)
			}
		}
		if after, err := os.ReadFile(logPath); err != nil || string(after) != string(before) {
			t.Fatalf("refused envelope %s was mutated: %v", format, err)
		}
	}
}

func TestStoreKindDistinguishesCampaignLogs(t *testing.T) {
	journal := newPreparedLifecycleJournal(t, "run-kind")
	if kind, err := StoreKind(journal.Path()); err != nil || kind != "campaign" {
		t.Fatalf("StoreKind(campaign) = %q, %v", kind, err)
	}
	if kind, err := StoreKind(t.TempDir()); err != nil || kind != "" {
		t.Fatalf("StoreKind(empty) = %q, %v", kind, err)
	}
}
