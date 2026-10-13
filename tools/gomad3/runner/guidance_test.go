package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/artifact"
	backend "go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/inspect"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
)

func TestGuidedCorpusSeedReproducesRetainedExecution(t *testing.T) {
	config := isolatedCampaign(t, conformanceTarget(t, "./environment"))
	config.Guide = true
	config.Corpus = filepath.Join(t.TempDir(), "corpus")
	config.Coverage = CoverageSemantic
	config.KeepSuccesses = KeepSuccessesAll
	config.SuccessArtifactLimit = 1
	config.SuccessBytesLimit = 32 << 20
	first := exploreIsolated(t, config)
	config.Artifacts = t.TempDir()
	config.GuideRegression = true
	second := exploreIsolated(t, config)
	if first.CorpusAdded != 1 || second.CorpusAdded != 0 || second.Attempted != 1 {
		t.Fatalf("same-identity guided campaigns: first=%#v second=%#v", first, second)
	}
	original, err := artifact.OpenArtifact(first.SuccessArtifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := artifact.OpenArtifact(second.SuccessArtifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original.Manifest().Outcome, repeated.Manifest().Outcome) || !reflect.DeepEqual(original.Manifest().Streams, repeated.Manifest().Streams) || original.Manifest().IOProfile.Transcript.SHA256 != repeated.Manifest().IOProfile.Transcript.SHA256 {
		t.Fatalf("corpus seed did not reproduce its retained execution")
	}
}

func TestGuidedSelectionExcludesAnsweredRequestedSeeds(t *testing.T) {
	for _, test := range []struct {
		name, seeds string
		want        []uint64
	}{
		{"partial", "0-3", []uint64{1, 2, 3}},
		{"all", "0", nil},
		{"outside", "10-13", []uint64{10, 11, 12, 13}},
	} {
		t.Run(test.name, func(t *testing.T) {
			corpus := filepath.Join(t.TempDir(), "corpus")
			result := func(uint64) backend.ExecutionResult {
				r := processResult(0, "", "")
				r.IOTranscript = completeEmptyTranscript()
				return r
			}
			config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{result: result}, "0", PolicyAll, 1)
			config.Guide = true
			config.Corpus = corpus
			config.Coverage = CoverageSemantic
			config.Replayer = &matchingReplayer{}
			if _, err := exploreWith(context.Background(), config, configDependencies); err != nil {
				t.Fatal(err)
			}
			executor := &fakeExecutor{result: result}
			config, configDependencies = testConfig(t, newFakePreparer(t), executor, test.seeds, PolicyAll, 1)
			config.Guide = true
			config.Corpus = corpus
			config.Coverage = CoverageSemantic
			config.Replayer = &matchingReplayer{}
			summary, err := exploreWith(context.Background(), config, configDependencies)
			if err != nil {
				t.Fatal(err)
			}
			if got := executorSeeds(executor); !slices.Equal(got, test.want) {
				t.Fatalf("executed seeds=%v want=%v", got, test.want)
			}
			if summary.Attempted != uint64(len(test.want)) {
				t.Fatalf("summary=%#v", summary)
			}
			opened, err := campaign.OpenCampaign(summary.CampaignPath)
			if err != nil {
				t.Fatal(err)
			}
			if opened.Record.SelectionCount != record.Uint64String(len(test.want)) {
				t.Fatalf("selection count=%d", opened.Record.SelectionCount)
			}
		})
	}
}

