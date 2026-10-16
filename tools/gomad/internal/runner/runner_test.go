package runner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	backend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	testbackend "go.temporal.io/server/tools/gomad/internal/backends/testfixture"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/evidence/record/testfixture"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/model/world"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/exploration"
	campaignoptions "go.temporal.io/server/tools/gomad/internal/runner/options"
)

func TestRunPreparesOnceBoundsParallelismAndGroupsMatchingFailures(t *testing.T) {
	preparer := newFakePreparer(t)
	executor := &fakeExecutor{result: func(seed uint64) backend.ExecutionResult {
		if seed%2 == 0 {
			return processResult(2, "same failure", "")
		}
		return processResult(0, "success", "")
	}}
	config, dependencies := testConfig(t, preparer, executor, "1-6", PolicyAll, 2)
	dependencies = scriptedPreparationDependencies(t, config.Preparer, dependencies.executor)
	summary, err := exploreWith(context.Background(), config, dependencies)
	if err != nil {
		t.Fatal(err)
	}
	if preparer.calls != 1 {
		t.Fatalf("preparation calls = %d, want 1", preparer.calls)
	}
	if executor.maximumActive > 2 {
		t.Fatalf("maximum active = %d, want <= 2", executor.maximumActive)
	}
	if summary.Attempted != 6 || summary.Succeeded != 3 || summary.Failures != 3 || summary.DistinctFailures != 1 || summary.StopReason != StopSeedsExhausted {
		t.Fatalf("summary = %#v", summary)
	}
	if len(summary.Artifacts) != 1 {
		t.Fatalf("artifacts = %v", summary.Artifacts)
	}
	if _, err := artifact.OpenArtifact(summary.Artifacts[0]); err != nil {
		t.Fatal(err)
	}
	opened, err := artifact.OpenArtifact(summary.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	if opened.Manifest().Target.BuildTags == nil {
		t.Fatal("empty target build tags encoded as null")
	}
	if _, err := campaign.OpenCampaign(summary.CampaignPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(summary.CampaignPath, ".prepared")); !os.IsNotExist(err) {
		t.Fatalf("prepared target retained after publication: %v", err)
	}
	if got := executor.environments(); len(got) != 6 {
		t.Fatalf("target environments = %v", got)
	} else {
		for _, environment := range got {
			if len(environment) != 4 || environment[1] != "GOMAD_IO_PROFILE=gomad-deterministic/v1" || environment[2] != "MODE=test" || environment[3] != "TZ=UTC" || !strings.HasPrefix(environment[0], "GOMADSEED=") {
				t.Fatalf("target environment = %v", environment)
			}
		}
	}
	if !allUnique(executor.directories()) {
		t.Fatalf("working directories were reused: %v", executor.directories())
	}
}

func TestRunReportsPreparationProgressAndCompletedCounts(t *testing.T) {
	config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(seed uint64) backend.ExecutionResult {
		if seed == 2 {
			return processResult(2, "failure", "")
		}
		return processResult(0, "success", "")
	}}, "1-3", PolicyAll, 2)
	var progress []CampaignEvent
	config.Progress = func(update CampaignEvent) error {
		progress = append(progress, update)
		return nil
	}
	configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if len(progress) < 4 || progress[0].Phase != ProgressPreparing || progress[0].CampaignPath == "" {
		t.Fatalf("progress = %#v", progress)
	}
	var observedRunning bool
	for _, update := range progress {
		if update.Phase == ProgressRunning && update.Running > 0 {
			observedRunning = true
		}
	}
	last := progress[len(progress)-1]
	if !observedRunning || last.Phase != ProgressComplete || last.Attempted != summary.Attempted || last.Running != 0 || last.Succeeded != 2 || last.Failures != 1 || last.DistinctFailures != 1 || len(last.Artifacts) != 1 {
		t.Fatalf("progress = %#v, summary = %#v", progress, summary)
	}
}

func TestRunReportsPeriodicProgressWhileTargetIsRunning(t *testing.T) {
	executor := &progressGatedExecutor{started: make(chan struct{}), release: make(chan struct{})}
	config, configDependencies := testConfig(t, newFakePreparer(t), executor, "1", PolicyAll, 1)
	release := sync.OnceFunc(func() { close(executor.release) })
	t.Cleanup(release)
	config.ProgressInterval = time.Millisecond
	updates := make(chan CampaignEvent, 16)
	config.Progress = func(update CampaignEvent) error {
		select {
		case updates <- update:
		default:
		}
		return nil
	}
	completed := make(chan error, 1)
	startupDeadline := time.NewTimer(config.OverallTimeout + config.TerminateGrace)
	defer startupDeadline.Stop()
	configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
	go func() {
		_, err := exploreWith(context.Background(), config, configDependencies)
		completed <- err
	}()
	if err := waitForProgressStart(executor.started, completed, startupDeadline.C); err != nil {
		t.Fatal(err)
	}
	heartbeats := 0
	deadline := time.After(time.Second)
	for heartbeats < 2 {
		select {
		case update := <-updates:
			if update.Phase == ProgressRunning && update.Running == 1 {
				heartbeats++
			}
		case <-deadline:
			t.Fatal("periodic running progress was not reported")
		}
	}
	release()
	if err := <-completed; err != nil {
		t.Fatal(err)
	}
}

func TestRunReportsPassiveSemanticCoverage(t *testing.T) {
	executor := &fakeExecutor{result: func(uint64) backend.ExecutionResult {
		result := processResult(0, "", "")
		result.IOTranscript = semanticTranscript(t, "stdlib.os.openfile")
		return result
	}}
	config, configDependencies := testConfig(t, newFakePreparer(t), executor, "1", PolicyAll, 1)
	config.Coverage = campaignoptions.CoverageSemantic
	configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if summary.SemanticCoverage == nil || len(summary.SemanticCoverage.Probes) != 1 || summary.SemanticCoverage.Probes[0] != "stdlib.os.openfile" {
		t.Fatalf("summary coverage = %#v", summary.SemanticCoverage)
	}
}

func TestRunFailsWhenRequiredSemanticProbeIsMissing(t *testing.T) {
	config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
	config.Coverage = campaignoptions.CoverageSemantic
	config.RequiredSemanticProbes = []string{"stdlib.os.openfile"}
	configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
	_, err := exploreWith(context.Background(), config, configDependencies)
	var missing *record.MissingSemanticProbesError
	if !errors.As(err, &missing) || len(missing.Probes) != 1 || missing.Probes[0] != "stdlib.os.openfile" {
		t.Fatalf("exploreWith() error = %v", err)
	}
}

