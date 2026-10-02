package runner

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
)

type diagnosticExecutor struct {
	delegate *explorationExecutor
	t        *testing.T
}

func (executor *diagnosticExecutor) Run(ctx context.Context, request execution.Spec) (execution.Result, error) {
	executor.t.Helper()
	if !request.Diagnostics {
		executor.t.Fatal("diagnostics lost before execution")
	}
	result, err := executor.delegate.Run(ctx, request)
	if err != nil {
		return result, err
	}
	data := make([]byte, 160)
	copy(data, []byte{'G', 'O', 'M', 'A', 'D', 'D', 'G', 1})
	binary.BigEndian.PutUint32(data[8:12], 1)
	data[12] = 1
	limit, err := choice.DiagnosticLimit(request.Choice.Limit)
	if err != nil {
		return result, err
	}
	binary.BigEndian.PutUint64(data[16:24], limit)
	binary.BigEndian.PutUint64(data[24:32], 160)
	binary.BigEndian.PutUint64(data[32:40], 1)
	result.DiagnosticTrace, err = choice.DecodeDiagnosticTrace(data)
	result.IOTranscript = completeEmptyTranscript()
	return result, err
}

func TestDiagnosticsRetainedWhenSuccessArtifactsAreDiscarded(t *testing.T) {
	preparer := newFakePreparer(t)
	executor := &diagnosticExecutor{t: t, delegate: &explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: 1 << 20}}
	config := testConfig(t, preparer, executor, "7", PolicyAll, 1)
	config.Diagnostics = true
	config.ChoiceTraceLimit = 1 << 20
	config.CollectExecutionEvidence = true
	config.Coverage = CoverageSemanticChoice
	summary, err := Explore(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.RetainedSuccesses != 0 || len(summary.SuccessArtifacts) != 0 || summary.Diagnostics == nil || summary.ExecutionEvidence.Diagnostics == nil {
		t.Fatalf("summary %+v", summary)
	}
	trace, err := choice.ReadDiagnosticTrace(summary.Diagnostics.Path)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Diagnostics.SHA256 != record.SHA256FromSum(trace.SHA256) {
		t.Fatal("retained identity changed")
	}
	info, err := os.Stat(summary.Diagnostics.Path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("trace permissions %v", info.Mode())
	}
	batch, err := campaign.OpenCampaign(summary.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Executions) != 1 || batch.Executions[0].SuccessArtifact != nil {
		t.Fatal("success artifact was retained")
	}
}

func TestDiagnosticsPlanShardAndGuidanceKeepTheProfile(t *testing.T) {
	preparer := newFakePreparer(t)
	executor := &diagnosticExecutor{t: t, delegate: &explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: 1 << 20}}
	config := testConfig(t, preparer, executor, "7", PolicyAll, 1)
	config.Diagnostics = true
	config.ChoiceTraceLimit = 1 << 20
	config.Coverage = CoverageSemanticChoice
	planned, err := CreateCampaignPlan(context.Background(), CampaignPlanSpec{Campaign: config, Output: filepath.Join(t.TempDir(), "plan.json")})
	if err != nil {
		t.Fatal(err)
	}
	summary, err := RunCampaignShard(context.Background(), CampaignShardSpec{PlanPath: planned.Path, Shard: CampaignShard{Index: 0, Count: 1}, Artifacts: t.TempDir(), RunnerBuild: config.RunnerBuild, Executor: executor})
	if err != nil || summary.Diagnostics == nil {
		t.Fatalf("shard diagnostics %+v: %v", summary.Diagnostics, err)
	}
	config.Guide = true
	config.Corpus = filepath.Join(t.TempDir(), "corpus")
	config.Replayer = &matchingReplayer{}
	summary, err = Explore(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.CorpusAdded != 1 || summary.Diagnostics == nil {
		t.Fatalf("guidance result %+v", summary)
	}
}

func TestDiagnosticArtifactReplaysWithoutCollectingSidecar(t *testing.T) {
	path, expected := publishReplayArtifactForTarget(t, nil, replayArtifactTarget{Choices: true, Environment: []record.Environment{{Name: choice.DiagnosticProfileEnvironment, Value: choice.DiagnosticProfile}}})
	executor := &fakeReplayExecutor{result: expected}
	replayed, err := Replay(context.Background(), ReplaySpec{ArtifactPath: path, ToolchainRoot: toolchainRoot(t), SupervisorCommand: []string{"unused"}, Executor: executor})
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Match || executor.request.Diagnostics {
		t.Fatalf("replay %+v", replayed)
	}
	for _, entry := range executor.request.Env {
		if strings.HasPrefix(entry, "GOMAD3_DIAGNOSTIC_") {
			t.Fatalf("diagnostic control leaked: %s", entry)
		}
	}
	opened, err := artifact.OpenArtifact(path)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	found := false
	for _, entry := range opened.Manifest.Environment {
		found = found || entry.Name == choice.DiagnosticProfileEnvironment
	}
	if !found {
		t.Fatal("replay rewrote diagnostic identity")
	}
}

type diagnosticInterruptExecutor struct {
	delegate *diagnosticExecutor
	calls    int
}

func (executor *diagnosticInterruptExecutor) Run(ctx context.Context, request execution.Spec) (execution.Result, error) {
	executor.calls++
	if executor.calls == 2 {
		return execution.Result{}, fmt.Errorf("fixture interruption")
	}
	return executor.delegate.Run(ctx, request)
}

func TestDiagnosticsResumeRestoresTheRecordedProfile(t *testing.T) {
	preparer := newFakePreparer(t)
	newExecutor := func() *diagnosticExecutor {
		return &diagnosticExecutor{t: t, delegate: &explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: 1 << 20}}
	}
	config := testConfig(t, preparer, &diagnosticInterruptExecutor{delegate: newExecutor()}, "7-8", PolicyAll, 1)
	config.Diagnostics = true
	config.ChoiceTraceLimit = 1 << 20
	config.Coverage = CoverageSemanticChoice
	partial, err := Explore(context.Background(), config)
	if err == nil || partial.Diagnostics == nil {
		t.Fatalf("interruption %+v: %v", partial, err)
	}
	originalPath := partial.Diagnostics.Path
	resumed, err := Explore(context.Background(), CampaignSpec{ResumeCampaign: partial.CampaignPath, RunnerBuild: config.RunnerBuild, SupervisorCommand: []string{"unused"}, Executor: newExecutor()})
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Succeeded != 2 || resumed.Diagnostics == nil {
		t.Fatalf("resumed %+v", resumed)
	}
	if _, err := choice.ReadDiagnosticTrace(originalPath); err != nil {
		t.Fatal(err)
	}
}