func TestGuidedPlanShardsPreserveSelectionWithoutLiveCorpus(t *testing.T) {
	for _, regression := range []bool{false, true} {
		t.Run(map[bool]string{false: "new", true: "regression"}[regression], func(t *testing.T) {
			result := func(uint64) backend.ExecutionResult {
				r := processResult(0, "", "")
				r.IOTranscript = completeEmptyTranscript()
				return r
			}
			config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{result: result}, "0", PolicyAll, 1)
			config.Guide = true
			config.Coverage = CoverageSemantic
			config.Corpus = filepath.Join(t.TempDir(), "corpus")
			config.Replayer = &matchingReplayer{}
			if _, err := exploreWith(context.Background(), config, configDependencies); err != nil {
				t.Fatal(err)
			}
			config.Preparer = newFakePreparer(t)
			config.Seeds = "0-3"
			config.GuideRegression = regression
			planPath := filepath.Join(t.TempDir(), "plan.json")
			planned, err := createCampaignPlanWith(context.Background(), CampaignPlanSpec{Campaign: config, Output: planPath}, configDependencies)
			if err != nil {
				t.Fatal(err)
			}
			opened, err := campaign.OpenPortablePlan(planPath)
			if err != nil {
				t.Fatal(err)
			}
			expected := "1-3"
			if regression {
				expected = "0,1-3"
			}
			if opened.Plan.Selection != expected || opened.Plan.Guidance.Regression != regression {
				t.Fatalf("plan=%#v", opened.Plan)
			}
			if err := os.Rename(config.Corpus, config.Corpus+"-detached"); err != nil {
				t.Fatal(err)
			}
			var paths []string
			var seeds []uint64
			for index := uint64(0); index < 2; index++ {
				executor := &fakeExecutor{result: result}
				summary, err := runCampaignShardWith(context.Background(), CampaignShardSpec{PlanPath: planPath, Shard: CampaignShard{Index: index, Count: 2}, Artifacts: t.TempDir(), RunnerBuild: config.RunnerBuild, Backend: scriptProvider()}, executionDependencies{executor: executor})
				if err != nil {
					t.Fatal(err)
				}
				paths = append(paths, summary.CampaignPath)
				seeds = append(seeds, executorSeeds(executor)...)
				if summary.Guidance == nil || summary.Guidance.Regression != regression {
					t.Fatalf("shard guidance=%#v", summary.Guidance)
				}
			}
			slices.Sort(seeds)
			want := []uint64{1, 2, 3}
			if regression {
				want = []uint64{0, 1, 2, 3}
			}
			if !slices.Equal(seeds, want) {
				t.Fatalf("shard seeds=%v want=%v", seeds, want)
			}
			merged, err := MergeCampaignShards(context.Background(), CampaignMergeSpec{PlanPath: planPath, Shards: paths, Output: filepath.Join(t.TempDir(), "merged")})
			if err != nil {
				t.Fatal(err)
			}
			if merged.Attempted != uint64(len(want)) || merged.PlanSHA256 != planned.SHA256 {
				t.Fatalf("merged=%#v", merged)
			}
		})
	}
}

func TestGuidedResumeRejectsChangedRegressionModeAndCountsNewExecutions(t *testing.T) {
	result := func(uint64) backend.ExecutionResult {
		r := processResult(0, "", "")
		r.IOTranscript = completeEmptyTranscript()
		return r
	}
	config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{result: result}, "0", PolicyAll, 1)
	config.Guide = true
	config.Coverage = CoverageSemantic
	config.Corpus = filepath.Join(t.TempDir(), "corpus")
	config.Replayer = &matchingReplayer{}
	if _, err := exploreWith(context.Background(), config, configDependencies); err != nil {
		t.Fatal(err)
	}
	config.Preparer = newFakePreparer(t)
	configDependencies.executor = &guidedInterruptExecutor{}
	config.Seeds = "0-3"
	config.GuideRegression = true
	ctx := cancelOnProgress(t, &config, func(progress CampaignEvent) bool { return progress.Succeeded == 1 })
	partial, err := exploreWith(ctx, config, configDependencies)
	if err == nil {
		t.Fatal("campaign was not interrupted")
	}
	changed := false
	_, err = resumeWith(context.Background(), ResumeSpec{CampaignPath: partial.CampaignPath, RunnerBuild: config.RunnerBuild, Backend: scriptProvider(), GuideRegression: &changed}, executionDependencies{executor: &fakeExecutor{result: result}})
	if err == nil || !strings.Contains(err.Error(), "regression mode") {
		t.Fatalf("changed mode error=%v", err)
	}
	executor := &fakeExecutor{result: result}
	resumed, err := resumeWith(context.Background(), ResumeSpec{CampaignPath: partial.CampaignPath, RunnerBuild: config.RunnerBuild, Backend: scriptProvider(), Replayer: &matchingReplayer{}}, executionDependencies{executor: executor})
	if err != nil {
		t.Fatal(err)
	}
	if got := executorSeeds(executor); !slices.Equal(got, []uint64{1, 2, 3}) {
		t.Fatalf("resume executed=%v", got)
	}
	if resumed.Guidance == nil || resumed.Guidance.NewExecutions != 3 || resumed.Attempted != 4 {
		t.Fatalf("resumed=%#v guidance=%#v", resumed, resumed.Guidance)
	}
}