func TestRunMergesParallelCompletionsInSelectionOrdinalOrder(t *testing.T) {
	executor := newOutOfOrderExecutor(t)
	config, configDependencies := testConfig(t, newFakePreparer(t), executor, "1-3", PolicyAll, 3)
	config.Coverage = campaignoptions.CoverageSemantic
	config.KeepSuccesses = KeepSuccessesNovel
	config.SuccessArtifactLimit = 2
	config.SuccessBytesLimit = 64 << 20
	configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	batch, err := campaign.OpenCampaign(summary.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := []record.Uint64String{batch.Executions[0].Seed, batch.Executions[1].Seed, batch.Executions[2].Seed}; !slices.Equal(got, []record.Uint64String{1, 2, 3}) {
		t.Fatalf("batch merge order = %v", got)
	}
	if batch.Executions[0].SuccessArtifact == nil || batch.Executions[1].SuccessArtifact != nil || batch.Executions[2].SuccessArtifact == nil {
		t.Fatalf("ordered novelty retention = %#v", batch.Executions)
	}
}

func TestRunCollectsBoundedQualificationEvidenceForOneSeed(t *testing.T) {
	executor := &fakeExecutor{result: func(uint64) backend.ExecutionResult {
		result := processResult(0, "stdout", "stderr")
		result.IOTranscript = semanticTranscript(t, "stdlib.os.openfile")
		return result
	}}
	config, configDependencies := testConfig(t, newFakePreparer(t), executor, "7", PolicyAll, 1)
	config.Coverage = campaignoptions.CoverageSemantic
	config.CollectExecutionEvidence = true
	configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if summary.ExecutionEvidence == nil || summary.ExecutionEvidence.Seed != 7 || summary.ExecutionEvidence.Target.SHA256 == "" || summary.ExecutionEvidence.Stdout.FullSHA256 != record.HashBytes([]byte("stdout")) || summary.ExecutionEvidence.Stderr.FullSHA256 != record.HashBytes([]byte("stderr")) || summary.ExecutionEvidence.IOTranscriptRecords != 1 || summary.ExecutionEvidence.SemanticCoverage.Digest == "" {
		t.Fatalf("execution evidence = %#v", summary.ExecutionEvidence)
	}
}

func TestExecutionEvidenceIgnoresAggregateCoordinatorDeadlineAdjustment(t *testing.T) {
	var runRecords []record.ExecutionEvidence
	for _, overallTimeout := range []time.Duration{9 * time.Second, 10 * time.Second} {
		config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "7", PolicyAll, 1)
		config.Coverage = campaignoptions.CoverageSemantic
		config.CollectExecutionEvidence = true
		config.OverallTimeout = overallTimeout
		summary, err := exploreWith(context.Background(), config, configDependencies)
		if err != nil {
			t.Fatal(err)
		}
		runRecords = append(runRecords, *summary.ExecutionEvidence)
	}
	first, err := json.Marshal(runRecords[0])
	if err != nil {
		t.Fatal(err)
	}
	second, err := json.Marshal(runRecords[1])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("evidence changed with aggregate deadline:\n%s\n%s", first, second)
	}
}

func TestRunResolvesRelativeArtifactRootBeforeTargetPreparation(t *testing.T) {
	workingDirectory := t.TempDir()
	originalDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(workingDirectory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalDirectory); err != nil {
			t.Error(err)
		}
	})
	preparer := newFakePreparer(t)
	executor := &fakeExecutor{}
	config, configDependencies := testConfig(t, preparer, executor, "7", PolicyAll, 1)
	config.Artifacts = "artifacts"
	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(summary.CampaignPath) {
		t.Fatalf("batch path = %q, want absolute path", summary.CampaignPath)
	}
	for _, directory := range executor.directories() {
		if !filepath.IsAbs(directory) {
			t.Fatalf("target working directory = %q, want absolute path", directory)
		}
	}
}

func TestFailureArtifactCapacityRejectsBeforePublication(t *testing.T) {
	signature := record.HashBytes([]byte("existing"))
	distinct := map[record.SHA256]string{signature: "/existing"}
	config := CampaignSpec{failureArtifactLimit: 1, failureBytesLimit: 10}
	storedBytes := uint64(5)
	storeRoot := t.TempDir()
	_, err := publishBoundedFailureArtifact(
		context.Background(), campaignRequestFromSpec(config), storeRoot, record.HashBytes([]byte("new")), distinct, &storedBytes, artifact.ArtifactInput{},
	)
	var capacityErr *campaign.ArtifactCapacityError
	if !errors.As(err, &capacityErr) || capacityErr.Limit != campaign.ArtifactLimitFailureCount || capacityErr.Outcome != campaign.CapacityInfrastructureFailure {
		t.Fatalf("publishBoundedFailureArtifact() error = %v", err)
	}
	entries, err := os.ReadDir(storeRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("artifact store mutated after capacity rejection: %v", entries)
	}
}

func TestRunFirstFailureCancelsActiveTargetsWithoutPublishingThem(t *testing.T) {
	preparer := newFakePreparer(t)
	executor := newFirstFailureExecutor(3)
	config, configDependencies := testConfig(t, preparer, executor, "1-10", PolicyFirst, 3)
	configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Attempted != 3 || summary.Failures != 1 || summary.Cancelled != 2 || summary.DistinctFailures != 1 || summary.StopReason != StopFirstFailure {
		t.Fatalf("summary = %#v", summary)
	}
	if len(summary.Artifacts) != 1 {
		t.Fatalf("artifacts = %v", summary.Artifacts)
	}
	partials, err := os.ReadDir(filepath.Join(summary.CampaignPath, ".partial"))
	if err != nil {
		t.Fatal(err)
	}
	if len(partials) != 2 {
		t.Fatalf("cancelled target partials = %v, want 2", partials)
	}
}

func TestRunBudgetCountsDistinctSignatures(t *testing.T) {
	preparer := newFakePreparer(t)
	executor := &fakeExecutor{result: func(seed uint64) backend.ExecutionResult {
		output := "same"
		if seed == 4 {
			output = "different"
		}
		return processResult(1, output, "")
	}}
	config, configDependencies := testConfig(t, preparer, executor, "1-10", PolicyBudget, 1)
	config.FailureBudget = 2
	configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Attempted != 4 || summary.Failures != 4 || summary.DistinctFailures != 2 || summary.StopReason != StopFailureBudget {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestRunPublishesConnectedWorldBundle(t *testing.T) {
	core, err := world.New(world.Config{Seed: 7, Limits: world.Limits{MaxRequests: 10, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 10, MaxPayloadBytes: 1024, MaxStringBytes: 64}})
	if err != nil {
		t.Fatal(err)
	}
	initial := core.Snapshot()
	if _, err := core.Quiesce(); err != nil {
		t.Fatal(err)
	}
	recording, err := world.EncodeRecording(world.Recording{Initial: initial, Final: core.Snapshot(), Terminal: world.Terminal{Kind: world.TerminalIdle}})
	if err != nil {
		t.Fatal(err)
	}
	config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(uint64) backend.ExecutionResult {
		result := processResult(1, "failure", "")
		result.WorldRecord = recording
		return result
	}}, "7", PolicyAll, 1)
	config.WorldTransitionLimit = 1 << 20
	configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := artifact.OpenArtifact(summary.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	if opened.Manifest().World.Initial.Schema != "gomad.world.snapshot/v1" || opened.Manifest().World.Transitions.Count != 1 || opened.Manifest().World.Terminal.Kind != "idle" {
		t.Fatalf("recorded World = %#v", opened.Manifest().World)
	}
}

func TestRunClassifiesConnectedWorldDeadlock(t *testing.T) {
	core, err := world.New(world.Config{Seed: 7, Limits: world.Limits{MaxRequests: 10, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 10, MaxPayloadBytes: 1024, MaxStringBytes: 64}})
	if err != nil {
		t.Fatal(err)
	}
	initial := core.Snapshot()
	if _, err := core.Register(world.Request{Kind: "wait", Resource: world.ResourceID{Adapter: "memory", Kind: "cell", Key: "a"}}); err != nil {
		t.Fatal(err)
	}
	if result, err := core.Quiesce(); err != nil || result.Kind != world.QuiescenceDeadlock {
		t.Fatalf("Quiesce() = %#v, %v", result, err)
	}
	recording, err := world.EncodeRecording(world.Recording{Initial: initial, Final: core.Snapshot(), Terminal: world.Terminal{Kind: world.TerminalDeadlock}})
	if err != nil {
		t.Fatal(err)
	}
	config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(uint64) backend.ExecutionResult {
		result := processResult(0, "", "")
		result.WorldRecord = recording
		return result
	}}, "7", PolicyAll, 1)
	config.WorldTransitionLimit = 1 << 20
	configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Failures != 1 || len(summary.Artifacts) != 1 {
		t.Fatalf("summary = %#v", summary)
	}
	opened, err := artifact.OpenArtifact(summary.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	if opened.Manifest().Outcome.Reason != "world_deadlock" || opened.Manifest().World.Terminal.Kind != "deadlock" {
		t.Fatalf("World deadlock manifest = %#v", opened.Manifest())
	}
}

