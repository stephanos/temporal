package runner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
)

func TestInspectReportsArtifactIdentityWithoutPresentation(t *testing.T) {
	published := publishInspectArtifact(t)
	report, err := Inspect(published.Path, InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Kind != "artifact" || report.Artifact == nil || report.Campaign != nil {
		t.Fatalf("report = %#v", report)
	}
	observed := report.Artifact
	if observed.Seed != 7 || observed.Target.Source != "./target" || observed.Outcome.Reason != "nonzero_exit" || observed.Transcript == nil || observed.Transcript.Records != 3 {
		t.Fatalf("artifact report = %#v", observed)
	}
	if !observed.Stdout.Truncated {
		t.Fatalf("artifact details = %#v", observed)
	}
	encoded, err := canonicaljson.CanonicalJSON(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "replay_command") {
		t.Fatalf("inspection contains CLI presentation: %s", encoded)
	}
}

func TestOpenReportsSimulationExplorationEvidence(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	executor := &simulationExplorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit, fail: true}
	config, configDependencies := testConfig(t, preparer, executor, "7", PolicyAll, 1)
	config.Strategy = StrategySimulationExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 1
	config.MaxForcedDecisions = 1
	config.MaxExplorationBytes = 1 << 20
	config.MaxExplorationResultBytes = 1 << 20
	config.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 1, Scenario: 1, Network: 1, Storage: 1, Fault: 1, Crash: 1}
	configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)

	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	report, err := Inspect(summary.Artifacts[0], InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	simulation := report.Artifact.Simulation
	if simulation == nil || simulation.ControllerSHA256 == "" || simulation.ExecutionSHA256 == "" || simulation.CandidateSHA256 == "" || simulation.OutcomeSHA256 == "" || simulation.FailureSHA256 == "" || simulation.Plan.Bytes == 0 || simulation.Plan.SHA256 == "" || simulation.Record.Bytes == 0 || simulation.Record.SHA256 == "" || simulation.Record.Limit != 128<<20 {
		t.Fatalf("simulation inspection = %#v", simulation)
	}
}