func TestGuidedRegressionEarlyFailureCountsOnlyAttemptedNewExecutions(t *testing.T) {
	result := func(uint64) backend.ExecutionResult {
		r := processResult(0, "", "")
		r.IOTranscript = completeEmptyTranscript()
		return r
	}
	config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{result: result}, "0", PolicyAll, 1)
	config.Guide = true
	config.Coverage = CoverageSemantic
	config.Corpus = filepath.Join(t.TempDir(), "corpus")
	config.Replayer = &matchingReplayer{}
	if _, err := exploreWith(context.Background(), config, configDependencies); err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{result: func(seed uint64) backend.ExecutionResult {
		r := processResult(7, "failure", "")
		r.IOTranscript = completeEmptyTranscript()
		return r
	}}
	config.Preparer = newFakePreparer(t)
	configDependencies.executor = executor
	config.Seeds = "10-13"
	config.OnFailure = PolicyFirst
	config.GuideRegression = true
	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Attempted != 1 || summary.Guidance == nil || summary.Guidance.NewExecutions != 0 || summary.Guidance.Requested != 4 || summary.Guidance.Guided != 1 {
		t.Fatalf("summary=%#v guidance=%#v", summary, summary.Guidance)
	}
}

func TestExcludeAnsweredSeedsPreservesMaximumRangesAndInputOrder(t *testing.T) {
	selection, err := ParseSeeds("18446744073709551615,1-18446744073709551614")
	if err != nil {
		t.Fatal(err)
	}
	reduced := excludeAnsweredSeeds(selection, []uint64{1, 2, 2, 4, 18446744073709551615})
	if reduced.Count() != 18446744073709551611 || reduced.String() != "3,5-18446744073709551614" {
		t.Fatalf("selection=%s count=%d", reduced.String(), reduced.Count())
	}
}

type guidedInterruptExecutor struct{}

func (*guidedInterruptExecutor) Run(ctx context.Context, spec backend.ExecutionRequest) (backend.ExecutionResult, error) {
	if seedFromEnvironment(spec.Env) == 0 {
		result := processResult(0, "", "")
		result.IOTranscript = completeEmptyTranscript()
		return result, nil
	}
	<-ctx.Done()
	return backend.ExecutionResult{}, ctx.Err()
}