func TestRunCountsConnectedWorldReplayDivergence(t *testing.T) {
	core, err := world.New(world.Config{Seed: 7, Limits: world.Limits{MaxRequests: 10, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 10, MaxPayloadBytes: 1024, MaxStringBytes: 64}})
	if err != nil {
		t.Fatal(err)
	}
	initial := core.Snapshot()
	recording, err := world.EncodeRecording(world.Recording{
		Initial: initial, Final: core.Snapshot(), Terminal: world.Terminal{Kind: world.TerminalReplayDivergence, Detail: "transition 3"},
	})
	if err != nil {
		t.Fatal(err)
	}
	config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(uint64) backend.ExecutionResult {
		result := processResult(0, "", "")
		result.WorldRecord = recording
		return result
	}}, "7", PolicyAll, 1)
	config.WorldTransitionLimit = 1 << 20
	configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Failures != 1 || summary.ReplayDivergences != 1 {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestRunRejectsInvalidConnectedWorldBeforePublication(t *testing.T) {
	core, err := world.New(world.Config{Seed: 7, Limits: world.Limits{MaxRequests: 10, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 10, MaxPayloadBytes: 1024, MaxStringBytes: 64}})
	if err != nil {
		t.Fatal(err)
	}
	initial := core.Snapshot()
	recording, err := world.EncodeRecording(world.Recording{Initial: initial, Final: initial, Terminal: world.Terminal{Kind: world.TerminalInvalidInput, Detail: "fixture"}})
	if err != nil {
		t.Fatal(err)
	}
	recording[len(recording)-1] ^= 1
	config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(uint64) backend.ExecutionResult {
		result := processResult(1, "failure", "")
		result.WorldRecord = recording
		return result
	}}, "7", PolicyAll, 1)
	config.WorldTransitionLimit = 1 << 20
	configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
	summary, err := exploreWith(context.Background(), config, configDependencies)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "world_record" {
		t.Fatalf("exploreWith() error = %#v", err)
	}
	if len(summary.Artifacts) != 1 {
		t.Fatalf("Runner failure artifacts = %v, want 1", summary.Artifacts)
	}
	opened, openErr := artifact.OpenArtifact(summary.Artifacts[0])
	if openErr != nil {
		t.Fatal(openErr)
	}
	if opened.Manifest().ArtifactKind != record.ArtifactRunnerFailure || opened.Manifest().Outcome.Reason != "world_record" || opened.Manifest().ReplayMode != record.ReplayNone {
		t.Fatalf("Runner failure manifest = %#v", opened.Manifest())
	}
}

func TestRunRejectsPreparedTargetMutationBeforeFailurePublication(t *testing.T) {
	config, configDependencies := testConfig(t, newFakePreparer(t), mutatingExecutor{}, "1", PolicyAll, 1)
	configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
	summary, err := exploreWith(context.Background(), config, configDependencies)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "prepared_target_integrity" {
		t.Fatalf("exploreWith() error = %#v", err)
	}
	if len(summary.Artifacts) != 0 {
		t.Fatalf("target mutation published replayable artifacts: %v", summary.Artifacts)
	}
}

func TestValidateConfigAcceptsReadOnlyMountsWithoutProfile(t *testing.T) {
	config, _ := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyFirst, 1)
	config.Environment = nil
	config.IOROMounts = []string{t.TempDir() + "=schema"}
	config.Target.WorkingDir = t.TempDir()
	if _, _, err := validateConfig(config); err != nil {
		t.Fatal(err)
	}
}

func TestRunPassesCanonicalReadOnlyMountsToExecutor(t *testing.T) {
	source := t.TempDir()
	executor := &fakeExecutor{}
	config, configDependencies := testConfig(t, newFakePreparer(t), executor, "1", PolicyAll, 1)
	config.Environment = nil
	config.RunnerBuild = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	config.IOROMounts = []string{source + "=schema"}
	config.Target = target.Spec{
		Kind: target.KindGoTest, Source: "./pkg", Args: []string{"-test.run=^TestScenario$"}, WorkingDir: t.TempDir(),
	}
	config.Preparer = profileFakePreparer(t, "-test.run=^TestScenario$")
	if _, err := exploreWith(context.Background(), config, configDependencies); err != nil {
		t.Fatal(err)
	}
	if len(executor.requests) != 1 || executor.requests[0].IO == nil || executor.requests[0].IO.ReadOnlyMount == nil || len(executor.requests[0].IO.ReadOnlyMount.Mappings) != 1 || executor.requests[0].IO.ReadOnlyMount.Mappings[0].Source != source || executor.requests[0].IO.ReadOnlyMount.Mappings[0].Target != "/schema" {
		t.Fatalf("executor mounts = %#v", executor.requests)
	}
}

func TestRunPassesChoiceProfileToExecutorAndArtifact(t *testing.T) {
	limit := choiceTraceLimit(t, 1)
	preparer := newFakePreparer(t)
	implementation, err := choice.ImplementationIdentity(preparer.prepared.BuildKey)
	if err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{result: func(uint64) backend.ExecutionResult {
		result := processResult(1, "failure", "")
		result.ChoiceTrace = completeChoiceTrace(t, preparer.prepared.BuildKey, limit, nil)
		return result
	}}
	config, configDependencies := testConfig(t, preparer, executor, "1", PolicyAll, 1)
	config.ChoiceTraceLimit = limit
	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if len(executor.requests) != 1 || executor.requests[0].Choice == nil || executor.requests[0].Choice.Mode != choice.ModeRecord || executor.requests[0].Choice.Profile != choice.Profile || executor.requests[0].Choice.ImplementationSHA256 != implementation || executor.requests[0].Choice.ExecutionIdentity.ImplementationSHA256 != implementation || executor.requests[0].Choice.Limit != limit {
		t.Fatalf("executor choice capability = %#v", executor.requests)
	}
	if summary.ChoiceTrace == nil || summary.ChoiceTrace.Profile != choice.Profile || summary.ChoiceTrace.TerminalState != "complete" {
		t.Fatalf("choice summary = %#v", summary.ChoiceTrace)
	}
	opened, err := artifact.OpenArtifact(summary.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := opened.Close(); err != nil {
			t.Error(err)
		}
	})
	if opened.Manifest().ChoiceProfile == nil || opened.Manifest().ChoiceProfile.Trace.Schema != "gomad.choice-trace/v3" || opened.Manifest().ChoiceProfile.Trace.Limit != record.Uint64String(limit) || opened.Manifest().ChoiceProfile.Trace.TapeSHA256 == "" {
		t.Fatalf("artifact choice profile = %#v", opened.Manifest().ChoiceProfile)
	}
}