func TestOpenReportsMinimizationLineageAndBounds(t *testing.T) {
	artifactPath, _ := publishReplayArtifactForTarget(t, nil, replayArtifactTarget{Choices: true, Simulation: true, ForcedSimulation: true})
	result, err := minimizeWith(context.Background(), MinimizeSpec{
		ArtifactPath: artifactPath, OutputRoot: t.TempDir(), AttemptBudget: 16,
		ToolchainRoot: toolchainRoot(t), Backend: replayNativeProvider(t),
		Replayer: &minimizationReplayer{},
	}, executionDependencies{executor: &minimizationExecutor{}},
	)
	if err != nil {
		t.Fatal(err)
	}
	report, err := Inspect(result.Artifact.Path, InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	minimization := report.Artifact.Minimization
	if minimization == nil || minimization.ParentRecordHash == "" || minimization.Attempts == 0 || minimization.AttemptBudget != 16 || minimization.OriginalForcedDecisions != 2 || minimization.FinalForcedDecisions != 1 || len(minimization.Accepted) != 1 || minimization.Accepted[0].Kind != "fault_entries" || len(minimization.Accepted[0].Removed) != 1 || !minimization.Predicate.ReplayMatch {
		t.Fatalf("minimization inspection = %#v", minimization)
	}
}

func TestOpenWithOptionsProjectsValidatedChoiceTrace(t *testing.T) {
	published := publishInspectArtifact(t)
	report, err := Inspect(published.Path, InspectOptions{Choices: true})
	if err != nil {
		t.Fatal(err)
	}
	choices := report.Artifact.Choices
	if choices == nil || choices.Records != 1 || choices.BranchingRecords != 1 || choices.Runnable != 1 || choices.SelectPoll != 0 || choices.SelectResult != 0 || len(choices.Sites) != 1 {
		t.Fatalf("choice inspection = %#v", choices)
	}
	if choices.Sites[0].Kind != "runnable" || choices.Sites[0].MaximumAlternatives != 2 || choices.Sites[0].Fingerprint == "" {
		t.Fatalf("choice site = %#v", choices.Sites)
	}
}

func TestOpenWithOptionsRejectsBatch(t *testing.T) {
	batch := writeInspectBatchForChoiceTest(t)
	if _, err := Inspect(batch, InspectOptions{Choices: true}); err == nil || !strings.Contains(err.Error(), "traced artifact") {
		t.Fatalf("batch choice inspection error = %v", err)
	}
}

func TestOpenReportsValidatedBatchRuns(t *testing.T) {
	journal, err := campaign.NewCampaignJournal(context.Background(), campaign.CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-inspect", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func(journal *campaign.CampaignJournal) {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	}(journal)
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendExecution(campaign.ExecutionRecord{
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit", ElapsedNanos: 5,
	}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Publish(campaign.CampaignSummary{Attempted: 1, Succeeded: 1, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	report, err := Inspect(journal.Path(), InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Kind != "campaign" || report.Campaign == nil || report.Lifecycle == nil || report.Lifecycle.State != "published" || report.Artifact != nil || report.Campaign.CampaignID != "run-inspect" || len(report.Campaign.Executions) != 1 || report.Campaign.Journal == nil || report.Campaign.Journal.Records != 1 {
		t.Fatalf("report = %#v", report)
	}
}

func TestOpenReportsSimulationExplorationBoundsAndRemainingWork(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	executor := &simulationExplorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit}
	config, configDependencies := testConfig(t, preparer, executor, "7", PolicyAll, 1)
	config.Strategy = StrategySimulationExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 4
	config.MaxForcedDecisions = 1
	config.MaxExplorationBytes = 1 << 20
	config.MaxExplorationResultBytes = 1 << 20
	config.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 2, Scenario: 1, Network: 2, Storage: 2, Fault: 2, Crash: 2}
	configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)

	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	report, err := Inspect(summary.CampaignPath, InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	combined := report.Campaign.SimulationExploration
	if combined == nil || combined.LogicalExecutions != 2 || combined.Pending != 0 || combined.Limits.Scenario != 1 || report.Campaign.SimulationExplorationImplementationSHA256 == "" || report.Campaign.SimulationExplorationChainSHA256 == "" {
		t.Fatalf("simulation exploration inspection = %#v", report.Campaign)
	}
	if len(report.Campaign.Executions) != 2 || report.Campaign.Executions[0].Strategy != string(StrategySimulationExploration) || report.Campaign.Executions[1].ForcedDepth == nil || *report.Campaign.Executions[1].ForcedDepth != 1 {
		t.Fatalf("simulation exploration runs = %#v", report.Campaign.Executions)
	}
}

func TestOpenReportsInterruptedBatchLifecycle(t *testing.T) {
	journal, err := campaign.NewCampaignJournal(context.Background(), campaign.CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-interrupted-inspect", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	})
	report, err := Inspect(journal.Path(), InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Kind != "campaign" || report.Campaign != nil || report.Artifact != nil || report.Lifecycle == nil || report.Lifecycle.State != "planned" || report.Lifecycle.Resumable || report.Lifecycle.Published {
		t.Fatalf("interrupted batch report = %#v", report)
	}
}

// An interrupted simulation exploration reports its committed rounds, its
// pending candidates and the round it started without committing, read from
// its log without changing it.
func TestOpenReportsInterruptedSimulationExplorationPendingAndStagedWork(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	baseExecutor := &simulationExplorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit}
	config, configDependencies := testConfig(t, preparer, simulationExplorationInterruptExecutor{exploration: baseExecutor}, "7", PolicyAll, 1)
	config.Strategy = StrategySimulationExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 4
	config.MaxForcedDecisions = 2
	config.MaxExplorationBytes = 1 << 20
	config.MaxExplorationResultBytes = 1 << 20
	config.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 2, Scenario: 2, Network: 2, Storage: 2, Fault: 2, Crash: 2}
	partial, err := exploreWith(context.Background(), config, configDependencies)
	if err == nil {
		t.Fatal("interrupted simulation exploration succeeded")
	}
	logPath := filepath.Join(partial.CampaignPath, "store", "log")
	before, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	report, err := Inspect(partial.CampaignPath, InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	combined, rounds := report.SimulationExploration, report.Exploration
	if report.Campaign != nil || combined == nil || rounds == nil || combined.Summary.Pending != 1 || len(combined.Pending) != 1 || combined.StagedRound == nil || combined.StagedRound.Index != 1 || combined.StagedRound.Attempted != 1 {
		t.Fatalf("interrupted simulation exploration report = %#v", report)
	}
	if rounds.Strategy != string(StrategySimulationExploration) || len(rounds.Rounds) != 1 || rounds.StagedRound == nil || rounds.StagedRound.Candidates != 1 || rounds.ChainSHA256 != combined.ChainSHA256 || rounds.Rounds[0].RoundBytes == 0 {
		t.Fatalf("interrupted exploration rounds = %#v", rounds)
	}
	after, err := os.ReadFile(logPath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("inspection changed the campaign log: %v", err)
	}
}

func TestOpenReportsAndValidatesRetainedSuccessfulRuns(t *testing.T) {
	journal, err := campaign.NewCampaignJournal(context.Background(), campaign.CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-inspect-success", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func(journal *campaign.CampaignJournal) {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	}(journal)
	retained := publishInspectArtifactAt(t, journal.SuccessesPath(), "run-inspect-success", true)
	relative, err := filepath.Rel(journal.Path(), retained.Path)
	if err != nil {
		t.Fatal(err)
	}
	bytes := record.Uint64String(retained.StoredBytes)
	choiceTrace := retained.Manifest.ChoiceProfile.Trace
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendExecution(campaign.ExecutionRecord{
		SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit", ElapsedNanos: 5,
		SuccessArtifact: &relative, SuccessArtifactBytes: &bytes, SemanticProbes: []string{"stdlib.os.openfile"}, NovelSemanticProbes: []string{"stdlib.os.openfile"},
		ChoiceTraceSHA256: &choiceTrace.SHA256, ChoiceTraceRecords: &choiceTrace.Records, ChoiceTraceBranchingRecords: &choiceTrace.BranchingRecords,
		ChoiceTraceTerminalState: &choiceTrace.TerminalState, ChoiceTapeSHA256: &choiceTrace.TapeSHA256, ChoiceDecisions: &choiceTrace.Decisions,
	}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Publish(campaign.CampaignSummary{Attempted: 1, Succeeded: 1, RetainedSuccesses: 1, RetainedSuccessBytes: retained.StoredBytes, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	report, err := Inspect(journal.Path(), InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Campaign == nil || report.Campaign.RetainedSuccesses != 1 || report.Campaign.RetainedSuccessBytes != retained.StoredBytes || len(report.Campaign.SuccessArtifacts) != 1 || report.Campaign.SuccessArtifacts[0].Path != retained.Path {
		t.Fatalf("batch report = %#v", report.Campaign)
	}
}

func publishInspectArtifact(t *testing.T) artifact.Artifact {
	t.Helper()
	return publishInspectArtifactAt(t, t.TempDir(), "batch-inspect", false)
}

func publishInspectArtifactAt(t *testing.T, root, batchID string, success bool) artifact.Artifact {
	t.Helper()
	targetPath := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(targetPath, []byte("target"), 0o700); err != nil {
		t.Fatal(err)
	}
	world, payloads := record.NoneWorld()
	exitCode := record.Uint64String(2)
	artifactKind := record.ArtifactTargetFailure
	domain := "target"
	reason := "nonzero_exit"
	if success {
		exitCode = 0
		artifactKind = record.ArtifactSuccess
		domain = "success"
		reason = "success"
	}
	transcript := []byte(strings.Repeat("x", 128*3))
	first := sha256.Sum256([]byte("first choice alternative"))
	second := sha256.Sum256([]byte("second choice alternative"))
	decision, err := choice.CanonicalDecision(0, choice.KindRunnable, 24, false, [][sha256.Size]byte{first, second}, second, 0)
	if err != nil {
		t.Fatal(err)
	}
	choiceTrace, err := choice.BuildTrace([]choice.Record{decision.Record()}, choice.TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	choicePayload := choiceTrace.Bytes
	choiceLimit, err := choice.TraceBytes(1)
	if err != nil {
		t.Fatal(err)
	}
	choiceImplementation, err := choice.ImplementationIdentity(strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	targetSHA256, err := record.HashBytes([]byte("target")).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	choiceReplayPlan, err := choice.ProjectReplayPlan(choiceTrace, choice.ExecutionIdentity{
		TargetSHA256: targetSHA256, ToolchainBuildKey: strings.Repeat("a", 64),
		GOOS: "darwin", GOARCH: "arm64", ImplementationSHA256: choiceImplementation,
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest := record.ExecutionRecord{
		SchemaVersion: record.SchemaVersion, ArtifactKind: artifactKind, CreatedAt: "2026-08-12T12:00:00Z", CampaignID: batchID, SelectionOrdinal: 0, Seed: 7, ReplayMode: record.ReplayExact,
		Runner:        record.Runner{RecordContract: record.RecordContract, RunnerBuild: "sha256:runner", HostOS: "darwin", HostArch: "arm64"},
		Toolchain:     record.Toolchain{GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Target:        record.Target{Kind: "go-test", Source: "./target", SHA256: record.HashBytes([]byte("target")), Size: 6, Argv: []string{"gomad3-target"}, BuildTags: []string{"gomad_fixture"}, Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}, BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"}},
		IOProfile:     record.IOProfile{Name: "gomad3-deterministic/v1", ImplementationSHA256: record.HashBytes([]byte("implementation")), Inventory: "{}", InventorySHA256: record.HashBytes([]byte("{}")), Transcript: &record.IOTranscript{Schema: "gomad3.io-transcript/v1", SHA256: record.HashBytes(transcript), Bytes: record.Uint64String(len(transcript)), Records: 3}},
		ChoiceProfile: &record.ChoiceProfile{Name: choice.Profile, ImplementationSHA256: record.SHA256FromSum(choiceImplementation), Trace: record.ChoiceTrace{Schema: "gomad3.choice-trace/v3", SHA256: record.SHA256FromSum(choiceTrace.SHA256), Bytes: record.Uint64String(len(choicePayload)), Records: 1, BranchingRecords: 1, TerminalState: "complete", Limit: record.Uint64String(choiceLimit), TapeSHA256: record.SHA256FromSum(choiceReplayPlan.SHA256), Decisions: 1}},
		Environment:   []record.Environment{{Name: "GOMAD3_CHOICE_PROFILE", Value: choice.Profile}, {Name: "GOMAD3_IO_PROFILE", Value: "gomad3-deterministic/v1"}, {Name: "GOMADSEED", Value: "7"}, {Name: "TZ", Value: "UTC"}}, Limits: record.Limits{ExecutionTimeoutNanos: 1, OverallTimeoutNanos: 2, OutputBytes: 4, WorldTransitionBytes: 64, IOTranscriptBytes: 1 << 20, ChoiceTraceBytes: record.Uint64String(choiceLimit)}, World: world,
		Outcome: record.Outcome{Domain: domain, Reason: reason, Termination: "exit", ExitCode: &exitCode},
		Streams: record.Streams{
			Stdout: record.Stream{FullSHA256: record.HashBytes([]byte("long output")), TotalBytes: 11, RetainedBytes: 4, DiscardedBytes: 7, Truncated: true},
			Stderr: record.Stream{FullSHA256: record.HashBytes(nil), TotalBytes: 0, RetainedBytes: 0},
		},
		Host: record.Host{StartedAt: "2026-08-12T12:00:00Z", FinishedAt: "2026-08-12T12:00:01Z", ElapsedNanos: 1},
	}
	published, err := artifact.PublishArtifact(artifact.Store{Root: root}, artifact.ArtifactInput{
		Manifest: manifest, TargetPath: targetPath, Stdout: []byte("long"), Stderr: nil, IOTranscript: transcript, ChoiceTrace: choicePayload, World: payloads,
	})
	if err != nil {
		t.Fatal(err)
	}
	return published
}

func writeInspectBatchForChoiceTest(t *testing.T) string {
	t.Helper()
	journal, err := campaign.NewCampaignJournal(context.Background(), campaign.CampaignConfig{Root: t.TempDir(), CampaignID: "run-choice-batch", Selection: "7", SelectionCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = journal.Close() })
	if err := journal.StartExecutions(); err != nil {
		t.Fatal(err)
	}
	if err := journal.AppendExecution(campaign.ExecutionRecord{SelectionOrdinal: 0, Seed: 7, Domain: "success", Reason: "success", Termination: "exit"}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Publish(campaign.CampaignSummary{Attempted: 1, Succeeded: 1, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	return journal.Path()
}

func TestInspectCleanupPreservesNoChoiceArtifact(t *testing.T) {
	published := publishInspectCleanupArtifact(t)
	exitCode := uint64(0)
	want := Inspection{
		Schema: "gomad3.inspect/v8", Kind: "artifact", Path: published.Path,
		Artifact: &ArtifactInspection{
			ArtifactKind: "gomad3.success/v1", RecordHash: published.Manifest.RecordHash, CampaignID: "inspect-cleanup", Seed: 7, ReplayMode: "exact",
			Runner:    record.Runner{RecordContract: record.RecordContract, RunnerBuild: "sha256:runner", HostOS: "darwin", HostArch: "arm64"},
			Toolchain: record.Toolchain{GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: "darwin", TargetGOARCH: "arm64"},
			Target: TargetReport{
				Kind: "go-test", Source: "./target", SHA256: record.HashBytes([]byte("target")), Size: 6,
				Argv: []string{"gomad3-target"}, BuildTags: []string{"gomad_fixture"},
				BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"}, CapabilityMode: "closure", Sharing: "private",
			},
			Outcome:    OutcomeReport{Domain: "success", Reason: "success", Termination: "exit", ExitCode: &exitCode, FailureSignature: published.Manifest.Outcome.FailureSignature},
			Transcript: &Transcript{Schema: "gomad3.io-transcript/v1", SHA256: record.HashBytes([]byte(strings.Repeat("x", 384))), Bytes: 384, Records: 3},
			Stdout:     StreamReport{FullSHA256: record.HashBytes([]byte("long output")), TotalBytes: 11, RetainedBytes: 4, DiscardedBytes: 7, Truncated: true},
			Stderr:     StreamReport{FullSHA256: record.HashBytes(nil)},
		},
	}
	wantBytes, err := canonicaljson.CanonicalJSON(want)
	if err != nil {
		t.Fatal(err)
	}
	for _, sharing := range []string{"private", "shared"} {
		t.Run(sharing, func(t *testing.T) {
			if sharing == "shared" {
				if err := os.Link(filepath.Join(published.Path, "target"), filepath.Join(t.TempDir(), "shared-target")); err != nil {
					t.Fatal(err)
				}
				want.Artifact.Target.Sharing = "shared"
				wantBytes, err = canonicaljson.CanonicalJSON(want)
				if err != nil {
					t.Fatal(err)
				}
			}
			report, err := Inspect(published.Path, InspectOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(report, want) {
				t.Fatalf("inspection = %#v, artifact = %#v, want %#v", report, report.Artifact, want.Artifact)
			}
			encoded, err := canonicaljson.CanonicalJSON(report)
			if err != nil {
				t.Fatal(err)
			}
			if string(encoded) != string(wantBytes) {
				t.Fatalf("canonical inspection = %s, want %s", encoded, wantBytes)
			}
		})
	}
	report, err := Inspect(published.Path, InspectOptions{Choices: true})
	if err == nil || err.Error() != "artifact has no choice trace" || reflect.TypeOf(err) != reflect.TypeOf(errors.New("")) || !reflect.DeepEqual(report, Inspection{}) {
		t.Fatalf("choice inspection = %#v, error = %v", report, err)
	}
}

func TestInspectCleanupPreservesEarlierErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	report, err := Inspect(missing, InspectOptions{Choices: true})
	pathError, direct := err.(*os.PathError)
	if !direct || !errors.Is(err, os.ErrNotExist) || pathError.Op != "lstat" || pathError.Path != missing || !reflect.DeepEqual(report, Inspection{}) {
		t.Fatalf("missing inspection = %#v, error = %v", report, err)
	}
	published := publishInspectCleanupArtifact(t)
	if err := os.Mkdir(filepath.Join(published.Path, "store"), 0o700); err != nil {
		t.Fatal(err)
	}
	report, err = Inspect(published.Path, InspectOptions{Choices: true})
	if err == nil || err.Error() != "inspection path contains conflicting artifact records" || reflect.TypeOf(err) != reflect.TypeOf(errors.New("")) || !reflect.DeepEqual(report, Inspection{}) {
		t.Fatalf("conflicting inspection = %#v, error = %v", report, err)
	}
}

func publishInspectCleanupArtifact(t *testing.T) artifact.Artifact {
	t.Helper()
	published := publishInspectArtifactAt(t, t.TempDir(), "inspect-cleanup", true)
	manifest := published.Manifest
	manifest.ChoiceProfile = nil
	manifest.Environment = manifest.Environment[1:]
	manifest.Limits.ChoiceTraceBytes = 0
	world, payloads := record.NoneWorld()
	manifest.World = world
	result, err := artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, artifact.ArtifactInput{
		Manifest: manifest, TargetPath: filepath.Join(published.Path, "target"), Stdout: []byte("long"),
		IOTranscript: []byte(strings.Repeat("x", 384)), World: payloads,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestInspectionCapacityProjectionPreservesReports(t *testing.T) {
	journal := campaign.ExecutionJournalPlan{MaximumExecutions: 1, MaximumBytes: 2, SegmentBytes: 3, SegmentRecords: 4, MaximumSegments: 5, MaximumPartialExecutions: 6, CapacityOutcome: "infrastructure_failure"}
	capacity := campaign.ArtifactCapacityPlan{FailureArtifacts: 11, FailureBytes: 12, SuccessArtifacts: 13, SuccessBytes: 14, TotalBytes: 15, TranscriptBytes: 16, FailureOutcome: "failure-bound", SuccessOutcome: "success-bound"}
	plan := projectCampaignPlan(openedCampaignPlan{plan: campaign.CampaignPlan{Journal: &journal, Artifacts: &capacity}})
	inspection, err := projectCampaign(campaign.Campaign{Record: campaign.CampaignRecord{Artifacts: &capacity}, Journal: &campaign.ExecutionJournalInfo{Limits: journal}})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		value any
		want  string
	}{
		{journal, `{"maximum_executions":"1","maximum_bytes":"2","segment_bytes":"3","segment_records":"4","maximum_segments":"5","maximum_partial_executions":"6","capacity_outcome":"infrastructure_failure"}`},
		{plan.Journal, `{"maximum_executions":"1","maximum_bytes":"2","segment_bytes":"3","segment_records":"4","maximum_segments":"5","maximum_partial_executions":"6","capacity_outcome":"infrastructure_failure"}`},
		{inspection.Journal.Limits, `{"maximum_executions":"1","maximum_bytes":"2","segment_bytes":"3","segment_records":"4","maximum_segments":"5","maximum_partial_executions":"6","capacity_outcome":"infrastructure_failure"}`},
		{capacity, `{"failure_artifacts":"11","failure_bytes":"12","success_artifacts":"13","success_bytes":"14","total_bytes":"15","transcript_bytes":"16","failure_outcome":"failure-bound","success_outcome":"success-bound"}`},
		{plan.ArtifactCapacity, `{"failure_artifacts":"11","failure_bytes":"12","success_artifacts":"13","success_bytes":"14","total_bytes":"15","transcript_bytes":"16","failure_outcome":"failure-bound","success_outcome":"success-bound"}`},
		{inspection.ArtifactCapacity, `{"failure_artifacts":"11","failure_bytes":"12","success_artifacts":"13","success_bytes":"14","total_bytes":"15","transcript_bytes":"16","failure_outcome":"failure-bound","success_outcome":"success-bound"}`},
	} {
		data, err := json.Marshal(test.value)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != test.want {
			t.Fatalf("capacity report = %s, want %s", data, test.want)
		}
	}
	absent, err := projectCampaign(campaign.Campaign{})
	if err != nil {
		t.Fatal(err)
	}
	if absent.ArtifactCapacity != nil || absent.Journal != nil {
		t.Fatalf("absent pointers changed: %+v", absent)
	}
	capacity.FailureBytes = 99
	journal.MaximumBytes = 99
	data, err := json.Marshal(plan.ArtifactCapacity)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"failure_artifacts":"11","failure_bytes":"12","success_artifacts":"13","success_bytes":"14","total_bytes":"15","transcript_bytes":"16","failure_outcome":"failure-bound","success_outcome":"success-bound"}` {
		t.Fatalf("report aliases source: %s", data)
	}
}