func TestFullyAnsweredGuidedPlanExecutesEmptyShardsAndMerges(t *testing.T) {
	result := func(uint64) backend.ExecutionResult {
		r := processResult(0, "", "")
		r.IOTranscript = completeEmptyTranscript()
		return r
	}
	config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{result: result}, "0", PolicyAll, 1)
	config.Guide = true
	config.Coverage = CoverageSemantic
	config.Corpus = filepath.Join(t.TempDir(), "corpus")
	config.Replayer = &matchingReplayer{}
	if _, err := exploreWith(context.Background(), config, configDependencies); err != nil {
		t.Fatal(err)
	}
	config.Preparer = newFakePreparer(t)
	path := filepath.Join(t.TempDir(), "plan.json")
	planned, err := createCampaignPlanWith(context.Background(), CampaignPlanSpec{Campaign: config, Output: path}, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if planned.SelectionCount != 0 {
		t.Fatalf("planned=%#v", planned)
	}
	var shards []string
	for index := uint64(0); index < 2; index++ {
		executor := &fakeExecutor{result: result}
		summary, err := runCampaignShardWith(context.Background(), CampaignShardSpec{PlanPath: path, Shard: CampaignShard{Index: index, Count: 2}, Artifacts: t.TempDir(), RunnerBuild: config.RunnerBuild, Backend: scriptProvider()}, executionDependencies{executor: executor})
		if err != nil {
			t.Fatal(err)
		}
		if summary.Attempted != 0 || summary.Guidance == nil || summary.Guidance.Answered != 1 || len(executorSeeds(executor)) != 0 {
			t.Fatalf("summary=%#v", summary)
		}
		shards = append(shards, summary.CampaignPath)
	}
	merged, err := MergeCampaignShards(context.Background(), CampaignMergeSpec{PlanPath: path, Shards: shards, Output: filepath.Join(t.TempDir(), "merged")})
	if err != nil {
		t.Fatal(err)
	}
	if merged.Attempted != 0 || merged.SelectionCount != 0 {
		t.Fatalf("merged=%#v", merged)
	}
	opened, err := campaign.OpenMergedCampaign(merged.Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(opened.Executions) != 0 || opened.Record.Attempted != 0 {
		t.Fatalf("reopened=%#v", opened)
	}
	inspected, err := inspect.Inspect(merged.Path, inspect.InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if inspected.Kind != "merged-campaign" || inspected.Merged == nil || inspected.Merged.Attempted != 0 {
		t.Fatalf("inspection=%#v", inspected)
	}
}

func TestRegressionGuidanceReservesRoundedUnguidedQuarter(t *testing.T) {
	base, err := ParseSeeds("0-6")
	if err != nil {
		t.Fatal(err)
	}
	mixed, err := mixGuidedSelection(base, []uint64{100, 101, 102, 103, 104, 105})
	if err != nil {
		t.Fatal(err)
	}
	if mixed.String() != "100,101,102,103,104,0-1" {
		t.Fatalf("selection=%s", mixed.String())
	}
}

func TestGuidanceRegressionModeChangesPortablePlanIdentity(t *testing.T) {
	result := func(uint64) backend.ExecutionResult {
		r := processResult(0, "", "")
		r.IOTranscript = completeEmptyTranscript()
		return r
	}
	config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{result: result}, "0", PolicyAll, 1)
	config.Guide = true
	config.Coverage = CoverageSemantic
	config.Corpus = filepath.Join(t.TempDir(), "corpus")
	config.Replayer = &matchingReplayer{}
	if _, err := exploreWith(context.Background(), config, configDependencies); err != nil {
		t.Fatal(err)
	}
	config.Seeds = "100"
	var identities []record.SHA256
	for _, regression := range []bool{false, true} {
		config.Preparer = newFakePreparer(t)
		config.GuideRegression = regression
		planned, err := createCampaignPlanWith(context.Background(), CampaignPlanSpec{Campaign: config, Output: filepath.Join(t.TempDir(), "plan.json")}, configDependencies)
		if err != nil {
			t.Fatal(err)
		}
		opened, err := campaign.OpenPortablePlan(planned.Path)
		if err != nil {
			t.Fatal(err)
		}
		if opened.Plan.Selection != "100" || opened.Plan.SelectionCount != 1 || opened.Plan.Guidance.GuidedCount != 0 {
			t.Fatalf("plan=%#v", opened.Plan)
		}
		identities = append(identities, planned.SHA256)
	}
	if identities[0] == identities[1] {
		t.Fatal("regression mode did not change otherwise identical plan identity")
	}
}

func TestGuidanceReopensOnlyWithMatchingEnvironmentAndTickPolicy(t *testing.T) {
	for _, coverage := range []CoverageMode{CoverageSemantic, CoverageChoice} {
		for _, test := range []struct {
			name, firstEnv, secondEnv, firstTick, secondTick string
			match                                            bool
		}{
			{name: "same configuration across seeds", firstEnv: "MODE=first", secondEnv: "MODE=first", match: true},
			{name: "changed environment", firstEnv: "MODE=first", secondEnv: "MODE=second"},
			{name: "strict to forward", secondTick: record.ClockTickForward},
			{name: "forward to strict", firstTick: record.ClockTickForward},
		} {
			t.Run(string(coverage)+"/"+test.name, func(t *testing.T) {
				preparer := newFakePreparer(t)
				limit := choiceTraceLimit(t, 1)
				executor := &fakeExecutor{result: func(uint64) backend.ExecutionResult {
					result := processResult(0, "", "")
					result.IOTranscript = completeEmptyTranscript()
					if coverage == CoverageChoice {
						result.ChoiceTrace = completeChoiceTrace(t, preparer.prepared.BuildKey, limit, []choice.Record{{Ordinal: 0, Kind: choice.KindRunnable, Flags: choice.FlagDecision, SiteOffset: 24, Alternatives: 2}})
					}
					return result
				}}
				corpus := filepath.Join(t.TempDir(), "corpus")
				run := func(seed, env, tick string) error {
					config, configDependencies := testConfig(t, preparer, executor, seed, PolicyAll, 1)
					config.Coverage = coverage
					config.Guide = true
					config.Corpus = corpus
					config.Replayer = &matchingReplayer{}
					config.ClockTick = tick
					if coverage == CoverageChoice {
						config.ChoiceTraceLimit = limit
					}
					if env != "" {
						config.Environment = []string{env}
					}
					_, err := exploreWith(context.Background(), config, configDependencies)
					return err
				}
				if err := run("7", test.firstEnv, test.firstTick); err != nil {
					t.Fatal(err)
				}
				err := run("8", test.secondEnv, test.secondTick)
				if test.match {
					if err != nil {
						t.Fatal(err)
					}
				} else if err == nil || !strings.Contains(err.Error(), "guided corpus identity does not match") {
					t.Fatalf("changed configuration error = %v", err)
				}
			})
		}
	}
}

func TestRunGuidesFromImmutableCorpusAndKeepsUnguidedSeeds(t *testing.T) {
	corpus := filepath.Join(t.TempDir(), "corpus")
	replayer := &matchingReplayer{}
	firstExecutor := &fakeExecutor{result: func(seed uint64) backend.ExecutionResult {
		result := processResult(0, "", "")
		probe := "stdlib.os.openfile"
		if seed%2 == 1 {
			probe = "stdlib.net.dialtcp"
		}
		result.IOTranscript = semanticTranscript(t, probe)
		return result
	}}
	firstConfig, firstConfigDependencies := testConfig(t, newFakePreparer(t), firstExecutor, "0-7", PolicyAll, 3)
	firstConfig.Coverage = CoverageSemantic
	firstConfig.Guide = true
	firstConfig.Corpus = corpus
	firstConfig.Replayer = replayer
	first, err := exploreWith(context.Background(), firstConfig, firstConfigDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if first.CorpusAdded != 2 || first.CorpusEntries != 2 || replayer.calls != 2 {
		t.Fatalf("first guided summary = %#v, replay calls = %d", first, replayer.calls)
	}

	secondExecutor := &fakeExecutor{result: firstExecutor.result}
	secondConfig, secondConfigDependencies := testConfig(t, newFakePreparer(t), secondExecutor, "100-107", PolicyAll, 3)
	secondConfig.Coverage = CoverageSemantic
	secondConfig.Guide = true
	secondConfig.GuideRegression = true
	secondConfig.Corpus = corpus
	secondConfig.Replayer = replayer
	second, err := exploreWith(context.Background(), secondConfig, secondConfigDependencies)
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
	executor := &fakeExecutor{result: func(seed uint64) backend.ExecutionResult {
		result := processResult(0, "", "")
		result.IOTranscript = completeEmptyTranscript()
		result.ChoiceTrace = completeChoiceTrace(t, preparer.prepared.BuildKey, limit, []choice.Record{{
			Ordinal: 0, Kind: choice.KindRunnable, Flags: choice.FlagDecision, SiteOffset: 24, Alternatives: 2, Selected: uint32(seed % 2),
		}})
		return result
	}}
	config, configDependencies := testConfig(t, preparer, executor, "1-2", PolicyAll, 1)
	config.Coverage = CoverageChoice
	config.ChoiceTraceLimit = limit
	config.Guide = true
	config.Corpus = filepath.Join(t.TempDir(), "corpus")
	config.Replayer = replayer
	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if summary.CorpusAdded != 2 || summary.CorpusEntries != 2 || replayer.calls != 2 {
		t.Fatalf("guided choice summary = %#v, replay calls = %d", summary, replayer.calls)
	}
}

func TestRunGuidanceWithoutReplayCapabilityFailsClosed(t *testing.T) {
	config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(uint64) backend.ExecutionResult {
		result := processResult(0, "", "")
		result.IOTranscript = semanticTranscript(t, "stdlib.os.openfile")
		return result
	}}, "1", PolicyAll, 1)
	config.Coverage = CoverageSemantic
	config.Guide = true
	config.Corpus = filepath.Join(t.TempDir(), "corpus")
	config.Backend = noReplayProvider{scriptProvider()}
	_, err := exploreWith(context.Background(), config, configDependencies)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "guided_corpus" {
		t.Fatalf("exploreWith() error = %v", err)
	}
}