func TestRunPublishesValidatedChoiceTraceOverflowAsRunnerFailure(t *testing.T) {
	preparer := newFakePreparer(t)
	implementation, err := choice.ImplementationIdentity(preparer.prepared.BuildKey)
	if err != nil {
		t.Fatal(err)
	}
	trace, err := choice.BuildTrace([]choice.Record{validTestChoiceRecord(t, choice.Record{
		Ordinal: 0, Kind: choice.KindRunnable, Flags: choice.FlagDecision, Alternatives: 2, Selected: 1,
	})}, choice.TerminalOverflow)
	if !errors.Is(err, choice.ErrOverflow) {
		t.Fatal(err)
	}
	limit := choiceTraceLimit(t, 1)
	result := processResult(0, "", "")
	result.ChoiceTrace = backend.ChoiceTrace{
		Profile: choice.Profile, ImplementationSHA256: implementation, Limit: limit,
		Trace: trace,
	}
	config, configDependencies := testConfig(t, preparer, terminalErrorExecutor{result: result, err: backend.ErrChoiceTraceOverflow}, "1", PolicyAll, 1)
	config.ChoiceTraceLimit = limit
	summary, err := exploreWith(context.Background(), config, configDependencies)
	var hostErr *HostError
	if !errors.As(err, &hostErr) || hostErr.Reason != "choice_trace_overflow" {
		t.Fatalf("exploreWith() error = %#v", err)
	}
	if len(summary.Artifacts) != 1 || summary.ChoiceTrace == nil || summary.ChoiceTrace.TerminalState != "overflow" {
		t.Fatalf("exploreWith() summary = %#v", summary)
	}
	opened, err := artifact.OpenArtifact(summary.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
	if opened.Manifest().ArtifactKind != record.ArtifactRunnerFailure || opened.Manifest().Outcome.Reason != "choice_trace_overflow" || opened.Manifest().ChoiceProfile == nil || opened.Manifest().ChoiceProfile.Trace.TerminalState != "overflow" {
		t.Fatalf("overflow manifest = %#v", opened.Manifest())
	}

	resumedExecutor := &fakeExecutor{result: func(uint64) backend.ExecutionResult {
		result := processResult(0, "", "")
		result.ChoiceTrace = completeChoiceTrace(t, preparer.prepared.BuildKey, limit, nil)
		return result
	}}
	resumed, err := exploreWith(context.Background(), CampaignSpec{
		ResumeCampaign: summary.CampaignPath, RunnerBuild: config.RunnerBuild, Backend: scriptProvider(),
	}, executionDependencies{executor: resumedExecutor},
	)
	if err != nil {
		t.Fatal(err)
	}
	if seeds := executorSeeds(resumedExecutor); !slices.Equal(seeds, []uint64{1}) || resumed.Succeeded != 1 {
		t.Fatalf("resumed seeds = %v, summary = %#v", seeds, resumed)
	}
}

func TestRunRejectsReservedDuplicateAndInvalidEnvironment(t *testing.T) {
	for name, environment := range map[string][]string{
		"reserved":        {"GOMAXPROCS=2"},
		"choice profile":  {"GOMAD_CHOICE_PROFILE=injected"},
		"choice trace fd": {"GOMAD_CHOICE_TRACE_FD=9"},
		"duplicate":       {"A=1", "A=2"},
		"invalid":         {"NOT-VALID=1"},
		"nul":             {"A=value\x00tail"},
	} {
		t.Run(name, func(t *testing.T) {
			config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
			config.Environment = environment
			if _, err := exploreWith(context.Background(), config, configDependencies); err == nil {
				t.Fatal("exploreWith() succeeded")
			}
		})
	}
}

// TestRunPreparationFailureLeavesClassifiedPartial pins the reason a target
// preparation failure, cancellation or overall timeout reports, and the
// explicit preparation partial each one leaves.
func TestRunPreparationFailureLeavesClassifiedPartial(t *testing.T) {
	for _, test := range []struct {
		name string
		// start returns the preparer and the context the Campaign runs with.
		start       func(t *testing.T) (Preparer, context.Context)
		wantReason  string
		wantCause   error
		wantPartial []string
	}{
		{
			name: "build failure",
			start: func(*testing.T) (Preparer, context.Context) {
				return errorPreparer{err: errors.New("build failed")}, context.Background()
			},
			wantReason: "target_preparation", wantPartial: []string{`"state":"recoverable-failure"`, `"reason":"target_preparation"`},
		},
		{
			name: "cancelled",
			start: func(t *testing.T) (Preparer, context.Context) {
				started := make(chan struct{})
				ctx, cancel := context.WithCancel(context.Background())
				t.Cleanup(cancel)
				go func() {
					<-started
					cancel()
				}()
				return waitingPreparer{started: started}, ctx
			},
			wantReason: "cancelled", wantCause: context.Canceled, wantPartial: []string{`"reason":"cancelled"`},
		},
		{
			name: "overall timeout",
			start: func(*testing.T) (Preparer, context.Context) {
				started := make(chan struct{})
				deadline := &controlledDeadlineContext{Context: context.Background(), done: make(chan struct{})}
				go func() {
					<-started
					close(deadline.done)
				}()
				return waitingPreparer{started: started}, deadline
			},
			wantReason: "overall_timeout", wantCause: context.DeadlineExceeded, wantPartial: []string{`"reason":"overall_timeout"`},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			preparer, ctx := test.start(t)
			config, configDependencies := testConfig(t, preparer, &fakeExecutor{}, "1", PolicyAll, 1)
			config.OverallTimeout = time.Hour
			config.TerminateGrace = 10 * time.Millisecond
			summary, err := exploreWith(ctx, config, configDependencies)
			var hostError *HostError
			if !errors.As(err, &hostError) || hostError.Reason != test.wantReason || test.wantCause != nil && !errors.Is(err, test.wantCause) {
				t.Fatalf("exploreWith() error = %#v", err)
			}
			committed, readErr := campaign.ReadCommittedState(summary.CampaignPath)
			if readErr != nil {
				t.Fatal(readErr)
			}
			partial, encodeErr := json.Marshal(committed.Lifecycle)
			if encodeErr != nil {
				t.Fatal(encodeErr)
			}
			for _, want := range test.wantPartial {
				if !strings.Contains(string(partial), want) {
					t.Fatalf("preparation partial = %s, missing %s", partial, want)
				}
			}
		})
	}
}

