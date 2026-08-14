package runner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/artifact"
	"go.temporal.io/server/tools/gomadv3/internal/choicewire"
	"go.temporal.io/server/tools/gomadv3/internal/ioprofile"
	"go.temporal.io/server/tools/gomadv3/internal/iowire"
	executionoutcome "go.temporal.io/server/tools/gomadv3/internal/outcome"
	"go.temporal.io/server/tools/gomadv3/internal/process"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/replay"
	"go.temporal.io/server/tools/gomadv3/internal/target"
	"go.temporal.io/server/tools/gomadv3/world"
)

func TestRunPreparesOnceBoundsParallelismAndGroupsMatchingFailures(t *testing.T) {
	preparer := newFakePreparer(t)
	executor := &fakeExecutor{result: func(seed uint64) process.Result {
		if seed%2 == 0 {
			return processResult(2, "same failure", "")
		}
		return processResult(0, "success", "")
	}}
	summary, err := Run(context.Background(), testConfig(t, preparer, executor, "1-6", PolicyAll, 2))
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
	if _, err := artifact.Open(summary.Artifacts[0]); err != nil {
		t.Fatal(err)
	}
	opened, err := artifact.Open(summary.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	if opened.Manifest.Target.BuildTags == nil {
		t.Fatal("empty target build tags encoded as null")
	}
	if _, err := os.Stat(filepath.Join(summary.BatchPath, "batch.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(summary.BatchPath, ".prepared")); !os.IsNotExist(err) {
		t.Fatalf("prepared target retained after publication: %v", err)
	}
	if got := executor.environments(); len(got) != 6 {
		t.Fatalf("target environments = %v", got)
	} else {
		for _, environment := range got {
			if len(environment) != 4 || environment[1] != "GOMADV3_IO_PROFILE=gomadv3-deterministic/v1" || environment[2] != "MODE=test" || environment[3] != "TZ=UTC" || !strings.HasPrefix(environment[0], "GOMADSEED=") {
				t.Fatalf("target environment = %v", environment)
			}
		}
	}
	if !allUnique(executor.directories()) {
		t.Fatalf("working directories were reused: %v", executor.directories())
	}
}

func TestRunReportsPreparationProgressAndCompletedCounts(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(seed uint64) process.Result {
		if seed == 2 {
			return processResult(2, "failure", "")
		}
		return processResult(0, "success", "")
	}}, "1-3", PolicyAll, 2)
	var progress []Progress
	config.Progress = func(update Progress) error {
		progress = append(progress, update)
		return nil
	}
	summary, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if len(progress) < 4 || progress[0].Phase != ProgressPreparing || progress[0].BatchPath == "" {
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
	config := testConfig(t, newFakePreparer(t), executor, "1", PolicyAll, 1)
	config.ProgressInterval = time.Millisecond
	updates := make(chan Progress, 16)
	config.Progress = func(update Progress) error {
		select {
		case updates <- update:
		default:
		}
		return nil
	}
	completed := make(chan error, 1)
	go func() {
		_, err := Run(context.Background(), config)
		completed <- err
	}()
	<-executor.started
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
	close(executor.release)
	if err := <-completed; err != nil {
		t.Fatal(err)
	}
}

func TestRunReportsPassiveSemanticCoverage(t *testing.T) {
	executor := &fakeExecutor{result: func(uint64) process.Result {
		result := processResult(0, "", "")
		result.IOTranscript = semanticTranscript(t, "stdlib.os.openfile")
		return result
	}}
	config := testConfig(t, newFakePreparer(t), executor, "1", PolicyAll, 1)
	config.Coverage = CoverageSemantic
	summary, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.SemanticCoverage == nil || len(summary.SemanticCoverage.Probes) != 1 || summary.SemanticCoverage.Probes[0] != "stdlib.os.openfile" {
		t.Fatalf("summary coverage = %#v", summary.SemanticCoverage)
	}
}

func TestRunFailsWhenRequiredSemanticProbeIsMissing(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
	config.Coverage = CoverageSemantic
	config.RequiredSemanticProbes = []string{"stdlib.os.openfile"}
	_, err := Run(context.Background(), config)
	var missing *ioprofile.MissingSemanticProbesError
	if !errors.As(err, &missing) || len(missing.Probes) != 1 || missing.Probes[0] != "stdlib.os.openfile" {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestRunRetainsOnlyProbeNovelSuccessesWithinExplicitBounds(t *testing.T) {
	executor := &fakeExecutor{result: func(seed uint64) process.Result {
		result := processResult(0, fmt.Sprintf("seed=%d", seed), "")
		probe := "stdlib.os.openfile"
		if seed == 3 {
			probe = "stdlib.net.dialtcp"
		}
		result.IOTranscript = semanticTranscript(t, probe)
		return result
	}}
	config := testConfig(t, newFakePreparer(t), executor, "1-3", PolicyAll, 1)
	config.Coverage = CoverageSemantic
	config.KeepSuccesses = KeepSuccessesNovel
	config.SuccessArtifactLimit = 2
	config.SuccessBytesLimit = 64 << 20
	summary, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Succeeded != 3 || summary.RetainedSuccesses != 2 || summary.RetainedSuccessBytes == 0 || len(summary.SuccessArtifacts) != 2 {
		t.Fatalf("summary = %#v", summary)
	}
	batch, err := artifact.OpenBatch(summary.BatchPath)
	if err != nil {
		t.Fatal(err)
	}
	if batch.Runs[0].SuccessArtifact == nil || batch.Runs[1].SuccessArtifact != nil || batch.Runs[2].SuccessArtifact == nil || !slices.Equal(batch.Runs[0].NovelSemanticProbes, []string{"stdlib.os.openfile"}) || !slices.Equal(batch.Runs[2].NovelSemanticProbes, []string{"stdlib.net.dialtcp"}) {
		t.Fatalf("batch runs = %#v", batch.Runs)
	}
	for _, path := range summary.SuccessArtifacts {
		opened, err := artifact.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		if opened.Manifest.ArtifactKind != record.ArtifactSuccess || opened.Manifest.Outcome.Domain != "success" || opened.Manifest.ReplayMode != record.ReplayExact {
			t.Fatalf("retained success = %#v", opened.Manifest)
		}
		if err := opened.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRunRetainsOnlyChoiceNovelSuccessesAndRecordsTheirFeatures(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := uint64(choicewire.HeaderBytes + 3*choicewire.RecordBytes)
	executor := &fakeExecutor{result: func(seed uint64) process.Result {
		result := processResult(0, "", "")
		result.IOTranscript = completeEmptyTranscript()
		selected := uint32(0)
		if seed == 3 {
			selected = 1
		}
		result.ChoiceTrace = completeChoiceTrace(t, preparer.prepared.BuildKey, limit, []choicewire.Record{{
			Ordinal: 0, Kind: choicewire.KindRunnable, Flags: choicewire.FlagDecision, SiteOffset: 24, Alternatives: 2, Selected: selected,
		}})
		return result
	}}
	config := testConfig(t, preparer, executor, "1-3", PolicyAll, 1)
	config.Coverage = CoverageChoice
	config.ChoiceTraceLimit = limit
	config.KeepSuccesses = KeepSuccessesNovel
	config.SuccessArtifactLimit = 2
	config.SuccessBytesLimit = 64 << 20
	summary, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.RetainedSuccesses != 2 || len(summary.SuccessArtifacts) != 2 {
		t.Fatalf("summary = %#v", summary)
	}
	batch, err := artifact.OpenBatch(summary.BatchPath)
	if err != nil {
		t.Fatal(err)
	}
	if batch.Runs[0].SuccessArtifact == nil || len(batch.Runs[0].NovelChoiceFeatures) == 0 || batch.Runs[1].SuccessArtifact != nil || len(batch.Runs[1].NovelChoiceFeatures) != 0 || batch.Runs[2].SuccessArtifact == nil || len(batch.Runs[2].NovelChoiceFeatures) == 0 {
		t.Fatalf("choice novelty runs = %#v", batch.Runs)
	}
	for _, run := range batch.Runs {
		if len(run.ChoiceFeatures) == 0 || !slices.IsSorted(run.ChoiceFeatures) {
			t.Fatalf("choice features = %#v", run.ChoiceFeatures)
		}
	}
}

func TestRunMergesParallelCompletionsInSelectionOrdinalOrder(t *testing.T) {
	executor := newOutOfOrderExecutor(t)
	config := testConfig(t, newFakePreparer(t), executor, "1-3", PolicyAll, 3)
	config.Coverage = CoverageSemantic
	config.KeepSuccesses = KeepSuccessesNovel
	config.SuccessArtifactLimit = 2
	config.SuccessBytesLimit = 64 << 20
	summary, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	batch, err := artifact.OpenBatch(summary.BatchPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := []record.Uint64String{batch.Runs[0].Seed, batch.Runs[1].Seed, batch.Runs[2].Seed}; !slices.Equal(got, []record.Uint64String{1, 2, 3}) {
		t.Fatalf("batch merge order = %v", got)
	}
	if batch.Runs[0].SuccessArtifact == nil || batch.Runs[1].SuccessArtifact != nil || batch.Runs[2].SuccessArtifact == nil {
		t.Fatalf("ordered novelty retention = %#v", batch.Runs)
	}
}

func TestRunGuidesFromImmutableCorpusAndKeepsUnguidedSeeds(t *testing.T) {
	corpus := filepath.Join(t.TempDir(), "corpus")
	replayer := &matchingReplayer{}
	firstExecutor := &fakeExecutor{result: func(seed uint64) process.Result {
		result := processResult(0, "", "")
		probe := "stdlib.os.openfile"
		if seed%2 == 1 {
			probe = "stdlib.net.dialtcp"
		}
		result.IOTranscript = semanticTranscript(t, probe)
		return result
	}}
	firstConfig := testConfig(t, newFakePreparer(t), firstExecutor, "0-7", PolicyAll, 3)
	firstConfig.Coverage = CoverageSemantic
	firstConfig.Guide = true
	firstConfig.Corpus = corpus
	firstConfig.Replayer = replayer
	first, err := Run(context.Background(), firstConfig)
	if err != nil {
		t.Fatal(err)
	}
	if first.CorpusAdded != 2 || first.CorpusEntries != 2 || replayer.calls != 2 {
		t.Fatalf("first guided summary = %#v, replay calls = %d", first, replayer.calls)
	}

	secondExecutor := &fakeExecutor{result: firstExecutor.result}
	secondConfig := testConfig(t, newFakePreparer(t), secondExecutor, "100-107", PolicyAll, 3)
	secondConfig.Coverage = CoverageSemantic
	secondConfig.Guide = true
	secondConfig.Corpus = corpus
	secondConfig.Replayer = replayer
	second, err := Run(context.Background(), secondConfig)
	if err != nil {
		t.Fatal(err)
	}
	if second.CorpusAdded != 0 || second.CorpusEntries != 2 || replayer.calls != 2 {
		t.Fatalf("second guided summary = %#v, replay calls = %d", second, replayer.calls)
	}
	batch, err := artifact.OpenBatch(second.BatchPath)
	if err != nil {
		t.Fatal(err)
	}
	if batch.Record.Selection != "0,1,100-105" || batch.Record.SelectionCount != 8 {
		t.Fatalf("guided batch selection = %q (%d)", batch.Record.Selection, batch.Record.SelectionCount)
	}
	seeds := make([]uint64, len(batch.Runs))
	for index, run := range batch.Runs {
		seeds[index] = uint64(run.Seed)
	}
	if !slices.Equal(seeds, []uint64{0, 1, 100, 101, 102, 103, 104, 105}) {
		t.Fatalf("second guided seeds = %v", seeds)
	}
}

func TestRunGuidesFromReplayVerifiedChoiceCoverage(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := uint64(choicewire.HeaderBytes + choicewire.RecordBytes)
	replayer := &matchingReplayer{}
	executor := &fakeExecutor{result: func(seed uint64) process.Result {
		result := processResult(0, "", "")
		result.IOTranscript = completeEmptyTranscript()
		result.ChoiceTrace = completeChoiceTrace(t, preparer.prepared.BuildKey, limit, []choicewire.Record{{
			Ordinal: 0, Kind: choicewire.KindRunnable, Flags: choicewire.FlagDecision, SiteOffset: 24, Alternatives: 2, Selected: uint32(seed % 2),
		}})
		return result
	}}
	config := testConfig(t, preparer, executor, "1-2", PolicyAll, 1)
	config.Coverage = CoverageChoice
	config.ChoiceTraceLimit = limit
	config.Guide = true
	config.Corpus = filepath.Join(t.TempDir(), "corpus")
	config.Replayer = replayer
	summary, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.CorpusAdded != 2 || summary.CorpusEntries != 2 || replayer.calls != 2 {
		t.Fatalf("guided choice summary = %#v, replay calls = %d", summary, replayer.calls)
	}
}

func TestRunGuidanceRequiresCorpusAndSemanticCoverage(t *testing.T) {
	for _, configure := range []func(*Config){
		func(config *Config) { config.Corpus = "" },
		func(config *Config) { config.Coverage = CoverageNone },
	} {
		config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
		config.Guide = true
		config.Corpus = t.TempDir()
		config.Coverage = CoverageSemantic
		configure(&config)
		if _, err := Run(context.Background(), config); err == nil {
			t.Fatal("Run accepted invalid guided configuration")
		}
	}
}

func TestRunGuidanceWithoutReplayCapabilityFailsClosed(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(uint64) process.Result {
		result := processResult(0, "", "")
		result.IOTranscript = semanticTranscript(t, "stdlib.os.openfile")
		return result
	}}, "1", PolicyAll, 1)
	config.Coverage = CoverageSemantic
	config.Guide = true
	config.Corpus = filepath.Join(t.TempDir(), "corpus")
	config.SupervisorCommand = nil
	_, err := Run(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "guided_corpus" {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestRunFailsClosedWhenSuccessRetentionCountIsExhausted(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(uint64) process.Result {
		result := processResult(0, "", "")
		result.IOTranscript = completeEmptyTranscript()
		return result
	}}, "1-2", PolicyAll, 1)
	config.KeepSuccesses = KeepSuccessesAll
	config.SuccessArtifactLimit = 1
	config.SuccessBytesLimit = 64 << 20
	summary, err := Run(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "success_retention_capacity" || summary.RetainedSuccesses != 1 {
		t.Fatalf("summary = %#v, error = %v", summary, err)
	}
}

func TestRunRequiresExplicitSuccessRetentionBounds(t *testing.T) {
	for _, configure := range []func(*Config){
		func(config *Config) { config.SuccessArtifactLimit = 0 },
		func(config *Config) { config.SuccessBytesLimit = 0 },
		func(config *Config) { config.KeepSuccesses = KeepSuccessesNovel; config.Coverage = CoverageNone },
	} {
		config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
		config.KeepSuccesses = KeepSuccessesAll
		config.SuccessArtifactLimit = 1
		config.SuccessBytesLimit = 1 << 20
		configure(&config)
		if _, err := Run(context.Background(), config); err == nil {
			t.Fatal("Run() accepted invalid success retention configuration")
		}
	}
}

func TestRunRequiresBoundedChoiceTraceCapacity(t *testing.T) {
	for _, limit := range []uint64{1, (64 << 20) + 1} {
		config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
		config.ChoiceTraceLimit = limit
		if _, err := Run(context.Background(), config); err == nil || !strings.Contains(err.Error(), "choice trace") {
			t.Fatalf("Run() with choice limit %d error = %v", limit, err)
		}
	}
}

func TestRunRejectsSuccessfulRetentionWithoutReplayTranscript(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
	config.KeepSuccesses = KeepSuccessesAll
	config.SuccessArtifactLimit = 1
	config.SuccessBytesLimit = 1 << 20
	_, err := Run(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "success_artifact_publication" || !strings.Contains(err.Error(), "complete I/O transcript") {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestRunCollectsBoundedQualificationEvidenceForOneSeed(t *testing.T) {
	executor := &fakeExecutor{result: func(uint64) process.Result {
		result := processResult(0, "stdout", "stderr")
		result.IOTranscript = semanticTranscript(t, "stdlib.os.openfile")
		return result
	}}
	config := testConfig(t, newFakePreparer(t), executor, "7", PolicyAll, 1)
	config.Coverage = CoverageSemantic
	config.CollectRunEvidence = true
	summary, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.RunEvidence == nil || summary.RunEvidence.Seed != 7 || summary.RunEvidence.Target.SHA256 == "" || summary.RunEvidence.Stdout.FullSHA256 != record.HashBytes([]byte("stdout")) || summary.RunEvidence.Stderr.FullSHA256 != record.HashBytes([]byte("stderr")) || summary.RunEvidence.IOTranscriptRecords != 1 || summary.RunEvidence.SemanticCoverage.Digest == "" {
		t.Fatalf("run evidence = %#v", summary.RunEvidence)
	}
}

func TestRunEvidenceRequiresOneSeedAndSemanticCoverage(t *testing.T) {
	for _, configure := range []func(*Config){
		func(config *Config) { config.Seeds = "1-2" },
		func(config *Config) { config.Coverage = CoverageNone },
	} {
		config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
		config.Coverage = CoverageSemantic
		config.CollectRunEvidence = true
		configure(&config)
		if _, err := Run(context.Background(), config); err == nil {
			t.Fatal("Run() accepted invalid evidence configuration")
		}
	}
}

func TestRunEvidenceIgnoresAggregateCoordinatorDeadlineAdjustment(t *testing.T) {
	var evidence []RunEvidence
	for _, overallTimeout := range []time.Duration{9 * time.Second, 10 * time.Second} {
		config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "7", PolicyAll, 1)
		config.Coverage = CoverageSemantic
		config.CollectRunEvidence = true
		config.OverallTimeout = overallTimeout
		summary, err := Run(context.Background(), config)
		if err != nil {
			t.Fatal(err)
		}
		evidence = append(evidence, *summary.RunEvidence)
	}
	first, err := record.CanonicalJSON(evidence[0])
	if err != nil {
		t.Fatal(err)
	}
	second, err := record.CanonicalJSON(evidence[1])
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
	config := testConfig(t, preparer, executor, "7", PolicyAll, 1)
	config.Artifacts = "artifacts"
	summary, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(summary.BatchPath) {
		t.Fatalf("batch path = %q, want absolute path", summary.BatchPath)
	}
	for _, directory := range executor.directories() {
		if !filepath.IsAbs(directory) {
			t.Fatalf("target working directory = %q, want absolute path", directory)
		}
	}
}

func TestRunFirstFailureCancelsActiveTargetsWithoutPublishingThem(t *testing.T) {
	preparer := newFakePreparer(t)
	executor := newFirstFailureExecutor(3)
	config := testConfig(t, preparer, executor, "1-10", PolicyFirst, 3)
	summary, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Attempted != 3 || summary.Failures != 1 || summary.Cancelled != 2 || summary.DistinctFailures != 1 || summary.StopReason != StopFirstFailure {
		t.Fatalf("summary = %#v", summary)
	}
	if len(summary.Artifacts) != 1 {
		t.Fatalf("artifacts = %v", summary.Artifacts)
	}
	partials, err := os.ReadDir(filepath.Join(summary.BatchPath, ".partial"))
	if err != nil {
		t.Fatal(err)
	}
	if len(partials) != 2 {
		t.Fatalf("cancelled target partials = %v, want 2", partials)
	}
}

func TestRunBudgetCountsDistinctSignatures(t *testing.T) {
	preparer := newFakePreparer(t)
	executor := &fakeExecutor{result: func(seed uint64) process.Result {
		output := "same"
		if seed == 4 {
			output = "different"
		}
		return processResult(1, output, "")
	}}
	config := testConfig(t, preparer, executor, "1-10", PolicyBudget, 1)
	config.FailureBudget = 2
	summary, err := Run(context.Background(), config)
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
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(uint64) process.Result {
		result := processResult(1, "failure", "")
		result.WorldRecord = recording
		return result
	}}, "7", PolicyAll, 1)
	config.WorldTransitionLimit = 1 << 20
	summary, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := artifact.Open(summary.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	if opened.Manifest.World.Initial.Schema != "gomadv3.world.snapshot/v1" || opened.Manifest.World.Transitions.Count != 1 || opened.Manifest.World.Terminal.Kind != "idle" {
		t.Fatalf("recorded World = %#v", opened.Manifest.World)
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
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(uint64) process.Result {
		result := processResult(0, "", "")
		result.WorldRecord = recording
		return result
	}}, "7", PolicyAll, 1)
	config.WorldTransitionLimit = 1 << 20
	summary, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Failures != 1 || len(summary.Artifacts) != 1 {
		t.Fatalf("summary = %#v", summary)
	}
	opened, err := artifact.Open(summary.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	if opened.Manifest.Outcome.Reason != "world_deadlock" || opened.Manifest.World.Terminal.Kind != "deadlock" {
		t.Fatalf("World deadlock manifest = %#v", opened.Manifest)
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
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(uint64) process.Result {
		result := processResult(0, "", "")
		result.WorldRecord = recording
		return result
	}}, "7", PolicyAll, 1)
	config.WorldTransitionLimit = 1 << 20
	summary, err := Run(context.Background(), config)
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
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(uint64) process.Result {
		result := processResult(1, "failure", "")
		result.WorldRecord = recording
		return result
	}}, "7", PolicyAll, 1)
	config.WorldTransitionLimit = 1 << 20
	summary, err := Run(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "world_record" {
		t.Fatalf("Run() error = %#v", err)
	}
	if len(summary.Artifacts) != 1 {
		t.Fatalf("Runner failure artifacts = %v, want 1", summary.Artifacts)
	}
	opened, openErr := artifact.Open(summary.Artifacts[0])
	if openErr != nil {
		t.Fatal(openErr)
	}
	if opened.Manifest.ArtifactKind != record.ArtifactRunnerFailure || opened.Manifest.Outcome.Reason != "world_record" || opened.Manifest.ReplayMode != record.ReplayNone {
		t.Fatalf("Runner failure manifest = %#v", opened.Manifest)
	}
}

func TestRunRejectsPreparedTargetMutationBeforeFailurePublication(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), mutatingExecutor{}, "1", PolicyAll, 1)
	summary, err := Run(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "prepared_target_integrity" {
		t.Fatalf("Run() error = %#v", err)
	}
	if len(summary.Artifacts) != 0 {
		t.Fatalf("target mutation published replayable artifacts: %v", summary.Artifacts)
	}
}

func TestValidateConfigAcceptsReadOnlyMountsWithoutProfile(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyFirst, 1)
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
	config := testConfig(t, newFakePreparer(t), executor, "1", PolicyAll, 1)
	config.Environment = nil
	config.RunnerBuild = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	config.IOROMounts = []string{source + "=schema"}
	config.Target = target.Spec{
		Kind: target.KindGoTest, Source: "./pkg", Args: []string{"-test.run=^TestScenario$"}, WorkingDir: t.TempDir(),
	}
	config.Preparer = profileFakePreparer(t, "-test.run=^TestScenario$")
	if _, err := Run(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	if len(executor.requests) != 1 || executor.requests[0].IO == nil || executor.requests[0].IO.ReadOnlyMount == nil || len(executor.requests[0].IO.ReadOnlyMount.Mappings) != 1 || executor.requests[0].IO.ReadOnlyMount.Mappings[0].Source != source || executor.requests[0].IO.ReadOnlyMount.Mappings[0].Target != "/schema" {
		t.Fatalf("executor mounts = %#v", executor.requests)
	}
}

func TestRunPassesChoiceProfileToExecutorAndArtifact(t *testing.T) {
	limit := uint64(choicewire.HeaderBytes + choicewire.RecordBytes)
	preparer := newFakePreparer(t)
	implementation, err := choicewire.ImplementationIdentity(preparer.prepared.BuildKey)
	if err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{result: func(uint64) process.Result {
		result := processResult(1, "failure", "")
		result.ChoiceTrace = process.ChoiceTrace{
			Profile: choicewire.Profile, ImplementationSHA256: implementation, Limit: limit,
			Trace: choicewire.Trace{SHA256: sha256.Sum256(nil), Summary: choicewire.Summary{Terminal: choicewire.TerminalComplete}},
		}
		return result
	}}
	config := testConfig(t, preparer, executor, "1", PolicyAll, 1)
	config.ChoiceTraceLimit = limit
	summary, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if len(executor.requests) != 1 || executor.requests[0].Choice == nil || executor.requests[0].Choice.Profile != choicewire.Profile || executor.requests[0].Choice.ImplementationSHA256 != implementation || executor.requests[0].Choice.Limit != limit {
		t.Fatalf("executor choice capability = %#v", executor.requests)
	}
	if summary.ChoiceTrace == nil || summary.ChoiceTrace.Profile != choicewire.Profile || summary.ChoiceTrace.TerminalState != "complete" {
		t.Fatalf("choice summary = %#v", summary.ChoiceTrace)
	}
	opened, err := artifact.Open(summary.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := opened.Close(); err != nil {
			t.Error(err)
		}
	})
	if opened.Manifest.ChoiceProfile == nil || opened.Manifest.ChoiceProfile.Trace.Limit != record.Uint64String(limit) {
		t.Fatalf("artifact choice profile = %#v", opened.Manifest.ChoiceProfile)
	}
}

func TestRunClassifiesInvalidChoiceTraceTerminalEvidence(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		reason string
	}{
		{name: "malformed", err: process.ErrChoiceTraceMalformed, reason: "choice_trace_malformed"},
		{name: "unterminated", err: process.ErrChoiceTraceUnterminated, reason: "choice_trace_unterminated"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := testConfig(t, newFakePreparer(t), terminalErrorExecutor{err: test.err}, "1", PolicyAll, 1)
			config.ChoiceTraceLimit = process.MinimumChoiceTraceBytes
			_, err := Run(context.Background(), config)
			var hostError *HostError
			if !errors.As(err, &hostError) || hostError.Reason != test.reason {
				t.Fatalf("Run() error = %v", err)
			}
		})
	}
}

func TestRunPublishesValidatedChoiceTraceOverflowAsRunnerFailure(t *testing.T) {
	preparer := newFakePreparer(t)
	implementation, err := choicewire.ImplementationIdentity(preparer.prepared.BuildKey)
	if err != nil {
		t.Fatal(err)
	}
	recordBytes, err := choicewire.EncodeRecord(choicewire.Record{
		Ordinal: 0, Kind: choicewire.KindRunnable, Flags: choicewire.FlagDecision, Alternatives: 2, Selected: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	payload := recordBytes[:]
	limit := uint64(choicewire.HeaderBytes + choicewire.RecordBytes)
	result := processResult(0, "", "")
	result.ChoiceTrace = process.ChoiceTrace{
		Profile: choicewire.Profile, ImplementationSHA256: implementation, Limit: limit,
		Trace: choicewire.Trace{
			Bytes: payload, SHA256: sha256.Sum256(payload),
			Summary: choicewire.Summary{Records: 1, Branching: 1, Runnable: 1, Terminal: choicewire.TerminalOverflow},
		},
	}
	config := testConfig(t, preparer, terminalErrorExecutor{result: result, err: process.ErrChoiceTraceOverflow}, "1", PolicyAll, 1)
	config.ChoiceTraceLimit = limit
	summary, err := Run(context.Background(), config)
	var hostErr *HostError
	if !errors.As(err, &hostErr) || hostErr.Reason != "choice_trace_overflow" {
		t.Fatalf("Run() error = %#v", err)
	}
	if len(summary.Artifacts) != 1 || summary.ChoiceTrace == nil || summary.ChoiceTrace.TerminalState != "overflow" {
		t.Fatalf("Run() summary = %#v", summary)
	}
	opened, err := artifact.Open(summary.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
	if opened.Manifest.ArtifactKind != record.ArtifactRunnerFailure || opened.Manifest.Outcome.Reason != "choice_trace_overflow" || opened.Manifest.ChoiceProfile == nil || opened.Manifest.ChoiceProfile.Trace.TerminalState != "overflow" {
		t.Fatalf("overflow manifest = %#v", opened.Manifest)
	}

	resumedExecutor := &fakeExecutor{result: func(uint64) process.Result {
		result := processResult(0, "", "")
		result.ChoiceTrace = process.ChoiceTrace{
			Profile: choicewire.Profile, ImplementationSHA256: implementation, Limit: limit,
			Trace: choicewire.Trace{SHA256: sha256.Sum256(nil), Summary: choicewire.Summary{Terminal: choicewire.TerminalComplete}},
		}
		return result
	}}
	resumed, err := Run(context.Background(), Config{
		ResumeBatch: summary.BatchPath, RunnerBuild: config.RunnerBuild, SupervisorCommand: []string{"unused"}, Executor: resumedExecutor,
	})
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
		"choice profile":  {"GOMADV3_CHOICE_PROFILE=injected"},
		"choice trace fd": {"GOMADV3_CHOICE_TRACE_FD=9"},
		"duplicate":       {"A=1", "A=2"},
		"invalid":         {"NOT-VALID=1"},
		"nul":             {"A=value\x00tail"},
	} {
		t.Run(name, func(t *testing.T) {
			config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
			config.Environment = environment
			if _, err := Run(context.Background(), config); err == nil {
				t.Fatal("Run() succeeded")
			}
		})
	}
}

func TestRunCancellationIsAHostFailure(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), blockingExecutor{}, "1", PolicyAll, 1)
	ctx := cancelOnProgress(t, &config, func(progress Progress) bool { return progress.Running == 1 })
	config.TerminateGrace = 10 * time.Millisecond
	summary, err := Run(ctx, config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "cancelled" || !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %#v", err)
	}
	if summary.Failures != 0 || len(summary.Artifacts) != 0 {
		t.Fatalf("cancelled summary = %#v", summary)
	}
	plan, planErr := artifact.ReadResumePlan(summary.BatchPath)
	if planErr != nil {
		t.Fatal(planErr)
	}
	if plan.Selection != "1" || plan.RunnerBuild != config.RunnerBuild || plan.Prepared.Target.SHA256 == "" {
		t.Fatalf("resume plan = %#v", plan)
	}
	partials, readErr := os.ReadDir(filepath.Join(summary.BatchPath, ".partial"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(partials) != 2 {
		t.Fatalf("cancelled partials = %v, want batch and target", partials)
	}
	if _, err := os.Stat(filepath.Join(summary.BatchPath, ".partial", "batch", "partial.json")); err != nil {
		t.Fatal(err)
	}
}

func TestRunResumesVerifiedBatchAndSkipsCompletedOrdinals(t *testing.T) {
	preparer := newFakePreparer(t)
	interrupted := &resumeInterruptExecutor{}
	config := testConfig(t, preparer, interrupted, "7-9", PolicyAll, 1)
	ctx := cancelOnProgress(t, &config, func(progress Progress) bool { return progress.Succeeded == 1 })
	config.TerminateGrace = 10 * time.Millisecond
	partial, err := Run(ctx, config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "cancelled" {
		t.Fatalf("interrupted Run() error = %v", err)
	}
	if seeds := interrupted.seeds(); !slices.Equal(seeds, []uint64{7, 8}) {
		t.Fatalf("interrupted seeds = %v", seeds)
	}

	resumedExecutor := &fakeExecutor{}
	resumed, err := Run(context.Background(), Config{
		ResumeBatch: partial.BatchPath, RunnerBuild: config.RunnerBuild, SupervisorCommand: []string{"unused"}, Executor: resumedExecutor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if seeds := executorSeeds(resumedExecutor); !slices.Equal(seeds, []uint64{8, 9}) {
		t.Fatalf("resumed seeds = %v", seeds)
	}
	if resumed.BatchPath != partial.BatchPath || resumed.SelectionCount != 3 || resumed.Attempted != 3 || resumed.Succeeded != 3 || resumed.Failures != 0 || resumed.StopReason != StopSeedsExhausted || preparer.calls != 1 {
		t.Fatalf("resumed summary = %#v, preparation calls = %d", resumed, preparer.calls)
	}
	batch, err := artifact.OpenBatch(resumed.BatchPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Runs) != 3 || batch.Runs[0].Seed != 7 || batch.Runs[1].Seed != 8 || batch.Runs[2].Seed != 9 {
		t.Fatalf("batch runs = %#v", batch.Runs)
	}
}

func TestRunResumeRestoresSeenChoiceFeaturesBeforeNovelRetention(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := uint64(choicewire.HeaderBytes + choicewire.RecordBytes)
	trace := completeChoiceTrace(t, preparer.prepared.BuildKey, limit, []choicewire.Record{{
		Ordinal: 0, Kind: choicewire.KindRunnable, Flags: choicewire.FlagDecision, SiteOffset: 24, Alternatives: 2, Selected: 0,
	}})
	interrupted := &choiceResumeInterruptExecutor{trace: trace}
	config := testConfig(t, preparer, interrupted, "7-8", PolicyAll, 1)
	ctx := cancelOnProgress(t, &config, func(progress Progress) bool { return progress.RetainedSuccesses == 1 })
	config.TerminateGrace = 10 * time.Millisecond
	config.Coverage = CoverageChoice
	config.ChoiceTraceLimit = limit
	config.KeepSuccesses = KeepSuccessesNovel
	config.SuccessArtifactLimit = 2
	config.SuccessBytesLimit = 64 << 20
	partial, err := Run(ctx, config)
	if err == nil || partial.RetainedSuccesses != 1 {
		t.Fatalf("interrupted summary = %#v, error = %v", partial, err)
	}

	resumedExecutor := &fakeExecutor{result: func(uint64) process.Result {
		result := processResult(0, "", "")
		result.IOTranscript = completeEmptyTranscript()
		result.ChoiceTrace = trace
		return result
	}}
	resumed, err := Run(context.Background(), Config{
		ResumeBatch: partial.BatchPath, RunnerBuild: config.RunnerBuild, SupervisorCommand: []string{"unused"}, Executor: resumedExecutor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resumed.RetainedSuccesses != 1 || len(resumed.SuccessArtifacts) != 1 {
		t.Fatalf("resumed summary = %#v", resumed)
	}
	batch, err := artifact.OpenBatch(resumed.BatchPath)
	if err != nil {
		t.Fatal(err)
	}
	if batch.Runs[1].SuccessArtifact != nil || len(batch.Runs[1].ChoiceFeatures) == 0 {
		t.Fatalf("resumed run = %#v", batch.Runs[1])
	}
}

func TestRunResumesGuidedBatchWithoutReselectingSeeds(t *testing.T) {
	corpus := filepath.Join(t.TempDir(), "corpus")
	replayer := &matchingReplayer{}
	interrupted := &resumeInterruptExecutor{}
	config := testConfig(t, newFakePreparer(t), interrupted, "7-8", PolicyAll, 1)
	ctx := cancelOnProgress(t, &config, func(progress Progress) bool { return progress.CorpusEntries == 1 })
	config.TerminateGrace = 10 * time.Millisecond
	config.Coverage = CoverageSemantic
	config.Guide = true
	config.Corpus = corpus
	config.Replayer = replayer
	partial, err := Run(ctx, config)
	if err == nil || partial.CorpusEntries != 1 {
		t.Fatalf("interrupted guided summary = %#v, error = %v", partial, err)
	}

	resumedExecutor := &fakeExecutor{result: func(uint64) process.Result {
		result := processResult(0, "", "")
		result.IOTranscript = completeEmptyTranscript()
		return result
	}}
	resumed, err := Run(context.Background(), Config{
		ResumeBatch: partial.BatchPath, RunnerBuild: config.RunnerBuild, SupervisorCommand: []string{"unused"}, Executor: resumedExecutor, Replayer: replayer,
	})
	if err != nil {
		t.Fatal(err)
	}
	if seeds := executorSeeds(resumedExecutor); !slices.Equal(seeds, []uint64{8}) || resumed.SelectionCount != 2 || resumed.CorpusEntries != 1 {
		t.Fatalf("resumed guided seeds = %v, summary = %#v", executorSeeds(resumedExecutor), resumed)
	}
}

func TestRunResumeRejectsChangedRunnerIdentity(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), blockingExecutor{}, "1", PolicyAll, 1)
	ctx := cancelOnProgress(t, &config, func(progress Progress) bool { return progress.Running == 1 })
	config.TerminateGrace = 10 * time.Millisecond
	partial, err := Run(ctx, config)
	if err == nil {
		t.Fatal("Run() did not leave an interrupted batch")
	}
	_, err = Run(context.Background(), Config{
		ResumeBatch: partial.BatchPath, RunnerBuild: "sha256:changed", SupervisorCommand: []string{"unused"}, Executor: &fakeExecutor{},
	})
	if err == nil || !strings.Contains(err.Error(), "Runner build identity") {
		t.Fatalf("resume error = %v", err)
	}
}

func TestRunResumeRejectsTamperedRetainedSuccessArtifact(t *testing.T) {
	interrupted := &resumeInterruptExecutor{}
	config := testConfig(t, newFakePreparer(t), interrupted, "7-8", PolicyAll, 1)
	ctx := cancelOnProgress(t, &config, func(progress Progress) bool { return progress.RetainedSuccesses == 1 })
	config.TerminateGrace = 10 * time.Millisecond
	config.KeepSuccesses = KeepSuccessesAll
	config.SuccessArtifactLimit = 2
	config.SuccessBytesLimit = 64 << 20
	partial, err := Run(ctx, config)
	if err == nil || len(partial.SuccessArtifacts) != 1 {
		t.Fatalf("interrupted summary = %#v, error = %v", partial, err)
	}
	if err := os.WriteFile(filepath.Join(partial.SuccessArtifacts[0], "stdout"), []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = Run(context.Background(), Config{
		ResumeBatch: partial.BatchPath, RunnerBuild: config.RunnerBuild, SupervisorCommand: []string{"unused"}, Executor: &fakeExecutor{},
	})
	if err == nil || !strings.Contains(err.Error(), "retained success") {
		t.Fatalf("resume error = %v", err)
	}
}

func TestRunPreparationFailureLeavesExplicitPartial(t *testing.T) {
	config := testConfig(t, errorPreparer{err: errors.New("build failed")}, &fakeExecutor{}, "1", PolicyAll, 1)
	summary, err := Run(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "target_preparation" {
		t.Fatalf("Run() error = %#v", err)
	}
	partial, readErr := os.ReadFile(filepath.Join(summary.BatchPath, ".partial", "preparation", "partial.json"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(partial), `"state":"failed"`) || !strings.Contains(string(partial), `"reason":"target_preparation"`) {
		t.Fatalf("preparation partial = %s", partial)
	}
}

func TestRunPreparationCancellationIsClassifiedSeparately(t *testing.T) {
	started := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() {
		<-started
		cancel()
	}()
	config := testConfig(t, waitingPreparer{started: started}, &fakeExecutor{}, "1", PolicyAll, 1)
	config.TerminateGrace = 10 * time.Millisecond
	summary, err := Run(ctx, config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "cancelled" || !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %#v", err)
	}
	partial, readErr := os.ReadFile(filepath.Join(summary.BatchPath, ".partial", "preparation", "partial.json"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(partial), `"reason":"cancelled"`) {
		t.Fatalf("preparation partial = %s", partial)
	}
}

func TestRunPreparationOverallTimeoutIsClassifiedSeparately(t *testing.T) {
	started := make(chan struct{})
	deadline := &controlledDeadlineContext{Context: context.Background(), done: make(chan struct{})}
	go func() {
		<-started
		close(deadline.done)
	}()
	config := testConfig(t, waitingPreparer{started: started}, &fakeExecutor{}, "1", PolicyAll, 1)
	config.OverallTimeout = time.Hour
	config.TerminateGrace = 10 * time.Millisecond
	summary, err := Run(deadline, config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "overall_timeout" || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run() error = %#v", err)
	}
	partial, readErr := os.ReadFile(filepath.Join(summary.BatchPath, ".partial", "preparation", "partial.json"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(partial), `"reason":"overall_timeout"`) {
		t.Fatalf("preparation partial = %s", partial)
	}
}

func TestClassifyStableTargetDiagnostics(t *testing.T) {
	for name, stderr := range map[string]string{
		"panic_or_runtime_fatal":         "panic: broken\n",
		"deterministic_deadlock":         "fatal error: all goroutines are asleep - deadlock!\n",
		"logical_test_timeout":           "panic: test timed out after 1m0s\n",
		"unsupported_deterministic_mode": "runtime: GOMADSEED does not support cgo or external linking\n",
	} {
		t.Run(name, func(t *testing.T) {
			outcome := executionoutcome.Classify(processResult(2, "", stderr), false, record.WorldTerminal{Kind: "none"})
			if outcome.Domain != "target" || outcome.Reason != name {
				t.Fatalf("outcome = %#v", outcome)
			}
		})
	}
}

func TestManifestForRunBindsIOProfileIdentity(t *testing.T) {
	preparer := newFakePreparer(t)
	config := testConfig(t, preparer, &fakeExecutor{}, "1", PolicyFirst, 1)
	manifest, err := manifestForRun(config, preparer.prepared, nil, runCompletion{
		job: runJob{seed: 1}, startedAt: time.Unix(1, 0), finishedAt: time.Unix(2, 0), result: processResult(1, "", ""),
	}, executionoutcome.Classification{Domain: "target", Reason: "nonzero_exit", Termination: "exit", ArtifactKind: record.ArtifactTargetFailure, ReplayMode: record.ReplayExact}, "run", record.World{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.IOProfile.Name != "gomadv3-deterministic/v1" || manifest.IOProfile.Inventory == "" || manifest.IOProfile.InventorySHA256 == "" || manifest.IOProfile.ImplementationSHA256 == "" {
		t.Fatalf("manifest I/O profile = %#v", manifest.IOProfile)
	}
}

func TestClassifyStructuredWorldFailures(t *testing.T) {
	for kind, reason := range map[string]string{
		"deadlock": "world_deadlock", "capacity": "world_capacity", "replay-divergence": "world_replay_divergence", "invalid-input": "world_invalid_input",
	} {
		t.Run(kind, func(t *testing.T) {
			outcome := executionoutcome.Classify(processResult(0, "", ""), false, record.WorldTerminal{Kind: kind, Detail: "detail"})
			if outcome.Domain != "target" || outcome.Reason != reason || outcome.Termination != "exit" || outcome.ExitCode == nil || *outcome.ExitCode != 0 {
				t.Fatalf("classify() = %#v", outcome)
			}
		})
	}
}

func TestIsolatedRunnerKillsAStuckCoordinatorInsideOverallDeadline(t *testing.T) {
	config := testConfig(t, nil, nil, "1", PolicyFirst, 1)
	config.OverallTimeout = 200 * time.Millisecond
	config.CoordinatorCommand = []string{os.Args[0], "-test.run=TestBlockingCoordinatorHelper"}
	started := time.Now()
	_, err := Run(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "overall_timeout" {
		t.Fatalf("Run() error = %v", err)
	}
	if elapsed := time.Since(started); elapsed > 350*time.Millisecond {
		t.Fatalf("Run() elapsed = %v", elapsed)
	}
}

func TestIsolatedRunnerPreservesContextReasonAfterCoordinatorExit(t *testing.T) {
	for _, test := range []struct {
		name       string
		cancel     bool
		wantReason string
	}{
		{name: "cancelled", cancel: true, wantReason: "cancelled"},
		{name: "deadline", wantReason: "overall_timeout"},
	} {
		t.Run(test.name, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "coordinator-exited")
			t.Setenv("GOMADV3_COORDINATOR_EXIT_MARKER", marker)
			config := testConfig(t, nil, nil, "1", PolicyFirst, 1)
			config.OverallTimeout = 250 * time.Millisecond
			config.CoordinatorCommand = []string{os.Args[0], "-test.run=^TestExitedCoordinatorWithOpenStdoutHelper$"}
			ctx := context.Background()
			if test.cancel {
				cancelCtx, cancel := context.WithCancel(ctx)
				t.Cleanup(cancel)
				ctx = cancelCtx
				go func() {
					for {
						if _, err := os.Stat(marker); err == nil {
							timer := time.NewTimer(50 * time.Millisecond)
							<-timer.C
							cancel()
							return
						}
						runtime.Gosched()
					}
				}()
			}
			_, err := Run(ctx, config)
			var hostError *HostError
			if !errors.As(err, &hostError) || hostError.Reason != test.wantReason {
				t.Fatalf("Run() error = %v", err)
			}
		})
	}
}

func TestIsolatedRunnerPreservesUnsupportedTargetError(t *testing.T) {
	config := testConfig(t, nil, nil, "1", PolicyFirst, 1)
	config.CoordinatorCommand = []string{os.Args[0], "-test.run=TestUnsupportedTargetCoordinatorHelper"}
	_, err := Run(context.Background(), config)
	var unsupported *target.UnsupportedCapabilityError
	if !errors.As(err, &unsupported) || unsupported.ImportPath != "example.com/target" || unsupported.Capability != "imports os/exec" {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestIsolatedRunnerPreservesMissingSemanticProbesError(t *testing.T) {
	config := testConfig(t, nil, nil, "1", PolicyFirst, 1)
	config.CoordinatorCommand = []string{os.Args[0], "-test.run=TestMissingSemanticProbesCoordinatorHelper"}
	_, err := Run(context.Background(), config)
	var missing *ioprofile.MissingSemanticProbesError
	if !errors.As(err, &missing) || len(missing.Probes) != 1 || missing.Probes[0] != "stdlib.os.openfile" {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestIsolatedRunnerPreservesBoundedRunEvidence(t *testing.T) {
	config := testConfig(t, nil, nil, "1", PolicyAll, 1)
	config.Coverage = CoverageSemantic
	config.CollectRunEvidence = true
	config.CoordinatorCommand = []string{os.Args[0], "-test.run=TestRunEvidenceCoordinatorHelper"}
	summary, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.RunEvidence == nil || summary.RunEvidence.Schema != RunEvidenceSchema || summary.RunEvidence.Seed != 1 || summary.RunEvidence.Target.SHA256 != "sha256:target" {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestIsolatedRunnerTransportsChoiceTraceConfiguration(t *testing.T) {
	config := testConfig(t, nil, nil, "1", PolicyAll, 1)
	config.ChoiceTraceLimit = process.MinimumChoiceTraceBytes
	config.CoordinatorCommand = []string{os.Args[0], "-test.run=TestChoiceTraceCoordinatorHelper"}
	summary, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.ChoiceTrace == nil || summary.ChoiceTrace.Limit != process.MinimumChoiceTraceBytes {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestIsolatedRunnerDrainsFastCoordinatorBeforeWaitClosesOutput(t *testing.T) {
	config := testConfig(t, nil, nil, "1", PolicyAll, 1)
	config.CoordinatorCommand = []string{os.Args[0], "-test.run=TestFastCoordinatorHelper"}
	config.Progress = func(Progress) error {
		deadline := time.Now().Add(100 * time.Millisecond)
		for time.Now().Before(deadline) {
			runtime.Gosched()
		}
		return nil
	}
	summary, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Attempted != 1 || summary.Succeeded != 1 {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestFastCoordinatorHelper(t *testing.T) {
	if os.Getenv("GOMADV3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	encoder := json.NewEncoder(os.Stdout)
	progress := Progress{Phase: ProgressRunning, Attempted: 1, Running: 1}
	if err := encoder.Encode(coordinatorMessage{Type: "progress", Progress: &progress}); err != nil {
		t.Fatal(err)
	}
	response := coordinatorResponse{Summary: Summary{Attempted: 1, Succeeded: 1}}
	if err := encoder.Encode(coordinatorMessage{Type: "result", Response: &response}); err != nil {
		t.Fatal(err)
	}
	os.Exit(0) //nolint:revive // This subprocess helper must exit before the parent test harness continues.
}

func TestUnsupportedTargetCoordinatorHelper(t *testing.T) {
	if os.Getenv("GOMADV3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	unsupported := &target.UnsupportedCapabilityError{ImportPath: "example.com/target", Capability: "imports os/exec"}
	response := coordinatorResponse{
		ErrorReason: "target_preparation", ErrorDetail: unsupported.Error(), UnsupportedTarget: unsupported,
	}
	if err := json.NewEncoder(os.Stdout).Encode(coordinatorMessage{Type: "result", Response: &response}); err != nil {
		t.Fatal(err)
	}
	os.Exit(0)
}

func TestMissingSemanticProbesCoordinatorHelper(t *testing.T) {
	if os.Getenv("GOMADV3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	missing := &ioprofile.MissingSemanticProbesError{Probes: []string{"stdlib.os.openfile"}}
	response := coordinatorResponse{
		ErrorReason: "semantic_coverage", ErrorDetail: missing.Error(), MissingSemanticProbes: missing.Probes,
	}
	if err := json.NewEncoder(os.Stdout).Encode(coordinatorMessage{Type: "result", Response: &response}); err != nil {
		t.Fatal(err)
	}
	os.Exit(0)
}

func TestRunEvidenceCoordinatorHelper(t *testing.T) {
	if os.Getenv("GOMADV3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	response := coordinatorResponse{Summary: Summary{RunEvidence: &RunEvidence{
		Schema: RunEvidenceSchema, Seed: 1, Target: record.Target{SHA256: "sha256:target"},
	}}}
	if err := json.NewEncoder(os.Stdout).Encode(coordinatorMessage{Type: "result", Response: &response}); err != nil {
		t.Fatal(err)
	}
	os.Exit(0)
}

func TestChoiceTraceCoordinatorHelper(t *testing.T) {
	if os.Getenv("GOMADV3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	var wire coordinatorConfig
	if err := json.NewDecoder(os.Stdin).Decode(&wire); err != nil {
		t.Fatal(err)
	}
	response := coordinatorResponse{Summary: Summary{ChoiceTrace: &ChoiceTraceSummary{Limit: wire.ChoiceTraceLimit}}}
	if err := json.NewEncoder(os.Stdout).Encode(coordinatorMessage{Type: "result", Response: &response}); err != nil {
		t.Fatal(err)
	}
	os.Exit(0)
}

func TestBlockingCoordinatorHelper(t *testing.T) {
	if os.Getenv("GOMADV3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	for {
		runtime.Gosched()
	}
}

func TestExitedCoordinatorWithOpenStdoutHelper(t *testing.T) {
	if os.Getenv("GOMADV3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	command := exec.Command(os.Args[0], "-test.run=^TestCoordinatorStdoutDescendantHelper$")
	command.Env = append(os.Environ(), "GOMADV3_COORDINATOR_STDOUT_DESCENDANT=1")
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("GOMADV3_COORDINATOR_EXIT_MARKER"), []byte("exiting"), 0o600); err != nil {
		t.Fatal(err)
	}
	os.Exit(0) //nolint:revive // This subprocess helper must exit while its descendant retains stdout.
}

func TestCoordinatorStdoutDescendantHelper(t *testing.T) {
	if os.Getenv("GOMADV3_COORDINATOR_STDOUT_DESCENDANT") != "1" {
		t.Skip("coordinator stdout descendant subprocess only")
	}
	<-time.After(10 * time.Second)
}

func TestIsolatedRunnerBoundsCoordinatorOutput(t *testing.T) {
	config := testConfig(t, nil, nil, "1", PolicyFirst, 1)
	config.CoordinatorCommand = []string{os.Args[0], "-test.run=TestOversizedCoordinatorHelper"}
	_, err := Run(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "coordinator_decode" {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestOversizedCoordinatorHelper(t *testing.T) {
	if os.Getenv("GOMADV3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	if _, err := os.Stdout.Write(make([]byte, maximumCoordinatorMessageBytes+1)); err != nil {
		t.Fatal(err)
	}
}

func TestIsolatedRunnerRemovesCoordinatorProcessGroup(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "descendant-survived")
	t.Setenv("GOMADV3_COORDINATOR_DESCENDANT_MARKER", marker)
	config := testConfig(t, nil, nil, "1", PolicyFirst, 1)
	config.OverallTimeout = 250 * time.Millisecond
	config.CoordinatorCommand = []string{os.Args[0], "-test.run=TestCoordinatorTreeHelper"}
	_, err := Run(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "overall_timeout" {
		t.Fatalf("Run() error = %v", err)
	}
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if _, statErr := os.Stat(marker); statErr == nil {
			t.Fatal("coordinator descendant survived cleanup")
		} else if !os.IsNotExist(statErr) {
			t.Fatal(statErr)
		}
		runtime.Gosched()
	}
}

func TestCoordinatorTreeHelper(t *testing.T) {
	if os.Getenv("GOMADV3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	command := exec.Command(os.Args[0], "-test.run=TestCoordinatorDescendantHelper")
	command.Env = append(os.Environ(), "GOMADV3_COORDINATOR_DESCENDANT=1")
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	for {
		runtime.Gosched()
	}
}

func TestCoordinatorDescendantHelper(t *testing.T) {
	if os.Getenv("GOMADV3_COORDINATOR_DESCENDANT") != "1" {
		t.Skip("coordinator descendant subprocess only")
	}
	signal.Ignore(syscall.SIGTERM)
	<-time.After(350 * time.Millisecond)
	if err := os.WriteFile(os.Getenv("GOMADV3_COORDINATOR_DESCENDANT_MARKER"), []byte("survived"), 0o600); err != nil {
		t.Fatal(err)
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
	return &fakePreparer{prepared: target.Prepared{
		Path: path, Kind: target.KindGoRun, Source: ".", SHA256: fmt.Sprintf("sha256:%x", digest), Size: uint64(len(data)),
		Argv: []string{"gomadv3-target"}, BuildTags: []string{}, Compatibility: []record.CompatibilityPack{}, BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"},
		GoVersion: "go1.26.4", BuildKey: "cbeccfefbc62a2ca026d9dded0316ecedfce33bd46b5c71b6645e86b67a0713e",
		TargetGOOS: "darwin", TargetGOARCH: "arm64",
	}}
}

func profileFakePreparer(t *testing.T, argument string) *fakePreparer {
	t.Helper()
	preparer := newFakePreparer(t)
	preparer.prepared.Kind = target.KindGoTest
	preparer.prepared.Source = "./pkg"
	preparer.prepared.Argv = []string{"gomadv3-target", argument}
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
	requests      []process.Request
	result        func(uint64) process.Result
}

type terminalErrorExecutor struct {
	result process.Result
	err    error
}

func (executor terminalErrorExecutor) Run(context.Context, process.Request) (process.Result, error) {
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

func (replayer *matchingReplayer) Replay(_ context.Context, config replay.Config) (replay.Result, error) {
	replayer.calls++
	opened, err := artifact.Open(config.ArtifactPath)
	if err != nil {
		return replay.Result{}, err
	}
	defer opened.Close()
	return replay.Result{Artifact: opened.Detached(), Verified: true, Match: true}, nil
}

func newOutOfOrderExecutor(t *testing.T) *outOfOrderExecutor {
	return &outOfOrderExecutor{t: t, later: make(chan struct{})}
}

func (executor *outOfOrderExecutor) Run(ctx context.Context, request process.Request) (process.Result, error) {
	seed := seedFromEnvironment(request.Env)
	if seed == 1 {
		select {
		case <-executor.later:
		case <-ctx.Done():
			return process.Result{}, ctx.Err()
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

func (executor *fakeExecutor) Run(_ context.Context, request process.Request) (process.Result, error) {
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
	trace process.ChoiceTrace
}

type progressGatedExecutor struct {
	started chan struct{}
	release chan struct{}
}

type mutatingExecutor struct{}

func (mutatingExecutor) Run(_ context.Context, request process.Request) (process.Result, error) {
	if err := os.Chmod(request.Command, 0o700); err != nil {
		return process.Result{}, err
	}
	file, err := os.OpenFile(request.Command, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return process.Result{}, err
	}
	if _, err := file.Write([]byte("mutation")); err != nil {
		file.Close()
		return process.Result{}, err
	}
	if err := file.Close(); err != nil {
		return process.Result{}, err
	}
	return processResult(1, "failure", ""), nil
}

func (blockingExecutor) Run(ctx context.Context, _ process.Request) (process.Result, error) {
	<-ctx.Done()
	result := processResult(0, "", "")
	result.Cancelled = true
	result.Termination = process.TerminationSignal
	result.Signal = "killed"
	return result, nil
}

func (executor *resumeInterruptExecutor) Run(ctx context.Context, request process.Request) (process.Result, error) {
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
	result.Termination = process.TerminationSignal
	result.Signal = "killed"
	return result, nil
}

func (executor *choiceResumeInterruptExecutor) Run(ctx context.Context, request process.Request) (process.Result, error) {
	if seedFromEnvironment(request.Env) == 7 {
		result := processResult(0, "", "")
		result.IOTranscript = completeEmptyTranscript()
		result.ChoiceTrace = executor.trace
		return result, nil
	}
	<-ctx.Done()
	result := processResult(0, "", "")
	result.Cancelled = true
	result.Termination = process.TerminationSignal
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

func (executor *progressGatedExecutor) Run(context.Context, process.Request) (process.Result, error) {
	close(executor.started)
	<-executor.release
	return processResult(0, "", ""), nil
}

func newFirstFailureExecutor(want int) *firstFailureExecutor {
	return &firstFailureExecutor{started: make(chan struct{}), want: want}
}

func (executor *firstFailureExecutor) Run(ctx context.Context, request process.Request) (process.Result, error) {
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
	result.Termination = process.TerminationSignal
	result.Signal = "killed"
	return result, nil
}

func testConfig(t *testing.T, preparer Preparer, executor Executor, seeds string, policy FailurePolicy, parallel int) Config {
	t.Helper()
	return Config{
		Seeds: seeds, Parallel: parallel, RunTimeout: time.Second, OverallTimeout: 10 * time.Second, TerminateGrace: 100 * time.Millisecond,
		OnFailure: policy, FailureBudget: 1, OutputLimit: 64, WorldTransitionLimit: 64, Artifacts: t.TempDir(),
		Environment: []string{"MODE=test"}, Target: target.Spec{Kind: target.KindGoRun, Source: "."}, SupervisorCommand: []string{"unused"},
		RunnerBuild: "sha256:0000000000000000000000000000000000000000000000000000000000000000", Preparer: preparer, Executor: executor,
	}
}

func cancelOnProgress(t *testing.T, config *Config, predicate func(Progress) bool) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	config.ProgressInterval = time.Millisecond
	config.Progress = func(progress Progress) error {
		if predicate(progress) {
			cancel()
		}
		return nil
	}
	return ctx
}

func processResult(exitCode int, stdout, stderr string) process.Result {
	return process.Result{
		Captured: true, Termination: process.TerminationExit, ExitCode: exitCode, GroupGone: true,
		Stdout: output(stdout), Stderr: output(stderr),
	}
}

func output(value string) process.Output {
	bytes := []byte(value)
	digest := sha256.Sum256(bytes)
	return process.Output{Bytes: bytes, FullSHA256: digest, RetainedSHA256: digest, TotalBytes: uint64(len(bytes)), RetainedBytes: uint64(len(bytes))}
}

func semanticTranscript(t *testing.T, probe string) process.IOTranscript {
	t.Helper()
	digest := sha256.Sum256([]byte("gomadv3-boundary-probe/v1\x00" + probe))
	var argument [8]byte
	binary.BigEndian.PutUint64(argument[:], binary.BigEndian.Uint64(digest[:8])&(1<<63-1))
	encoded, err := iowire.EncodeTranscriptRecord(iowire.TranscriptRecord{
		Operation: "boundary.probe", ArgumentHash: iowire.Hash(argument[:]),
	})
	if err != nil {
		t.Fatal(err)
	}
	payload := encoded[:]
	return process.IOTranscript{Bytes: payload, SHA256: sha256.Sum256(payload), Records: 1, Complete: true}
}

func completeEmptyTranscript() process.IOTranscript {
	return process.IOTranscript{Complete: true, SHA256: sha256.Sum256(nil)}
}

func completeChoiceTrace(t *testing.T, buildKey string, limit uint64, records []choicewire.Record) process.ChoiceTrace {
	t.Helper()
	payload := make([]byte, 0, len(records)*choicewire.RecordBytes)
	for _, choiceRecord := range records {
		encoded, err := choicewire.EncodeRecord(choiceRecord)
		if err != nil {
			t.Fatal(err)
		}
		payload = append(payload, encoded[:]...)
	}
	digest := sha256.Sum256(payload)
	terminal := choicewire.EncodeTerminal(choicewire.Terminal{
		State: choicewire.TerminalComplete, Records: uint64(len(records)), MappingBytes: choicewire.HeaderBytes + uint64(len(payload)), PayloadHash: digest,
	})
	trace, err := choicewire.DecodeTrace(payload, terminal[:], limit)
	if err != nil {
		t.Fatal(err)
	}
	implementation, err := choicewire.ImplementationIdentity(buildKey)
	if err != nil {
		t.Fatal(err)
	}
	return process.ChoiceTrace{Profile: choicewire.Profile, ImplementationSHA256: implementation, Limit: limit, Trace: trace}
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
