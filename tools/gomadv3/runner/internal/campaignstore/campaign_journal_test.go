package campaignstore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/evidence"
)

func TestBatchJournalPublishesTheCanonicalBatchLifecycle(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-fixed", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	if err := journal.BeginPreparation(); err != nil {
		t.Fatal(err)
	}
	if err := journal.CompletePreparation(); err != nil {
		t.Fatal(err)
	}
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	run, err := journal.BeginExecution(0, 7)
	if err != nil {
		t.Fatal(err)
	}
	partialPath := filepath.Join(run.Path(), "partial.json")
	assertFileContents(t, partialPath, `{"schema_version":5,"seed":"7","selection_ordinal":"0","state":"staging"}`)
	if err := run.Transition(ExecutionStarting); err != nil {
		t.Fatal(err)
	}
	stdout, err := run.CreateOutput("stdout")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stdout.Write([]byte("output")); err != nil {
		t.Fatal(err)
	}
	if err := stdout.Close(); err != nil {
		t.Fatal(err)
	}
	for _, state := range []ExecutionState{ExecutionExited, ExecutionCaptured, ExecutionClassified} {
		if err := run.Transition(state); err != nil {
			t.Fatal(err)
		}
	}
	if err := journal.AppendExecution(ExecutionRecord{
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit", ElapsedNanos: 5,
	}); err != nil {
		t.Fatal(err)
	}
	if err := run.Complete(); err != nil {
		t.Fatal(err)
	}
	if err := journal.Publish(CampaignSummary{Attempted: 1, Succeeded: 1, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	const wantRuns = "{\"artifact\":null,\"domain\":\"success\",\"elapsed_nanos\":\"5\",\"failure_signature\":null,\"io_transcript_records\":null,\"io_transcript_sha256\":null,\"reason\":\"success\",\"seed\":\"7\",\"selection_ordinal\":\"0\",\"termination\":\"exit\"}\n"
	assertFileContents(t, filepath.Join(journal.Path(), "runs", "00000000000000000000.jsonl"), wantRuns)
	batchBytes, err := os.ReadFile(filepath.Join(journal.Path(), "batch.json"))
	if err != nil {
		t.Fatal(err)
	}
	var batch CampaignRecord
	if err := evidence.DecodeCanonicalJSON(batchBytes, &batch); err != nil {
		t.Fatal(err)
	}
	if batch.Schema != "gomadv3.batch/v3" || batch.Journal == nil || batch.Journal.Records != 1 || batch.Journal.Segments != 1 || batch.RunsSHA256 != "" {
		t.Fatalf("published batch = %#v", batch)
	}
	for _, removed := range []string{journal.PreparedPath(), filepath.Join(journal.Path(), ".partial", "batch"), run.Path()} {
		if _, err := os.Stat(removed); !os.IsNotExist(err) {
			t.Fatalf("completed path %q remains: %v", removed, err)
		}
	}
}

func TestOpenBatchRetainsTheLegacySeedReader(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-legacy-seed", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendExecution(ExecutionRecord{SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit"}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Publish(CampaignSummary{Attempted: 1, Succeeded: 1, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	batchPath := journal.Path()
	contents, err := os.ReadFile(filepath.Join(batchPath, "batch.json"))
	if err != nil {
		t.Fatal(err)
	}
	var batch CampaignRecord
	if err := evidence.DecodeCanonicalJSON(contents, &batch); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(batchPath)
	if err != nil {
		t.Fatal(err)
	}
	runs, _, err := readPublishedRunJournal(root, batch)
	closeErr := root.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	runsBytes, err := encodeExecutionRecords(runs)
	if err != nil {
		t.Fatal(err)
	}
	batch.Schema = "gomadv3.batch/v1"
	batch.Strategy = ""
	batch.RunsSHA256 = evidence.HashBytes(runsBytes)
	batch.Journal = nil
	legacy, err := evidence.CanonicalJSON(batch)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(batchPath, "runs.jsonl"), runsBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(batchPath, "batch.json"), legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(batchPath, "runs")); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenCampaign(batchPath); err != nil {
		t.Fatal(err)
	}
	batch.Schema = "gomadv3.batch/v2"
	batch.Strategy = "seed"
	previous, err := evidence.CanonicalJSON(batch)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(batchPath, "batch.json"), previous, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenCampaign(batchPath); err != nil {
		t.Fatal(err)
	}
}

func TestValidateBatchPlanRetainsOnlyTheLegacySeedContract(t *testing.T) {
	plan := testBatchPlan(nil, evidence.HashBytes([]byte("target")), 1)
	plan.Schema = LegacyBatchPlanSchema
	plan.Strategy = ""
	plan.Journal = nil
	plan.Artifacts = nil
	plan.Prepared.Target.CapabilityMode = ""
	if err := validateCampaignPlan(plan); err != nil {
		t.Fatal(err)
	}
	plan.MaxRuns = 1
	if err := validateCampaignPlan(plan); err == nil {
		t.Fatal("validateBatchPlan() accepted frontier fields in a legacy plan")
	}
}

func TestValidateBatchPlanRejectsChangedArtifactCapacity(t *testing.T) {
	plan := testBatchPlan(nil, evidence.HashBytes([]byte("target")), 1)
	journalLimits := RunJournalLimits{
		MaximumRuns: 3, MaximumBytes: 3 << 20, SegmentBytes: 1 << 20,
		SegmentRecords: 1024, MaximumSegments: 3, MaximumPartialRuns: 2,
	}
	journalPlan := recordRunJournalLimits(journalLimits)
	plan.Journal = &journalPlan
	artifacts, err := DeriveArtifactCapacityPlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	plan.Artifacts = &artifacts
	if err := validateCampaignPlan(plan); err != nil {
		t.Fatal(err)
	}
	plan.Artifacts.FailureBytes--
	if err := validateCampaignPlan(plan); err == nil || !strings.Contains(err.Error(), "artifact limits") {
		t.Fatalf("validateCampaignPlan() error = %v", err)
	}
}

func TestOpenBatchValidatesPublishedJournal(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-open", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendExecution(ExecutionRecord{
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit", ElapsedNanos: 5,
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
	if opened.Path != journal.Path() || opened.Record.CampaignID != "run-open" || len(opened.Runs) != 1 || opened.Runs[0].Seed != 7 {
		t.Fatalf("opened batch = %#v", opened)
	}
}

func TestBatchJournalRecordsCanonicalResumePlan(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-resume-plan", Selection: "7-9", SelectionCount: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
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
	plan := testBatchPlan(journal, evidence.HashBytes(targetBytes), uint64(len(targetBytes)))
	plan.Toolchain.BuildKey = strings.Repeat("a", 64)
	implementation, err := choice.ImplementationIdentity(plan.Toolchain.BuildKey)
	if err != nil {
		t.Fatal(err)
	}
	plan.ChoiceProfile = &ChoiceProfilePlan{Name: choice.Profile, ImplementationSHA256: evidence.SHA256FromSum(implementation), Limit: 8 << 20}
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
	opened, err := ReadResumePlan(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if opened.Selection != "7-9" || opened.Prepared.Target.SHA256 != plan.Prepared.Target.SHA256 || opened.Prepared.Path != ".prepared/build/target" || opened.OverallTimeoutNanos != evidence.Uint64String(10*time.Second) || opened.ChoiceProfile == nil || opened.ChoiceProfile.Limit != 8<<20 {
		t.Fatalf("resume plan = %#v", opened)
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
	if _, err := ReadResumePlan(journal.Path()); err == nil || !strings.Contains(err.Error(), "prepared target identity") {
		t.Fatalf("ReadResumePlan() error = %v", err)
	}
}

func TestResumeBatchJournalReusesVerifiedRunsAndArchivesIncompleteWork(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-resume", Selection: "7-9", SelectionCount: 3,
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
	targetBytes := []byte("prepared target")
	if err := os.WriteFile(preparedPath, targetBytes, 0o500); err != nil {
		t.Fatal(err)
	}
	if err := journal.RecordPlan(testBatchPlan(journal, evidence.HashBytes(targetBytes), uint64(len(targetBytes)))); err != nil {
		t.Fatal(err)
	}
	if err := journal.CompletePreparation(); err != nil {
		t.Fatal(err)
	}
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendExecution(ExecutionRecord{SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit"}); err != nil {
		t.Fatal(err)
	}
	incomplete, err := journal.BeginExecution(1, 8)
	if err != nil {
		t.Fatal(err)
	}
	if err := incomplete.Transition(ExecutionStarting); err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}

	resumed, state, err := ResumeCampaignJournal(context.Background(), journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	defer resumed.Close()
	if len(state.Runs) != 1 || state.Runs[0].Seed != 7 || state.Plan.Selection != "7-9" {
		t.Fatalf("resume state = %#v", state)
	}
	for ordinal, seed := range []uint64{8, 9} {
		run, err := resumed.BeginExecution(uint64(ordinal+1), seed)
		if err != nil {
			t.Fatal(err)
		}
		for _, next := range []ExecutionState{ExecutionStarting, ExecutionExited, ExecutionCaptured, ExecutionClassified} {
			if err := run.Transition(next); err != nil {
				t.Fatal(err)
			}
		}
		if err := resumed.AppendExecution(ExecutionRecord{SelectionOrdinal: evidence.Uint64String(ordinal + 1), Seed: evidence.Uint64String(seed), Domain: "success", Reason: "success", Termination: "exit"}); err != nil {
			t.Fatal(err)
		}
		if err := run.Complete(); err != nil {
			t.Fatal(err)
		}
	}
	if err := resumed.Publish(CampaignSummary{Attempted: 3, Succeeded: 3, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	opened, err := OpenCampaign(journal.Path())
	if err != nil {
		t.Fatal(err)
	}
	if len(opened.Runs) != 3 {
		t.Fatalf("published runs = %#v", opened.Runs)
	}
	archives, err := filepath.Glob(filepath.Join(journal.Path(), ".partial", "resume", "*", "partials", filepath.Base(incomplete.Path())))
	if err != nil || len(archives) != 1 {
		t.Fatalf("archived partials = %v, %v", archives, err)
	}
}

func TestResumeBatchJournalAcceptsRunsSharingDeduplicatedFailureEvidence(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "batch-1", Selection: "7-9", SelectionCount: 3,
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
	input := artifactInput(t)
	targetBytes, err := os.ReadFile(input.TargetPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(preparedPath, targetBytes, 0o500); err != nil {
		t.Fatal(err)
	}
	plan := testBatchPlan(journal, evidence.HashBytes(targetBytes), uint64(len(targetBytes)))
	plan.RunnerBuild = input.Manifest.Runner.RunnerBuild
	plan.Toolchain = input.Manifest.Toolchain
	plan.Prepared.Target = input.Manifest.Target
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
	input.Manifest.CampaignID = filepath.Base(journal.Path())
	published, err := PublishArtifact(evidence.Store{Root: journal.FailuresPath()}, input)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := filepath.Rel(journal.Path(), published.Path)
	if err != nil {
		t.Fatal(err)
	}
	signature := published.Manifest.Outcome.FailureSignature
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	for ordinal, seed := range []uint64{7, 8} {
		if err := journal.AppendExecution(ExecutionRecord{
			SelectionOrdinal: evidence.Uint64String(ordinal), Seed: evidence.Uint64String(seed), Domain: "target",
			Reason: published.Manifest.Outcome.Reason, Termination: published.Manifest.Outcome.Termination,
			FailureSignature: &signature, Artifact: &reference,
		}); err != nil {
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
	if len(state.Runs) != 2 || state.Runs[0].Seed != 7 || state.Runs[1].Seed != 8 || *state.Runs[0].Artifact != *state.Runs[1].Artifact {
		t.Fatalf("resume state = %#v", state)
	}
}

func testBatchPlan(journal *CampaignJournal, digest evidence.SHA256, size uint64) CampaignPlan {
	var journalPlan *RunJournalPlan
	parallel := evidence.Uint64String(2)
	if journal != nil {
		value := journal.RunJournalPlan()
		journalPlan = &value
		parallel = value.MaximumPartialRuns
	}
	plan := CampaignPlan{
		Schema: CampaignPlanSchema, Strategy: "seed", Selection: "7-9", SelectionCount: 3, Parallel: parallel,
		Journal:         journalPlan,
		RunTimeoutNanos: evidence.Uint64String(time.Second), OverallTimeoutNanos: evidence.Uint64String(10 * time.Second), TerminateGraceNanos: evidence.Uint64String(100 * time.Millisecond),
		OnFailure: "all", FailureBudget: 1, OutputBytes: 64, WorldTransitionBytes: 1024,
		RunnerBuild: string(evidence.HashBytes([]byte("runner"))), Toolchain: evidence.Toolchain{GoVersion: "go1.26.4", BuildKey: "build", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Prepared: PreparedTargetPlan{
			Path:   ".prepared/build/target",
			Target: evidence.Target{Kind: "go-test", Source: "./pkg", SHA256: digest, Size: evidence.Uint64String(size), Argv: []string{"gomadv3-target"}, BuildTags: []string{"gomad_fixture"}, Adapters: []evidence.TargetAdapter{}, Compatibility: []evidence.CompatibilityPack{}, CapabilityMode: "closure", BuildInfo: evidence.BuildInfo{GoVersion: "go1.26.4", Path: "example.test/pkg.test"}},
		},
		IOProfile:   IOProfilePlan{Name: "gomadv3-deterministic/v1", ImplementationSHA256: deterministicio.Digest(evidence.HashBytes([]byte("io"))), InventorySHA256: deterministicio.Digest(evidence.HashBytes([]byte("inventory")))},
		Environment: []evidence.Environment{{Name: "GOMADV3_IO_PROFILE", Value: "gomadv3-deterministic/v1"}},
		Coverage:    "semantic", RequiredSemanticProbes: []string{"stdlib.os.openfile"}, KeepSuccesses: "none",
	}
	artifacts, err := DeriveArtifactCapacityPlan(plan)
	if err != nil {
		panic(err)
	}
	plan.Artifacts = &artifacts
	return plan
}

func TestBatchJournalRoundTripsChoiceProfileAndRunSummary(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{Root: t.TempDir(), CampaignID: "run-choices", Selection: "7", SelectionCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	digest := evidence.HashBytes([]byte("choices"))
	records := evidence.Uint64String(4)
	branching := evidence.Uint64String(2)
	decisions := evidence.Uint64String(3)
	tapeDigest := evidence.HashBytes([]byte("choice tape"))
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
	if len(opened.Runs) != 1 || opened.Runs[0].ChoiceTraceBranchingRecords == nil || *opened.Runs[0].ChoiceTraceBranchingRecords != 2 || opened.Runs[0].ChoiceTapeSHA256 == nil || *opened.Runs[0].ChoiceDecisions != 3 {
		t.Fatalf("choice run summary = %#v", opened.Runs)
	}
}

func TestChoiceProfileMatchesResumePlanExactly(t *testing.T) {
	plan := CampaignPlan{ChoiceProfile: &ChoiceProfilePlan{Name: choice.Profile, ImplementationSHA256: evidence.HashBytes([]byte("choice implementation")), Limit: 8 << 20}}
	manifest := evidence.ExecutionRecord{ChoiceProfile: &evidence.ChoiceProfile{
		Name: choice.Profile, ImplementationSHA256: plan.ChoiceProfile.ImplementationSHA256,
		Trace: evidence.ChoiceTrace{Limit: 8 << 20},
	}}
	if !choiceProfileMatchesPlan(plan, manifest) {
		t.Fatal("matching choice profile was rejected")
	}
	manifest.ChoiceProfile.Trace.Limit++
	if choiceProfileMatchesPlan(plan, manifest) {
		t.Fatal("changed choice limit was accepted")
	}
	if choiceProfileMatchesPlan(CampaignPlan{}, manifest) || choiceProfileMatchesPlan(plan, evidence.ExecutionRecord{}) {
		t.Fatal("enabled and disabled choice profiles were treated as compatible")
	}
}

func TestOpenBatchRejectsChangedRuns(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-tampered", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendExecution(ExecutionRecord{
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit", ElapsedNanos: 5,
	}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Publish(CampaignSummary{Attempted: 1, Succeeded: 1, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	runs := filepath.Join(journal.Path(), "runs", "00000000000000000000.jsonl")
	if err := os.WriteFile(runs, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenCampaign(journal.Path()); err == nil || !strings.Contains(err.Error(), "identity changed") {
		t.Fatalf("OpenBatch() error = %v", err)
	}
}

func TestBatchJournalPreservesPreparedStateWhenManifestPublicationFails(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-publish-failure", Selection: "7", SelectionCount: 1,
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
	if err := os.WriteFile(filepath.Join(journal.PreparedPath(), "resume-data"), []byte("required for resume"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := journal.CompletePreparation(); err != nil {
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
	if err := os.Mkdir(filepath.Join(journal.Path(), "batch.json"), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := journal.Publish(CampaignSummary{Attempted: 1, Succeeded: 1, StopReason: "seeds_exhausted"}); err == nil {
		t.Fatal("Publish() succeeded with an obstructed batch manifest")
	}
	assertFileContents(t, filepath.Join(journal.PreparedPath(), "resume-data"), "required for resume")
}

func TestBatchJournalRejectsRunBeforeExceedingReaderCapacity(t *testing.T) {
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
	err = journal.AppendExecution(ExecutionRecord{
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: strings.Repeat("x", defaultRunSegmentBytes), Termination: "exit",
	})
	var capacityErr *JournalCapacityError
	if !errors.As(err, &capacityErr) || capacityErr.Limit != JournalLimitSegmentBytes || capacityErr.Outcome != CapacityInfrastructureFailure {
		t.Fatalf("AppendRun() error = %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(journal.Path(), ".partial", "runs"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("active run journal entries = %v after rejected append", entries)
	}
}

func TestBatchJournalPreservesExplicitFailureState(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-failed", Selection: "1", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	if err := journal.BeginPreparation(); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("build failed")
	if err := journal.FailPreparation("target_preparation", failure); err != nil {
		t.Fatal(err)
	}
	const wantFailure = `{"detail":"build failed","reason":"target_preparation","schema_version":5,"state":"failed"}`
	assertFileContents(t, filepath.Join(journal.Path(), ".partial", "preparation", "partial.json"), wantFailure)
	if err := journal.Fail("target_preparation", failure); err != nil {
		t.Fatal(err)
	}
	const wantLifecycleFailure = `{"campaign_id":"run-failed","detail":"build failed","last_stable_state":"planned","reason":"target_preparation","schema":"gomadv3.batch-lifecycle/v1","schema_version":5,"state":"recoverable-failure"}`
	assertFileContents(t, filepath.Join(journal.Path(), ".partial", "batch", "partial.json"), wantLifecycleFailure)
}

func TestRunJournalRejectsInvalidStateTransitions(t *testing.T) {
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-state", Selection: "1", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
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

func assertFileContents(t *testing.T, path, want string) {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != want {
		t.Fatalf("%s = %q, want %q", filepath.Base(path), contents, want)
	}
}