func TestManifestForRunBindsIOProfileIdentity(t *testing.T) {
	preparer := newFakePreparer(t)
	config, _ := testConfig(t, preparer, &fakeExecutor{}, "1", PolicyFirst, 1)
	request := campaignRequestFromSpec(config)
	var err error
	request.profile, err = request.Backend.Describe(t.Context(), backend.Configuration{Target: request.Target})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := manifestForRun(request, preparer.prepared, nil, runCompletion{
		job: runJob{seed: 1}, startedAt: time.Unix(1, 0), finishedAt: time.Unix(2, 0), result: processResult(1, "", ""),
	}, backend.Classification{Domain: "target", Reason: "nonzero_exit", Termination: "exit", ArtifactKind: record.ArtifactTargetFailure, ReplayMode: record.ReplayExact}, "run", record.World{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.IOProfile.Name != "gomad-deterministic/v1" || manifest.IOProfile.Inventory == "" || manifest.IOProfile.InventorySHA256 == "" || manifest.IOProfile.ImplementationSHA256 == "" {
		t.Fatalf("manifest I/O profile = %#v", manifest.IOProfile)
	}
	if !reflect.DeepEqual(manifest.IOProfile, request.profile.IOProfile) {
		t.Fatalf("manifest I/O profile = %#v, want described profile %#v", manifest.IOProfile, request.profile.IOProfile)
	}
}

type fakePreparer struct {
	prepared target.Prepared
	calls    int
}

type errorPreparer struct {
	err error
}

func (preparer errorPreparer) Prepare(context.Context, target.Spec) (target.Prepared, error) {
	return target.Prepared{}, preparer.err
}

type waitingPreparer struct {
	started chan<- struct{}
}

func (preparer waitingPreparer) Prepare(ctx context.Context, _ target.Spec) (target.Prepared, error) {
	close(preparer.started)
	<-ctx.Done()
	return target.Prepared{}, ctx.Err()
}

type controlledDeadlineContext struct {
	context.Context
	done chan struct{}
}

func (ctx *controlledDeadlineContext) Done() <-chan struct{} {
	return ctx.done
}

func (ctx *controlledDeadlineContext) Err() error {
	select {
	case <-ctx.done:
		return context.DeadlineExceeded
	default:
		return nil
	}
}

func newFakePreparer(t *testing.T) *fakePreparer {
	t.Helper()
	path := filepath.Join(t.TempDir(), "target")
	data := []byte("fake prepared target")
	if err := os.WriteFile(path, data, 0o500); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o500); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	contract := testfixture.Default().TargetContract()
	return &fakePreparer{prepared: target.Prepared{
		Path: path, Kind: target.KindGoRun, Source: ".", SHA256: fmt.Sprintf("sha256:%x", digest), Size: uint64(len(data)),
		Argv: []string{"gomad-target"}, BuildTags: []string{}, Compatibility: []record.CompatibilityPack{}, BuildInfo: record.BuildInfo{GoVersion: contract.GoVersion, Path: "example.com/target"},
		GoVersion: contract.GoVersion, BuildKey: "cbeccfefbc62a2ca026d9dded0316ecedfce33bd46b5c71b6645e86b67a0713e",
		TargetGOOS: contract.GOOS, TargetGOARCH: contract.GOARCH,
	}}
}

func profileFakePreparer(t *testing.T, argument string) *fakePreparer {
	t.Helper()
	preparer := newFakePreparer(t)
	preparer.prepared.Kind = target.KindGoTest
	preparer.prepared.Source = "./pkg"
	preparer.prepared.Argv = []string{"gomad-target", argument}
	preparer.prepared.BuildTags = []string{"gomad_fixture"}
	preparer.prepared.BuildInfo.Path = "example.test/project/pkg.test"
	return preparer
}

func (preparer *fakePreparer) Prepare(_ context.Context, spec target.Spec) (target.Prepared, error) {
	preparer.calls++
	if err := os.MkdirAll(spec.PreparationRoot, 0o700); err != nil {
		return target.Prepared{}, err
	}
	data, err := os.ReadFile(preparer.prepared.Path)
	if err != nil {
		return target.Prepared{}, err
	}
	path := filepath.Join(spec.PreparationRoot, "target")
	if err := os.WriteFile(path, data, 0o500); err != nil {
		return target.Prepared{}, err
	}
	if err := os.Chmod(path, 0o500); err != nil {
		return target.Prepared{}, err
	}
	prepared := preparer.prepared
	prepared.Path = path
	return prepared, nil
}

type fakeExecutor struct {
	mu            sync.Mutex
	active        int
	maximumActive int
	requests      []backend.ExecutionRequest
	result        func(uint64) backend.ExecutionResult
}

type explorationExecutor struct {
	t        *testing.T
	buildKey string
	limit    uint64
	exitCode int
	// alternatives is the width of the one decision every execution records;
	// zero means two.
	alternatives uint32
	// noOpSelect adds a two-case select with one ready case after that
	// decision, whose poll order the explorer must leave unexpanded.
	noOpSelect bool
	mu         sync.Mutex
	requests   []backend.ExecutionRequest
}

type simulationExplorationExecutor struct {
	t        *testing.T
	buildKey string
	limit    uint64
	fail     bool
	// scenarios is the width of the one scenario decision every execution
	// records; zero means two.
	scenarios int
	mu        sync.Mutex
	requests  []backend.ExecutionRequest
}

type explorationInterruptExecutor struct {
	exploration *explorationExecutor
}

type explorationDivergenceExecutor struct {
	exploration      *explorationExecutor
	beforeDivergence chan struct{}
	afterDivergence  chan struct{}
	ranks            [3]uint32
}

func (executor *explorationDivergenceExecutor) Run(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	result, err := executor.exploration.Run(ctx, request)
	if err != nil {
		return result, err
	}
	if request.Choice.Mode != choice.ModePrefix {
		index := 0
		for rank := uint32(0); rank < 4; rank++ {
			if rank != result.ChoiceTrace.Trace.Records[0].Selected {
				executor.ranks[index] = rank
				index++
			}
		}
		return result, nil
	}
	expected := request.Choice.ReplayPlan.Decisions[0]
	switch expected.Selected {
	case executor.ranks[0]:
		select {
		case <-executor.beforeDivergence:
		case <-ctx.Done():
			return backend.ExecutionResult{}, ctx.Err()
		}
		defer close(executor.afterDivergence)
		observed := result.ChoiceTrace.Trace.Records[0]
		observedDecision := choice.Decision{Ordinal: observed.Ordinal, Kind: observed.Kind, SiteOffset: observed.SiteOffset, Alternatives: observed.Alternatives, Selected: observed.Selected, Data: observed.Data, SelectedIdentity: observed.SelectedIdentity, AlternativeSetDigest: observed.AlternativeSetDigest}
		observedDecision.AlternativeSetDigest = sha256.Sum256([]byte("changed alternatives"))
		return processResult(125, "", ""), &backend.ChoiceReplayDivergenceError{Divergence: choice.Divergence{
			Ordinal: 0, Reason: choice.DivergenceAlternativeSet, Expected: &expected, Observed: &observedDecision, TapeRecords: 1,
		}}
	case executor.ranks[1]:
		defer close(executor.beforeDivergence)
	case executor.ranks[2]:
		select {
		case <-executor.afterDivergence:
		case <-ctx.Done():
			return backend.ExecutionResult{}, ctx.Err()
		}
		if err := ctx.Err(); err != nil {
			return backend.ExecutionResult{}, err
		}
	}
	return result, nil
}

type simulationExplorationInterruptExecutor struct {
	exploration *simulationExplorationExecutor
}

type explorationRootInterruptExecutor struct{}

func (explorationRootInterruptExecutor) Run(context.Context, backend.ExecutionRequest) (backend.ExecutionResult, error) {
	return backend.ExecutionResult{}, errors.New("simulated exploration interruption")
}

func (executor explorationInterruptExecutor) Run(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	if request.Choice != nil && request.Choice.Mode == choice.ModePrefix {
		return backend.ExecutionResult{}, errors.New("simulated exploration interruption")
	}
	return executor.exploration.Run(ctx, request)
}

func (executor simulationExplorationInterruptExecutor) Run(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	var plan struct {
		Overrides []json.RawMessage `json:"overrides"`
	}
	if request.Simulation == nil || json.Unmarshal(request.Simulation.ExplorationPlan, &plan) != nil {
		return backend.ExecutionResult{}, errors.New("simulation exploration simulation plan is unavailable")
	}
	if len(plan.Overrides) != 0 {
		return backend.ExecutionResult{}, errors.New("simulated simulation exploration interruption")
	}
	return executor.exploration.Run(ctx, request)
}

func (executor *explorationExecutor) Run(_ context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	executor.mu.Lock()
	executor.requests = append(executor.requests, request)
	executor.mu.Unlock()
	result := processResult(executor.exitCode, "", "")
	records := []choice.Record{validTestChoiceRecord(executor.t, choice.Record{
		Ordinal: 0, Kind: choice.KindRunnable, Flags: choice.FlagDecision, Alternatives: max(executor.alternatives, 2),
	})}
	if executor.noOpSelect {
		readiness, err := choice.NewReadiness(1, 0)
		if err != nil {
			return backend.ExecutionResult{}, err
		}
		records = append(records,
			validTestChoiceRecord(executor.t, choice.Record{Ordinal: 1, Kind: choice.KindSelectPoll, Flags: choice.FlagDecision, SiteOffset: 0x30, Alternatives: 2, Selected: 1, Data: 1}),
			choice.Record{Ordinal: 2, Kind: choice.KindSelectResult, Flags: choice.FlagObservation, SiteOffset: 0x30, Alternatives: 2, Data: 2, Readiness: readiness, Origin: 1},
		)
	}
	if request.Choice != nil && request.Choice.Mode == choice.ModePrefix {
		// The decisions precede the only observation, so a forced decision's
		// ordinal is its record's.
		for ordinal, forced := range request.Choice.ReplayPlan.Decisions {
			records[ordinal] = forceTestChoiceRank(records[ordinal], forced.Selected)
		}
	}
	result.ChoiceTrace = completeChoiceTrace(executor.t, executor.buildKey, executor.limit, records)
	return result, nil
}

// forceTestChoiceRank selects the alternative at rank of a decision record
// built by validTestChoiceRecord, whose alternative set is ordered by identity.
func forceTestChoiceRank(choiceRecord choice.Record, rank uint32) choice.Record {
	identities := make([][sha256.Size]byte, choiceRecord.Alternatives)
	for index := range identities {
		identities[index] = sha256.Sum256([]byte(fmt.Sprintf("choice/%d/alternative/%d", choiceRecord.Ordinal, index)))
	}
	slices.SortFunc(identities, func(left, right [sha256.Size]byte) int { return bytes.Compare(left[:], right[:]) })
	choiceRecord.Selected = rank
	choiceRecord.SelectedIdentity = identities[rank]
	return choiceRecord
}

func (executor *simulationExplorationExecutor) Run(_ context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	executor.mu.Lock()
	executor.requests = append(executor.requests, request)
	executor.mu.Unlock()
	if request.Simulation == nil || len(request.Simulation.ExplorationPlan) == 0 {
		return backend.ExecutionResult{}, errors.New("simulation exploration simulation plan is unavailable")
	}
	var plan struct {
		BaseSeed  uint64 `json:"base_seed"`
		Overrides []struct {
			Dimension exploration.Dimension `json:"dimension"`
			Selected  uint32                `json:"selected"`
		} `json:"overrides"`
	}
	if err := json.Unmarshal(request.Simulation.ExplorationPlan, &plan); err != nil {
		return backend.ExecutionResult{}, err
	}
	selected := uint32(0)
	for _, override := range plan.Overrides {
		if override.Dimension == exploration.DimensionScenario {
			selected = override.Selected
		}
	}
	scenarios := []string{"alpha", "beta", "gamma"}[:max(executor.scenarios, 2)]
	alternatives := make([]record.SHA256, len(scenarios))
	for index, scenario := range scenarios {
		alternatives[index] = record.HashBytes([]byte(scenario))
	}
	decision, err := exploration.CanonicalDecision(
		exploration.DimensionScenario, 0, record.HashBytes([]byte("route")), alternatives, selected,
	)
	if err != nil {
		return backend.ExecutionResult{}, err
	}
	outcome := "completed"
	if executor.fail {
		outcome = "oracle_failed"
	}
	record, err := json.Marshal(struct {
		Schema               string                 `json:"schema"`
		Seed                 uint64                 `json:"seed"`
		SpecSHA256           record.SHA256          `json:"spec_sha256"`
		Outcome              string                 `json:"outcome"`
		FailureIdentity      record.SHA256          `json:"failure_identity,omitempty"`
		ExplorationPlan      json.RawMessage        `json:"exploration_plan"`
		ExplorationDecisions []exploration.Decision `json:"exploration_decisions"`
		ScenarioTape         []string               `json:"scenario_tape"`
		Identity             record.SHA256          `json:"identity"`
	}{
		Schema: "gomad.cluster-record/v8", Seed: plan.BaseSeed, SpecSHA256: record.HashBytes([]byte("spec")), Outcome: outcome,
		FailureIdentity: record.HashBytes([]byte("normalized oracle failure")),
		ExplorationPlan: request.Simulation.ExplorationPlan, ExplorationDecisions: []exploration.Decision{decision},
		ScenarioTape: []string{scenarios[selected]}, Identity: record.HashBytes([]byte(fmt.Sprintf("record-%d", selected))),
	})
	if err != nil {
		return backend.ExecutionResult{}, err
	}
	exitCode := 0
	if executor.fail {
		exitCode = 2
	}
	result := processResult(exitCode, "", "")
	result.ChoiceTrace = completeChoiceTrace(executor.t, executor.buildKey, executor.limit, nil)
	result.SimulationRecords = [][]byte{record}
	return result, nil
}

type terminalErrorExecutor struct {
	result backend.ExecutionResult
	err    error
}

func (executor terminalErrorExecutor) Run(context.Context, backend.ExecutionRequest) (backend.ExecutionResult, error) {
	result := executor.result
	result.Captured = true
	return result, executor.err
}

type outOfOrderExecutor struct {
	t        *testing.T
	later    chan struct{}
	mu       sync.Mutex
	finished int
}

type matchingReplayer struct {
	calls int
}

func (replayer *matchingReplayer) Replay(_ context.Context, config ReplaySpec) (_ ReplayResult, retErr error) {
	replayer.calls++
	opened, err := artifact.OpenArtifact(config.ArtifactPath)
	if err != nil {
		return ReplayResult{}, err
	}
	defer func(opened *artifact.Opened) {
		if err := opened.Close(); err != nil {
			if retErr == nil {
				retErr = err
			} else {
				retErr = errors.Join(retErr, err)
			}
		}
	}(opened)
	return ReplayResult{Artifact: opened.Snapshot(), Verified: true, Match: true}, nil
}

func newOutOfOrderExecutor(t *testing.T) *outOfOrderExecutor {
	return &outOfOrderExecutor{t: t, later: make(chan struct{})}
}

func (executor *outOfOrderExecutor) Run(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	seed := seedFromEnvironment(request.Env)
	if seed == 1 {
		select {
		case <-executor.later:
		case <-ctx.Done():
			return backend.ExecutionResult{}, ctx.Err()
		}
	} else {
		executor.mu.Lock()
		executor.finished++
		if executor.finished == 2 {
			close(executor.later)
		}
		executor.mu.Unlock()
	}
	result := processResult(0, "", "")
	probe := "stdlib.os.openfile"
	if seed == 3 {
		probe = "stdlib.net.dialtcp"
	}
	result.IOTranscript = semanticTranscript(executor.t, probe)
	return result, nil
}

func (executor *fakeExecutor) Run(_ context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	executor.mu.Lock()
	executor.active++
	if executor.active > executor.maximumActive {
		executor.maximumActive = executor.active
	}
	executor.requests = append(executor.requests, request)
	executor.mu.Unlock()
	seed := seedFromEnvironment(request.Env)
	result := processResult(0, "", "")
	if executor.result != nil {
		result = executor.result(seed)
	}
	executor.mu.Lock()
	executor.active--
	executor.mu.Unlock()
	return result, nil
}

func (executor *fakeExecutor) environments() [][]string {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	result := make([][]string, len(executor.requests))
	for index, request := range executor.requests {
		result[index] = append([]string(nil), request.Env...)
	}
	return result
}

func (executor *fakeExecutor) directories() []string {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	result := make([]string, len(executor.requests))
	for index, request := range executor.requests {
		result[index] = request.Dir
	}
	return result
}

type firstFailureExecutor struct {
	started chan struct{}
	once    sync.Once
	want    int
	mu      sync.Mutex
	count   int
}

type blockingExecutor struct{}

type resumeInterruptExecutor struct {
	mu    sync.Mutex
	calls []uint64
}

type choiceResumeInterruptExecutor struct {
	trace backend.ChoiceTrace
}

type progressGatedExecutor struct {
	started chan struct{}
	release chan struct{}
}

type mutatingExecutor struct{}

func (mutatingExecutor) Run(_ context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	if err := os.Chmod(request.Command, 0o700); err != nil {
		return backend.ExecutionResult{}, err
	}
	file, err := os.OpenFile(request.Command, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return backend.ExecutionResult{}, err
	}
	if _, err := file.Write([]byte("mutation")); err != nil {
		if closeErr := file.Close(); closeErr != nil {
			return backend.ExecutionResult{}, errors.Join(err, closeErr)
		}
		return backend.ExecutionResult{}, err
	}
	if err := file.Close(); err != nil {
		return backend.ExecutionResult{}, err
	}
	return processResult(1, "failure", ""), nil
}

func (blockingExecutor) Run(ctx context.Context, _ backend.ExecutionRequest) (backend.ExecutionResult, error) {
	<-ctx.Done()
	result := processResult(0, "", "")
	result.Cancelled = true
	result.Termination = backend.TerminationSignal
	result.Signal = "killed"
	return result, nil
}

func (executor *resumeInterruptExecutor) Run(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	seed := seedFromEnvironment(request.Env)
	executor.mu.Lock()
	executor.calls = append(executor.calls, seed)
	executor.mu.Unlock()
	if seed == 7 {
		result := processResult(0, "", "")
		result.IOTranscript = completeEmptyTranscript()
		return result, nil
	}
	<-ctx.Done()
	result := processResult(0, "", "")
	result.Cancelled = true
	result.Termination = backend.TerminationSignal
	result.Signal = "killed"
	return result, nil
}

func (executor *choiceResumeInterruptExecutor) Run(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	if seedFromEnvironment(request.Env) == 7 {
		result := processResult(0, "", "")
		result.IOTranscript = completeEmptyTranscript()
		result.ChoiceTrace = executor.trace
		return result, nil
	}
	<-ctx.Done()
	result := processResult(0, "", "")
	result.Cancelled = true
	result.Termination = backend.TerminationSignal
	result.Signal = "killed"
	return result, nil
}

func (executor *resumeInterruptExecutor) seeds() []uint64 {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	return append([]uint64(nil), executor.calls...)
}

func executorSeeds(executor *fakeExecutor) []uint64 {
	environments := executor.environments()
	seeds := make([]uint64, len(environments))
	for index, environment := range environments {
		seeds[index] = seedFromEnvironment(environment)
	}
	return seeds
}

func (executor *progressGatedExecutor) Run(context.Context, backend.ExecutionRequest) (backend.ExecutionResult, error) {
	close(executor.started)
	<-executor.release
	return processResult(0, "", ""), nil
}

func newFirstFailureExecutor(want int) *firstFailureExecutor {
	return &firstFailureExecutor{started: make(chan struct{}), want: want}
}

func (executor *firstFailureExecutor) Run(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	seed := seedFromEnvironment(request.Env)
	executor.mu.Lock()
	executor.count++
	if executor.count == executor.want {
		executor.once.Do(func() { close(executor.started) })
	}
	executor.mu.Unlock()
	<-executor.started
	if seed == 1 {
		return processResult(1, "first failure", ""), nil
	}
	<-ctx.Done()
	result := processResult(0, "", "")
	result.Cancelled = true
	result.Termination = backend.TerminationSignal
	result.Signal = "killed"
	return result, nil
}

type scriptedProvider struct{ *testbackend.Provider }

type noReplayProvider struct{ backend.Provider }

func (provider noReplayProvider) Capabilities() backend.Capabilities {
	caps := provider.Provider.Capabilities()
	caps.ReplayModes = nil
	return caps
}

func scriptProvider() backend.Provider {
	return scriptedProvider{testbackend.New(testbackend.Config{})}
}

func (provider scriptedProvider) ValidatePrepared(_ target.Spec, prepared target.Prepared, _ []string) error {
	return prepared.Verify()
}

type targetPreparer struct{}

func (targetPreparer) Prepare(ctx context.Context, spec target.Spec) (target.Prepared, error) {
	return testbackend.Prepare(ctx, spec)
}

func testConfig(t *testing.T, preparer Preparer, executor executionRunner, seeds string, policy FailurePolicy, parallel int) (CampaignSpec, executionDependencies) {
	t.Helper()
	return CampaignSpec{
		Seeds: seeds, Parallel: parallel, ExecutionTimeout: time.Second, OverallTimeout: 10 * time.Second, TerminateGrace: 100 * time.Millisecond,
		OnFailure: policy, FailureBudget: 1, OutputLimit: 64, WorldTransitionLimit: 64, Artifacts: t.TempDir(),
		Environment: []string{"MODE=test"}, Target: target.Spec{Kind: target.KindGoRun, Source: "."}, Backend: scriptProvider(),
		RunnerBuild: "sha256:0000000000000000000000000000000000000000000000000000000000000000", Preparer: preparer,
	}, executionDependencies{executor: executor}
}

func cancelOnProgress(t *testing.T, config *CampaignSpec, predicate func(CampaignEvent) bool) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	config.ProgressInterval = time.Millisecond
	config.Progress = func(progress CampaignEvent) error {
		if predicate(progress) {
			cancel()
		}
		return nil
	}
	return ctx
}

