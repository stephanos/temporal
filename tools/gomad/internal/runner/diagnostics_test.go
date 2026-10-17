package runner

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	backend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/campaign"
	campaignoptions "go.temporal.io/server/tools/gomad/internal/runner/options"
)

type diagnosticExecutor struct {
	delegate *explorationExecutor
	t        *testing.T
}

func (executor *diagnosticExecutor) Run(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
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
	config, configDependencies := testConfig(t, preparer, executor, "7", PolicyAll, 1)
	config.Diagnostics = true
	config.ChoiceTraceLimit = 1 << 20
	config.CollectExecutionEvidence = true
	config.Coverage = campaignoptions.CoverageSemanticChoice
	configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
	summary, err := exploreWith(context.Background(), config, configDependencies)
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
	config, configDependencies := testConfig(t, preparer, executor, "7", PolicyAll, 1)
	config.Diagnostics = true
	config.ChoiceTraceLimit = 1 << 20
	config.Coverage = campaignoptions.CoverageSemanticChoice
	planned, err := createCampaignPlanWith(context.Background(), CampaignPlanSpec{Campaign: config, Output: filepath.Join(t.TempDir(), "plan.json")}, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	summary, err := runCampaignShardWith(context.Background(), CampaignShardSpec{Backend: scriptProvider(), PlanPath: planned.Path, Shard: CampaignShard{Index: 0, Count: 1}, Artifacts: t.TempDir(), RunnerBuild: config.RunnerBuild}, executionDependencies{executor: executor})
	if err != nil || summary.Diagnostics == nil {
		t.Fatalf("shard diagnostics %+v: %v", summary.Diagnostics, err)
	}
	config.Guide = true
	config.Corpus = filepath.Join(t.TempDir(), "corpus")
	config.Replayer = &matchingReplayer{}
	summary, err = exploreWith(context.Background(), config, configDependencies)
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
	replayed, err := replayWith(context.Background(), ReplaySpec{ArtifactPath: path, ToolchainRoot: toolchainRoot(t), Backend: replayNativeProvider(t)}, executionDependencies{executor: executor})
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Match || executor.request.Diagnostics {
		t.Fatalf("replay %+v", replayed)
	}
	for _, entry := range executor.request.Env {
		if strings.HasPrefix(entry, "GOMAD_DIAGNOSTIC_") {
			t.Fatalf("diagnostic control leaked: %s", entry)
		}
	}
	opened, err := artifact.OpenArtifact(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func(opened *artifact.Opened) {
		if err := opened.Close(); err != nil {
			t.Error(err)
		}
	}(opened)
	found := false
	for _, entry := range opened.Manifest().Environment {
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

func (executor *diagnosticInterruptExecutor) Run(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	executor.calls++
	if executor.calls == 2 {
		return backend.ExecutionResult{}, fmt.Errorf("fixture interruption")
	}
	return executor.delegate.Run(ctx, request)
}

func TestDiagnosticsResumeRestoresTheRecordedProfile(t *testing.T) {
	preparer := newFakePreparer(t)
	newExecutor := func() *diagnosticExecutor {
		return &diagnosticExecutor{t: t, delegate: &explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: 1 << 20}}
	}
	config, configDependencies := testConfig(t, preparer, &diagnosticInterruptExecutor{delegate: newExecutor()}, "7-8", PolicyAll, 1)
	config.Diagnostics = true
	config.ChoiceTraceLimit = 1 << 20
	config.Coverage = campaignoptions.CoverageSemanticChoice
	partial, err := exploreWith(context.Background(), config, configDependencies)
	if err == nil || partial.Diagnostics == nil {
		t.Fatalf("interruption %+v: %v", partial, err)
	}
	originalPath := partial.Diagnostics.Path
	resumed, err := exploreWith(context.Background(), CampaignSpec{ResumeCampaign: partial.CampaignPath, RunnerBuild: config.RunnerBuild, Backend: scriptProvider()}, executionDependencies{executor: newExecutor()})
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

func TestDiagnosticsOffPreservesExistingIdentities(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("retained baseline is darwin/arm64")
	}
	for _, traced := range []bool{false, true} {
		name := "plain"
		if traced {
			name = "choices"
		}
		t.Run(name, func(t *testing.T) {
			preparer := newFakePreparer(t)
			var executor executionRunner = &fakeExecutor{result: func(uint64) backend.ExecutionResult { return processResult(1, "baseline output", "baseline failure") }}
			if traced {
				executor = &explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: 1 << 20, exitCode: 1}
			}
			capture := &diagnosticBaselineExecutor{delegate: executor}
			config, configDependencies := testConfig(t, preparer, capture, "7", PolicyAll, 1)
			capture.root = config.Artifacts
			if traced {
				config.ChoiceTraceLimit = 1 << 20
			}
			planPath := filepath.Join(t.TempDir(), "campaign.plan.json")
			planned, err := createCampaignPlanWith(context.Background(), CampaignPlanSpec{Campaign: config, Output: planPath}, configDependencies)
			if err != nil {
				t.Fatal(err)
			}
			planBytes, err := os.ReadFile(planPath)
			if err != nil {
				t.Fatal(err)
			}
			summary, err := exploreWith(context.Background(), config, configDependencies)
			if err != nil {
				t.Fatal(err)
			}
			if len(summary.Artifacts) != 1 {
				t.Fatalf("artifacts: %v", summary.Artifacts)
			}
			campaignPlan := capture.plan
			opened, err := artifact.OpenArtifact(summary.Artifacts[0])
			if err != nil {
				t.Fatal(err)
			}
			manifest := opened.Manifest()
			if err := opened.Close(); err != nil {
				t.Fatal(err)
			}
			manifest.CampaignID = "fixed-campaign"
			manifest.CreatedAt = "2026-10-02T00:00:00Z"
			manifest.Host.StartedAt = "2026-10-02T00:00:00Z"
			manifest.Host.FinishedAt = "2026-10-02T00:00:00Z"
			manifest.Host.ElapsedNanos = 0
			projection := struct {
				PortablePlan       json.RawMessage `json:"portable_plan"`
				PortablePlanSHA256 string          `json:"portable_plan_sha256"`
				CampaignPlan       json.RawMessage `json:"campaign_plan"`
				Artifact           any             `json:"artifact"`
			}{json.RawMessage(planBytes), string(planned.SHA256), json.RawMessage(campaignPlan), manifest}
			encoded, err := json.Marshal(projection)
			if err != nil {
				t.Fatal(err)
			}
			repeated, err := json.Marshal(projection)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(encoded, repeated) {
				t.Fatalf("diagnostics-off identity projection is not repeatable (%s)", name)
			}
			flavor := "portable"
			if nativeTestsEnabled {
				flavor = "native"
			}
			expected, err := os.ReadFile(filepath.Join("testdata", "diagnostic-identity-"+name+"-"+flavor+".json"))
			if err != nil {
				t.Fatal(err)
			}
			equal, err := semanticJSONEqual(expected, encoded)
			if err != nil {
				t.Fatal(err)
			}
			if !equal {
				t.Fatalf("diagnostics-off identities changed (%s): %s", name, encoded)
			}
		})
	}
}

func semanticJSONEqual(left, right []byte) (bool, error) {
	leftValue, err := decodeSemanticJSON(left)
	if err != nil {
		return false, fmt.Errorf("decode expected JSON: %w", err)
	}
	rightValue, err := decodeSemanticJSON(right)
	if err != nil {
		return false, fmt.Errorf("decode actual JSON: %w", err)
	}
	return reflect.DeepEqual(leftValue, rightValue), nil
}

func decodeSemanticJSON(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("multiple JSON documents")
		}
		return nil, err
	}
	return value, nil
}

