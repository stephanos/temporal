package runner

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomadv3/artifact"
	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomadv3/record"
	"go.temporal.io/server/tools/gomadv3/runner/internal/campaign"
	simulationengine "go.temporal.io/server/tools/gomadv3/runner/internal/exploration/simulation"
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
	config := testConfig(t, preparer, executor, "7", PolicyAll, 1)
	config.Strategy = StrategySimulationExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 1
	config.MaxForcedDecisions = 1
	config.MaxExplorationBytes = 1 << 20
	config.MaxExplorationResultBytes = 1 << 20
	config.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 1, Scenario: 1, Network: 1, Storage: 1, Fault: 1, Crash: 1}

	summary, err := Explore(context.Background(), config)
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
	result, err := Minimize(context.Background(), MinimizeSpec{
		ArtifactPath: artifactPath, OutputRoot: t.TempDir(), AttemptBudget: 16,
		ToolchainRoot: toolchainRoot(t), SupervisorCommand: []string{"unused"},
		Executor: &minimizationExecutor{}, Replayer: &minimizationReplayer{},
	})
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
	defer journal.Close()
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
	config := testConfig(t, preparer, executor, "7", PolicyAll, 1)
	config.Strategy = StrategySimulationExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 4
	config.MaxForcedDecisions = 1
	config.MaxExplorationBytes = 1 << 20
	config.MaxExplorationResultBytes = 1 << 20
	config.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 2, Scenario: 1, Network: 2, Storage: 2, Fault: 2, Crash: 2}

	summary, err := Explore(context.Background(), config)
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

func TestOpenReportsInterruptedSimulationExplorationPendingAndStagedWork(t *testing.T) {
	journal, err := campaign.NewCampaignJournal(context.Background(), campaign.CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-interrupted-combined", Strategy: string(StrategySimulationExploration), Selection: "7", SelectionCount: 1,
		MaxExecutions: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	})
	state, err := simulationengine.New(simulationengine.Config{
		ExecutionSHA256: record.HashBytes([]byte("execution")), ControllerSHA256: simulationengine.ImplementationSHA256(),
		BaseSeed: 7, Parallel: 2, MaxExecutions: 8, MaxForcedDecisions: 4, MaxExplorationBytes: 1 << 20, MaxResultBytes: 1 << 20, FailureBudget: 4,
		Limits: simulationengine.DimensionLimits{Runtime: 1, Scenario: 2, Network: 3, Storage: 4, Fault: 5, Crash: 6},
	})
	if err != nil {
		t.Fatal(err)
	}
	exploration, err := campaign.NewSimulationExplorationJournal(context.Background(), journal.Path(), state, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	round, _ := state.NextRound()
	staged, err := exploration.StageRound(round)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := staged.BeginExecution(0, 7); err != nil {
		t.Fatal(err)
	}

	report, err := Inspect(journal.Path(), InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	combined := report.SimulationExploration
	if report.Campaign != nil || combined == nil || combined.Summary.Pending != 1 || len(combined.Pending) != 1 || combined.Pending[0].SHA256 != round.Candidates[0].SHA256 || combined.StagedRound == nil || combined.StagedRound.Attempted != 1 {
		t.Fatalf("interrupted simulation exploration report = %#v", report)
	}
	if _, err := os.Stat(staged.Path()); err != nil {
		t.Fatalf("inspection mutated staged round: %v", err)
	}
}

func TestOpenReportsAndValidatesRetainedSuccessfulRuns(t *testing.T) {
	journal, err := campaign.NewCampaignJournal(context.Background(), campaign.CampaignConfig{
		Root: t.TempDir(), CampaignID: "run-inspect-success", Selection: "7", SelectionCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
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
		Target:        record.Target{Kind: "go-test", Source: "./target", SHA256: record.HashBytes([]byte("target")), Size: 6, Argv: []string{"gomadv3-target"}, BuildTags: []string{"gomad_fixture"}, Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}, BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"}},
		IOProfile:     record.IOProfile{Name: "gomadv3-deterministic/v1", ImplementationSHA256: record.HashBytes([]byte("implementation")), Inventory: "{}", InventorySHA256: record.HashBytes([]byte("{}")), Transcript: &record.IOTranscript{Schema: "gomadv3.io-transcript/v1", SHA256: record.HashBytes(transcript), Bytes: record.Uint64String(len(transcript)), Records: 3}},
		ChoiceProfile: &record.ChoiceProfile{Name: choice.Profile, ImplementationSHA256: record.SHA256FromSum(choiceImplementation), Trace: record.ChoiceTrace{Schema: "gomadv3.choice-trace/v2", SHA256: record.SHA256FromSum(choiceTrace.SHA256), Bytes: record.Uint64String(len(choicePayload)), Records: 1, BranchingRecords: 1, TerminalState: "complete", Limit: record.Uint64String(choiceLimit), TapeSHA256: record.SHA256FromSum(choiceReplayPlan.SHA256), Decisions: 1}},
		Environment:   []record.Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "GOMADV3_CHOICE_PROFILE", Value: choice.Profile}, {Name: "GOMADV3_IO_PROFILE", Value: "gomadv3-deterministic/v1"}, {Name: "TZ", Value: "UTC"}}, Limits: record.Limits{ExecutionTimeoutNanos: 1, OverallTimeoutNanos: 2, OutputBytes: 4, WorldTransitionBytes: 64, IOTranscriptBytes: 1 << 20, ChoiceTraceBytes: record.Uint64String(choiceLimit)}, World: world,
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