func processResult(exitCode int, stdout, stderr string) backend.ExecutionResult {
	return backend.ExecutionResult{
		Captured: true, Termination: backend.TerminationExit, ExitCode: exitCode, GroupGone: true,
		Stdout: output(stdout), Stderr: output(stderr),
	}
}

func output(value string) backend.Output {
	bytes := []byte(value)
	digest := sha256.Sum256(bytes)
	return backend.Output{Bytes: bytes, FullSHA256: digest, RetainedSHA256: digest, TotalBytes: uint64(len(bytes)), RetainedBytes: uint64(len(bytes))}
}

func semanticTranscript(t *testing.T, probe string) backend.Transcript {
	t.Helper()
	digest := sha256.Sum256([]byte("gomad-boundary-probe/v1\x00" + probe))
	var argument [8]byte
	binary.BigEndian.PutUint64(argument[:], binary.BigEndian.Uint64(digest[:8])&(1<<63-1))
	payload, err := testfixture.EncodeTranscript([]testfixture.Operation{{
		Name: "boundary.probe", ArgumentHash: testfixture.HashArgument(argument[:]),
	}})
	if err != nil {
		t.Fatal(err)
	}
	return backend.Transcript{Bytes: payload, SHA256: sha256.Sum256(payload), Records: 1, Complete: true}
}

