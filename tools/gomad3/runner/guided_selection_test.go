package runner

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
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
	if !reflect.DeepEqual(original.Manifest.Outcome, repeated.Manifest.Outcome) || !reflect.DeepEqual(original.Manifest.Streams, repeated.Manifest.Streams) || original.Manifest.IOProfile.Transcript.SHA256 != repeated.Manifest.IOProfile.Transcript.SHA256 {
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
			result := func(uint64) execution.Result {
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
			result := func(uint64) execution.Result {
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
			opened, err := openCampaignPlan(planPath)
			if err != nil {
				t.Fatal(err)
			}
			expected := "1-3"
			if regression {
				expected = "0,1-3"
			}
			if opened.plan.Selection != expected || opened.plan.Guidance.Regression != regression {
				t.Fatalf("plan=%#v", opened.plan)
			}
			if err := os.Rename(config.Corpus, config.Corpus+"-detached"); err != nil {
				t.Fatal(err)
			}
			var paths []string
			var seeds []uint64
			for index := uint64(0); index < 2; index++ {
				executor := &fakeExecutor{result: result}
				summary, err := runCampaignShardWith(context.Background(), CampaignShardSpec{PlanPath: planPath, Shard: CampaignShard{Index: index, Count: 2}, Artifacts: t.TempDir(), RunnerBuild: config.RunnerBuild, SupervisorCommand: []string{"unused"}}, executionDependencies{executor: executor})
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
	result := func(uint64) execution.Result {
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
	_, err = resumeWith(context.Background(), ResumeSpec{CampaignPath: partial.CampaignPath, RunnerBuild: config.RunnerBuild, SupervisorCommand: []string{"unused"}, GuideRegression: &changed}, executionDependencies{executor: &fakeExecutor{result: result}})
	if err == nil || !strings.Contains(err.Error(), "regression mode") {
		t.Fatalf("changed mode error=%v", err)
	}
	executor := &fakeExecutor{result: result}
	resumed, err := resumeWith(context.Background(), ResumeSpec{CampaignPath: partial.CampaignPath, RunnerBuild: config.RunnerBuild, SupervisorCommand: []string{"unused"}, Replayer: &matchingReplayer{}}, executionDependencies{executor: executor})
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
	result := func(uint64) execution.Result {
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
	executor := &fakeExecutor{result: func(seed uint64) execution.Result {
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

func (*guidedInterruptExecutor) Run(ctx context.Context, spec execution.Spec) (execution.Result, error) {
	if seedFromEnvironment(spec.Env) == 0 {
		result := processResult(0, "", "")
		result.IOTranscript = completeEmptyTranscript()
		return result, nil
	}
	<-ctx.Done()
	return execution.Result{}, ctx.Err()
}

func TestFullyAnsweredGuidedPlanExecutesEmptyShardsAndMerges(t *testing.T) {
	result := func(uint64) execution.Result {
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
		summary, err := runCampaignShardWith(context.Background(), CampaignShardSpec{PlanPath: path, Shard: CampaignShard{Index: index, Count: 2}, Artifacts: t.TempDir(), RunnerBuild: config.RunnerBuild, SupervisorCommand: []string{"unused"}}, executionDependencies{executor: executor})
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
	inspected, err := Inspect(merged.Path, InspectOptions{})
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
	result := func(uint64) execution.Result {
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
		opened, err := openCampaignPlan(planned.Path)
		if err != nil {
			t.Fatal(err)
		}
		if opened.plan.Selection != "100" || opened.plan.SelectionCount != 1 || opened.plan.Guidance.GuidedCount != 0 {
			t.Fatalf("plan=%#v", opened.plan)
		}
		identities = append(identities, planned.SHA256)
	}
	if identities[0] == identities[1] {
		t.Fatal("regression mode did not change otherwise identical plan identity")
	}
}
