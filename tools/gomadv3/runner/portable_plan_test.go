package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/campaignstore"
	"go.temporal.io/server/tools/gomadv3/runner/internal/execution"
)

func TestCreateCampaignPlanPublishesCanonicalPreparedTargetBundle(t *testing.T) {
	preparer := newFakePreparer(t)
	config := testConfig(t, preparer, &fakeExecutor{}, "9-11,2", PolicyAll, 2)
	firstPath := filepath.Join(t.TempDir(), "campaign.plan.json")
	first, err := CreateCampaignPlan(context.Background(), CampaignPlanSpec{Campaign: config, Output: firstPath})
	if err != nil {
		t.Fatal(err)
	}
	secondPath := filepath.Join(t.TempDir(), "renamed.plan.json")
	second, err := CreateCampaignPlan(context.Background(), CampaignPlanSpec{Campaign: config, Output: secondPath})
	if err != nil {
		t.Fatal(err)
	}
	if first.SHA256 != second.SHA256 || first.SelectionCount != 4 || first.TargetSHA256 != second.TargetSHA256 {
		t.Fatalf("plan identities differ: %#v %#v", first, second)
	}
	firstBytes, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	secondBytes, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatal("canonical plan bytes depend on their output path")
	}
	opened, err := openCampaignPlan(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	if opened.identity != first.SHA256 || opened.plan.Selection != config.Seeds || opened.prepared.SHA256 != string(first.TargetSHA256) {
		t.Fatalf("opened plan = %#v", opened)
	}
	inspection, err := Inspect(firstPath, InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Kind != "campaign-plan" || inspection.Plan == nil || inspection.Plan.SHA256 != first.SHA256 || inspection.Plan.SelectionCount != 4 {
		t.Fatalf("campaign plan inspection = %#v", inspection)
	}

	targetPath := filepath.Join(firstPath+campaignPlanBundleSuffix, campaignPlanTargetFile)
	if err := os.Chmod(targetPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetPath, []byte("changed"), 0o500); err != nil {
		t.Fatal(err)
	}
	if _, err := openCampaignPlan(firstPath); err == nil {
		t.Fatal("openCampaignPlan() accepted a changed prepared target")
	}
}

func TestCreateCampaignPlanRejectsDynamicallyDiscoveredOrEarlyStopWork(t *testing.T) {
	for _, configure := range []func(*CampaignSpec){
		func(config *CampaignSpec) { config.OnFailure = PolicyFirst },
		func(config *CampaignSpec) {
			config.Guide = true
			config.Corpus = t.TempDir()
			config.Coverage = CoverageSemantic
		},
		func(config *CampaignSpec) {
			config.Strategy = StrategyChoiceFrontier
			config.Seeds = "7"
			config.ChoiceTraceLimit = 1 << 20
			config.MaxRuns = 2
			config.MaxChoiceDepth = 1
			config.MaxFrontierBytes = 1 << 20
		},
	} {
		config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1-2", PolicyAll, 1)
		configure(&config)
		_, err := CreateCampaignPlan(context.Background(), CampaignPlanSpec{Campaign: config, Output: filepath.Join(t.TempDir(), "campaign.plan.json")})
		if err == nil {
			t.Fatal("CreateCampaignPlan() accepted a non-static campaign")
		}
	}
}

func TestRunCampaignShardRejectsChangedReadOnlyMountBeforeExecution(t *testing.T) {
	mount := t.TempDir()
	if err := os.WriteFile(filepath.Join(mount, "value"), []byte("planned"), 0o600); err != nil {
		t.Fatal(err)
	}
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
	config.Target.WorkingDir = t.TempDir()
	config.IOROMounts = []string{mount + "=input"}
	planPath := filepath.Join(t.TempDir(), "campaign.plan.json")
	_, err := CreateCampaignPlan(context.Background(), CampaignPlanSpec{Campaign: config, Output: planPath})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(planPath+campaignPlanBundleSuffix, "mounts", "000000", "value"), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{result: func(uint64) execution.Result { return processResult(0, "", "") }}
	_, err = RunCampaignShard(context.Background(), CampaignShardSpec{
		PlanPath: planPath, Shard: CampaignShard{Index: 0, Count: 1}, Artifacts: t.TempDir(), RunnerBuild: config.RunnerBuild,
		SupervisorCommand: []string{"unused"}, Executor: executor,
	})
	if err == nil {
		t.Fatal("RunCampaignShard() accepted a changed read-only mount")
	}
	if len(executor.requests) != 0 {
		t.Fatal("executor started before mount preflight completed")
	}
}

func TestCreateCampaignPlanReadOnlyMountsArePortableAndDetachedFromTheirSource(t *testing.T) {
	createSource := func() string {
		root := t.TempDir()
		if err := os.Mkdir(filepath.Join(root, "nested"), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "nested", "value"), []byte("planned"), 0o640); err != nil {
			t.Fatal(err)
		}
		return root
	}
	firstSource := createSource()
	secondSource := createSource()
	runnerBuild := ""
	create := func(source, output string) CampaignPlanResult {
		config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
		runnerBuild = config.RunnerBuild
		config.Target.WorkingDir = t.TempDir()
		config.IOROMounts = []string{source + "=input"}
		result, err := CreateCampaignPlan(context.Background(), CampaignPlanSpec{Campaign: config, Output: output})
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	firstPath := filepath.Join(t.TempDir(), "first.plan.json")
	secondPath := filepath.Join(t.TempDir(), "second.plan.json")
	first := create(firstSource, firstPath)
	second := create(secondSource, secondPath)
	if first.SHA256 != second.SHA256 {
		t.Fatalf("identical mount trees produced different plan identities: %s != %s", first.SHA256, second.SHA256)
	}
	opened, err := openCampaignPlan(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(opened.plan.IOROMounts) != 1 || opened.plan.IOROMounts[0] != "mounts/000000=input" {
		t.Fatalf("portable mount mappings = %q", opened.plan.IOROMounts)
	}
	if err := os.WriteFile(filepath.Join(firstSource, "nested", "value"), []byte("changed"), 0o640); err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{result: func(uint64) execution.Result { return processResult(0, "", "") }}
	result, err := RunCampaignShard(context.Background(), CampaignShardSpec{
		PlanPath: firstPath, Shard: CampaignShard{Index: 0, Count: 1}, Artifacts: t.TempDir(), RunnerBuild: runnerBuild,
		SupervisorCommand: []string{"unused"}, Executor: executor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Succeeded != 1 || len(executor.requests) != 1 {
		t.Fatalf("detached mount shard result = %#v requests=%d", result, len(executor.requests))
	}
}

func TestRunCampaignShardExecutesOnlyItsGlobalOrdinalPartition(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "9-11,2", PolicyAll, 2)
	planPath := filepath.Join(t.TempDir(), "campaign.plan.json")
	planned, err := CreateCampaignPlan(context.Background(), CampaignPlanSpec{Campaign: config, Output: planPath})
	if err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{result: func(uint64) execution.Result { return processResult(0, "", "") }}
	result, err := RunCampaignShard(context.Background(), CampaignShardSpec{
		PlanPath: planPath, Shard: CampaignShard{Index: 1, Count: 2}, Artifacts: t.TempDir(), RunnerBuild: config.RunnerBuild,
		SupervisorCommand: []string{"unused"}, Executor: executor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.SelectionCount != 2 || result.Attempted != 2 || result.Succeeded != 2 {
		t.Fatalf("shard result = %#v", result)
	}
	opened, err := campaignstore.OpenCampaign(result.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Record.Schema != "gomadv3.batch/v4" || opened.Record.PlanSHA256 != planned.SHA256 || opened.Record.Shard == nil || opened.Record.Shard.Index != 1 || opened.Record.Shard.Count != 2 {
		t.Fatalf("shard batch identity = %#v", opened.Record)
	}
	if len(opened.Runs) != 2 || opened.Runs[0].SelectionOrdinal != 1 || opened.Runs[0].Seed != 10 || opened.Runs[1].SelectionOrdinal != 3 || opened.Runs[1].Seed != 2 {
		t.Fatalf("shard runs = %#v", opened.Runs)
	}
}

func TestRunCampaignShardResumePreservesItsPlanAndAssignment(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "7-10", PolicyAll, 1)
	planPath := filepath.Join(t.TempDir(), "campaign.plan.json")
	planned, err := CreateCampaignPlan(context.Background(), CampaignPlanSpec{Campaign: config, Output: planPath})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	interrupted := &resumeInterruptExecutor{}
	partial, err := RunCampaignShard(ctx, CampaignShardSpec{
		PlanPath: planPath, Shard: CampaignShard{Index: 0, Count: 2}, Artifacts: t.TempDir(), RunnerBuild: config.RunnerBuild,
		SupervisorCommand: []string{"unused"}, Executor: interrupted, ProgressInterval: time.Millisecond,
		Progress: func(event CampaignEvent) error {
			if event.Succeeded == 1 {
				cancel()
			}
			return nil
		},
	})
	if err == nil {
		t.Fatal("RunCampaignShard() was not interrupted")
	}
	resumedExecutor := &fakeExecutor{result: func(uint64) execution.Result { return processResult(0, "", "") }}
	resumed, err := Resume(context.Background(), ResumeSpec{CampaignPath: partial.CampaignPath, RunnerBuild: config.RunnerBuild, SupervisorCommand: []string{"unused"}, Executor: resumedExecutor})
	if err != nil {
		t.Fatal(err)
	}
	if resumed.SelectionCount != 2 || resumed.Attempted != 2 || resumed.Succeeded != 2 {
		t.Fatalf("resumed shard = %#v", resumed)
	}
	opened, err := campaignstore.OpenCampaign(resumed.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Record.PlanSHA256 != planned.SHA256 || opened.Record.Shard == nil || opened.Record.Shard.Index != 0 || opened.Record.Shard.Count != 2 || len(opened.Runs) != 2 || opened.Runs[0].SelectionOrdinal != 0 || opened.Runs[1].SelectionOrdinal != 2 {
		t.Fatalf("resumed shard batch = %#v runs=%#v", opened.Record, opened.Runs)
	}
}

func TestMergeCampaignShardsIsOrderIndependentAndRequiresCompleteness(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "9-11,2", PolicyAll, 2)
	planPath := filepath.Join(t.TempDir(), "campaign.plan.json")
	_, err := CreateCampaignPlan(context.Background(), CampaignPlanSpec{Campaign: config, Output: planPath})
	if err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{result: func(uint64) execution.Result { return processResult(0, "", "") }}
	shards := make([]string, 2)
	for index := range shards {
		result, err := RunCampaignShard(context.Background(), CampaignShardSpec{
			PlanPath: planPath, Shard: CampaignShard{Index: uint64(index), Count: 2}, Artifacts: t.TempDir(), RunnerBuild: config.RunnerBuild,
			SupervisorCommand: []string{"unused"}, Executor: executor,
		})
		if err != nil {
			t.Fatal(err)
		}
		shards[index] = result.CampaignPath
	}
	first, err := MergeCampaignShards(context.Background(), CampaignMergeSpec{PlanPath: planPath, Shards: []string{shards[1], shards[0]}, Output: filepath.Join(t.TempDir(), "merged")})
	if err != nil {
		t.Fatal(err)
	}
	second, err := MergeCampaignShards(context.Background(), CampaignMergeSpec{PlanPath: planPath, Shards: shards, Output: filepath.Join(t.TempDir(), "merged")})
	if err != nil {
		t.Fatal(err)
	}
	firstManifest, err := os.ReadFile(filepath.Join(first.Path, "merge.json"))
	if err != nil {
		t.Fatal(err)
	}
	secondManifest, err := os.ReadFile(filepath.Join(second.Path, "merge.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(firstManifest) != string(secondManifest) || first.Attempted != 4 || first.Partial || first.Shards != 2 {
		t.Fatalf("merged campaigns differ: %#v %#v", first, second)
	}
	opened, err := campaignstore.OpenMergedCampaign(first.Path)
	if err != nil {
		t.Fatal(err)
	}
	for ordinal, run := range opened.Runs {
		if run.Run.SelectionOrdinal != evidence.Uint64String(ordinal) {
			t.Fatalf("merged run %d = %#v", ordinal, run)
		}
	}
	inspection, err := Inspect(first.Path, InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Kind != "merged-batch" || inspection.Merged == nil || inspection.Merged.PlanSHA256 != first.PlanSHA256 || inspection.Merged.Attempted != 4 {
		t.Fatalf("merged inspection = %#v", inspection)
	}

	if _, err := MergeCampaignShards(context.Background(), CampaignMergeSpec{PlanPath: planPath, Shards: shards[:1], Output: filepath.Join(t.TempDir(), "complete")}); err == nil {
		t.Fatal("complete merge accepted a missing shard")
	}
	partial, err := MergeCampaignShards(context.Background(), CampaignMergeSpec{PlanPath: planPath, Shards: shards[:1], Output: filepath.Join(t.TempDir(), "partial"), Partial: true})
	if err != nil {
		t.Fatal(err)
	}
	if !partial.Partial || len(partial.Missing) != 2 || partial.Missing[0] != (CampaignOrdinalRange{Start: 1, End: 1}) || partial.Missing[1] != (CampaignOrdinalRange{Start: 3, End: 3}) {
		t.Fatalf("partial merge = %#v", partial)
	}
	if _, err := MergeCampaignShards(context.Background(), CampaignMergeSpec{PlanPath: planPath, Shards: []string{shards[0], shards[0]}, Output: filepath.Join(t.TempDir(), "duplicate"), Partial: true}); err == nil {
		t.Fatal("merge accepted a duplicate shard")
	}
	otherConfig := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "20-23", PolicyAll, 1)
	otherPlanPath := filepath.Join(t.TempDir(), "other.plan.json")
	if _, err := CreateCampaignPlan(context.Background(), CampaignPlanSpec{Campaign: otherConfig, Output: otherPlanPath}); err != nil {
		t.Fatal(err)
	}
	if _, err := MergeCampaignShards(context.Background(), CampaignMergeSpec{PlanPath: otherPlanPath, Shards: shards[:1], Output: filepath.Join(t.TempDir(), "wrong-plan"), Partial: true}); err == nil {
		t.Fatal("merge accepted a shard from a different plan")
	}
}

func TestMergeCampaignShardsDeduplicatesSharedFailureEvidence(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1-2", PolicyAll, 1)
	planPath := filepath.Join(t.TempDir(), "campaign.plan.json")
	_, err := CreateCampaignPlan(context.Background(), CampaignPlanSpec{Campaign: config, Output: planPath})
	if err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{result: func(uint64) execution.Result { return processResult(7, "same failure", "") }}
	shards := make([]string, 2)
	for index := range shards {
		result, err := RunCampaignShard(context.Background(), CampaignShardSpec{
			PlanPath: planPath, Shard: CampaignShard{Index: uint64(index), Count: 2}, Artifacts: t.TempDir(), RunnerBuild: config.RunnerBuild,
			SupervisorCommand: []string{"unused"}, Executor: executor,
		})
		if err != nil {
			t.Fatal(err)
		}
		shards[index] = result.CampaignPath
	}
	merged, err := MergeCampaignShards(context.Background(), CampaignMergeSpec{PlanPath: planPath, Shards: shards, Output: filepath.Join(t.TempDir(), "merged")})
	if err != nil {
		t.Fatal(err)
	}
	if merged.Failures != 2 || merged.DistinctFailures != 1 || merged.RetainedEvidence != 1 {
		t.Fatalf("merged failure evidence = %#v", merged)
	}
}