func completeEmptyTranscript() backend.Transcript {
	return backend.Transcript{Complete: true, SHA256: sha256.Sum256(nil)}
}

func completeChoiceTrace(t *testing.T, buildKey string, limit uint64, records []choice.Record) backend.ChoiceTrace {
	t.Helper()
	for index, choiceRecord := range records {
		records[index] = validTestChoiceRecord(t, choiceRecord)
	}
	trace, err := choice.BuildTrace(records, choice.TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	implementation, err := choice.ImplementationIdentity(buildKey)
	if err != nil {
		t.Fatal(err)
	}
	return backend.ChoiceTrace{Profile: choice.Profile, ImplementationSHA256: implementation, Limit: limit, Trace: trace}
}

func choiceTraceLimit(t *testing.T, records uint64) uint64 {
	t.Helper()
	limit, err := choice.TraceBytes(records)
	if err != nil {
		t.Fatal(err)
	}
	return limit
}

func validTestChoiceRecord(t *testing.T, choiceRecord choice.Record) choice.Record {
	t.Helper()
	if choiceRecord.Kind == choice.KindSelectResult && !choiceRecord.Readiness.Known() {
		var err error
		if choiceRecord.Readiness, err = choice.NewReadiness(0, 0); err != nil {
			t.Fatal(err)
		}
	}
	if choiceRecord.Flags&choice.FlagDecision == 0 || choiceRecord.SelectedIdentity != ([sha256.Size]byte{}) {
		return choiceRecord
	}
	alternatives := make([][sha256.Size]byte, choiceRecord.Alternatives)
	for index := range alternatives {
		alternatives[index] = sha256.Sum256([]byte(fmt.Sprintf("choice/%d/alternative/%d", choiceRecord.Ordinal, index)))
	}
	decision, err := choice.CanonicalDecision(
		choiceRecord.Ordinal, choiceRecord.Kind, choiceRecord.SiteOffset, choiceRecord.Flags&choice.FlagSiteMissing != 0,
		alternatives, alternatives[choiceRecord.Selected], choiceRecord.Data,
	)
	if err != nil {
		t.Fatal(err)
	}
	return decision.Record()
}

func seedFromEnvironment(environment []string) uint64 {
	for _, entry := range environment {
		if value, found := strings.CutPrefix(entry, "GOMADSEED="); found {
			seed, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				panic(err)
			}
			return seed
		}
	}
	panic("missing GOMADSEED")
}

func allUnique(values []string) bool {
	sort.Strings(values)
	for index := 1; index < len(values); index++ {
		if values[index] == values[index-1] {
			return false
		}
	}
	return true
}

// retainedTargets returns every file under root that holds the content, and
// how many of them are artifact targets.
func retainedTargets(t *testing.T, root string, content []byte) (paths []string, artifacts int) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.Type().IsRegular() {
			return err
		}
		held, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(held, content) {
			return err
		}
		paths = append(paths, path)
		if entry.Name() == "target" && strings.HasPrefix(filepath.Base(filepath.Dir(path)), "sha256-") {
			artifacts++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return paths, artifacts
}

// distinctFiles counts the files the paths name, counting hard links to one
// file once.
func distinctFiles(t *testing.T, paths []string) int {
	t.Helper()
	var distinct []os.FileInfo
next:
	for _, path := range paths {
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, seen := range distinct {
			if os.SameFile(seen, info) {
				continue next
			}
		}
		distinct = append(distinct, info)
	}
	return len(distinct)
}

