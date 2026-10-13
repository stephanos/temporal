package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	backend "go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/backends/native/transcript"
	"go.temporal.io/server/tools/gomad3/internal/strictjson"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/inspect"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestCreateCampaignPlanPublishesStablePreparedTargetBundle(t *testing.T) {
	preparer := newFakePreparer(t)
	config, configDependencies := testConfig(t, preparer, &fakeExecutor{}, "9-11,2", PolicyAll, 2)
	firstPath := filepath.Join(t.TempDir(), "campaign.plan.json")
	first, err := createCampaignPlanWith(context.Background(), CampaignPlanSpec{Campaign: config, Output: firstPath}, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	secondPath := filepath.Join(t.TempDir(), "renamed.plan.json")
	second, err := createCampaignPlanWith(context.Background(), CampaignPlanSpec{Campaign: config, Output: secondPath}, configDependencies)
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
		t.Fatal("plan bytes depend on their output path")
	}
	opened, err := campaign.OpenPortablePlan(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Identity != first.SHA256 || opened.Plan.Selection != config.Seeds || opened.Prepared.SHA256 != string(first.TargetSHA256) {
		t.Fatalf("opened plan = %#v", opened)
	}
	inspection, err := inspect.Inspect(firstPath, inspect.InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Kind != "campaign-plan" || inspection.Plan == nil || inspection.Plan.SHA256 != first.SHA256 || inspection.Plan.SelectionCount != 4 || inspection.Plan.Schema != campaign.PortablePlanSchema || inspection.Plan.Storage != campaign.CampaignLogFormat || inspection.Log != nil {
		t.Fatalf("campaign plan inspection = %#v", inspection)
	}
	ordinaryPath := filepath.Join(t.TempDir(), "ordinary.plan.json")
	var ordinary bytes.Buffer
	if err := json.Indent(&ordinary, firstBytes, "", "  "); err != nil {
		t.Fatal(err)
	}
	ordinary.WriteByte('\n')
	if err := os.WriteFile(ordinaryPath, ordinary.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(ordinaryPath+campaign.PortablePlanBundleSuffix, os.DirFS(firstPath+campaign.PortablePlanBundleSuffix)); err != nil {
		t.Fatal(err)
	}
	for _, fix := range []struct {
		path string
		mode os.FileMode
	}{{ordinaryPath + campaign.PortablePlanBundleSuffix, 0o700}, {filepath.Join(ordinaryPath+campaign.PortablePlanBundleSuffix, campaign.PortablePlanTargetFile), 0o500}} {
		if err := os.Chmod(fix.path, fix.mode); err != nil {
			t.Fatal(err)
		}
	}
	ordinaryOpened, err := campaign.OpenPortablePlan(ordinaryPath)
	if err != nil {
		t.Fatalf("open ordinary JSON plan: %v", err)
	}
	if ordinaryOpened.Identity != record.HashBytes(ordinary.Bytes()) || !reflect.DeepEqual(ordinaryOpened.Plan, opened.Plan) {
		t.Fatalf("ordinary plan = %#v, want semantic plan %#v", ordinaryOpened, opened)
	}

	targetPath := filepath.Join(firstPath+campaign.PortablePlanBundleSuffix, campaign.PortablePlanTargetFile)
	if err := os.Chmod(targetPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetPath, []byte("changed"), 0o500); err != nil {
		t.Fatal(err)
	}
	if _, err := campaign.OpenPortablePlan(firstPath); err == nil {
		t.Fatal("campaign.OpenPortablePlan() accepted a changed prepared target")
	}
}

func TestCreateCampaignPlanRejectsDynamicallyDiscoveredOrEarlyStopWork(t *testing.T) {
	for _, configure := range []func(*CampaignSpec){
		func(config *CampaignSpec) { config.OnFailure = PolicyFirst },
		func(config *CampaignSpec) {
			config.Guide = true
			config.Corpus = t.TempDir()
			config.Coverage = CoverageSemantic
			config.OnFailure = PolicyFirst
		},
		func(config *CampaignSpec) {
			config.Strategy = StrategyChoiceExploration
			config.Seeds = "7"
			config.ChoiceTraceLimit = 1 << 20
			config.MaxExecutions = 2
			config.MaxChoiceDepth = 1
			config.MaxExplorationBytes = 1 << 20
		},
	} {
		config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1-2", PolicyAll, 1)
		configure(&config)
		_, err := createCampaignPlanWith(context.Background(), CampaignPlanSpec{Campaign: config, Output: filepath.Join(t.TempDir(), "campaign.plan.json")}, configDependencies)
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
	config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
	config.Target.WorkingDir = t.TempDir()
	config.IOROMounts = []string{mount + "=input"}
	planPath := filepath.Join(t.TempDir(), "campaign.plan.json")
	_, err := createCampaignPlanWith(context.Background(), CampaignPlanSpec{Campaign: config, Output: planPath}, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(planPath+campaign.PortablePlanBundleSuffix, "mounts", "000000", "value"), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{result: func(uint64) backend.ExecutionResult { return processResult(0, "", "") }}
	_, err = runCampaignShardWith(context.Background(), CampaignShardSpec{
		PlanPath: planPath, Shard: CampaignShard{Index: 0, Count: 1}, Artifacts: t.TempDir(), RunnerBuild: config.RunnerBuild,
		Backend: scriptProvider(),
	}, executionDependencies{executor: executor},
	)
	if err == nil {
		t.Fatal("runCampaignShardWith() accepted a changed read-only mount")
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
		config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
		runnerBuild = config.RunnerBuild
		config.Target.WorkingDir = t.TempDir()
		config.IOROMounts = []string{source + "=input"}
		result, err := createCampaignPlanWith(context.Background(), CampaignPlanSpec{Campaign: config, Output: output}, configDependencies)
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
	opened, err := campaign.OpenPortablePlan(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(opened.Plan.IOROMounts) != 1 || opened.Plan.IOROMounts[0] != "mounts/000000=input" {
		t.Fatalf("portable mount mappings = %q", opened.Plan.IOROMounts)
	}
	if err := os.WriteFile(filepath.Join(firstSource, "nested", "value"), []byte("changed"), 0o640); err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{result: func(uint64) backend.ExecutionResult { return processResult(0, "", "") }}
	result, err := runCampaignShardWith(context.Background(), CampaignShardSpec{
		PlanPath: firstPath, Shard: CampaignShard{Index: 0, Count: 1}, Artifacts: t.TempDir(), RunnerBuild: runnerBuild,
		Backend: scriptProvider(),
	}, executionDependencies{executor: executor},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Succeeded != 1 || len(executor.requests) != 1 {
		t.Fatalf("detached mount shard result = %#v requests=%d", result, len(executor.requests))
	}
}

func TestRunCampaignShardExecutesOnlyItsGlobalOrdinalPartition(t *testing.T) {
	config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "9-11,2", PolicyAll, 2)
	planPath := filepath.Join(t.TempDir(), "campaign.plan.json")
	planned, err := createCampaignPlanWith(context.Background(), CampaignPlanSpec{Campaign: config, Output: planPath}, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{result: func(uint64) backend.ExecutionResult { return processResult(0, "", "") }}
	result, err := runCampaignShardWith(context.Background(), CampaignShardSpec{
		PlanPath: planPath, Shard: CampaignShard{Index: 1, Count: 2}, Artifacts: t.TempDir(), RunnerBuild: config.RunnerBuild,
		Backend: scriptProvider(),
	}, executionDependencies{executor: executor},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.SelectionCount != 2 || result.Attempted != 2 || result.Succeeded != 2 {
		t.Fatalf("shard result = %#v", result)
	}
	opened, err := campaign.OpenCampaign(result.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Record.Schema != "gomad3.campaign/v1" || opened.Record.PlanSHA256 != planned.SHA256 || opened.Record.Shard == nil || opened.Record.Shard.Index != 1 || opened.Record.Shard.Count != 2 {
		t.Fatalf("shard batch identity = %#v", opened.Record)
	}
	if len(opened.Executions) != 2 || opened.Executions[0].SelectionOrdinal != 1 || opened.Executions[0].Seed != 10 || opened.Executions[1].SelectionOrdinal != 3 || opened.Executions[1].Seed != 2 {
		t.Fatalf("shard runs = %#v", opened.Executions)
	}
}

func TestRunCampaignShardResumePreservesItsPlanAndAssignment(t *testing.T) {
	config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "7-10", PolicyAll, 1)
	planPath := filepath.Join(t.TempDir(), "campaign.plan.json")
	planned, err := createCampaignPlanWith(context.Background(), CampaignPlanSpec{Campaign: config, Output: planPath}, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	interrupted := &resumeInterruptExecutor{}
	partial, err := runCampaignShardWith(ctx, CampaignShardSpec{
		PlanPath: planPath, Shard: CampaignShard{Index: 0, Count: 2}, Artifacts: t.TempDir(), RunnerBuild: config.RunnerBuild,
		Backend: scriptProvider(), ProgressInterval: time.Millisecond,
		Progress: func(event CampaignEvent) error {
			if event.Succeeded == 1 {
				cancel()
			}
			return nil
		},
	}, executionDependencies{executor: interrupted},
	)
	if err == nil {
		t.Fatal("runCampaignShardWith() was not interrupted")
	}
	resumedExecutor := &fakeExecutor{result: func(uint64) backend.ExecutionResult { return processResult(0, "", "") }}
	resumed, err := resumeWith(context.Background(), ResumeSpec{CampaignPath: partial.CampaignPath, RunnerBuild: config.RunnerBuild, Backend: scriptProvider()}, executionDependencies{executor: resumedExecutor})
	if err != nil {
		t.Fatal(err)
	}
	if resumed.SelectionCount != 2 || resumed.Attempted != 2 || resumed.Succeeded != 2 {
		t.Fatalf("resumed shard = %#v", resumed)
	}
	opened, err := campaign.OpenCampaign(resumed.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Record.PlanSHA256 != planned.SHA256 || opened.Record.Shard == nil || opened.Record.Shard.Index != 0 || opened.Record.Shard.Count != 2 || len(opened.Executions) != 2 || opened.Executions[0].SelectionOrdinal != 0 || opened.Executions[1].SelectionOrdinal != 2 {
		t.Fatalf("resumed shard batch = %#v runs=%#v", opened.Record, opened.Executions)
	}
}

func TestMergeCampaignShardsIsOrderIndependentAndRequiresCompleteness(t *testing.T) {
	config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "9-11,2", PolicyAll, 2)
	planPath := filepath.Join(t.TempDir(), "campaign.plan.json")
	_, err := createCampaignPlanWith(context.Background(), CampaignPlanSpec{Campaign: config, Output: planPath}, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{result: func(uint64) backend.ExecutionResult { return processResult(0, "", "") }}
	shards := make([]string, 2)
	for index := range shards {
		result, err := runCampaignShardWith(context.Background(), CampaignShardSpec{
			PlanPath: planPath, Shard: CampaignShard{Index: uint64(index), Count: 2}, Artifacts: t.TempDir(), RunnerBuild: config.RunnerBuild,
			Backend: scriptProvider(),
		}, executionDependencies{executor: executor},
		)
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
	firstManifest, err := os.ReadFile(filepath.Join(first.Path, "store", "log"))
	if err != nil {
		t.Fatal(err)
	}
	secondManifest, err := os.ReadFile(filepath.Join(second.Path, "store", "log"))
	if err != nil {
		t.Fatal(err)
	}
	if string(firstManifest) != string(secondManifest) || first.Attempted != 4 || first.Partial || first.Shards != 2 {
		t.Fatalf("merged campaigns differ: %#v %#v", first, second)
	}
	opened, err := campaign.OpenMergedCampaign(first.Path)
	if err != nil {
		t.Fatal(err)
	}
	for ordinal, run := range opened.Executions {
		if run.Execution.SelectionOrdinal != record.Uint64String(ordinal) {
			t.Fatalf("merged run %d = %#v", ordinal, run)
		}
	}
	inspection, err := inspect.Inspect(first.Path, inspect.InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Kind != "merged-campaign" || inspection.Merged == nil || inspection.Merged.PlanSHA256 != first.PlanSHA256 || inspection.Merged.Attempted != 4 {
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
	otherConfig, otherConfigDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "20-23", PolicyAll, 1)
	otherPlanPath := filepath.Join(t.TempDir(), "other.plan.json")
	if _, err := createCampaignPlanWith(context.Background(), CampaignPlanSpec{Campaign: otherConfig, Output: otherPlanPath}, otherConfigDependencies); err != nil {
		t.Fatal(err)
	}
	if _, err := MergeCampaignShards(context.Background(), CampaignMergeSpec{PlanPath: otherPlanPath, Shards: shards[:1], Output: filepath.Join(t.TempDir(), "wrong-plan"), Partial: true}); err == nil {
		t.Fatal("merge accepted a shard from a different plan")
	}
}

func TestMergeCampaignShardsDeduplicatesSharedFailureEvidence(t *testing.T) {
	config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1-2", PolicyAll, 1)
	planPath := filepath.Join(t.TempDir(), "campaign.plan.json")
	_, err := createCampaignPlanWith(context.Background(), CampaignPlanSpec{Campaign: config, Output: planPath}, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{result: func(uint64) backend.ExecutionResult { return processResult(7, "same failure", "") }}
	shards := make([]string, 2)
	for index := range shards {
		result, err := runCampaignShardWith(context.Background(), CampaignShardSpec{
			PlanPath: planPath, Shard: CampaignShard{Index: uint64(index), Count: 2}, Artifacts: t.TempDir(), RunnerBuild: config.RunnerBuild,
			Backend: scriptProvider(),
		}, executionDependencies{executor: executor},
		)
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

// The merged log concatenates the shards' own logs, including their
// admissions and receipts, and its inspection reports the totals the shards
// report combined. Neither merge nor inspection changes a shard.
func TestMergedCampaignInspectionReportsTheShardsCombinedTotals(t *testing.T) {
	config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1-6", PolicyAll, 2)
	planPath := filepath.Join(t.TempDir(), "campaign.plan.json")
	if _, err := createCampaignPlanWith(context.Background(), CampaignPlanSpec{Campaign: config, Output: planPath}, configDependencies); err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{result: func(seed uint64) backend.ExecutionResult {
		if seed%2 == 1 {
			return processResult(7, "same failure", "")
		}
		return processResult(0, "", "")
	}}
	roots := make([]string, 2)
	shards := make([]string, 2)
	var attempted, succeeded, failures uint64
	for index := range shards {
		roots[index] = t.TempDir()
		result, err := runCampaignShardWith(context.Background(), CampaignShardSpec{
			PlanPath: planPath, Shard: CampaignShard{Index: uint64(index), Count: 2}, Artifacts: roots[index], RunnerBuild: config.RunnerBuild, Backend: scriptProvider(),
		}, executionDependencies{executor: executor})
		if err != nil {
			t.Fatal(err)
		}
		shards[index] = result.CampaignPath
		attempted += result.Attempted
		succeeded += result.Succeeded
		failures += result.Failures
	}
	before := make([]map[string]string, len(roots))
	for index, root := range roots {
		before[index] = runnerTreeSnapshot(t, root)
	}
	merged, err := MergeCampaignShards(context.Background(), CampaignMergeSpec{PlanPath: planPath, Shards: shards, Output: filepath.Join(t.TempDir(), "merged")})
	if err != nil {
		t.Fatal(err)
	}
	inspection, err := inspect.Inspect(merged.Path, inspect.InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	report := inspection.Merged
	if inspection.Kind != "merged-campaign" || report == nil || report.Attempted != attempted || report.Succeeded != succeeded || report.Failures != failures || attempted != 6 || failures == 0 ||
		report.DistinctFailures != 1 || report.Shards != 2 || report.Partial || len(report.SourceCampaignIDs) != 2 || uint64(len(report.EvidenceIdentities)) != report.RetainedEvidence {
		t.Fatalf("merged inspection = %#v, shards attempted %d succeeded %d failed %d", report, attempted, succeeded, failures)
	}
	if merged.Attempted != report.Attempted || merged.RetainedEvidence != report.RetainedEvidence || merged.RetainedBytes != report.EvidenceBytes || merged.JournalBytes != report.JournalBytes {
		t.Fatalf("merge result %#v differs from its inspection %#v", merged, report)
	}
	log := inspection.Log
	if log == nil || log.Format != campaign.MergedCampaignLogFormat || log.State != "valid" || log.Records == 0 || log.ValidBytes != log.SnapshotBytes || log.RepairedBytes != 0 {
		t.Fatalf("merged log inspection = %#v", log)
	}
	cleanLog := bytes.Clone(readInspectLog(t, merged.Path))
	if report.Limits.MaximumExecutions != 6 || report.Limits.MaximumPartialExecutions == 0 || report.ArtifactCapacity.TotalBytes == 0 || len(report.Sources) != len(shards) {
		t.Fatalf("merged limits and sources = %#v", report)
	}
	for index, source := range report.Sources {
		if source.Index != uint64(index) || source.Count != 2 || source.Path != shards[index] || source.Records == 0 || source.LogSHA256 == "" || source.CampaignID != report.SourceCampaignIDs[index] {
			t.Fatalf("merged source %d = %#v", index, source)
		}
	}
	for index, root := range roots {
		if after := runnerTreeSnapshot(t, root); !reflect.DeepEqual(after, before[index]) {
			t.Fatalf("shard root %d changed by merge or inspection", index)
		}
	}
	// A merge publishes its log complete, so a torn tail is an invalid store
	// rather than state to report.
	appendInspectLog(t, merged.Path, inspectLogFrame(19, []byte(`{}`))[:9])
	if _, err := inspect.Inspect(merged.Path, inspect.InspectOptions{}); !errors.Is(err, campaign.ErrIncompleteLog) {
		t.Fatalf("inspect.Inspect(torn merged log) error = %v", err)
	} else if invalid := new(inspect.InvalidStoreError); !errors.As(err, &invalid) || invalid.Report.Kind != "merged-campaign" || invalid.Report.Log.Invalid.Problem != "incomplete" || invalid.Report.Log.Tail != "short-frame" || invalid.Report.Log.Invalid.Records != log.Records {
		t.Fatalf("torn merged log diagnosis = %#v", err)
	}
	if err := os.WriteFile(filepath.Join(merged.Path, "store", "log"), cleanLog, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		mutate func()
	}{
		{name: "inconsistent summary", mutate: func() {
			rewriteInspectEnvelope(t, merged.Path, func(envelope map[string]any) { envelope["attempted"] = "7" })
		}},
		{name: "relative source path", mutate: func() {
			rewriteInspectEnvelope(t, merged.Path, func(envelope map[string]any) {
				shards := envelope["shards"].([]any)
				shards[0].(map[string]any)["path"] = "relative/shard"
			})
		}},
		{name: "source scope without publication", mutate: func() {
			const mergedSourceKind, campaignPublishKind = 18, 5
			inSource := false
			removeInspectRecord(t, merged.Path, func(kind uint16, _ []byte) bool {
				if kind == mergedSourceKind {
					inSource = true
					return false
				}
				return inSource && kind == campaignPublishKind
			})
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(merged.Path, "store", "log"), cleanLog, 0o600); err != nil {
				t.Fatal(err)
			}
			test.mutate()
			if _, err := inspect.Inspect(merged.Path, inspect.InspectOptions{}); err == nil {
				t.Fatal("inspect.Inspect() accepted a semantically invalid merged log")
			} else if invalid := new(inspect.InvalidStoreError); !errors.As(err, &invalid) || invalid.Report.Kind != "merged-campaign" || invalid.Report.Log == nil || invalid.Report.Log.Invalid == nil ||
				invalid.Report.Log.Invalid.Problem != "corrupt" || invalid.Report.Log.Invalid.Offset != 0 || invalid.Report.Log.Invalid.Records != 0 {
				t.Fatalf("semantically invalid merged log diagnosis = %#v", err)
			}
		})
	}
}

func runnerTreeSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		value := info.Mode().String()
		if info.Mode().IsRegular() {
			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			value += " " + string(record.HashBytes(contents))
		}
		snapshot[path] = value
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func TestOpenCampaignPlanRejectsStrictlyInvalidDocuments(t *testing.T) {
	ordinary, err := json.Marshal(campaign.PortablePlan{Schema: campaign.PortablePlanSchema, Storage: campaign.PortablePlanStorage, Mapping: campaign.PortablePlanMapping})
	if err != nil {
		t.Fatal(err)
	}
	invalidIdentity, err := json.Marshal(campaign.PortablePlan{Schema: campaign.PortablePlanSchema, Storage: campaign.PortablePlanStorage, Mapping: "ordinal-modulo/v0"})
	if err != nil {
		t.Fatal(err)
	}
	lineage := `"schema":"gomad3.portable-campaign-plan/v1","storage":"gomad3.campaign-log/v1"`
	for _, test := range []struct {
		name     string
		contents string
		want     string
	}{
		{name: "incomplete object", contents: `{` + lineage + `,"mapping":"ordinal-modulo/v1"}`, want: "campaign plan protocol identity is invalid"},
		{name: "trailing data", contents: string(ordinary) + "{}", want: "decode campaign plan:"},
		{name: "unknown field", contents: `{"extra":true,` + lineage + `}`, want: "decode campaign plan:"},
		{name: "duplicate field", contents: `{"schema":"gomad3.portable-campaign-plan/v1",` + lineage + `}`, want: "decode campaign plan:"},
		{name: "malformed", contents: `{"schema":`, want: "decode campaign plan:"},
		{name: "invalid UTF-8", contents: `{` + lineage + `,"mapping":"` + string([]byte{0xff}) + `"}`, want: "decode campaign plan:"},
		{name: "invalid protocol identity", contents: string(invalidIdentity), want: "campaign plan protocol identity is invalid"},
		{name: "earlier lineage", contents: `{"campaign":{},"mapping":"ordinal-modulo/v1","schema":"gomad3.campaign-plan/v1"}`, want: `unsupported campaign plan format "gomad3.campaign-plan/v1" for storage ""`},
		{name: "another storage", contents: `{"schema":"gomad3.portable-campaign-plan/v1","storage":"gomad3.campaign-directory/v1"}`, want: `unsupported campaign plan format "gomad3.portable-campaign-plan/v1" for storage "gomad3.campaign-directory/v1"`},
		{name: "current schema without storage", contents: `{"schema":"gomad3.portable-campaign-plan/v1"}`, want: `unsupported campaign plan format "gomad3.portable-campaign-plan/v1" for storage ""`},
		{name: "no lineage", contents: `{}`, want: "campaign plan protocol identity is invalid"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "campaign.plan.json")
			if err := os.WriteFile(path, []byte(test.contents), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := campaign.OpenPortablePlan(path); err == nil || !strings.HasPrefix(err.Error(), test.want) {
				t.Fatalf("campaign.OpenPortablePlan() error = %v, want %s", err, test.want)
			}
		})
	}
}

func TestEncodePortableCampaignPlanRejectsInvalidOriginalStrings(t *testing.T) {
	invalid := string([]byte{0xff})
	for _, test := range []struct {
		name   string
		change func(*campaign.PortablePlan)
	}{
		{name: "document", change: func(document *campaign.PortablePlan) { document.Schema = invalid }},
		{name: "campaign", change: func(document *campaign.PortablePlan) { document.Campaign.RunnerBuild = invalid }},
		{name: "environment", change: func(document *campaign.PortablePlan) {
			document.Campaign.Environment = []record.Environment{{Name: "KEY", Value: invalid}}
		}},
		{name: "target", change: func(document *campaign.PortablePlan) { document.Campaign.Prepared.Target.Argv = []string{invalid} }},
		{name: "mounts", change: func(document *campaign.PortablePlan) {
			document.Mounts = &campaign.PlanMountIdentity{Mappings: []string{invalid}}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			document := campaign.PortablePlan{Schema: campaign.PortablePlanSchema, Storage: campaign.PortablePlanStorage, Mapping: campaign.PortablePlanMapping}
			test.change(&document)
			if _, err := campaign.EncodePortablePlan(document); err == nil || !strings.Contains(err.Error(), "valid UTF-8") {
				t.Fatalf("campaign.EncodePortablePlan() error = %v", err)
			}
		})
	}
}

// rewriteCampaignPlan re-encodes a published plan document after change, as
// a plan of another lineage or with other contents would be written.
func rewriteCampaignPlan(t *testing.T, path string, change func(*campaign.PortablePlan)) {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document campaign.PortablePlan
	if err := strictjson.Decode(contents, &document); err != nil {
		t.Fatal(err)
	}
	change(&document)
	encoded, err := campaign.EncodePortablePlan(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
}

// A plan of the earlier lineage bound its shards to the segmented journal.
// It is refused by name before anything runs, merges or is inspected, and a
// current plan whose execution limits are not the log's is refused too.
func TestCampaignPlanOfAnotherLineageIsRefusedBeforeExecutionMergeAndInspection(t *testing.T) {
	config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1-2", PolicyAll, 1)
	planPath := filepath.Join(t.TempDir(), "campaign.plan.json")
	if _, err := createCampaignPlanWith(context.Background(), CampaignPlanSpec{Campaign: config, Output: planPath}, configDependencies); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), `"schema":"gomad3.portable-campaign-plan/v1"`) || !strings.Contains(string(contents), `"storage":"gomad3.campaign-log/v1"`) {
		t.Fatalf("published plan does not name its lineage: %s", contents)
	}
	executor := &fakeExecutor{result: func(uint64) backend.ExecutionResult { return processResult(0, "", "") }}
	shard, err := runCampaignShardWith(context.Background(), CampaignShardSpec{
		PlanPath: planPath, Shard: CampaignShard{Index: 0, Count: 1}, Artifacts: t.TempDir(), RunnerBuild: config.RunnerBuild, Backend: scriptProvider(),
	}, executionDependencies{executor: executor})
	if err != nil {
		t.Fatal(err)
	}
	for name, test := range map[string]struct {
		change func(*campaign.PortablePlan)
		check  func(error) bool
	}{
		"earlier lineage": {change: func(document *campaign.PortablePlan) {
			document.Schema, document.Storage = "gomad3.campaign-plan/v1", ""
		}, check: func(err error) bool {
			var unsupported *campaign.UnsupportedCampaignPlanError
			return errors.As(err, &unsupported) && unsupported.Schema == "gomad3.campaign-plan/v1" && unsupported.Storage == ""
		}},
		"retired storage": {change: func(document *campaign.PortablePlan) {
			document.Storage = campaign.LegacyCampaignFormat
		}, check: func(err error) bool {
			var unsupported *campaign.UnsupportedCampaignPlanError
			return errors.As(err, &unsupported) && unsupported.Storage == campaign.LegacyCampaignFormat
		}},
		"execution limits that are not the log's": {change: func(document *campaign.PortablePlan) {
			journal := *document.Campaign.Journal
			journal.MaximumBytes *= 2
			document.Campaign.Journal = &journal
		}, check: func(err error) bool {
			return err != nil && err.Error() == "campaign plan execution limits are not the campaign log limits"
		}},
	} {
		t.Run(name, func(t *testing.T) {
			changed := filepath.Join(t.TempDir(), "changed.plan.json")
			if err := os.WriteFile(changed, contents, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.CopyFS(changed+campaign.PortablePlanBundleSuffix, os.DirFS(planPath+campaign.PortablePlanBundleSuffix)); err != nil {
				t.Fatal(err)
			}
			for _, fix := range []struct {
				path string
				mode os.FileMode
			}{{changed + campaign.PortablePlanBundleSuffix, 0o700}, {filepath.Join(changed+campaign.PortablePlanBundleSuffix, campaign.PortablePlanTargetFile), 0o500}} {
				if err := os.Chmod(fix.path, fix.mode); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := campaign.OpenPortablePlan(changed); err != nil {
				t.Fatalf("copied plan does not open before it changes: %v", err)
			}
			rewriteCampaignPlan(t, changed, test.change)
			if _, err := campaign.OpenPortablePlan(changed); !test.check(err) {
				t.Fatalf("campaign.OpenPortablePlan() error = %v", err)
			}
			executor := &fakeExecutor{result: func(uint64) backend.ExecutionResult { return processResult(0, "", "") }}
			artifacts := t.TempDir()
			if _, err := runCampaignShardWith(context.Background(), CampaignShardSpec{
				PlanPath: changed, Shard: CampaignShard{Index: 0, Count: 1}, Artifacts: artifacts, RunnerBuild: config.RunnerBuild, Backend: scriptProvider(),
			}, executionDependencies{executor: executor}); !test.check(err) {
				t.Fatalf("runCampaignShardWith() error = %v", err)
			}
			if entries, err := os.ReadDir(artifacts); err != nil || len(entries) != 0 || len(executor.requests) != 0 {
				t.Fatalf("refused plan executed or stored work: %v %v, %d requests", entries, err, len(executor.requests))
			}
			output := filepath.Join(t.TempDir(), "merged")
			if _, err := MergeCampaignShards(context.Background(), CampaignMergeSpec{PlanPath: changed, Shards: []string{shard.CampaignPath}, Output: output}); !test.check(err) {
				t.Fatalf("MergeCampaignShards() error = %v", err)
			}
			if _, err := os.Lstat(output); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("refused plan merged: %v", err)
			}
			if _, err := inspect.Inspect(changed, inspect.InspectOptions{}); !test.check(err) {
				t.Fatalf("inspect.Inspect() error = %v", err)
			}
		})
	}
}

// Shards of a current plan still revalidate its prepared bundle before the
// target runs.
func TestRunCampaignShardRevalidatesThePreparedBundleBeforeExecution(t *testing.T) {
	config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1-2", PolicyAll, 1)
	planPath := filepath.Join(t.TempDir(), "campaign.plan.json")
	if _, err := createCampaignPlanWith(context.Background(), CampaignPlanSpec{Campaign: config, Output: planPath}, configDependencies); err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(planPath+campaign.PortablePlanBundleSuffix, campaign.PortablePlanTargetFile)
	original, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	changed := append([]byte(nil), original...)
	changed[len(changed)-1] ^= 0xff
	if err := os.Chmod(targetPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetPath, changed, 0o500); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(targetPath, 0o500); err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{result: func(uint64) backend.ExecutionResult { return processResult(0, "", "") }}
	artifacts := t.TempDir()
	if _, err := runCampaignShardWith(context.Background(), CampaignShardSpec{
		PlanPath: planPath, Shard: CampaignShard{Index: 0, Count: 1}, Artifacts: artifacts, RunnerBuild: config.RunnerBuild, Backend: scriptProvider(),
	}, executionDependencies{executor: executor}); err == nil {
		t.Fatal("runCampaignShardWith() accepted a changed prepared target")
	}
	if entries, err := os.ReadDir(artifacts); err != nil || len(entries) != 0 || len(executor.requests) != 0 {
		t.Fatalf("changed bundle executed or stored work: %v %v, %d requests", entries, err, len(executor.requests))
	}
}

func TestCreateCampaignPlanPreparesAdapterAfterBundleRootExists(t *testing.T) {
	config, dependencies := testConfig(t, nil, &fakeExecutor{}, "1", PolicyAll, 1)
	module, err := filepath.Abs(filepath.Join("..", "backends/native/transcript", "testdata", "sprig"))
	if err != nil {
		t.Fatal(err)
	}
	config.Target = target.Spec{
		Kind: target.KindGoTest, Source: ".", WorkingDir: module,
		ToolchainRoot: toolchainRoot(t),
	}
	_, _, err = transcript.Default().PrepareTargetBuildAdapters(t.Context(), config.Target)
	if err == nil || !strings.Contains(err.Error(), "preparation root") {
		t.Fatalf("pre-bundle adapter preparation error = %v", err)
	}
	path := filepath.Join(t.TempDir(), "campaign.plan.json")
	if _, err := createCampaignPlanWith(t.Context(), CampaignPlanSpec{Campaign: config, Output: path}, dependencies); err != nil {
		t.Fatal(err)
	}
	opened, err := campaign.OpenPortablePlan(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(opened.Prepared.Adapters) != 1 || opened.Prepared.Adapters[0].Module != "github.com/Masterminds/sprig/v3" {
		t.Fatalf("plan adapters = %#v", opened.Prepared.Adapters)
	}
}