func TestSemanticJSONEqualIgnoresObjectOrdering(t *testing.T) {
	equal, err := semanticJSONEqual(
		[]byte(`{"artifact":{"kind":"failure"},"identity":"sha256:same"}`),
		[]byte(`{"identity":"sha256:same","artifact":{"kind":"failure"}}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !equal {
		t.Fatal("semantically equal JSON documents compared unequal")
	}
	equal, err = semanticJSONEqual(
		[]byte(`{"identity":"sha256:before"}`),
		[]byte(`{"identity":"sha256:after"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	if equal {
		t.Fatal("semantically different JSON documents compared equal")
	}
}

type diagnosticBaselineExecutor struct {
	delegate executionRunner
	root     string
	plan     []byte
}

func (e *diagnosticBaselineExecutor) Run(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	paths, err := filepath.Glob(filepath.Join(e.root, "v1", "*"))
	if err != nil {
		return backend.ExecutionResult{}, err
	}
	if len(paths) != 1 {
		return backend.ExecutionResult{}, fmt.Errorf("campaign plan paths: %v", paths)
	}
	plan, err := campaign.ReadResumePlan(paths[0])
	if err != nil {
		return backend.ExecutionResult{}, err
	}
	e.plan, err = json.Marshal(plan)
	if err != nil {
		return backend.ExecutionResult{}, err
	}
	return e.delegate.Run(ctx, request)
}