func TestRunKeepsOneTargetCopyAcrossArtifactsRoundsAndCampaigns(t *testing.T) {
	artifacts := t.TempDir()
	published := 0
	for _, strategy := range completionStrategies {
		for _, fail := range []bool{false, true} {
			config, executor, configDependencies := retentionCampaign(t, strategy, false, fail)
			config.Artifacts = artifacts
			keepSuccesses(&config, KeepSuccessesAll)
			executor.shape = rankProbes(t, probeA, probeA, probeB)
			summary, err := exploreWith(context.Background(), config, configDependencies)
			if err != nil {
				t.Fatal(err)
			}
			if exploration := summary.ChoiceExploration; exploration != nil && exploration.CommittedRounds != 3 {
				t.Fatalf("choice exploration committed %d rounds, want 3", exploration.CommittedRounds)
			}
			if exploration := summary.SimulationExploration; exploration != nil && exploration.CommittedRounds != 3 {
				t.Fatalf("simulation exploration committed %d rounds, want 3", exploration.CommittedRounds)
			}
			published += len(summary.Artifacts) + len(summary.SuccessArtifacts)
		}
	}
	// Three successes per succeeding campaign, three distinct failures per
	// failing seed and choice campaign, and the one failure identity the
	// simulation candidates share.
	paths, held := retainedTargets(t, artifacts, []byte("fake prepared target"))
	if published != 16 || held != published || distinctFiles(t, paths) != 1 {
		t.Fatalf("%d published artifacts, %d of them hold a target, in %d files; want 16 artifacts and one file", published, held, distinctFiles(t, paths))
	}
}

// sharedTargetReplayer records every replayed artifact whose target is not a
// link to the pool of its store's owner: the minimizer's work directory for a
// candidate, the output root for the published artifact.
type sharedTargetReplayer struct {
	ArtifactReplayer
	replayed int
	private  []string
}

func (replayer *sharedTargetReplayer) Replay(ctx context.Context, spec ReplaySpec) (ReplayResult, error) {
	replayer.replayed++
	store := filepath.Dir(spec.ArtifactPath)
	var entries []string
	for _, owner := range []string{store, filepath.Dir(store)} {
		pooled, err := filepath.Glob(filepath.Join(artifact.TargetPool(owner), "sha256-*"))
		if err != nil {
			return ReplayResult{}, err
		}
		entries = append(entries, pooled...)
	}
	target, err := os.Lstat(filepath.Join(spec.ArtifactPath, "target"))
	if err != nil {
		return ReplayResult{}, err
	}
	shared := false
	for _, entry := range entries {
		info, err := os.Lstat(entry)
		if err != nil {
			return ReplayResult{}, err
		}
		shared = shared || os.SameFile(info, target)
	}
	if !shared {
		replayer.private = append(replayer.private, spec.ArtifactPath)
	}
	return replayer.ArtifactReplayer.Replay(ctx, spec)
}

func TestCleanupMatchingReplayerPreservesDetachedResultAndCalls(t *testing.T) {
	published := publishInspectArtifact(t)
	replayer := &matchingReplayer{}
	result, err := replayer.Replay(t.Context(), ReplaySpec{ArtifactPath: published.Path})
	if err != nil {
		t.Fatal(err)
	}
	want := ReplayResult{Artifact: published, Verified: true, Match: true}
	want.Artifact.TargetSharing = ""
	if !reflect.DeepEqual(result, want) || replayer.calls != 1 {
		t.Fatalf("first replay = %#v, calls = %d, want %#v", result, replayer.calls, want)
	}
	result.Artifact.Manifest.Target.Argv[0] = "changed snapshot"
	result, err = replayer.Replay(t.Context(), ReplaySpec{ArtifactPath: published.Path})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result, want) || replayer.calls != 2 {
		t.Fatalf("second replay = %#v, calls = %d, want %#v", result, replayer.calls, want)
	}
	result, err = replayer.Replay(t.Context(), ReplaySpec{ArtifactPath: filepath.Join(t.TempDir(), "missing")})
	if !errors.Is(err, os.ErrNotExist) || !reflect.DeepEqual(result, ReplayResult{}) || replayer.calls != 3 {
		t.Fatalf("missing replay = %#v, %v, calls = %d", result, err, replayer.calls)
	}
}

func TestCleanupMutatingExecutorPreservesPreparedTargetAndErrors(t *testing.T) {
	for _, name := range []string{"file", "missing", "directory"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "target")
			data := []byte("prepared target")
			switch name {
			case "file":
				if err := os.WriteFile(path, data, 0o500); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(path, 0o500); err != nil {
					t.Fatal(err)
				}
			case "missing":
			}
			result, err := (mutatingExecutor{}).Run(context.Background(), backend.ExecutionRequest{Command: path})
			if name != "file" {
				var pathErr *os.PathError
				if !errors.As(err, &pathErr) || pathErr.Path != path || !reflect.DeepEqual(result, backend.ExecutionResult{}) {
					t.Fatalf("mutation error = %#v, %v", result, err)
				}
				wantOp := "open"
				if name == "missing" {
					wantOp = "chmod"
					if !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("missing target error = %v", err)
					}
				}
				if pathErr.Op != wantOp {
					t.Fatalf("mutation operation = %q, want %q", pathErr.Op, wantOp)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(result, processResult(1, "failure", "")) {
				t.Fatalf("mutation result = %#v, %v", result, err)
			}
			actual, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(actual) != "prepared targetmutation" {
				t.Fatalf("mutated bytes = %q", actual)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0o700 {
				t.Fatalf("mutated mode = %v", info.Mode())
			}
		})
	}
}

func TestWaitForProgressStart(t *testing.T) {
	for _, test := range []struct {
		name      string
		started   bool
		completed bool
		deadline  bool
		wantError string
	}{
		{name: "started", started: true},
		{name: "nil completion", completed: true, wantError: "runner completed before target execution started: <nil>"},
		{name: "deadline", deadline: true, wantError: "target execution did not start before startup deadline"},
	} {
		t.Run(test.name, func(t *testing.T) {
			started := make(chan struct{})
			start := sync.OnceFunc(func() { close(started) })
			t.Cleanup(start)
			completed := make(chan error, 1)
			deadline := make(chan time.Time, 1)
			if test.started {
				start()
			}
			if test.completed {
				completed <- nil
			}
			if test.deadline {
				deadline <- time.Time{}
			}
			watchdog := time.AfterFunc(time.Second, start)
			defer watchdog.Stop()
			err := waitForProgressStart(started, completed, deadline)
			if test.wantError == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || err.Error() != test.wantError {
				t.Fatalf("startup error = %v, want %q", err, test.wantError)
			}
		})
	}
}

func TestWaitForProgressStartObservesPreparationFailure(t *testing.T) {
	executor := &progressGatedExecutor{started: make(chan struct{}), release: make(chan struct{})}
	config, dependencies := testConfig(t, errorPreparer{err: errors.New("progress-start preparation sentinel")}, executor, "1", PolicyAll, 1)
	t.Cleanup(sync.OnceFunc(func() { close(executor.release) }))
	start := sync.OnceFunc(func() { close(executor.started) })
	t.Cleanup(start)
	completed := make(chan error, 1)
	go func() {
		_, err := exploreWith(context.Background(), config, dependencies)
		completed <- err
	}()
	watchdog := time.AfterFunc(time.Second, start)
	defer watchdog.Stop()
	deadline := time.NewTimer(config.OverallTimeout + config.TerminateGrace)
	defer deadline.Stop()
	err := waitForProgressStart(executor.started, completed, deadline.C)
	if err == nil || !strings.Contains(err.Error(), "progress-start preparation sentinel") {
		t.Fatalf("startup error = %v, want preparation sentinel", err)
	}
	select {
	case <-executor.started:
		t.Fatal("executor started after preparation failure")
	default:
	}
}

func waitForProgressStart(started <-chan struct{}, completed <-chan error, deadline <-chan time.Time) error {
	select {
	case <-started:
		return nil
	case err := <-completed:
		return fmt.Errorf("runner completed before target execution started: %v", err)
	case <-deadline:
		return errors.New("target execution did not start before startup deadline")
	}
}

func TestIsCapacityErrorFindsTypedCapacityThroughHostFailure(t *testing.T) {
	err := &HostError{Reason: "artifact_publication", Err: &campaign.ArtifactCapacityError{
		Limit: campaign.ArtifactLimitTotalBytes, Required: 2, Maximum: 1, Outcome: campaign.CapacityInfrastructureFailure,
	}}
	if !IsCapacityError(err) {
		t.Fatalf("IsCapacityError(%v) = false", err)
	}
	if IsCapacityError(errors.New("capacity")) {
		t.Fatal("IsCapacityError() classified an untyped message")
	}
}
