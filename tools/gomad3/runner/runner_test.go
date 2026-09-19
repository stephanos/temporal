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

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/internal/hostexec"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	simulationengine "go.temporal.io/server/tools/gomad3/runner/internal/exploration/simulation"
	simulationrecord "go.temporal.io/server/tools/gomad3/runner/internal/exploration/simulationrecord"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad3/world"
)

func TestRunPreparesOnceBoundsParallelismAndGroupsMatchingFailures(t *testing.T) {
	preparer := newFakePreparer(t)
	executor := &fakeExecutor{result: func(seed uint64) execution.Result {
		if seed%2 == 0 {
			return processResult(2, "same failure", "")
		}
		return processResult(0, "success", "")
	}}
	summary, err := Explore(context.Background(), testConfig(t, preparer, executor, "1-6", PolicyAll, 2))
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
	if opened.Manifest.Target.BuildTags == nil {
		t.Fatal("empty target build tags encoded as null")
	}
	if _, err := os.Stat(filepath.Join(summary.CampaignPath, "campaign.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(summary.CampaignPath, ".prepared")); !os.IsNotExist(err) {
		t.Fatalf("prepared target retained after publication: %v", err)
	}
	if got := executor.environments(); len(got) != 6 {
		t.Fatalf("target environments = %v", got)
	} else {
		for _, environment := range got {
			if len(environment) != 4 || environment[1] != "GOMAD3_IO_PROFILE=gomad3-deterministic/v1" || environment[2] != "MODE=test" || environment[3] != "TZ=UTC" || !strings.HasPrefix(environment[0], "GOMADSEED=") {
				t.Fatalf("target environment = %v", environment)
			}
		}
	}
	if !allUnique(executor.directories()) {
		t.Fatalf("working directories were reused: %v", executor.directories())
	}
}

func TestRunReportsPreparationProgressAndCompletedCounts(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(seed uint64) execution.Result {
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
	summary, err := Explore(context.Background(), config)
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
	config := testConfig(t, newFakePreparer(t), executor, "1", PolicyAll, 1)
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
	go func() {
		_, err := Explore(context.Background(), config)
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
	executor := &fakeExecutor{result: func(uint64) execution.Result {
		result := processResult(0, "", "")
		result.IOTranscript = semanticTranscript(t, "stdlib.os.openfile")
		return result
	}}
	config := testConfig(t, newFakePreparer(t), executor, "1", PolicyAll, 1)
	config.Coverage = CoverageSemantic
	summary, err := Explore(context.Background(), config)
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
	_, err := Explore(context.Background(), config)
	var missing *deterministicio.MissingSemanticProbesError
	if !errors.As(err, &missing) || len(missing.Probes) != 1 || missing.Probes[0] != "stdlib.os.openfile" {
		t.Fatalf("Explore() error = %v", err)
	}
}

func TestRunRetainsOnlyProbeNovelSuccessesWithinExplicitBounds(t *testing.T) {
	executor := &fakeExecutor{result: func(seed uint64) execution.Result {
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
	summary, err := Explore(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Succeeded != 3 || summary.RetainedSuccesses != 2 || summary.RetainedSuccessBytes == 0 || len(summary.SuccessArtifacts) != 2 {
		t.Fatalf("summary = %#v", summary)
	}
	batch, err := campaign.OpenCampaign(summary.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if batch.Executions[0].SuccessArtifact == nil || batch.Executions[1].SuccessArtifact != nil || batch.Executions[2].SuccessArtifact == nil || !slices.Equal(batch.Executions[0].NovelSemanticProbes, []string{"stdlib.os.openfile"}) || !slices.Equal(batch.Executions[2].NovelSemanticProbes, []string{"stdlib.net.dialtcp"}) {
		t.Fatalf("batch runs = %#v", batch.Executions)
	}
	for _, path := range summary.SuccessArtifacts {
		opened, err := artifact.OpenArtifact(path)
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
	limit := choiceTraceLimit(t, 3)
	executor := &fakeExecutor{result: func(seed uint64) execution.Result {
		result := processResult(0, "", "")
		result.IOTranscript = completeEmptyTranscript()
		selected := uint32(0)
		if seed == 3 {
			selected = 1
		}
		result.ChoiceTrace = completeChoiceTrace(t, preparer.prepared.BuildKey, limit, []choice.Record{{
			Ordinal: 0, Kind: choice.KindRunnable, Flags: choice.FlagDecision, SiteOffset: 24, Alternatives: 2, Selected: selected,
		}})
		return result
	}}
	config := testConfig(t, preparer, executor, "1-3", PolicyAll, 1)
	config.Coverage = CoverageChoice
	config.ChoiceTraceLimit = limit
	config.KeepSuccesses = KeepSuccessesNovel
	config.SuccessArtifactLimit = 2
	config.SuccessBytesLimit = 64 << 20
	summary, err := Explore(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.RetainedSuccesses != 2 || len(summary.SuccessArtifacts) != 2 {
		t.Fatalf("summary = %#v", summary)
	}
	batch, err := campaign.OpenCampaign(summary.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if batch.Executions[0].SuccessArtifact == nil || len(batch.Executions[0].NovelChoiceFeatures) == 0 || batch.Executions[1].SuccessArtifact != nil || len(batch.Executions[1].NovelChoiceFeatures) != 0 || batch.Executions[2].SuccessArtifact == nil || len(batch.Executions[2].NovelChoiceFeatures) == 0 {
		t.Fatalf("choice novelty runs = %#v", batch.Executions)
	}
	for _, run := range batch.Executions {
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
	summary, err := Explore(context.Background(), config)
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

func TestRunGuidesFromImmutableCorpusAndKeepsUnguidedSeeds(t *testing.T) {
	corpus := filepath.Join(t.TempDir(), "corpus")
	replayer := &matchingReplayer{}
	firstExecutor := &fakeExecutor{result: func(seed uint64) execution.Result {
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
	first, err := Explore(context.Background(), firstConfig)
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
	second, err := Explore(context.Background(), secondConfig)
	if err != nil {
		t.Fatal(err)
	}
	if second.CorpusAdded != 0 || second.CorpusEntries != 2 || replayer.calls != 2 {
		t.Fatalf("second guided summary = %#v, replay calls = %d", second, replayer.calls)
	}
	batch, err := campaign.OpenCampaign(second.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if batch.Record.Selection != "0,1,100-105" || batch.Record.SelectionCount != 8 {
		t.Fatalf("guided batch selection = %q (%d)", batch.Record.Selection, batch.Record.SelectionCount)
	}
	seeds := make([]uint64, len(batch.Executions))
	for index, run := range batch.Executions {
		seeds[index] = uint64(run.Seed)
	}
	if !slices.Equal(seeds, []uint64{0, 1, 100, 101, 102, 103, 104, 105}) {
		t.Fatalf("second guided seeds = %v", seeds)
	}
}

func TestRunGuidesFromReplayVerifiedChoiceCoverage(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	replayer := &matchingReplayer{}
	executor := &fakeExecutor{result: func(seed uint64) execution.Result {
		result := processResult(0, "", "")
		result.IOTranscript = completeEmptyTranscript()
		result.ChoiceTrace = completeChoiceTrace(t, preparer.prepared.BuildKey, limit, []choice.Record{{
			Ordinal: 0, Kind: choice.KindRunnable, Flags: choice.FlagDecision, SiteOffset: 24, Alternatives: 2, Selected: uint32(seed % 2),
		}})
		return result
	}}
	config := testConfig(t, preparer, executor, "1-2", PolicyAll, 1)
	config.Coverage = CoverageChoice
	config.ChoiceTraceLimit = limit
	config.Guide = true
	config.Corpus = filepath.Join(t.TempDir(), "corpus")
	config.Replayer = replayer
	summary, err := Explore(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.CorpusAdded != 2 || summary.CorpusEntries != 2 || replayer.calls != 2 {
		t.Fatalf("guided choice summary = %#v, replay calls = %d", summary, replayer.calls)
	}
}

func TestRunGuidanceRequiresCorpusAndSemanticCoverage(t *testing.T) {
	for _, configure := range []func(*CampaignSpec){
		func(config *CampaignSpec) { config.Corpus = "" },
		func(config *CampaignSpec) { config.Coverage = CoverageNone },
	} {
		config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
		config.Guide = true
		config.Corpus = t.TempDir()
		config.Coverage = CoverageSemantic
		configure(&config)
		if _, err := Explore(context.Background(), config); err == nil {
			t.Fatal("Run accepted invalid guided configuration")
		}
	}
}

func TestRunGuidanceWithoutReplayCapabilityFailsClosed(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(uint64) execution.Result {
		result := processResult(0, "", "")
		result.IOTranscript = semanticTranscript(t, "stdlib.os.openfile")
		return result
	}}, "1", PolicyAll, 1)
	config.Coverage = CoverageSemantic
	config.Guide = true
	config.Corpus = filepath.Join(t.TempDir(), "corpus")
	config.SupervisorCommand = nil
	_, err := Explore(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "guided_corpus" {
		t.Fatalf("Explore() error = %v", err)
	}
}

func TestRunFailsClosedWhenSuccessRetentionCountIsExhausted(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(uint64) execution.Result {
		result := processResult(0, "", "")
		result.IOTranscript = completeEmptyTranscript()
		return result
	}}, "1-2", PolicyAll, 1)
	config.KeepSuccesses = KeepSuccessesAll
	config.SuccessArtifactLimit = 1
	config.SuccessBytesLimit = 64 << 20
	summary, err := Explore(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "success_retention_capacity" || summary.RetainedSuccesses != 1 {
		t.Fatalf("summary = %#v, error = %v", summary, err)
	}
}

func TestRunRequiresExplicitSuccessRetentionBounds(t *testing.T) {
	for _, configure := range []func(*CampaignSpec){
		func(config *CampaignSpec) { config.SuccessArtifactLimit = 0 },
		func(config *CampaignSpec) { config.SuccessBytesLimit = 0 },
		func(config *CampaignSpec) { config.KeepSuccesses = KeepSuccessesNovel; config.Coverage = CoverageNone },
	} {
		config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
		config.KeepSuccesses = KeepSuccessesAll
		config.SuccessArtifactLimit = 1
		config.SuccessBytesLimit = 1 << 20
		configure(&config)
		if _, err := Explore(context.Background(), config); err == nil {
			t.Fatal("Explore() accepted invalid success retention configuration")
		}
	}
}

func TestRunRequiresBoundedChoiceTraceCapacity(t *testing.T) {
	for _, limit := range []uint64{1, (64 << 20) + 1} {
		config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
		config.ChoiceTraceLimit = limit
		if _, err := Explore(context.Background(), config); err == nil || !strings.Contains(err.Error(), "choice trace") {
			t.Fatalf("Explore() with choice limit %d error = %v", limit, err)
		}
	}
}

func TestValidateConfigRequiresBoundedSingleSeedChoiceExploration(t *testing.T) {
	valid := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "7", PolicyAll, 1)
	valid.Strategy = StrategyChoiceExploration
	valid.ChoiceTraceLimit = execution.MinimumChoiceTraceBytes
	valid.MaxExecutions = 8
	valid.MaxChoiceDepth = 4
	valid.MaxExplorationBytes = 1 << 20
	if _, _, err := validateConfig(valid); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name      string
		configure func(*CampaignSpec)
		want      string
	}{
		{name: "multiple seeds", configure: func(config *CampaignSpec) { config.Seeds = "7-8" }, want: "exactly one base seed"},
		{name: "guidance", configure: func(config *CampaignSpec) {
			config.Guide = true
			config.Corpus = t.TempDir()
			config.Coverage = CoverageSemantic
		}, want: "does not support guided exploration"},
		{name: "missing choice trace", configure: func(config *CampaignSpec) { config.ChoiceTraceLimit = 0 }, want: "requires an enabled choice trace"},
		{name: "missing execution bound", configure: func(config *CampaignSpec) { config.MaxExecutions = 0 }, want: "max executions"},
		{name: "missing depth bound", configure: func(config *CampaignSpec) { config.MaxChoiceDepth = 0 }, want: "choice depth"},
		{name: "missing exploration bound", configure: func(config *CampaignSpec) { config.MaxExplorationBytes = 0 }, want: "exploration bytes"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := valid
			test.configure(&config)
			if _, _, err := validateConfig(config); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("validateConfig() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestValidateConfigRequiresBoundedSingleSeedSimulationExploration(t *testing.T) {
	valid := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "7", PolicyAll, 1)
	valid.Strategy = StrategySimulationExploration
	valid.ChoiceTraceLimit = execution.MinimumChoiceTraceBytes
	valid.MaxExecutions = 8
	valid.MaxForcedDecisions = 4
	valid.MaxExplorationBytes = 1 << 20
	valid.MaxExplorationResultBytes = 1 << 20
	valid.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 4, Scenario: 4, Network: 4, Storage: 4, Fault: 4, Crash: 4}
	if _, _, err := validateConfig(valid); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name      string
		configure func(*CampaignSpec)
		want      string
	}{
		{name: "multiple seeds", configure: func(config *CampaignSpec) { config.Seeds = "7-8" }, want: "exactly one base seed"},
		{name: "guidance", configure: func(config *CampaignSpec) {
			config.Guide = true
			config.Corpus = t.TempDir()
			config.Coverage = CoverageSemantic
		}, want: "does not support guided exploration"},
		{name: "missing choice trace", configure: func(config *CampaignSpec) { config.ChoiceTraceLimit = 0 }, want: "requires an enabled choice trace"},
		{name: "missing execution bound", configure: func(config *CampaignSpec) { config.MaxExecutions = 0 }, want: "max executions"},
		{name: "missing forced-decision bound", configure: func(config *CampaignSpec) { config.MaxForcedDecisions = 0 }, want: "forced decisions"},
		{name: "missing exploration bound", configure: func(config *CampaignSpec) { config.MaxExplorationBytes = 0 }, want: "exploration bytes"},
		{name: "missing result bound", configure: func(config *CampaignSpec) { config.MaxExplorationResultBytes = 0 }, want: "result bytes"},
		{name: "missing dimension bound", configure: func(config *CampaignSpec) { config.SimulationDimensionLimits.Network = 0 }, want: "network dimension"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := valid
			test.configure(&config)
			if _, _, err := validateConfig(config); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("validateConfig() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestValidateConfigRejectsExplorationBoundsForSeedStrategy(t *testing.T) {
	for _, configure := range []func(*CampaignSpec){
		func(config *CampaignSpec) { config.MaxExecutions = 1 },
		func(config *CampaignSpec) { config.MaxChoiceDepth = 1 },
		func(config *CampaignSpec) { config.MaxExplorationBytes = 1 },
	} {
		config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "7", PolicyAll, 1)
		configure(&config)
		if _, _, err := validateConfig(config); err == nil || !strings.Contains(err.Error(), "choice-exploration strategy") {
			t.Fatalf("validateConfig() error = %v", err)
		}
	}
}

func TestRunRejectsSuccessfulRetentionWithoutReplayTranscript(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
	config.KeepSuccesses = KeepSuccessesAll
	config.SuccessArtifactLimit = 1
	config.SuccessBytesLimit = 1 << 20
	_, err := Explore(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "success_artifact_publication" || !strings.Contains(err.Error(), "complete I/O transcript") {
		t.Fatalf("Explore() error = %v", err)
	}
}

func TestRunCollectsBoundedQualificationEvidenceForOneSeed(t *testing.T) {
	executor := &fakeExecutor{result: func(uint64) execution.Result {
		result := processResult(0, "stdout", "stderr")
		result.IOTranscript = semanticTranscript(t, "stdlib.os.openfile")
		return result
	}}
	config := testConfig(t, newFakePreparer(t), executor, "7", PolicyAll, 1)
	config.Coverage = CoverageSemantic
	config.CollectExecutionEvidence = true
	summary, err := Explore(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.ExecutionEvidence == nil || summary.ExecutionEvidence.Seed != 7 || summary.ExecutionEvidence.Target.SHA256 == "" || summary.ExecutionEvidence.Stdout.FullSHA256 != record.HashBytes([]byte("stdout")) || summary.ExecutionEvidence.Stderr.FullSHA256 != record.HashBytes([]byte("stderr")) || summary.ExecutionEvidence.IOTranscriptRecords != 1 || summary.ExecutionEvidence.SemanticCoverage.Digest == "" {
		t.Fatalf("execution evidence = %#v", summary.ExecutionEvidence)
	}
}

func TestExecutionEvidenceRequiresOneSeedAndSemanticCoverage(t *testing.T) {
	for _, configure := range []func(*CampaignSpec){
		func(config *CampaignSpec) { config.Seeds = "1-2" },
		func(config *CampaignSpec) { config.Coverage = CoverageNone },
	} {
		config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
		config.Coverage = CoverageSemantic
		config.CollectExecutionEvidence = true
		configure(&config)
		if _, err := Explore(context.Background(), config); err == nil {
			t.Fatal("Explore() accepted invalid evidence configuration")
		}
	}
}

func TestExecutionEvidenceIgnoresAggregateCoordinatorDeadlineAdjustment(t *testing.T) {
	var runRecords []ExecutionEvidence
	for _, overallTimeout := range []time.Duration{9 * time.Second, 10 * time.Second} {
		config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "7", PolicyAll, 1)
		config.Coverage = CoverageSemantic
		config.CollectExecutionEvidence = true
		config.OverallTimeout = overallTimeout
		summary, err := Explore(context.Background(), config)
		if err != nil {
			t.Fatal(err)
		}
		runRecords = append(runRecords, *summary.ExecutionEvidence)
	}
	first, err := canonicaljson.CanonicalJSON(runRecords[0])
	if err != nil {
		t.Fatal(err)
	}
	second, err := canonicaljson.CanonicalJSON(runRecords[1])
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
	summary, err := Explore(context.Background(), config)
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
		context.Background(), config, storeRoot, record.HashBytes([]byte("new")), distinct, &storedBytes, artifact.ArtifactInput{},
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
	config := testConfig(t, preparer, executor, "1-10", PolicyFirst, 3)
	summary, err := Explore(context.Background(), config)
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
	executor := &fakeExecutor{result: func(seed uint64) execution.Result {
		output := "same"
		if seed == 4 {
			output = "different"
		}
		return processResult(1, output, "")
	}}
	config := testConfig(t, preparer, executor, "1-10", PolicyBudget, 1)
	config.FailureBudget = 2
	summary, err := Explore(context.Background(), config)
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
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(uint64) execution.Result {
		result := processResult(1, "failure", "")
		result.WorldRecord = recording
		return result
	}}, "7", PolicyAll, 1)
	config.WorldTransitionLimit = 1 << 20
	summary, err := Explore(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := artifact.OpenArtifact(summary.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	if opened.Manifest.World.Initial.Schema != "gomad3.world.snapshot/v1" || opened.Manifest.World.Transitions.Count != 1 || opened.Manifest.World.Terminal.Kind != "idle" {
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
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(uint64) execution.Result {
		result := processResult(0, "", "")
		result.WorldRecord = recording
		return result
	}}, "7", PolicyAll, 1)
	config.WorldTransitionLimit = 1 << 20
	summary, err := Explore(context.Background(), config)
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
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(uint64) execution.Result {
		result := processResult(0, "", "")
		result.WorldRecord = recording
		return result
	}}, "7", PolicyAll, 1)
	config.WorldTransitionLimit = 1 << 20
	summary, err := Explore(context.Background(), config)
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
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(uint64) execution.Result {
		result := processResult(1, "failure", "")
		result.WorldRecord = recording
		return result
	}}, "7", PolicyAll, 1)
	config.WorldTransitionLimit = 1 << 20
	summary, err := Explore(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "world_record" {
		t.Fatalf("Explore() error = %#v", err)
	}
	if len(summary.Artifacts) != 1 {
		t.Fatalf("Runner failure artifacts = %v, want 1", summary.Artifacts)
	}
	opened, openErr := artifact.OpenArtifact(summary.Artifacts[0])
	if openErr != nil {
		t.Fatal(openErr)
	}
	if opened.Manifest.ArtifactKind != record.ArtifactRunnerFailure || opened.Manifest.Outcome.Reason != "world_record" || opened.Manifest.ReplayMode != record.ReplayNone {
		t.Fatalf("Runner failure manifest = %#v", opened.Manifest)
	}
}

func TestRunRejectsPreparedTargetMutationBeforeFailurePublication(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), mutatingExecutor{}, "1", PolicyAll, 1)
	summary, err := Explore(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "prepared_target_integrity" {
		t.Fatalf("Explore() error = %#v", err)
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
	if _, err := Explore(context.Background(), config); err != nil {
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
	executor := &fakeExecutor{result: func(uint64) execution.Result {
		result := processResult(1, "failure", "")
		result.ChoiceTrace = completeChoiceTrace(t, preparer.prepared.BuildKey, limit, nil)
		return result
	}}
	config := testConfig(t, preparer, executor, "1", PolicyAll, 1)
	config.ChoiceTraceLimit = limit
	summary, err := Explore(context.Background(), config)
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
	if opened.Manifest.ChoiceProfile == nil || opened.Manifest.ChoiceProfile.Trace.Schema != "gomad3.choice-trace/v2" || opened.Manifest.ChoiceProfile.Trace.Limit != record.Uint64String(limit) || opened.Manifest.ChoiceProfile.Trace.TapeSHA256 == "" {
		t.Fatalf("artifact choice profile = %#v", opened.Manifest.ChoiceProfile)
	}
}

func TestSimulationCapabilityForJobCarriesDetachedExplorationPlanToInjectedExecutor(t *testing.T) {
	plan := []byte(`{"schema":"gomad3.simulation-exploration-plan/v1"}`)
	job := runJob{simulationPlan: string(plan), simulationRecordLimit: 1 << 20, simulationRecordCount: 1}
	capability, err := simulationCapabilityForJob(&fakeExecutor{}, job)
	if err != nil {
		t.Fatal(err)
	}
	plan[0] = '!'
	if capability == nil || capability.Role != execution.SimulationRoleCoordinator || string(capability.ExplorationPlan) != `{"schema":"gomad3.simulation-exploration-plan/v1"}` || capability.ExplorationRecordLimit != 1<<20 || capability.ExplorationRecordCount != 1 {
		t.Fatalf("simulation exploration capability = %#v", capability)
	}
	if _, err := simulationCapabilityForJob(&fakeExecutor{}, runJob{simulationPlan: "plan"}); err == nil {
		t.Fatal("simulationCapabilityForJob() accepted missing record bounds")
	}
}

func TestRunChoiceExplorationExecutesRootAndEveryNonSelectedRank(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	executor := &explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit}
	config := testConfig(t, preparer, executor, "7", PolicyAll, 2)
	config.Strategy = StrategyChoiceExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 8
	config.MaxChoiceDepth = 4
	config.MaxExplorationBytes = 1 << 20

	summary, err := Explore(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Attempted != 2 || summary.Succeeded != 2 || summary.StopReason != StopExplorationExhausted || summary.ChoiceExploration == nil || summary.ChoiceExploration.CommittedRounds != 2 || summary.ChoiceExploration.SeenPrefixes != 2 {
		t.Fatalf("exploration summary = %#v", summary)
	}
	executor.mu.Lock()
	requests := append([]execution.Spec(nil), executor.requests...)
	executor.mu.Unlock()
	if len(requests) != 2 || requests[0].Choice == nil || requests[0].Choice.Mode != choice.ModeRecord || requests[1].Choice == nil || requests[1].Choice.Mode != choice.ModePrefix || requests[1].Choice.ReplayPlan == nil || requests[1].Choice.ReplayPlan.Decisions[0].Selected != 1 {
		t.Fatalf("exploration requests = %#v", requests)
	}
	batch, err := campaign.OpenCampaign(summary.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if batch.Record.Strategy != string(StrategyChoiceExploration) || batch.Record.ChoiceExploration == nil || len(batch.Executions) != 2 || batch.Executions[1].ParentCandidateSHA256 == "" || batch.Executions[1].ForcedDepth == nil || *batch.Executions[1].ForcedDepth != 1 {
		t.Fatalf("exploration batch = %#v", batch)
	}
	segmentPath := filepath.Join(summary.CampaignPath, "choice-exploration", "rounds", "00000000000000000001", "segment.json")
	if err := os.WriteFile(segmentPath, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := campaign.OpenCampaign(summary.CampaignPath); err == nil {
		t.Fatal("OpenBatch() accepted a corrupt exploration segment")
	}
}

func TestRunSimulationExplorationExecutesRootAndEveryScenarioRank(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	executor := &simulationExplorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit}
	config := testConfig(t, preparer, executor, "7", PolicyAll, 1)
	config.Strategy = StrategySimulationExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 4
	config.MaxForcedDecisions = 2
	config.MaxExplorationBytes = 1 << 20
	config.MaxExplorationResultBytes = 1 << 20
	config.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 2, Scenario: 2, Network: 2, Storage: 2, Fault: 2, Crash: 2}

	summary, err := Explore(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Attempted != 2 || summary.Succeeded != 2 || summary.StopReason != StopExplorationExhausted || summary.SimulationExploration == nil || summary.SimulationExploration.CommittedRounds != 2 || summary.SimulationExploration.SeenCandidates != 2 {
		t.Fatalf("simulation exploration summary = %#v", summary)
	}
	executor.mu.Lock()
	requests := append([]execution.Spec(nil), executor.requests...)
	executor.mu.Unlock()
	if len(requests) != 2 || requests[0].Simulation == nil || len(requests[0].Simulation.ExplorationPlan) == 0 || requests[1].Simulation == nil || len(requests[1].Simulation.ExplorationPlan) == 0 || requests[0].Choice == nil || requests[0].Choice.Mode != choice.ModeRecord || requests[1].Choice == nil || requests[1].Choice.Mode != choice.ModeRecord {
		t.Fatalf("simulation exploration requests = %#v", requests)
	}
	batch, err := campaign.OpenCampaign(summary.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if batch.Record.Strategy != string(StrategySimulationExploration) || batch.Record.SimulationExploration == nil || len(batch.Executions) != 2 || batch.Executions[1].ParentCandidateSHA256 == "" || batch.Executions[1].ForcedDepth == nil || *batch.Executions[1].ForcedDepth != 1 {
		t.Fatalf("simulation exploration batch = %#v", batch)
	}
}

func TestRunSimulationExplorationRetainsExactDeduplicatedSimulationFailure(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	executor := &simulationExplorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit, fail: true}
	config := testConfig(t, preparer, executor, "7", PolicyAll, 1)
	config.Strategy = StrategySimulationExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 4
	config.MaxForcedDecisions = 2
	config.MaxExplorationBytes = 1 << 20
	config.MaxExplorationResultBytes = 1 << 20
	config.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 2, Scenario: 2, Network: 2, Storage: 2, Fault: 2, Crash: 2}

	summary, err := Explore(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Attempted != 2 || summary.Failures != 2 || summary.DistinctFailures != 1 || len(summary.Artifacts) != 1 {
		t.Fatalf("simulation exploration failure summary = %#v", summary)
	}
	opened, err := artifact.OpenArtifact(summary.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	profile := opened.Manifest.SimulationProfile
	if profile == nil {
		t.Fatal("simulation exploration failure artifact omitted simulation exploration evidence")
	}
	plan, err := artifact.ReadPayload(opened, profile.Plan.File, uint64(profile.Plan.Bytes))
	if err != nil {
		t.Fatal(err)
	}
	record, err := artifact.ReadPayload(opened, profile.Record.File, uint64(profile.Record.Bytes))
	if err != nil {
		t.Fatal(err)
	}
	if err := simulationrecord.ValidateArtifact(*profile, plan, record); err != nil {
		t.Fatal(err)
	}
	batch, err := campaign.OpenCampaign(summary.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Executions) != 2 || batch.Executions[0].Artifact == nil || batch.Executions[1].Artifact == nil || *batch.Executions[0].Artifact != *batch.Executions[1].Artifact {
		t.Fatalf("deduplicated combined failure references = %#v", batch.Executions)
	}
}

func TestRunChoiceExplorationResumeRerunsTheWholeIncompleteRound(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	baseExecutor := &explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit}
	config := testConfig(t, preparer, explorationInterruptExecutor{exploration: baseExecutor}, "7", PolicyAll, 2)
	config.Strategy = StrategyChoiceExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 8
	config.MaxChoiceDepth = 4
	config.MaxExplorationBytes = 1 << 20

	partial, err := Explore(context.Background(), config)
	var hostErr *HostError
	if !errors.As(err, &hostErr) || partial.Attempted != 1 || partial.ChoiceExploration == nil || partial.ChoiceExploration.CommittedRounds != 1 {
		t.Fatalf("partial exploration = %#v, error = %v", partial, err)
	}
	resumedExecutor := &explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit}
	resumed, err := Explore(context.Background(), CampaignSpec{
		ResumeCampaign: partial.CampaignPath, RunnerBuild: config.RunnerBuild, SupervisorCommand: []string{"unused"}, Executor: resumedExecutor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Attempted != 2 || resumed.Succeeded != 2 || resumed.RecoveryExecutions != 1 || resumed.ChoiceExploration == nil || resumed.ChoiceExploration.CommittedRounds != 2 || resumed.StopReason != StopExplorationExhausted {
		t.Fatalf("resumed exploration = %#v", resumed)
	}
	batch, err := campaign.OpenCampaign(resumed.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Executions) != 2 || batch.Record.RecoveryExecutions != 1 {
		t.Fatalf("resumed exploration batch = %#v", batch)
	}
}

func TestRunSimulationExplorationResumePreservesCommittedCandidates(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	baseExecutor := &simulationExplorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit}
	config := testConfig(t, preparer, simulationExplorationInterruptExecutor{exploration: baseExecutor}, "7", PolicyAll, 1)
	config.Strategy = StrategySimulationExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 4
	config.MaxForcedDecisions = 2
	config.MaxExplorationBytes = 1 << 20
	config.MaxExplorationResultBytes = 1 << 20
	config.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 2, Scenario: 2, Network: 2, Storage: 2, Fault: 2, Crash: 2}

	partial, err := Explore(context.Background(), config)
	var hostErr *HostError
	if !errors.As(err, &hostErr) || partial.Attempted != 1 || partial.SimulationExploration == nil || partial.SimulationExploration.CommittedRounds != 1 {
		t.Fatalf("partial simulation exploration = %#v, error = %v", partial, err)
	}
	resumedExecutor := &simulationExplorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit}
	resumed, err := Explore(context.Background(), CampaignSpec{
		ResumeCampaign: partial.CampaignPath, RunnerBuild: config.RunnerBuild, SupervisorCommand: []string{"unused"}, Executor: resumedExecutor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Attempted != 2 || resumed.Succeeded != 2 || resumed.RecoveryExecutions != 1 || resumed.SimulationExploration == nil || resumed.SimulationExploration.CommittedRounds != 2 || resumed.StopReason != StopExplorationExhausted {
		t.Fatalf("resumed simulation exploration = %#v", resumed)
	}
	resumedExecutor.mu.Lock()
	resumedRequests := len(resumedExecutor.requests)
	resumedExecutor.mu.Unlock()
	if resumedRequests != 1 {
		t.Fatalf("resumed simulation exploration executed %d candidates, want only the uncommitted candidate", resumedRequests)
	}
}

func TestRunChoiceExplorationExpandsCompleteTargetFailures(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	executor := &explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit, exitCode: 2}
	config := testConfig(t, preparer, executor, "7", PolicyAll, 2)
	config.Strategy = StrategyChoiceExploration
	config.ChoiceTraceLimit = limit
	config.MaxExecutions = 8
	config.MaxChoiceDepth = 4
	config.MaxExplorationBytes = 1 << 20

	summary, err := Explore(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Attempted != 2 || summary.Failures != 2 || summary.Succeeded != 0 || summary.StopReason != StopExplorationExhausted || len(summary.Artifacts) != 2 {
		t.Fatalf("exploration target failures = %#v", summary)
	}
}

func TestRunChoiceExplorationPinnedOutcomeEfficiencyMatchesEqualBudgetSeedSampling(t *testing.T) {
	toolchainRoot, err := filepath.Abs(filepath.Join("..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	testdata, err := filepath.Abs(filepath.Join("..", "internal", "gomadtool", "conformance", "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	supervisor := filepath.Join(t.TempDir(), "gomad")
	command := exec.Command(filepath.Join(toolchainRoot, "bin", "go"), "build", "-trimpath", "-o", supervisor, "./cmd/gomad")
	command.Dir = filepath.Join(testdata, "..", "..", "..", "..")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build pinned supervisor: %v: %s", err, output)
	}
	config := CampaignSpec{
		Parallel:         2,
		ExecutionTimeout: 10 * time.Second, OverallTimeout: time.Minute, TerminateGrace: time.Second,
		OnFailure: PolicyAll, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20,
		KeepSuccesses: KeepSuccessesAll, SuccessArtifactLimit: 16, SuccessBytesLimit: 128 << 20,
		Target: target.Spec{
			Kind: target.KindGoRun, Source: "./choice_exploration", WorkingDir: testdata, ToolchainRoot: toolchainRoot,
		},
		SupervisorCommand: []string{supervisor, "__supervisor"}, RunnerBuild: "sha256:" + strings.Repeat("0", 64),
	}
	seedConfig := config
	seedConfig.Strategy = StrategySeed
	seedConfig.Seeds = "211-226"
	seedConfig.Artifacts = t.TempDir()
	seedSummary, err := Explore(context.Background(), seedConfig)
	if err != nil {
		t.Fatal(err)
	}
	explorationConfig := config
	explorationConfig.Strategy = StrategyChoiceExploration
	explorationConfig.Seeds = "211"
	explorationConfig.ChoiceTraceLimit = 1 << 20
	explorationConfig.MaxExecutions = 16
	explorationConfig.MaxChoiceDepth = 32
	explorationConfig.MaxExplorationBytes = 4 << 20
	explorationConfig.Artifacts = t.TempDir()
	explorationSummary, err := Explore(context.Background(), explorationConfig)
	if err != nil {
		t.Fatal(err)
	}
	seedOutcomes := distinctSuccessStdoutOutcomes(t, seedSummary.SuccessArtifacts)
	explorationOutcomes := distinctSuccessStdoutOutcomes(t, explorationSummary.SuccessArtifacts)
	if seedSummary.Attempted != 16 || len(seedOutcomes) != 2 {
		t.Fatalf("equal-budget seed summary = %#v, outcomes = %v", seedSummary, seedOutcomes)
	}
	if explorationSummary.Attempted != 16 || len(explorationOutcomes) != 2 || explorationSummary.ChoiceExploration == nil || explorationSummary.ChoiceExploration.DeduplicatedOutcomes != 2 {
		t.Fatalf("equal-budget exploration summary = %#v, outcomes = %v", explorationSummary, explorationOutcomes)
	}
	if uint64(len(explorationOutcomes))*seedSummary.Attempted != uint64(len(seedOutcomes))*explorationSummary.Attempted {
		t.Fatalf("pinned outcomes per execution differ: seed=%d/%d exploration=%d/%d", len(seedOutcomes), seedSummary.Attempted, len(explorationOutcomes), explorationSummary.Attempted)
	}
}

func distinctSuccessStdoutOutcomes(t *testing.T, paths []string) map[record.SHA256]struct{} {
	t.Helper()
	outcomes := make(map[record.SHA256]struct{}, len(paths))
	for _, path := range paths {
		artifact, err := artifact.OpenArtifact(path)
		if err != nil {
			t.Fatal(err)
		}
		outcomes[artifact.Manifest.Streams.Stdout.FullSHA256] = struct{}{}
		if err := artifact.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return outcomes
}

func TestRunClassifiesInvalidChoiceTraceTerminalEvidence(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		reason string
	}{
		{name: "malformed", err: execution.ErrChoiceTraceMalformed, reason: "choice_trace_malformed"},
		{name: "unterminated", err: execution.ErrChoiceTraceUnterminated, reason: "choice_trace_unterminated"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := testConfig(t, newFakePreparer(t), terminalErrorExecutor{err: test.err}, "1", PolicyAll, 1)
			config.ChoiceTraceLimit = execution.MinimumChoiceTraceBytes
			_, err := Explore(context.Background(), config)
			var hostError *HostError
			if !errors.As(err, &hostError) || hostError.Reason != test.reason {
				t.Fatalf("Explore() error = %v", err)
			}
		})
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
	result.ChoiceTrace = execution.ChoiceTrace{
		Profile: choice.Profile, ImplementationSHA256: implementation, Limit: limit,
		Trace: trace,
	}
	config := testConfig(t, preparer, terminalErrorExecutor{result: result, err: execution.ErrChoiceTraceOverflow}, "1", PolicyAll, 1)
	config.ChoiceTraceLimit = limit
	summary, err := Explore(context.Background(), config)
	var hostErr *HostError
	if !errors.As(err, &hostErr) || hostErr.Reason != "choice_trace_overflow" {
		t.Fatalf("Explore() error = %#v", err)
	}
	if len(summary.Artifacts) != 1 || summary.ChoiceTrace == nil || summary.ChoiceTrace.TerminalState != "overflow" {
		t.Fatalf("Explore() summary = %#v", summary)
	}
	opened, err := artifact.OpenArtifact(summary.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
	if opened.Manifest.ArtifactKind != record.ArtifactRunnerFailure || opened.Manifest.Outcome.Reason != "choice_trace_overflow" || opened.Manifest.ChoiceProfile == nil || opened.Manifest.ChoiceProfile.Trace.TerminalState != "overflow" {
		t.Fatalf("overflow manifest = %#v", opened.Manifest)
	}

	resumedExecutor := &fakeExecutor{result: func(uint64) execution.Result {
		result := processResult(0, "", "")
		result.ChoiceTrace = completeChoiceTrace(t, preparer.prepared.BuildKey, limit, nil)
		return result
	}}
	resumed, err := Explore(context.Background(), CampaignSpec{
		ResumeCampaign: summary.CampaignPath, RunnerBuild: config.RunnerBuild, SupervisorCommand: []string{"unused"}, Executor: resumedExecutor,
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
		"choice profile":  {"GOMAD3_CHOICE_PROFILE=injected"},
		"choice trace fd": {"GOMAD3_CHOICE_TRACE_FD=9"},
		"duplicate":       {"A=1", "A=2"},
		"invalid":         {"NOT-VALID=1"},
		"nul":             {"A=value\x00tail"},
	} {
		t.Run(name, func(t *testing.T) {
			config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
			config.Environment = environment
			if _, err := Explore(context.Background(), config); err == nil {
				t.Fatal("Explore() succeeded")
			}
		})
	}
}

func TestRunCancellationIsAHostFailure(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), blockingExecutor{}, "1", PolicyAll, 1)
	ctx := cancelOnProgress(t, &config, func(progress CampaignEvent) bool { return progress.Running == 1 })
	config.TerminateGrace = 10 * time.Millisecond
	summary, err := Explore(ctx, config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "cancelled" || !errors.Is(err, context.Canceled) {
		t.Fatalf("Explore() error = %#v", err)
	}
	if summary.Failures != 0 || len(summary.Artifacts) != 0 {
		t.Fatalf("cancelled summary = %#v", summary)
	}
	plan, planErr := campaign.ReadResumePlan(summary.CampaignPath)
	if planErr != nil {
		t.Fatal(planErr)
	}
	if plan.Selection != "1" || plan.RunnerBuild != config.RunnerBuild || plan.Prepared.Target.SHA256 == "" {
		t.Fatalf("resume plan = %#v", plan)
	}
	partials, readErr := os.ReadDir(filepath.Join(summary.CampaignPath, ".partial"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(partials) != 3 {
		t.Fatalf("cancelled partials = %v, want campaign, executions, and target", partials)
	}
	if _, err := os.Stat(filepath.Join(summary.CampaignPath, ".partial", "campaign", "partial.json")); err != nil {
		t.Fatal(err)
	}
}

func TestRunResumesVerifiedBatchAndSkipsCompletedOrdinals(t *testing.T) {
	preparer := newFakePreparer(t)
	interrupted := &resumeInterruptExecutor{}
	config := testConfig(t, preparer, interrupted, "7-9", PolicyAll, 1)
	ctx := cancelOnProgress(t, &config, func(progress CampaignEvent) bool { return progress.Succeeded == 1 })
	config.TerminateGrace = 10 * time.Millisecond
	partial, err := Explore(ctx, config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "cancelled" {
		t.Fatalf("interrupted Explore() error = %v", err)
	}
	if seeds := interrupted.seeds(); !slices.Equal(seeds, []uint64{7, 8}) {
		t.Fatalf("interrupted seeds = %v", seeds)
	}

	resumedExecutor := &fakeExecutor{}
	resumed, err := Explore(context.Background(), CampaignSpec{
		ResumeCampaign: partial.CampaignPath, RunnerBuild: config.RunnerBuild, SupervisorCommand: []string{"unused"}, Executor: resumedExecutor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if seeds := executorSeeds(resumedExecutor); !slices.Equal(seeds, []uint64{8, 9}) {
		t.Fatalf("resumed seeds = %v", seeds)
	}
	if resumed.CampaignPath != partial.CampaignPath || resumed.SelectionCount != 3 || resumed.Attempted != 3 || resumed.Succeeded != 3 || resumed.Failures != 0 || resumed.StopReason != StopSeedsExhausted || preparer.calls != 1 {
		t.Fatalf("resumed summary = %#v, preparation calls = %d", resumed, preparer.calls)
	}
	batch, err := campaign.OpenCampaign(resumed.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Executions) != 3 || batch.Executions[0].Seed != 7 || batch.Executions[1].Seed != 8 || batch.Executions[2].Seed != 9 {
		t.Fatalf("batch runs = %#v", batch.Executions)
	}
}

func TestRunResumeRestoresSeenChoiceFeaturesBeforeNovelRetention(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	trace := completeChoiceTrace(t, preparer.prepared.BuildKey, limit, []choice.Record{{
		Ordinal: 0, Kind: choice.KindRunnable, Flags: choice.FlagDecision, SiteOffset: 24, Alternatives: 2, Selected: 0,
	}})
	interrupted := &choiceResumeInterruptExecutor{trace: trace}
	config := testConfig(t, preparer, interrupted, "7-8", PolicyAll, 1)
	ctx := cancelOnProgress(t, &config, func(progress CampaignEvent) bool { return progress.RetainedSuccesses == 1 })
	config.TerminateGrace = 10 * time.Millisecond
	config.Coverage = CoverageChoice
	config.ChoiceTraceLimit = limit
	config.KeepSuccesses = KeepSuccessesNovel
	config.SuccessArtifactLimit = 2
	config.SuccessBytesLimit = 64 << 20
	partial, err := Explore(ctx, config)
	if err == nil || partial.RetainedSuccesses != 1 {
		t.Fatalf("interrupted summary = %#v, error = %v", partial, err)
	}

	resumedExecutor := &fakeExecutor{result: func(uint64) execution.Result {
		result := processResult(0, "", "")
		result.IOTranscript = completeEmptyTranscript()
		result.ChoiceTrace = trace
		return result
	}}
	resumed, err := Explore(context.Background(), CampaignSpec{
		ResumeCampaign: partial.CampaignPath, RunnerBuild: config.RunnerBuild, SupervisorCommand: []string{"unused"}, Executor: resumedExecutor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resumed.RetainedSuccesses != 1 || len(resumed.SuccessArtifacts) != 1 {
		t.Fatalf("resumed summary = %#v", resumed)
	}
	batch, err := campaign.OpenCampaign(resumed.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if batch.Executions[1].SuccessArtifact != nil || len(batch.Executions[1].ChoiceFeatures) == 0 {
		t.Fatalf("resumed run = %#v", batch.Executions[1])
	}
}

func TestRunResumesGuidedBatchWithoutReselectingSeeds(t *testing.T) {
	corpus := filepath.Join(t.TempDir(), "corpus")
	replayer := &matchingReplayer{}
	interrupted := &resumeInterruptExecutor{}
	config := testConfig(t, newFakePreparer(t), interrupted, "7-8", PolicyAll, 1)
	ctx := cancelOnProgress(t, &config, func(progress CampaignEvent) bool { return progress.CorpusEntries == 1 })
	config.TerminateGrace = 10 * time.Millisecond
	config.Coverage = CoverageSemantic
	config.Guide = true
	config.Corpus = corpus
	config.Replayer = replayer
	partial, err := Explore(ctx, config)
	if err == nil || partial.CorpusEntries != 1 {
		t.Fatalf("interrupted guided summary = %#v, error = %v", partial, err)
	}

	resumedExecutor := &fakeExecutor{result: func(uint64) execution.Result {
		result := processResult(0, "", "")
		result.IOTranscript = completeEmptyTranscript()
		return result
	}}
	resumed, err := Explore(context.Background(), CampaignSpec{
		ResumeCampaign: partial.CampaignPath, RunnerBuild: config.RunnerBuild, SupervisorCommand: []string{"unused"}, Executor: resumedExecutor, Replayer: replayer,
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
	ctx := cancelOnProgress(t, &config, func(progress CampaignEvent) bool { return progress.Running == 1 })
	config.TerminateGrace = 10 * time.Millisecond
	partial, err := Explore(ctx, config)
	if err == nil {
		t.Fatal("Explore() did not leave an interrupted batch")
	}
	_, err = Explore(context.Background(), CampaignSpec{
		ResumeCampaign: partial.CampaignPath, RunnerBuild: "sha256:changed", SupervisorCommand: []string{"unused"}, Executor: &fakeExecutor{},
	})
	if err == nil || !strings.Contains(err.Error(), "Runner build identity") {
		t.Fatalf("resume error = %v", err)
	}
}

func TestRunResumeRejectsTamperedRetainedSuccessArtifact(t *testing.T) {
	interrupted := &resumeInterruptExecutor{}
	config := testConfig(t, newFakePreparer(t), interrupted, "7-8", PolicyAll, 1)
	ctx := cancelOnProgress(t, &config, func(progress CampaignEvent) bool { return progress.RetainedSuccesses == 1 })
	config.TerminateGrace = 10 * time.Millisecond
	config.KeepSuccesses = KeepSuccessesAll
	config.SuccessArtifactLimit = 2
	config.SuccessBytesLimit = 64 << 20
	partial, err := Explore(ctx, config)
	if err == nil || len(partial.SuccessArtifacts) != 1 {
		t.Fatalf("interrupted summary = %#v, error = %v", partial, err)
	}
	if err := os.WriteFile(filepath.Join(partial.SuccessArtifacts[0], "stdout"), []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = Explore(context.Background(), CampaignSpec{
		ResumeCampaign: partial.CampaignPath, RunnerBuild: config.RunnerBuild, SupervisorCommand: []string{"unused"}, Executor: &fakeExecutor{},
	})
	if err == nil || !strings.Contains(err.Error(), "retained success") {
		t.Fatalf("resume error = %v", err)
	}
}

func TestRunPreparationFailureLeavesExplicitPartial(t *testing.T) {
	config := testConfig(t, errorPreparer{err: errors.New("build failed")}, &fakeExecutor{}, "1", PolicyAll, 1)
	summary, err := Explore(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "target_preparation" {
		t.Fatalf("Explore() error = %#v", err)
	}
	partial, readErr := os.ReadFile(filepath.Join(summary.CampaignPath, ".partial", "preparation", "partial.json"))
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
	summary, err := Explore(ctx, config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "cancelled" || !errors.Is(err, context.Canceled) {
		t.Fatalf("Explore() error = %#v", err)
	}
	partial, readErr := os.ReadFile(filepath.Join(summary.CampaignPath, ".partial", "preparation", "partial.json"))
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
	summary, err := Explore(deadline, config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "overall_timeout" || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Explore() error = %#v", err)
	}
	partial, readErr := os.ReadFile(filepath.Join(summary.CampaignPath, ".partial", "preparation", "partial.json"))
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
			outcome := execution.Classify(processResult(2, "", stderr), false, record.WorldTerminal{Kind: "none"})
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
	}, execution.Classification{Domain: "target", Reason: "nonzero_exit", Termination: "exit", ArtifactKind: record.ArtifactTargetFailure, ReplayMode: record.ReplayExact}, "run", record.World{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.IOProfile.Name != "gomad3-deterministic/v1" || manifest.IOProfile.Inventory == "" || manifest.IOProfile.InventorySHA256 == "" || manifest.IOProfile.ImplementationSHA256 == "" {
		t.Fatalf("manifest I/O profile = %#v", manifest.IOProfile)
	}
}

func TestClassifyStructuredWorldFailures(t *testing.T) {
	for kind, reason := range map[string]string{
		"deadlock": "world_deadlock", "capacity": "world_capacity", "replay-divergence": "world_replay_divergence", "invalid-input": "world_invalid_input",
	} {
		t.Run(kind, func(t *testing.T) {
			outcome := execution.Classify(processResult(0, "", ""), false, record.WorldTerminal{Kind: kind, Detail: "detail"})
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
	_, err := Explore(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "overall_timeout" {
		t.Fatalf("Explore() error = %v", err)
	}
	if elapsed := time.Since(started); elapsed > 350*time.Millisecond {
		t.Fatalf("Explore() elapsed = %v", elapsed)
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
			t.Setenv("GOMAD3_COORDINATOR_EXIT_MARKER", marker)
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
			_, err := Explore(ctx, config)
			var hostError *HostError
			if !errors.As(err, &hostError) || hostError.Reason != test.wantReason {
				t.Fatalf("Explore() error = %v", err)
			}
		})
	}
}

func TestIsolatedRunnerPreservesUnsupportedTargetError(t *testing.T) {
	config := testConfig(t, nil, nil, "1", PolicyFirst, 1)
	config.CoordinatorCommand = []string{os.Args[0], "-test.run=TestUnsupportedTargetCoordinatorHelper"}
	_, err := Explore(context.Background(), config)
	var unsupported *target.UnsupportedCapabilityError
	if !errors.As(err, &unsupported) || unsupported.ImportPath != "example.com/target" || unsupported.Capability != "imports os/exec" {
		t.Fatalf("Explore() error = %v", err)
	}
}

func TestIsolatedRunnerPreservesMissingSemanticProbesError(t *testing.T) {
	config := testConfig(t, nil, nil, "1", PolicyFirst, 1)
	config.CoordinatorCommand = []string{os.Args[0], "-test.run=TestMissingSemanticProbesCoordinatorHelper"}
	_, err := Explore(context.Background(), config)
	var missing *deterministicio.MissingSemanticProbesError
	if !errors.As(err, &missing) || len(missing.Probes) != 1 || missing.Probes[0] != "stdlib.os.openfile" {
		t.Fatalf("Explore() error = %v", err)
	}
}

func TestIsolatedRunnerPreservesBoundedExecutionEvidence(t *testing.T) {
	config := testConfig(t, nil, nil, "1", PolicyAll, 1)
	config.Coverage = CoverageSemantic
	config.CollectExecutionEvidence = true
	config.CoordinatorCommand = []string{os.Args[0], "-test.run=TestExecutionEvidenceCoordinatorHelper"}
	summary, err := Explore(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.ExecutionEvidence == nil || summary.ExecutionEvidence.Schema != ExecutionEvidenceSchema || summary.ExecutionEvidence.Seed != 1 || summary.ExecutionEvidence.Target.SHA256 != "sha256:target" {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestIsolatedRunnerTransportsChoiceTraceConfiguration(t *testing.T) {
	config := testConfig(t, nil, nil, "1", PolicyAll, 1)
	config.ChoiceTraceLimit = execution.MinimumChoiceTraceBytes
	config.CoordinatorCommand = []string{os.Args[0], "-test.run=TestChoiceTraceCoordinatorHelper"}
	summary, err := Explore(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.ChoiceTrace == nil || summary.ChoiceTrace.Limit != execution.MinimumChoiceTraceBytes {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestIsolatedRunnerDrainsFastCoordinatorBeforeWaitClosesOutput(t *testing.T) {
	config := testConfig(t, nil, nil, "1", PolicyAll, 1)
	config.CoordinatorCommand = []string{os.Args[0], "-test.run=TestFastCoordinatorHelper"}
	config.Progress = func(CampaignEvent) error {
		deadline := time.Now().Add(100 * time.Millisecond)
		for time.Now().Before(deadline) {
			runtime.Gosched()
		}
		return nil
	}
	summary, err := Explore(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Attempted != 1 || summary.Succeeded != 1 {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestFastCoordinatorHelper(t *testing.T) {
	if os.Getenv("GOMAD3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	encoder := json.NewEncoder(os.Stdout)
	progress := CampaignEvent{Phase: ProgressRunning, Attempted: 1, Running: 1}
	if err := encoder.Encode(coordinatorMessage{Type: "progress", Progress: &progress}); err != nil {
		t.Fatal(err)
	}
	response := coordinatorResponse{CampaignResult: CampaignResult{Attempted: 1, Succeeded: 1}}
	if err := encoder.Encode(coordinatorMessage{Type: "result", Response: &response}); err != nil {
		t.Fatal(err)
	}
	os.Exit(0) //nolint:revive // This subprocess helper must exit before the parent test harness continues.
}

func TestUnsupportedTargetCoordinatorHelper(t *testing.T) {
	if os.Getenv("GOMAD3_RUNNER_COORDINATOR") != "1" {
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
	if os.Getenv("GOMAD3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	missing := &deterministicio.MissingSemanticProbesError{Probes: []string{"stdlib.os.openfile"}}
	response := coordinatorResponse{
		ErrorReason: "semantic_coverage", ErrorDetail: missing.Error(), MissingSemanticProbes: missing.Probes,
	}
	if err := json.NewEncoder(os.Stdout).Encode(coordinatorMessage{Type: "result", Response: &response}); err != nil {
		t.Fatal(err)
	}
	os.Exit(0)
}

func TestExecutionEvidenceCoordinatorHelper(t *testing.T) {
	if os.Getenv("GOMAD3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	response := coordinatorResponse{CampaignResult: CampaignResult{ExecutionEvidence: &ExecutionEvidence{
		Schema: ExecutionEvidenceSchema, Seed: 1, Target: record.Target{SHA256: "sha256:target"},
	}}}
	if err := json.NewEncoder(os.Stdout).Encode(coordinatorMessage{Type: "result", Response: &response}); err != nil {
		t.Fatal(err)
	}
	os.Exit(0)
}

func TestChoiceTraceCoordinatorHelper(t *testing.T) {
	if os.Getenv("GOMAD3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	var wire coordinatorConfig
	if err := json.NewDecoder(os.Stdin).Decode(&wire); err != nil {
		t.Fatal(err)
	}
	response := coordinatorResponse{CampaignResult: CampaignResult{ChoiceTrace: &ChoiceTraceSummary{Limit: wire.ChoiceTraceLimit}}}
	if err := json.NewEncoder(os.Stdout).Encode(coordinatorMessage{Type: "result", Response: &response}); err != nil {
		t.Fatal(err)
	}
	os.Exit(0)
}

func TestBlockingCoordinatorHelper(t *testing.T) {
	if os.Getenv("GOMAD3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	for {
		runtime.Gosched()
	}
}

func TestExitedCoordinatorWithOpenStdoutHelper(t *testing.T) {
	if os.Getenv("GOMAD3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	command := exec.Command(os.Args[0], "-test.run=^TestCoordinatorStdoutDescendantHelper$")
	command.Env = append(os.Environ(), "GOMAD3_COORDINATOR_STDOUT_DESCENDANT=1")
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("GOMAD3_COORDINATOR_EXIT_MARKER"), []byte("exiting"), 0o600); err != nil {
		t.Fatal(err)
	}
	os.Exit(0) //nolint:revive // This subprocess helper must exit while its descendant retains stdout.
}

func TestCoordinatorStdoutDescendantHelper(t *testing.T) {
	if os.Getenv("GOMAD3_COORDINATOR_STDOUT_DESCENDANT") != "1" {
		t.Skip("coordinator stdout descendant subprocess only")
	}
	<-time.After(10 * time.Second)
}

func TestIsolatedRunnerBoundsCoordinatorOutput(t *testing.T) {
	config := testConfig(t, nil, nil, "1", PolicyFirst, 1)
	config.CoordinatorCommand = []string{os.Args[0], "-test.run=TestOversizedCoordinatorHelper"}
	_, err := Explore(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "coordinator_decode" {
		t.Fatalf("Explore() error = %v", err)
	}
}

func TestOversizedCoordinatorHelper(t *testing.T) {
	if os.Getenv("GOMAD3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	if _, err := os.Stdout.Write(make([]byte, maximumCoordinatorMessageBytes+1)); err != nil {
		t.Fatal(err)
	}
}

func TestIsolatedRunnerRemovesCoordinatorProcessGroup(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "descendant-survived")
	t.Setenv("GOMAD3_COORDINATOR_DESCENDANT_MARKER", marker)
	config := testConfig(t, nil, nil, "1", PolicyFirst, 1)
	config.OverallTimeout = 250 * time.Millisecond
	config.CoordinatorCommand = []string{os.Args[0], "-test.run=TestCoordinatorTreeHelper"}
	_, err := Explore(context.Background(), config)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "overall_timeout" {
		t.Fatalf("Explore() error = %v", err)
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
	if os.Getenv("GOMAD3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	command := exec.Command(os.Args[0], "-test.run=TestCoordinatorDescendantHelper")
	command.Env = append(os.Environ(), "GOMAD3_COORDINATOR_DESCENDANT=1")
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
	if os.Getenv("GOMAD3_COORDINATOR_DESCENDANT") != "1" {
		t.Skip("coordinator descendant subprocess only")
	}
	signal.Ignore(syscall.SIGTERM)
	<-time.After(350 * time.Millisecond)
	if err := os.WriteFile(os.Getenv("GOMAD3_COORDINATOR_DESCENDANT_MARKER"), []byte("survived"), 0o600); err != nil {
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
		Argv: []string{"gomad3-target"}, BuildTags: []string{}, Compatibility: []record.CompatibilityPack{}, BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"},
		GoVersion: "go1.26.4", BuildKey: "cbeccfefbc62a2ca026d9dded0316ecedfce33bd46b5c71b6645e86b67a0713e",
		TargetGOOS: "darwin", TargetGOARCH: "arm64",
	}}
}

func profileFakePreparer(t *testing.T, argument string) *fakePreparer {
	t.Helper()
	preparer := newFakePreparer(t)
	preparer.prepared.Kind = target.KindGoTest
	preparer.prepared.Source = "./pkg"
	preparer.prepared.Argv = []string{"gomad3-target", argument}
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
	requests      []execution.Spec
	result        func(uint64) execution.Result
}

type explorationExecutor struct {
	t        *testing.T
	buildKey string
	limit    uint64
	exitCode int
	mu       sync.Mutex
	requests []execution.Spec
}

type simulationExplorationExecutor struct {
	t        *testing.T
	buildKey string
	limit    uint64
	fail     bool
	mu       sync.Mutex
	requests []execution.Spec
}

type explorationInterruptExecutor struct {
	exploration *explorationExecutor
}

type simulationExplorationInterruptExecutor struct {
	exploration *simulationExplorationExecutor
}

func (executor explorationInterruptExecutor) Run(ctx context.Context, request execution.Spec) (execution.Result, error) {
	if request.Choice != nil && request.Choice.Mode == choice.ModePrefix {
		return execution.Result{}, errors.New("simulated exploration interruption")
	}
	return executor.exploration.Run(ctx, request)
}

func (executor simulationExplorationInterruptExecutor) Run(ctx context.Context, request execution.Spec) (execution.Result, error) {
	var plan struct {
		Overrides []json.RawMessage `json:"overrides"`
	}
	if request.Simulation == nil || json.Unmarshal(request.Simulation.ExplorationPlan, &plan) != nil {
		return execution.Result{}, errors.New("simulation exploration simulation plan is unavailable")
	}
	if len(plan.Overrides) != 0 {
		return execution.Result{}, errors.New("simulated simulation exploration interruption")
	}
	return executor.exploration.Run(ctx, request)
}

func (executor *explorationExecutor) Run(_ context.Context, request execution.Spec) (execution.Result, error) {
	executor.mu.Lock()
	executor.requests = append(executor.requests, request)
	executor.mu.Unlock()
	selected := uint32(0)
	if request.Choice != nil && request.Choice.Mode == choice.ModePrefix {
		selected = request.Choice.ReplayPlan.Decisions[len(request.Choice.ReplayPlan.Decisions)-1].Selected
	}
	result := processResult(executor.exitCode, "", "")
	result.ChoiceTrace = completeChoiceTrace(executor.t, executor.buildKey, executor.limit, []choice.Record{{
		Ordinal: 0, Kind: choice.KindRunnable, Flags: choice.FlagDecision, Alternatives: 2, Selected: selected,
	}})
	return result, nil
}

func (executor *simulationExplorationExecutor) Run(_ context.Context, request execution.Spec) (execution.Result, error) {
	executor.mu.Lock()
	executor.requests = append(executor.requests, request)
	executor.mu.Unlock()
	if request.Simulation == nil || len(request.Simulation.ExplorationPlan) == 0 {
		return execution.Result{}, errors.New("simulation exploration simulation plan is unavailable")
	}
	var plan struct {
		BaseSeed  uint64 `json:"base_seed"`
		Overrides []struct {
			Dimension simulationengine.Dimension `json:"dimension"`
			Selected  uint32                     `json:"selected"`
		} `json:"overrides"`
	}
	if err := json.Unmarshal(request.Simulation.ExplorationPlan, &plan); err != nil {
		return execution.Result{}, err
	}
	selected := uint32(0)
	for _, override := range plan.Overrides {
		if override.Dimension == simulationengine.DimensionScenario {
			selected = override.Selected
		}
	}
	decision, err := simulationengine.CanonicalDecision(
		simulationengine.DimensionScenario, 0, record.HashBytes([]byte("route")),
		[]record.SHA256{record.HashBytes([]byte("alpha")), record.HashBytes([]byte("beta"))}, selected,
	)
	if err != nil {
		return execution.Result{}, err
	}
	outcome := "completed"
	if executor.fail {
		outcome = "oracle_failed"
	}
	record, err := json.Marshal(struct {
		Schema               string                      `json:"schema"`
		Seed                 uint64                      `json:"seed"`
		SpecSHA256           record.SHA256               `json:"spec_sha256"`
		Outcome              string                      `json:"outcome"`
		FailureIdentity      record.SHA256               `json:"failure_identity,omitempty"`
		ExplorationPlan      json.RawMessage             `json:"exploration_plan"`
		ExplorationDecisions []simulationengine.Decision `json:"exploration_decisions"`
		ScenarioTape         []string                    `json:"scenario_tape"`
		Identity             record.SHA256               `json:"identity"`
	}{
		Schema: "gomad3.cluster-record/v7", Seed: plan.BaseSeed, SpecSHA256: record.HashBytes([]byte("spec")), Outcome: outcome,
		FailureIdentity: record.HashBytes([]byte("normalized oracle failure")),
		ExplorationPlan: request.Simulation.ExplorationPlan, ExplorationDecisions: []simulationengine.Decision{decision},
		ScenarioTape: []string{[]string{"alpha", "beta"}[selected]}, Identity: record.HashBytes([]byte(fmt.Sprintf("record-%d", selected))),
	})
	if err != nil {
		return execution.Result{}, err
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
	result execution.Result
	err    error
}

func (executor terminalErrorExecutor) Run(context.Context, execution.Spec) (execution.Result, error) {
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

func (replayer *matchingReplayer) Replay(_ context.Context, config ReplaySpec) (ReplayResult, error) {
	replayer.calls++
	opened, err := artifact.OpenArtifact(config.ArtifactPath)
	if err != nil {
		return ReplayResult{}, err
	}
	defer opened.Close()
	return ReplayResult{Artifact: opened.Detached(), Verified: true, Match: true}, nil
}

func newOutOfOrderExecutor(t *testing.T) *outOfOrderExecutor {
	return &outOfOrderExecutor{t: t, later: make(chan struct{})}
}

func (executor *outOfOrderExecutor) Run(ctx context.Context, request execution.Spec) (execution.Result, error) {
	seed := seedFromEnvironment(request.Env)
	if seed == 1 {
		select {
		case <-executor.later:
		case <-ctx.Done():
			return execution.Result{}, ctx.Err()
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

func (executor *fakeExecutor) Run(_ context.Context, request execution.Spec) (execution.Result, error) {
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
	trace execution.ChoiceTrace
}

type progressGatedExecutor struct {
	started chan struct{}
	release chan struct{}
}

type mutatingExecutor struct{}

func (mutatingExecutor) Run(_ context.Context, request execution.Spec) (execution.Result, error) {
	if err := os.Chmod(request.Command, 0o700); err != nil {
		return execution.Result{}, err
	}
	file, err := os.OpenFile(request.Command, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return execution.Result{}, err
	}
	if _, err := file.Write([]byte("mutation")); err != nil {
		file.Close()
		return execution.Result{}, err
	}
	if err := file.Close(); err != nil {
		return execution.Result{}, err
	}
	return processResult(1, "failure", ""), nil
}

func (blockingExecutor) Run(ctx context.Context, _ execution.Spec) (execution.Result, error) {
	<-ctx.Done()
	result := processResult(0, "", "")
	result.Cancelled = true
	result.Termination = execution.TerminationSignal
	result.Signal = "killed"
	return result, nil
}

func (executor *resumeInterruptExecutor) Run(ctx context.Context, request execution.Spec) (execution.Result, error) {
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
	result.Termination = execution.TerminationSignal
	result.Signal = "killed"
	return result, nil
}

func (executor *choiceResumeInterruptExecutor) Run(ctx context.Context, request execution.Spec) (execution.Result, error) {
	if seedFromEnvironment(request.Env) == 7 {
		result := processResult(0, "", "")
		result.IOTranscript = completeEmptyTranscript()
		result.ChoiceTrace = executor.trace
		return result, nil
	}
	<-ctx.Done()
	result := processResult(0, "", "")
	result.Cancelled = true
	result.Termination = execution.TerminationSignal
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

func (executor *progressGatedExecutor) Run(context.Context, execution.Spec) (execution.Result, error) {
	close(executor.started)
	<-executor.release
	return processResult(0, "", ""), nil
}

func newFirstFailureExecutor(want int) *firstFailureExecutor {
	return &firstFailureExecutor{started: make(chan struct{}), want: want}
}

func (executor *firstFailureExecutor) Run(ctx context.Context, request execution.Spec) (execution.Result, error) {
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
	result.Termination = execution.TerminationSignal
	result.Signal = "killed"
	return result, nil
}

func testConfig(t *testing.T, preparer Preparer, executor Executor, seeds string, policy FailurePolicy, parallel int) CampaignSpec {
	t.Helper()
	return CampaignSpec{
		Seeds: seeds, Parallel: parallel, ExecutionTimeout: time.Second, OverallTimeout: 10 * time.Second, TerminateGrace: 100 * time.Millisecond,
		OnFailure: policy, FailureBudget: 1, OutputLimit: 64, WorldTransitionLimit: 64, Artifacts: t.TempDir(),
		Environment: []string{"MODE=test"}, Target: target.Spec{Kind: target.KindGoRun, Source: "."}, SupervisorCommand: []string{"unused"},
		RunnerBuild: "sha256:0000000000000000000000000000000000000000000000000000000000000000", Preparer: preparer, Executor: executor,
	}
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

func processResult(exitCode int, stdout, stderr string) execution.Result {
	return execution.Result{
		Captured: true, Termination: execution.TerminationExit, ExitCode: exitCode, GroupGone: true,
		Stdout: output(stdout), Stderr: output(stderr),
	}
}

func output(value string) hostexec.Output {
	bytes := []byte(value)
	digest := sha256.Sum256(bytes)
	return hostexec.Output{Bytes: bytes, FullSHA256: digest, RetainedSHA256: digest, TotalBytes: uint64(len(bytes)), RetainedBytes: uint64(len(bytes))}
}

func semanticTranscript(t *testing.T, probe string) deterministicio.Transcript {
	t.Helper()
	digest := sha256.Sum256([]byte("gomad3-boundary-probe/v1\x00" + probe))
	var argument [8]byte
	binary.BigEndian.PutUint64(argument[:], binary.BigEndian.Uint64(digest[:8])&(1<<63-1))
	payload, err := deterministicio.EncodeTranscript([]deterministicio.Operation{{
		Name: "boundary.probe", ArgumentHash: deterministicio.HashArgument(argument[:]),
	}})
	if err != nil {
		t.Fatal(err)
	}
	return deterministicio.Transcript{Bytes: payload, SHA256: sha256.Sum256(payload), Records: 1, Complete: true}
}

func completeEmptyTranscript() deterministicio.Transcript {
	return deterministicio.Transcript{Complete: true, SHA256: sha256.Sum256(nil)}
}

func completeChoiceTrace(t *testing.T, buildKey string, limit uint64, records []choice.Record) execution.ChoiceTrace {
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
	return execution.ChoiceTrace{Profile: choice.Profile, ImplementationSHA256: implementation, Limit: limit, Trace: trace}
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
